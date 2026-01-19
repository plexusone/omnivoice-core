// Package pipeline provides components for connecting voice processing stages.
//
// The pipeline package connects STT, LLM, and TTS providers to transport
// connections, handling audio streaming, buffering, and error handling.
package pipeline

import (
	"context"
	"io"
	"log/slog"
	"sync"

	"github.com/agentplexus/omnivoice/stt"
	"github.com/agentplexus/omnivoice/transport"
	"github.com/agentplexus/omnivoice/tts"
)

// TTSPipeline connects a TTS provider's streaming output to a transport connection.
// It handles buffering, error handling, and graceful shutdown.
type TTSPipeline struct {
	ttsProvider tts.Provider
	config      TTSPipelineConfig

	mu       sync.Mutex
	active   bool
	cancelFn context.CancelFunc
}

// TTSPipelineConfig configures the TTS pipeline.
type TTSPipelineConfig struct {
	// VoiceID is the voice to use for synthesis.
	VoiceID string

	// OutputFormat is the audio format (e.g., "ulaw", "pcm").
	// Use "ulaw" for Twilio Media Streams.
	OutputFormat string

	// SampleRate is the audio sample rate.
	// Use 8000 for telephony (mu-law).
	SampleRate int

	// Model is the TTS model to use (provider-specific).
	Model string

	// OnError is called when an error occurs during streaming.
	OnError func(error)

	// OnComplete is called when synthesis completes.
	OnComplete func()
}

// DefaultTTSConfig returns sensible defaults for telephony.
func DefaultTTSConfig() TTSPipelineConfig {
	return TTSPipelineConfig{
		OutputFormat: "ulaw",
		SampleRate:   8000,
	}
}

// NewTTSPipeline creates a new TTS pipeline.
func NewTTSPipeline(provider tts.Provider, config TTSPipelineConfig) *TTSPipeline {
	return &TTSPipeline{
		ttsProvider: provider,
		config:      config,
	}
}

// SynthesizeToConnection synthesizes text and streams audio to the connection.
// This is a non-blocking call that runs in a goroutine.
func (p *TTSPipeline) SynthesizeToConnection(ctx context.Context, text string, conn transport.Connection) error {
	p.mu.Lock()
	if p.active {
		p.mu.Unlock()
		return ErrPipelineActive
	}
	p.active = true

	ctx, cancel := context.WithCancel(ctx)
	p.cancelFn = cancel
	p.mu.Unlock()

	logger := slog.Default()

	synthesisConfig := tts.SynthesisConfig{
		VoiceID:      p.config.VoiceID,
		OutputFormat: p.config.OutputFormat,
		SampleRate:   p.config.SampleRate,
		Model:        p.config.Model,
	}

	stream, err := p.ttsProvider.SynthesizeStream(ctx, text, synthesisConfig)
	if err != nil {
		p.mu.Lock()
		p.active = false
		p.mu.Unlock()
		return err
	}

	go p.streamToConnection(ctx, stream, conn, logger)

	return nil
}

// streamToConnection reads from the TTS stream and writes to the connection.
func (p *TTSPipeline) streamToConnection(ctx context.Context, stream <-chan tts.StreamChunk, conn transport.Connection, logger *slog.Logger) {
	defer func() {
		p.mu.Lock()
		p.active = false
		p.mu.Unlock()

		if p.config.OnComplete != nil {
			p.config.OnComplete()
		}
	}()

	audioIn := conn.AudioIn()

	for {
		select {
		case <-ctx.Done():
			return
		case chunk, ok := <-stream:
			if !ok {
				return
			}

			if chunk.Error != nil {
				if p.config.OnError != nil {
					p.config.OnError(chunk.Error)
				}
				logger.Error("TTS stream error", "error", chunk.Error)
				return
			}

			if len(chunk.Audio) > 0 {
				if _, err := audioIn.Write(chunk.Audio); err != nil {
					if p.config.OnError != nil {
						p.config.OnError(err)
					}
					logger.Error("failed to write audio to connection", "error", err)
					return
				}
			}

			if chunk.IsFinal {
				return
			}
		}
	}
}

// Stop stops any active synthesis.
func (p *TTSPipeline) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancelFn != nil {
		p.cancelFn()
	}
}

// IsActive returns whether the pipeline is currently synthesizing.
func (p *TTSPipeline) IsActive() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.active
}

// StreamingTTSPipeline extends TTSPipeline with input streaming support.
// It connects an io.Reader (e.g., LLM streaming output) to TTS to transport.
type StreamingTTSPipeline struct {
	*TTSPipeline
	streamingProvider tts.StreamingProvider
}

// NewStreamingTTSPipeline creates a pipeline that accepts streaming text input.
func NewStreamingTTSPipeline(provider tts.StreamingProvider, config TTSPipelineConfig) *StreamingTTSPipeline {
	return &StreamingTTSPipeline{
		TTSPipeline:       NewTTSPipeline(provider, config),
		streamingProvider: provider,
	}
}

// StreamToConnection reads text from a reader and streams synthesized audio to the connection.
// This enables low-latency streaming from LLM output to voice output.
func (p *StreamingTTSPipeline) StreamToConnection(ctx context.Context, textReader io.Reader, conn transport.Connection) error {
	p.mu.Lock()
	if p.active {
		p.mu.Unlock()
		return ErrPipelineActive
	}
	p.active = true

	ctx, cancel := context.WithCancel(ctx)
	p.cancelFn = cancel
	p.mu.Unlock()

	logger := slog.Default()

	synthesisConfig := tts.SynthesisConfig{
		VoiceID:      p.config.VoiceID,
		OutputFormat: p.config.OutputFormat,
		SampleRate:   p.config.SampleRate,
		Model:        p.config.Model,
	}

	stream, err := p.streamingProvider.SynthesizeFromReader(ctx, textReader, synthesisConfig)
	if err != nil {
		p.mu.Lock()
		p.active = false
		p.mu.Unlock()
		return err
	}

	go p.streamToConnection(ctx, stream, conn, logger)

	return nil
}

// STTPipeline connects a transport connection's audio output to an STT provider.
// It handles streaming audio from the caller to the STT service and forwards transcripts.
type STTPipeline struct {
	sttProvider stt.StreamingProvider
	config      STTPipelineConfig

	mu       sync.Mutex
	active   bool
	cancelFn context.CancelFunc
	writer   io.WriteCloser
}

// STTPipelineConfig configures the STT pipeline.
type STTPipelineConfig struct {
	// Model is the STT model to use (provider-specific).
	Model string

	// Language is the BCP-47 language code (e.g., "en-US").
	Language string

	// Encoding is the audio encoding (e.g., "mulaw", "pcm").
	// Use "mulaw" for Twilio Media Streams.
	Encoding string

	// SampleRate is the audio sample rate.
	// Use 8000 for telephony (mu-law).
	SampleRate int

	// Channels is the number of audio channels (1 = mono).
	Channels int

	// OnTranscript is called when a transcript is received.
	// The bool indicates if it's a final (non-interim) result.
	OnTranscript func(transcript string, isFinal bool)

	// OnError is called when an error occurs during streaming.
	OnError func(error)

	// OnSpeechStart is called when speech is detected.
	OnSpeechStart func()

	// OnSpeechEnd is called when speech ends (utterance complete).
	OnSpeechEnd func()
}

// DefaultSTTConfig returns sensible defaults for telephony.
func DefaultSTTConfig() STTPipelineConfig {
	return STTPipelineConfig{
		Encoding:   "mulaw",
		SampleRate: 8000,
		Channels:   1,
		Language:   "en-US",
	}
}

// NewSTTPipeline creates a new STT pipeline.
func NewSTTPipeline(provider stt.StreamingProvider, config STTPipelineConfig) *STTPipeline {
	return &STTPipeline{
		sttProvider: provider,
		config:      config,
	}
}

// StartFromConnection starts transcribing audio from the connection.
// This is a non-blocking call that runs in goroutines.
// Call Stop() to end transcription.
func (p *STTPipeline) StartFromConnection(ctx context.Context, conn transport.Connection) error {
	p.mu.Lock()
	if p.active {
		p.mu.Unlock()
		return ErrPipelineActive
	}
	p.active = true

	ctx, cancel := context.WithCancel(ctx)
	p.cancelFn = cancel
	p.mu.Unlock()

	logger := slog.Default()

	transcriptionConfig := stt.TranscriptionConfig{
		Model:      p.config.Model,
		Language:   p.config.Language,
		Encoding:   p.config.Encoding,
		SampleRate: p.config.SampleRate,
		Channels:   p.config.Channels,
	}

	// Start the STT stream
	writer, events, err := p.sttProvider.TranscribeStream(ctx, transcriptionConfig)
	if err != nil {
		p.mu.Lock()
		p.active = false
		p.mu.Unlock()
		return err
	}

	p.mu.Lock()
	p.writer = writer
	p.mu.Unlock()

	// Start goroutine to read audio from connection and send to STT
	go p.streamFromConnection(ctx, conn, writer, logger)

	// Start goroutine to process STT events
	go p.processEvents(ctx, events, logger)

	return nil
}

// streamFromConnection reads audio from the connection and writes to STT.
func (p *STTPipeline) streamFromConnection(ctx context.Context, conn transport.Connection, writer io.WriteCloser, logger *slog.Logger) {
	defer func() {
		if err := writer.Close(); err != nil {
			logger.Error("failed to close STT writer", "error", err)
		}
	}()

	audioOut := conn.AudioOut()
	buf := make([]byte, 1024) // Read in chunks

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := audioOut.Read(buf)
			if err != nil {
				if err == io.EOF {
					return
				}
				if p.config.OnError != nil {
					p.config.OnError(err)
				}
				logger.Error("failed to read audio from connection", "error", err)
				return
			}

			if n > 0 {
				if _, err := writer.Write(buf[:n]); err != nil {
					if p.config.OnError != nil {
						p.config.OnError(err)
					}
					logger.Error("failed to write audio to STT", "error", err)
					return
				}
			}
		}
	}
}

// processEvents handles STT events and calls appropriate callbacks.
func (p *STTPipeline) processEvents(ctx context.Context, events <-chan stt.StreamEvent, logger *slog.Logger) {
	defer func() {
		p.mu.Lock()
		p.active = false
		p.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}

			switch event.Type {
			case stt.EventTranscript:
				if p.config.OnTranscript != nil && event.Transcript != "" {
					p.config.OnTranscript(event.Transcript, event.IsFinal)
				}

			case stt.EventSpeechStart:
				if p.config.OnSpeechStart != nil {
					p.config.OnSpeechStart()
				}

			case stt.EventSpeechEnd:
				if p.config.OnSpeechEnd != nil {
					p.config.OnSpeechEnd()
				}

			case stt.EventError:
				if p.config.OnError != nil && event.Error != nil {
					p.config.OnError(event.Error)
				}
				logger.Error("STT error", "error", event.Error)
			}
		}
	}
}

// Stop stops the STT pipeline.
func (p *STTPipeline) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancelFn != nil {
		p.cancelFn()
	}

	if p.writer != nil {
		_ = p.writer.Close()
		p.writer = nil
	}
}

// IsActive returns whether the pipeline is currently transcribing.
func (p *STTPipeline) IsActive() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.active
}

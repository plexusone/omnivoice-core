package gateway

import (
	"context"
	"sync"
	"time"

	"github.com/plexusone/omnivoice-core/audio/converter"
	"github.com/plexusone/omnivoice-core/audio/format"
	"github.com/plexusone/omnivoice-core/realtime"
)

// RealtimeBridge bridges telephony audio streams with a realtime.Provider.
// It handles audio format conversion and event routing.
type RealtimeBridge struct {
	provider   realtime.Provider
	config     realtime.ProcessConfig
	converter  *converter.Converter
	fromFormat format.AudioFormat
	toFormat   format.AudioFormat

	// Audio channels
	audioIn  chan []byte
	audioOut chan []byte

	// Event handling
	eventCh chan Event
	metrics *bridgeMetrics

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Transcript accumulation
	mu          sync.Mutex
	transcript  []Turn
	currentTurn *Turn
}

// bridgeMetrics tracks bridge performance.
type bridgeMetrics struct {
	mu                    sync.Mutex
	startTime             time.Time
	audioBytesReceived    int64
	audioBytesSent        int64
	turnCount             int
	interruptionCount     int
	toolCallCount         int
	userSpeechDurationMs  int
	agentSpeechDurationMs int
}

// BridgeConfig configures a RealtimeBridge.
type BridgeConfig struct {
	// Provider is the realtime provider to use.
	Provider realtime.Provider

	// ProcessConfig is passed to the provider's ProcessAudioStream.
	ProcessConfig realtime.ProcessConfig

	// FromFormat is the telephony audio format (e.g., format.Twilio).
	FromFormat format.AudioFormat

	// ToFormat is the realtime provider's format (e.g., format.OpenAI).
	ToFormat format.AudioFormat

	// BufferSize is the size of audio channel buffers (default: 100).
	BufferSize int
}

// NewRealtimeBridge creates a new bridge between telephony and a realtime provider.
func NewRealtimeBridge(cfg BridgeConfig) *RealtimeBridge {
	if cfg.BufferSize == 0 {
		cfg.BufferSize = 100
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &RealtimeBridge{
		provider:   cfg.Provider,
		config:     cfg.ProcessConfig,
		converter:  converter.New(),
		fromFormat: cfg.FromFormat,
		toFormat:   cfg.ToFormat,
		audioIn:    make(chan []byte, cfg.BufferSize),
		audioOut:   make(chan []byte, cfg.BufferSize),
		eventCh:    make(chan Event, cfg.BufferSize),
		metrics:    &bridgeMetrics{startTime: time.Now()},
		ctx:        ctx,
		cancel:     cancel,
		transcript: make([]Turn, 0),
	}
}

// Start begins processing audio through the bridge.
// Returns channels for sending/receiving telephony audio.
func (b *RealtimeBridge) Start(ctx context.Context) error {
	// Create a context that's cancelled by either the parent or our own cancel
	ctx, cancel := context.WithCancel(ctx)
	b.cancel = cancel

	// Convert audio input channel for realtime provider
	providerIn := make(chan []byte, cap(b.audioIn))

	// Start input conversion goroutine
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		defer close(providerIn)

		for {
			select {
			case <-ctx.Done():
				return
			case audio, ok := <-b.audioIn:
				if !ok {
					return
				}

				b.metrics.mu.Lock()
				b.metrics.audioBytesReceived += int64(len(audio))
				b.metrics.mu.Unlock()

				// Convert from telephony format to provider format
				converted, err := b.converter.Convert(audio, b.fromFormat, b.toFormat)
				if err != nil {
					b.sendEvent(Event{
						Type:      EventError,
						Timestamp: time.Now(),
						Error:     err,
					})
					continue
				}

				select {
				case providerIn <- converted:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// Start the realtime provider
	audioCh, transcriptCh, err := b.provider.ProcessAudioStream(ctx, providerIn, b.config)
	if err != nil {
		cancel()
		return err
	}

	// Start output processing goroutine
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		b.processOutput(ctx, audioCh, transcriptCh)
	}()

	// Send session started event
	b.sendEvent(Event{
		Type:      EventSessionStarted,
		Timestamp: time.Now(),
	})

	return nil
}

// processOutput handles audio and transcript output from the provider.
func (b *RealtimeBridge) processOutput(ctx context.Context, audioCh <-chan realtime.AudioChunk, transcriptCh <-chan realtime.Transcript) {
	var agentSpeaking bool
	var speechStart time.Time

	for {
		select {
		case <-ctx.Done():
			return

		case chunk, ok := <-audioCh:
			if !ok {
				return
			}

			if chunk.IsFinal {
				if agentSpeaking {
					duration := time.Since(speechStart)
					b.metrics.mu.Lock()
					b.metrics.agentSpeechDurationMs += int(duration.Milliseconds())
					b.metrics.mu.Unlock()

					b.sendEvent(Event{
						Type:      EventAgentSpeechEnd,
						Timestamp: time.Now(),
					})
					agentSpeaking = false
				}
				continue
			}

			if len(chunk.Audio) == 0 {
				continue
			}

			if !agentSpeaking {
				agentSpeaking = true
				speechStart = time.Now()
				b.sendEvent(Event{
					Type:      EventAgentSpeechStart,
					Timestamp: time.Now(),
				})
			}

			// Convert from provider format to telephony format
			converted, err := b.converter.Convert(chunk.Audio, b.toFormat, b.fromFormat)
			if err != nil {
				b.sendEvent(Event{
					Type:      EventError,
					Timestamp: time.Now(),
					Error:     err,
				})
				continue
			}

			b.metrics.mu.Lock()
			b.metrics.audioBytesSent += int64(len(converted))
			b.metrics.mu.Unlock()

			select {
			case b.audioOut <- converted:
			case <-ctx.Done():
				return
			}

		case transcript, ok := <-transcriptCh:
			if !ok {
				return
			}

			if transcript.IsFinal {
				b.finalizeTurn(transcript.IsInput)
				continue
			}

			if transcript.Text != "" {
				eventType := EventAgentTranscript
				role := "agent"
				if transcript.IsInput {
					eventType = EventUserTranscript
					role = "user"
				}

				b.mu.Lock()
				if b.currentTurn == nil || b.currentTurn.Role != role {
					// Start new turn
					b.currentTurn = &Turn{
						Role:      role,
						Timestamp: time.Now(),
					}
				}
				b.currentTurn.Text += transcript.Text
				b.mu.Unlock()

				b.sendEvent(Event{
					Type:      eventType,
					Timestamp: time.Now(),
					Data:      transcript.Text,
				})
			}
		}
	}
}

// finalizeTurn completes the current turn and adds it to transcript.
func (b *RealtimeBridge) finalizeTurn(isInput bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.currentTurn != nil {
		durationMs := int(time.Since(b.currentTurn.Timestamp).Milliseconds())
		b.currentTurn.DurationMs = durationMs
		b.transcript = append(b.transcript, *b.currentTurn)
		b.metrics.mu.Lock()
		b.metrics.turnCount++
		if isInput {
			b.metrics.userSpeechDurationMs += durationMs
		} else {
			b.metrics.agentSpeechDurationMs += durationMs
		}
		b.metrics.mu.Unlock()
		b.currentTurn = nil
	}
}

// SendAudio sends telephony audio to the bridge for processing.
func (b *RealtimeBridge) SendAudio(audio []byte) error {
	select {
	case b.audioIn <- audio:
		return nil
	case <-b.ctx.Done():
		return b.ctx.Err()
	}
}

// AudioOut returns the channel for receiving telephony audio.
func (b *RealtimeBridge) AudioOut() <-chan []byte {
	return b.audioOut
}

// Events returns the channel for session events.
func (b *RealtimeBridge) Events() <-chan Event {
	return b.eventCh
}

// Transcript returns the conversation transcript.
func (b *RealtimeBridge) Transcript() []Turn {
	b.mu.Lock()
	defer b.mu.Unlock()

	result := make([]Turn, len(b.transcript))
	copy(result, b.transcript)
	return result
}

// Metrics returns the bridge performance metrics.
func (b *RealtimeBridge) Metrics() Metrics {
	b.metrics.mu.Lock()
	defer b.metrics.mu.Unlock()

	return Metrics{
		SessionDurationMs:     int(time.Since(b.metrics.startTime).Milliseconds()),
		TurnCount:             b.metrics.turnCount,
		UserSpeechDurationMs:  b.metrics.userSpeechDurationMs,
		AgentSpeechDurationMs: b.metrics.agentSpeechDurationMs,
		InterruptionCount:     b.metrics.interruptionCount,
		ToolCallCount:         b.metrics.toolCallCount,
		AudioBytesReceived:    b.metrics.audioBytesReceived,
		AudioBytesSent:        b.metrics.audioBytesSent,
	}
}

// Interrupt cancels the current agent response.
func (b *RealtimeBridge) Interrupt() {
	b.metrics.mu.Lock()
	b.metrics.interruptionCount++
	b.metrics.mu.Unlock()

	b.sendEvent(Event{
		Type:      EventInterruption,
		Timestamp: time.Now(),
	})

	// Note: The actual interruption is handled by the realtime provider
	// through its VAD or explicit cancellation mechanism
}

// Close stops the bridge and releases resources.
func (b *RealtimeBridge) Close() error {
	// Send session ended event before closing channels
	b.sendEvent(Event{
		Type:      EventSessionEnded,
		Timestamp: time.Now(),
	})

	b.cancel()
	close(b.audioIn)

	// Wait for goroutines to finish
	b.wg.Wait()

	close(b.audioOut)
	close(b.eventCh)

	return b.provider.Close()
}

// sendEvent sends an event to the event channel.
func (b *RealtimeBridge) sendEvent(event Event) {
	select {
	case b.eventCh <- event:
	default:
		// Drop event if channel is full
	}
}

// NewRealtimeBridgeForTwilio creates a bridge configured for Twilio + OpenAI.
func NewRealtimeBridgeForTwilio(provider realtime.Provider, config realtime.ProcessConfig) *RealtimeBridge {
	return NewRealtimeBridge(BridgeConfig{
		Provider:      provider,
		ProcessConfig: config,
		FromFormat:    format.Twilio,
		ToFormat:      format.OpenAI,
	})
}

// NewRealtimeBridgeForTwilioGemini creates a bridge configured for Twilio + Gemini.
func NewRealtimeBridgeForTwilioGemini(provider realtime.Provider, config realtime.ProcessConfig) *RealtimeBridge {
	return NewRealtimeBridge(BridgeConfig{
		Provider:      provider,
		ProcessConfig: config,
		FromFormat:    format.Twilio,
		ToFormat:      format.GeminiInput,
	})
}

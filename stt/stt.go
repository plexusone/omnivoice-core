// Package stt provides a unified interface for Speech-to-Text providers.
package stt

import (
	"context"
	"io"
	"time"

	"github.com/plexusone/omnivoice-core/observability"
)

// TranscriptionConfig configures a STT transcription request.
type TranscriptionConfig struct {
	// Language is the BCP-47 language code (e.g., "en-US").
	// Leave empty for automatic detection.
	Language string

	// Model is the provider-specific model identifier (optional).
	Model string

	// SampleRate is the audio sample rate in Hz.
	SampleRate int

	// Channels is the number of audio channels (1 = mono, 2 = stereo).
	Channels int

	// Encoding is the audio encoding ("pcm", "mp3", "wav", "opus", "flac").
	Encoding string

	// EnablePunctuation adds punctuation to transcription.
	EnablePunctuation bool

	// EnableWordTimestamps includes word-level timestamps.
	EnableWordTimestamps bool

	// EnableSpeakerDiarization identifies different speakers.
	EnableSpeakerDiarization bool

	// MaxSpeakers is the maximum number of speakers to detect (for diarization).
	MaxSpeakers int

	// Keywords are words/phrases to boost recognition accuracy.
	Keywords []string

	// VocabularyID is a provider-specific custom vocabulary ID.
	VocabularyID string

	// Extensions holds provider-specific settings.
	// Keys should be namespaced by provider (e.g., "deepgram.tier", "elevenlabs.num_speakers").
	// Use provider-specific helper functions for type-safe access.
	Extensions map[string]any

	// Hook provides observability for STT operations.
	// If nil, no hooks are called.
	Hook observability.STTHook
}

// Word represents a single transcribed word with timing.
type Word struct {
	// Text is the transcribed word.
	Text string

	// StartTime is when the word starts.
	StartTime time.Duration

	// EndTime is when the word ends.
	EndTime time.Duration

	// Confidence is the recognition confidence (0.0 to 1.0).
	Confidence float64

	// Speaker is the speaker identifier (if diarization enabled).
	Speaker string
}

// Segment represents a segment of transcription (sentence, phrase).
type Segment struct {
	// Text is the transcribed text for this segment.
	Text string

	// StartTime is when the segment starts.
	StartTime time.Duration

	// EndTime is when the segment ends.
	EndTime time.Duration

	// Confidence is the average confidence for this segment.
	Confidence float64

	// Speaker is the speaker identifier (if diarization enabled).
	Speaker string

	// Words contains word-level details (if enabled).
	Words []Word

	// Language is the detected language for this segment.
	Language string
}

// TranscriptionResult contains the result of a STT transcription.
type TranscriptionResult struct {
	// Text is the full transcription text.
	Text string

	// Segments contains segment-level details.
	Segments []Segment

	// Language is the detected language.
	Language string

	// LanguageConfidence is the confidence in language detection.
	LanguageConfidence float64

	// Duration is the audio duration.
	Duration time.Duration
}

// StreamEvent represents an event from streaming transcription.
type StreamEvent struct {
	// Type is the event type.
	Type StreamEventType

	// Transcript is the current transcript (partial or final).
	Transcript string

	// IsFinal indicates if this is a final (non-interim) result.
	IsFinal bool

	// Segment contains segment details for final results.
	Segment *Segment

	// SpeechStarted indicates voice activity started.
	SpeechStarted bool

	// SpeechEnded indicates voice activity ended.
	SpeechEnded bool

	// Error contains any error that occurred.
	Error error
}

// StreamEventType identifies the type of stream event.
type StreamEventType string

const (
	// EventTranscript is a transcription result (partial or final).
	EventTranscript StreamEventType = "transcript"

	// EventSpeechStart indicates the user started speaking.
	EventSpeechStart StreamEventType = "speech_start"

	// EventSpeechEnd indicates the user stopped speaking.
	EventSpeechEnd StreamEventType = "speech_end"

	// EventError indicates an error occurred.
	EventError StreamEventType = "error"
)

// Provider defines the interface for STT providers.
type Provider interface {
	// Name returns the provider name.
	Name() string

	// Transcribe converts audio to text (batch mode).
	Transcribe(ctx context.Context, audio []byte, config TranscriptionConfig) (*TranscriptionResult, error)

	// TranscribeFile transcribes audio from a file path.
	TranscribeFile(ctx context.Context, filePath string, config TranscriptionConfig) (*TranscriptionResult, error)

	// TranscribeURL transcribes audio from a URL.
	TranscribeURL(ctx context.Context, url string, config TranscriptionConfig) (*TranscriptionResult, error)
}

// StreamingProvider extends Provider with real-time streaming support.
type StreamingProvider interface {
	Provider

	// TranscribeStream starts a streaming transcription session.
	// Returns a writer for sending audio and a channel for receiving events.
	TranscribeStream(ctx context.Context, config TranscriptionConfig) (io.WriteCloser, <-chan StreamEvent, error)
}

// Client provides a unified interface across multiple STT providers.
type Client struct {
	providers map[string]Provider
	primary   string
	fallbacks []string
	hook      observability.STTHook
}

// NewClient creates a new STT client with the specified providers.
func NewClient(providers ...Provider) *Client {
	c := &Client{
		providers: make(map[string]Provider),
	}
	for i, p := range providers {
		c.providers[p.Name()] = p
		if i == 0 {
			c.primary = p.Name()
		} else {
			c.fallbacks = append(c.fallbacks, p.Name())
		}
	}
	return c
}

// SetPrimary sets the primary provider by name.
func (c *Client) SetPrimary(name string) {
	c.primary = name
}

// SetFallbacks sets the fallback provider order.
func (c *Client) SetFallbacks(names ...string) {
	c.fallbacks = names
}

// SetHook sets the observability hook for all STT operations.
func (c *Client) SetHook(hook observability.STTHook) {
	c.hook = hook
}

// Hook returns the current observability hook.
func (c *Client) Hook() observability.STTHook {
	return c.hook
}

// Provider returns a specific provider by name.
func (c *Client) Provider(name string) (Provider, bool) {
	p, ok := c.providers[name]
	return p, ok
}

// Transcribe uses the primary provider with automatic fallback.
func (c *Client) Transcribe(ctx context.Context, audio []byte, config TranscriptionConfig) (*TranscriptionResult, error) {
	// Apply client hook if config doesn't have one
	if config.Hook == nil && c.hook != nil {
		config.Hook = c.hook
	}

	// Try primary provider
	if p, ok := c.providers[c.primary]; ok {
		result, err := p.Transcribe(ctx, audio, config)
		if err == nil {
			return result, nil
		}
	}

	// Try fallbacks
	for _, name := range c.fallbacks {
		if p, ok := c.providers[name]; ok {
			result, err := p.Transcribe(ctx, audio, config)
			if err == nil {
				return result, nil
			}
		}
	}

	return nil, ErrNoAvailableProvider
}

// TranscribeStream attempts streaming transcription with the primary provider.
// Falls back to batch transcription if streaming is not available.
func (c *Client) TranscribeStream(ctx context.Context, config TranscriptionConfig) (io.WriteCloser, <-chan StreamEvent, error) {
	// Apply client hook if config doesn't have one
	if config.Hook == nil && c.hook != nil {
		config.Hook = c.hook
	}

	// Try primary provider
	if p, ok := c.providers[c.primary]; ok {
		if sp, ok := p.(StreamingProvider); ok {
			return sp.TranscribeStream(ctx, config)
		}
	}

	// Try fallbacks
	for _, name := range c.fallbacks {
		if p, ok := c.providers[name]; ok {
			if sp, ok := p.(StreamingProvider); ok {
				return sp.TranscribeStream(ctx, config)
			}
		}
	}

	return nil, nil, ErrStreamingNotSupported
}

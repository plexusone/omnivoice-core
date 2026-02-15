// Package tts provides a unified interface for Text-to-Speech providers.
package tts

import (
	"context"
	"io"
)

// Voice represents a voice configuration for TTS.
type Voice struct {
	// ID is the provider-specific voice identifier.
	ID string

	// Name is a human-readable name for the voice.
	Name string

	// Language is the BCP-47 language code (e.g., "en-US").
	Language string

	// Gender is the voice gender ("male", "female", "neutral").
	Gender string

	// Provider is the name of the TTS provider.
	Provider string

	// Metadata contains provider-specific additional information.
	Metadata map[string]any
}

// SynthesisConfig configures a TTS synthesis request.
type SynthesisConfig struct {
	// VoiceID is the voice to use for synthesis.
	VoiceID string

	// Model is the provider-specific model identifier (optional).
	Model string

	// OutputFormat specifies the audio format ("mp3", "pcm", "wav", "opus").
	OutputFormat string

	// SampleRate is the audio sample rate in Hz (e.g., 22050, 44100).
	SampleRate int

	// Speed is the speech speed multiplier (1.0 = normal).
	Speed float64

	// Pitch adjusts the voice pitch (-1.0 to 1.0, 0 = normal).
	Pitch float64

	// Stability controls voice consistency (0.0 to 1.0, provider-specific).
	Stability float64

	// SimilarityBoost enhances voice similarity (0.0 to 1.0, provider-specific).
	SimilarityBoost float64

	// Extensions holds provider-specific settings.
	// Keys should be namespaced by provider (e.g., "elevenlabs.style", "deepgram.tier").
	// Use provider-specific helper functions for type-safe access.
	Extensions map[string]any
}

// SynthesisResult contains the result of a TTS synthesis.
type SynthesisResult struct {
	// Audio is the synthesized audio data.
	Audio []byte

	// Format is the audio format of the result.
	Format string

	// SampleRate is the sample rate of the audio.
	SampleRate int

	// DurationMs is the duration of the audio in milliseconds.
	DurationMs int

	// CharacterCount is the number of characters processed.
	CharacterCount int
}

// StreamChunk represents a chunk of streaming audio.
type StreamChunk struct {
	// Audio is a chunk of audio data.
	Audio []byte

	// IsFinal indicates if this is the last chunk.
	IsFinal bool

	// Error contains any error that occurred during streaming.
	Error error
}

// Provider defines the interface for TTS providers.
type Provider interface {
	// Name returns the provider name.
	Name() string

	// Synthesize converts text to speech and returns audio data.
	Synthesize(ctx context.Context, text string, config SynthesisConfig) (*SynthesisResult, error)

	// SynthesizeStream converts text to speech with streaming output.
	SynthesizeStream(ctx context.Context, text string, config SynthesisConfig) (<-chan StreamChunk, error)

	// ListVoices returns available voices from this provider.
	ListVoices(ctx context.Context) ([]Voice, error)

	// GetVoice returns a specific voice by ID.
	GetVoice(ctx context.Context, voiceID string) (*Voice, error)
}

// StreamingProvider extends Provider with input streaming support.
type StreamingProvider interface {
	Provider

	// SynthesizeFromReader reads text from a reader and streams audio output.
	// Useful for streaming LLM output directly to TTS.
	SynthesizeFromReader(ctx context.Context, reader io.Reader, config SynthesisConfig) (<-chan StreamChunk, error)
}

// Client provides a unified interface across multiple TTS providers.
type Client struct {
	providers map[string]Provider
	primary   string
	fallbacks []string
}

// NewClient creates a new TTS client with the specified providers.
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

// Provider returns a specific provider by name.
func (c *Client) Provider(name string) (Provider, bool) {
	p, ok := c.providers[name]
	return p, ok
}

// Synthesize uses the primary provider with automatic fallback.
func (c *Client) Synthesize(ctx context.Context, text string, config SynthesisConfig) (*SynthesisResult, error) {
	// Try primary provider
	if p, ok := c.providers[c.primary]; ok {
		result, err := p.Synthesize(ctx, text, config)
		if err == nil {
			return result, nil
		}
		// Log error and try fallbacks
	}

	// Try fallbacks
	for _, name := range c.fallbacks {
		if p, ok := c.providers[name]; ok {
			result, err := p.Synthesize(ctx, text, config)
			if err == nil {
				return result, nil
			}
		}
	}

	return nil, ErrNoAvailableProvider
}

// SynthesizeStream uses the primary provider with automatic fallback.
func (c *Client) SynthesizeStream(ctx context.Context, text string, config SynthesisConfig) (<-chan StreamChunk, error) {
	// Try primary provider
	if p, ok := c.providers[c.primary]; ok {
		stream, err := p.SynthesizeStream(ctx, text, config)
		if err == nil {
			return stream, nil
		}
	}

	// Try fallbacks
	for _, name := range c.fallbacks {
		if p, ok := c.providers[name]; ok {
			stream, err := p.SynthesizeStream(ctx, text, config)
			if err == nil {
				return stream, nil
			}
		}
	}

	return nil, ErrNoAvailableProvider
}

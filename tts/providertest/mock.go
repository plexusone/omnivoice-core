package providertest

import (
	"context"
	"io"

	"github.com/agentplexus/omnivoice/tts"
)

// Verify MockProvider implements the interfaces.
var (
	_ tts.Provider          = (*MockProvider)(nil)
	_ tts.StreamingProvider = (*MockProvider)(nil)
)

// MockProvider is a minimal TTS provider for testing the test suite itself.
// It returns fake audio data and can be configured to simulate various behaviors.
type MockProvider struct {
	name   string
	voices []tts.Voice

	// SynthesizeFunc allows customizing Synthesize behavior for testing.
	// If nil, returns default fake audio.
	SynthesizeFunc func(ctx context.Context, text string, config tts.SynthesisConfig) (*tts.SynthesisResult, error)
}

// NewMockProvider creates a new mock TTS provider.
func NewMockProvider() *MockProvider {
	return &MockProvider{
		name: "mock",
		voices: []tts.Voice{
			{ID: "mock-voice-1", Name: "Mock Voice 1", Language: "en-US", Gender: "female", Provider: "mock"},
			{ID: "mock-voice-2", Name: "Mock Voice 2", Language: "en-US", Gender: "male", Provider: "mock"},
			{ID: "mock-voice-3", Name: "Mock Voice 3", Language: "en-GB", Gender: "female", Provider: "mock"},
		},
	}
}

// Name returns the provider name.
func (m *MockProvider) Name() string {
	return m.name
}

// ListVoices returns the mock voice list.
func (m *MockProvider) ListVoices(_ context.Context) ([]tts.Voice, error) {
	return m.voices, nil
}

// GetVoice returns a voice by ID or ErrVoiceNotFound.
func (m *MockProvider) GetVoice(_ context.Context, voiceID string) (*tts.Voice, error) {
	for _, v := range m.voices {
		if v.ID == voiceID {
			return &v, nil
		}
	}
	return nil, tts.ErrVoiceNotFound
}

// Synthesize returns fake audio data.
func (m *MockProvider) Synthesize(ctx context.Context, text string, config tts.SynthesisConfig) (*tts.SynthesisResult, error) {
	if m.SynthesizeFunc != nil {
		return m.SynthesizeFunc(ctx, text, config)
	}

	// Check context
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Return fake MP3 data (ID3 header prefix makes it look like MP3)
	format := config.OutputFormat
	if format == "" {
		format = "mp3"
	}

	var audio []byte
	switch format {
	case "mp3":
		// Fake MP3 with ID3 header
		audio = append([]byte("ID3"), make([]byte, 100)...)
	case "wav":
		// Fake WAV with RIFF header
		audio = append([]byte("RIFF"), make([]byte, 100)...)
	case "flac":
		// Fake FLAC with fLaC header
		audio = append([]byte("fLaC"), make([]byte, 100)...)
	case "opus", "ogg":
		// Fake Ogg with OggS header
		audio = append([]byte("OggS"), make([]byte, 100)...)
	default:
		// Generic fake audio
		audio = make([]byte, 100)
	}

	return &tts.SynthesisResult{
		Audio:          audio,
		Format:         format,
		SampleRate:     24000,
		DurationMs:     len(text) * 50, // ~50ms per character
		CharacterCount: len(text),
	}, nil
}

// SynthesizeStream returns fake streaming audio chunks.
func (m *MockProvider) SynthesizeStream(ctx context.Context, text string, config tts.SynthesisConfig) (<-chan tts.StreamChunk, error) {
	// Check context
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	ch := make(chan tts.StreamChunk, 10)
	go func() {
		defer close(ch)

		// Simulate streaming chunks
		format := config.OutputFormat
		if format == "" {
			format = "mp3"
		}

		// First chunk with format header
		var header []byte
		switch format {
		case "mp3":
			header = []byte("ID3")
		case "wav":
			header = []byte("RIFF")
		case "flac":
			header = []byte("fLaC")
		case "opus", "ogg":
			header = []byte("OggS")
		default:
			header = []byte{0x00, 0x00, 0x00}
		}

		select {
		case <-ctx.Done():
			ch <- tts.StreamChunk{Error: ctx.Err()}
			return
		case ch <- tts.StreamChunk{Audio: header}:
		}

		// Send a few data chunks
		for i := 0; i < 3; i++ {
			select {
			case <-ctx.Done():
				ch <- tts.StreamChunk{Error: ctx.Err()}
				return
			case ch <- tts.StreamChunk{Audio: make([]byte, 50)}:
			}
		}

		// Final chunk
		ch <- tts.StreamChunk{IsFinal: true}
	}()

	return ch, nil
}

// SynthesizeFromReader reads text and returns streaming audio chunks.
func (m *MockProvider) SynthesizeFromReader(ctx context.Context, reader io.Reader, config tts.SynthesisConfig) (<-chan tts.StreamChunk, error) {
	// Read all text first (in real providers, this would be incremental)
	text, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// Delegate to SynthesizeStream
	return m.SynthesizeStream(ctx, string(text), config)
}

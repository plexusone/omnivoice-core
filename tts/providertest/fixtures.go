package providertest

import (
	"context"
	"encoding/binary"
	"errors"
	"time"

	"github.com/plexusone/omnivoice-core/tts"
)

// Common test errors for error injection.
var (
	ErrMockRateLimit     = errors.New("mock: rate limit exceeded")
	ErrMockQuotaExceeded = errors.New("mock: quota exceeded")
	ErrMockNetworkError  = errors.New("mock: network error")
	ErrMockInvalidVoice  = errors.New("mock: invalid voice")
)

// AudioFixture represents a pre-built audio sample for testing.
type AudioFixture struct {
	Data       []byte
	Format     string
	SampleRate int
	DurationMs int
}

// GenerateWAVFixture creates a valid WAV file with silence of the specified duration.
// This produces audio that can be parsed by ffprobe and other audio tools.
func GenerateWAVFixture(durationMs int, sampleRate int) AudioFixture {
	if sampleRate == 0 {
		sampleRate = 22050
	}

	// Calculate sizes
	numChannels := 1
	bitsPerSample := 16
	bytesPerSample := bitsPerSample / 8
	numSamples := (sampleRate * durationMs) / 1000
	dataSize := numSamples * numChannels * bytesPerSample
	fileSize := 44 + dataSize - 8 // 44 byte header, minus 8 for RIFF header

	// Build WAV header (44 bytes)
	header := make([]byte, 44)

	// RIFF chunk
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(fileSize)) //nolint:gosec // G115: test fixture with bounded values
	copy(header[8:12], "WAVE")

	// fmt subchunk
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)                                            // Subchunk1Size (16 for PCM)
	binary.LittleEndian.PutUint16(header[20:22], 1)                                             // AudioFormat (1 = PCM)
	binary.LittleEndian.PutUint16(header[22:24], uint16(numChannels))                           // NumChannels
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))                            // SampleRate
	binary.LittleEndian.PutUint32(header[28:32], uint32(sampleRate*numChannels*bytesPerSample)) //nolint:gosec // G115: test fixture with bounded values
	binary.LittleEndian.PutUint16(header[32:34], uint16(numChannels*bytesPerSample))            // BlockAlign
	binary.LittleEndian.PutUint16(header[34:36], uint16(bitsPerSample))                         // BitsPerSample

	// data subchunk
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], uint32(dataSize)) //nolint:gosec // G115: test fixture with bounded values

	// Combine header with silent audio data (zeros)
	audio := make([]byte, 44+dataSize)
	copy(audio, header)
	// Data portion is already zeros (silence)

	return AudioFixture{
		Data:       audio,
		Format:     "wav",
		SampleRate: sampleRate,
		DurationMs: durationMs,
	}
}

// GenerateShortWAV creates a 100ms WAV fixture at 22050 Hz.
// Useful for quick unit tests.
func GenerateShortWAV() AudioFixture {
	return GenerateWAVFixture(100, 22050)
}

// GenerateOneSecondWAV creates a 1-second WAV fixture at 22050 Hz.
func GenerateOneSecondWAV() AudioFixture {
	return GenerateWAVFixture(1000, 22050)
}

// MockProviderOption configures MockProvider behavior.
type MockProviderOption func(*MockProvider)

// WithName sets the provider name.
func WithName(name string) MockProviderOption {
	return func(m *MockProvider) {
		m.name = name
	}
}

// WithVoices sets the available voices.
func WithVoices(voices []tts.Voice) MockProviderOption {
	return func(m *MockProvider) {
		m.voices = voices
	}
}

// WithFixedDuration makes Synthesize return audio of a fixed duration.
func WithFixedDuration(durationMs int) MockProviderOption {
	return func(m *MockProvider) {
		fixture := GenerateWAVFixture(durationMs, 22050)
		m.SynthesizeFunc = func(ctx context.Context, text string, config tts.SynthesisConfig) (*tts.SynthesisResult, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			return &tts.SynthesisResult{
				Audio:          fixture.Data,
				Format:         fixture.Format,
				SampleRate:     fixture.SampleRate,
				DurationMs:     fixture.DurationMs,
				CharacterCount: len(text),
			}, nil
		}
	}
}

// WithRealisticTiming makes duration proportional to text length (~50ms per char).
func WithRealisticTiming() MockProviderOption {
	return func(m *MockProvider) {
		m.SynthesizeFunc = func(ctx context.Context, text string, config tts.SynthesisConfig) (*tts.SynthesisResult, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			durationMs := len(text) * 50
			if durationMs < 100 {
				durationMs = 100
			}
			fixture := GenerateWAVFixture(durationMs, 22050)
			return &tts.SynthesisResult{
				Audio:          fixture.Data,
				Format:         fixture.Format,
				SampleRate:     fixture.SampleRate,
				DurationMs:     fixture.DurationMs,
				CharacterCount: len(text),
			}, nil
		}
	}
}

// WithError makes Synthesize always return the specified error.
func WithError(err error) MockProviderOption {
	return func(m *MockProvider) {
		m.SynthesizeFunc = func(ctx context.Context, text string, config tts.SynthesisConfig) (*tts.SynthesisResult, error) {
			return nil, err
		}
	}
}

// WithLatency adds artificial delay to Synthesize calls.
func WithLatency(delay time.Duration) MockProviderOption {
	return func(m *MockProvider) {
		original := m.SynthesizeFunc
		m.SynthesizeFunc = func(ctx context.Context, text string, config tts.SynthesisConfig) (*tts.SynthesisResult, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			if original != nil {
				return original(ctx, text, config)
			}
			// Default behavior
			fixture := GenerateShortWAV()
			return &tts.SynthesisResult{
				Audio:          fixture.Data,
				Format:         fixture.Format,
				SampleRate:     fixture.SampleRate,
				DurationMs:     fixture.DurationMs,
				CharacterCount: len(text),
			}, nil
		}
	}
}

// WithFailAfterN makes the provider fail after N successful calls.
func WithFailAfterN(n int, err error) MockProviderOption {
	return func(m *MockProvider) {
		callCount := 0
		m.SynthesizeFunc = func(ctx context.Context, text string, config tts.SynthesisConfig) (*tts.SynthesisResult, error) {
			callCount++
			if callCount > n {
				return nil, err
			}
			fixture := GenerateShortWAV()
			return &tts.SynthesisResult{
				Audio:          fixture.Data,
				Format:         fixture.Format,
				SampleRate:     fixture.SampleRate,
				DurationMs:     fixture.DurationMs,
				CharacterCount: len(text),
			}, nil
		}
	}
}

// NewMockProviderWithOptions creates a mock provider with the specified options.
func NewMockProviderWithOptions(opts ...MockProviderOption) *MockProvider {
	m := NewMockProvider()
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// NewElevenLabsMock creates a mock that simulates ElevenLabs behavior.
func NewElevenLabsMock() *MockProvider {
	return NewMockProviderWithOptions(
		WithName("elevenlabs"),
		WithVoices([]tts.Voice{
			{ID: "21m00Tcm4TlvDq8ikWAM", Name: "Rachel", Language: "en-US", Gender: "female", Provider: "elevenlabs"},
			{ID: "EXAVITQu4vr4xnSDxMaL", Name: "Bella", Language: "en-US", Gender: "female", Provider: "elevenlabs"},
			{ID: "ErXwobaYiN019PkySvjV", Name: "Antoni", Language: "en-US", Gender: "male", Provider: "elevenlabs"},
		}),
		WithRealisticTiming(),
	)
}

// NewDeepgramMock creates a mock that simulates Deepgram TTS behavior.
func NewDeepgramMock() *MockProvider {
	return NewMockProviderWithOptions(
		WithName("deepgram"),
		WithVoices([]tts.Voice{
			{ID: "aura-asteria-en", Name: "Asteria", Language: "en-US", Gender: "female", Provider: "deepgram"},
			{ID: "aura-luna-en", Name: "Luna", Language: "en-US", Gender: "female", Provider: "deepgram"},
			{ID: "aura-orion-en", Name: "Orion", Language: "en-US", Gender: "male", Provider: "deepgram"},
		}),
		WithRealisticTiming(),
	)
}

// NewOpenAIMock creates a mock that simulates OpenAI TTS behavior.
func NewOpenAIMock() *MockProvider {
	return NewMockProviderWithOptions(
		WithName("openai"),
		WithVoices([]tts.Voice{
			{ID: "alloy", Name: "Alloy", Language: "en-US", Gender: "neutral", Provider: "openai"},
			{ID: "echo", Name: "Echo", Language: "en-US", Gender: "male", Provider: "openai"},
			{ID: "fable", Name: "Fable", Language: "en-US", Gender: "neutral", Provider: "openai"},
			{ID: "onyx", Name: "Onyx", Language: "en-US", Gender: "male", Provider: "openai"},
			{ID: "nova", Name: "Nova", Language: "en-US", Gender: "female", Provider: "openai"},
			{ID: "shimmer", Name: "Shimmer", Language: "en-US", Gender: "female", Provider: "openai"},
		}),
		WithRealisticTiming(),
	)
}

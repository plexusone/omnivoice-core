package providertest

import (
	"context"
	"testing"
	"time"

	"github.com/plexusone/omnivoice-core/tts"
)

func TestGenerateWAVFixture(t *testing.T) {
	fixture := GenerateWAVFixture(1000, 22050)

	// Verify WAV header
	if string(fixture.Data[0:4]) != "RIFF" {
		t.Errorf("WAV should start with RIFF, got %q", string(fixture.Data[0:4]))
	}
	if string(fixture.Data[8:12]) != "WAVE" {
		t.Errorf("WAV should have WAVE format, got %q", string(fixture.Data[8:12]))
	}

	// Verify metadata
	if fixture.Format != "wav" {
		t.Errorf("Format = %s, want wav", fixture.Format)
	}
	if fixture.SampleRate != 22050 {
		t.Errorf("SampleRate = %d, want 22050", fixture.SampleRate)
	}
	if fixture.DurationMs != 1000 {
		t.Errorf("DurationMs = %d, want 1000", fixture.DurationMs)
	}

	// Verify size is reasonable (header + samples)
	expectedSamples := (22050 * 1000) / 1000 * 2 // 22050 samples * 2 bytes
	expectedSize := 44 + expectedSamples
	if len(fixture.Data) != expectedSize {
		t.Errorf("Data size = %d, want %d", len(fixture.Data), expectedSize)
	}
}

func TestGenerateShortWAV(t *testing.T) {
	fixture := GenerateShortWAV()
	if fixture.DurationMs != 100 {
		t.Errorf("DurationMs = %d, want 100", fixture.DurationMs)
	}
	if fixture.SampleRate != 22050 {
		t.Errorf("SampleRate = %d, want 22050", fixture.SampleRate)
	}
}

func TestGenerateOneSecondWAV(t *testing.T) {
	fixture := GenerateOneSecondWAV()
	if fixture.DurationMs != 1000 {
		t.Errorf("DurationMs = %d, want 1000", fixture.DurationMs)
	}
}

func TestWithFixedDuration(t *testing.T) {
	m := NewMockProviderWithOptions(WithFixedDuration(2000))

	result, err := m.Synthesize(context.Background(), "test", tts.SynthesisConfig{})
	if err != nil {
		t.Fatalf("Synthesize error: %v", err)
	}
	if result.DurationMs != 2000 {
		t.Errorf("DurationMs = %d, want 2000", result.DurationMs)
	}
	if result.Format != "wav" {
		t.Errorf("Format = %s, want wav", result.Format)
	}
}

func TestWithRealisticTiming(t *testing.T) {
	m := NewMockProviderWithOptions(WithRealisticTiming())

	// Short text
	result1, _ := m.Synthesize(context.Background(), "Hi", tts.SynthesisConfig{})
	// Long text
	result2, _ := m.Synthesize(context.Background(), "This is a much longer text for testing", tts.SynthesisConfig{})

	if result2.DurationMs <= result1.DurationMs {
		t.Errorf("Longer text should have longer duration: %d vs %d", result2.DurationMs, result1.DurationMs)
	}
}

func TestWithError(t *testing.T) {
	m := NewMockProviderWithOptions(WithError(ErrMockRateLimit))

	_, err := m.Synthesize(context.Background(), "test", tts.SynthesisConfig{})
	if err != ErrMockRateLimit {
		t.Errorf("Synthesize error = %v, want ErrMockRateLimit", err)
	}
}

func TestWithLatency(t *testing.T) {
	delay := 100 * time.Millisecond
	m := NewMockProviderWithOptions(WithLatency(delay))

	start := time.Now()
	_, err := m.Synthesize(context.Background(), "test", tts.SynthesisConfig{})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Synthesize error: %v", err)
	}
	if elapsed < delay {
		t.Errorf("Latency = %v, want >= %v", elapsed, delay)
	}
}

func TestWithLatency_ContextCancellation(t *testing.T) {
	m := NewMockProviderWithOptions(WithLatency(5 * time.Second))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := m.Synthesize(ctx, "test", tts.SynthesisConfig{})
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Synthesize should return error on context cancellation")
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("Should have cancelled quickly, took %v", elapsed)
	}
}

func TestWithFailAfterN(t *testing.T) {
	m := NewMockProviderWithOptions(WithFailAfterN(2, ErrMockQuotaExceeded))

	// First two should succeed
	_, err1 := m.Synthesize(context.Background(), "test1", tts.SynthesisConfig{})
	_, err2 := m.Synthesize(context.Background(), "test2", tts.SynthesisConfig{})

	if err1 != nil || err2 != nil {
		t.Errorf("First two calls should succeed: err1=%v, err2=%v", err1, err2)
	}

	// Third should fail
	_, err3 := m.Synthesize(context.Background(), "test3", tts.SynthesisConfig{})
	if err3 != ErrMockQuotaExceeded {
		t.Errorf("Third call error = %v, want ErrMockQuotaExceeded", err3)
	}
}

func TestNewElevenLabsMock(t *testing.T) {
	m := NewElevenLabsMock()

	if m.Name() != "elevenlabs" {
		t.Errorf("Name() = %s, want elevenlabs", m.Name())
	}

	voices, _ := m.ListVoices(context.Background())
	if len(voices) == 0 {
		t.Error("Should have voices")
	}
	for _, v := range voices {
		if v.Provider != "elevenlabs" {
			t.Errorf("Voice provider = %s, want elevenlabs", v.Provider)
		}
	}
}

func TestNewDeepgramMock(t *testing.T) {
	m := NewDeepgramMock()

	if m.Name() != "deepgram" {
		t.Errorf("Name() = %s, want deepgram", m.Name())
	}

	voices, _ := m.ListVoices(context.Background())
	if len(voices) == 0 {
		t.Error("Should have voices")
	}
}

func TestNewOpenAIMock(t *testing.T) {
	m := NewOpenAIMock()

	if m.Name() != "openai" {
		t.Errorf("Name() = %s, want openai", m.Name())
	}

	voices, _ := m.ListVoices(context.Background())
	if len(voices) != 6 {
		t.Errorf("OpenAI mock should have 6 voices, got %d", len(voices))
	}
}

func TestProviderMocks_Conformance(t *testing.T) {
	mocks := []struct {
		name string
		mock *MockProvider
	}{
		{"elevenlabs", NewElevenLabsMock()},
		{"deepgram", NewDeepgramMock()},
		{"openai", NewOpenAIMock()},
	}

	for _, tc := range mocks {
		t.Run(tc.name, func(t *testing.T) {
			// Run interface tests on each mock
			RunInterfaceTests(t, Config{
				Provider:        tc.mock,
				SkipIntegration: true,
			})
		})
	}
}

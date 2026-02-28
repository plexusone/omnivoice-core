package providertest

import (
	"context"
	"errors"
	"testing"

	"github.com/plexusone/omnivoice/tts"
)

// TestRunAll_MockProvider verifies the test suite works with the mock provider.
func TestRunAll_MockProvider(t *testing.T) {
	mock := NewMockProvider()

	RunAll(t, Config{
		Provider:          mock,
		StreamingProvider: mock,
		SkipIntegration:   false, // Mock doesn't need real API
		TestVoiceID:       "mock-voice-1",
	})
}

// TestInterfaceTests_MockProvider verifies interface tests pass with mock.
func TestInterfaceTests_MockProvider(t *testing.T) {
	mock := NewMockProvider()

	RunInterfaceTests(t, Config{
		Provider: mock,
	})
}

// TestBehaviorTests_MockProvider verifies behavior tests pass with mock.
func TestBehaviorTests_MockProvider(t *testing.T) {
	mock := NewMockProvider()

	RunBehaviorTests(t, Config{
		Provider:        mock,
		SkipIntegration: false,
		TestVoiceID:     "mock-voice-1",
	})
}

// TestIntegrationTests_MockProvider verifies integration tests pass with mock.
func TestIntegrationTests_MockProvider(t *testing.T) {
	mock := NewMockProvider()

	RunIntegrationTests(t, Config{
		Provider:          mock,
		StreamingProvider: mock,
		SkipIntegration:   false,
		TestVoiceID:       "mock-voice-1",
	})
}

// TestIntegrationTests_Skipped verifies integration tests are skipped when configured.
func TestIntegrationTests_Skipped(t *testing.T) {
	mock := NewMockProvider()

	// This should skip without running actual tests
	RunIntegrationTests(t, Config{
		Provider:        mock,
		SkipIntegration: true,
	})
}

// TestMockProvider_Name verifies mock provider name.
func TestMockProvider_Name(t *testing.T) {
	mock := NewMockProvider()
	if mock.Name() != "mock" {
		t.Errorf("Name() = %q, want %q", mock.Name(), "mock")
	}
}

// TestMockProvider_ListVoices verifies mock provider returns voices.
func TestMockProvider_ListVoices(t *testing.T) {
	mock := NewMockProvider()
	voices, err := mock.ListVoices(context.Background())
	if err != nil {
		t.Fatalf("ListVoices() error: %v", err)
	}
	if len(voices) != 3 {
		t.Errorf("ListVoices() returned %d voices, want 3", len(voices))
	}
}

// TestMockProvider_GetVoice verifies mock provider voice lookup.
func TestMockProvider_GetVoice(t *testing.T) {
	mock := NewMockProvider()

	// Valid voice
	voice, err := mock.GetVoice(context.Background(), "mock-voice-1")
	if err != nil {
		t.Fatalf("GetVoice() error: %v", err)
	}
	if voice.ID != "mock-voice-1" {
		t.Errorf("GetVoice().ID = %q, want %q", voice.ID, "mock-voice-1")
	}

	// Invalid voice
	_, err = mock.GetVoice(context.Background(), "nonexistent")
	if !errors.Is(err, tts.ErrVoiceNotFound) {
		t.Errorf("GetVoice(nonexistent) = %v, want ErrVoiceNotFound", err)
	}
}

// TestMockProvider_Synthesize verifies mock provider synthesis.
func TestMockProvider_Synthesize(t *testing.T) {
	mock := NewMockProvider()

	result, err := mock.Synthesize(context.Background(), "Hello", tts.SynthesisConfig{
		VoiceID:      "mock-voice-1",
		OutputFormat: "mp3",
	})
	if err != nil {
		t.Fatalf("Synthesize() error: %v", err)
	}
	if len(result.Audio) == 0 {
		t.Error("Synthesize() returned empty audio")
	}
	if result.Format != "mp3" {
		t.Errorf("Synthesize().Format = %q, want %q", result.Format, "mp3")
	}

	// Verify MP3 header
	if len(result.Audio) > 3 && string(result.Audio[:3]) != "ID3" {
		t.Errorf("Synthesize() audio doesn't start with ID3 header")
	}
}

// TestMockProvider_SynthesizeStream verifies mock provider streaming.
func TestMockProvider_SynthesizeStream(t *testing.T) {
	mock := NewMockProvider()

	chunkCh, err := mock.SynthesizeStream(context.Background(), "Hello", tts.SynthesisConfig{
		VoiceID:      "mock-voice-1",
		OutputFormat: "mp3",
	})
	if err != nil {
		t.Fatalf("SynthesizeStream() error: %v", err)
	}

	var totalBytes int
	var gotFinal bool

	for chunk := range chunkCh {
		if chunk.Error != nil {
			t.Fatalf("SynthesizeStream() chunk error: %v", chunk.Error)
		}
		totalBytes += len(chunk.Audio)
		if chunk.IsFinal {
			gotFinal = true
		}
	}

	if totalBytes == 0 {
		t.Error("SynthesizeStream() received no audio bytes")
	}
	if !gotFinal {
		t.Error("SynthesizeStream() did not receive IsFinal")
	}
}

// TestMockProvider_ContextCancellation verifies mock provider respects context.
func TestMockProvider_ContextCancellation(t *testing.T) {
	mock := NewMockProvider()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := mock.Synthesize(ctx, "Hello", tts.SynthesisConfig{
		VoiceID: "mock-voice-1",
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Synthesize() with cancelled context = %v, want context.Canceled", err)
	}
}

// TestMockProvider_AudioFormats verifies mock provider returns correct format headers.
func TestMockProvider_AudioFormats(t *testing.T) {
	mock := NewMockProvider()
	ctx := context.Background()

	tests := []struct {
		format string
		header string
	}{
		{"mp3", "ID3"},
		{"wav", "RIFF"},
		{"flac", "fLaC"},
		{"opus", "OggS"},
		{"ogg", "OggS"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result, err := mock.Synthesize(ctx, "Hello", tts.SynthesisConfig{
				VoiceID:      "mock-voice-1",
				OutputFormat: tt.format,
			})
			if err != nil {
				t.Fatalf("Synthesize() error: %v", err)
			}
			if len(result.Audio) < len(tt.header) {
				t.Fatalf("Synthesize() audio too short for header check")
			}
			if string(result.Audio[:len(tt.header)]) != tt.header {
				t.Errorf("Synthesize() audio header = %q, want %q", result.Audio[:len(tt.header)], tt.header)
			}
		})
	}
}

// TestConfig_Defaults verifies Config defaults are applied.
func TestConfig_Defaults(t *testing.T) {
	cfg := Config{}.withDefaults()

	if cfg.TestText == "" {
		t.Error("withDefaults() should set TestText")
	}
	if cfg.Timeout == 0 {
		t.Error("withDefaults() should set Timeout")
	}
}

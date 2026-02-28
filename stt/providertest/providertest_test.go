package providertest

import (
	"context"
	"errors"
	"testing"

	"github.com/plexusone/omnivoice/stt"
)

// TestRunAll_MockProvider verifies the test suite works with the mock provider.
func TestRunAll_MockProvider(t *testing.T) {
	mock := NewMockProvider()

	RunAll(t, Config{
		Provider:          mock,
		StreamingProvider: mock,
		SkipIntegration:   false, // Mock doesn't need real API
		TestExpectedText:  "test",
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
	})
}

// TestIntegrationTests_MockProvider verifies integration tests pass with mock.
func TestIntegrationTests_MockProvider(t *testing.T) {
	mock := NewMockProvider()

	RunIntegrationTests(t, Config{
		Provider:          mock,
		StreamingProvider: mock,
		SkipIntegration:   false,
		TestExpectedText:  "test",
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

// TestMockProvider_Transcribe verifies mock provider transcription.
func TestMockProvider_Transcribe(t *testing.T) {
	mock := NewMockProvider()

	result, err := mock.Transcribe(context.Background(), []byte("test-audio"), stt.TranscriptionConfig{
		Language:   "en-US",
		SampleRate: 16000,
	})
	if err != nil {
		t.Fatalf("Transcribe() error: %v", err)
	}
	if result.Text == "" {
		t.Error("Transcribe() returned empty text")
	}
	if result.Language != "en-US" {
		t.Errorf("Transcribe().Language = %q, want %q", result.Language, "en-US")
	}
	if len(result.Segments) == 0 {
		t.Error("Transcribe() returned no segments")
	}
	if len(result.Segments[0].Words) == 0 {
		t.Error("Transcribe() returned no words in segment")
	}
}

// TestMockProvider_TranscribeFile verifies mock provider file transcription.
func TestMockProvider_TranscribeFile(t *testing.T) {
	mock := NewMockProvider()

	result, err := mock.TranscribeFile(context.Background(), "/path/to/audio.wav", stt.TranscriptionConfig{})
	if err != nil {
		t.Fatalf("TranscribeFile() error: %v", err)
	}
	if result.Text == "" {
		t.Error("TranscribeFile() returned empty text")
	}
}

// TestMockProvider_TranscribeURL verifies mock provider URL transcription.
func TestMockProvider_TranscribeURL(t *testing.T) {
	mock := NewMockProvider()

	result, err := mock.TranscribeURL(context.Background(), "https://example.com/audio.wav", stt.TranscriptionConfig{})
	if err != nil {
		t.Fatalf("TranscribeURL() error: %v", err)
	}
	if result.Text == "" {
		t.Error("TranscribeURL() returned empty text")
	}
}

// TestMockProvider_TranscribeStream verifies mock provider streaming.
func TestMockProvider_TranscribeStream(t *testing.T) {
	mock := NewMockProvider()

	writer, events, err := mock.TranscribeStream(context.Background(), stt.TranscriptionConfig{})
	if err != nil {
		t.Fatalf("TranscribeStream() error: %v", err)
	}

	// Write some audio
	_, writeErr := writer.Write([]byte("test-audio-data"))
	if writeErr != nil {
		t.Fatalf("Write() error: %v", writeErr)
	}

	// Close writer to signal end
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	// Collect events
	var gotSpeechStart, gotSpeechEnd, gotFinal bool
	for event := range events {
		switch event.Type {
		case stt.EventSpeechStart:
			gotSpeechStart = true
		case stt.EventSpeechEnd:
			gotSpeechEnd = true
		case stt.EventTranscript:
			if event.IsFinal {
				gotFinal = true
			}
		}
	}

	if !gotSpeechStart {
		t.Error("TranscribeStream() did not emit speech start")
	}
	if !gotSpeechEnd {
		t.Error("TranscribeStream() did not emit speech end")
	}
	if !gotFinal {
		t.Error("TranscribeStream() did not emit final transcript")
	}
}

// TestMockProvider_ContextCancellation verifies mock provider respects context.
func TestMockProvider_ContextCancellation(t *testing.T) {
	mock := NewMockProvider()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := mock.Transcribe(ctx, []byte("test"), stt.TranscriptionConfig{})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Transcribe() with cancelled context = %v, want context.Canceled", err)
	}
}

// TestConfig_Defaults verifies Config defaults are applied.
func TestConfig_Defaults(t *testing.T) {
	cfg := Config{}.withDefaults()

	if cfg.TestAudio == nil {
		t.Error("withDefaults() should set TestAudio")
	}
	if cfg.Timeout == 0 {
		t.Error("withDefaults() should set Timeout")
	}
	if cfg.TestAudioConfig.SampleRate == 0 {
		t.Error("withDefaults() should set TestAudioConfig.SampleRate")
	}
	if cfg.TestAudioConfig.Encoding == "" {
		t.Error("withDefaults() should set TestAudioConfig.Encoding")
	}
}

// TestMockProvider_StreamWriteAfterClose verifies stream handles write after close.
func TestMockProvider_StreamWriteAfterClose(t *testing.T) {
	mock := NewMockProvider()

	writer, _, err := mock.TranscribeStream(context.Background(), stt.TranscriptionConfig{})
	if err != nil {
		t.Fatalf("TranscribeStream() error: %v", err)
	}

	// Close writer
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	// Write after close should error
	_, writeErr := writer.Write([]byte("test"))
	if !errors.Is(writeErr, stt.ErrStreamClosed) {
		t.Errorf("Write() after close = %v, want ErrStreamClosed", writeErr)
	}
}

// Package providertest provides conformance tests for STT provider implementations.
//
// Provider implementations can use this package to verify they correctly implement
// the stt.Provider and stt.StreamingProvider interfaces with consistent behavior.
//
// Basic usage:
//
//	func TestConformance(t *testing.T) {
//	    p, err := New(WithAPIKey(apiKey))
//	    if err != nil {
//	        t.Fatal(err)
//	    }
//
//	    providertest.RunAll(t, providertest.Config{
//	        Provider:        p,
//	        SkipIntegration: apiKey == "",
//	        TestAudio:       testAudioBytes,
//	    })
//	}
package providertest

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/plexusone/omnivoice-core/stt"
)

// Config configures the STT conformance test suite.
type Config struct {
	// Provider is the STT provider implementation to test.
	Provider stt.Provider

	// StreamingProvider is optional; set if provider implements streaming.
	// If nil, TranscribeStream tests are skipped.
	StreamingProvider stt.StreamingProvider

	// SkipIntegration skips tests that require real API calls.
	// Set to true for unit tests without API credentials.
	SkipIntegration bool

	// TestAudio is audio data for integration tests.
	// Should contain speech that can be transcribed.
	// If nil, a default synthetic audio is used (may not work with all providers).
	TestAudio []byte

	// TestExpectedText is a word/phrase expected in the transcription.
	// Used for fuzzy matching in integration tests.
	// If empty, any non-empty transcription is accepted.
	TestExpectedText string

	// TestAudioConfig is the config matching TestAudio format.
	// Defaults to reasonable values if not specified.
	TestAudioConfig stt.TranscriptionConfig

	// TestAudioFile is a path to an audio file for TranscribeFile tests.
	// If empty, TranscribeFile test is skipped.
	TestAudioFile string

	// TestAudioURL is a URL to an audio file for TranscribeURL tests.
	// If empty, TranscribeURL test is skipped.
	TestAudioURL string

	// Timeout for individual test operations.
	// Defaults to 30 seconds if zero.
	Timeout time.Duration
}

// withDefaults returns a copy of Config with default values applied.
func (c Config) withDefaults() Config {
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	if c.TestAudio == nil {
		// Generate minimal synthetic audio data (silence)
		// Real integration tests should provide actual audio
		c.TestAudio = makeSilentAudio()
	}
	if c.TestAudioConfig.SampleRate == 0 {
		c.TestAudioConfig.SampleRate = 16000
	}
	if c.TestAudioConfig.Channels == 0 {
		c.TestAudioConfig.Channels = 1
	}
	if c.TestAudioConfig.Encoding == "" {
		c.TestAudioConfig.Encoding = "linear16"
	}
	return c
}

// RunAll runs all conformance tests for an STT provider.
func RunAll(t *testing.T, cfg Config) {
	t.Helper()
	cfg = cfg.withDefaults()

	// Interface tests (always run)
	t.Run("Interface", func(t *testing.T) {
		RunInterfaceTests(t, cfg)
	})

	// Behavior tests (always run)
	t.Run("Behavior", func(t *testing.T) {
		RunBehaviorTests(t, cfg)
	})

	// Integration tests (skipped if no API)
	if !cfg.SkipIntegration {
		t.Run("Integration", func(t *testing.T) {
			RunIntegrationTests(t, cfg)
		})
	}
}

// RunInterfaceTests runs only interface compliance tests.
// These tests verify the provider correctly implements the interface contract
// and do not require API credentials.
func RunInterfaceTests(t *testing.T, cfg Config) {
	t.Helper()
	cfg = cfg.withDefaults()

	t.Run("Name", func(t *testing.T) { testName(t, cfg) })
}

// RunBehaviorTests runs only behavioral contract tests.
// These tests verify edge case handling and may require API credentials
// depending on the provider implementation.
func RunBehaviorTests(t *testing.T, cfg Config) {
	t.Helper()
	cfg = cfg.withDefaults()

	t.Run("Transcribe_EmptyAudio", func(t *testing.T) { testTranscribeEmptyAudio(t, cfg) })
	t.Run("Context_Cancellation", func(t *testing.T) { testContextCancellation(t, cfg) })
}

// RunIntegrationTests runs only integration tests (requires API).
// These tests verify actual transcription functionality.
func RunIntegrationTests(t *testing.T, cfg Config) {
	t.Helper()
	cfg = cfg.withDefaults()

	if cfg.SkipIntegration {
		t.Skip("integration tests skipped")
	}
	t.Run("Transcribe", func(t *testing.T) { testTranscribe(t, cfg) })
	if cfg.TestAudioFile != "" {
		t.Run("TranscribeFile", func(t *testing.T) { testTranscribeFile(t, cfg) })
	}
	if cfg.TestAudioURL != "" {
		t.Run("TranscribeURL", func(t *testing.T) { testTranscribeURL(t, cfg) })
	}
	if cfg.StreamingProvider != nil {
		t.Run("TranscribeStream", func(t *testing.T) { testTranscribeStream(t, cfg) })
	}
}

// Interface Tests

func testName(t *testing.T, cfg Config) {
	t.Helper()
	name := cfg.Provider.Name()
	if name == "" {
		t.Error("Name() returned empty string")
	}
	// Verify name is lowercase, alphanumeric with hyphens/underscores
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			t.Errorf("Name() contains invalid character %q; should be lowercase alphanumeric with hyphens/underscores", r)
		}
	}
}

// Behavior Tests

func testTranscribeEmptyAudio(t *testing.T, cfg Config) {
	t.Helper()
	if cfg.SkipIntegration {
		t.Skip("skipping behavior test that may require API")
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Empty audio should either:
	// 1. Return an error (ErrInvalidAudio, ErrAudioTooShort, or similar)
	// 2. Return empty transcription (acceptable)
	// It should NOT panic or hang
	result, err := cfg.Provider.Transcribe(ctx, []byte{}, cfg.TestAudioConfig)
	if err != nil {
		// Error is acceptable for empty audio
		t.Logf("Transcribe(empty) returned error (acceptable): %v", err)
		return
	}
	// Empty result is also acceptable
	t.Logf("Transcribe(empty) returned %q (acceptable)", result.Text)
}

func testContextCancellation(t *testing.T, cfg Config) {
	t.Helper()
	if cfg.SkipIntegration {
		t.Skip("skipping context cancellation test that requires API")
	}

	// Create already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should return quickly with context error
	_, err := cfg.Provider.Transcribe(ctx, cfg.TestAudio, cfg.TestAudioConfig)
	if err == nil {
		t.Error("Transcribe with cancelled context should return error")
	}
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		// Some providers wrap the error
		t.Logf("Transcribe with cancelled context returned: %v (should contain context error)", err)
	}
}

// Integration Tests

func testTranscribe(t *testing.T, cfg Config) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	result, err := cfg.Provider.Transcribe(ctx, cfg.TestAudio, cfg.TestAudioConfig)
	if err != nil {
		t.Fatalf("Transcribe() error: %v", err)
	}

	if result == nil {
		t.Fatal("Transcribe() returned nil result")
	}

	// Check for expected text if specified
	if cfg.TestExpectedText != "" {
		if !strings.Contains(strings.ToLower(result.Text), strings.ToLower(cfg.TestExpectedText)) {
			t.Errorf("Transcribe() text %q does not contain expected %q", result.Text, cfg.TestExpectedText)
		}
	}

	t.Logf("Transcribe() returned: %q", result.Text)
}

func testTranscribeFile(t *testing.T, cfg Config) {
	t.Helper()
	if cfg.TestAudioFile == "" {
		t.Skip("TestAudioFile not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	result, err := cfg.Provider.TranscribeFile(ctx, cfg.TestAudioFile, cfg.TestAudioConfig)
	if err != nil {
		t.Fatalf("TranscribeFile() error: %v", err)
	}

	if result == nil {
		t.Fatal("TranscribeFile() returned nil result")
	}

	// Check for expected text if specified
	if cfg.TestExpectedText != "" {
		if !strings.Contains(strings.ToLower(result.Text), strings.ToLower(cfg.TestExpectedText)) {
			t.Errorf("TranscribeFile() text %q does not contain expected %q", result.Text, cfg.TestExpectedText)
		}
	}

	t.Logf("TranscribeFile() returned: %q", result.Text)
}

func testTranscribeURL(t *testing.T, cfg Config) {
	t.Helper()
	if cfg.TestAudioURL == "" {
		t.Skip("TestAudioURL not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	result, err := cfg.Provider.TranscribeURL(ctx, cfg.TestAudioURL, cfg.TestAudioConfig)
	if err != nil {
		t.Fatalf("TranscribeURL() error: %v", err)
	}

	if result == nil {
		t.Fatal("TranscribeURL() returned nil result")
	}

	// Check for expected text if specified
	if cfg.TestExpectedText != "" {
		if !strings.Contains(strings.ToLower(result.Text), strings.ToLower(cfg.TestExpectedText)) {
			t.Errorf("TranscribeURL() text %q does not contain expected %q", result.Text, cfg.TestExpectedText)
		}
	}

	t.Logf("TranscribeURL() returned: %q", result.Text)
}

func testTranscribeStream(t *testing.T, cfg Config) {
	t.Helper()
	if cfg.StreamingProvider == nil {
		t.Skip("provider does not implement StreamingProvider")
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	writer, events, err := cfg.StreamingProvider.TranscribeStream(ctx, cfg.TestAudioConfig)
	if err != nil {
		t.Fatalf("TranscribeStream() error: %v", err)
	}

	// Write audio in a goroutine
	done := make(chan error, 1)
	go func() {
		_, writeErr := writer.Write(cfg.TestAudio)
		if writeErr != nil {
			done <- writeErr
			return
		}
		done <- writer.Close()
	}()

	// Collect events
	var transcripts []string
	var gotSpeechStart, gotSpeechEnd, gotFinal bool
	var streamErr error

	eventLoop := true
	for eventLoop {
		select {
		case event, ok := <-events:
			if !ok {
				eventLoop = false
				break
			}
			switch event.Type {
			case stt.EventTranscript:
				if event.IsFinal {
					transcripts = append(transcripts, event.Transcript)
					gotFinal = true
				}
			case stt.EventSpeechStart:
				gotSpeechStart = true
			case stt.EventSpeechEnd:
				gotSpeechEnd = true
			case stt.EventError:
				streamErr = event.Error
				eventLoop = false
			}
		case writeErr := <-done:
			if writeErr != nil {
				t.Fatalf("Write error: %v", writeErr)
			}
			// Continue reading events until channel closes
		case <-ctx.Done():
			eventLoop = false
		}
	}

	if streamErr != nil {
		t.Fatalf("TranscribeStream() stream error: %v", streamErr)
	}

	// Log results
	t.Logf("TranscribeStream() got final=%v, speechStart=%v, speechEnd=%v", gotFinal, gotSpeechStart, gotSpeechEnd)
	if len(transcripts) > 0 {
		t.Logf("TranscribeStream() transcripts: %v", transcripts)
	}

	// Validate expected text if specified
	if cfg.TestExpectedText != "" && gotFinal {
		fullText := strings.Join(transcripts, " ")
		if !strings.Contains(strings.ToLower(fullText), strings.ToLower(cfg.TestExpectedText)) {
			t.Errorf("TranscribeStream() text %q does not contain expected %q", fullText, cfg.TestExpectedText)
		}
	}
}

// Helper functions

// makeSilentAudio generates minimal silent audio data in linear16 format.
// This is a fallback for tests - real integration tests should provide actual audio.
func makeSilentAudio() []byte {
	// Generate 1 second of silence at 16kHz mono linear16
	// Each sample is 2 bytes (16-bit), 16000 samples per second
	samples := 16000
	audio := make([]byte, samples*2)
	return audio
}

// Package providertest provides conformance tests for TTS provider implementations.
//
// Provider implementations can use this package to verify they correctly implement
// the tts.Provider and tts.StreamingProvider interfaces with consistent behavior.
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
//	        TestVoiceID:     "voice-id",
//	    })
//	}
package providertest

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/agentplexus/omnivoice/tts"
)

// Config configures the TTS conformance test suite.
type Config struct {
	// Provider is the TTS provider implementation to test.
	Provider tts.Provider

	// StreamingProvider is optional; set if provider implements streaming input.
	// If nil, SynthesizeFromReader tests are skipped.
	StreamingProvider tts.StreamingProvider

	// SkipIntegration skips tests that require real API calls.
	// Set to true for unit tests without API credentials.
	SkipIntegration bool

	// TestVoiceID is a valid voice ID for integration tests.
	// Required if SkipIntegration is false.
	TestVoiceID string

	// TestText is the text to synthesize in integration tests.
	// Defaults to "Hello, this is a test." if empty.
	TestText string

	// Timeout for individual test operations.
	// Defaults to 30 seconds if zero.
	Timeout time.Duration
}

// withDefaults returns a copy of Config with default values applied.
func (c Config) withDefaults() Config {
	if c.TestText == "" {
		c.TestText = "Hello, this is a test."
	}
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	return c
}

// RunAll runs all conformance tests for a TTS provider.
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
	t.Run("ListVoices", func(t *testing.T) { testListVoices(t, cfg) })
	t.Run("GetVoice", func(t *testing.T) { testGetVoice(t, cfg) })
	t.Run("GetVoice_NotFound", func(t *testing.T) { testGetVoiceNotFound(t, cfg) })
}

// RunBehaviorTests runs only behavioral contract tests.
// These tests verify edge case handling and may require API credentials
// depending on the provider implementation.
func RunBehaviorTests(t *testing.T, cfg Config) {
	t.Helper()
	cfg = cfg.withDefaults()

	t.Run("Synthesize_EmptyText", func(t *testing.T) { testSynthesizeEmptyText(t, cfg) })
	t.Run("Context_Cancellation", func(t *testing.T) { testContextCancellation(t, cfg) })
}

// RunIntegrationTests runs only integration tests (requires API).
// These tests verify actual synthesis functionality.
func RunIntegrationTests(t *testing.T, cfg Config) {
	t.Helper()
	cfg = cfg.withDefaults()

	if cfg.SkipIntegration {
		t.Skip("integration tests skipped")
	}
	t.Run("Synthesize", func(t *testing.T) { testSynthesize(t, cfg) })
	t.Run("SynthesizeStream", func(t *testing.T) { testSynthesizeStream(t, cfg) })
	if cfg.StreamingProvider != nil {
		t.Run("SynthesizeFromReader", func(t *testing.T) { testSynthesizeFromReader(t, cfg) })
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

func testListVoices(t *testing.T, cfg Config) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	voices, err := cfg.Provider.ListVoices(ctx)
	if err != nil {
		t.Fatalf("ListVoices() error: %v", err)
	}
	if len(voices) == 0 {
		t.Error("ListVoices() returned empty list")
	}

	for i, v := range voices {
		if v.ID == "" {
			t.Errorf("voices[%d].ID is empty", i)
		}
		if v.Name == "" {
			t.Errorf("voices[%d].Name is empty (ID: %s)", i, v.ID)
		}
		if v.Provider == "" {
			t.Errorf("voices[%d].Provider is empty (ID: %s)", i, v.ID)
		}
		if v.Provider != cfg.Provider.Name() {
			t.Errorf("voices[%d].Provider = %q, want %q", i, v.Provider, cfg.Provider.Name())
		}
	}
}

func testGetVoice(t *testing.T, cfg Config) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// First get a valid voice ID from ListVoices
	voices, err := cfg.Provider.ListVoices(ctx)
	if err != nil {
		t.Fatalf("ListVoices() error: %v", err)
	}
	if len(voices) == 0 {
		t.Skip("no voices available to test GetVoice")
	}

	voiceID := voices[0].ID
	voice, err := cfg.Provider.GetVoice(ctx, voiceID)
	if err != nil {
		t.Fatalf("GetVoice(%q) error: %v", voiceID, err)
	}
	if voice == nil {
		t.Fatalf("GetVoice(%q) returned nil voice", voiceID)
	}
	if voice.ID != voiceID {
		t.Errorf("GetVoice(%q).ID = %q", voiceID, voice.ID)
	}
}

func testGetVoiceNotFound(t *testing.T, cfg Config) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	_, err := cfg.Provider.GetVoice(ctx, "nonexistent-voice-id-xyz-12345")
	if err == nil {
		t.Error("GetVoice(nonexistent) should return error")
	}
	if !errors.Is(err, tts.ErrVoiceNotFound) {
		t.Errorf("GetVoice(nonexistent) = %v, want ErrVoiceNotFound", err)
	}
}

// Behavior Tests

func testSynthesizeEmptyText(t *testing.T, cfg Config) {
	t.Helper()
	if cfg.SkipIntegration {
		t.Skip("skipping behavior test that may require API")
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	config := tts.SynthesisConfig{VoiceID: cfg.TestVoiceID}

	// Empty text should either:
	// 1. Return an error (ErrInvalidConfig or similar)
	// 2. Return empty audio (acceptable)
	// It should NOT panic or hang
	result, err := cfg.Provider.Synthesize(ctx, "", config)
	if err != nil {
		// Error is acceptable for empty text
		t.Logf("Synthesize(\"\") returned error (acceptable): %v", err)
		return
	}
	// Empty result is also acceptable
	t.Logf("Synthesize(\"\") returned %d bytes (acceptable)", len(result.Audio))
}

func testContextCancellation(t *testing.T, cfg Config) {
	t.Helper()
	if cfg.SkipIntegration {
		t.Skip("skipping context cancellation test that requires API")
	}

	// Create already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	config := tts.SynthesisConfig{VoiceID: cfg.TestVoiceID}

	// Should return quickly with context error
	_, err := cfg.Provider.Synthesize(ctx, cfg.TestText, config)
	if err == nil {
		t.Error("Synthesize with cancelled context should return error")
	}
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		// Some providers wrap the error
		t.Logf("Synthesize with cancelled context returned: %v (should contain context error)", err)
	}
}

// Integration Tests

func testSynthesize(t *testing.T, cfg Config) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	config := tts.SynthesisConfig{
		VoiceID:      cfg.TestVoiceID,
		OutputFormat: "mp3",
	}

	result, err := cfg.Provider.Synthesize(ctx, cfg.TestText, config)
	if err != nil {
		t.Fatalf("Synthesize() error: %v", err)
	}

	if result == nil {
		t.Fatal("Synthesize() returned nil result")
	}
	if len(result.Audio) == 0 {
		t.Error("Synthesize() returned empty audio")
	}
	if result.Format == "" {
		t.Error("Synthesize() returned empty format")
	}

	// Basic audio validation - check for common headers
	if len(result.Audio) > 4 {
		validateAudioFormat(t, result.Audio, config.OutputFormat)
	}

	t.Logf("Synthesize() returned %d bytes of %s audio", len(result.Audio), result.Format)
}

func testSynthesizeStream(t *testing.T, cfg Config) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Use linear16 for streaming tests since some providers (e.g., Deepgram)
	// don't support mp3 for WebSocket streaming
	config := tts.SynthesisConfig{
		VoiceID:      cfg.TestVoiceID,
		OutputFormat: "linear16",
	}

	chunkCh, err := cfg.Provider.SynthesizeStream(ctx, cfg.TestText, config)
	if err != nil {
		t.Fatalf("SynthesizeStream() error: %v", err)
	}

	var totalBytes int
	var chunkCount int
	var gotFinal bool
	var streamErr error

	for chunk := range chunkCh {
		if chunk.Error != nil {
			streamErr = chunk.Error
			break
		}
		if len(chunk.Audio) > 0 {
			totalBytes += len(chunk.Audio)
			chunkCount++
		}
		if chunk.IsFinal {
			gotFinal = true
		}
	}

	if streamErr != nil {
		t.Fatalf("SynthesizeStream() stream error: %v", streamErr)
	}
	if totalBytes == 0 {
		t.Error("SynthesizeStream() received no audio bytes")
	}
	if chunkCount == 0 {
		t.Error("SynthesizeStream() received no audio chunks")
	}
	if !gotFinal {
		t.Error("SynthesizeStream() did not receive IsFinal chunk")
	}

	t.Logf("SynthesizeStream() received %d bytes in %d chunks", totalBytes, chunkCount)
}

func testSynthesizeFromReader(t *testing.T, cfg Config) {
	t.Helper()
	if cfg.StreamingProvider == nil {
		t.Skip("provider does not implement StreamingProvider")
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Use linear16 for streaming tests since some providers (e.g., Deepgram)
	// don't support mp3 for WebSocket streaming
	config := tts.SynthesisConfig{
		VoiceID:      cfg.TestVoiceID,
		OutputFormat: "linear16",
	}

	// Use a simple string reader as input
	reader := &stringReader{text: cfg.TestText}

	chunkCh, err := cfg.StreamingProvider.SynthesizeFromReader(ctx, reader, config)
	if err != nil {
		t.Fatalf("SynthesizeFromReader() error: %v", err)
	}

	var totalBytes int
	var chunkCount int
	var gotFinal bool
	var streamErr error

	for chunk := range chunkCh {
		if chunk.Error != nil {
			streamErr = chunk.Error
			break
		}
		if len(chunk.Audio) > 0 {
			totalBytes += len(chunk.Audio)
			chunkCount++
		}
		if chunk.IsFinal {
			gotFinal = true
		}
	}

	if streamErr != nil {
		t.Fatalf("SynthesizeFromReader() stream error: %v", streamErr)
	}
	if totalBytes == 0 {
		t.Error("SynthesizeFromReader() received no audio bytes")
	}
	if chunkCount == 0 {
		t.Error("SynthesizeFromReader() received no audio chunks")
	}
	if !gotFinal {
		t.Error("SynthesizeFromReader() did not receive IsFinal chunk")
	}

	t.Logf("SynthesizeFromReader() received %d bytes in %d chunks", totalBytes, chunkCount)
}

// Helper types and functions

// stringReader is a simple io.Reader that returns text.
type stringReader struct {
	text string
	pos  int
}

func (r *stringReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.text) {
		return 0, io.EOF
	}
	n := copy(p, r.text[r.pos:])
	r.pos += n
	return n, nil
}

// validateAudioFormat performs basic validation of audio data format.
func validateAudioFormat(t *testing.T, audio []byte, expectedFormat string) {
	t.Helper()

	if len(audio) < 4 {
		return
	}

	// MP3 files start with ID3 or 0xFF 0xFB
	isMP3 := (audio[0] == 'I' && audio[1] == 'D' && audio[2] == '3') ||
		(audio[0] == 0xFF && (audio[1]&0xF0) == 0xF0)

	// WAV files start with RIFF
	isWAV := audio[0] == 'R' && audio[1] == 'I' && audio[2] == 'F' && audio[3] == 'F'

	// FLAC files start with fLaC
	isFLAC := audio[0] == 'f' && audio[1] == 'L' && audio[2] == 'a' && audio[3] == 'C'

	// Ogg/Opus files start with OggS
	isOgg := audio[0] == 'O' && audio[1] == 'g' && audio[2] == 'g' && audio[3] == 'S'

	switch expectedFormat {
	case "mp3":
		if !isMP3 {
			t.Logf("Warning: requested mp3 but audio doesn't look like MP3 (first 4 bytes: %x)", audio[:4])
		}
	case "wav":
		if !isWAV {
			t.Logf("Warning: requested wav but audio doesn't look like WAV (first 4 bytes: %x)", audio[:4])
		}
	case "flac":
		if !isFLAC {
			t.Logf("Warning: requested flac but audio doesn't look like FLAC (first 4 bytes: %x)", audio[:4])
		}
	case "opus", "ogg":
		if !isOgg {
			t.Logf("Warning: requested %s but audio doesn't look like Ogg (first 4 bytes: %x)", expectedFormat, audio[:4])
		}
	}
}

# Technical Design: Provider Conformance Testing

This document describes the design for standardized conformance testing packages that OmniVoice provider implementations can use to verify interface compliance and behavior consistency.

## Goals

1. Provide reusable test suites for TTS and STT provider implementations
2. Ensure consistent behavior across different providers (Deepgram, ElevenLabs, etc.)
3. Catch interface violations and edge cases early
4. Serve as executable documentation for provider implementers

## Package Structure

```
omnivoice/
├── tts/
│   ├── tts.go                    # Interfaces (existing)
│   ├── errors.go                 # Errors (existing)
│   └── providertest/             # NEW: Conformance tests
│       ├── providertest.go       # Test suite runner and config
│       ├── interface_test.go     # Interface compliance tests
│       ├── behavior_test.go      # Behavioral contract tests
│       ├── integration_test.go   # Integration test helpers
│       └── mock.go               # Mock provider for self-testing
├── stt/
│   ├── stt.go                    # Interfaces (existing)
│   ├── errors.go                 # Errors (existing)
│   └── providertest/             # NEW: Conformance tests
│       ├── providertest.go       # Test suite runner and config
│       ├── interface_test.go     # Interface compliance tests
│       ├── behavior_test.go      # Behavioral contract tests
│       ├── integration_test.go   # Integration test helpers
│       └── mock.go               # Mock provider for self-testing
```

## Test Categories

### 1. Interface Tests (No API Required)

Verify that providers correctly implement the interface contract:

| Test | TTS | STT | Description |
|------|-----|-----|-------------|
| `Name` | ✓ | ✓ | Returns non-empty provider name |
| `ListVoices` | ✓ | — | Returns non-empty voice list with required fields |
| `GetVoice` | ✓ | — | Returns voice for valid ID |
| `GetVoice_NotFound` | ✓ | — | Returns `ErrVoiceNotFound` for invalid ID |

### 2. Behavior Tests (No API Required)

Verify edge case handling and error conditions:

| Test | TTS | STT | Description |
|------|-----|-----|-------------|
| `Synthesize_EmptyText` | ✓ | — | Handles empty text input gracefully |
| `Synthesize_InvalidConfig` | ✓ | — | Returns `ErrInvalidConfig` for bad config |
| `Transcribe_EmptyAudio` | — | ✓ | Handles empty audio input gracefully |
| `Context_Cancellation` | ✓ | ✓ | Respects context cancellation |

### 3. Integration Tests (API Required)

Verify actual synthesis/transcription works:

| Test | TTS | STT | Description |
|------|-----|-----|-------------|
| `Synthesize` | ✓ | — | Returns valid audio bytes |
| `SynthesizeStream` | ✓ | — | Streams audio chunks correctly |
| `SynthesizeFromReader` | ✓ | — | Handles streaming input |
| `Transcribe` | — | ✓ | Returns valid transcription |
| `TranscribeStream` | — | ✓ | Streams transcription events |

## API Design

### TTS Provider Test Suite

```go
package providertest

import (
    "context"
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
func RunInterfaceTests(t *testing.T, cfg Config) {
    t.Helper()
    t.Run("Name", func(t *testing.T) { testName(t, cfg) })
    t.Run("ListVoices", func(t *testing.T) { testListVoices(t, cfg) })
    t.Run("GetVoice", func(t *testing.T) { testGetVoice(t, cfg) })
    t.Run("GetVoice_NotFound", func(t *testing.T) { testGetVoiceNotFound(t, cfg) })
}

// RunBehaviorTests runs only behavioral contract tests.
func RunBehaviorTests(t *testing.T, cfg Config) {
    t.Helper()
    t.Run("Synthesize_EmptyText", func(t *testing.T) { testSynthesizeEmptyText(t, cfg) })
    t.Run("Context_Cancellation", func(t *testing.T) { testContextCancellation(t, cfg) })
}

// RunIntegrationTests runs only integration tests (requires API).
func RunIntegrationTests(t *testing.T, cfg Config) {
    t.Helper()
    if cfg.SkipIntegration {
        t.Skip("integration tests skipped")
    }
    t.Run("Synthesize", func(t *testing.T) { testSynthesize(t, cfg) })
    t.Run("SynthesizeStream", func(t *testing.T) { testSynthesizeStream(t, cfg) })
    if cfg.StreamingProvider != nil {
        t.Run("SynthesizeFromReader", func(t *testing.T) { testSynthesizeFromReader(t, cfg) })
    }
}
```

### STT Provider Test Suite

```go
package providertest

import (
    "context"
    "testing"
    "time"

    "github.com/agentplexus/omnivoice/stt"
)

// Config configures the STT conformance test suite.
type Config struct {
    // Provider is the STT provider implementation to test.
    Provider stt.Provider

    // StreamingProvider is optional; set if provider implements streaming.
    // If nil, TranscribeStream tests are skipped.
    StreamingProvider stt.StreamingProvider

    // SkipIntegration skips tests that require real API calls.
    SkipIntegration bool

    // TestAudio is audio data for integration tests.
    // Should contain speech saying TestExpectedText.
    // If nil, a default test audio file is used.
    TestAudio []byte

    // TestExpectedText is the expected transcription of TestAudio.
    // Used for fuzzy matching in integration tests.
    TestExpectedText string

    // TestAudioConfig is the config matching TestAudio format.
    TestAudioConfig stt.TranscriptionConfig

    // Timeout for individual test operations.
    Timeout time.Duration
}

// RunAll runs all conformance tests for an STT provider.
func RunAll(t *testing.T, cfg Config) {
    t.Helper()
    cfg = cfg.withDefaults()

    t.Run("Interface", func(t *testing.T) {
        RunInterfaceTests(t, cfg)
    })

    t.Run("Behavior", func(t *testing.T) {
        RunBehaviorTests(t, cfg)
    })

    if !cfg.SkipIntegration {
        t.Run("Integration", func(t *testing.T) {
            RunIntegrationTests(t, cfg)
        })
    }
}
```

## Provider Usage

### Deepgram TTS Provider

```go
// omnivoice-deepgram/omnivoice/tts/provider_conformance_test.go
package tts

import (
    "os"
    "testing"

    "github.com/agentplexus/omnivoice/tts/providertest"
)

func TestConformance(t *testing.T) {
    apiKey := os.Getenv("DEEPGRAM_API_KEY")

    p, err := New(WithAPIKey(apiKey))
    if err != nil && apiKey == "" {
        t.Skip("DEEPGRAM_API_KEY not set, skipping conformance tests")
    }
    if err != nil {
        t.Fatalf("failed to create provider: %v", err)
    }

    providertest.RunAll(t, providertest.Config{
        Provider:          p,
        StreamingProvider: p, // Provider implements StreamingProvider
        SkipIntegration:   apiKey == "",
        TestVoiceID:       "aura-asteria-en",
    })
}
```

### ElevenLabs TTS Provider

```go
// go-elevenlabs/omnivoice/tts/provider_conformance_test.go
package tts

import (
    "os"
    "testing"

    "github.com/agentplexus/omnivoice/tts/providertest"
)

func TestConformance(t *testing.T) {
    apiKey := os.Getenv("ELEVENLABS_API_KEY")

    p, err := New(WithAPIKey(apiKey))
    if err != nil && apiKey == "" {
        t.Skip("ELEVENLABS_API_KEY not set, skipping conformance tests")
    }
    if err != nil {
        t.Fatalf("failed to create provider: %v", err)
    }

    providertest.RunAll(t, providertest.Config{
        Provider:          p,
        StreamingProvider: p,
        SkipIntegration:   apiKey == "",
        TestVoiceID:       "21m00Tcm4TlvDq8ikWAM", // Rachel voice
    })
}
```

## Test Implementation Details

### Interface Tests

```go
func testName(t *testing.T, cfg Config) {
    t.Helper()
    name := cfg.Provider.Name()
    if name == "" {
        t.Error("Name() returned empty string")
    }
    // Verify name is lowercase, alphanumeric
    for _, r := range name {
        if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
            t.Errorf("Name() contains invalid character %q; should be lowercase alphanumeric", r)
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
    if err != tts.ErrVoiceNotFound {
        t.Errorf("GetVoice(nonexistent) = %v, want ErrVoiceNotFound", err)
    }
}
```

### Behavior Tests

```go
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
    if err != context.Canceled && err != context.DeadlineExceeded {
        // Some providers wrap the error
        t.Logf("Synthesize with cancelled context returned: %v (should contain context error)", err)
    }
}
```

### Integration Tests

```go
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
        // MP3 files start with ID3 or 0xFF 0xFB
        isMP3 := (result.Audio[0] == 'I' && result.Audio[1] == 'D' && result.Audio[2] == '3') ||
                 (result.Audio[0] == 0xFF && (result.Audio[1]&0xF0) == 0xF0)
        // WAV files start with RIFF
        isWAV := result.Audio[0] == 'R' && result.Audio[1] == 'I' && result.Audio[2] == 'F' && result.Audio[3] == 'F'

        if config.OutputFormat == "mp3" && !isMP3 {
            t.Logf("Warning: requested mp3 but audio doesn't look like MP3 (first 4 bytes: %x)", result.Audio[:4])
        }
        if config.OutputFormat == "wav" && !isWAV {
            t.Logf("Warning: requested wav but audio doesn't look like WAV (first 4 bytes: %x)", result.Audio[:4])
        }
    }

    t.Logf("Synthesize() returned %d bytes of %s audio", len(result.Audio), result.Format)
}

func testSynthesizeStream(t *testing.T, cfg Config) {
    t.Helper()
    ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
    defer cancel()

    config := tts.SynthesisConfig{
        VoiceID:      cfg.TestVoiceID,
        OutputFormat: "mp3",
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
            cancel() // Signal completion
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
```

## Mock Provider

For self-testing the test suite:

```go
// mock.go
package providertest

import (
    "context"
    "github.com/agentplexus/omnivoice/tts"
)

// MockProvider is a minimal TTS provider for testing the test suite itself.
type MockProvider struct {
    voices []tts.Voice
}

func NewMockProvider() *MockProvider {
    return &MockProvider{
        voices: []tts.Voice{
            {ID: "mock-voice-1", Name: "Mock Voice 1", Language: "en-US", Gender: "female", Provider: "mock"},
            {ID: "mock-voice-2", Name: "Mock Voice 2", Language: "en-US", Gender: "male", Provider: "mock"},
        },
    }
}

func (m *MockProvider) Name() string { return "mock" }

func (m *MockProvider) ListVoices(ctx context.Context) ([]tts.Voice, error) {
    return m.voices, nil
}

func (m *MockProvider) GetVoice(ctx context.Context, voiceID string) (*tts.Voice, error) {
    for _, v := range m.voices {
        if v.ID == voiceID {
            return &v, nil
        }
    }
    return nil, tts.ErrVoiceNotFound
}

func (m *MockProvider) Synthesize(ctx context.Context, text string, config tts.SynthesisConfig) (*tts.SynthesisResult, error) {
    // Return fake MP3 data
    return &tts.SynthesisResult{
        Audio:          []byte("ID3fake-audio-data"),
        Format:         "mp3",
        SampleRate:     24000,
        CharacterCount: len(text),
    }, nil
}

func (m *MockProvider) SynthesizeStream(ctx context.Context, text string, config tts.SynthesisConfig) (<-chan tts.StreamChunk, error) {
    ch := make(chan tts.StreamChunk, 10)
    go func() {
        defer close(ch)
        ch <- tts.StreamChunk{Audio: []byte("chunk1")}
        ch <- tts.StreamChunk{Audio: []byte("chunk2")}
        ch <- tts.StreamChunk{IsFinal: true}
    }()
    return ch, nil
}
```

## Implementation Checklist

### Phase 1: TTS Provider Tests

- [x] Create `tts/providertest/providertest.go` with Config and runners
- [x] Implement interface tests (Name, ListVoices, GetVoice)
- [x] Implement behavior tests (empty text, context cancellation)
- [x] Implement integration tests (Synthesize, SynthesizeStream)
- [x] Create mock provider for self-testing
- [x] Add tests for the test suite itself

### Phase 2: STT Provider Tests

- [x] Create `stt/providertest/providertest.go` with Config and runners
- [x] Implement interface tests (Name)
- [x] Implement behavior tests (empty audio, context cancellation)
- [x] Implement integration tests (Transcribe, TranscribeStream)
- [x] Create mock provider for self-testing
- [ ] Add test audio fixtures

### Phase 3: Provider Integration

- [ ] Add conformance tests to omnivoice-deepgram
- [ ] Add conformance tests to go-elevenlabs/omnivoice
- [ ] Document conformance testing in README

## References

- [gocloud.dev drivertest](https://github.com/google/go-cloud/tree/master/blob/drivertest) - Similar pattern for blob storage
- [golang.org/x/net/nettest](https://pkg.go.dev/golang.org/x/net/nettest) - Conformance tests for net.Conn
- [Go testing package](https://pkg.go.dev/testing) - Standard testing patterns

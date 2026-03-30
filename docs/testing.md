# Testing with Mock Providers

The `tts/providertest` package provides mock TTS providers and audio fixtures for testing your integrations without requiring API keys or network access.

## Quick Start

```go
import "github.com/plexusone/omnivoice-core/tts/providertest"

func TestMyFeature(t *testing.T) {
    // Create a mock provider
    mock := providertest.NewMockProvider()

    // Use it like a real provider
    result, err := mock.Synthesize(ctx, "Hello world", tts.SynthesisConfig{})
    if err != nil {
        t.Fatal(err)
    }

    // Result contains valid audio data
    t.Logf("Generated %d bytes of %s audio", len(result.Audio), result.Format)
}
```

## Provider-Specific Mocks

Pre-configured mocks that simulate specific provider behavior with realistic voice lists:

### ElevenLabs Mock

```go
mock := providertest.NewElevenLabsMock()

// Name() returns "elevenlabs"
// ListVoices() returns Rachel, Bella, Antoni
voices, _ := mock.ListVoices(ctx)
for _, v := range voices {
    fmt.Printf("%s: %s (%s)\n", v.ID, v.Name, v.Gender)
}
// Output:
// 21m00Tcm4TlvDq8ikWAM: Rachel (female)
// EXAVITQu4vr4xnSDxMaL: Bella (female)
// ErXwobaYiN019PkySvjV: Antoni (male)
```

### Deepgram Mock

```go
mock := providertest.NewDeepgramMock()

// Name() returns "deepgram"
// ListVoices() returns Asteria, Luna, Orion
```

### OpenAI Mock

```go
mock := providertest.NewOpenAIMock()

// Name() returns "openai"
// ListVoices() returns all 6 OpenAI TTS voices:
// Alloy, Echo, Fable, Onyx, Nova, Shimmer
```

## Configurable Mock Behaviors

Use `MockProviderOption` functions to customize mock behavior:

### Fixed Duration Audio

```go
// Always return 2-second audio regardless of text length
mock := providertest.NewMockProviderWithOptions(
    providertest.WithFixedDuration(2000), // milliseconds
)
```

### Realistic Timing

```go
// Duration proportional to text length (~50ms per character)
mock := providertest.NewMockProviderWithOptions(
    providertest.WithRealisticTiming(),
)

// Short text = short audio
r1, _ := mock.Synthesize(ctx, "Hi", config)       // ~100ms (minimum)

// Long text = longer audio
r2, _ := mock.Synthesize(ctx, "Hello world!", config)  // ~600ms
```

### Simulating Latency

```go
// Add 100ms delay to each call
mock := providertest.NewMockProviderWithOptions(
    providertest.WithLatency(100 * time.Millisecond),
)
```

### Error Injection

```go
// Always return an error
mock := providertest.NewMockProviderWithOptions(
    providertest.WithError(providertest.ErrMockRateLimit),
)

_, err := mock.Synthesize(ctx, "test", config)
// err == ErrMockRateLimit
```

Available error types:

| Error | Description |
|-------|-------------|
| `ErrMockRateLimit` | Simulates rate limiting |
| `ErrMockQuotaExceeded` | Simulates quota exhaustion |
| `ErrMockNetworkError` | Simulates network failures |
| `ErrMockInvalidVoice` | Simulates invalid voice ID |

### Fail After N Calls

```go
// Succeed 3 times, then fail (for testing retry/failover logic)
mock := providertest.NewMockProviderWithOptions(
    providertest.WithFailAfterN(3, providertest.ErrMockQuotaExceeded),
)

mock.Synthesize(ctx, "1", config) // Success
mock.Synthesize(ctx, "2", config) // Success
mock.Synthesize(ctx, "3", config) // Success
mock.Synthesize(ctx, "4", config) // Error: quota exceeded
```

### Combining Options

```go
mock := providertest.NewMockProviderWithOptions(
    providertest.WithName("my-provider"),
    providertest.WithRealisticTiming(),
    providertest.WithLatency(50 * time.Millisecond),
)
```

## WAV Audio Fixtures

Generate valid WAV files for testing audio processing:

### Basic Usage

```go
// Generate 1-second WAV at 22050 Hz
fixture := providertest.GenerateWAVFixture(1000, 22050)

fmt.Printf("Format: %s\n", fixture.Format)         // "wav"
fmt.Printf("Duration: %dms\n", fixture.DurationMs) // 1000
fmt.Printf("Sample Rate: %d\n", fixture.SampleRate) // 22050
fmt.Printf("Size: %d bytes\n", len(fixture.Data))  // 44144
```

### Convenience Functions

```go
// 100ms WAV at 22050 Hz (quick tests)
short := providertest.GenerateShortWAV()

// 1 second WAV at 22050 Hz
oneSecond := providertest.GenerateOneSecondWAV()
```

### WAV Format Details

Generated WAV files are valid PCM audio:

- **Format**: PCM (uncompressed)
- **Channels**: 1 (mono)
- **Bit Depth**: 16-bit
- **Content**: Silence (zeros)
- **Header**: Full 44-byte RIFF/WAV header

The audio can be parsed by ffprobe, ffmpeg, and other audio tools.

### Verifying WAV Headers

```go
fixture := providertest.GenerateWAVFixture(1000, 22050)

// Check RIFF header
if string(fixture.Data[0:4]) != "RIFF" {
    t.Error("Invalid WAV: missing RIFF header")
}

// Check WAVE format
if string(fixture.Data[8:12]) != "WAVE" {
    t.Error("Invalid WAV: missing WAVE format")
}
```

## Common Test Patterns

### Testing TTS Integration

```go
func TestTTSClient(t *testing.T) {
    mock := providertest.NewElevenLabsMock()
    client := mytts.NewClient(mock)

    audio, err := client.TextToSpeech(ctx, "Hello")
    if err != nil {
        t.Fatalf("TextToSpeech failed: %v", err)
    }

    if len(audio) == 0 {
        t.Error("Expected audio data")
    }
}
```

### Testing Error Handling

```go
func TestRateLimitHandling(t *testing.T) {
    mock := providertest.NewMockProviderWithOptions(
        providertest.WithError(providertest.ErrMockRateLimit),
    )
    client := mytts.NewClient(mock)

    _, err := client.TextToSpeech(ctx, "test")

    if !errors.Is(err, providertest.ErrMockRateLimit) {
        t.Errorf("Expected rate limit error, got: %v", err)
    }
}
```

### Testing Retry Logic

```go
func TestRetryOnFailure(t *testing.T) {
    callCount := 0
    mock := providertest.NewMockProviderWithOptions(
        providertest.WithFailAfterN(2, providertest.ErrMockNetworkError),
    )

    client := mytts.NewClientWithRetry(mock, 3) // 3 retries

    // First call uses the mock's first 2 successful attempts,
    // then fails and retries should kick in
    _, err := client.TextToSpeech(ctx, "test")

    // Verify retry behavior based on your implementation
}
```

### Testing Provider Failover

```go
func TestFailover(t *testing.T) {
    // Primary fails immediately
    primary := providertest.NewMockProviderWithOptions(
        providertest.WithError(providertest.ErrMockQuotaExceeded),
    )

    // Fallback succeeds
    fallback := providertest.NewDeepgramMock()

    client := mytts.NewFailoverClient(primary, fallback)

    result, err := client.TextToSpeech(ctx, "Hello")
    if err != nil {
        t.Fatalf("Failover should have succeeded: %v", err)
    }

    // Verify fallback was used
    if result.Provider != "deepgram" {
        t.Errorf("Expected deepgram fallback, got %s", result.Provider)
    }
}
```

### Testing Context Cancellation

```go
func TestContextCancellation(t *testing.T) {
    mock := providertest.NewMockProviderWithOptions(
        providertest.WithLatency(5 * time.Second),
    )

    ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
    defer cancel()

    start := time.Now()
    _, err := mock.Synthesize(ctx, "test", tts.SynthesisConfig{})
    elapsed := time.Since(start)

    if err == nil {
        t.Error("Expected context cancellation error")
    }

    if elapsed > 500*time.Millisecond {
        t.Errorf("Should have cancelled quickly, took %v", elapsed)
    }
}
```

### Testing with Conformance Suite

```go
func TestMyMock_Conformance(t *testing.T) {
    mock := providertest.NewElevenLabsMock()

    providertest.RunInterfaceTests(t, providertest.Config{
        Provider:        mock,
        SkipIntegration: true,
    })
}
```

## Best Practices

1. **Use provider-specific mocks** when testing provider-dependent behavior (voice lists, naming conventions)

2. **Use configurable mocks** when testing error handling, retry logic, or timing-sensitive code

3. **Generate fresh fixtures** for each test to avoid shared state issues

4. **Test context cancellation** to ensure your code properly handles timeouts

5. **Combine with conformance tests** to verify your integration works with any provider

## See Also

- [Provider Conformance Testing](provider-conformance-testing.md) - TRD for conformance test design
- [v0.7.0 Release Notes](releases/v0.7.0.md) - Full list of mock provider features

# Observability

The `observability` package provides instrumentation interfaces for monitoring and debugging voice operations. It enables tracking of call lifecycle events, TTS synthesis metrics, and STT transcription performance.

## Overview

OmniVoice observability consists of two main components:

1. **Voice Events** - Call lifecycle events (initiated, answered, ended, etc.)
2. **Operation Hooks** - TTS and STT instrumentation for latency, throughput, and error tracking

```
┌─────────────────────────────────────────────────────────────────┐
│                        Voice Application                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────┐   │
│  │     TTS      │    │     STT      │    │   CallSystem     │   │
│  │   + Hook     │    │   + Hook     │    │   + Observer     │   │
│  └──────┬───────┘    └──────┬───────┘    └────────┬─────────┘   │
│         │                   │                     │              │
│         ▼                   ▼                     ▼              │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                    Observability Layer                      │ │
│  ├──────────────────┬──────────────────┬───────────────────────┤ │
│  │    TTSHook       │    STTHook       │    VoiceObserver      │ │
│  │  - Latency       │  - Latency       │  - Call Events        │ │
│  │  - Audio Size    │  - Confidence    │  - Media Events       │ │
│  │  - Errors        │  - Errors        │  - DTMF Events        │ │
│  └──────────────────┴──────────────────┴───────────────────────┘ │
│                              │                                   │
│                              ▼                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                      Backends                               │ │
│  │   Prometheus  │  OpenTelemetry  │  Logging  │  Custom       │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Voice Events

### Event Types

The package defines event types for the call lifecycle:

| Event | Description |
|-------|-------------|
| `call.initiated` | Call started (outbound) or received (inbound) |
| `call.ringing` | Outbound call is ringing |
| `call.answered` | Call was answered |
| `call.ended` | Call ended normally |
| `call.failed` | Call failed |
| `call.busy` | Line was busy |
| `call.no_answer` | No answer |
| `media.connected` | Media streaming connected |
| `media.disconnected` | Media streaming disconnected |
| `media.error` | Media streaming error |
| `dtmf.received` | DTMF tones received |

### VoiceEvent Structure

```go
type VoiceEvent struct {
    Type      EventType         // Event type (e.g., "call.answered")
    Timestamp time.Time         // When the event occurred
    CallID    string            // Unique call identifier
    Provider  string            // Provider name (e.g., "twilio")
    Direction string            // "inbound" or "outbound"
    From      string            // Caller ID
    To        string            // Called number
    Duration  time.Duration     // Call duration (for ended events)
    Error     error             // Error details (for failed events)
    Metadata  map[string]any    // Provider-specific data
}
```

### VoiceObserver Interface

Implement `VoiceObserver` to receive voice events:

```go
type VoiceObserver interface {
    OnEvent(ctx context.Context, event VoiceEvent)
}
```

### Basic Usage

```go
import "github.com/plexusone/omnivoice-core/observability"

// Create an observer using the function adapter
observer := observability.VoiceObserverFunc(func(ctx context.Context, event observability.VoiceEvent) {
    log.Printf("[%s] %s: %s -> %s",
        event.Type, event.CallID, event.From, event.To)
})

// Use with CallSystem
call, err := provider.MakeCall(ctx, "+15559876543",
    callsystem.WithObserver(observer),
)
```

### Emitting Events

Providers use `EmitEvent` to send events with functional options:

```go
observability.EmitEvent(ctx, observer, observability.EventCallAnswered, callID, "twilio",
    observability.WithDirection("outbound"),
    observability.WithFrom("+15551234567"),
    observability.WithTo("+15559876543"),
)
```

### Multi-Observer

Fan out events to multiple observers:

```go
multi := observability.NewMultiObserver(
    metricsObserver,
    loggingObserver,
    analyticsObserver,
)

call, err := provider.MakeCall(ctx, to, callsystem.WithObserver(multi))
```

## TTS Hooks

The `TTSHook` interface instruments text-to-speech operations:

```go
type TTSHook interface {
    // Called before synthesis
    BeforeSynthesize(ctx context.Context, info TTSCallInfo, req TTSRequest) context.Context

    // Called after synthesis completes
    AfterSynthesize(ctx context.Context, info TTSCallInfo, req TTSRequest, resp *TTSResponse, err error)

    // Wraps streaming audio for byte counting
    WrapStream(ctx context.Context, info TTSCallInfo, req TTSRequest, stream <-chan []byte) <-chan []byte
}
```

### TTSCallInfo

```go
type TTSCallInfo struct {
    CallID    string    // Unique identifier for correlation
    Provider  string    // Provider name (e.g., "elevenlabs")
    StartTime time.Time // Operation start time
    VoiceID   string    // Voice being used
    Model     string    // TTS model
}
```

### TTSRequest / TTSResponse

```go
type TTSRequest struct {
    Text         string // Text to synthesize
    TextLength   int    // Character count
    OutputFormat string // Audio format (e.g., "mp3")
    SampleRate   int    // Audio sample rate
}

type TTSResponse struct {
    AudioSize int64         // Generated audio size in bytes
    Duration  time.Duration // Audio duration
    Latency   time.Duration // Time to first byte (streaming)
}
```

### Using TTS Hooks

```go
// Set hook on client (applies to all operations)
ttsClient.SetHook(myTTSHook)

// Or per-request via config
result, err := provider.Synthesize(ctx, text, tts.SynthesisConfig{
    VoiceID: "voice-id",
    Hook:    myTTSHook,
})
```

### Example: Metrics Hook

```go
type MetricsTTSHook struct {
    synthesizeLatency prometheus.Histogram
    audioBytes        prometheus.Counter
    errors            prometheus.Counter
}

func (h *MetricsTTSHook) BeforeSynthesize(ctx context.Context, info observability.TTSCallInfo, req observability.TTSRequest) context.Context {
    return ctx // Could add trace span to context
}

func (h *MetricsTTSHook) AfterSynthesize(ctx context.Context, info observability.TTSCallInfo, req observability.TTSRequest, resp *observability.TTSResponse, err error) {
    if err != nil {
        h.errors.Inc()
        return
    }
    h.synthesizeLatency.Observe(resp.Latency.Seconds())
    h.audioBytes.Add(float64(resp.AudioSize))
}

func (h *MetricsTTSHook) WrapStream(ctx context.Context, info observability.TTSCallInfo, req observability.TTSRequest, stream <-chan []byte) <-chan []byte {
    return stream // Could wrap to count bytes
}
```

## STT Hooks

The `STTHook` interface instruments speech-to-text operations:

```go
type STTHook interface {
    // Called before transcription
    BeforeTranscribe(ctx context.Context, info STTCallInfo, req STTRequest) context.Context

    // Called after transcription completes
    AfterTranscribe(ctx context.Context, info STTCallInfo, req STTRequest, resp *STTResponse, err error)

    // Wraps audio writer for byte tracking
    WrapStreamWriter(ctx context.Context, info STTCallInfo, req STTRequest, writer io.WriteCloser) io.WriteCloser

    // Called for each streaming result
    OnStreamResult(ctx context.Context, info STTCallInfo, resp STTResponse)
}
```

### STTCallInfo

```go
type STTCallInfo struct {
    CallID    string    // Unique identifier for correlation
    Provider  string    // Provider name (e.g., "deepgram")
    StartTime time.Time // Operation start time
    Model     string    // STT model
    Language  string    // Expected language
}
```

### STTRequest / STTResponse

```go
type STTRequest struct {
    AudioSize   int64  // Audio size in bytes
    Encoding    string // Audio encoding (e.g., "pcm")
    SampleRate  int    // Sample rate
    Channels    int    // Number of channels
    IsStreaming bool   // Streaming transcription
}

type STTResponse struct {
    Transcript       string        // Transcribed text
    TranscriptLength int           // Character count
    Confidence       float64       // Confidence score (0-1)
    AudioDuration    time.Duration // Audio processed
    Latency          time.Duration // Processing latency
    IsFinal          bool          // Final result (streaming)
}
```

### Using STT Hooks

```go
// Set hook on client
sttClient.SetHook(mySTTHook)

// Or per-request
result, err := provider.Transcribe(ctx, audio, stt.TranscriptionConfig{
    Model: "nova-2",
    Hook:  mySTTHook,
})
```

## NoOp Implementations

For testing or optional observability, use the provided no-op implementations:

```go
// These do nothing but satisfy the interfaces
var _ observability.TTSHook = observability.NoOpTTSHook{}
var _ observability.STTHook = observability.NoOpSTTHook{}
```

## Integration Patterns

### OpenTelemetry

```go
type OTelTTSHook struct {
    tracer trace.Tracer
}

func (h *OTelTTSHook) BeforeSynthesize(ctx context.Context, info observability.TTSCallInfo, req observability.TTSRequest) context.Context {
    ctx, span := h.tracer.Start(ctx, "tts.synthesize",
        trace.WithAttributes(
            attribute.String("tts.provider", info.Provider),
            attribute.String("tts.voice", info.VoiceID),
            attribute.Int("tts.text_length", req.TextLength),
        ),
    )
    return ctx
}

func (h *OTelTTSHook) AfterSynthesize(ctx context.Context, info observability.TTSCallInfo, req observability.TTSRequest, resp *observability.TTSResponse, err error) {
    span := trace.SpanFromContext(ctx)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    } else {
        span.SetAttributes(
            attribute.Int64("tts.audio_bytes", resp.AudioSize),
            attribute.Float64("tts.latency_ms", float64(resp.Latency.Milliseconds())),
        )
    }
    span.End()
}
```

### Logging

```go
type LoggingObserver struct {
    logger *slog.Logger
}

func (o *LoggingObserver) OnEvent(ctx context.Context, event observability.VoiceEvent) {
    o.logger.Info("voice event",
        "type", event.Type,
        "call_id", event.CallID,
        "provider", event.Provider,
        "direction", event.Direction,
        "from", event.From,
        "to", event.To,
    )
}
```

## Best Practices

1. **Keep hooks lightweight** - Observers are called synchronously; avoid blocking operations
2. **Handle errors internally** - Hooks should not panic or return errors
3. **Use context for correlation** - Pass trace IDs through context in `BeforeSynthesize`/`BeforeTranscribe`
4. **Aggregate metrics** - Use counters and histograms rather than logging every event
5. **Filter events** - Not all events need processing; filter by type as needed

## API Reference

See the [GoDoc](https://pkg.go.dev/github.com/plexusone/omnivoice-core/observability) for complete API documentation.

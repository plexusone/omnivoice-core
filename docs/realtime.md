# Realtime Voice-to-Voice

The `realtime` package provides a unified interface for native voice-to-voice LLM providers like OpenAI Realtime API and Gemini Live API. This enables ultra-low latency (~100-200ms) voice conversations compared to the traditional STT→LLM→TTS pipeline (~500-1000ms).

## Overview

```
Traditional Pipeline (PipelineModeText):
  Phone → Twilio WS → [STT → LLM (text) → TTS] → Twilio WS → Phone
  Latency: ~500-1000ms

Realtime Pipeline (PipelineModeRealtime):
  Phone → Twilio WS → [realtime.Provider] → Twilio WS → Phone
  Latency: ~100-200ms
```

## Provider Interface

The `realtime.Provider` interface defines the contract for voice-to-voice providers:

```go
import "github.com/plexusone/omnivoice-core/realtime"

type Provider interface {
    // ProcessAudioStream processes bidirectional audio stream
    ProcessAudioStream(
        ctx context.Context,
        audioIn <-chan []byte,
        config ProcessConfig,
    ) (audioCh <-chan AudioChunk, transcriptCh <-chan Transcript, err error)

    // Name returns the provider name
    Name() string

    // Close releases resources
    Close() error
}
```

### Configuration

```go
type ProcessConfig struct {
    // Instructions is the system prompt for the conversation
    Instructions string

    // Voice is the voice for audio output (e.g., "alloy", "Puck")
    Voice string

    // Functions are tools the model can call during conversation
    Functions []FunctionDeclaration

    // OnFunctionCall handles function calls from the model
    OnFunctionCall func(id, name, args string) (result any, err error)

    // Temperature controls response randomness (0-2)
    Temperature float64

    // Extensions allows provider-specific configuration
    Extensions map[string]any
}
```

### Output Types

```go
// AudioChunk represents audio output from the provider
type AudioChunk struct {
    Audio   []byte // PCM16 audio data
    IsFinal bool   // True when response is complete
}

// Transcript represents text output from the provider
type Transcript struct {
    Text    string // Transcript text
    IsInput bool   // True for user speech, false for agent
    IsFinal bool   // True when segment is complete
}
```

## Implementations

### OpenAI Realtime

```go
import openaiRealtime "github.com/plexusone/omni-openai/omnivoice/realtime"

provider := openaiRealtime.NewProvider(
    os.Getenv("OPENAI_API_KEY"),
    openaiRealtime.WithModel("gpt-4o-realtime-preview-2024-12-17"),
    openaiRealtime.WithVoice("alloy"),
    openaiRealtime.WithInstructions("You are a helpful voice assistant..."),
)
defer provider.Close()

audioCh, transcriptCh, err := provider.ProcessAudioStream(ctx, audioIn, realtime.ProcessConfig{})
```

### Gemini Live

```go
import googleRealtime "github.com/plexusone/omni-google/omnivoice/realtime"

provider := googleRealtime.NewRealtimeProvider(
    os.Getenv("GEMINI_API_KEY"),
    googleRealtime.WithModel("gemini-2.0-flash-live"),
    googleRealtime.WithVoice("Puck"),
    googleRealtime.WithInstructions("You are a helpful voice assistant..."),
)
defer provider.Close()

audioCh, transcriptCh, err := provider.ProcessAudioStream(ctx, audioIn, realtime.ProcessConfig{})
```

## Audio Format Conversion

Telephony providers use different audio formats than realtime providers:

| Provider | Format | Sample Rate | Encoding |
|----------|--------|-------------|----------|
| Twilio | mulaw | 8 kHz | 8-bit |
| Telnyx | mulaw | 8 kHz | 8-bit |
| OpenAI Realtime | PCM16 | 24 kHz | 16-bit |
| Gemini Live (input) | PCM16 | 16 kHz | 16-bit |
| Gemini Live (output) | PCM16 | 24 kHz | 16-bit |

### Using the Converter

```go
import (
    "github.com/plexusone/omnivoice-core/audio/converter"
    "github.com/plexusone/omnivoice-core/audio/format"
)

// One-shot conversion
conv := converter.New()
pcm24k, err := conv.Convert(mulawAudio, format.Twilio, format.OpenAI)

// Convenience functions
pcm24k, _ := converter.TwilioToOpenAI(mulawAudio)    // Twilio → OpenAI
mulaw8k, _ := converter.OpenAIToTwilio(pcmAudio)     // OpenAI → Twilio
pcm16k, _ := converter.TwilioToGemini(mulawAudio)    // Twilio → Gemini
mulaw8k, _ := converter.GeminiToTwilio(pcmAudio)     // Gemini → Twilio

// Streaming conversion for better performance
sc := converter.NewStreamConverter(format.Twilio, format.OpenAI)
for chunk := range audioChunks {
    converted, err := sc.Convert(chunk)
    // use converted audio
}
```

## RealtimeBridge

The `RealtimeBridge` connects telephony WebSocket audio to realtime providers, handling format conversion automatically:

```go
import (
    "github.com/plexusone/omnivoice-core/gateway"
    "github.com/plexusone/omnivoice-core/realtime"
)

// Create bridge for Twilio + OpenAI Realtime
bridge := gateway.NewRealtimeBridgeForTwilio(provider, realtime.ProcessConfig{
    Instructions: "You are a helpful voice assistant...",
    Voice:        "alloy",
})

// Or for Twilio + Gemini Live
bridge := gateway.NewRealtimeBridgeForTwilioGemini(provider, realtime.ProcessConfig{
    Instructions: "You are a helpful voice assistant...",
    Voice:        "Puck",
})

// Start processing
if err := bridge.Start(ctx); err != nil {
    log.Fatal(err)
}
defer bridge.Close()

// Forward telephony audio to bridge
go func() {
    for audio := range twilioAudioIn {
        bridge.SendAudio(audio)
    }
}()

// Receive converted audio for telephony
go func() {
    for audio := range bridge.AudioOut() {
        sendToTwilio(audio)
    }
}()

// Monitor events
for event := range bridge.Events() {
    switch event.Type {
    case gateway.EventSessionStarted:
        log.Println("Session started")
    case gateway.EventUserTranscript:
        log.Printf("User: %s", event.Data)
    case gateway.EventAgentTranscript:
        log.Printf("Agent: %s", event.Data)
    case gateway.EventInterruption:
        log.Println("User interrupted agent")
    case gateway.EventSessionEnded:
        log.Println("Session ended")
    }
}

// Get metrics
metrics := bridge.Metrics()
log.Printf("Duration: %dms, Turns: %d", metrics.SessionDurationMs, metrics.TurnCount)

// Get transcript
transcript := bridge.Transcript()
for _, turn := range transcript {
    log.Printf("[%s] %s", turn.Role, turn.Text)
}
```

## Pipeline Mode Selection

The gateway supports both text and realtime pipeline modes:

```go
import "github.com/plexusone/omnivoice-core/gateway"

// Text pipeline: STT → LLM → TTS (~500-1000ms latency)
textConfig := gateway.Config{
    Mode:        gateway.PipelineModeText,
    STTProvider: "deepgram",
    STTAPIKey:   os.Getenv("DEEPGRAM_API_KEY"),
    LLMProvider: "openai",
    LLMAPIKey:   os.Getenv("OPENAI_API_KEY"),
    LLMModel:    "gpt-4o",
    TTSProvider: "elevenlabs",
    TTSAPIKey:   os.Getenv("ELEVENLABS_API_KEY"),
    TTSVoiceID:  "21m00Tcm4TlvDq8ikWAM",
}

// Realtime pipeline: voice-to-voice (~100-200ms latency)
realtimeConfig := gateway.Config{
    Mode:             gateway.PipelineModeRealtime,
    RealtimeProvider: openaiRealtime.NewFactory(),
    RealtimeConfig: &gateway.RealtimeConfig{
        Provider:     "openai",
        APIKey:       os.Getenv("OPENAI_API_KEY"),
        Model:        "gpt-4o-realtime-preview-2024-12-17",
        Voice:        "alloy",
        Instructions: "You are a helpful voice assistant...",
    },
}
```

## Multi-Provider Client

For redundancy, use the `Client` with automatic fallback:

```go
import "github.com/plexusone/omnivoice-core/realtime"

// Create client with primary and fallback providers
client := realtime.NewClient(
    openaiProvider,   // Primary
    geminiProvider,   // Fallback 1
)
defer client.Close()

// Will automatically fallback on connection errors
audioCh, transcriptCh, err := client.ProcessAudioStream(ctx, audioIn, config)
```

## Function Calling

Realtime providers support function calling during conversations:

```go
config := realtime.ProcessConfig{
    Instructions: "You are a helpful assistant that can check the weather.",
    Functions: []realtime.FunctionDeclaration{
        {
            Name:        "get_weather",
            Description: "Get current weather for a location",
            Parameters: map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "location": map[string]any{
                        "type":        "string",
                        "description": "City name",
                    },
                },
                "required": []string{"location"},
            },
        },
    },
    OnFunctionCall: func(id, name, args string) (any, error) {
        if name == "get_weather" {
            var params struct{ Location string }
            json.Unmarshal([]byte(args), &params)
            // Call weather API...
            return map[string]any{"temperature": 72, "condition": "sunny"}, nil
        }
        return nil, fmt.Errorf("unknown function: %s", name)
    },
}
```

## Error Handling

Common errors from the `realtime` package:

```go
import "github.com/plexusone/omnivoice-core/realtime"

var (
    realtime.ErrSessionClosed     // Session was closed
    realtime.ErrConnectionFailed  // WebSocket connection failed
    realtime.ErrAuthFailed        // Authentication failed
    realtime.ErrRateLimited       // Rate limit exceeded
    realtime.ErrModelUnavailable  // Model not available
)
```

## Best Practices

1. **Choose the right mode**: Use realtime mode for conversational AI where latency matters. Use text mode when you need specific STT/TTS providers or more control.

2. **Handle interruptions**: Listen for `EventInterruption` to know when users speak over the agent.

3. **Monitor metrics**: Track latency and turn counts to ensure quality.

4. **Graceful shutdown**: Always call `Close()` to release resources.

5. **API key security**: Use environment variables for API keys, never hardcode them.

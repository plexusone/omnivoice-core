# OmniVoice

Voice abstraction layer for AgentPlexus supporting TTS, STT, and Voice Agents across multiple providers and transport protocols.

[![Build Status](https://github.com/plexusone/omnivoice-core/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/plexusone/omnivoice-core/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/plexusone/omnivoice-core)](https://goreportcard.com/report/github.com/plexusone/omnivoice-core)
[![GoDoc](https://pkg.go.dev/badge/github.com/plexusone/omnivoice-core)](https://pkg.go.dev/github.com/plexusone/omnivoice-core)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/plexusone/omnivoice-core/blob/master/LICENSE)

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              OmniVoice                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────────────────┐  │
│  │     TTS     │    │     STT     │    │          Voice Agent            │  │
│  │             │    │             │    │                                 │  │
│  │ Text → Audio│    │ Audio → Text│    │  Real-time bidirectional voice  │  │
│  └──────┬──────┘    └──────┬──────┘    └───────────────┬─────────────────┘  │
│         │                  │                           │                    │
│         ▼                  ▼                           ▼                    │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                         Provider Layer                              │    │
│  ├─────────────┬─────────────┬─────────────┬─────────────┬─────────────┤    │
│  │ ElevenLabs  │  Deepgram   │ Google Cloud│    AWS      │   Azure     │    │
│  │ Cartesia    │  Whisper    │ AssemblyAI  │   Polly     │   Speech    │    │
│  └─────────────┴─────────────┴─────────────┴─────────────┴─────────────┘    │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                         Transport Layer                             │    │
│  ├─────────────┬─────────────┬─────────────┬─────────────┬─────────────┤    │
│  │   WebRTC    │     SIP     │    PSTN     │  WebSocket  │    HTTP     │    │
│  └─────────────┴─────────────┴─────────────┴─────────────┴─────────────┘    │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                      Call System Integration                        │    │
│  ├─────────────┬─────────────┬─────────────┬─────────────┬─────────────┤    │
│  │   Twilio    │ RingCentral │    Zoom     │   LiveKit   │   Daily     │    │
│  └─────────────┴─────────────┴─────────────┴─────────────┴─────────────┘    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Key Features

- **Provider Agnostic** - Swap TTS/STT providers without code changes
- **Modular Architecture** - Use only the layers you need
- **Production Ready** - Designed for real-time, low-latency voice applications
- **Full Stack** - From phone calls to audio processing
- **Observability** - Built-in hooks for TTS/STT instrumentation and call event tracking
- **Multi-Provider Failover** - CallSystem client with automatic fallback support
- **Resilient Error Handling** - Smart fallback with error classification and retry logic
- **Canonical Transcript Format** - Standardized JSON output for STT with embedded schema

## Package Structure

```
omnivoice/
├── tts/                    # Text-to-Speech
│   ├── tts.go              # Interface definitions
│   └── providertest/       # Conformance test suite
│
├── stt/                    # Speech-to-Text
│   ├── stt.go              # Interface definitions
│   ├── transcript.go       # Canonical Transcript format
│   └── providertest/       # Conformance test suite
│
├── schema/                 # Embedded JSON Schemas
│   ├── schema.go           # //go:embed directives
│   └── transcript-v1.schema.json
│
├── agent/                  # Voice Agent orchestration
│   ├── agent.go            # Interface definitions
│   └── session.go          # Conversation session management
│
├── transport/              # Audio transport protocols
│   └── transport.go        # Interface definitions
│
├── callsystem/             # Call system integrations
│   ├── callsystem.go       # Interface definitions
│   ├── client.go           # Multi-provider client with failover
│   └── sms.go              # SMSProvider interface
│
├── observability/          # Voice instrumentation
│   ├── events.go           # VoiceEvent, VoiceObserver
│   └── hooks.go            # TTSHook, STTHook interfaces
│
├── resilience/             # Error handling and retry logic
│   ├── category.go         # Error categories
│   ├── error.go            # ProviderError type
│   ├── retry.go            # Retry with backoff
│   └── backoff.go          # Backoff strategies
│
├── registry/               # Provider discovery
│   ├── registry.go         # Registry interface
│   └── options.go          # ProviderConfig, ProviderOption
│
├── audio/                  # Audio codec utilities
│   └── codec/              # PCM, mu-law, a-law
│
├── subtitle/               # Subtitle generation
│   └── subtitle.go         # SRT/VTT from transcription results
│
├── mcp/                    # MCP server for voice interactions
│   └── server.go
│
└── pipeline/               # Pipeline components
    └── pipeline.go         # STT/TTS/Transport pipelines
```

## Quick Start

```go
import (
    "context"
    "github.com/plexusone/omnivoice-core/tts"
)

// Create a provider (e.g., ElevenLabs)
provider, err := elevenlabs.New(elevenlabs.WithAPIKey(apiKey))
if err != nil {
    log.Fatal(err)
}

// Synthesize speech
result, err := provider.Synthesize(ctx, "Hello, world!", tts.SynthesisConfig{
    VoiceID:      "voice-id",
    OutputFormat: "mp3",
})
```

## Use Case Recommendations

| Use Case | Call System | Transport | Notes |
|----------|-------------|-----------|-------|
| **IVR / Call Center** | Twilio ConversationRelay | PSTN/SIP | Best managed solution |
| **Business Phone** | RingCentral | WebRTC/SIP | Native AI Receptionist available |
| **Custom Web App** | LiveKit or Daily | WebRTC | Open source, flexible |
| **Zoom Meetings** | Recall.ai + Zoom | SDK → WebSocket | Avoid building Zoom bot yourself |
| **Browser Widget** | Direct WebSocket | WebSocket | ElevenLabs widget or custom |
| **Mobile App** | LiveKit | WebRTC | Cross-platform support |

## Latency Targets

For natural conversation, total round-trip latency should be under 500ms:

| Metric | Target | Acceptable | Poor |
|--------|--------|------------|------|
| Total round-trip | < 500ms | < 1000ms | > 1500ms |
| STT latency | < 200ms | < 300ms | > 500ms |
| LLM latency | < 300ms | < 500ms | > 1000ms |
| TTS latency | < 150ms | < 250ms | > 400ms |

## Testing

OmniVoice includes mock providers and audio fixtures for testing without API keys:

```go
import "github.com/plexusone/omnivoice-core/tts/providertest"

// Provider-specific mocks
mock := providertest.NewElevenLabsMock()  // or NewDeepgramMock(), NewOpenAIMock()

// Configurable behavior
mock := providertest.NewMockProviderWithOptions(
    providertest.WithLatency(100 * time.Millisecond),
    providertest.WithError(providertest.ErrMockRateLimit),
)

// Generate valid WAV fixtures
fixture := providertest.GenerateWAVFixture(1000, 22050)
```

See the [Testing with Mocks](testing.md) guide for comprehensive documentation.

## Transcript Format

OmniVoice provides a canonical JSON transcript format for STT output:

```go
import "github.com/plexusone/omnivoice-core/stt"

// Convert transcription result to canonical format
transcript := stt.NewTranscript(result, "deepgram", "nova-2", "audio.mp3", config)

// Save as JSON (durations serialize as milliseconds)
err := transcript.SaveJSON("output.transcript.json")

// Load from JSON
loaded, err := stt.LoadTranscript("output.transcript.json")

// Access timing information
fmt.Printf("Duration: %v\n", transcript.TotalDuration())
for _, seg := range transcript.Segments {
    fmt.Printf("Segment: %s (%v - %v)\n", seg.Text, seg.Start.Duration(), seg.End.Duration())
}
```

See the [v0.9.0 release notes](releases/v0.9.0.md) for full API documentation.

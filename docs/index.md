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

## Package Structure

```
omnivoice/
├── tts/                    # Text-to-Speech
│   ├── tts.go              # Interface definitions
│   └── providertest/       # Conformance test suite
│
├── stt/                    # Speech-to-Text
│   ├── stt.go              # Interface definitions
│   └── providertest/       # Conformance test suite
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

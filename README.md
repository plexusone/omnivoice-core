# OmniVoice

[![Build Status][build-status-svg]][build-status-url]
[![Lint Status][lint-status-svg]][lint-status-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![License][license-svg]][license-url]

Voice abstraction layer for AgentPlexus supporting TTS, STT, and Voice Agents across multiple providers and transport protocols.

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

## Package Structure

```
omnivoice/
├── tts/                    # Text-to-Speech
│   ├── tts.go              # Interface definitions
│   ├── elevenlabs/         # ElevenLabs provider
│   ├── polly/              # AWS Polly provider
│   ├── google/             # Google Cloud TTS
│   ├── azure/              # Azure Speech
│   └── cartesia/           # Cartesia provider
│
├── stt/                    # Speech-to-Text
│   ├── stt.go              # Interface definitions
│   ├── whisper/            # OpenAI Whisper
│   ├── deepgram/           # Deepgram provider
│   ├── google/             # Google Speech-to-Text
│   ├── azure/              # Azure Speech
│   └── assemblyai/         # AssemblyAI provider
│
├── agent/                  # Voice Agent orchestration
│   ├── agent.go            # Interface definitions
│   ├── session.go          # Conversation session management
│   ├── elevenlabs/         # ElevenLabs Agents
│   ├── vapi/               # Vapi.ai
│   ├── retell/             # Retell AI
│   └── custom/             # Custom agent (STT + LLM + TTS)
│
├── transport/              # Audio transport protocols
│   ├── transport.go        # Interface definitions
│   ├── webrtc/             # WebRTC transport
│   ├── websocket/          # WebSocket streaming
│   ├── sip/                # SIP protocol
│   └── http/               # HTTP-based (batch)
│
├── callsystem/             # Call system integrations
│   ├── callsystem.go       # Interface definitions
│   ├── twilio/             # Twilio ConversationRelay
│   ├── ringcentral/        # RingCentral Voice API
│   ├── zoom/               # Zoom SDK integration
│   ├── livekit/            # LiveKit rooms
│   └── daily/              # Daily.co
│
├── subtitle/               # Subtitle generation
│   └── subtitle.go         # SRT/VTT from transcription results
│
└── examples/
    ├── simple-tts/         # Basic TTS example
    ├── voice-agent/        # Voice agent with Twilio
    └── multi-provider/     # Provider fallback example
```

## Call System Integration

### How Voice Agents Connect to Phone/Video Calls

Voice AI agents need a **transport layer** to receive and send audio. The choice depends on the use case:

```
┌───────────────────────────────────────────────────────────────────────┐
│                        Call System Options                            │
├────────────────┬───────────────┬─────────────────┬────────────────────┤
│    Platform    │   Protocol    │   Best For      │   Complexity       │
├────────────────┼───────────────┼─────────────────┼────────────────────┤
│ Twilio         │ WebRTC/SIP/   │ Phone calls,    │ Medium - managed   │
│ Conversation-  │ PSTN          │ IVR, call       │ infrastructure     │
│ Relay          │               │ centers         │                    │
├────────────────┼───────────────┼─────────────────┼────────────────────┤
│ RingCentral    │ WebRTC/SIP    │ Enterprise PBX, │ Medium - native    │
│ Voice API      │               │ business phones │ AI receptionist    │
├────────────────┼───────────────┼─────────────────┼────────────────────┤
│ Zoom SDK       │ Proprietary   │ Video meetings  │ High - requires    │
│                │ (via SDK)     │ with voice bots │ native SDK         │
├────────────────┼───────────────┼─────────────────┼────────────────────┤
│ LiveKit        │ WebRTC        │ Custom apps,    │ Low - open source  │
│                │               │ real-time AI    │ WebRTC rooms       │
├────────────────┼───────────────┼─────────────────┼────────────────────┤
│ Daily.co       │ WebRTC        │ Embedded video, │ Low - simple API   │
│                │               │ browser-based   │                    │
├────────────────┼───────────────┼─────────────────┼────────────────────┤
│ WebSocket      │ WS/WSS        │ Web apps,       │ Low - direct       │
│ (Direct)       │               │ custom UIs      │ streaming          │
└────────────────┴───────────────┴─────────────────┴────────────────────┘
```

### Wiring Diagram: Voice Agent in a Phone Call

```
┌────────────────────────────────────────────────────────────────────────────────┐
│                     PSTN/WebRTC Call Flow                                      │
│                                                                                │
│   ┌─────────┐         ┌─────────────┐          ┌───────────────────────────┐   │
│   │  User   │◄───────►│   Twilio    │◄────────►│        OmniVoice          │   │
│   │ (Phone) │  PSTN   │ Conversation│ WebSocket│                           │   │
│   │         │         │   Relay     │          │  ┌─────────────────────┐  │   │
│   └─────────┘         └─────────────┘          │  │   Voice Agent       │  │   │
│                                                │  │                     │  │   │
│                                                │  │  ┌───────┐          │  │   │
│                         Audio In ─────────────►│  │  │  STT  │──┐       │  │   │
│                                                │  │  └───────┘  │       │  │   │
│                                                │  │             ▼       │  │   │
│                                                │  │  ┌───────────────┐  │  │   │
│                                                │  │  │  LLM / Agent  │  │  │   │
│                                                │  │  │  (Eino, etc.) │  │  │   │
│                                                │  │  └───────────────┘  │  │   │
│                                                │  │             │       │  │   │
│                                                │  │             ▼       │  │   │
│                                                │  │  ┌───────┐          │  │   │
│                         Audio Out ◄────────────│  │  │  TTS  │◄─┘       │  │   │
│                                                │  │  └───────┘          │  │   │
│                                                │  └─────────────────────┘  │   │
│                                                └───────────────────────────┘   │
│                                                                                │
└────────────────────────────────────────────────────────────────────────────────┘
```

### Wiring Diagram: Voice Agent in a Zoom Meeting

```
┌────────────────────────────────────────────────────────────────────────────┐
│                     Zoom Meeting Flow                                      │
│                                                                            │
│   ┌────────────────────────────────────────────────────────────────────┐   │
│   │                         Zoom Meeting                               │   │
│   │                                                                    │   │
│   │   ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────────────┐   │   │
│   │   │ User 1  │  │ User 2  │  │ User 3  │  │     Bot Client      │   │   │
│   │   │ (Human) │  │ (Human) │  │ (Human) │  │   (Zoom SDK)        │   │   │
│   │   └─────────┘  └─────────┘  └─────────┘  └──────────┬──────────┘   │   │
│   │                                                     │              │   │
│   └─────────────────────────────────────────────────────┼──────────────┘   │
│                                                         │                  │
│                                        Raw Audio Stream │                  │
│                                                         ▼                  │
│   ┌────────────────────────────────────────────────────────────────────┐   │
│   │                        OmniVoice Agent                             │   │
│   │                                                                    │   │
│   │   Option A: Use Recall.ai (recommended)                            │   │
│   │   ┌─────────────┐                                                  │   │
│   │   │  Recall.ai  │──► Handles Zoom SDK complexity                   │   │
│   │   │     Bot     │──► Provides audio stream via WebSocket           │   │
│   │   └─────────────┘                                                  │   │
│   │                                                                    │   │
│   │   Option B: Self-hosted Zoom SDK Bot                               │   │
│   │   ┌─────────────┐                                                  │   │
│   │   │ Zoom Linux  │──► Complex: requires native SDK                  │   │
│   │   │   SDK Bot   │──► One instance per meeting                      │   │
│   │   └─────────────┘──► Months of engineering                         │   │
│   │                                                                    │   │
│   └────────────────────────────────────────────────────────────────────┘   │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
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

## Latency Considerations

For natural conversation, total round-trip latency should be under 500ms:

```
User speaks → STT (100-300ms) → LLM (200-500ms) → TTS (100-200ms) → User hears

Target: < 500ms total for "instant" feel
Acceptable: < 1000ms for natural conversation
Poor: > 1500ms feels laggy
```

### Optimization Strategies

1. **Streaming STT**: Start processing before user finishes speaking
2. **Streaming TTS**: Start playing audio before full response generated
3. **Edge inference**: Use providers with edge nodes (Deepgram, ElevenLabs)
4. **Turn detection**: Use voice activity detection (VAD) for quick turn-taking

## Provider Comparison

### TTS Providers

| Provider | Latency | Quality | Voices | Streaming | Price |
|----------|---------|---------|--------|-----------|-------|
| ElevenLabs | Low | Excellent | 5000+ | Yes | $$$ |
| Cartesia | Very Low | Good | 100+ | Yes | $$ |
| AWS Polly | Low | Good | 60+ | Yes | $ |
| Google TTS | Low | Good | 200+ | Yes | $ |
| Azure Speech | Low | Excellent | 400+ | Yes | $$ |

### STT Providers

| Provider | Latency | Accuracy | Streaming | Languages | Price |
|----------|---------|----------|-----------|-----------|-------|
| Deepgram | Very Low | Excellent | Yes | 30+ | $$ |
| Whisper (OpenAI) | Medium | Excellent | No* | 50+ | $ |
| Google Speech | Low | Excellent | Yes | 125+ | $$ |
| AssemblyAI | Low | Excellent | Yes | 20+ | $$ |
| Azure Speech | Low | Excellent | Yes | 100+ | $$ |

*Whisper requires self-hosting for streaming (e.g., faster-whisper)

### Voice Agent Platforms

| Provider | Customization | Latency | Telephony | Price |
|----------|---------------|---------|-----------|-------|
| ElevenLabs Agents | Medium | Low | Via Twilio | $$$ |
| Vapi | High | Low | Built-in | $$ |
| Retell AI | High | Low | Built-in | $$ |
| Custom (OmniVoice) | Full | Variable | Via integration | Variable |

## Provider Conformance Testing

OmniVoice includes conformance test suites that provider implementations can use to verify they correctly implement the TTS and STT interfaces with consistent behavior.

### Using Conformance Tests

Provider implementations should import the `providertest` packages and run the conformance tests:

```go
// In your provider's conformance_test.go
import (
    "github.com/plexusone/omnivoice/stt/providertest"
    // or for TTS:
    // "github.com/plexusone/omnivoice/tts/providertest"
)

func TestConformance(t *testing.T) {
    p, err := New(WithAPIKey(apiKey))
    if err != nil {
        t.Fatal(err)
    }

    providertest.RunAll(t, providertest.Config{
        Provider:        p,
        TestAudioFile:   "/path/to/test.mp3",
        TestAudioURL:    "https://example.com/test.mp3",
        // ...
    })
}
```

### Test Categories

| Category | Description | API Required |
|----------|-------------|--------------|
| Interface | Verify provider implements interface contract (Name, etc.) | No |
| Behavior | Verify edge case handling (empty input, context cancellation) | Sometimes |
| Integration | Verify actual synthesis/transcription works | Yes |

### STT Integration Tests

| Test | Description |
|------|-------------|
| `Transcribe` | Batch transcription from audio bytes |
| `TranscribeFile` | Batch transcription from local file path |
| `TranscribeURL` | Batch transcription from remote URL |
| `TranscribeStream` | Real-time streaming transcription |

### TTS Integration Tests

| Test | Description |
|------|-------------|
| `Synthesize` | Returns valid audio bytes |
| `SynthesizeStream` | Streams audio chunks |
| `SynthesizeFromReader` | Handles streaming text input |

See [Provider Conformance Testing TRD](https://plexusone.github.io/omnivoice/provider-conformance-testing/) for detailed design documentation.

## Resources

### Call Systems
- [Twilio ConversationRelay](https://www.twilio.com/en-us/blog/developers/tutorials/product/voice-ai-conversationrelay-twilio-voice-sdk)
- [RingCentral Voice API](https://developers.ringcentral.com/voice-api)
- [LiveKit Voice AI](https://livekit.io/)
- [Daily.co](https://www.daily.co/)
- [Recall.ai](https://www.recall.ai/) - Meeting bot infrastructure

### Voice AI Providers
- [ElevenLabs](https://elevenlabs.io/)
- [Deepgram](https://deepgram.com/)
- [Cartesia](https://cartesia.ai/)
- [Vapi](https://vapi.ai/)
- [Retell AI](https://www.retellai.com/)

 [build-status-svg]: https://github.com/plexusone/omnivoice/actions/workflows/ci.yaml/badge.svg?branch=main
 [build-status-url]: https://github.com/plexusone/omnivoice/actions/workflows/ci.yaml
 [lint-status-svg]: https://github.com/plexusone/omnivoice/actions/workflows/lint.yaml/badge.svg?branch=main
 [lint-status-url]: https://github.com/plexusone/omnivoice/actions/workflows/lint.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/plexusone/omnivoice
 [goreport-url]: https://goreportcard.com/report/github.com/plexusone/omnivoice
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/omnivoice
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/omnivoice
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/omnivoice/blob/master/LICENSE
 [used-by-svg]: https://sourcegraph.com/github.com/plexusone/omnivoice/-/badge.svg
 [used-by-url]: https://sourcegraph.com/github.com/plexusone/omnivoice?badge

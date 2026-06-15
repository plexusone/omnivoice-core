# OmniVoice

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/plexusone/omnivoice-core/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/plexusone/omnivoice-core/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/plexusone/omnivoice-core/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/plexusone/omnivoice-core/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/plexusone/omnivoice-core/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/plexusone/omnivoice-core/actions/workflows/go-sast-codeql.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/plexusone/omnivoice-core
 [goreport-url]: https://goreportcard.com/report/github.com/plexusone/omnivoice-core
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/omnivoice-core
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/omnivoice-core
 [viz-svg]: https://img.shields.io/badge/visualizaton-Go-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=plexusone%2Fomnivoice-core
 [loc-svg]: https://tokei.rs/b1/github/plexusone/omnivoice-core
 [repo-url]: https://github.com/plexusone/omnivoice-core
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/omnivoice-core/blob/master/LICENSE

Voice abstraction layer for AgentPlexus supporting TTS, STT, and Voice Agents across multiple providers and transport protocols.

## Voice Architecture: Traditional vs Native

OmniVoice supports two fundamentally different approaches for real-time voice:

### Traditional Pipeline (STT → LLM → TTS)

```
Audio In → [STT Provider] → Text → [LLM] → Text → [TTS Provider] → Audio Out
```

- **Latency**: 500-1500ms (sum of STT + LLM + TTS)
- **Flexibility**: Mix and match any STT, LLM, and TTS providers
- **Use case**: Custom voice selection, specialized STT for domain-specific terminology

### Native Voice-to-Voice

```
Audio In → [OpenAI Realtime / Gemini Live] → Audio Out
```

- **Latency**: 100-200ms (model handles audio directly)
- **Simplicity**: Single API, no separate STT/TTS configuration
- **Use case**: Low-latency conversations, natural barge-in handling

| Aspect | Traditional | Native Voice-to-Voice |
|--------|-------------|----------------------|
| Latency | 500-1500ms | 100-200ms |
| STT/TTS Config | Required | Built-in |
| Core Interface | `stt.Provider`, `tts.Provider` | `realtime.Provider` |
| Provider Packages | `tts/`, `stt/` | `omni-openai/omnivoice/realtime`, `omni-google/omnivoice/realtime` |
| Gateway Bridge | Pipeline-based | `RealtimeBridge` in `gateway/` |
| Barge-in | Via `bargein/` package | Native support |

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
│  │   Twilio    │   Telnyx    │   Vonage    │    Plivo    │   LiveKit   │    │
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
│   ├── transcript.go       # Canonical Transcript format
│   ├── whisper/            # OpenAI Whisper
│   ├── deepgram/           # Deepgram provider
│   ├── google/             # Google Speech-to-Text
│   ├── azure/              # Azure Speech
│   └── assemblyai/         # AssemblyAI provider
│
├── schema/                 # Embedded JSON Schemas
│   ├── schema.go           # //go:embed directives
│   └── transcript-v1.schema.json  # Transcript format schema
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
│   ├── client.go           # Multi-provider client with failover
│   ├── sms.go              # SMSProvider interface
│   ├── twilio/             # Twilio ConversationRelay
│   ├── ringcentral/        # RingCentral Voice API
│   ├── zoom/               # Zoom SDK integration
│   ├── livekit/            # LiveKit rooms
│   └── daily/              # Daily.co
│
├── observability/          # Voice instrumentation
│   ├── events.go           # VoiceEvent, VoiceObserver
│   └── hooks.go            # TTSHook, STTHook interfaces
│
├── resilience/             # Error handling and retry logic
│   ├── category.go         # Error categories (transient, rate_limit, auth, etc.)
│   ├── error.go            # ProviderError with classification metadata
│   ├── classifier.go       # ErrorClassifier interface
│   ├── retry.go            # Retry and RetryWithResult functions
│   └── backoff.go          # Backoff strategies (exponential, linear, etc.)
│
├── storage/                # Session state persistence
│   ├── store.go            # SessionStore interface
│   ├── types.go            # SessionState, Turn, Metrics types
│   ├── memory.go           # In-memory implementation
│   └── redis.go            # Redis implementation
│
├── bargein/                # Barge-in detection
│   ├── config.go           # InterruptionMode (immediate, after_sentence, disabled)
│   └── detector.go         # BargeInDetector with TTS/STT integration
│
├── realtime/               # Native voice-to-voice
│   ├── provider.go         # Provider interface for OpenAI Realtime / Gemini Live
│   ├── client.go           # Multi-provider client with fallback
│   └── errors.go           # Common realtime errors
│
├── audio/                  # Audio processing
│   ├── format/             # Audio format definitions
│   │   └── format.go       # Encoding type with normalization, provider format constants
│   ├── converter/          # Audio format conversion
│   │   └── converter.go    # TwilioToOpenAI, OpenAIToTwilio, etc.
│   └── codec/              # Audio codecs (mulaw, alaw, PCM)
│
├── gateway/                # Voice gateway integration
│   ├── gateway.go          # Gateway, Session, Config interfaces
│   └── bridge.go           # RealtimeBridge for telephony ↔ realtime
│
├── registry.go             # Global provider registry (STT, TTS, CallSystem, Gateway, Realtime)
├── registry/               # Provider discovery types
│   ├── registry.go         # Registry interface, factory types, Gateway/RealtimeProvider interfaces
│   └── options.go          # ProviderConfig, ProviderOption (WithVoice, WithModel, etc.)
│
├── subtitle/               # Subtitle generation
│   └── subtitle.go         # SRT/VTT from transcription results
│
└── examples/
    ├── simple-tts/         # Basic TTS example
    ├── voice-agent/        # Voice agent with Twilio
    └── multi-provider/     # Provider fallback example
```

## Voice Gateway Interfaces

OmniVoice provides two gateway interfaces for different use cases:

### `Gateway` - PSTN Phone Calls

For traditional phone calls via Twilio, Telnyx, Vonage, or Plivo:

```go
type Gateway interface {
    Name() ProviderName
    Start(ctx context.Context) error
    Stop() error
    OnCall(handler CallHandler)              // Phone call comes in
    MakeCall(ctx, to string) (Session, error) // Dial phone number
    GetSession(callID string) (Session, bool)
    ListSessions() []Session
}
```

### `WebRTCGateway` - Browser/Mobile Apps

For WebRTC-based voice via LiveKit, Daily, etc.:

```go
type WebRTCGateway interface {
    Name() ProviderName
    Start(ctx context.Context) error
    Stop() error
    OnParticipantJoined(handler ParticipantHandler) // User joins room
    JoinRoom(ctx, roomName string) error
    LeaveRoom() error
    CurrentRoom() string
    GetSession(participantID string) (WebRTCSession, bool)
    ListSessions() []WebRTCSession
    GenerateClientToken(roomName, identity, displayName string) (string, error)
}
```

### Comparison

| Aspect | `Gateway` (PSTN) | `WebRTCGateway` |
|--------|------------------|-----------------|
| Connection | Phone number | Room name |
| Incoming | `OnCall()` | `OnParticipantJoined()` |
| Outgoing | `MakeCall(phoneNumber)` | `JoinRoom(roomName)` |
| Latency | 500ms+ | <200ms |
| Clients | Phone calls | Browser/mobile apps |

## Call System Integration

### How Voice Agents Connect to Phone/Video Calls

Voice AI agents need a **transport layer** to receive and send audio. The choice depends on the use case:

```
┌───────────────────────────────────────────────────────────────────────┐
│                   Voice Gateway Providers (Bidirectional)             │
├────────────────┬───────────────┬─────────────────┬────────────────────┤
│    Platform    │   Protocol    │   Audio Format  │   Auth Method      │
├────────────────┼───────────────┼─────────────────┼────────────────────┤
│ Twilio         │ Media Streams │ mulaw 8kHz      │ Account SID/Token  │
│ Media Streams  │ WebSocket     │                 │                    │
├────────────────┼───────────────┼─────────────────┼────────────────────┤
│ Telnyx         │ Media         │ L16 16kHz       │ API Key            │
│ Media Streaming│ WebSocket     │                 │                    │
├────────────────┼───────────────┼─────────────────┼────────────────────┤
│ Vonage         │ Voice         │ L16 16kHz       │ JWT (RS256)        │
│ Voice WebSocket│ WebSocket     │                 │                    │
├────────────────┼───────────────┼─────────────────┼────────────────────┤
│ Plivo          │ Stream API    │ L16 16kHz       │ Auth ID/Token      │
│ Audio Streaming│ WebSocket     │                 │                    │
└────────────────┴───────────────┴─────────────────┴────────────────────┘

┌───────────────────────────────────────────────────────────────────────┐
│                     Other Call System Options                         │
├────────────────┬───────────────┬─────────────────┬────────────────────┤
│    Platform    │   Protocol    │   Best For      │   Complexity       │
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
│                     PSTN/WebSocket Call Flow                                   │
│                                                                                │
│   ┌─────────┐         ┌─────────────┐          ┌───────────────────────────┐   │
│   │  User   │◄───────►│  Provider   │◄────────►│        OmniVoice          │   │
│   │ (Phone) │  PSTN   │  (Twilio/   │ WebSocket│                           │   │
│   │         │         │   Telnyx/   │          │  ┌─────────────────────┐  │   │
│   └─────────┘         │   Vonage/   │          │  │   Voice Agent       │  │   │
│                       │   Plivo)    │          │  │                     │  │   │
│                       └─────────────┘          │  │  ┌───────┐          │  │   │
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
| **IVR / Call Center** | Twilio, Telnyx, Plivo | PSTN/WebSocket | Managed infrastructure |
| **International Calls** | Plivo, Vonage | PSTN/WebSocket | Good international rates |
| **Enterprise Voice** | Vonage, Telnyx | PSTN/WebSocket | Flexible call control |
| **Custom Web App** | LiveKit or Daily | WebRTC | Open source, flexible |
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
    "github.com/plexusone/omnivoice-core/stt/providertest"
    // or for TTS:
    // "github.com/plexusone/omnivoice-core/tts/providertest"
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

### Mock Providers for Testing

The `tts/providertest` package includes mock providers and fixtures for testing TTS integrations without API keys:

```go
import "github.com/plexusone/omnivoice-core/tts/providertest"

// Provider-specific mocks with realistic voices
elevenLabs := providertest.NewElevenLabsMock()  // Rachel, Bella, Antoni
deepgram := providertest.NewDeepgramMock()      // Asteria, Luna, Orion
openai := providertest.NewOpenAIMock()          // Alloy, Echo, Fable, Onyx, Nova, Shimmer

// Configurable mock behaviors
mock := providertest.NewMockProviderWithOptions(
    providertest.WithLatency(100 * time.Millisecond),  // Simulate network delay
    providertest.WithError(providertest.ErrMockRateLimit),  // Error injection
    providertest.WithFailAfterN(3, providertest.ErrMockQuotaExceeded),  // Failover testing
)

// Generate valid WAV fixtures
fixture := providertest.GenerateWAVFixture(1000, 22050)  // 1 second at 22050 Hz
```

## Native Voice-to-Voice Providers

For lowest latency, use native voice-to-voice APIs that bypass traditional STT/TTS:

| Provider | Package | Latency | Audio Format |
|----------|---------|---------|--------------|
| **OpenAI Realtime** | [`omni-openai/omnivoice/realtime`](https://github.com/plexusone/omni-openai) | ~100ms | PCM16 24kHz |
| **Gemini Live** | [`omni-google/omnivoice`](https://github.com/plexusone/omni-google) | ~200ms | PCM16 16kHz in, 24kHz out |

These providers implement the `RealtimeProvider` interface:

```go
type RealtimeProvider interface {
    ProcessAudioStream(
        ctx context.Context,
        audioIn <-chan []byte,
        config ProcessConfig,
    ) (<-chan AudioChunk, <-chan Transcript, error)
    Name() string
}
```

## Resources

### Voice Gateway Providers
- [Twilio Media Streams](https://www.twilio.com/docs/voice/media-streams)
- [Telnyx Media Streaming](https://developers.telnyx.com/docs/voice/programmable-voice/media-streaming)
- [Vonage Voice WebSocket](https://developer.vonage.com/en/voice/voice-api/guides/websockets)
- [Plivo Audio Streaming](https://www.plivo.com/docs/voice/api/stream/)

### Other Call Systems
- [LiveKit Voice AI](https://livekit.io/)
- [Daily.co](https://www.daily.co/)
- [Recall.ai](https://www.recall.ai/) - Meeting bot infrastructure

### Voice AI Providers
- [ElevenLabs](https://elevenlabs.io/)
- [Deepgram](https://deepgram.com/)
- [Cartesia](https://cartesia.ai/)
- [Vapi](https://vapi.ai/)
- [Retell AI](https://www.retellai.com/)

 [build-status-svg]: https://github.com/plexusone/omnivoice-core/actions/workflows/ci.yaml/badge.svg?branch=main
 [build-status-url]: https://github.com/plexusone/omnivoice-core/actions/workflows/ci.yaml
 [lint-status-svg]: https://github.com/plexusone/omnivoice-core/actions/workflows/lint.yaml/badge.svg?branch=main
 [lint-status-url]: https://github.com/plexusone/omnivoice-core/actions/workflows/lint.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/plexusone/omnivoice-core
 [goreport-url]: https://goreportcard.com/report/github.com/plexusone/omnivoice-core
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/omnivoice-core
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/omnivoice-core
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/omnivoice-core/blob/master/LICENSE
 [used-by-svg]: https://sourcegraph.com/github.com/plexusone/omnivoice-core/-/badge.svg
 [used-by-url]: https://sourcegraph.com/github.com/plexusone/omnivoice-core?badge

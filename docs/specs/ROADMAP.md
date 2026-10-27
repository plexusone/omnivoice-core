# OmniVoice Roadmap

This document tracks the implementation roadmap for OmniVoice.

## Overview

OmniVoice is a voice abstraction layer for AgentPlexus supporting:
- **TTS** (Text-to-Speech)
- **STT** (Speech-to-Text)
- **Voice Agents** (Real-time bidirectional conversation)

## Current State

### Completed (Interfaces Only)

| Package | Status | Description |
|---------|--------|-------------|
| `tts/tts.go` | Interface | Provider interface, Client with fallback |
| `tts/errors.go` | Complete | Error definitions |
| `stt/stt.go` | Interface | Provider interface, streaming support |
| `stt/errors.go` | Complete | Error definitions |
| `agent/agent.go` | Interface | Session, events, tools, metrics |
| `transport/transport.go` | Interface | WebRTC, SIP, WebSocket protocols |
| `callsystem/callsystem.go` | Interface | Call/Meeting system integration |

### Examples (Sketches)

| Example | Status | Description |
|---------|--------|-------------|
| `examples/simple-tts/` | Skeleton | Basic TTS with provider fallback |
| `examples/twilio-agent/` | Skeleton | Twilio ConversationRelay integration |
| `examples/zoom-agent/` | Skeleton | Zoom via Recall.ai |

## Implementation Roadmap

### Phase 1: TTS Providers

Priority order based on AgentPlexus use cases (slide decks → LMS videos):

```
omnivoice/
└── tts/
    ├── elevenlabs/           # Priority 1 - Primary provider
    │   └── elevenlabs.go     # Uses github.com/plexusone/go-elevenlabs
    ├── google/               # Priority 2 - Wide language support
    │   └── google.go
    ├── polly/                # Priority 3 - AWS integration
    │   └── polly.go
    ├── azure/                # Priority 4 - Enterprise
    │   └── azure.go
    └── cartesia/             # Priority 5 - Low latency
        └── cartesia.go
```

**Dependencies:**
- `github.com/plexusone/go-elevenlabs` (move from grokify first)

### Phase 2: STT Providers

```
omnivoice/
└── stt/
    ├── deepgram/             # Priority 1 - Best real-time latency
    │   └── deepgram.go
    ├── whisper/              # Priority 2 - OpenAI, good accuracy
    │   └── whisper.go
    ├── google/               # Priority 3 - Wide language support
    │   └── google.go
    ├── assemblyai/           # Priority 4 - Good features
    │   └── assemblyai.go
    └── azure/                # Priority 5 - Enterprise
        └── azure.go
```

### Phase 3: Transport Layer

```
omnivoice/
└── transport/
    ├── websocket/            # Priority 1 - Most common for voice agents
    │   └── websocket.go
    ├── webrtc/               # Priority 2 - Browser/app integration
    │   └── webrtc.go
    └── http/                 # Priority 3 - Batch processing
        └── http.go
```

### Phase 4: Call System Integrations

```
omnivoice/
└── callsystem/
    ├── twilio/               # Priority 1 - ConversationRelay
    │   └── twilio.go
    ├── recallai/             # Priority 2 - Zoom/Teams/Meet bots
    │   └── recallai.go
    ├── livekit/              # Priority 3 - Open source WebRTC
    │   └── livekit.go
    ├── daily/                # Priority 4 - Simple WebRTC
    │   └── daily.go
    └── ringcentral/          # Priority 5 - Enterprise PBX
        └── ringcentral.go
```

### Phase 5: Voice Agent Providers

```
omnivoice/
└── agent/
    ├── custom/               # Priority 1 - STT + LLM + TTS orchestration
    │   └── custom.go         # Uses OmniLLM for LLM calls
    ├── elevenlabs/           # Priority 2 - ElevenLabs Agents platform
    │   └── elevenlabs.go
    ├── vapi/                 # Priority 3 - Vapi.ai
    │   └── vapi.go
    └── retell/               # Priority 4 - Retell AI
        └── retell.go
```

## Call System Wiring Reference

### How Voice Agents Connect to Calls

Voice agents need a **transport layer** to receive/send audio:

| Use Case | Call System | Protocol | Notes |
|----------|-------------|----------|-------|
| Phone calls (IVR, call center) | Twilio ConversationRelay | PSTN → WebSocket | Managed, recommended |
| Business phones | RingCentral | WebRTC/SIP | Native AI Receptionist available |
| Zoom meetings | Recall.ai | SDK → WebSocket | Avoid building Zoom SDK bot yourself |
| Teams meetings | Recall.ai | SDK → WebSocket | Same as Zoom |
| Google Meet | Recall.ai | SDK → WebSocket | Same as Zoom |
| Custom web app | LiveKit / Daily | WebRTC | Open source options |
| Mobile app | LiveKit | WebRTC | Cross-platform |

### Wiring Diagram: Phone Call

```
┌──────────┐        ┌─────────────────┐        ┌───────────────────┐
│  Caller  │◄──────►│     Twilio      │◄──────►│   OmniVoice       │
│  (PSTN)  │  PSTN  │ ConversationRelay│  WS   │   Voice Agent     │
└──────────┘        └─────────────────┘        │                   │
                                               │  STT → LLM → TTS  │
                                               └───────────────────┘
```

### Wiring Diagram: Zoom Meeting

```
┌─────────────────────────────────────────────────────────────────┐
│                        Zoom Meeting                              │
│   ┌─────────┐  ┌─────────┐  ┌─────────────────────────────────┐ │
│   │  User 1 │  │  User 2 │  │     Recall.ai Bot               │ │
│   └─────────┘  └─────────┘  └────────────────┬────────────────┘ │
└──────────────────────────────────────────────┼──────────────────┘
                                               │ WebSocket
                                               ▼
                              ┌─────────────────────────────────┐
                              │        OmniVoice Agent          │
                              │  STT → LLM → TTS                │
                              └─────────────────────────────────┘
```

## Dependencies

### External SDKs Needed

| Provider | Go SDK | Status |
|----------|--------|--------|
| ElevenLabs | `github.com/plexusone/go-elevenlabs` | Move from grokify |
| Deepgram | `github.com/deepgram/deepgram-go-sdk` | Available |
| OpenAI (Whisper) | `github.com/sashabaranov/go-openai` | Available |
| Google Cloud | `cloud.google.com/go/speech` | Available |
| AWS (Polly) | `github.com/aws/aws-sdk-go-v2` | Available |
| Azure | `github.com/Azure/azure-sdk-for-go` | Available |
| Twilio | `github.com/twilio/twilio-go` | Available |
| LiveKit | `github.com/livekit/server-sdk-go` | Available |
| Recall.ai | None (REST API) | Build client |

### Internal Dependencies

| Package | Depends On |
|---------|------------|
| `tts/elevenlabs` | `go-elevenlabs` |
| `agent/custom` | `OmniLLM`, `tts/`, `stt/` |
| `callsystem/twilio` | `transport/websocket` |
| `callsystem/recallai` | `transport/websocket` |

## Agent Framework Integration

OmniVoice can be used with Go-based agent frameworks:

| Framework | Integration Point |
|-----------|-------------------|
| [Eino](https://github.com/cloudwego/eino) | Use OmniVoice for voice I/O in Eino agents |
| Google ADK | Bidirectional audio streaming support |
| Custom | Direct STT/TTS/Agent interfaces |

## Latency Targets

For natural conversation:

| Metric | Target | Acceptable | Poor |
|--------|--------|------------|------|
| Total round-trip | < 500ms | < 1000ms | > 1500ms |
| STT latency | < 200ms | < 300ms | > 500ms |
| LLM latency | < 300ms | < 500ms | > 1000ms |
| TTS latency | < 150ms | < 250ms | > 400ms |

## Next Steps

1. **Move go-elevenlabs** to `github.com/plexusone/go-elevenlabs`
2. **Implement `tts/elevenlabs`** using the moved SDK
3. **Implement `stt/deepgram`** for real-time transcription
4. **Implement `transport/websocket`** for streaming
5. **Implement `callsystem/twilio`** for phone call integration
6. **Implement `agent/custom`** to orchestrate STT + OmniLLM + TTS

## Resources

### Call Systems
- [Twilio ConversationRelay](https://www.twilio.com/en-us/blog/developers/tutorials/product/voice-ai-conversationrelay-twilio-voice-sdk)
- [RingCentral Voice API](https://developers.ringcentral.com/voice-api)
- [LiveKit](https://livekit.io/)
- [Recall.ai](https://www.recall.ai/)

### Voice AI Providers
- [ElevenLabs](https://elevenlabs.io/)
- [Deepgram](https://deepgram.com/)
- [Vapi](https://vapi.ai/)
- [Retell AI](https://www.retellai.com/)

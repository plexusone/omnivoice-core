---
marp: true
theme: plexusone
paginate: true
---

# OmniVoice

## Voice Abstraction Layer for AI Agents

Build voice-enabled AI agents with pluggable providers for
**TTS**, **STT**, **Transport**, and **Call Systems**

---

# What is OmniVoice?

OmniVoice is a **Go library** that provides a unified interface for building voice AI applications.

**Key Benefits:**

- **Provider Agnostic** - Swap TTS/STT providers without code changes
- **Modular Architecture** - Use only the layers you need
- **Production Ready** - Designed for real-time, low-latency voice applications
- **Full Stack** - From phone calls to audio processing

```go
// Same interface, any provider
ttsProvider, _ := elevenlabs.New()  // or twilio.New(), or polly.New()
result, _ := ttsProvider.Synthesize(ctx, "Hello world", config)
```

---

# The Problem OmniVoice Solves

Building voice agents requires integrating **multiple systems**:

| Challenge | Without OmniVoice | With OmniVoice |
|-----------|-------------------|----------------|
| Provider Lock-in | Rewrite code to switch | Change one line |
| Transport Complexity | Handle WebRTC, SIP, PSTN | Unified interface |
| Call System Integration | Custom per platform | Pluggable adapters |
| Latency Optimization | Manual tuning | Built-in streaming |

**Result:** Build voice agents in days, not months.

---

# Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                           OmniVoice                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────┐   ┌──────────┐   ┌────────────────────────────────┐   │
│  │   TTS    │   │   STT    │   │         Voice Agent            │   │
│  │Text→Audio│   │Audio→Text│   │   Real-time bidirectional      │   │
│  └────┬─────┘   └────┬─────┘   └──────────────┬─────────────────┘   │
│       │              │                        │                     │
│       └──────────────┴────────────────────────┘                     │
│                              │                                      │
│  ┌───────────────────────────┴───────────────────────────────────┐  │
│  │                     Provider Layer                            │  │
│  │  ElevenLabs │ Twilio │ Deepgram │ AWS Polly │ Google │ Azure  │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                     Transport Layer                           │  │
│  │       WebRTC  │  WebSocket  │  SIP  │  PSTN  │  HTTP          │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                   Call System Integration                     │  │
│  │     Twilio  │  RingCentral  │  Zoom  │  LiveKit  │  Daily     │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

---

# Layer 1: Text-to-Speech (TTS)

Convert text to natural-sounding speech.

```go
type Provider interface {
    Name() string
    Synthesize(ctx context.Context, text string, config SynthesisConfig) (*SynthesisResult, error)
    ListVoices(ctx context.Context) ([]Voice, error)
    GetVoice(ctx context.Context, voiceID string) (*Voice, error)
}

type StreamingProvider interface {
    Provider
    SynthesizeStream(ctx context.Context, text string, config SynthesisConfig) (<-chan StreamChunk, error)
}
```

**Key Features:**
- Batch and streaming synthesis
- Voice discovery and selection
- Audio format configuration (MP3, PCM, WAV)

---

# Layer 2: Speech-to-Text (STT)

Convert audio to text with real-time streaming.

```go
type Provider interface {
    Name() string
    Transcribe(ctx context.Context, audio []byte, config TranscriptionConfig) (*TranscriptionResult, error)
    TranscribeFile(ctx context.Context, filePath string, config TranscriptionConfig) (*TranscriptionResult, error)
}

type StreamingProvider interface {
    Provider
    TranscribeStream(ctx context.Context, config TranscriptionConfig) (io.WriteCloser, <-chan StreamEvent, error)
}
```

**Key Features:**
- Batch and streaming transcription
- Interim results for real-time feedback
- Word-level timestamps and confidence scores

---

# Layer 3: Transport

Handle audio streaming between your agent and users.

```go
type Transport interface {
    Name() string
    Protocol() string
    Listen(ctx context.Context, addr string) (<-chan Connection, error)
    Connect(ctx context.Context, addr string, config Config) (Connection, error)
    Close() error
}

type Connection interface {
    ID() string
    AudioIn() io.WriteCloser   // Send audio to remote
    AudioOut() io.Reader       // Receive audio from remote
    Events() <-chan Event
    Close() error
}
```

**Supports:** WebSocket, WebRTC, SIP, PSTN

---

# Layer 4: Call System

Integrate with telephony and meeting platforms.

```go
type CallSystem interface {
    Name() string
    Configure(config CallSystemConfig) error
    OnIncomingCall(handler CallHandler)
    MakeCall(ctx context.Context, to string, opts ...CallOption) (Call, error)
    GetCall(ctx context.Context, callID string) (Call, error)
    ListCalls(ctx context.Context) ([]Call, error)
    Close() error
}

type Call interface {
    ID() string
    From() string
    To() string
    Answer(ctx context.Context) error
    Hangup(ctx context.Context) error
    Transport() transport.Connection
    AttachAgent(ctx context.Context, session agent.Session) error
}
```

---

# Layer 5: Voice Agent

Orchestrate TTS + STT for conversational AI.

```go
type Provider interface {
    Name() string
    CreateSession(ctx context.Context, config Config) (Session, error)
    GetSession(ctx context.Context, sessionID string) (Session, error)
    ListSessions(ctx context.Context) ([]string, error)
}

type Session interface {
    ID() string
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    AudioIn() io.WriteCloser   // User's speech
    AudioOut() io.Reader       // Agent's speech
    Events() <-chan Event      // Transcripts, turn events
    SendText(ctx context.Context, text string) error
    Interrupt(ctx context.Context) error
}
```

---

# Current Provider Support

## Production Ready

| Layer | Twilio | ElevenLabs |
|-------|--------|------------|
| **TTS** | TwiML `<Say>` verb (Alice, Polly, Google voices) | High-quality neural voices, streaming |
| **STT** | TwiML `<Gather>`, real-time transcription | WebSocket streaming transcription |
| **Transport** | Media Streams (WebSocket) | WebSocket audio streaming |
| **Call System** | PSTN calls, incoming/outgoing | - |
| **Agent** | - | Conversational AI (planned) |

**Best Practice:** Use Twilio for telephony + ElevenLabs for voice quality

---

# Twilio Provider

`omnivoice-twilio` - Full telephony stack for phone call agents.

```go
import (
    "github.com/plexusone/omnivoice-twilio/callsystem"
    "github.com/plexusone/omnivoice-twilio/transport"
    "github.com/plexusone/omnivoice-twilio/tts"
    "github.com/plexusone/omnivoice-twilio/stt"
)

// Create call system
cs, _ := callsystem.New(
    callsystem.WithAccountSID("ACxxx"),
    callsystem.WithAuthToken("xxx"),
    callsystem.WithPhoneNumber("+15551234567"),
)

// Handle incoming calls
cs.OnIncomingCall(func(call callsystem.Call) error {
    return call.Answer(ctx)
})

// Make outbound call
call, _ := cs.MakeCall(ctx, "+15559876543")
```

---

# ElevenLabs Provider

`omnivoice-elevenlabs` - Premium voice quality for TTS and STT.

```go
import (
    "github.com/plexusone/omnivoice-elevenlabs/tts"
    "github.com/plexusone/omnivoice-elevenlabs/stt"
)

// Create TTS provider
ttsProvider, _ := tts.New(
    tts.WithAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
)

// Synthesize with streaming
chunks, _ := ttsProvider.SynthesizeStream(ctx, "Hello!", tts.SynthesisConfig{
    VoiceID:      "21m00Tcm4TlvDq8ikWAM",
    Model:        "eleven_turbo_v2_5",
    OutputFormat: "pcm_16000",
})

// Create STT provider
sttProvider, _ := stt.New(
    stt.WithAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
)
```

---

# Building a Complete Voice Agent

Combine Twilio (telephony) + ElevenLabs (voice quality):

```go
// Phone handling: Twilio
callSystem, _ := twiliocs.New(...)
transport, _ := twiliotransport.New(...)

// High-quality voice: ElevenLabs
ttsProvider, _ := eleventts.New(...)
sttProvider, _ := elevenstt.New(...)

// Handle incoming call
callSystem.OnIncomingCall(func(call callsystem.Call) error {
    call.Answer(ctx)

    conn := call.Transport()

    // Process audio: STT -> Your Logic -> TTS
    go processAudio(conn.AudioOut(), sttProvider, yourAgent, ttsProvider, conn.AudioIn())

    return nil
})
```

---

# Call Flow: Voice Agent on Phone

```
┌─────────┐         ┌─────────────┐         ┌──────────────────────────┐
│  User   │◄───────►│   Twilio    │◄───────►│       OmniVoice          │
│ (Phone) │  PSTN   │   Media     │WebSocket│                          │
│         │         │  Streams    │         │  ┌────────────────────┐  │
└─────────┘         └─────────────┘         │  │    Voice Agent     │  │
                                            │  │                    │  │
                                            │  │  ┌──────┐          │  │
                       Audio In ───────────►│  │  │ STT  │──┐       │  │
                                            │  │  └──────┘  │       │  │
                                            │  │            ▼       │  │
                                            │  │  ┌────────────┐    │  │
                                            │  │  │ LLM/Agent  │    │  │
                                            │  │  └────────────┘    │  │
                                            │  │            │       │  │
                                            │  │            ▼       │  │
                                            │  │  ┌──────┐          │  │
                       Audio Out ◄──────────│  │  │ TTS  │◄─┘       │  │
                                            │  │  └──────┘          │  │
                                            │  └────────────────────┘  │
                                            └──────────────────────────┘
```

---

# Provider Mix & Match

Choose the best provider for each layer:

| Layer | Budget Option | Premium Option |
|-------|--------------|----------------|
| **Call System** | Twilio | Twilio |
| **Transport** | Twilio Media Streams | Twilio Media Streams |
| **TTS** | Twilio (Polly voices) | ElevenLabs |
| **STT** | Twilio Gather | ElevenLabs / Deepgram |
| **Agent** | Custom (STT+LLM+TTS) | ElevenLabs Agents |

**Example: Budget Setup**
```go
// All Twilio - simple, integrated
tts, _ := twiliotts.New()
stt, _ := twiliostt.New()
```

**Example: Premium Setup**
```go
// Twilio for calls, ElevenLabs for voice
tts, _ := eleventts.New()
stt, _ := elevenstt.New()
```

---

# Latency Optimization

For natural conversation, target **< 500ms** round-trip:

```
User speaks → STT (100-300ms) → LLM (200-500ms) → TTS (100-200ms) → User hears
             └─────────────────────────────────────────────────────────────┘
                                    Target: < 500ms
```

**OmniVoice Optimization Strategies:**

| Strategy | How OmniVoice Helps |
|----------|---------------------|
| **Streaming STT** | `TranscribeStream()` - process as user speaks |
| **Streaming TTS** | `SynthesizeStream()` - play before response completes |
| **Connection Pooling** | Reuse WebSocket connections |
| **Provider Selection** | Choose low-latency providers (ElevenLabs Turbo) |

---

# Code Example: Streaming TTS

```go
// Stream audio as it's generated - don't wait for full response
chunks, _ := ttsProvider.SynthesizeStream(ctx, longText, config)

for chunk := range chunks {
    // Play audio immediately as chunks arrive
    conn.AudioIn().Write(chunk.Audio)

    if chunk.IsFinal {
        break
    }
}
```

**Benefits:**
- First audio plays in ~100ms vs ~1s for batch
- User perceives faster response
- Better conversational flow

---

# Code Example: Streaming STT

```go
// Start transcription stream
writer, events, _ := sttProvider.TranscribeStream(ctx, config)

// Feed audio as it arrives
go func() {
    defer writer.Close()
    io.Copy(writer, conn.AudioOut())
}()

// Process transcription events
for event := range events {
    switch event.Type {
    case stt.EventTranscript:
        if event.IsFinal {
            // Complete utterance - send to LLM
            response := llm.Generate(ctx, event.Transcript)
            speak(response)
        } else {
            // Interim - show typing indicator
            showTyping(event.Transcript)
        }
    }
}
```

---

# Installation

```bash
# Core library (interfaces only)
go get github.com/plexusone/omnivoice

# Twilio provider (telephony + TTS + STT)
go get github.com/plexusone/omnivoice-twilio

# ElevenLabs provider (premium TTS + STT)
go get github.com/plexusone/omnivoice-elevenlabs
```

**Environment Variables:**
```bash
# Twilio
export TWILIO_ACCOUNT_SID="ACxxx"
export TWILIO_AUTH_TOKEN="xxx"

# ElevenLabs
export ELEVENLABS_API_KEY="xxx"
```

---

# Quick Start: Phone Agent

```go
package main

import (
    "context"
    "github.com/plexusone/omnivoice-twilio/callsystem"
)

func main() {
    cs, _ := callsystem.New(
        callsystem.WithPhoneNumber("+15551234567"),
        callsystem.WithWebhookURL("wss://your-server.com/media"),
    )

    cs.OnIncomingCall(func(call callsystem.Call) error {
        call.Answer(context.Background())
        // Your agent logic here
        return nil
    })

    // Start webhook server...
}
```

---

# Roadmap

## Planned Providers

| Provider | Type | Status |
|----------|------|--------|
| Deepgram | STT | Planned |
| OpenAI Whisper | STT | Planned |
| AWS Polly | TTS | Planned |
| Google Cloud | TTS/STT | Planned |
| LiveKit | Transport/CallSystem | Planned |
| Daily.co | Transport/CallSystem | Planned |
| Vapi | Agent | Planned |
| Retell AI | Agent | Planned |

## Planned Features
- Provider fallback chains
- Automatic quality-based routing
- Metrics and observability

---

# Summary

**OmniVoice** provides a unified interface for building voice AI agents:

| What | Why |
|------|-----|
| **Abstraction Layer** | Swap providers without code changes |
| **5 Layers** | TTS, STT, Transport, CallSystem, Agent |
| **Go Native** | High performance, type safe |
| **Production Ready** | Designed for real-time, low-latency |

**Current Providers:**
- **Twilio**: Full stack for phone calls
- **ElevenLabs**: Premium voice quality

**Get Started:**
```bash
go get github.com/plexusone/omnivoice
go get github.com/plexusone/omnivoice-twilio
go get github.com/plexusone/omnivoice-elevenlabs
```

---

# Resources

**Repositories:**
- [github.com/plexusone/omnivoice](https://github.com/plexusone/omnivoice)
- [github.com/plexusone/omnivoice-twilio](https://github.com/plexusone/omnivoice-twilio)
- [github.com/plexusone/omnivoice-elevenlabs](https://github.com/plexusone/omnivoice-elevenlabs)

**Documentation:**
- [pkg.go.dev/github.com/plexusone/omnivoice](https://pkg.go.dev/github.com/plexusone/omnivoice)

**Provider Docs:**
- [Twilio Media Streams](https://www.twilio.com/docs/voice/media-streams)
- [ElevenLabs API](https://elevenlabs.io/docs)

---

<!-- _class: lead -->

# Questions?

**OmniVoice** - Voice abstraction for AI agents

github.com/plexusone/omnivoice

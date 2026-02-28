# Twilio Voice Integration - Technical Requirements Document

This document describes the architecture options for integrating OmniVoice with Twilio for production voice calls.

## Overview

Twilio provides multiple approaches for building voice AI applications. Each has different tradeoffs for latency, control, and complexity.

## Architecture Options

### Option A: TwiML-Based TTS via UpdateCall

**How it works:**
```
┌─────────┐         ┌─────────────┐         ┌───────────────┐
│  User   │◄───────►│   Twilio    │◄───────►│  OmniVoice    │
│ (Phone) │  PSTN   │   Voice     │  REST   │               │
└─────────┘         └─────────────┘  API    └───────────────┘
```

1. Call is initiated with TwiML containing `<Say>` or `<Gather>`
2. For each AI response, call `UpdateCall` API with new TwiML
3. Twilio plays audio, then returns control for next turn

**Pros:**
- Simple implementation
- Uses Twilio's built-in TTS voices (Polly, Google)
- No audio format conversion needed

**Cons:**
- **High latency** - Each turn requires REST API call (~200-500ms overhead)
- **Not full-duplex** - Turn-based, user cannot interrupt
- **Poor UX** - Feels robotic, not conversational

**Best for:** Simple IVR menus, non-conversational flows

---

### Option B: Media Streams with External TTS/STT (Recommended)

**How it works:**
```
┌─────────┐         ┌─────────────┐         ┌───────────────────────────────┐
│  User   │◄───────►│   Twilio    │◄───────►│         OmniVoice             │
│ (Phone) │  PSTN   │   Media     │WebSocket│                               │
└─────────┘         │   Streams   │ (μ-law) │  ┌─────┐  ┌─────┐  ┌─────┐   │
                    └─────────────┘         │  │ STT │  │ LLM │  │ TTS │   │
                                            │  └──┬──┘  └──┬──┘  └──┬──┘   │
                                            │     └────────┴────────┘      │
                                            └───────────────────────────────┘
```

1. Call connects to Media Streams via WebSocket
2. Raw audio (mu-law, 8kHz) flows bidirectionally
3. Audio from user → STT provider (Deepgram, Whisper)
4. Transcript → LLM for response
5. Response → TTS provider (ElevenLabs) → audio back to call

**Pros:**
- **True full-duplex** - Natural conversation flow
- **Low latency** - Streaming audio, no API call overhead
- **Interruption support** - User can interrupt AI mid-sentence
- **Voice quality** - Use premium TTS (ElevenLabs, Cartesia)
- **Production-grade** - How Vapi, Retell, Bland.ai work

**Cons:**
- More complex implementation
- Requires audio format conversion (mu-law ↔ PCM/MP3)
- Need external TTS/STT providers

**Best for:** Production voice AI, conversational agents, customer service

---

### Option C: Twilio ConversationRelay

**How it works:**
```
┌─────────┐         ┌─────────────────────┐         ┌───────────────┐
│  User   │◄───────►│ Twilio Conversation │◄───────►│  OmniVoice    │
│ (Phone) │  PSTN   │       Relay         │WebSocket│   (Text)      │
└─────────┘         │  ┌─────┐  ┌─────┐   │  JSON   └───────────────┘
                    │  │ STT │  │ TTS │   │
                    │  └─────┘  └─────┘   │
                    └─────────────────────┘
```

1. Twilio handles STT and TTS internally
2. Agent receives text transcriptions via WebSocket
3. Agent sends text responses back
4. Twilio converts to speech and plays to user

**Pros:**
- Simpler implementation (text-only interface)
- Twilio handles audio complexity
- Built-in voice activity detection

**Cons:**
- **Less control** - Limited TTS voice options
- **Higher latency** - Additional Twilio processing
- **Limited customization** - Can't use custom TTS/STT

**Best for:** Quick prototypes, simple use cases

---

### Option D: ElevenLabs Conversational AI (Managed Platform)

**How it works:**
```
┌─────────┐         ┌─────────────┐         ┌───────────────────────────────┐
│  User   │◄───────►│   Twilio    │◄───────►│  ElevenLabs Conversational AI │
│ (Phone) │  PSTN   │   (Phone)   │  TwiML  │                               │
└─────────┘         └─────────────┘         │  ┌─────┐  ┌─────┐  ┌─────┐   │
                                            │  │ STT │  │ LLM │  │ TTS │   │
                                            │  └──┬──┘  └──┬──┘  └──┬──┘   │
                                            │     │   Managed │     │      │
                                            │     └────────┴───────┘       │
                                            │            │                 │
                                            │            ▼                 │
                                            │  ┌─────────────────────┐     │
                                            │  │ Custom LLM Endpoint │     │
                                            │  │ (Your ADK Agent)    │     │
                                            │  └─────────────────────┘     │
                                            └───────────────────────────────┘
```

1. Twilio call connects to ElevenLabs via TwiML
2. ElevenLabs handles STT, TTS, and conversation orchestration
3. LLM can be built-in (GPT-4, Claude) or **custom endpoint** (your agent)
4. Tool calling via webhooks or MCP servers

**Custom LLM Support:**
- Point to any OpenAI Chat Completions-compatible endpoint
- Use your own ADK agents (Claude + tools) as the brain
- ElevenLabs handles all voice complexity

**Pros:**
- **Fastest deployment** - Hours instead of weeks
- **Premium voice quality** - ElevenLabs TTS built-in
- **Low latency** - Optimized voice pipeline (~300ms)
- **Custom LLM support** - Use your own agents via endpoint
- **Tool calling** - Webhooks and MCP server support

**Cons:**
- **Less control** - Platform manages conversation flow
- **Single agent model** - One agent per call (no multi-agent orchestration)
- **Platform dependency** - Tied to ElevenLabs infrastructure
- **Cost** - Platform fee on top of API costs

**Best for:** Single-agent voice apps, rapid prototyping, when voice quality is priority

**Go SDK:** `github.com/plexusone/go-elevenlabs` - Full ConvAI support including:
- `client.Twilio().RegisterCall()` - Register incoming calls
- `client.Twilio().OutboundCall()` - Make outbound calls
- Custom LLM configuration via agent settings

---

## Comparison Matrix

| Criteria | Option A (TwiML) | Option B (Media Streams) | Option C (ConversationRelay) | Option D (ElevenLabs ConvAI) |
|----------|------------------|--------------------------|------------------------------|------------------------------|
| **Latency** | High (500ms+) | Low (100-200ms) | Medium (200-400ms) | Low (~300ms) |
| **Full-duplex** | No | Yes | Partial | Yes |
| **Interruption** | None | Excellent | Good | Excellent |
| **Voice quality** | Twilio voices | Any TTS provider | Twilio voices | ElevenLabs (excellent) |
| **Implementation** | Simple | Complex | Medium | Simple |
| **Control** | Full | Full | Medium | Medium (custom LLM) |
| **Multi-agent** | Yes | Yes | Yes | Limited |
| **Cost** | Low | Medium | Low | Higher (platform fee) |
| **Production-ready** | No | Yes | Limited | Yes |

---

## Decision: Option B (Primary) with Option D (Alternative)

We choose **Option B: Media Streams with External TTS/STT** as the primary approach, with **Option D: ElevenLabs ConvAI** as a supported alternative.

### Why Option B (Primary)

1. **Production quality** - This is how successful voice AI companies (Vapi, Retell, Bland.ai) implement their systems
2. **User experience** - True full-duplex enables natural conversations
3. **Flexibility** - Can use best-in-class TTS (ElevenLabs) and STT (Deepgram)
4. **Latency** - Critical for voice; streaming minimizes delays
5. **Multi-agent support** - Full control over agent orchestration (critical for systems like stats-agent-team)
6. **Future-proof** - Architecture supports advanced features (voice cloning, emotion detection)

### Why Option D (Alternative)

1. **Rapid deployment** - Get voice working in hours, not weeks
2. **Premium voice** - ElevenLabs TTS quality without implementation effort
3. **Custom LLM support** - Can still use your own Claude/ADK agents via custom endpoint
4. **Single-agent use cases** - Perfect when you don't need multi-agent orchestration

### When to Use Which

| Use Case | Recommended Option |
|----------|-------------------|
| Multi-agent systems (stats-agent-team) | Option B |
| Complex conversation flows | Option B |
| Single-agent voice assistant | Option D |
| Rapid prototyping | Option D |
| Maximum voice quality with minimal effort | Option D |
| Full control over audio pipeline | Option B |

## Implementation Requirements

### Audio Pipeline

```
┌──────────────────────────────────────────────────────────────────────────┐
│                        Audio Pipeline (Option B)                         │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  INBOUND (User → AI)                                                     │
│  ┌─────────────┐    ┌──────────────┐    ┌─────────────┐    ┌─────────┐  │
│  │ Twilio      │───►│ μ-law 8kHz   │───►│ PCM 16kHz   │───►│   STT   │  │
│  │ Media Stream│    │ (WebSocket)  │    │ (resample)  │    │(Deepgram│  │
│  └─────────────┘    └──────────────┘    └─────────────┘    └────┬────┘  │
│                                                                  │       │
│                                                                  ▼       │
│                                                           ┌───────────┐  │
│                                                           │ Transcript│  │
│                                                           └─────┬─────┘  │
│                                                                 │        │
│  OUTBOUND (AI → User)                                           ▼        │
│  ┌─────────────┐    ┌──────────────┐    ┌─────────────┐    ┌─────────┐  │
│  │ Twilio      │◄───│ μ-law 8kHz   │◄───│ PCM/MP3     │◄───│   TTS   │  │
│  │ Media Stream│    │ (WebSocket)  │    │ (convert)   │    │(Eleven- │  │
│  └─────────────┘    └──────────────┘    └─────────────┘    │  Labs)  │  │
│                                                            └────┬────┘  │
│                                                                 │        │
│                                                                 ▲        │
│                                                           ┌─────┴─────┐  │
│                                                           │    LLM    │  │
│                                                           │ (Claude)  │  │
│                                                           └───────────┘  │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### Components Needed

1. **Audio Format Converter**
   - mu-law (8kHz, 8-bit) ↔ PCM (16kHz, 16-bit)
   - PCM ↔ MP3/WAV for TTS providers

   **Note:** mu-law (G.711 μ-law) is an ITU-T standard, not Twilio-specific. The codec implementation lives in `omnivoice/audio/codec` for reuse across telephony providers:

   | Provider/System | Audio Format |
   |-----------------|--------------|
   | Twilio Media Streams | mu-law 8kHz |
   | RingCentral | mu-law/A-law |
   | Most SIP systems | G.711 (mu-law/A-law) |
   | PSTN (North America, Japan) | mu-law |
   | PSTN (Europe) | A-law |
   | FreeSWITCH | G.711 |

   ```
   omnivoice/
   └── audio/
       ├── codec/
       │   ├── mulaw.go      # G.711 μ-law (North America, Japan)
       │   ├── alaw.go       # G.711 A-law (Europe)
       │   └── pcm.go        # PCM utilities
       └── resample/
           └── resample.go   # Sample rate conversion (8kHz ↔ 16kHz)
   ```

2. **Streaming STT Integration**
   - Deepgram (recommended) or Whisper
   - Real-time transcription with interim results

3. **Streaming TTS Integration**
   - ElevenLabs (recommended) for quality
   - Cartesia for low latency
   - Must support streaming output

4. **Voice Activity Detection (VAD)**
   - Detect when user starts/stops speaking
   - Enable AI interruption handling

5. **Turn Management**
   - Track conversation state
   - Handle overlapping speech
   - Manage barge-in (user interrupts AI)

### Provider Recommendations

| Component | Primary | Fallback | Rationale |
|-----------|---------|----------|-----------|
| **STT** | Deepgram | Whisper (self-hosted) | Lowest latency, excellent accuracy |
| **TTS** | ElevenLabs | Cartesia | Best voice quality |
| **LLM** | Claude | GPT-4 | Best reasoning, tool use |

### Latency Budget

Target: **< 500ms** end-to-end for natural conversation

| Component | Target | Notes |
|-----------|--------|-------|
| STT | 100-150ms | Deepgram streaming |
| LLM | 200-300ms | Claude streaming |
| TTS | 100-150ms | ElevenLabs streaming |
| Network | 50ms | Twilio to server |
| **Total** | **450-650ms** | Acceptable for voice |

## Implementation Phases

### Phase 1: Audio Pipeline Foundation ✅
- [x] Implement mu-law codec (`omnivoice/audio/codec/mulaw.go`)
- [x] Implement A-law codec (`omnivoice/audio/codec/alaw.go`)
- [x] Implement PCM utilities (`omnivoice/audio/codec/pcm.go`)
- [x] Add codec tests with 100% pass rate
- [ ] Connect Media Streams to transport layer
- [ ] Pipe audio to/from transport

### Phase 2: TTS Integration (ElevenLabs)
**Priority:** Implement first since `go-elevenlabs` SDK is ready.

- [ ] Create TTS provider interface in `omnivoice/tts/`
- [ ] Implement ElevenLabs streaming TTS provider
- [ ] Support native `ulaw_8000` output (no conversion needed)
- [ ] Fallback: PCM output with mu-law conversion
- [ ] Connect TTS to outbound transport

**Key Feature:** ElevenLabs WebSocket TTS supports native `ulaw_8000` output format, eliminating the need for audio conversion on the outbound path.

```
LLM Response → ElevenLabs WebSocket TTS (ulaw_8000) → Twilio Media Streams
                        ↑
              No conversion needed!
```

### Phase 3: STT Integration (Deepgram)
- [ ] Create STT provider interface in `omnivoice/stt/`
- [ ] Implement Deepgram streaming STT provider
- [ ] Connect inbound audio to STT (mu-law → PCM → Deepgram)
- [ ] Handle interim and final transcripts
- [ ] Add transcript event channel

### Phase 4: Conversation Management
- [ ] Implement VAD for turn detection
- [ ] Add barge-in (interruption) handling
- [ ] Track conversation state
- [ ] Connect STT → LLM → TTS pipeline

### Phase 5: Production Hardening
- [ ] Add error handling and reconnection
- [ ] Implement graceful degradation
- [ ] Add metrics and monitoring
- [ ] Provider fallback (e.g., Cartesia if ElevenLabs fails)

## References

- [Twilio Media Streams](https://www.twilio.com/docs/voice/media-streams)
- [Deepgram Streaming](https://developers.deepgram.com/docs/streaming)
- [ElevenLabs Streaming](https://elevenlabs.io/docs/api-reference/streaming)
- [mu-law Encoding](https://en.wikipedia.org/wiki/G.711)

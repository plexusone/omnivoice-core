# Voice Integration Approaches - Product Requirements Document

This document describes the different approaches for adding voice capabilities to AI agents, with a focus on integration options for ADK-based agents (like stats-agent-team).

## Overview

There are two primary approaches to voice-enabling AI agents:

| Approach | Description | Control Level | Best For |
|----------|-------------|---------------|----------|
| **Managed Platform** | Use a voice AI platform (ElevenLabs ConvAI, Vapi, Retell) | Medium | Quick deployment, simpler setup |
| **Custom Pipeline** | Build your own STT → LLM → TTS pipeline | Full | Complex workflows, multi-agent systems |

## Approach A: ElevenLabs Conversational AI

ElevenLabs offers a **fully-managed Conversational AI platform** that handles voice interactions end-to-end.

### Architecture

```
┌────────────────────────────────────────────────────────────────────────┐
│                    ElevenLabs Conversational AI                        │
│                                                                        │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────────────┐  │
│  │   User   │───►│   STT    │───►│   LLM    │───►│      TTS         │  │
│  │  (Voice) │◄───│(Managed) │◄───│ (Custom) │◄───│ (ElevenLabs)     │  │
│  └──────────┘    └──────────┘    └────┬─────┘    └──────────────────┘  │
│                                       │                                │
│                                       ▼                                │
│                              ┌────────────────┐                        │
│                              │  Webhook Tools │                        │
│                              │  MCP Servers   │                        │
│                              └────────────────┘                        │
└────────────────────────────────────────────────────────────────────────┘
```

### Custom LLM Support

ElevenLabs ConvAI supports **custom LLM endpoints**, enabling integration with your own agents:

```go
// Configuration for custom LLM (from go-elevenlabs)
type CustomLLM struct {
    URL            string            // Your Chat Completions-compatible endpoint
    ModelID        string            // Optional model identifier
    APIKey         string            // Authentication (via workspace secrets)
    CustomHeaders  map[string]string // Additional request headers
    APIVersion     string            // API version string
    APIType        string            // "chat_completions" or "responses"
}
```

**Supported LLM Types:**

- Built-in: GPT-4o, Claude 3.5/4.5, Gemini 2.0, Grok, etc.
- Custom: Any OpenAI Chat Completions-compatible endpoint

### Tool/Function Calling

ElevenLabs ConvAI supports multiple tool calling mechanisms:

**1. Webhook Tools**

```
Agent ──► HTTP POST ──► Your Server ──► Response
```

- Make HTTP calls to external endpoints during agent execution
- Full control over request parameters and headers
- Response timeout configuration (5-120 seconds)
- Extract values from tool responses into dynamic variables

**2. MCP (Model Context Protocol) Servers**

```
Agent ──► SSE Connection ──► MCP Server ──► Tools
```

- Direct support for connecting MCP servers
- SSE transport support
- Per-tool approval policies
- Tool configuration overrides

**3. Built-in System Tools**

- Transfer to agent/number
- Language detection
- Voicemail detection
- DTMF playback
- Call end handling

### Integration with ADK Agents

**Can ElevenLabs ConvAI be used with ADK agents (like stats-agent-team)?**

**Yes**, with two integration patterns:

#### Pattern 1: Custom LLM Endpoint

Expose your ADK agent as a Chat Completions-compatible endpoint:

```
ElevenLabs ConvAI → Custom LLM URL → ADK Agent Gateway → Claude + Tools
```

```go
// Your ADK agent exposes a /chat/completions endpoint
// that wraps the agent's conversation handling

POST /v1/chat/completions
{
  "messages": [{"role": "user", "content": "Find statistics about climate change"}],
  "stream": true
}
```

**Pros:**
- ElevenLabs handles all voice complexity
- Low latency (optimized voice pipeline)
- Simple deployment

**Cons:**
- Limited control over conversation flow
- Must adapt ADK agent to Chat Completions API
- Tool calling flows through ElevenLabs

#### Pattern 2: Webhook Tools

Keep your ADK agents as-is, call them via webhooks:

```
ElevenLabs ConvAI (built-in LLM) → Webhook Tool → ADK Agent
```

```go
// ElevenLabs agent configuration
{
  "tools": [{
    "type": "webhook",
    "name": "research_statistics",
    "description": "Research statistics on a topic",
    "url": "https://your-server.com/agents/research",
    "method": "POST"
  }]
}
```

**Pros:**
- No changes to ADK agents
- Use ADK agents as specialized tools
- Keep agent expertise modular

**Cons:**
- Higher latency (LLM → Tool → Agent → LLM)
- Less conversational (tool call pattern)

### Phone Integration

ElevenLabs ConvAI includes native phone integration:

```go
// Register incoming Twilio call
resp, _ := client.Twilio().RegisterCall(ctx, &TwilioRegisterCallRequest{
    AgentID:    "your-agent-id",
    FirstMessage: "Hello, how can I help you today?",
})
// Returns TwiML to send to Twilio

// Make outbound call
resp, _ := client.Twilio().OutboundCall(ctx, &TwilioOutboundCallRequest{
    AgentID:           "your-agent-id",
    AgentPhoneNumberID: "your-phone-id",
    ToNumber:          "+1234567890",
})
```

### Go Library Support

The `go-elevenlabs` library provides full support:

```go
import "github.com/agentplexus/go-elevenlabs"

client := elevenlabs.NewClient(apiKey)

// Access Conversational AI features
client.Twilio()       // Phone integration
client.PhoneNumbers() // Phone number management
client.API()          // Full ConvAI API access (agents, tools, knowledge bases)
```

---

## Approach B: Custom Voice Pipeline (OmniVoice)

Build your own voice pipeline with full control over each component.

### Architecture

```
┌────────────────────────────────────────────────────────────────────────┐
│                         OmniVoice Pipeline                             │
│                                                                        │
│  ┌──────────┐    ┌──────────────┐    ┌───────────────────────────┐    │
│  │  Twilio  │    │   Deepgram   │    │       ADK Agent           │    │
│  │  Media   │───►│     STT      │───►│                           │    │
│  │ Streams  │    │  (Streaming) │    │  ┌─────────────────────┐  │    │
│  │          │    └──────────────┘    │  │  Claude + Tools     │  │    │
│  │          │                        │  │  (Research, Verify) │  │    │
│  │          │    ┌──────────────┐    │  └─────────────────────┘  │    │
│  │          │◄───│  ElevenLabs  │◄───│                           │    │
│  │          │    │     TTS      │    │                           │    │
│  └──────────┘    │  (Streaming) │    └───────────────────────────┘    │
│                  └──────────────┘                                      │
│                                                                        │
│  Audio Format: mu-law 8kHz ◄──► PCM 16kHz ◄──► Provider formats       │
└────────────────────────────────────────────────────────────────────────┘
```

### Component Selection

| Component | Recommended | Alternative | Rationale |
|-----------|-------------|-------------|-----------|
| **STT** | Deepgram | Whisper (self-hosted) | Lowest latency, streaming |
| **TTS** | ElevenLabs | Cartesia | Best quality, streaming |
| **LLM** | Claude | GPT-4 | Best reasoning, tool use |
| **Transport** | Twilio Media Streams | LiveKit | Production telephony |

### Audio Pipeline

The audio pipeline uses the `omnivoice/audio/codec` package:

```
Inbound:  Twilio (mu-law 8kHz) → MulawDecode → Resample → Deepgram (PCM 16kHz)
Outbound: ElevenLabs (ulaw_8000) → Twilio (mu-law 8kHz)
          OR
          ElevenLabs (pcm_16000) → Resample → MulawEncode → Twilio
```

**Note:** ElevenLabs supports native `ulaw_8000` output, eliminating conversion overhead.

### Integration with ADK Agents

Full control over agent integration:

```go
// In your voice handler
func handleVoiceSession(conn *transport.Connection) {
    // 1. Receive audio from Twilio
    audioIn := conn.AudioIn()

    // 2. Transcribe with Deepgram
    transcript := sttClient.TranscribeStream(audioIn)

    // 3. Process with ADK agent
    response := adkAgent.ProcessMessage(ctx, transcript)

    // 4. Stream response through TTS
    audioOut := ttsClient.SynthesizeStream(response)

    // 5. Send to Twilio
    conn.AudioOut().Write(audioOut)
}
```

---

## Comparison Matrix

| Criteria | ElevenLabs ConvAI | Custom Pipeline (OmniVoice) |
|----------|-------------------|----------------------------|
| **Setup Time** | Hours | Days-Weeks |
| **Voice Latency** | ~300ms | ~400-600ms (depends on implementation) |
| **Voice Quality** | Excellent | Excellent (same TTS) |
| **Agent Control** | Medium (via custom LLM) | Full |
| **Multi-Agent Support** | Limited (one agent per call) | Full (orchestrate freely) |
| **Tool Calling** | Via webhooks/MCP | Native ADK tools |
| **Conversation Flow** | Platform-managed | Custom logic |
| **Cost** | Higher (platform fee) | Lower (pay per API call) |
| **Phone Integration** | Built-in | Via Twilio/etc |

---

## Recommendations

### Use ElevenLabs ConvAI When:

- You need voice capabilities quickly
- Your agent fits the single-agent model
- You want managed infrastructure
- Voice quality and latency are critical
- You don't need complex multi-agent orchestration

### Use Custom Pipeline When:

- You have complex multi-agent workflows (like stats-agent-team)
- You need full control over conversation flow
- You want to use ADK's native tool system
- You have specific latency/cost requirements
- You're building a platform (not a single agent)

### Hybrid Approach

For stats-agent-team and similar projects, consider a **hybrid approach**:

1. **Use ElevenLabs WebSocket TTS** for voice synthesis (best quality)
2. **Use Deepgram** for speech recognition (lowest latency)
3. **Use your ADK agents** for the actual work
4. **OmniVoice** orchestrates the pipeline

This gives you the best of both worlds: premium voice quality from ElevenLabs without giving up control of your agent architecture.

---

## Implementation References

### ElevenLabs ConvAI

- Library: `github.com/agentplexus/go-elevenlabs`
- Services: `Twilio()`, `PhoneNumbers()`, `WebSocketTTS()`
- Docs: [ElevenLabs Conversational AI](https://elevenlabs.io/docs/conversational-ai)

### OmniVoice Custom Pipeline

- Audio codecs: `omnivoice/audio/codec` (mu-law, A-law, PCM)
- Transport: `omnivoice-twilio/transport` (Media Streams WebSocket)
- TRD: `omnivoice/docs/twilio-integration-trd.md`

### ADK Agents

- Library: `google.golang.org/adk`
- Example: `github.com/agentplexus/stats-agent-team`
- Pattern: BaseAgent → LLM Agent → Tools → A2A Protocol

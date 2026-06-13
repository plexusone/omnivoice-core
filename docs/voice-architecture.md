# Voice Architecture: Traditional vs Native

This guide compares the two fundamental approaches for real-time voice AI: the traditional STT+LLM+TTS pipeline and native voice-to-voice APIs.

## Architecture Overview

### Traditional Pipeline (STT → LLM → TTS)

```
Audio In → [STT Provider] → Text → [LLM Provider] → Text → [TTS Provider] → Audio Out
              ~200ms              ~300-500ms              ~200ms
                              Total: 700-900ms+
```

Three separate API calls, each adding latency:

1. **STT**: Deepgram, Whisper, Google Speech, AssemblyAI
2. **LLM**: Claude, GPT, Gemini (text mode)
3. **TTS**: ElevenLabs, Deepgram Aura, Cartesia, OpenAI TTS

### Native Voice-to-Voice

```
Audio In → [OpenAI Realtime / Gemini Live] → Audio Out
                      ~100-200ms
```

Single WebSocket connection, model processes audio directly.

## Comparison Summary

| Aspect | Traditional (STT+LLM+TTS) | Native Voice-to-Voice |
|--------|---------------------------|----------------------|
| **Latency** | 500-1500ms | 100-200ms |
| **API Calls** | 3 separate calls | 1 WebSocket stream |
| **Configuration** | 3 providers to configure | 1 provider |
| **Barge-in** | Complex coordination | Native support |
| **Turn detection** | Manual VAD integration | Built-in VAD |
| **Voice options** | 1000s (clones, custom) | 5-11 preset voices |

## Latency Breakdown

### Traditional Pipeline

| Component | Latency | Notes |
|-----------|---------|-------|
| STT transcription | 150-300ms | Depends on utterance length |
| Network (STT response) | 20-50ms | |
| LLM inference | 200-500ms | First token latency |
| Network (LLM response) | 20-50ms | |
| TTS synthesis | 150-300ms | Time to first audio chunk |
| Network (TTS response) | 20-50ms | |
| **Total** | **560-1250ms** | Before user hears response |

### Native Voice-to-Voice

| Component | Latency | Notes |
|-----------|---------|-------|
| Audio buffering | 20-50ms | VAD + chunk collection |
| Model processing | 80-150ms | Direct audio-to-audio |
| **Total** | **100-200ms** | Single round-trip |

## Voice Quality Comparison

| Aspect | Traditional | Native |
|--------|-------------|--------|
| **Voice options** | 1000+ (ElevenLabs library, custom clones) | 5-11 preset voices |
| **Voice cloning** | Yes (ElevenLabs, PlayHT, Cartesia) | No |
| **Custom voices** | Yes (train on your audio) | No |
| **Naturalness** | Excellent (ElevenLabs) to Good | Good |
| **Emotion control** | Yes (some providers) | Limited |
| **SSML support** | Yes (most providers) | No |

### Available Voices

**OpenAI Realtime (11 voices):**
alloy, ash, ballad, coral, echo, fable, nova, onyx, sage, shimmer, verse

**Gemini Live (5 voices):**
Puck, Charon, Kore, Fenrir, Aoede

**ElevenLabs (Traditional):**
5000+ voices including custom clones

## STT Accuracy Comparison

| Aspect | Traditional | Native |
|--------|-------------|--------|
| **Provider choice** | Deepgram, Whisper, Google, AssemblyAI | Built-in only |
| **Domain tuning** | Yes (medical, legal, technical) | Limited |
| **Custom vocabulary** | Yes (Deepgram keywords, boost) | No |
| **Language support** | 100+ languages | Fewer (~30) |
| **Diarization** | Yes (speaker identification) | No |
| **Word timestamps** | Yes | Limited |

### Best STT Accuracy by Domain

| Domain | Recommended Provider |
|--------|---------------------|
| General | Deepgram Nova-2, Whisper |
| Medical | Deepgram (with medical model) |
| Legal | Deepgram (with custom vocabulary) |
| Call Center | Deepgram, AssemblyAI |
| Low-resource languages | Whisper |

## Cost Comparison

### Per-Minute Costs (Approximate)

| Approach | Component | Cost/Minute |
|----------|-----------|-------------|
| **Traditional** | | |
| | Deepgram STT (Nova-2) | $0.0043 |
| | Claude Sonnet | $0.01-0.03 |
| | ElevenLabs TTS | $0.18 |
| | **Subtotal** | **$0.19-0.21** |
| | | |
| **Native** | | |
| | OpenAI Realtime (audio in) | $0.06 |
| | OpenAI Realtime (audio out) | $0.24 |
| | **Subtotal** | **$0.30** |

### Cost Optimization Strategies

**Traditional Pipeline:**

- Use Deepgram Aura for TTS (cheaper than ElevenLabs)
- Cache common responses
- Use smaller LLM for simple queries

**Native Voice-to-Voice:**

- Shorter conversations = lower cost
- Use Gemini Live (may be cheaper for some use cases)

## Features Comparison

| Feature | Traditional | OpenAI Realtime | Gemini Live |
|---------|-------------|-----------------|-------------|
| Function calling | Via LLM | Yes | Yes |
| Streaming | Partial (TTS only) | Full duplex | Full duplex |
| Interruption (barge-in) | Manual | Native | Native |
| Context window | LLM dependent | 128k tokens | 1M tokens |
| Vision input | Separate API | No | Yes (video) |
| Google Search | No | No | Yes (grounding) |
| Code execution | No | No | Yes |
| Session persistence | Manual | Built-in | Built-in |

## Barge-in Handling

### Traditional Pipeline

Requires coordinating multiple systems:

```go
// 1. Detect user speech via STT VAD events
sttEvents := sttProvider.StreamEvents()

// 2. Stop TTS playback when user speaks
for event := range sttEvents {
    if event.Type == stt.EventSpeechStart {
        ttsPipeline.Stop()  // Stop current audio
        // Clear audio buffers
        // Signal LLM to handle interruption
    }
}
```

Use the `bargein` package for this coordination:

```go
import "github.com/plexusone/omnivoice-core/bargein"

detector := bargein.New(bargein.Config{
    Mode: bargein.ModeImmediate,
    MinSpeechDurationMs: 200,
})
detector.AttachTTS(ttsPipeline)
detector.AttachSTTEvents(sttEvents)
detector.OnInterrupt(handleInterrupt)
```

### Native Voice-to-Voice

Barge-in is handled automatically:

```go
// OpenAI Realtime - automatic interruption
// When user speaks, model stops and listens

// Gemini Live - explicit interrupt available
session.Interrupt()  // Or automatic via VAD
```

## When to Use Each

### Use Native Voice-to-Voice When:

- **Low latency is critical** - Customer service, real-time IVR, voice assistants
- **Natural conversation flow** - Barge-in and turn-taking are important
- **Simpler architecture** - Fewer moving parts, easier to deploy
- **Preset voices are acceptable** - Don't need custom/cloned voices

### Use Traditional Pipeline When:

- **Custom voices required** - Brand voice, cloned voices, specific persona
- **Domain-specific STT** - Medical, legal, technical terminology
- **Language support** - Languages not available in native APIs
- **Best-of-breed mixing** - Deepgram STT + Claude + ElevenLabs
- **Cost optimization** - Can be cheaper for low-volume or with caching
- **Compliance requirements** - Need specific provider certifications

## Hybrid Approach

Combine both approaches for optimal results:

```
Real-time conversation: OpenAI Realtime (low latency)
IVR menus/announcements: ElevenLabs (high-quality branded voice)
Voicemail transcription: Deepgram (accuracy + timestamps)
```

Example configuration:

```yaml
voice:
  # Primary: Native voice-to-voice for conversation
  realtime:
    provider: openai
    voice: alloy

  # Fallback: Traditional for specific use cases
  tts:
    provider: elevenlabs
    voice_id: branded-voice-id  # For announcements

  stt:
    provider: deepgram
    model: nova-2  # For voicemail transcription
```

## Audio Format Reference

| Provider | Input Format | Output Format |
|----------|--------------|---------------|
| **OpenAI Realtime** | PCM16 24kHz mono | PCM16 24kHz mono |
| **Gemini Live** | PCM16 16kHz mono | PCM16 24kHz mono |
| **Deepgram STT** | Various (mp3, wav, etc.) | Text |
| **Deepgram TTS** | Text | mp3, wav, pcm |
| **ElevenLabs** | Text | mp3, pcm |
| **Twilio Media Streams** | mulaw 8kHz | mulaw 8kHz |

### Sample Rate Conversion

When connecting Twilio to native voice-to-voice:

```go
// Twilio → OpenAI Realtime
twilioAudio := receive8kMulaw()
pcm16 := convertMulawToPCM16(twilioAudio)
pcm24k := resample8kTo24k(pcm16)
sendToOpenAI(pcm24k)

// OpenAI Realtime → Twilio
openaiAudio := receiveFromOpenAI()  // 24kHz PCM16
pcm8k := resample24kTo8k(openaiAudio)
mulaw := convertPCM16ToMulaw(pcm8k)
sendToTwilio(mulaw)
```

## Provider Packages

| Approach | Package | Documentation |
|----------|---------|---------------|
| **Native Voice-to-Voice** | | |
| OpenAI Realtime | `github.com/plexusone/omni-openai/omnivoice/realtime` | [Realtime Guide](https://plexusone.dev/omni-openai/providers/realtime) |
| Gemini Live | `github.com/plexusone/omni-google/omnivoice` | [Gemini Live Guide](https://plexusone.dev/omni-google/omnivoice) |
| **Traditional STT** | | |
| Deepgram | `github.com/plexusone/omnivoice-core/stt/deepgram` | |
| Whisper | `github.com/plexusone/omni-openai/omnivoice` | |
| Google Speech | `github.com/plexusone/omni-google/omnivoice` | |
| **Traditional TTS** | | |
| ElevenLabs | `github.com/plexusone/omnivoice-core/tts/elevenlabs` | |
| Deepgram Aura | `github.com/plexusone/omnivoice-core/tts/deepgram` | |
| Cartesia | `github.com/plexusone/omnivoice-core/tts/cartesia` | |
| **Infrastructure** | | |
| Barge-in Detection | `github.com/plexusone/omnivoice-core/bargein` | [Barge-in Guide](bargein.md) |
| Session Storage | `github.com/plexusone/omnivoice-core/storage` | [Storage Guide](storage.md) |

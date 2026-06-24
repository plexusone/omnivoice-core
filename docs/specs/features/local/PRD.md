# Local Voice Providers - Product Requirements Document

This document describes the requirements for adding local/on-device TTS providers to OmniVoice.

## Overview

Local voice providers enable text-to-speech synthesis on local hardware (Apple Silicon Macs) without cloud API dependencies. This enables:

- **Privacy** - Audio never leaves the device
- **Cost reduction** - No per-request API fees
- **Offline operation** - Works without internet
- **Voice cloning** - Custom voice profiles from reference audio

## Primary Use Case: VideoAsCode

The first integration target is [VideoAsCode](https://github.com/grokify/videoascode) (`vac`), a tool that converts Marp presentations with voiceovers to video files.

**Current workflow (cloud TTS):**

```
Marp Presentation + Transcript
         ↓
   ElevenLabs TTS (cloud)  ← API costs per request
         ↓
   Audio files (.wav)
         ↓
   ffmpeg + Marp PNG export
         ↓
   Video with audio + subtitles
```

**Target workflow (local TTS):**

```
Marp Presentation + Transcript
         ↓
   F5-TTS MLX (local)  ← No API costs, offline capable
         ↓
   Audio files (.wav)
         ↓
   ffmpeg + Marp PNG export
         ↓
   Video with audio + subtitles
```

**Benefits for VideoAsCode:**

| Aspect | Cloud (ElevenLabs) | Local (F5-TTS MLX) |
|--------|-------------------|-------------------|
| Cost per video | ~$0.50-5.00 | $0 |
| Requires internet | Yes | No |
| Voice cloning | Paid feature | Free (reference audio) |
| Batch processing | Rate limited | Unlimited |
| Privacy | Audio sent to cloud | Audio stays local |

**Integration point:** VideoAsCode already uses OmniVoice for TTS. Adding local providers requires no code changes—just provider selection:

```go
// Current (cloud)
provider, _ := omnivoice.GetTTSProvider("elevenlabs")

// Future (local)
provider, _ := omnivoice.GetTTSProvider("f5tts")
```

## Target Hardware

| Hardware | Memory | Status |
|----------|--------|--------|
| M1 Max | 32GB / 64GB | Primary target |
| M2 Max / M3 Max | 32GB+ | Supported |
| M4 Max | 64GB+ | Supported |

**Runtime:** MLX (Apple's ML framework for Apple Silicon)

## Use Cases

### UC-1: Local Voice Synthesis

**Actor:** Developer using OmniVoice
**Goal:** Synthesize speech locally without cloud API calls

```go
provider, _ := omnivoice.GetTTSProvider("f5tts")
audio, _ := provider.Synthesize(ctx, tts.Request{
    Text:   "Hello from local TTS",
    Voice:  "default",
    Format: "wav",
})
```

### UC-2: Voice Cloning with Reference Audio

**Actor:** Developer creating custom voice profiles
**Goal:** Clone a voice from a short audio sample

```go
profile, _ := provider.CloneVoice(ctx, CloneVoiceRequest{
    Name:          "john",
    ReferenceAudio: audioData,
    ReferenceText:  "The transcript of the reference audio.",
})

audio, _ := provider.SynthesizeWithReference(ctx, ReferenceSpeechRequest{
    Text:    "Hello in my cloned voice",
    Profile: profile,
})
```

### UC-3: Streaming Audio Output

**Actor:** Voice agent application
**Goal:** Start playing audio before synthesis completes (lower latency)

```go
stream, _ := provider.SynthesizeStream(ctx, tts.Request{
    Text: "This is a longer piece of text that streams as it generates.",
})
for chunk := range stream {
    playAudio(chunk)
}
```

### UC-4: Multi-Model Voice Profiles

**Actor:** Developer supporting multiple TTS backends
**Goal:** Use the same voice profile across different models

```
voice-profiles/john/
  manifest.json       # Name, language, metadata
  reference.wav       # Canonical reference audio
  reference.txt       # Transcript
  embeddings/
    f5tts-mlx/        # F5-TTS cached embedding
    qwen3-tts/        # Qwen3-TTS cached embedding
```

## Supported Models

| Model | Use Case | Quality | Speed |
|-------|----------|---------|-------|
| **F5-TTS MLX** | Highest quality voice cloning | Excellent | Moderate |
| **Qwen3-TTS** | Good quality, Mac-native | Very Good | Good |
| **Piper** | Fast, simple TTS | Good | Fast |
| **Kokoro** | Fast local TTS | Good | Fast |
| **macOS AVSpeechSynthesizer** | System integration | Basic | Instant |

**Priority:** F5-TTS MLX first (highest quality voice cloning)

## Provider Architecture

Local providers follow the same pattern as cloud providers:

```
omnivoice-core           ← Core interfaces
     ↑
omni-f5tts               ← F5-TTS provider (first)
omni-qwen                ← Qwen3-TTS provider
omni-piper               ← Piper provider
omni-apple               ← macOS native provider
     ↑
omnivoice                ← Batteries-included bundle
```

Each model is a **first-class provider**, not grouped under "local":

```go
// Same pattern as cloud providers
provider := f5tts.NewVoiceProvider(...)
provider := qwen.NewVoiceProvider(...)
provider := elevenlabs.NewVoiceProvider(...)
```

## Requirements

### Functional Requirements

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-1 | Synthesize speech from text locally | P0 |
| FR-2 | Stream audio output as it generates | P0 |
| FR-3 | Clone voices from reference audio + transcript | P0 |
| FR-4 | Cache model-specific embeddings per voice profile | P1 |
| FR-5 | Support multiple audio output formats (WAV, PCM, MP3) | P1 |
| FR-6 | Health check / model readiness detection | P1 |
| FR-7 | Model loading and unloading | P2 |
| FR-8 | Resource usage reporting (memory, GPU) | P2 |

### Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-1 | Time to first audio byte | < 500ms |
| NFR-2 | Real-time factor (audio duration / generation time) | > 1.0x |
| NFR-3 | Memory usage per model | < 8GB |
| NFR-4 | Support offline operation | Required |
| NFR-5 | Go client to Python/MLX service latency | < 10ms |

## Integration

### Go Application Integration

```go
import (
    "github.com/plexusone/omnivoice"
    _ "github.com/plexusone/omni-f5tts/omnivoice"
)

func main() {
    // Get local TTS provider
    provider, err := omnivoice.GetTTSProvider("f5tts",
        omnivoice.WithEndpoint("unix:///tmp/f5tts.sock"),
    )

    // Use same interface as cloud providers
    audio, err := provider.Synthesize(ctx, tts.Request{
        Text:  "Hello world",
        Voice: "john",
    })
}
```

### Voice Profile Storage

Voice profiles are stored locally and are model-agnostic:

```
~/.omnivoice/
  voice-profiles/
    john/
      manifest.json
      reference.wav
      reference.txt
      embeddings/
        f5tts-mlx/embedding.bin
        qwen3-tts/embedding.bin
```

## Success Criteria

1. **F5-TTS MLX provider** working on M1 Max with voice cloning
2. **Streaming synthesis** with < 500ms time to first byte
3. **gRPC over UDS** communication between Go and Python/MLX
4. **Voice profiles** portable across models
5. **Same interface** as cloud TTS providers

## Out of Scope (v1)

- Windows/Linux support (MLX is Apple-only)
- GPU inference on NVIDIA (use cloud providers)
- Real-time voice conversion (voice-to-voice)
- Speech-to-text (STT) - separate feature
- Training custom models

## References

- [F5-TTS MLX](https://github.com/lucasnewman/f5-tts-mlx)
- [Qwen3-TTS Apple Silicon](https://github.com/kapi2800/qwen3-tts-apple-silicon)
- [MLX](https://opensource.apple.com/projects/mlx)
- [IDEATION_CHAT_LOCAL.md](../../../IDEATION_CHAT_LOCAL.md)

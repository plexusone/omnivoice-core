# F5-TTS MLX Provider

F5-TTS provider for OmniVoice, running locally on Apple Silicon via MLX.

## Features

- Local TTS synthesis (no cloud API required)
- Zero-shot voice cloning from 5-15 second reference audio
- Streaming audio output
- Voice profile caching
- gRPC over Unix Domain Socket

## Requirements

- Apple Silicon Mac (M1/M2/M3/M4)
- macOS 13+
- Python 3.10+

## Usage

### 1. Start the Server

```bash
cd server
pip install -r requirements.txt
./generate_proto.sh
./run.sh --auto-load
```

### 2. Use from Go

```go
import (
    "github.com/plexusone/omnivoice"
    _ "github.com/plexusone/omnivoice-core/providers/f5tts"
)

func main() {
    provider, _ := omnivoice.GetTTSProvider("f5tts")
    result, _ := provider.Synthesize(ctx, "Hello world", tts.SynthesisConfig{})
}
```

### Voice Cloning

```go
cloner := provider.(tts.VoiceCloner)
profile, _ := cloner.CloneVoice(ctx, tts.CloneVoiceRequest{
    Name:           "my-voice",
    ReferenceAudio: referenceWAV,
    ReferenceText:  "Transcript of the reference audio.",
})

result, _ := provider.Synthesize(ctx, "Hello", tts.SynthesisConfig{
    VoiceID: profile.ID,
})
```

## Architecture

```
Go Application                     Python Server
┌─────────────────┐  gRPC/UDS     ┌─────────────────┐
│ f5tts.Provider  │◄─────────────►│ f5tts_server.py │
│ (gRPC client)   │               │ (MLX + F5-TTS)  │
└─────────────────┘               └─────────────────┘
        unix:///tmp/omnivoice-f5tts.sock
```

## Capability Interfaces

| Interface | Description |
|-----------|-------------|
| `tts.VoiceCloner` | Voice cloning |
| `tts.ReferenceSynthesizer` | Zero-shot synthesis |
| `tts.ProfileCacher` | Profile management |
| `tts.ModelManager` | Load/unload model |
| `tts.RuntimeChecker` | Runtime info |
| `tts.HealthChecker` | Health check |

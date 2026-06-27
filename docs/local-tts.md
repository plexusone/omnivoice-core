# Local TTS Providers

OmniVoice supports local text-to-speech providers that run on your own hardware, providing privacy, zero API costs, and offline capability.

## Overview

Local providers communicate via gRPC over Unix Domain Socket for low-latency, secure local IPC:

```
┌─────────────────────┐     gRPC/UDS      ┌─────────────────────┐
│      Go Client      │◄─────────────────►│   Python Server     │
│   (omnivoice-core)  │                   │   (MLX / PyTorch)   │
│                     │                   │                     │
│  - TTS interface    │                   │  - F5-TTS MLX       │
│  - Voice cloning    │                   │  - Model inference  │
│  - Profile caching  │                   │  - Audio generation │
└─────────────────────┘                   └─────────────────────┘
```

## Available Local Providers

| Provider | Model | Hardware | Voice Cloning | Status |
|----------|-------|----------|---------------|--------|
| **F5-TTS** | F5-TTS MLX | Apple Silicon | Yes | Available |
| Qwen3-TTS | Qwen3-TTS | Apple Silicon / CUDA | Yes | Planned |
| Piper | Piper | CPU | No | Planned |
| Apple TTS | AVSpeechSynthesizer | macOS | No | Planned |

## F5-TTS Provider

F5-TTS is a high-quality voice cloning model that supports zero-shot synthesis from reference audio.

### Requirements

- Apple Silicon Mac (M1/M2/M3/M4)
- Python 3.11+ (ARM64)
- ~2GB disk space for model weights

### Installation

```bash
# Navigate to the server directory
cd omnivoice-core/providers/f5tts/server

# Create ARM64 virtual environment
arch -arm64 python3 -m venv .venv

# Install dependencies
arch -arm64 .venv/bin/pip install -r requirements.txt

# Generate Python proto stubs
./generate_proto.sh
```

### Starting the Server

```bash
# Start without auto-loading model (faster startup)
arch -arm64 .venv/bin/python3 f5tts_server.py

# Start with model pre-loaded (ready for immediate synthesis)
arch -arm64 .venv/bin/python3 f5tts_server.py --auto-load
```

The server listens on `unix:///tmp/omnivoice-f5tts.sock` by default.

### Go Client Usage

```go
import (
    "github.com/plexusone/omnivoice"
    _ "github.com/plexusone/omnivoice-core/providers/f5tts" // Auto-register
)

// Create provider using the registry
provider, err := omnivoice.GetTTSProvider("f5tts",
    omnivoice.WithEndpoint("unix:///tmp/omnivoice-f5tts.sock"),
)
if err != nil {
    log.Fatal(err)
}

// Load the model (downloads ~2GB on first run)
if loader, ok := provider.(tts.ModelManager); ok {
    result, err := loader.LoadModel(ctx)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Model loaded in %dms", result.LoadTimeMs)
}

// Synthesize speech
result, err := provider.Synthesize(ctx, "Hello from local TTS!", tts.SynthesisConfig{
    OutputFormat: "wav",
})
```

### Voice Cloning

F5-TTS supports zero-shot voice cloning from a reference audio sample:

```go
// Option 1: Inline reference synthesis
if synth, ok := provider.(tts.ReferenceSynthesizer); ok {
    result, err := synth.SynthesizeWithReference(ctx, tts.ReferenceSynthesizeRequest{
        Text:           "Hello in your cloned voice!",
        ReferenceAudio: referenceWAV,  // []byte of reference audio
        ReferenceText:  "This is what I said in the reference.", // Transcript
        Config: tts.SynthesisConfig{
            OutputFormat: "wav",
        },
    })
}

// Option 2: Pre-cache voice profile for faster repeated synthesis
if cacher, ok := provider.(tts.ProfileCacher); ok {
    // Prepare profile once
    profile, err := cacher.PrepareVoiceProfile(ctx, tts.PrepareVoiceProfileRequest{
        ProfileID:      "my-voice",
        ReferenceAudio: referenceWAV,
        ReferenceText:  "This is what I said in the reference.",
    })

    // Use cached profile for synthesis
    result, err := provider.Synthesize(ctx, "Hello!", tts.SynthesisConfig{
        VoiceID:      "my-voice",  // Use cached profile
        OutputFormat: "wav",
    })
}
```

## Capability Interfaces

Local providers implement additional capability interfaces beyond the base `tts.Provider`:

### VoiceCloner

```go
type VoiceCloner interface {
    CloneVoice(ctx context.Context, req CloneVoiceRequest) (*VoiceProfile, error)
}
```

Creates a reusable voice profile from reference audio.

### ReferenceSynthesizer

```go
type ReferenceSynthesizer interface {
    SynthesizeWithReference(ctx context.Context, req ReferenceSynthesizeRequest) (*SynthesisResult, error)
}
```

One-shot synthesis using inline reference audio (no pre-caching).

### ProfileCacher

```go
type ProfileCacher interface {
    PrepareVoiceProfile(ctx context.Context, req PrepareVoiceProfileRequest) (*PreparedProfile, error)
    ListPreparedProfiles(ctx context.Context) ([]*PreparedProfile, error)
    DeletePreparedProfile(ctx context.Context, profileID string) error
}
```

Pre-compute and cache voice embeddings for faster repeated synthesis.

### ModelManager

```go
type ModelManager interface {
    LoadModel(ctx context.Context) (*LoadModelResult, error)
    UnloadModel(ctx context.Context) (*UnloadModelResult, error)
    IsModelLoaded() bool
}
```

Control model lifecycle for memory management.

### RuntimeChecker

```go
type RuntimeChecker interface {
    RuntimeInfo(ctx context.Context) (*RuntimeInfo, error)
}
```

Query runtime environment (device type, memory usage, framework version).

### HealthChecker

```go
type HealthChecker interface {
    Health(ctx context.Context) (*HealthStatus, error)
}
```

Check provider health and model status.

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `F5TTS_ENDPOINT` | gRPC endpoint | `unix:///tmp/omnivoice-f5tts.sock` |
| `F5TTS_TEST_REFERENCE_AUDIO` | Path to reference audio for tests | - |
| `F5TTS_TEST_REFERENCE_TEXT` | Transcript for test reference | - |

### Registry Options

```go
// Custom endpoint
omnivoice.WithEndpoint("unix:///custom/path.sock")

// TCP endpoint (for remote servers)
omnivoice.WithEndpoint("localhost:50051")
```

## Performance

### Latency (M1 Max, 32GB)

| Operation | First Run | Cached |
|-----------|-----------|--------|
| Model Load | ~30s | ~5s |
| Synthesis (short) | ~2s | ~500ms |
| Synthesis (long) | ~5s | ~2s |
| Voice Cloning | ~3s | ~1s |

### Memory Usage

| State | Memory |
|-------|--------|
| Server idle | ~200MB |
| Model loaded | ~2.2GB |
| During synthesis | ~3GB peak |

## Troubleshooting

### Server won't start

1. Check Python architecture:
   ```bash
   arch -arm64 python3 -c "import platform; print(platform.machine())"
   # Should output: arm64
   ```

2. Verify MLX is installed:
   ```bash
   arch -arm64 .venv/bin/python3 -c "import mlx; print('MLX OK')"
   ```

### Model download fails

The model downloads from Hugging Face on first use. If it fails:

1. Check network connectivity
2. Try manual download:
   ```bash
   arch -arm64 .venv/bin/python3 -c "from f5_tts_mlx.generate import generate; generate('test', steps=1)"
   ```

### gRPC connection refused

1. Check socket exists: `ls -la /tmp/omnivoice-f5tts.sock`
2. Restart server: `pkill -f f5tts_server.py && ./run.sh`

## Proto Definition

The local voice service is defined in `proto/localvoice/v1/localvoice.proto`:

```protobuf
service LocalVoice {
  rpc Synthesize(SynthesizeRequest) returns (stream AudioChunk);
  rpc SynthesizeWithReference(ReferenceSynthesizeRequest) returns (stream AudioChunk);
  rpc PrepareVoiceProfile(PrepareVoiceProfileRequest) returns (PrepareVoiceProfileResponse);
  rpc Health(HealthRequest) returns (HealthResponse);
  rpc LoadModel(LoadModelRequest) returns (LoadModelResponse);
  rpc UnloadModel(UnloadModelRequest) returns (UnloadModelResponse);
  rpc RuntimeInfo(RuntimeInfoRequest) returns (RuntimeInfoResponse);
}
```

## See Also

- [Voice Cloning Guide](voice-cloning.md) - General voice cloning concepts
- [Provider Registry](registry.md) - How provider registration works
- [Local Provider TRD](specs/features/local/TRD.md) - Technical design document

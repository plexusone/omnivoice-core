# Local Voice Providers - Technical Requirements Document

This document describes the technical architecture for local TTS providers in OmniVoice.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Go Application                                 │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     omnivoice-core                               │   │
│  │                                                                  │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │   │
│  │  │providers/    │  │providers/    │  │ omni-elevenlabs      │   │   │
│  │  │  f5tts       │  │  qwen (TODO) │  │ (thick SDK client)   │   │   │
│  │  │ (gRPC client)│  │ (gRPC client)│  │                      │   │   │
│  │  └──────┬───────┘  └──────┬───────┘  └──────────────────────┘   │   │
│  │         │                 │                                      │   │
│  └─────────┼─────────────────┼──────────────────────────────────────┘   │
│            │                 │                                          │
└────────────┼─────────────────┼──────────────────────────────────────────┘
             │                 │
             │ gRPC/UDS        │ gRPC/UDS
             ▼                 ▼
┌─────────────────────┐  ┌─────────────────────┐
│  F5-TTS MLX Server  │  │  Qwen3-TTS Server   │
│  (Python + MLX)     │  │  (Python + MLX)     │
│                     │  │                     │
│  unix:///tmp/       │  │  unix:///tmp/       │
│  omnivoice-f5tts    │  │  omnivoice-qwen3tts │
└─────────────────────┘  └─────────────────────┘
```

## Communication Protocol: gRPC over Unix Domain Socket

### Why gRPC over UDS

| Aspect | HTTP/JSON | gRPC/TCP | gRPC/UDS |
|--------|-----------|----------|----------|
| Audio payload | Base64 (33% overhead) | Native bytes | Native bytes |
| Streaming | Chunked/SSE | Native | Native |
| Latency | ~5-10ms | ~2-5ms | **~1-2ms** |
| Connection | Per-request | Persistent | Persistent |
| Contract | Informal | Protobuf | Protobuf |

**Decision:** gRPC over UDS provides lowest latency for local IPC while maintaining streaming support and strong typing.

### Socket Paths

| Provider | Socket Path |
|----------|-------------|
| F5-TTS MLX | `unix:///tmp/omnivoice-f5tts.sock` |
| Qwen3-TTS | `unix:///tmp/omnivoice-qwen3tts.sock` |
| Piper | `unix:///tmp/omnivoice-piper.sock` |

## Protobuf Definition

Location: `omnivoice-core/proto/localvoice/v1/localvoice.proto`

```protobuf
syntax = "proto3";

package omnivoice.localvoice.v1;

option go_package = "github.com/plexusone/omnivoice-core/proto/localvoice/v1;localvoicev1";

// LocalVoice service for local TTS inference
service LocalVoice {
  // Synthesize speech from text, streaming audio chunks
  rpc Synthesize(SynthesizeRequest) returns (stream AudioChunk);

  // Synthesize with a reference audio for voice cloning
  rpc SynthesizeWithReference(ReferenceSynthesizeRequest) returns (stream AudioChunk);

  // Prepare/cache a voice profile embedding for faster synthesis
  rpc PrepareVoiceProfile(PrepareVoiceProfileRequest) returns (PrepareVoiceProfileResponse);

  // Health check and model status
  rpc Health(HealthRequest) returns (HealthResponse);

  // Load model into memory
  rpc LoadModel(LoadModelRequest) returns (LoadModelResponse);

  // Unload model from memory
  rpc UnloadModel(UnloadModelRequest) returns (UnloadModelResponse);

  // Get runtime information (memory, device, etc.)
  rpc RuntimeInfo(RuntimeInfoRequest) returns (RuntimeInfoResponse);
}

message SynthesizeRequest {
  string text = 1;
  string voice_id = 2;           // Voice profile ID or "default"
  AudioFormat format = 3;
  optional float speed = 4;       // Speech rate multiplier (default 1.0)
}

message ReferenceSynthesizeRequest {
  string text = 1;
  bytes reference_audio = 2;      // WAV/PCM audio bytes
  string reference_text = 3;      // Transcript of reference audio
  AudioFormat format = 4;
  optional float speed = 5;
}

message AudioChunk {
  bytes data = 1;                 // Raw audio bytes
  bool is_final = 2;              // True for last chunk
  optional AudioMetadata metadata = 3;  // Only on first chunk
}

message AudioMetadata {
  AudioFormat format = 1;
  int32 sample_rate = 2;
  int32 channels = 3;
  int32 bit_depth = 4;
}

enum AudioFormat {
  AUDIO_FORMAT_UNSPECIFIED = 0;
  AUDIO_FORMAT_WAV = 1;
  AUDIO_FORMAT_PCM_S16LE = 2;     // Raw PCM, signed 16-bit little-endian
  AUDIO_FORMAT_PCM_F32LE = 3;     // Raw PCM, float32 little-endian
  AUDIO_FORMAT_MP3 = 4;
  AUDIO_FORMAT_MULAW_8K = 5;      // G.711 mu-law, 8kHz (telephony)
}

message PrepareVoiceProfileRequest {
  string profile_id = 1;
  bytes reference_audio = 2;
  string reference_text = 3;
}

message PrepareVoiceProfileResponse {
  string profile_id = 1;
  bool cached = 2;                // True if embedding was cached
  int64 embedding_size_bytes = 3;
}

message HealthRequest {}

message HealthResponse {
  bool healthy = 1;
  bool model_loaded = 2;
  string model_name = 3;
  string model_version = 4;
}

message LoadModelRequest {
  optional string model_path = 1; // Override default model path
}

message LoadModelResponse {
  bool success = 1;
  int64 load_time_ms = 2;
  int64 memory_used_mb = 3;
}

message UnloadModelRequest {}

message UnloadModelResponse {
  bool success = 1;
  int64 memory_freed_mb = 2;
}

message RuntimeInfoRequest {}

message RuntimeInfoResponse {
  string device_type = 1;         // "mlx", "mps", "cpu"
  int64 memory_used_mb = 2;
  int64 memory_available_mb = 3;
  string mlx_version = 4;
  string python_version = 5;
}
```

## Go Interface Extensions

Add capability interfaces to `omnivoice-core` for local provider features:

```go
// omnivoice-core/tts/local.go

// StreamingSynthesizer supports streaming audio output
type StreamingSynthesizer interface {
    SynthesizeStream(ctx context.Context, req Request) (<-chan AudioChunk, error)
}

// VoiceCloner supports voice cloning from reference audio
type VoiceCloner interface {
    CloneVoice(ctx context.Context, req CloneVoiceRequest) (*VoiceProfile, error)
}

// ReferenceSynthesizer supports synthesis with reference audio
type ReferenceSynthesizer interface {
    SynthesizeWithReference(ctx context.Context, req ReferenceSynthesizeRequest) (*Response, error)
}

// ProfileCacher supports pre-computing voice embeddings
type ProfileCacher interface {
    PrepareVoiceProfile(ctx context.Context, req PrepareVoiceProfileRequest) (*PreparedProfile, error)
}

// ModelManager supports loading/unloading models
type ModelManager interface {
    LoadModel(ctx context.Context) error
    UnloadModel(ctx context.Context) error
    IsModelLoaded() bool
}

// RuntimeChecker provides runtime information
type RuntimeChecker interface {
    RuntimeInfo(ctx context.Context) (*RuntimeInfo, error)
}

// AudioChunk represents a chunk of streaming audio
type AudioChunk struct {
    Data     []byte
    IsFinal  bool
    Metadata *AudioMetadata
}

// RuntimeInfo contains local runtime details
type RuntimeInfo struct {
    DeviceType        string // "mlx", "mps", "cpu"
    MemoryUsedMB      int64
    MemoryAvailableMB int64
    MLXVersion        string
    PythonVersion     string
}
```

## Provider Implementation: f5tts

Local providers are included in `omnivoice-core/providers/` (thin clients without thick SDK dependencies). Thick providers using official SDKs are in separate `omni-{provider}` modules.

### Provider Structure

```
omnivoice-core/providers/f5tts/
├── f5tts.go             # TTSProvider implementation (gRPC client)
├── f5tts_test.go
├── README.md
└── server/
    ├── requirements.txt
    ├── f5tts_server.py  # Python gRPC server
    ├── generate_proto.sh
    └── run.sh           # Server startup script
```

### Go Client Implementation

```go
// omnivoice-core/providers/f5tts/f5tts.go

package f5tts

import (
    "context"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    omnivoice "github.com/plexusone/omnivoice-core"
    "github.com/plexusone/omnivoice-core/tts"
    pb "github.com/plexusone/omnivoice-core/proto/localvoice/v1"
)

const (
    DefaultSocketPath = "unix:///tmp/omnivoice-f5tts.sock"
    ProviderName      = "f5tts"
)

func init() {
    omnivoice.RegisterTTSProvider(ProviderName, NewProvider, omnivoice.PriorityThick)
}

type Provider struct {
    conn   *grpc.ClientConn
    client pb.LocalVoiceClient
}

func NewProvider(cfg *omnivoice.ProviderConfig) (tts.Provider, error) {
    endpoint := cfg.Endpoint
    if endpoint == "" {
        endpoint = DefaultSocketPath
    }

    conn, err := grpc.NewClient(endpoint,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        return nil, err
    }

    return &Provider{
        conn:   conn,
        client: pb.NewLocalVoiceClient(conn),
    }, nil
}

func (p *Provider) Name() string {
    return ProviderName
}

func (p *Provider) Synthesize(ctx context.Context, req tts.Request) (*tts.Response, error) {
    stream, err := p.client.Synthesize(ctx, &pb.SynthesizeRequest{
        Text:    req.Text,
        VoiceId: req.Voice,
        Format:  toProtoFormat(req.Format),
    })
    if err != nil {
        return nil, err
    }

    // Collect all chunks
    var audio []byte
    for {
        chunk, err := stream.Recv()
        if err != nil {
            break
        }
        audio = append(audio, chunk.Data...)
        if chunk.IsFinal {
            break
        }
    }

    return &tts.Response{
        Audio:      audio,
        Format:     req.Format,
        SampleRate: 24000, // F5-TTS default
    }, nil
}

// SynthesizeStream implements StreamingSynthesizer
func (p *Provider) SynthesizeStream(ctx context.Context, req tts.Request) (<-chan tts.AudioChunk, error) {
    stream, err := p.client.Synthesize(ctx, &pb.SynthesizeRequest{
        Text:    req.Text,
        VoiceId: req.Voice,
        Format:  toProtoFormat(req.Format),
    })
    if err != nil {
        return nil, err
    }

    ch := make(chan tts.AudioChunk)
    go func() {
        defer close(ch)
        for {
            chunk, err := stream.Recv()
            if err != nil {
                return
            }
            ch <- tts.AudioChunk{
                Data:    chunk.Data,
                IsFinal: chunk.IsFinal,
            }
            if chunk.IsFinal {
                return
            }
        }
    }()

    return ch, nil
}

func (p *Provider) Close() error {
    return p.conn.Close()
}
```

### Python gRPC Server

```python
# omni-f5tts/server/f5tts_server.py

import grpc
from concurrent import futures
import logging

import mlx
from f5_tts_mlx import F5TTS

from omnivoice.localvoice.v1 import localvoice_pb2 as pb
from omnivoice.localvoice.v1 import localvoice_pb2_grpc as pb_grpc

class LocalVoiceServicer(pb_grpc.LocalVoiceServicer):
    def __init__(self):
        self.model = None
        self.model_loaded = False

    def LoadModel(self, request, context):
        try:
            self.model = F5TTS()
            self.model_loaded = True
            return pb.LoadModelResponse(
                success=True,
                load_time_ms=0,  # TODO: measure
                memory_used_mb=0,  # TODO: measure
            )
        except Exception as e:
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return pb.LoadModelResponse(success=False)

    def Synthesize(self, request, context):
        if not self.model_loaded:
            context.set_code(grpc.StatusCode.FAILED_PRECONDITION)
            context.set_details("Model not loaded")
            return

        # Generate audio
        audio = self.model.generate(
            text=request.text,
            # voice_id handling...
        )

        # Stream chunks
        chunk_size = 4096
        for i in range(0, len(audio), chunk_size):
            chunk = audio[i:i + chunk_size]
            is_final = i + chunk_size >= len(audio)
            yield pb.AudioChunk(
                data=chunk,
                is_final=is_final,
            )

    def Health(self, request, context):
        return pb.HealthResponse(
            healthy=True,
            model_loaded=self.model_loaded,
            model_name="f5-tts-mlx",
            model_version="1.0.0",
        )

    def RuntimeInfo(self, request, context):
        return pb.RuntimeInfoResponse(
            device_type="mlx",
            memory_used_mb=0,  # TODO
            memory_available_mb=0,  # TODO
            mlx_version=mlx.__version__,
        )


def serve(socket_path: str = "/tmp/omnivoice-f5tts.sock"):
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    pb_grpc.add_LocalVoiceServicer_to_server(LocalVoiceServicer(), server)
    server.add_insecure_port(f"unix://{socket_path}")
    server.start()
    logging.info(f"F5-TTS gRPC server listening on {socket_path}")
    server.wait_for_termination()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    serve()
```

## Voice Profile Storage

### Directory Structure

```
~/.omnivoice/
├── config.yaml                    # Global config
└── voice-profiles/
    └── {profile-id}/
        ├── manifest.json          # Profile metadata
        ├── reference.wav          # Canonical reference audio
        ├── reference.txt          # Reference transcript
        └── embeddings/
            ├── f5tts-mlx/
            │   └── embedding.bin  # Cached F5-TTS embedding
            └── qwen3-tts/
                └── embedding.bin  # Cached Qwen3-TTS embedding
```

### Manifest Schema

```json
{
  "id": "john",
  "name": "John's Voice",
  "language": "en",
  "created_at": "2024-01-15T10:30:00Z",
  "reference": {
    "audio_file": "reference.wav",
    "transcript_file": "reference.txt",
    "duration_seconds": 8.5,
    "sample_rate": 24000
  },
  "embeddings": {
    "f5tts-mlx": {
      "file": "embeddings/f5tts-mlx/embedding.bin",
      "created_at": "2024-01-15T10:31:00Z",
      "model_version": "1.0.0"
    }
  }
}
```

## Error Handling

### gRPC Status Codes

| Scenario | gRPC Code | Description |
|----------|-----------|-------------|
| Model not loaded | `FAILED_PRECONDITION` | Call LoadModel first |
| Invalid voice profile | `NOT_FOUND` | Profile doesn't exist |
| Out of memory | `RESOURCE_EXHAUSTED` | Not enough GPU memory |
| Synthesis failed | `INTERNAL` | Model inference error |
| Server unavailable | `UNAVAILABLE` | Server not running |

### Go Error Types

```go
// omnivoice-core/tts/errors.go

var (
    ErrModelNotLoaded     = errors.New("model not loaded")
    ErrProfileNotFound    = errors.New("voice profile not found")
    ErrResourceExhausted  = errors.New("insufficient memory")
    ErrServerUnavailable  = errors.New("local server unavailable")
)
```

## Testing Strategy

### Unit Tests (Go)

- Mock gRPC client for provider tests
- Test streaming audio collection
- Test error handling and retries

### Integration Tests

- Start Python server in subprocess
- Run synthesis through full stack
- Verify audio output format

### Benchmark Tests

- Measure time to first byte
- Measure total synthesis latency
- Measure gRPC overhead vs HTTP baseline

## Security Considerations

1. **UDS permissions** - Socket files should be user-readable only (0600)
2. **No network exposure** - UDS is local-only by design
3. **Model integrity** - Verify model checksums on load
4. **Profile isolation** - Voice profiles stored in user home directory

## Dependencies

### Go Dependencies

```
github.com/plexusone/omnivoice-core
google.golang.org/grpc
google.golang.org/protobuf
```

### Python Dependencies

```
grpcio>=1.60.0
grpcio-tools>=1.60.0
mlx>=0.5.0
f5-tts-mlx>=1.0.0
```

## References

- [gRPC Go](https://grpc.io/docs/languages/go/)
- [gRPC Python](https://grpc.io/docs/languages/python/)
- [Unix Domain Sockets](https://en.wikipedia.org/wiki/Unix_domain_socket)
- [F5-TTS MLX](https://github.com/lucasnewman/f5-tts-mlx)
- [MLX Framework](https://ml-explore.github.io/mlx/)

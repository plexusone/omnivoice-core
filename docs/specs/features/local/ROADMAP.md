# Local Voice Providers - Implementation Roadmap

This document tracks the implementation phases for local TTS providers in OmniVoice.

## Overview

```
Phase 1: Foundation     → Proto + interfaces + test harness
Phase 2: F5-TTS MLX     → First provider end-to-end
Phase 3: Voice Profiles → Storage + caching
Phase 4: Streaming      → Low-latency audio output
Phase 5: Additional     → Qwen3-TTS, Piper, Apple
```

## Phase 1: Foundation

**Goal:** Establish gRPC contract and Go interfaces

### Tasks

| Task | Package | Status |
|------|---------|--------|
| Create proto directory structure | `omnivoice-core/proto/localvoice/v1/` | ✅ Complete |
| Define `localvoice.proto` | `omnivoice-core/proto/localvoice/v1/` | ✅ Complete |
| Generate Go code from proto | `omnivoice-core/proto/localvoice/v1/` | ✅ Complete |
| Add capability interfaces | `omnivoice-core/tts/local.go` | ✅ Complete |
| Add `AudioChunk` type | `omnivoice-core/tts/tts.go` | ✅ Already exists (`StreamChunk`) |
| Add `RuntimeInfo` type | `omnivoice-core/tts/local.go` | ✅ Complete |
| Update `ProviderConfig` with `Endpoint` | `omnivoice-core/registry/registry.go` | ✅ Complete |
| Add `WithEndpoint` option | `omnivoice-core/registry/options.go` | ✅ Complete |
| Add buf/protoc configuration | `omnivoice-core/proto/` | ✅ Complete |

### Proto Generation

```bash
# Option 1: Using buf (recommended)
brew install buf
cd omnivoice-core && buf generate proto

# Option 2: Using protoc directly
brew install protobuf
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
cd omnivoice-core/proto && make generate-protoc
```

### Deliverables

- [x] `omnivoice-core/proto/localvoice/v1/localvoice.proto`
- [x] `omnivoice-core/proto/localvoice/v1/localvoice.pb.go`
- [x] `omnivoice-core/proto/localvoice/v1/localvoice_grpc.pb.go`
- [x] `omnivoice-core/tts/local.go` (capability interfaces)
- [x] `omnivoice-core/registry/registry.go` (`Endpoint` field)
- [x] `omnivoice-core/registry/options.go` (`WithEndpoint` option)
- [x] `omnivoice-core/proto/buf.yaml` (buf configuration)
- [x] `omnivoice-core/buf.gen.yaml` (buf generation config - in project root)
- [x] `omnivoice-core/proto/Makefile` (build helpers)
- [x] `omnivoice-core/go.mod` (added grpc + protobuf dependencies)

---

## Phase 2: F5-TTS MLX Provider

**Goal:** End-to-end local TTS with F5-TTS MLX

### Tasks

| Task | Package | Status |
|------|---------|--------|
| Create f5tts provider | `omnivoice-core/providers/f5tts/` | ✅ Complete |
| Implement gRPC client | `providers/f5tts/f5tts.go` | ✅ Complete |
| Register provider via `init()` | `providers/f5tts/f5tts.go` | ✅ Complete |
| Create Python gRPC server | `providers/f5tts/server/f5tts_server.py` | ✅ Complete |
| Generate Python proto stubs | `providers/f5tts/server/` | Pending (run `./generate_proto.sh`) |
| Add server startup script | `providers/f5tts/server/run.sh` | ✅ Complete |
| Add requirements.txt | `providers/f5tts/server/requirements.txt` | ✅ Complete |
| Write integration tests | `providers/f5tts/f5tts_test.go` | Pending |

### Provider Structure

```
omnivoice-core/providers/f5tts/
├── f5tts.go            # gRPC client implementing TTS interfaces
├── README.md
└── server/
    ├── f5tts_server.py
    ├── requirements.txt
    ├── generate_proto.sh
    └── run.sh
│   └── options.go
└── server/
    ├── requirements.txt
    ├── f5tts_server.py
    ├── run.sh
    └── proto/
        └── (generated Python stubs)
```

### Validation

```bash
# Start server
cd omni-f5tts/server && ./run.sh

# Run Go tests
cd omni-f5tts && go test -v ./...

# Manual test
go run ./examples/local-tts/main.go --text "Hello from F5-TTS"
```

### Deliverables

- [ ] `omni-f5tts` Go module
- [ ] Python gRPC server for F5-TTS MLX
- [ ] Integration tests passing
- [ ] Example application

---

## Phase 3: Voice Profile System

**Goal:** Store and manage voice profiles for voice cloning

### Tasks

| Task | Package | Status |
|------|---------|--------|
| Define profile manifest schema | `omnivoice-core/tts/profile.go` | Pending |
| Implement profile storage | `omnivoice-core/tts/profile.go` | Pending |
| Add profile manager interface | `omnivoice-core/tts/profile.go` | Pending |
| Implement `SynthesizeWithReference` | `omni-f5tts/omnivoice/tts.go` | Pending |
| Implement `PrepareVoiceProfile` | `omni-f5tts/omnivoice/tts.go` | Pending |
| Add embedding caching | `omni-f5tts/server/f5tts_server.py` | Pending |
| Write profile tests | `omnivoice-core/tts/profile_test.go` | Pending |

### Profile Manager API

```go
// omnivoice-core/tts/profile.go

type ProfileManager interface {
    Create(ctx context.Context, req CreateProfileRequest) (*VoiceProfile, error)
    Get(ctx context.Context, id string) (*VoiceProfile, error)
    List(ctx context.Context) ([]*VoiceProfile, error)
    Delete(ctx context.Context, id string) error
}

type VoiceProfile struct {
    ID          string
    Name        string
    Language    string
    CreatedAt   time.Time
    ReferencePath string
    Embeddings  map[string]EmbeddingInfo  // provider -> embedding
}
```

### Deliverables

- [ ] Profile storage implementation
- [ ] Voice cloning working end-to-end
- [ ] Embedding caching working
- [ ] Profile management CLI (optional)

---

## Phase 4: Streaming Audio

**Goal:** Stream audio output for lower latency

### Tasks

| Task | Package | Status |
|------|---------|--------|
| Implement `SynthesizeStream` in Go client | `omni-f5tts/omnivoice/tts.go` | Pending |
| Add chunked audio output in Python server | `omni-f5tts/server/f5tts_server.py` | Pending |
| Add streaming playback example | `omni-f5tts/examples/` | Pending |
| Benchmark time to first byte | `omni-f5tts/omnivoice/` | Pending |
| Optimize chunk size | `omni-f5tts/server/` | Pending |

### Streaming Flow

```
Go Client                    Python Server
    |                              |
    |------ Synthesize(text) ----->|
    |                              | (start generation)
    |<----- AudioChunk[0] ---------|  <- first byte target: <500ms
    |<----- AudioChunk[1] ---------|
    |<----- AudioChunk[2] ---------|
    |<----- AudioChunk[n] final ---|
    |                              |
```

### Deliverables

- [ ] Streaming synthesis working
- [ ] Time to first byte < 500ms
- [ ] Streaming playback example

---

## Phase 5: Additional Providers

**Goal:** Add more local TTS backends

### Qwen3-TTS

| Task | Package | Status |
|------|---------|--------|
| Create `omni-qwen` module | `omni-qwen/` | Pending |
| Implement gRPC client | `omni-qwen/omnivoice/tts.go` | Pending |
| Create Python gRPC server | `omni-qwen/server/qwen_server.py` | Pending |

### Piper

| Task | Package | Status |
|------|---------|--------|
| Create `omni-piper` module | `omni-piper/` | Pending |
| Implement gRPC client | `omni-piper/omnivoice/tts.go` | Pending |
| Create Go server (Piper is Go-native) | `omni-piper/server/` | Pending |

### Apple AVSpeechSynthesizer

| Task | Package | Status |
|------|---------|--------|
| Create `omni-apple` module | `omni-apple/` | Pending |
| Implement via CGO or subprocess | `omni-apple/omnivoice/tts.go` | Pending |

### Deliverables

- [ ] `omni-qwen` provider
- [ ] `omni-piper` provider
- [ ] `omni-apple` provider

---

## Milestone Summary

| Milestone | Target | Dependencies |
|-----------|--------|--------------|
| M1: Proto + Interfaces | Phase 1 complete | None |
| M2: F5-TTS Working | Phase 2 complete | M1 |
| M3: Voice Cloning | Phase 3 complete | M2 |
| M4: Streaming | Phase 4 complete | M2 |
| M5: Multi-Provider | Phase 5 complete | M1 |

## Testing Matrix

| Provider | Unit Tests | Integration | Benchmark |
|----------|------------|-------------|-----------|
| f5tts | Required | Required | Required |
| qwen | Required | Required | Optional |
| piper | Required | Required | Optional |
| apple | Required | Required | Optional |

## Documentation

| Document | Location | Status |
|----------|----------|--------|
| PRD | `docs/specs/features/local/PRD.md` | Complete |
| TRD | `docs/specs/features/local/TRD.md` | Complete |
| Roadmap | `docs/specs/features/local/ROADMAP.md` | Complete |
| API Reference | `docs/local-voice.md` | Pending |
| Getting Started | `docs/guides/local-tts.md` | Pending |

## Open Questions

1. **Model distribution** - Should we include model download scripts, or assume users have models?
2. **Server lifecycle** - Should Go client auto-start Python server, or require manual startup?
3. **Multi-model** - Can multiple models share a single gRPC server, or one server per model?
4. **Windows/Linux** - Future consideration for non-MLX runtimes (ONNX, PyTorch)?

## References

- [PRD.md](./PRD.md) - Product requirements
- [TRD.md](./TRD.md) - Technical requirements
- [IDEATION_CHAT_LOCAL.md](../../../../omnivoice/IDEATION_CHAT_LOCAL.md) - Original ideation

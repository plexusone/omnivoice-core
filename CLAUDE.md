# CLAUDE.md

Project-specific guidelines for omnivoice-core.

## Overview

`omnivoice-core` is the core voice abstraction library providing:

- **Provider interfaces** - STT, TTS, and CallSystem provider contracts (`stt/`, `tts/`, `callsystem/`)
- **Global registry** - Priority-based provider registration (`registry.go`)
- **Gateway framework** - Voice gateway interfaces and types (`gateway/`)
- **Session storage** - Call state persistence (`storage/`)
- **Barge-in detection** - User interruption handling (`bargein/`)
- **Audio codecs** - Encoding/decoding utilities (`codec/`)

## Package Architecture

```
omnivoice-core/           # Core interfaces + global registry
    ├── registry.go       # Global provider registration
    ├── stt/              # STT provider interface
    ├── tts/              # TTS provider interface
    ├── callsystem/       # CallSystem provider interface
    ├── gateway/          # Voice gateway framework
    ├── storage/          # Session persistence
    ├── bargein/          # Interruption detection
    └── codec/            # Audio encoding/decoding

omni-deepgram/            # Deepgram STT/TTS provider
omni-openai/              # OpenAI TTS provider
omni-elevenlabs/          # ElevenLabs TTS provider
omni-twilio/              # Twilio CallSystem provider
omni-telnyx/              # Telnyx CallSystem provider

omnivoice/                # Batteries-included package
    └── providers/all/    # Imports all provider packages
```

## Provider Registry Pattern

### Priority System

Providers register with a priority level:

| Priority | Constant | Description |
|----------|----------|-------------|
| 0 | `PriorityThin` | Stdlib-only implementations (no external dependencies) |
| 10 | `PriorityThick` | Official SDK implementations (full feature support) |

Higher priority providers override lower priority registrations for the same name.

### Registration via init()

Provider packages use `init()` to auto-register when imported:

```go
// In omni-deepgram/stt.go
package omnideepgram

import omnivoice "github.com/plexusone/omnivoice-core"

func init() {
    omnivoice.RegisterSTTProvider("deepgram", NewSTTProvider, omnivoice.PriorityThick)
}

func NewSTTProvider(cfg *registry.ProviderConfig) (stt.Provider, error) {
    // Create and return provider
}
```

### Using Providers

```go
import (
    omnivoice "github.com/plexusone/omnivoice-core"
    _ "github.com/plexusone/omni-deepgram"  // Auto-registers "deepgram"
)

func main() {
    // Get registered provider
    sttProvider, err := omnivoice.GetSTTProvider("deepgram",
        registry.WithAPIKey(os.Getenv("DEEPGRAM_API_KEY")),
    )

    // List available providers
    names := omnivoice.ListSTTProviders()

    // Check if provider exists
    if omnivoice.HasSTTProvider("deepgram") {
        // ...
    }
}
```

## Dependency Architecture

```
                    ┌─────────────────┐
                    │  omnivoice-core │  ← Core interfaces + registry
                    └────────┬────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
        ▼                    ▼                    ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────┐
│ omni-deepgram │   │  omni-twilio  │   │ omni-elevenlabs│  ← Provider packages
└───────┬───────┘   └───────┬───────┘   └───────┬───────┘
        │                   │                   │
        └───────────────────┼───────────────────┘
                            │
                            ▼
                    ┌───────────────┐
                    │   omnivoice   │  ← Batteries-included (imports all providers)
                    └───────────────┘
```

**Key Rules:**

1. **Provider packages depend on omnivoice-core, NOT omnivoice**
2. **omnivoice imports providers** via `_ "github.com/plexusone/omni-*/..."`
3. **Applications choose** which to import:
   - Import `omnivoice` for all providers (batteries-included)
   - Import specific provider packages for minimal dependencies

## Adding a New Provider

1. Create a new module (e.g., `omni-newprovider/`)

2. Implement the provider interface:
   ```go
   type STTProvider struct { /* ... */ }

   func (p *STTProvider) Transcribe(ctx context.Context, audio io.Reader) (*stt.Result, error) {
       // Implementation
   }
   ```

3. Register in `init()`:
   ```go
   func init() {
       omnivoice.RegisterSTTProvider("newprovider", NewSTTProvider, omnivoice.PriorityThick)
   }
   ```

4. Add import to `omnivoice/providers/all/all.go`:
   ```go
   _ "github.com/plexusone/omni-newprovider"
   ```

## Testing

Run all tests:

```bash
go test ./...
```

Run with verbose output:

```bash
go test -v ./...
```

## Linting

```bash
golangci-lint run
```

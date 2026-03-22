# Provider Registry

The `registry` package provides types for provider registration and discovery, enabling dynamic provider selection and configuration at runtime.

## Overview

The registry pattern allows applications to:

- Register provider factories at startup
- Discover available providers at runtime
- Create providers with unified configuration
- Support plugin-style extensibility

```
┌─────────────────────────────────────────────────────────────────┐
│                         Application                              │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                       Registry                            │   │
│  ├──────────────────┬───────────────────┬───────────────────┤   │
│  │   TTS Providers  │   STT Providers   │ CallSystem Provs  │   │
│  │                  │                   │                   │   │
│  │  - elevenlabs    │  - deepgram       │  - twilio         │   │
│  │  - openai        │  - openai         │  - telnyx         │   │
│  │  - google        │  - google         │  - vonage         │   │
│  │  - azure         │  - assemblyai     │  - bandwidth      │   │
│  └──────────────────┴───────────────────┴───────────────────┘   │
│                              │                                   │
│                              ▼                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                   Provider Creation                       │   │
│  │                                                           │   │
│  │   provider, err := registry.GetTTSProvider("elevenlabs",  │   │
│  │       registry.WithAPIKey(apiKey),                        │   │
│  │   )                                                       │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Registry Interface

The `Registry` interface defines methods for each provider category:

```go
type Registry interface {
    // TTS providers
    RegisterTTSProvider(name string, factory TTSProviderFactory)
    GetTTSProvider(name string, opts ...ProviderOption) (tts.Provider, error)
    ListTTSProviders() []string
    HasTTSProvider(name string) bool

    // STT providers
    RegisterSTTProvider(name string, factory STTProviderFactory)
    GetSTTProvider(name string, opts ...ProviderOption) (stt.Provider, error)
    ListSTTProviders() []string
    HasSTTProvider(name string) bool

    // CallSystem providers
    RegisterCallSystemProvider(name string, factory CallSystemProviderFactory)
    GetCallSystemProvider(name string, opts ...ProviderOption) (callsystem.CallSystem, error)
    ListCallSystemProviders() []string
    HasCallSystemProvider(name string) bool
}
```

## Provider Factories

Factories are functions that create providers from configuration:

```go
// TTS provider factory
type TTSProviderFactory func(config ProviderConfig) (tts.Provider, error)

// STT provider factory
type STTProviderFactory func(config ProviderConfig) (stt.Provider, error)

// CallSystem provider factory
type CallSystemProviderFactory func(config ProviderConfig) (callsystem.CallSystem, error)
```

## Provider Configuration

### ProviderConfig

Common configuration for all providers:

```go
type ProviderConfig struct {
    // APIKey is the authentication key for the provider
    APIKey string

    // BaseURL is an optional custom API endpoint
    BaseURL string

    // Extensions holds provider-specific configuration
    Extensions map[string]any
}
```

### ProviderOption

Functional options for configuring providers:

```go
type ProviderOption func(*ProviderConfig)

// Apply options to create config
config := registry.ApplyOptions(
    registry.WithAPIKey(apiKey),
    registry.WithBaseURL(customURL),
)
```

## Usage Examples

### Registering Providers

Register provider factories at application startup:

```go
import (
    "github.com/plexusone/omnivoice-core/registry"
    "github.com/plexusone/omnivoice-core/tts"
)

func init() {
    // Register ElevenLabs TTS provider
    registry.RegisterTTSProvider("elevenlabs", func(cfg registry.ProviderConfig) (tts.Provider, error) {
        return elevenlabs.New(
            elevenlabs.WithAPIKey(cfg.APIKey),
        )
    })

    // Register OpenAI TTS provider
    registry.RegisterTTSProvider("openai", func(cfg registry.ProviderConfig) (tts.Provider, error) {
        return openaitts.New(
            openaitts.WithAPIKey(cfg.APIKey),
        )
    })
}
```

### Getting Providers

Retrieve providers by name with configuration options:

```go
// Get TTS provider
ttsProvider, err := registry.GetTTSProvider("elevenlabs",
    registry.WithAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
)
if err != nil {
    log.Fatal(err)
}

// Get STT provider
sttProvider, err := registry.GetSTTProvider("deepgram",
    registry.WithAPIKey(os.Getenv("DEEPGRAM_API_KEY")),
)
if err != nil {
    log.Fatal(err)
}

// Get CallSystem provider
callProvider, err := registry.GetCallSystemProvider("twilio",
    registry.WithAPIKey(os.Getenv("TWILIO_AUTH_TOKEN")),
    registry.WithExtension("account_sid", os.Getenv("TWILIO_ACCOUNT_SID")),
)
if err != nil {
    log.Fatal(err)
}
```

### Listing Available Providers

Discover what providers are registered:

```go
// List all TTS providers
ttsProviders := registry.ListTTSProviders()
fmt.Println("Available TTS providers:", ttsProviders)
// Output: Available TTS providers: [elevenlabs openai google azure]

// Check if a provider exists
if registry.HasSTTProvider("deepgram") {
    // Use Deepgram
}
```

### Provider-Specific Configuration

Use `Extensions` for provider-specific settings:

```go
provider, err := registry.GetCallSystemProvider("twilio",
    registry.WithAPIKey(authToken),
    registry.WithExtension("account_sid", accountSID),
    registry.WithExtension("region", "us1"),
    registry.WithExtension("edge", "ashburn"),
)
```

## Configuration-Driven Provider Selection

### From Environment Variables

```go
func getProviderFromEnv(providerType, envPrefix string) (string, []registry.ProviderOption) {
    name := os.Getenv(envPrefix + "_PROVIDER")
    apiKey := os.Getenv(envPrefix + "_API_KEY")

    opts := []registry.ProviderOption{
        registry.WithAPIKey(apiKey),
    }

    if baseURL := os.Getenv(envPrefix + "_BASE_URL"); baseURL != "" {
        opts = append(opts, registry.WithBaseURL(baseURL))
    }

    return name, opts
}

// Usage
ttsName, ttsOpts := getProviderFromEnv("tts", "TTS")
ttsProvider, err := registry.GetTTSProvider(ttsName, ttsOpts...)
```

### From Configuration File

```go
type VoiceConfig struct {
    TTS struct {
        Provider string            `yaml:"provider"`
        APIKey   string            `yaml:"api_key"`
        Options  map[string]any    `yaml:"options"`
    } `yaml:"tts"`
    STT struct {
        Provider string            `yaml:"provider"`
        APIKey   string            `yaml:"api_key"`
        Options  map[string]any    `yaml:"options"`
    } `yaml:"stt"`
}

func createProvidersFromConfig(cfg VoiceConfig) (tts.Provider, stt.Provider, error) {
    ttsOpts := []registry.ProviderOption{
        registry.WithAPIKey(cfg.TTS.APIKey),
    }
    for k, v := range cfg.TTS.Options {
        ttsOpts = append(ttsOpts, registry.WithExtension(k, v))
    }

    ttsProvider, err := registry.GetTTSProvider(cfg.TTS.Provider, ttsOpts...)
    if err != nil {
        return nil, nil, fmt.Errorf("tts provider: %w", err)
    }

    sttOpts := []registry.ProviderOption{
        registry.WithAPIKey(cfg.STT.APIKey),
    }
    for k, v := range cfg.STT.Options {
        sttOpts = append(sttOpts, registry.WithExtension(k, v))
    }

    sttProvider, err := registry.GetSTTProvider(cfg.STT.Provider, sttOpts...)
    if err != nil {
        return nil, nil, fmt.Errorf("stt provider: %w", err)
    }

    return ttsProvider, sttProvider, nil
}
```

## CallSystem Client

The `callsystem` package includes a `Client` type that manages multiple providers with automatic failover:

```go
import "github.com/plexusone/omnivoice-core/callsystem"

// Create providers
twilioProvider, _ := registry.GetCallSystemProvider("twilio", ...)
telnyxProvider, _ := registry.GetCallSystemProvider("telnyx", ...)

// Create client with multiple providers
client := callsystem.NewClient(twilioProvider, telnyxProvider)
client.SetPrimary("twilio")
client.SetFallbacks("telnyx")

// MakeCall automatically falls back on failure
call, err := client.MakeCall(ctx, "+15559876543",
    callsystem.WithFrom("+15551234567"),
)
```

See the [CallSystem Client](#callsystem-client) section for more details.

## Implementing a Custom Registry

The `registry` package provides types; implement your own registry:

```go
type MyRegistry struct {
    ttsFactories  map[string]registry.TTSProviderFactory
    sttFactories  map[string]registry.STTProviderFactory
    callFactories map[string]registry.CallSystemProviderFactory
    mu            sync.RWMutex
}

func NewRegistry() *MyRegistry {
    return &MyRegistry{
        ttsFactories:  make(map[string]registry.TTSProviderFactory),
        sttFactories:  make(map[string]registry.STTProviderFactory),
        callFactories: make(map[string]registry.CallSystemProviderFactory),
    }
}

func (r *MyRegistry) RegisterTTSProvider(name string, factory registry.TTSProviderFactory) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.ttsFactories[name] = factory
}

func (r *MyRegistry) GetTTSProvider(name string, opts ...registry.ProviderOption) (tts.Provider, error) {
    r.mu.RLock()
    factory, ok := r.ttsFactories[name]
    r.mu.RUnlock()

    if !ok {
        return nil, fmt.Errorf("unknown TTS provider: %s", name)
    }

    config := registry.ApplyOptions(opts...)
    return factory(config)
}

// ... implement remaining methods
```

## Best Practices

1. **Register at init()** - Register provider factories in `init()` functions for early availability

2. **Validate configuration** - Factories should validate required fields and return descriptive errors

3. **Use Extensions sparingly** - Prefer typed configuration in factories over generic Extensions

4. **Thread-safe registration** - Registries should be safe for concurrent access

5. **Lazy initialization** - Create providers when needed, not at registration time

6. **Provider names** - Use lowercase, hyphenated names (e.g., "elevenlabs", "google-cloud")

## API Reference

See the [GoDoc](https://pkg.go.dev/github.com/plexusone/omnivoice-core/registry) for complete API documentation.

# Provider Registry

OmniVoice provides a **global provider registry** with priority-based registration, enabling provider packages to register themselves via `init()` without circular dependencies.

## Overview

The registry pattern allows applications to:

- Register provider factories at startup via `init()`
- Support thin (stdlib-only) vs thick (SDK-based) provider layering
- Discover available providers at runtime
- Create providers with unified configuration
- Support plugin-style extensibility

## Global Registry (v0.11.0+)

The global registry is accessed via package-level functions in `omnivoice-core`:

```go
import omnivoice "github.com/plexusone/omnivoice-core"

// Registration (typically in provider package init())
omnivoice.RegisterSTTProvider("deepgram", factory, omnivoice.PriorityThick)
omnivoice.RegisterTTSProvider("elevenlabs", factory, omnivoice.PriorityThick)
omnivoice.RegisterCallSystemProvider("twilio", factory, omnivoice.PriorityThick)

// Retrieval
stt, err := omnivoice.GetSTTProvider("deepgram", registry.WithAPIKey(key))
tts, err := omnivoice.GetTTSProvider("elevenlabs", registry.WithAPIKey(key))

// Discovery
names := omnivoice.ListSTTProviders()      // ["deepgram", "openai", ...]
exists := omnivoice.HasTTSProvider("elevenlabs")
priority := omnivoice.GetSTTProviderPriority("deepgram")  // 10
```

### Gateway and Realtime Registries (v0.14.0+)

Voice gateway and realtime (voice-to-voice) providers also have registry support:

```go
import (
    omnivoice "github.com/plexusone/omnivoice-core"
    "github.com/plexusone/omnivoice-core/registry"
    _ "github.com/plexusone/omni-twilio/omnivoice/gateway"   // Auto-registers "twilio"
    _ "github.com/plexusone/omni-openai/omnivoice/realtime"  // Auto-registers "openai"
)

// Gateway providers (Twilio, Telnyx)
omnivoice.RegisterGatewayProvider("twilio", factory, omnivoice.PriorityThick)
gateway, err := omnivoice.GetGatewayProvider("twilio",
    registry.WithAccountSID(accountSID),
    registry.WithAuthToken(authToken),
    registry.WithPhoneNumber("+15551234567"),
)
names := omnivoice.ListGatewayProviders()     // ["twilio", "telnyx"]
exists := omnivoice.HasGatewayProvider("twilio")

// Realtime providers (OpenAI Realtime, Gemini Live)
omnivoice.RegisterRealtimeProvider("openai", factory, omnivoice.PriorityThick)
realtime, err := omnivoice.GetRealtimeProvider("openai",
    registry.WithAPIKey(apiKey),
    registry.WithModel("gpt-4o-realtime-preview"),
    registry.WithVoice("alloy"),
)
names := omnivoice.ListRealtimeProviders()    // ["openai", "gemini"]
exists := omnivoice.HasRealtimeProvider("gemini")
```

### Priority System

Providers register with a priority level. Higher priority overrides lower:

| Constant | Value | Description |
|----------|-------|-------------|
| `PriorityThin` | 0 | Stdlib-only implementations (no external dependencies) |
| `PriorityThick` | 10 | Official SDK implementations (full feature support) |

This enables thin/thick layering:

```go
// In omnivoice-core (thin provider, stdlib HTTP)
omnivoice.RegisterSTTProvider("deepgram", thinFactory, omnivoice.PriorityThin)

// In omni-deepgram (thick provider, uses official SDK)
omnivoice.RegisterSTTProvider("deepgram", thickFactory, omnivoice.PriorityThick)

// Application imports both → thick wins (priority 10 > 0)
```

### Dependency Architecture

```
omnivoice-core           ← Core interfaces + global registry
     ↑
provider packages        ← Register via init(), depend on omnivoice-core
(omni-deepgram, etc.)      NOT on omnivoice (avoids circular deps)
     ↑
omnivoice               ← Batteries-included (imports all providers)
```

## Local Registry

```
┌───────────────────────────────────────────────────────────────────────────────┐
│                                  Application                                  │
│                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │                              Registry                                   │  │
│  ├─────────────┬─────────────┬─────────────┬─────────────┬─────────────────┤  │
│  │ TTS Provs   │ STT Provs   │ CallSystem  │ Gateway     │ Realtime        │  │
│  │             │             │             │             │                 │  │
│  │ - elevenlabs│ - deepgram  │ - twilio    │ - twilio    │ - openai        │  │
│  │ - openai    │ - openai    │ - telnyx    │ - telnyx    │ - gemini        │  │
│  │ - google    │ - google    │ - vonage    │             │                 │  │
│  │ - azure     │ - assemblyai│ - bandwidth │             │                 │  │
│  └─────────────┴─────────────┴─────────────┴─────────────┴─────────────────┘  │
│                                      │                                        │
│                                      ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │                         Provider Creation                               │  │
│  │                                                                         │  │
│  │   // Voice pipeline providers                                           │  │
│  │   tts, _ := omnivoice.GetTTSProvider("elevenlabs", opts...)             │  │
│  │   stt, _ := omnivoice.GetSTTProvider("deepgram", opts...)               │  │
│  │                                                                         │  │
│  │   // Gateway and realtime providers (v0.14.0+)                          │  │
│  │   gateway, _ := omnivoice.GetGatewayProvider("twilio", opts...)         │  │
│  │   realtime, _ := omnivoice.GetRealtimeProvider("openai", opts...)       │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│                                                                               │
└───────────────────────────────────────────────────────────────────────────────┘
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

    // Gateway providers (v0.14.0+)
    RegisterGatewayProvider(name string, factory GatewayProviderFactory)
    GetGatewayProvider(name string, opts ...ProviderOption) (Gateway, error)
    ListGatewayProviders() []string
    HasGatewayProvider(name string) bool

    // Realtime providers (v0.14.0+)
    RegisterRealtimeProvider(name string, factory RealtimeProviderFactory)
    GetRealtimeProvider(name string, opts ...ProviderOption) (RealtimeProvider, error)
    ListRealtimeProviders() []string
    HasRealtimeProvider(name string) bool
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

// Gateway provider factory (v0.14.0+)
type GatewayProviderFactory func(config ProviderConfig) (Gateway, error)

// Realtime provider factory (v0.14.0+)
type RealtimeProviderFactory func(config ProviderConfig) (RealtimeProvider, error)
```

### Gateway and Realtime Interfaces

The registry provides minimal interfaces for Gateway and Realtime providers:

```go
// Gateway interface for voice gateways (PSTN, WebRTC)
type Gateway interface {
    Name() string
    Start(ctx any) error
    Stop() error
}

// RealtimeProvider interface for native voice-to-voice
type RealtimeProvider interface {
    Name() string
    Close() error
}
```

Provider wrappers typically expose a method to access the full concrete type:

```go
// Get the underlying concrete gateway
gw, _ := omnivoice.GetGatewayProvider("twilio", opts...)
if wrapper, ok := gw.(interface{ Gateway() *twilioGateway.Gateway }); ok {
    concreteGW := wrapper.Gateway()
    concreteGW.OnCall(handler)  // Full API access
}
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

### Common Options

The registry provides typed option functions for common configuration:

```go
// Authentication
registry.WithAPIKey(apiKey)
registry.WithAccountSID(accountSID)    // Twilio
registry.WithAuthToken(authToken)      // Twilio

// Server configuration
registry.WithPhoneNumber(number)
registry.WithWebhookURL(url)
registry.WithRegion(region)

// Gateway options (v0.14.0+)
registry.WithListener(listener)        // net.Listener for custom servers
registry.WithPublicURL(url)            // Public URL for webhooks
registry.WithListenAddr(addr)          // Server listen address
registry.WithConnectionID(id)          // Telnyx connection ID

// Realtime options (v0.14.0+)
registry.WithVoice(voice)              // Voice selection
registry.WithModel(model)              // Model selection
registry.WithInstructions(prompt)      // System prompt
```

### Pipeline Configuration Options (v0.14.0+)

Type-safe options for voice pipeline configuration:

```go
// STT configuration
registry.WithSTTProvider("deepgram")
registry.WithSTTAPIKey(apiKey)
registry.WithSTTModel("nova-2")
registry.WithSTTLanguage("en-US")

// TTS configuration
registry.WithTTSProvider("elevenlabs")
registry.WithTTSAPIKey(apiKey)
registry.WithTTSVoiceID("rachel")
registry.WithTTSModel("eleven_turbo_v2")

// LLM configuration
registry.WithLLMProvider("anthropic")
registry.WithLLMAPIKey(apiKey)
registry.WithLLMModel("claude-sonnet-4-20250514")
registry.WithLLMSystemPrompt("You are a helpful assistant.")

// Session configuration
registry.WithGreeting("Hello, how can I help?")
registry.WithMaxSessionDuration(30 * time.Minute)
registry.WithInterruptionMode("immediate")
registry.WithLogger(logger)
registry.WithPipelineMode("realtime")  // "text" or "realtime"
```

### Provider-Specific Options

Provider packages export their own type-safe options for provider-specific configuration:

```go
import (
    "github.com/plexusone/omnivoice-core/registry"
    twilioGateway "github.com/plexusone/omni-twilio/omnivoice/gateway"
)

// Type-safe tool configuration
tools := []twilioGateway.ToolDefinition{
    {Name: "get_weather", Description: "Get weather", Parameters: params},
}
handlers := map[string]twilioGateway.ToolHandler{
    "get_weather": weatherHandler,
}

gateway, err := omnivoice.GetGatewayProvider("twilio",
    // Common registry options
    registry.WithAccountSID(accountSID),
    registry.WithAuthToken(authToken),
    registry.WithSTTProvider("deepgram"),
    registry.WithTTSProvider("elevenlabs"),

    // Provider-specific type-safe options
    twilioGateway.WithTools(tools),
    twilioGateway.WithToolHandlers(handlers),
    twilioGateway.WithRealtimeProviderFactory(factory),
    twilioGateway.WithRealtimeConfig(realtimeConfig),
)
```

## Usage Examples

### Registering Providers

Provider packages register factories in `init()` with priority:

```go
// In omni-elevenlabs/init.go
import (
    omnivoice "github.com/plexusone/omnivoice-core"
    "github.com/plexusone/omnivoice-core/registry"
    "github.com/plexusone/omnivoice-core/tts"
)

func init() {
    // Register ElevenLabs TTS provider (thick - uses SDK)
    omnivoice.RegisterTTSProvider("elevenlabs", func(cfg registry.ProviderConfig) (tts.Provider, error) {
        return elevenlabs.New(
            elevenlabs.WithAPIKey(cfg.APIKey),
        )
    }, omnivoice.PriorityThick)
}

// In omni-openai/init.go
func init() {
    // Register OpenAI TTS provider (thick - uses SDK)
    omnivoice.RegisterTTSProvider("openai", func(cfg registry.ProviderConfig) (tts.Provider, error) {
        return openaitts.New(
            openaitts.WithAPIKey(cfg.APIKey),
        )
    }, omnivoice.PriorityThick)
}
```

### Getting Providers

Retrieve providers by name with configuration options:

```go
import (
    omnivoice "github.com/plexusone/omnivoice-core"
    "github.com/plexusone/omnivoice-core/registry"
    _ "github.com/plexusone/omni-deepgram"    // Auto-registers "deepgram"
    _ "github.com/plexusone/omni-elevenlabs"  // Auto-registers "elevenlabs"
    _ "github.com/plexusone/omni-twilio"      // Auto-registers "twilio"
)

// Get TTS provider
ttsProvider, err := omnivoice.GetTTSProvider("elevenlabs",
    registry.WithAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
)
if err != nil {
    log.Fatal(err)
}

// Get STT provider
sttProvider, err := omnivoice.GetSTTProvider("deepgram",
    registry.WithAPIKey(os.Getenv("DEEPGRAM_API_KEY")),
)
if err != nil {
    log.Fatal(err)
}

// Get CallSystem provider
callProvider, err := omnivoice.GetCallSystemProvider("twilio",
    registry.WithAPIKey(os.Getenv("TWILIO_AUTH_TOKEN")),
    registry.WithExtension("accountSID", os.Getenv("TWILIO_ACCOUNT_SID")),
)
if err != nil {
    log.Fatal(err)
}
```

### Getting Gateway Providers (v0.14.0+)

Create voice gateways via the registry with full pipeline configuration:

```go
import (
    omnivoice "github.com/plexusone/omnivoice-core"
    "github.com/plexusone/omnivoice-core/registry"
    twilioGateway "github.com/plexusone/omni-twilio/omnivoice/gateway"
    _ "github.com/plexusone/omni-twilio/omnivoice/gateway"  // Auto-registers "twilio"
)

// Create Twilio gateway with full pipeline config
gateway, err := omnivoice.GetGatewayProvider("twilio",
    // Credentials
    registry.WithAccountSID(os.Getenv("TWILIO_ACCOUNT_SID")),
    registry.WithAuthToken(os.Getenv("TWILIO_AUTH_TOKEN")),
    registry.WithPhoneNumber("+15551234567"),

    // Server
    registry.WithPublicURL("https://example.ngrok.io"),
    registry.WithListenAddr(":8080"),

    // STT → LLM → TTS pipeline
    registry.WithSTTProvider("deepgram"),
    registry.WithSTTAPIKey(os.Getenv("DEEPGRAM_API_KEY")),
    registry.WithTTSProvider("elevenlabs"),
    registry.WithTTSAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
    registry.WithTTSVoiceID("rachel"),
    registry.WithLLMProvider("anthropic"),
    registry.WithLLMModel("claude-sonnet-4-20250514"),
    registry.WithLLMSystemPrompt("You are a helpful assistant."),

    // Session
    registry.WithGreeting("Hello! How can I help you today?"),
    registry.WithMaxSessionDuration(30 * time.Minute),

    // Provider-specific tools (type-safe)
    twilioGateway.WithTools(tools),
    twilioGateway.WithToolHandlers(handlers),
)
```

### Getting Realtime Providers (v0.14.0+)

Create native voice-to-voice providers:

```go
import (
    omnivoice "github.com/plexusone/omnivoice-core"
    "github.com/plexusone/omnivoice-core/registry"
    _ "github.com/plexusone/omni-openai/omnivoice/realtime"  // Auto-registers "openai"
    _ "github.com/plexusone/omni-google/omnivoice/realtime"  // Auto-registers "gemini"
)

// OpenAI Realtime (~100ms latency)
openaiRT, err := omnivoice.GetRealtimeProvider("openai",
    registry.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    registry.WithModel("gpt-4o-realtime-preview"),
    registry.WithVoice("alloy"),
    registry.WithInstructions("You are a helpful voice assistant."),
)

// Gemini Live (~200ms latency)
geminiRT, err := omnivoice.GetRealtimeProvider("gemini",
    registry.WithAPIKey(os.Getenv("GEMINI_API_KEY")),
    registry.WithModel("gemini-2.0-flash-live"),
    registry.WithVoice("Puck"),
    registry.WithInstructions("You are a helpful voice assistant."),
)
```

### Listing Available Providers

Discover what providers are registered:

```go
// List all TTS providers
ttsProviders := omnivoice.ListTTSProviders()
fmt.Println("Available TTS providers:", ttsProviders)
// Output: Available TTS providers: [elevenlabs openai google azure]

// List gateway providers (v0.14.0+)
gatewayProviders := omnivoice.ListGatewayProviders()
fmt.Println("Available gateway providers:", gatewayProviders)
// Output: Available gateway providers: [twilio telnyx]

// List realtime providers (v0.14.0+)
realtimeProviders := omnivoice.ListRealtimeProviders()
fmt.Println("Available realtime providers:", realtimeProviders)
// Output: Available realtime providers: [openai gemini]

// Check if a provider exists
if omnivoice.HasSTTProvider("deepgram") {
    // Use Deepgram
}

// Check priority (useful for debugging thin/thick layering)
priority := omnivoice.GetSTTProviderPriority("deepgram")
if priority == omnivoice.PriorityThick {
    fmt.Println("Using SDK-based Deepgram provider")
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

// Package registry provides types for provider registration and discovery.
package registry

import (
	"github.com/plexusone/omnivoice-core/callsystem"
	"github.com/plexusone/omnivoice-core/stt"
	"github.com/plexusone/omnivoice-core/tts"
)

// ProviderConfig holds common configuration options for creating providers.
type ProviderConfig struct {
	// APIKey is the authentication key for the provider.
	APIKey string //nolint:gosec // G117: This is a config struct, not storing secrets

	// BaseURL is an optional custom API endpoint.
	BaseURL string

	// Extensions holds provider-specific configuration.
	Extensions map[string]any
}

// ProviderOption configures a ProviderConfig.
type ProviderOption func(*ProviderConfig)

// ApplyOptions applies provider options to a config.
func ApplyOptions(opts ...ProviderOption) ProviderConfig {
	config := ProviderConfig{}
	for _, opt := range opts {
		opt(&config)
	}
	return config
}

// TTSProviderFactory creates a TTS provider with the given configuration.
type TTSProviderFactory func(config ProviderConfig) (tts.Provider, error)

// STTProviderFactory creates an STT provider with the given configuration.
type STTProviderFactory func(config ProviderConfig) (stt.Provider, error)

// CallSystemProviderFactory creates a CallSystem provider with the given configuration.
type CallSystemProviderFactory func(config ProviderConfig) (callsystem.CallSystem, error)

// GatewayProviderFactory creates a voice Gateway with the given configuration.
// Gateway providers handle full-duplex voice calls via telephony providers.
type GatewayProviderFactory func(config ProviderConfig) (Gateway, error)

// Gateway is the interface that voice gateway providers must implement.
// This is defined here to avoid import cycles with the gateway package.
// See gateway.Gateway for the full interface documentation.
type Gateway interface {
	// Name returns the provider name.
	Name() string

	// Start starts the gateway server.
	Start(ctx any) error

	// Stop gracefully shuts down the gateway.
	Stop() error
}

// RealtimeProviderFactory creates realtime voice-to-voice providers.
// Realtime providers enable native voice-to-voice conversations with ~100-300ms latency.
type RealtimeProviderFactory func(config ProviderConfig) (RealtimeProvider, error)

// RealtimeProvider is the interface that realtime voice-to-voice providers must implement.
// This is defined here to avoid import cycles with the realtime package.
// See realtime.Provider for the full interface documentation.
type RealtimeProvider interface {
	// Name returns the provider name (e.g., "openai", "gemini").
	Name() string

	// Close closes the provider and releases resources.
	Close() error
}

// Registry defines the interface for provider discovery.
type Registry interface {
	// TTS registration
	RegisterTTSProvider(name string, factory TTSProviderFactory)
	GetTTSProvider(name string, opts ...ProviderOption) (tts.Provider, error)
	ListTTSProviders() []string
	HasTTSProvider(name string) bool

	// STT registration
	RegisterSTTProvider(name string, factory STTProviderFactory)
	GetSTTProvider(name string, opts ...ProviderOption) (stt.Provider, error)
	ListSTTProviders() []string
	HasSTTProvider(name string) bool

	// CallSystem registration
	RegisterCallSystemProvider(name string, factory CallSystemProviderFactory)
	GetCallSystemProvider(name string, opts ...ProviderOption) (callsystem.CallSystem, error)
	ListCallSystemProviders() []string
	HasCallSystemProvider(name string) bool

	// Gateway registration
	RegisterGatewayProvider(name string, factory GatewayProviderFactory)
	GetGatewayProvider(name string, opts ...ProviderOption) (Gateway, error)
	ListGatewayProviders() []string
	HasGatewayProvider(name string) bool

	// Realtime registration
	RegisterRealtimeProvider(name string, factory RealtimeProviderFactory)
	GetRealtimeProvider(name string, opts ...ProviderOption) (RealtimeProvider, error)
	ListRealtimeProviders() []string
	HasRealtimeProvider(name string) bool
}

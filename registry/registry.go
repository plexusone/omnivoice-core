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
}

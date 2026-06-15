package omnivoice

import (
	"fmt"
	"sync"

	"github.com/plexusone/omnivoice-core/callsystem"
	"github.com/plexusone/omnivoice-core/registry"
	"github.com/plexusone/omnivoice-core/stt"
	"github.com/plexusone/omnivoice-core/tts"
)

// Priority constants for provider registration.
// Higher priority values override lower priority registrations.
const (
	// PriorityThin is the priority for thin (stdlib-only) provider implementations.
	// These have no external dependencies beyond the standard library.
	PriorityThin = 0

	// PriorityThick is the priority for thick (official SDK) provider implementations.
	// These use official provider SDKs for full feature support.
	PriorityThick = 10
)

// registeredSTTProvider holds a factory with its priority.
type registeredSTTProvider struct {
	factory  registry.STTProviderFactory
	priority int
}

// registeredTTSProvider holds a factory with its priority.
type registeredTTSProvider struct {
	factory  registry.TTSProviderFactory
	priority int
}

// registeredCallSystemProvider holds a factory with its priority.
type registeredCallSystemProvider struct {
	factory  registry.CallSystemProviderFactory
	priority int
}

// registeredGatewayProvider holds a factory with its priority.
type registeredGatewayProvider struct {
	factory  registry.GatewayProviderFactory
	priority int
}

// registeredRealtimeProvider holds a factory with its priority.
type registeredRealtimeProvider struct {
	factory  registry.RealtimeProviderFactory
	priority int
}

var (
	sttRegistry        = make(map[string]registeredSTTProvider)
	ttsRegistry        = make(map[string]registeredTTSProvider)
	callSystemRegistry = make(map[string]registeredCallSystemProvider)
	gatewayRegistry    = make(map[string]registeredGatewayProvider)
	realtimeRegistry   = make(map[string]registeredRealtimeProvider)

	sttMu        sync.RWMutex
	ttsMu        sync.RWMutex
	callSystemMu sync.RWMutex
	gatewayMu    sync.RWMutex
	realtimeMu   sync.RWMutex
)

// RegisterSTTProvider registers an STT provider factory with the given name and priority.
// Higher priority values override lower priority registrations.
//
// Example:
//
//	// In omni-deepgram/init.go (thick, priority 10)
//	func init() {
//	    omnivoice.RegisterSTTProvider("deepgram", NewSTTProvider, omnivoice.PriorityThick)
//	}
func RegisterSTTProvider(name string, factory registry.STTProviderFactory, priority int) {
	sttMu.Lock()
	defer sttMu.Unlock()

	existing, ok := sttRegistry[name]
	if !ok || priority >= existing.priority {
		sttRegistry[name] = registeredSTTProvider{
			factory:  factory,
			priority: priority,
		}
	}
}

// GetSTTProvider creates an STT provider instance from the registry.
// Returns an error if the provider is not registered or if creation fails.
func GetSTTProvider(name string, opts ...registry.ProviderOption) (stt.Provider, error) {
	sttMu.RLock()
	rp, ok := sttRegistry[name]
	sttMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("STT provider not registered: %s (available: %v)", name, ListSTTProviders())
	}

	config := registry.ApplyOptions(opts...)
	return rp.factory(config)
}

// ListSTTProviders returns a list of all registered STT provider names.
func ListSTTProviders() []string {
	sttMu.RLock()
	defer sttMu.RUnlock()

	names := make([]string, 0, len(sttRegistry))
	for name := range sttRegistry {
		names = append(names, name)
	}
	return names
}

// HasSTTProvider returns true if an STT provider with the given name is registered.
func HasSTTProvider(name string) bool {
	sttMu.RLock()
	defer sttMu.RUnlock()
	_, ok := sttRegistry[name]
	return ok
}

// GetSTTProviderPriority returns the priority of the registered STT provider.
// Returns -1 if the provider is not registered.
func GetSTTProviderPriority(name string) int {
	sttMu.RLock()
	defer sttMu.RUnlock()

	if rp, ok := sttRegistry[name]; ok {
		return rp.priority
	}
	return -1
}

// RegisterTTSProvider registers a TTS provider factory with the given name and priority.
// Higher priority values override lower priority registrations.
//
// Example:
//
//	// In omni-elevenlabs/init.go (thick, priority 10)
//	func init() {
//	    omnivoice.RegisterTTSProvider("elevenlabs", NewTTSProvider, omnivoice.PriorityThick)
//	}
func RegisterTTSProvider(name string, factory registry.TTSProviderFactory, priority int) {
	ttsMu.Lock()
	defer ttsMu.Unlock()

	existing, ok := ttsRegistry[name]
	if !ok || priority >= existing.priority {
		ttsRegistry[name] = registeredTTSProvider{
			factory:  factory,
			priority: priority,
		}
	}
}

// GetTTSProvider creates a TTS provider instance from the registry.
// Returns an error if the provider is not registered or if creation fails.
func GetTTSProvider(name string, opts ...registry.ProviderOption) (tts.Provider, error) {
	ttsMu.RLock()
	rp, ok := ttsRegistry[name]
	ttsMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("TTS provider not registered: %s (available: %v)", name, ListTTSProviders())
	}

	config := registry.ApplyOptions(opts...)
	return rp.factory(config)
}

// ListTTSProviders returns a list of all registered TTS provider names.
func ListTTSProviders() []string {
	ttsMu.RLock()
	defer ttsMu.RUnlock()

	names := make([]string, 0, len(ttsRegistry))
	for name := range ttsRegistry {
		names = append(names, name)
	}
	return names
}

// HasTTSProvider returns true if a TTS provider with the given name is registered.
func HasTTSProvider(name string) bool {
	ttsMu.RLock()
	defer ttsMu.RUnlock()
	_, ok := ttsRegistry[name]
	return ok
}

// GetTTSProviderPriority returns the priority of the registered TTS provider.
// Returns -1 if the provider is not registered.
func GetTTSProviderPriority(name string) int {
	ttsMu.RLock()
	defer ttsMu.RUnlock()

	if rp, ok := ttsRegistry[name]; ok {
		return rp.priority
	}
	return -1
}

// RegisterCallSystemProvider registers a CallSystem provider factory with the given name and priority.
// Higher priority values override lower priority registrations.
//
// Example:
//
//	// In omni-twilio/init.go (thick, priority 10)
//	func init() {
//	    omnivoice.RegisterCallSystemProvider("twilio", NewCallSystemProvider, omnivoice.PriorityThick)
//	}
func RegisterCallSystemProvider(name string, factory registry.CallSystemProviderFactory, priority int) {
	callSystemMu.Lock()
	defer callSystemMu.Unlock()

	existing, ok := callSystemRegistry[name]
	if !ok || priority >= existing.priority {
		callSystemRegistry[name] = registeredCallSystemProvider{
			factory:  factory,
			priority: priority,
		}
	}
}

// GetCallSystemProvider creates a CallSystem provider instance from the registry.
// Returns an error if the provider is not registered or if creation fails.
func GetCallSystemProvider(name string, opts ...registry.ProviderOption) (callsystem.CallSystem, error) {
	callSystemMu.RLock()
	rp, ok := callSystemRegistry[name]
	callSystemMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("CallSystem provider not registered: %s (available: %v)", name, ListCallSystemProviders())
	}

	config := registry.ApplyOptions(opts...)
	return rp.factory(config)
}

// ListCallSystemProviders returns a list of all registered CallSystem provider names.
func ListCallSystemProviders() []string {
	callSystemMu.RLock()
	defer callSystemMu.RUnlock()

	names := make([]string, 0, len(callSystemRegistry))
	for name := range callSystemRegistry {
		names = append(names, name)
	}
	return names
}

// HasCallSystemProvider returns true if a CallSystem provider with the given name is registered.
func HasCallSystemProvider(name string) bool {
	callSystemMu.RLock()
	defer callSystemMu.RUnlock()
	_, ok := callSystemRegistry[name]
	return ok
}

// GetCallSystemProviderPriority returns the priority of the registered CallSystem provider.
// Returns -1 if the provider is not registered.
func GetCallSystemProviderPriority(name string) int {
	callSystemMu.RLock()
	defer callSystemMu.RUnlock()

	if rp, ok := callSystemRegistry[name]; ok {
		return rp.priority
	}
	return -1
}

// RegisterGatewayProvider registers a Gateway provider factory with the given name and priority.
// Higher priority values override lower priority registrations.
//
// Example:
//
//	// In omni-twilio/omnivoice/gateway/init.go (thick, priority 10)
//	func init() {
//	    omnivoice.RegisterGatewayProvider("twilio", NewGatewayProvider, omnivoice.PriorityThick)
//	}
func RegisterGatewayProvider(name string, factory registry.GatewayProviderFactory, priority int) {
	gatewayMu.Lock()
	defer gatewayMu.Unlock()

	existing, ok := gatewayRegistry[name]
	if !ok || priority >= existing.priority {
		gatewayRegistry[name] = registeredGatewayProvider{
			factory:  factory,
			priority: priority,
		}
	}
}

// GetGatewayProvider creates a Gateway provider instance from the registry.
// Returns an error if the provider is not registered or if creation fails.
func GetGatewayProvider(name string, opts ...registry.ProviderOption) (registry.Gateway, error) {
	gatewayMu.RLock()
	rp, ok := gatewayRegistry[name]
	gatewayMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("Gateway provider not registered: %s (available: %v)", name, ListGatewayProviders())
	}

	config := registry.ApplyOptions(opts...)
	return rp.factory(config)
}

// ListGatewayProviders returns a list of all registered Gateway provider names.
func ListGatewayProviders() []string {
	gatewayMu.RLock()
	defer gatewayMu.RUnlock()

	names := make([]string, 0, len(gatewayRegistry))
	for name := range gatewayRegistry {
		names = append(names, name)
	}
	return names
}

// HasGatewayProvider returns true if a Gateway provider with the given name is registered.
func HasGatewayProvider(name string) bool {
	gatewayMu.RLock()
	defer gatewayMu.RUnlock()
	_, ok := gatewayRegistry[name]
	return ok
}

// GetGatewayProviderPriority returns the priority of the registered Gateway provider.
// Returns -1 if the provider is not registered.
func GetGatewayProviderPriority(name string) int {
	gatewayMu.RLock()
	defer gatewayMu.RUnlock()

	if rp, ok := gatewayRegistry[name]; ok {
		return rp.priority
	}
	return -1
}

// RegisterRealtimeProvider registers a Realtime provider factory with the given name and priority.
// Higher priority values override lower priority registrations.
//
// Example:
//
//	// In omni-openai/omnivoice/realtime/init.go (thick, priority 10)
//	func init() {
//	    omnivoice.RegisterRealtimeProvider("openai", NewRealtimeProvider, omnivoice.PriorityThick)
//	}
func RegisterRealtimeProvider(name string, factory registry.RealtimeProviderFactory, priority int) {
	realtimeMu.Lock()
	defer realtimeMu.Unlock()

	existing, ok := realtimeRegistry[name]
	if !ok || priority >= existing.priority {
		realtimeRegistry[name] = registeredRealtimeProvider{
			factory:  factory,
			priority: priority,
		}
	}
}

// GetRealtimeProvider creates a Realtime provider instance from the registry.
// Returns an error if the provider is not registered or if creation fails.
func GetRealtimeProvider(name string, opts ...registry.ProviderOption) (registry.RealtimeProvider, error) {
	realtimeMu.RLock()
	rp, ok := realtimeRegistry[name]
	realtimeMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("Realtime provider not registered: %s (available: %v)", name, ListRealtimeProviders())
	}

	config := registry.ApplyOptions(opts...)
	return rp.factory(config)
}

// ListRealtimeProviders returns a list of all registered Realtime provider names.
func ListRealtimeProviders() []string {
	realtimeMu.RLock()
	defer realtimeMu.RUnlock()

	names := make([]string, 0, len(realtimeRegistry))
	for name := range realtimeRegistry {
		names = append(names, name)
	}
	return names
}

// HasRealtimeProvider returns true if a Realtime provider with the given name is registered.
func HasRealtimeProvider(name string) bool {
	realtimeMu.RLock()
	defer realtimeMu.RUnlock()
	_, ok := realtimeRegistry[name]
	return ok
}

// GetRealtimeProviderPriority returns the priority of the registered Realtime provider.
// Returns -1 if the provider is not registered.
func GetRealtimeProviderPriority(name string) int {
	realtimeMu.RLock()
	defer realtimeMu.RUnlock()

	if rp, ok := realtimeRegistry[name]; ok {
		return rp.priority
	}
	return -1
}

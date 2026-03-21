package callsystem

import (
	"context"
	"fmt"
	"sync"
)

// Client manages multiple CallSystem providers with fallback support.
type Client struct {
	mu        sync.RWMutex
	providers map[string]CallSystem
	primary   string
	fallbacks []string
}

// NewClient creates a new Client with the given providers.
// The first provider becomes the primary by default.
func NewClient(providers ...CallSystem) *Client {
	c := &Client{
		providers: make(map[string]CallSystem),
	}

	for _, p := range providers {
		c.providers[p.Name()] = p
		if c.primary == "" {
			c.primary = p.Name()
		}
	}

	return c
}

// AddProvider adds a provider to the client.
func (c *Client) AddProvider(p CallSystem) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.providers[p.Name()] = p
	if c.primary == "" {
		c.primary = p.Name()
	}
}

// SetPrimary sets the primary provider by name.
func (c *Client) SetPrimary(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.providers[name]; !ok {
		return fmt.Errorf("provider %q not found", name)
	}
	c.primary = name
	return nil
}

// SetFallbacks sets the fallback providers in order.
func (c *Client) SetFallbacks(names ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, name := range names {
		if _, ok := c.providers[name]; !ok {
			return fmt.Errorf("provider %q not found", name)
		}
	}
	c.fallbacks = names
	return nil
}

// Provider returns a specific provider by name.
func (c *Client) Provider(name string) (CallSystem, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	p, ok := c.providers[name]
	return p, ok
}

// Primary returns the primary provider.
func (c *Client) Primary() (CallSystem, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.primary == "" {
		return nil, false
	}
	p, ok := c.providers[c.primary]
	return p, ok
}

// ListProviders returns the names of all registered providers.
func (c *Client) ListProviders() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	names := make([]string, 0, len(c.providers))
	for name := range c.providers {
		names = append(names, name)
	}
	return names
}

// MakeCall initiates an outbound call using the primary provider.
// If the primary fails and fallbacks are configured, it tries each fallback in order.
func (c *Client) MakeCall(ctx context.Context, to string, opts ...CallOption) (Call, error) {
	c.mu.RLock()
	primary := c.primary
	fallbacks := c.fallbacks
	providers := c.providers
	c.mu.RUnlock()

	if primary == "" {
		return nil, fmt.Errorf("no primary provider configured")
	}

	// Try primary first
	p, ok := providers[primary]
	if !ok {
		return nil, fmt.Errorf("primary provider %q not found", primary)
	}

	call, err := p.MakeCall(ctx, to, opts...)
	if err == nil {
		return call, nil
	}
	primaryErr := err

	// Try fallbacks
	for _, name := range fallbacks {
		p, ok := providers[name]
		if !ok {
			continue
		}

		call, err := p.MakeCall(ctx, to, opts...)
		if err == nil {
			return call, nil
		}
	}

	// All providers failed, return primary error
	return nil, fmt.Errorf("all providers failed; primary error: %w", primaryErr)
}

// GetCall retrieves a call by ID from the specified provider.
// If no provider is specified, searches all providers.
func (c *Client) GetCall(ctx context.Context, callID string, providerName ...string) (Call, error) {
	c.mu.RLock()
	providers := c.providers
	c.mu.RUnlock()

	// If specific provider requested
	if len(providerName) > 0 && providerName[0] != "" {
		p, ok := providers[providerName[0]]
		if !ok {
			return nil, fmt.Errorf("provider %q not found", providerName[0])
		}
		return p.GetCall(ctx, callID)
	}

	// Search all providers
	for _, p := range providers {
		call, err := p.GetCall(ctx, callID)
		if err == nil {
			return call, nil
		}
	}

	return nil, fmt.Errorf("call %q not found in any provider", callID)
}

// Close shuts down all providers.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var lastErr error
	for _, p := range c.providers {
		if err := p.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

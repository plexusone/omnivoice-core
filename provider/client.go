package provider

import "sync"

// Named is the interface that all providers must implement.
type Named interface {
	Name() string
}

// Client provides multi-provider management with fallback support.
// It is designed to be embedded in domain-specific client types.
type Client[T Named] struct {
	providers map[string]T
	primary   string
	fallbacks []string
	mu        sync.RWMutex
}

// NewClient creates a client with the specified providers.
// The first provider is set as primary.
func NewClient[T Named](providers ...T) *Client[T] {
	c := &Client[T]{
		providers: make(map[string]T),
	}
	for i, p := range providers {
		c.providers[p.Name()] = p
		if i == 0 {
			c.primary = p.Name()
		} else {
			c.fallbacks = append(c.fallbacks, p.Name())
		}
	}
	return c
}

// SetPrimary sets the primary provider by name.
func (c *Client[T]) SetPrimary(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.primary = name
}

// SetFallbacks sets the fallback provider order.
func (c *Client[T]) SetFallbacks(names ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fallbacks = names
}

// Provider returns a specific provider by name.
func (c *Client[T]) Provider(name string) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p, ok := c.providers[name]
	return p, ok
}

// Primary returns the primary provider.
func (c *Client[T]) Primary() (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p, ok := c.providers[c.primary]
	return p, ok
}

// Fallbacks returns all fallback providers in order.
func (c *Client[T]) Fallbacks() []T {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]T, 0, len(c.fallbacks))
	for _, name := range c.fallbacks {
		if p, ok := c.providers[name]; ok {
			result = append(result, p)
		}
	}
	return result
}

// PrimaryName returns the name of the primary provider.
func (c *Client[T]) PrimaryName() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.primary
}

// FallbackNames returns the names of fallback providers in order.
func (c *Client[T]) FallbackNames() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]string(nil), c.fallbacks...)
}

// All returns all providers.
func (c *Client[T]) All() map[string]T {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make(map[string]T, len(c.providers))
	for k, v := range c.providers {
		result[k] = v
	}
	return result
}

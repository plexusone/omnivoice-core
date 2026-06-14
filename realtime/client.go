package realtime

import (
	"context"
	"sync"
)

// Client provides a unified interface across multiple real-time providers.
// It supports provider selection and fallback.
type Client struct {
	providers map[string]Provider
	primary   string
	fallbacks []string
	mu        sync.RWMutex
}

// NewClient creates a new real-time client with the specified providers.
// The first provider is set as the primary.
func NewClient(providers ...Provider) *Client {
	c := &Client{
		providers: make(map[string]Provider),
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
func (c *Client) SetPrimary(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.primary = name
}

// SetFallbacks sets the fallback provider order.
func (c *Client) SetFallbacks(names ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fallbacks = names
}

// Provider returns a specific provider by name.
func (c *Client) Provider(name string) (Provider, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p, ok := c.providers[name]
	return p, ok
}

// Primary returns the primary provider.
func (c *Client) Primary() (Provider, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p, ok := c.providers[c.primary]
	return p, ok
}

// ProcessAudioStream uses the primary provider with fallback on connection errors.
func (c *Client) ProcessAudioStream(ctx context.Context, audioIn <-chan []byte, config ProcessConfig) (<-chan AudioChunk, <-chan Transcript, error) {
	c.mu.RLock()
	primary := c.primary
	fallbacks := c.fallbacks
	c.mu.RUnlock()

	var lastErr error

	// Try primary provider
	if p, ok := c.providers[primary]; ok {
		audioCh, transcriptCh, err := p.ProcessAudioStream(ctx, audioIn, config)
		if err == nil {
			return audioCh, transcriptCh, nil
		}
		lastErr = err

		// Only fallback on connection errors, not on config errors
		if shouldFallback(err) {
			// Try fallbacks
			for _, name := range fallbacks {
				if p, ok := c.providers[name]; ok {
					audioCh, transcriptCh, err := p.ProcessAudioStream(ctx, audioIn, config)
					if err == nil {
						return audioCh, transcriptCh, nil
					}
					lastErr = err
					if !shouldFallback(err) {
						break
					}
				}
			}
		}
	}

	if lastErr != nil {
		return nil, nil, lastErr
	}
	return nil, nil, ErrProviderUnavailable
}

// Close closes all providers.
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

// shouldFallback determines if an error should trigger fallback.
// Returns true for connection/availability errors, false for config errors.
func shouldFallback(err error) bool {
	switch err {
	case ErrConnectionFailed, ErrProviderUnavailable, ErrRateLimited:
		return true
	case ErrInvalidConfig, ErrAuthenticationFailed:
		return false
	default:
		// Unknown errors - try fallback
		return true
	}
}

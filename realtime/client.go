package realtime

import (
	"context"

	"github.com/plexusone/omnivoice-core/provider"
)

// Client provides a unified interface across multiple real-time providers.
// It supports provider selection and fallback.
type Client struct {
	*provider.Client[Provider]
}

// NewClient creates a new real-time client with the specified providers.
// The first provider is set as the primary.
func NewClient(providers ...Provider) *Client {
	return &Client{
		Client: provider.NewClient(providers...),
	}
}

// ProcessAudioStream uses the primary provider with fallback on connection errors.
func (c *Client) ProcessAudioStream(ctx context.Context, audioIn <-chan []byte, config ProcessConfig) (<-chan AudioChunk, <-chan Transcript, error) {
	var lastErr error

	// Try primary provider
	if p, ok := c.Primary(); ok {
		audioCh, transcriptCh, err := p.ProcessAudioStream(ctx, audioIn, config)
		if err == nil {
			return audioCh, transcriptCh, nil
		}
		lastErr = err

		// Only fallback on connection errors, not on config errors
		if shouldFallback(err) {
			// Try fallbacks
			for _, p := range c.Fallbacks() {
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

	if lastErr != nil {
		return nil, nil, lastErr
	}
	return nil, nil, ErrProviderUnavailable
}

// Close closes all providers.
func (c *Client) Close() error {
	var lastErr error
	for _, p := range c.All() {
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

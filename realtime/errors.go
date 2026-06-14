package realtime

import "errors"

// Common errors returned by real-time providers.
var (
	// ErrSessionClosed is returned when operating on a closed session.
	ErrSessionClosed = errors.New("session closed")

	// ErrConnectionFailed is returned when the WebSocket connection fails.
	ErrConnectionFailed = errors.New("connection failed")

	// ErrAuthenticationFailed is returned when API authentication fails.
	ErrAuthenticationFailed = errors.New("authentication failed")

	// ErrRateLimited is returned when the provider rate limits the request.
	ErrRateLimited = errors.New("rate limited")

	// ErrInvalidConfig is returned when the configuration is invalid.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrProviderUnavailable is returned when the provider service is unavailable.
	ErrProviderUnavailable = errors.New("provider unavailable")

	// ErrContextCancelled is returned when the context is cancelled.
	ErrContextCancelled = errors.New("context cancelled")
)

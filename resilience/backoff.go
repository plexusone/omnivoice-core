package resilience

import (
	"math"
	"math/rand"
	"time"
)

// BackoffStrategy computes delay between retries.
type BackoffStrategy interface {
	// NextDelay returns the delay before the next retry attempt.
	// attempt is 1-indexed (first retry is attempt 1).
	NextDelay(attempt int) time.Duration

	// Reset resets the backoff state for a new operation.
	Reset()
}

// ExponentialBackoff implements exponential backoff with optional jitter.
//
// The delay for attempt n is: min(Initial * Multiplier^(n-1) * (1 ± Jitter), Max)
//
// Example with Initial=1s, Multiplier=2, Max=30s:
//   - Attempt 1: 1s
//   - Attempt 2: 2s
//   - Attempt 3: 4s
//   - Attempt 4: 8s
//   - Attempt 5: 16s
//   - Attempt 6+: 30s (capped)
type ExponentialBackoff struct {
	// Initial is the delay for the first retry (default: 1s).
	Initial time.Duration

	// Max is the maximum delay (default: 30s).
	Max time.Duration

	// Multiplier is the factor by which delay increases (default: 2.0).
	Multiplier float64

	// Jitter is the random factor applied to delays (0-1, default: 0.1).
	// A jitter of 0.1 means ±10% randomization.
	Jitter float64
}

// NextDelay returns the delay before the next retry attempt.
func (b *ExponentialBackoff) NextDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	initial := b.Initial
	if initial <= 0 {
		initial = time.Second
	}

	maxDelay := b.Max
	if maxDelay <= 0 {
		maxDelay = 30 * time.Second
	}

	multiplier := b.Multiplier
	if multiplier <= 0 {
		multiplier = 2.0
	}

	// Calculate base delay: initial * multiplier^(attempt-1)
	delay := float64(initial) * math.Pow(multiplier, float64(attempt-1))

	// Cap at max
	if delay > float64(maxDelay) {
		delay = float64(maxDelay)
	}

	// Apply jitter (using math/rand is appropriate here - we don't need
	// cryptographic security for timing jitter, just variation to prevent
	// thundering herd)
	if b.Jitter > 0 {
		jitter := b.Jitter
		if jitter > 1 {
			jitter = 1
		}
		// Random factor between (1-jitter) and (1+jitter)
		factor := 1 + (rand.Float64()*2-1)*jitter //nolint:gosec // math/rand is fine for jitter
		delay *= factor
	}

	return time.Duration(delay)
}

// Reset resets the backoff state (no-op for stateless implementation).
func (b *ExponentialBackoff) Reset() {
	// ExponentialBackoff is stateless, nothing to reset
}

// ConstantBackoff always returns the same delay.
type ConstantBackoff struct {
	// Delay is the constant delay between retries.
	Delay time.Duration
}

// NextDelay returns the constant delay.
func (b *ConstantBackoff) NextDelay(attempt int) time.Duration {
	if b.Delay <= 0 {
		return time.Second
	}
	return b.Delay
}

// Reset resets the backoff state (no-op for stateless implementation).
func (b *ConstantBackoff) Reset() {}

// LinearBackoff increases delay linearly with each attempt.
//
// The delay for attempt n is: min(Initial + Increment*(n-1), Max)
type LinearBackoff struct {
	// Initial is the delay for the first retry (default: 1s).
	Initial time.Duration

	// Increment is added for each subsequent attempt (default: 1s).
	Increment time.Duration

	// Max is the maximum delay (default: 30s).
	Max time.Duration
}

// NextDelay returns the delay before the next retry attempt.
func (b *LinearBackoff) NextDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	initial := b.Initial
	if initial <= 0 {
		initial = time.Second
	}

	increment := b.Increment
	if increment <= 0 {
		increment = time.Second
	}

	maxDelay := b.Max
	if maxDelay <= 0 {
		maxDelay = 30 * time.Second
	}

	delay := initial + time.Duration(attempt-1)*increment
	if delay > maxDelay {
		delay = maxDelay
	}

	return delay
}

// Reset resets the backoff state (no-op for stateless implementation).
func (b *LinearBackoff) Reset() {}

// NoBackoff returns zero delay (immediate retry).
// Use with caution; typically only for testing.
type NoBackoff struct{}

// NextDelay returns zero delay.
func (b *NoBackoff) NextDelay(attempt int) time.Duration {
	return 0
}

// Reset resets the backoff state (no-op).
func (b *NoBackoff) Reset() {}

// DefaultBackoff returns a sensible default exponential backoff strategy.
func DefaultBackoff() BackoffStrategy {
	return &ExponentialBackoff{
		Initial:    time.Second,
		Max:        30 * time.Second,
		Multiplier: 2.0,
		Jitter:     0.1,
	}
}

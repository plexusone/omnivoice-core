package resilience

import (
	"context"
	"fmt"
	"time"
)

// RetryConfig controls retry behavior.
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts (including initial).
	// Default: 3 (1 initial + 2 retries).
	MaxAttempts int

	// Backoff is the strategy for computing delays between retries.
	// Default: ExponentialBackoff with sensible defaults.
	Backoff BackoffStrategy

	// Classifier categorizes errors to determine retryability.
	// Default: DefaultClassifier.
	Classifier ErrorClassifier

	// RetryIf is an optional predicate for custom retry decisions.
	// If set, overrides the classifier's retryability decision.
	// Return true to retry, false to stop.
	RetryIf func(err error) bool

	// OnRetry is called before each retry attempt.
	// Can be used for logging or metrics.
	OnRetry func(attempt int, err error, delay time.Duration)
}

// DefaultRetryConfig returns a RetryConfig with sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		Backoff:     DefaultBackoff(),
		Classifier:  &DefaultClassifier{},
	}
}

// Validate ensures the config has valid values, applying defaults where needed.
func (c *RetryConfig) Validate() {
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 3
	}
	if c.Backoff == nil {
		c.Backoff = DefaultBackoff()
	}
	if c.Classifier == nil {
		c.Classifier = &DefaultClassifier{}
	}
}

// shouldRetry determines if an error should be retried.
func (c *RetryConfig) shouldRetry(err error) bool {
	if c.RetryIf != nil {
		return c.RetryIf(err)
	}

	// Check if it's already a ProviderError with retryability info
	if pe, ok := IsProviderError(err); ok {
		return pe.IsRetryable()
	}

	// Use classifier to determine retryability
	info := c.Classifier.Classify(err)
	return info.Retryable
}

// getDelay returns the delay before the next retry.
func (c *RetryConfig) getDelay(attempt int, err error) time.Duration {
	// Check for Retry-After hint in error
	if retryAfter := GetRetryAfter(err); retryAfter > 0 {
		return retryAfter
	}

	// Use backoff strategy
	return c.Backoff.NextDelay(attempt)
}

// RetryError wraps the last error with retry context.
type RetryError struct {
	// Attempts is the number of attempts made.
	Attempts int

	// Err is the last error encountered.
	Err error
}

// Error returns the error message.
func (e *RetryError) Error() string {
	return fmt.Sprintf("retry exhausted after %d attempts: %v", e.Attempts, e.Err)
}

// Unwrap returns the underlying error.
func (e *RetryError) Unwrap() error {
	return e.Err
}

// Retry executes fn with retries according to config.
//
// It returns nil on success, or the last error after all retries are exhausted.
// If the context is canceled, it returns the context error immediately.
//
// Example:
//
//	err := resilience.Retry(ctx, resilience.DefaultRetryConfig(), func() error {
//	    return api.Call()
//	})
func Retry(ctx context.Context, config RetryConfig, fn func() error) error {
	config.Validate()
	config.Backoff.Reset()

	var lastErr error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Check context before attempt
		if err := ctx.Err(); err != nil {
			return err
		}

		// Execute the function
		lastErr = fn()
		if lastErr == nil {
			return nil // Success
		}

		// Check if we should retry
		if attempt >= config.MaxAttempts {
			break // No more attempts
		}

		if !config.shouldRetry(lastErr) {
			return lastErr // Non-retryable error
		}

		// Calculate delay
		delay := config.getDelay(attempt, lastErr)

		// Call OnRetry callback if set
		if config.OnRetry != nil {
			config.OnRetry(attempt, lastErr, delay)
		}

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All retries exhausted
	return &RetryError{
		Attempts: config.MaxAttempts,
		Err:      lastErr,
	}
}

// RetryWithResult executes fn with retries and returns the result on success.
//
// Example:
//
//	result, err := resilience.RetryWithResult(ctx, config, func() (string, error) {
//	    return api.GetValue()
//	})
func RetryWithResult[T any](ctx context.Context, config RetryConfig, fn func() (T, error)) (T, error) {
	config.Validate()
	config.Backoff.Reset()

	var zero T
	var lastErr error
	var result T

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Check context before attempt
		if err := ctx.Err(); err != nil {
			return zero, err
		}

		// Execute the function
		result, lastErr = fn()
		if lastErr == nil {
			return result, nil // Success
		}

		// Check if we should retry
		if attempt >= config.MaxAttempts {
			break // No more attempts
		}

		if !config.shouldRetry(lastErr) {
			return zero, lastErr // Non-retryable error
		}

		// Calculate delay
		delay := config.getDelay(attempt, lastErr)

		// Call OnRetry callback if set
		if config.OnRetry != nil {
			config.OnRetry(attempt, lastErr, delay)
		}

		// Wait before retry
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All retries exhausted
	return zero, &RetryError{
		Attempts: config.MaxAttempts,
		Err:      lastErr,
	}
}

// Do is a convenience wrapper for Retry with default config.
func Do(ctx context.Context, fn func() error) error {
	return Retry(ctx, DefaultRetryConfig(), fn)
}

// DoWithResult is a convenience wrapper for RetryWithResult with default config.
func DoWithResult[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	return RetryWithResult(ctx, DefaultRetryConfig(), fn)
}

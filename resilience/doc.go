// Package resilience provides error handling, retry logic, and backoff strategies
// for building resilient voice applications.
//
// The package is designed to work with the AX (Agent Experience) specification,
// enabling intelligent error recovery based on error categorization.
//
// # Error Categories
//
// Errors are classified into categories that determine handling strategy:
//
//   - CategoryTransient: Temporary failures, retry with backoff
//   - CategoryRateLimit: Rate limited, retry with longer backoff
//   - CategoryValidation: Invalid input, don't retry
//   - CategoryAuth: Authentication failure, don't retry
//   - CategoryNotFound: Resource not found, don't retry
//   - CategoryServer: Server error, retry with backoff
//   - CategoryQuota: Quota exceeded, don't retry
//   - CategoryUnknown: Unknown error, use default behavior
//
// # Retry Logic
//
// The Retry and RetryWithResult functions execute operations with automatic
// retries for retryable errors:
//
//	result, err := resilience.RetryWithResult(ctx, resilience.DefaultRetryConfig(), func() (string, error) {
//	    return api.Call()
//	})
//
// # Backoff Strategies
//
// ExponentialBackoff provides exponential backoff with jitter:
//
//	backoff := &resilience.ExponentialBackoff{
//	    Initial:    time.Second,
//	    Max:        30 * time.Second,
//	    Multiplier: 2.0,
//	    Jitter:     0.1,
//	}
//
// # Error Wrapping
//
// ProviderError wraps provider errors with AX metadata:
//
//	err := &resilience.ProviderError{
//	    Provider: "elevenlabs",
//	    Op:       "Synthesize",
//	    Err:      originalErr,
//	    Info: resilience.ErrorInfo{
//	        Category:   resilience.CategoryRateLimit,
//	        Retryable:  true,
//	        Code:       "RATE_LIMITED",
//	        Suggestion: "Wait and retry with exponential backoff",
//	    },
//	}
package resilience

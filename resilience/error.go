package resilience

import (
	"errors"
	"fmt"
	"time"
)

// ErrorInfo provides actionable metadata about an error.
type ErrorInfo struct {
	// Category classifies the error for handling decisions.
	Category ErrorCategory

	// Retryable indicates if the operation can be safely retried.
	Retryable bool

	// Code is a machine-readable error code (e.g., "RATE_LIMITED").
	Code string

	// Message is a human-readable error description.
	Message string

	// Suggestion provides guidance for error recovery.
	Suggestion string

	// RetryAfter hints how long to wait before retrying.
	// Zero means use default backoff strategy.
	RetryAfter time.Duration
}

// ProviderError wraps provider errors with AX metadata.
// It implements the error interface and supports error unwrapping.
type ProviderError struct {
	// Provider is the name of the provider (e.g., "elevenlabs").
	Provider string

	// Op is the operation that failed (e.g., "Synthesize").
	Op string

	// Err is the underlying error.
	Err error

	// Info contains AX metadata about the error.
	Info ErrorInfo
}

// Error returns the error message.
func (e *ProviderError) Error() string {
	if e.Info.Code != "" {
		return fmt.Sprintf("%s %s: [%s] %s", e.Provider, e.Op, e.Info.Code, e.Info.Message)
	}
	if e.Info.Message != "" {
		return fmt.Sprintf("%s %s: %s", e.Provider, e.Op, e.Info.Message)
	}
	if e.Err != nil {
		return fmt.Sprintf("%s %s: %v", e.Provider, e.Op, e.Err)
	}
	return fmt.Sprintf("%s %s failed", e.Provider, e.Op)
}

// Unwrap returns the underlying error.
func (e *ProviderError) Unwrap() error {
	return e.Err
}

// Is reports whether any error in the chain matches target.
func (e *ProviderError) Is(target error) bool {
	if target == nil {
		return false
	}
	// Check if target is a ProviderError with matching provider and code
	var pe *ProviderError
	if errors.As(target, &pe) {
		if pe.Provider != "" && pe.Provider != e.Provider {
			return false
		}
		if pe.Info.Code != "" && pe.Info.Code != e.Info.Code {
			return false
		}
		return true
	}
	return false
}

// IsRetryable returns true if the error can be retried.
func (e *ProviderError) IsRetryable() bool {
	return e.Info.Retryable
}

// GetCategory returns the error category.
func (e *ProviderError) GetCategory() ErrorCategory {
	return e.Info.Category
}

// GetCode returns the error code.
func (e *ProviderError) GetCode() string {
	return e.Info.Code
}

// GetSuggestion returns the recovery suggestion.
func (e *ProviderError) GetSuggestion() string {
	if e.Info.Suggestion != "" {
		return e.Info.Suggestion
	}
	return e.Info.Category.SuggestedAction()
}

// GetRetryAfter returns the suggested retry delay.
func (e *ProviderError) GetRetryAfter() time.Duration {
	return e.Info.RetryAfter
}

// NewProviderError creates a new ProviderError.
func NewProviderError(provider, op string, err error, info ErrorInfo) *ProviderError {
	return &ProviderError{
		Provider: provider,
		Op:       op,
		Err:      err,
		Info:     info,
	}
}

// IsProviderError checks if err is a ProviderError and returns it.
func IsProviderError(err error) (*ProviderError, bool) {
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe, true
	}
	return nil, false
}

// GetErrorInfo extracts ErrorInfo from an error.
// If the error is not a ProviderError, returns an unknown category.
func GetErrorInfo(err error) ErrorInfo {
	if pe, ok := IsProviderError(err); ok {
		return pe.Info
	}
	return ErrorInfo{
		Category:  CategoryUnknown,
		Retryable: false,
		Message:   err.Error(),
	}
}

// IsRetryable checks if an error should be retried.
func IsRetryable(err error) bool {
	if pe, ok := IsProviderError(err); ok {
		return pe.IsRetryable()
	}
	return false
}

// GetRetryAfter extracts retry-after duration from an error.
// Returns zero if not a ProviderError or no retry-after specified.
func GetRetryAfter(err error) time.Duration {
	if pe, ok := IsProviderError(err); ok {
		return pe.GetRetryAfter()
	}
	return 0
}

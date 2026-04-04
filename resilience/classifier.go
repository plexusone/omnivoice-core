package resilience

import (
	"errors"
	"net"
	"strings"
	"syscall"
)

// ErrorClassifier categorizes errors from any source.
// Implementations should be provider-specific.
type ErrorClassifier interface {
	// Classify analyzes an error and returns its metadata.
	Classify(err error) ErrorInfo
}

// DefaultClassifier provides basic error classification based on common patterns.
// Use this as a fallback when no provider-specific classifier is available.
type DefaultClassifier struct{}

// Classify categorizes an error using common heuristics.
func (c *DefaultClassifier) Classify(err error) ErrorInfo {
	if err == nil {
		return ErrorInfo{Category: CategoryUnknown}
	}

	// Check for wrapped ProviderError first
	if pe, ok := IsProviderError(err); ok {
		return pe.Info
	}

	// Check for network errors (transient)
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return ErrorInfo{
				Category:   CategoryTransient,
				Retryable:  true,
				Code:       "TIMEOUT",
				Message:    "Request timed out",
				Suggestion: "Retry with exponential backoff",
			}
		}
		return ErrorInfo{
			Category:   CategoryTransient,
			Retryable:  true,
			Code:       "NETWORK_ERROR",
			Message:    "Network error occurred",
			Suggestion: "Check network connectivity and retry",
		}
	}

	// Check for connection refused (server unavailable)
	if errors.Is(err, syscall.ECONNREFUSED) {
		return ErrorInfo{
			Category:   CategoryServer,
			Retryable:  true,
			Code:       "CONNECTION_REFUSED",
			Message:    "Connection refused",
			Suggestion: "Server may be down, retry later",
		}
	}

	// Check for common error patterns in message
	msg := strings.ToLower(err.Error())

	if strings.Contains(msg, "rate limit") || strings.Contains(msg, "too many requests") {
		return ErrorInfo{
			Category:   CategoryRateLimit,
			Retryable:  true,
			Code:       "RATE_LIMITED",
			Message:    "Rate limit exceeded",
			Suggestion: "Wait and retry with exponential backoff",
		}
	}

	if strings.Contains(msg, "unauthorized") || strings.Contains(msg, "authentication") {
		return ErrorInfo{
			Category:   CategoryAuth,
			Retryable:  false,
			Code:       "UNAUTHORIZED",
			Message:    "Authentication failed",
			Suggestion: "Check API credentials",
		}
	}

	if strings.Contains(msg, "forbidden") || strings.Contains(msg, "permission") {
		return ErrorInfo{
			Category:   CategoryAuth,
			Retryable:  false,
			Code:       "FORBIDDEN",
			Message:    "Permission denied",
			Suggestion: "Check permissions for this operation",
		}
	}

	if strings.Contains(msg, "not found") {
		return ErrorInfo{
			Category:   CategoryNotFound,
			Retryable:  false,
			Code:       "NOT_FOUND",
			Message:    "Resource not found",
			Suggestion: "Verify the resource exists",
		}
	}

	if strings.Contains(msg, "invalid") || strings.Contains(msg, "validation") {
		return ErrorInfo{
			Category:   CategoryValidation,
			Retryable:  false,
			Code:       "VALIDATION_ERROR",
			Message:    "Invalid request",
			Suggestion: "Check request parameters",
		}
	}

	if strings.Contains(msg, "quota") || strings.Contains(msg, "limit exceeded") {
		return ErrorInfo{
			Category:   CategoryQuota,
			Retryable:  false,
			Code:       "QUOTA_EXCEEDED",
			Message:    "Quota exceeded",
			Suggestion: "Wait for quota reset or increase limit",
		}
	}

	if strings.Contains(msg, "internal") || strings.Contains(msg, "server error") {
		return ErrorInfo{
			Category:   CategoryServer,
			Retryable:  true,
			Code:       "SERVER_ERROR",
			Message:    "Internal server error",
			Suggestion: "Retry with exponential backoff",
		}
	}

	// Unknown error
	return ErrorInfo{
		Category:   CategoryUnknown,
		Retryable:  false,
		Code:       "UNKNOWN",
		Message:    err.Error(),
		Suggestion: "Check the error details",
	}
}

// HTTPStatusClassifier classifies errors based on HTTP status codes.
type HTTPStatusClassifier struct{}

// ClassifyStatus returns ErrorInfo based on HTTP status code.
func (c *HTTPStatusClassifier) ClassifyStatus(status int, message string) ErrorInfo {
	switch {
	case status == 400:
		return ErrorInfo{
			Category:   CategoryValidation,
			Retryable:  false,
			Code:       "BAD_REQUEST",
			Message:    message,
			Suggestion: "Check request parameters",
		}
	case status == 401:
		return ErrorInfo{
			Category:   CategoryAuth,
			Retryable:  false,
			Code:       "UNAUTHORIZED",
			Message:    message,
			Suggestion: "Check API credentials",
		}
	case status == 403:
		return ErrorInfo{
			Category:   CategoryAuth,
			Retryable:  false,
			Code:       "FORBIDDEN",
			Message:    message,
			Suggestion: "Check permissions",
		}
	case status == 404:
		return ErrorInfo{
			Category:   CategoryNotFound,
			Retryable:  false,
			Code:       "NOT_FOUND",
			Message:    message,
			Suggestion: "Verify the resource exists",
		}
	case status == 409:
		return ErrorInfo{
			Category:   CategoryValidation,
			Retryable:  false,
			Code:       "CONFLICT",
			Message:    message,
			Suggestion: "Resolve the conflict and retry",
		}
	case status == 422:
		return ErrorInfo{
			Category:   CategoryValidation,
			Retryable:  false,
			Code:       "UNPROCESSABLE_ENTITY",
			Message:    message,
			Suggestion: "Check request body format",
		}
	case status == 429:
		return ErrorInfo{
			Category:   CategoryRateLimit,
			Retryable:  true,
			Code:       "RATE_LIMITED",
			Message:    message,
			Suggestion: "Wait and retry with exponential backoff",
		}
	case status >= 500 && status < 600:
		return ErrorInfo{
			Category:   CategoryServer,
			Retryable:  true,
			Code:       "SERVER_ERROR",
			Message:    message,
			Suggestion: "Retry with exponential backoff",
		}
	default:
		return ErrorInfo{
			Category:   CategoryUnknown,
			Retryable:  false,
			Code:       "UNKNOWN",
			Message:    message,
			Suggestion: "Check the error details",
		}
	}
}

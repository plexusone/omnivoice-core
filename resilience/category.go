package resilience

// ErrorCategory classifies errors for handling decisions.
type ErrorCategory string

const (
	// CategoryTransient indicates a temporary failure that may succeed on retry.
	// Examples: network timeout, connection reset.
	CategoryTransient ErrorCategory = "transient"

	// CategoryRateLimit indicates the request was rate limited.
	// Retry with longer backoff, respect Retry-After header if present.
	CategoryRateLimit ErrorCategory = "rate_limit"

	// CategoryValidation indicates invalid input that will never succeed.
	// Do not retry; fix the request first.
	CategoryValidation ErrorCategory = "validation"

	// CategoryAuth indicates an authentication or authorization failure.
	// Do not retry; re-authenticate or check permissions.
	CategoryAuth ErrorCategory = "auth"

	// CategoryNotFound indicates the requested resource does not exist.
	// Do not retry; the resource must be created first.
	CategoryNotFound ErrorCategory = "not_found"

	// CategoryServer indicates an internal server error.
	// May succeed on retry; use exponential backoff.
	CategoryServer ErrorCategory = "server"

	// CategoryQuota indicates a quota or limit has been exceeded.
	// Do not retry; quota must be increased or reset.
	CategoryQuota ErrorCategory = "quota"

	// CategoryUnknown indicates the error could not be classified.
	// Use default behavior (typically no retry).
	CategoryUnknown ErrorCategory = "unknown"
)

// String returns the string representation of the category.
func (c ErrorCategory) String() string {
	return string(c)
}

// IsRetryable returns true if errors in this category should typically be retried.
func (c ErrorCategory) IsRetryable() bool {
	switch c {
	case CategoryTransient, CategoryRateLimit, CategoryServer:
		return true
	default:
		return false
	}
}

// SuggestedAction returns a human-readable suggestion for handling this category.
func (c ErrorCategory) SuggestedAction() string {
	switch c {
	case CategoryTransient:
		return "Retry with exponential backoff"
	case CategoryRateLimit:
		return "Wait and retry with longer backoff"
	case CategoryValidation:
		return "Fix the request and try again"
	case CategoryAuth:
		return "Check credentials or permissions"
	case CategoryNotFound:
		return "Verify the resource exists"
	case CategoryServer:
		return "Retry with exponential backoff"
	case CategoryQuota:
		return "Wait for quota reset or increase limit"
	default:
		return "Check the error details"
	}
}

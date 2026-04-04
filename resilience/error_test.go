package resilience

import (
	"errors"
	"testing"
	"time"
)

func TestProviderError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *ProviderError
		want string
	}{
		{
			name: "with code and message",
			err: &ProviderError{
				Provider: "elevenlabs",
				Op:       "Synthesize",
				Info: ErrorInfo{
					Code:    "RATE_LIMITED",
					Message: "Too many requests",
				},
			},
			want: "elevenlabs Synthesize: [RATE_LIMITED] Too many requests",
		},
		{
			name: "with message only",
			err: &ProviderError{
				Provider: "elevenlabs",
				Op:       "Synthesize",
				Info: ErrorInfo{
					Message: "Something went wrong",
				},
			},
			want: "elevenlabs Synthesize: Something went wrong",
		},
		{
			name: "with underlying error",
			err: &ProviderError{
				Provider: "elevenlabs",
				Op:       "Synthesize",
				Err:      errors.New("connection refused"),
			},
			want: "elevenlabs Synthesize: connection refused",
		},
		{
			name: "minimal",
			err: &ProviderError{
				Provider: "elevenlabs",
				Op:       "Synthesize",
			},
			want: "elevenlabs Synthesize failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("ProviderError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProviderError_Unwrap(t *testing.T) {
	underlying := errors.New("original error")
	pe := &ProviderError{
		Provider: "test",
		Op:       "test",
		Err:      underlying,
	}

	if got := pe.Unwrap(); got != underlying {
		t.Errorf("ProviderError.Unwrap() = %v, want %v", got, underlying)
	}
}

func TestProviderError_Is(t *testing.T) {
	pe1 := &ProviderError{
		Provider: "elevenlabs",
		Op:       "Synthesize",
		Info:     ErrorInfo{Code: "RATE_LIMITED"},
	}

	tests := []struct {
		name   string
		target error
		want   bool
	}{
		{
			name:   "matching provider and code",
			target: &ProviderError{Provider: "elevenlabs", Info: ErrorInfo{Code: "RATE_LIMITED"}},
			want:   true,
		},
		{
			name:   "matching code only",
			target: &ProviderError{Info: ErrorInfo{Code: "RATE_LIMITED"}},
			want:   true,
		},
		{
			name:   "matching provider only",
			target: &ProviderError{Provider: "elevenlabs"},
			want:   true,
		},
		{
			name:   "different provider",
			target: &ProviderError{Provider: "deepgram", Info: ErrorInfo{Code: "RATE_LIMITED"}},
			want:   false,
		},
		{
			name:   "different code",
			target: &ProviderError{Provider: "elevenlabs", Info: ErrorInfo{Code: "AUTH_ERROR"}},
			want:   false,
		},
		{
			name:   "nil target",
			target: nil,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pe1.Is(tt.target); got != tt.want {
				t.Errorf("ProviderError.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProviderError_Accessors(t *testing.T) {
	pe := &ProviderError{
		Provider: "elevenlabs",
		Op:       "Synthesize",
		Info: ErrorInfo{
			Category:   CategoryRateLimit,
			Retryable:  true,
			Code:       "RATE_LIMITED",
			Suggestion: "Wait and retry",
			RetryAfter: 5 * time.Second,
		},
	}

	if got := pe.IsRetryable(); !got {
		t.Error("IsRetryable() should return true")
	}

	if got := pe.GetCategory(); got != CategoryRateLimit {
		t.Errorf("GetCategory() = %v, want %v", got, CategoryRateLimit)
	}

	if got := pe.GetCode(); got != "RATE_LIMITED" {
		t.Errorf("GetCode() = %q, want %q", got, "RATE_LIMITED")
	}

	if got := pe.GetSuggestion(); got != "Wait and retry" {
		t.Errorf("GetSuggestion() = %q, want %q", got, "Wait and retry")
	}

	if got := pe.GetRetryAfter(); got != 5*time.Second {
		t.Errorf("GetRetryAfter() = %v, want %v", got, 5*time.Second)
	}
}

func TestProviderError_GetSuggestion_Fallback(t *testing.T) {
	pe := &ProviderError{
		Provider: "test",
		Op:       "test",
		Info: ErrorInfo{
			Category: CategoryRateLimit,
			// No explicit suggestion
		},
	}

	got := pe.GetSuggestion()
	if got == "" {
		t.Error("GetSuggestion() should return category suggestion as fallback")
	}
}

func TestNewProviderError(t *testing.T) {
	underlying := errors.New("test error")
	info := ErrorInfo{
		Category:  CategoryValidation,
		Retryable: false,
		Code:      "INVALID_INPUT",
	}

	pe := NewProviderError("elevenlabs", "Synthesize", underlying, info)

	if pe.Provider != "elevenlabs" {
		t.Errorf("Provider = %q, want %q", pe.Provider, "elevenlabs")
	}
	if pe.Op != "Synthesize" {
		t.Errorf("Op = %q, want %q", pe.Op, "Synthesize")
	}
	if pe.Err != underlying {
		t.Error("Err should be the underlying error")
	}
	if pe.Info.Code != "INVALID_INPUT" {
		t.Errorf("Info.Code = %q, want %q", pe.Info.Code, "INVALID_INPUT")
	}
}

func TestIsProviderError(t *testing.T) {
	pe := &ProviderError{Provider: "test", Op: "test"}
	wrapped := errors.New("wrapped: " + pe.Error())

	// Direct ProviderError
	if got, ok := IsProviderError(pe); !ok || got != pe {
		t.Error("IsProviderError should return true for ProviderError")
	}

	// Non-ProviderError
	if _, ok := IsProviderError(errors.New("regular error")); ok {
		t.Error("IsProviderError should return false for regular error")
	}

	// Nil
	if _, ok := IsProviderError(nil); ok {
		t.Error("IsProviderError should return false for nil")
	}

	// Note: wrapped errors would need proper wrapping to work
	_ = wrapped
}

func TestGetErrorInfo(t *testing.T) {
	pe := &ProviderError{
		Provider: "test",
		Op:       "test",
		Info: ErrorInfo{
			Category:  CategoryRateLimit,
			Retryable: true,
			Code:      "RATE_LIMITED",
		},
	}

	// ProviderError
	info := GetErrorInfo(pe)
	if info.Category != CategoryRateLimit {
		t.Errorf("GetErrorInfo().Category = %v, want %v", info.Category, CategoryRateLimit)
	}

	// Regular error
	info = GetErrorInfo(errors.New("regular error"))
	if info.Category != CategoryUnknown {
		t.Errorf("GetErrorInfo().Category = %v, want %v", info.Category, CategoryUnknown)
	}
}

func TestIsRetryable(t *testing.T) {
	retryable := &ProviderError{
		Info: ErrorInfo{Retryable: true},
	}
	notRetryable := &ProviderError{
		Info: ErrorInfo{Retryable: false},
	}
	regular := errors.New("regular error")

	if !IsRetryable(retryable) {
		t.Error("IsRetryable should return true for retryable ProviderError")
	}
	if IsRetryable(notRetryable) {
		t.Error("IsRetryable should return false for non-retryable ProviderError")
	}
	if IsRetryable(regular) {
		t.Error("IsRetryable should return false for regular error")
	}
}

func TestGetRetryAfter(t *testing.T) {
	withRetryAfter := &ProviderError{
		Info: ErrorInfo{RetryAfter: 5 * time.Second},
	}
	withoutRetryAfter := &ProviderError{
		Info: ErrorInfo{},
	}
	regular := errors.New("regular error")

	if got := GetRetryAfter(withRetryAfter); got != 5*time.Second {
		t.Errorf("GetRetryAfter() = %v, want %v", got, 5*time.Second)
	}
	if got := GetRetryAfter(withoutRetryAfter); got != 0 {
		t.Errorf("GetRetryAfter() = %v, want 0", got)
	}
	if got := GetRetryAfter(regular); got != 0 {
		t.Errorf("GetRetryAfter() = %v, want 0", got)
	}
}

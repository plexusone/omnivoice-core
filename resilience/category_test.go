package resilience

import "testing"

func TestErrorCategory_String(t *testing.T) {
	tests := []struct {
		category ErrorCategory
		want     string
	}{
		{CategoryTransient, "transient"},
		{CategoryRateLimit, "rate_limit"},
		{CategoryValidation, "validation"},
		{CategoryAuth, "auth"},
		{CategoryNotFound, "not_found"},
		{CategoryServer, "server"},
		{CategoryQuota, "quota"},
		{CategoryUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.category.String(); got != tt.want {
				t.Errorf("ErrorCategory.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorCategory_IsRetryable(t *testing.T) {
	tests := []struct {
		category  ErrorCategory
		retryable bool
	}{
		{CategoryTransient, true},
		{CategoryRateLimit, true},
		{CategoryServer, true},
		{CategoryValidation, false},
		{CategoryAuth, false},
		{CategoryNotFound, false},
		{CategoryQuota, false},
		{CategoryUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.category.String(), func(t *testing.T) {
			if got := tt.category.IsRetryable(); got != tt.retryable {
				t.Errorf("ErrorCategory.IsRetryable() = %v, want %v", got, tt.retryable)
			}
		})
	}
}

func TestErrorCategory_SuggestedAction(t *testing.T) {
	// Ensure all categories have suggestions
	categories := []ErrorCategory{
		CategoryTransient,
		CategoryRateLimit,
		CategoryValidation,
		CategoryAuth,
		CategoryNotFound,
		CategoryServer,
		CategoryQuota,
		CategoryUnknown,
	}

	for _, cat := range categories {
		t.Run(cat.String(), func(t *testing.T) {
			suggestion := cat.SuggestedAction()
			if suggestion == "" {
				t.Errorf("ErrorCategory.SuggestedAction() returned empty for %s", cat)
			}
		})
	}
}

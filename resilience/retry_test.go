package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry_Success(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxAttempts: 3,
		Backoff:     &NoBackoff{},
	}

	calls := 0
	err := Retry(ctx, config, func() error {
		calls++
		return nil
	})

	if err != nil {
		t.Errorf("Retry() error = %v, want nil", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestRetry_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxAttempts: 3,
		Backoff:     &NoBackoff{},
		Classifier:  &DefaultClassifier{},
	}

	calls := 0
	err := Retry(ctx, config, func() error {
		calls++
		if calls < 3 {
			return &ProviderError{
				Info: ErrorInfo{Retryable: true},
			}
		}
		return nil
	})

	if err != nil {
		t.Errorf("Retry() error = %v, want nil", err)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestRetry_ExhaustedRetries(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxAttempts: 3,
		Backoff:     &NoBackoff{},
	}

	testErr := &ProviderError{
		Provider: "test",
		Op:       "test",
		Info:     ErrorInfo{Retryable: true, Code: "TEST_ERROR"},
	}

	calls := 0
	err := Retry(ctx, config, func() error {
		calls++
		return testErr
	})

	if err == nil {
		t.Error("Retry() error = nil, want error")
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}

	// Should be wrapped in RetryError
	var retryErr *RetryError
	if !errors.As(err, &retryErr) {
		t.Error("error should be *RetryError")
	} else {
		if retryErr.Attempts != 3 {
			t.Errorf("RetryError.Attempts = %d, want 3", retryErr.Attempts)
		}
	}
}

func TestRetry_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxAttempts: 3,
		Backoff:     &NoBackoff{},
	}

	testErr := &ProviderError{
		Provider: "test",
		Op:       "test",
		Info:     ErrorInfo{Retryable: false, Code: "VALIDATION_ERROR"},
	}

	calls := 0
	err := Retry(ctx, config, func() error {
		calls++
		return testErr
	})

	if err != testErr {
		t.Errorf("Retry() error = %v, want %v", err, testErr)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (no retry for non-retryable)", calls)
	}
}

func TestRetry_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := RetryConfig{
		MaxAttempts: 10,
		Backoff:     &ConstantBackoff{Delay: time.Hour}, // Long delay
	}

	calls := 0
	errCh := make(chan error, 1)

	go func() {
		errCh <- Retry(ctx, config, func() error {
			calls++
			return &ProviderError{Info: ErrorInfo{Retryable: true}}
		})
	}()

	// Cancel after first attempt
	time.Sleep(10 * time.Millisecond)
	cancel()

	err := <-errCh
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Retry() error = %v, want context.Canceled", err)
	}
}

func TestRetry_CustomRetryIf(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxAttempts: 3,
		Backoff:     &NoBackoff{},
		RetryIf: func(err error) bool {
			// Only retry errors containing "retry"
			return err != nil && err.Error() == "retry me"
		},
	}

	// Should retry
	calls := 0
	_ = Retry(ctx, config, func() error {
		calls++
		if calls < 3 {
			return errors.New("retry me")
		}
		return nil
	})
	if calls != 3 {
		t.Errorf("with RetryIf match: calls = %d, want 3", calls)
	}

	// Should not retry
	calls = 0
	_ = Retry(ctx, config, func() error {
		calls++
		return errors.New("don't retry")
	})
	if calls != 1 {
		t.Errorf("without RetryIf match: calls = %d, want 1", calls)
	}
}

func TestRetry_OnRetryCallback(t *testing.T) {
	ctx := context.Background()

	var callbacks []int
	config := RetryConfig{
		MaxAttempts: 3,
		Backoff:     &NoBackoff{},
		OnRetry: func(attempt int, err error, delay time.Duration) {
			callbacks = append(callbacks, attempt)
		},
	}

	_ = Retry(ctx, config, func() error {
		return &ProviderError{Info: ErrorInfo{Retryable: true}}
	})

	// OnRetry should be called before attempts 2 and 3
	if len(callbacks) != 2 {
		t.Errorf("OnRetry called %d times, want 2", len(callbacks))
	}
	if len(callbacks) >= 2 && (callbacks[0] != 1 || callbacks[1] != 2) {
		t.Errorf("OnRetry attempts = %v, want [1, 2]", callbacks)
	}
}

func TestRetry_RespectsRetryAfter(t *testing.T) {
	ctx := context.Background()

	start := time.Now()
	config := RetryConfig{
		MaxAttempts: 2,
		Backoff:     &ConstantBackoff{Delay: time.Hour}, // Would be very long
	}

	retryAfterDelay := 50 * time.Millisecond

	calls := 0
	_ = Retry(ctx, config, func() error {
		calls++
		if calls < 2 {
			return &ProviderError{
				Info: ErrorInfo{
					Retryable:  true,
					RetryAfter: retryAfterDelay,
				},
			}
		}
		return nil
	})

	elapsed := time.Since(start)

	// Should have used RetryAfter, not the backoff delay
	if elapsed > 200*time.Millisecond {
		t.Errorf("Retry took %v, should have used RetryAfter (%v)", elapsed, retryAfterDelay)
	}
}

func TestRetryWithResult_Success(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxAttempts: 3,
		Backoff:     &NoBackoff{},
	}

	result, err := RetryWithResult(ctx, config, func() (string, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("RetryWithResult() error = %v, want nil", err)
	}
	if result != "success" {
		t.Errorf("result = %q, want %q", result, "success")
	}
}

func TestRetryWithResult_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxAttempts: 3,
		Backoff:     &NoBackoff{},
	}

	calls := 0
	result, err := RetryWithResult(ctx, config, func() (int, error) {
		calls++
		if calls < 3 {
			return 0, &ProviderError{Info: ErrorInfo{Retryable: true}}
		}
		return 42, nil
	})

	if err != nil {
		t.Errorf("RetryWithResult() error = %v, want nil", err)
	}
	if result != 42 {
		t.Errorf("result = %d, want 42", result)
	}
}

func TestRetryWithResult_ExhaustedRetries(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxAttempts: 3,
		Backoff:     &NoBackoff{},
	}

	result, err := RetryWithResult(ctx, config, func() (string, error) {
		return "partial", &ProviderError{Info: ErrorInfo{Retryable: true}}
	})

	if err == nil {
		t.Error("RetryWithResult() error = nil, want error")
	}
	if result != "" {
		t.Errorf("result = %q, want empty string", result)
	}
}

func TestDo(t *testing.T) {
	ctx := context.Background()

	err := Do(ctx, func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Do() error = %v, want nil", err)
	}
}

func TestDoWithResult(t *testing.T) {
	ctx := context.Background()

	result, err := DoWithResult(ctx, func() (string, error) {
		return "hello", nil
	})

	if err != nil {
		t.Errorf("DoWithResult() error = %v, want nil", err)
	}
	if result != "hello" {
		t.Errorf("result = %q, want %q", result, "hello")
	}
}

func TestRetryConfig_Validate(t *testing.T) {
	config := RetryConfig{} // All zero values
	config.Validate()

	if config.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", config.MaxAttempts)
	}
	if config.Backoff == nil {
		t.Error("Backoff should not be nil after Validate")
	}
	if config.Classifier == nil {
		t.Error("Classifier should not be nil after Validate")
	}
}

func TestRetryError(t *testing.T) {
	underlying := errors.New("test error")
	retryErr := &RetryError{
		Attempts: 3,
		Err:      underlying,
	}

	// Test Error()
	errMsg := retryErr.Error()
	if errMsg == "" {
		t.Error("RetryError.Error() should not be empty")
	}

	// Test Unwrap()
	if retryErr.Unwrap() != underlying {
		t.Error("RetryError.Unwrap() should return underlying error")
	}

	// Test errors.Is
	if !errors.Is(retryErr, underlying) {
		t.Error("errors.Is should find underlying error")
	}
}

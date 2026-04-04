package resilience

import (
	"testing"
	"time"
)

func TestExponentialBackoff_NextDelay(t *testing.T) {
	backoff := &ExponentialBackoff{
		Initial:    time.Second,
		Max:        30 * time.Second,
		Multiplier: 2.0,
		Jitter:     0, // No jitter for predictable testing
	}

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 8 * time.Second},
		{5, 16 * time.Second},
		{6, 30 * time.Second}, // Capped at max
		{7, 30 * time.Second}, // Still capped
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := backoff.NextDelay(tt.attempt)
			if got != tt.want {
				t.Errorf("NextDelay(%d) = %v, want %v", tt.attempt, got, tt.want)
			}
		})
	}
}

func TestExponentialBackoff_WithJitter(t *testing.T) {
	backoff := &ExponentialBackoff{
		Initial:    time.Second,
		Max:        30 * time.Second,
		Multiplier: 2.0,
		Jitter:     0.1, // 10% jitter
	}

	// Run multiple times and check range
	for i := 0; i < 100; i++ {
		delay := backoff.NextDelay(1)
		minExpected := time.Duration(float64(time.Second) * 0.9)
		maxExpected := time.Duration(float64(time.Second) * 1.1)

		if delay < minExpected || delay > maxExpected {
			t.Errorf("NextDelay(1) = %v, want between %v and %v", delay, minExpected, maxExpected)
		}
	}
}

func TestExponentialBackoff_Defaults(t *testing.T) {
	backoff := &ExponentialBackoff{} // All zero values

	delay := backoff.NextDelay(1)
	if delay != time.Second {
		t.Errorf("Default NextDelay(1) = %v, want %v", delay, time.Second)
	}

	delay = backoff.NextDelay(10)
	if delay > 30*time.Second {
		t.Errorf("Default NextDelay(10) = %v, should be capped at 30s", delay)
	}
}

func TestExponentialBackoff_InvalidAttempt(t *testing.T) {
	backoff := &ExponentialBackoff{
		Initial:    time.Second,
		Multiplier: 2.0,
	}

	// Attempt 0 or negative should be treated as 1
	if got := backoff.NextDelay(0); got != time.Second {
		t.Errorf("NextDelay(0) = %v, want %v", got, time.Second)
	}
	if got := backoff.NextDelay(-1); got != time.Second {
		t.Errorf("NextDelay(-1) = %v, want %v", got, time.Second)
	}
}

func TestConstantBackoff_NextDelay(t *testing.T) {
	backoff := &ConstantBackoff{Delay: 5 * time.Second}

	for attempt := 1; attempt <= 5; attempt++ {
		if got := backoff.NextDelay(attempt); got != 5*time.Second {
			t.Errorf("NextDelay(%d) = %v, want %v", attempt, got, 5*time.Second)
		}
	}
}

func TestConstantBackoff_Default(t *testing.T) {
	backoff := &ConstantBackoff{} // Zero delay

	if got := backoff.NextDelay(1); got != time.Second {
		t.Errorf("Default NextDelay(1) = %v, want %v", got, time.Second)
	}
}

func TestLinearBackoff_NextDelay(t *testing.T) {
	backoff := &LinearBackoff{
		Initial:   time.Second,
		Increment: time.Second,
		Max:       5 * time.Second,
	}

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 3 * time.Second},
		{4, 4 * time.Second},
		{5, 5 * time.Second}, // At max
		{6, 5 * time.Second}, // Capped
		{7, 5 * time.Second}, // Still capped
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := backoff.NextDelay(tt.attempt)
			if got != tt.want {
				t.Errorf("NextDelay(%d) = %v, want %v", tt.attempt, got, tt.want)
			}
		})
	}
}

func TestLinearBackoff_Defaults(t *testing.T) {
	backoff := &LinearBackoff{} // All zero values

	delay := backoff.NextDelay(1)
	if delay != time.Second {
		t.Errorf("Default NextDelay(1) = %v, want %v", delay, time.Second)
	}
}

func TestNoBackoff_NextDelay(t *testing.T) {
	backoff := &NoBackoff{}

	for attempt := 1; attempt <= 5; attempt++ {
		if got := backoff.NextDelay(attempt); got != 0 {
			t.Errorf("NextDelay(%d) = %v, want 0", attempt, got)
		}
	}
}

func TestDefaultBackoff(t *testing.T) {
	backoff := DefaultBackoff()

	if backoff == nil {
		t.Fatal("DefaultBackoff() returned nil")
	}

	// Should return exponential backoff
	exp, ok := backoff.(*ExponentialBackoff)
	if !ok {
		t.Fatal("DefaultBackoff() should return *ExponentialBackoff")
	}

	if exp.Initial != time.Second {
		t.Errorf("Default Initial = %v, want %v", exp.Initial, time.Second)
	}
	if exp.Max != 30*time.Second {
		t.Errorf("Default Max = %v, want %v", exp.Max, 30*time.Second)
	}
	if exp.Multiplier != 2.0 {
		t.Errorf("Default Multiplier = %v, want %v", exp.Multiplier, 2.0)
	}
	if exp.Jitter != 0.1 {
		t.Errorf("Default Jitter = %v, want %v", exp.Jitter, 0.1)
	}
}

func TestBackoffStrategy_Reset(t *testing.T) {
	// All backoff strategies should implement Reset without panicking
	strategies := []BackoffStrategy{
		&ExponentialBackoff{},
		&ConstantBackoff{},
		&LinearBackoff{},
		&NoBackoff{},
	}

	for _, s := range strategies {
		s.Reset() // Should not panic
	}
}

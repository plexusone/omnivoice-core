package tts

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/plexusone/omnivoice-core/resilience"
)

// mockProvider implements Provider for testing.
type mockProvider struct {
	name          string
	synthesizeErr error
	streamErr     error
	calls         int
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Synthesize(ctx context.Context, text string, config SynthesisConfig) (*SynthesisResult, error) {
	m.calls++
	if m.synthesizeErr != nil {
		return nil, m.synthesizeErr
	}
	return &SynthesisResult{Audio: []byte("audio")}, nil
}

func (m *mockProvider) SynthesizeStream(ctx context.Context, text string, config SynthesisConfig) (<-chan StreamChunk, error) {
	m.calls++
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	ch := make(chan StreamChunk, 1)
	ch <- StreamChunk{IsFinal: true}
	close(ch)
	return ch, nil
}

func (m *mockProvider) SynthesizeFromReader(ctx context.Context, reader io.Reader, config SynthesisConfig) (<-chan StreamChunk, error) {
	return m.SynthesizeStream(ctx, "", config)
}

func (m *mockProvider) ListVoices(ctx context.Context) ([]Voice, error) {
	return nil, nil
}

func (m *mockProvider) GetVoice(ctx context.Context, voiceID string) (*Voice, error) {
	return nil, nil
}

func TestShouldFallback(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "retryable ProviderError should not fallback",
			err: resilience.NewProviderError("test", "op", errors.New("rate limit"), resilience.ErrorInfo{
				Category:  resilience.CategoryRateLimit,
				Retryable: true,
			}),
			want: false,
		},
		{
			name: "non-retryable ProviderError should fallback",
			err: resilience.NewProviderError("test", "op", errors.New("auth error"), resilience.ErrorInfo{
				Category:  resilience.CategoryAuth,
				Retryable: false,
			}),
			want: true,
		},
		{
			name: "ErrVoiceNotFound should not fallback",
			err:  ErrVoiceNotFound,
			want: false,
		},
		{
			name: "ErrInvalidConfig should not fallback",
			err:  ErrInvalidConfig,
			want: false,
		},
		{
			name: "unknown error should fallback",
			err:  errors.New("unknown error"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldFallback(tt.err)
			if got != tt.want {
				t.Errorf("shouldFallback() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_SmartFallback_Synthesize(t *testing.T) {
	ctx := context.Background()

	t.Run("fallback on permanent error", func(t *testing.T) {
		primary := &mockProvider{
			name: "primary",
			synthesizeErr: resilience.NewProviderError("primary", "Synthesize", errors.New("auth"), resilience.ErrorInfo{
				Category:  resilience.CategoryAuth,
				Retryable: false,
			}),
		}
		fallback := &mockProvider{name: "fallback"}

		client := NewClient(primary, fallback)
		_, err := client.Synthesize(ctx, "hello", SynthesisConfig{})

		if err != nil {
			t.Errorf("expected success after fallback, got error: %v", err)
		}
		if primary.calls != 1 {
			t.Errorf("primary.calls = %d, want 1", primary.calls)
		}
		if fallback.calls != 1 {
			t.Errorf("fallback.calls = %d, want 1", fallback.calls)
		}
	})

	t.Run("no fallback on retryable error", func(t *testing.T) {
		primary := &mockProvider{
			name: "primary",
			synthesizeErr: resilience.NewProviderError("primary", "Synthesize", errors.New("rate limit"), resilience.ErrorInfo{
				Category:  resilience.CategoryRateLimit,
				Retryable: true,
			}),
		}
		fallback := &mockProvider{name: "fallback"}

		client := NewClient(primary, fallback)
		_, err := client.Synthesize(ctx, "hello", SynthesisConfig{})

		if err == nil {
			t.Error("expected error, got nil")
		}
		if primary.calls != 1 {
			t.Errorf("primary.calls = %d, want 1", primary.calls)
		}
		if fallback.calls != 0 {
			t.Errorf("fallback.calls = %d, want 0 (should not fallback on retryable)", fallback.calls)
		}
	})
}

func TestClient_SmartFallback_SynthesizeStream(t *testing.T) {
	ctx := context.Background()

	t.Run("fallback on permanent error", func(t *testing.T) {
		primary := &mockProvider{
			name: "primary",
			streamErr: resilience.NewProviderError("primary", "SynthesizeStream", errors.New("auth"), resilience.ErrorInfo{
				Category:  resilience.CategoryAuth,
				Retryable: false,
			}),
		}
		fallback := &mockProvider{name: "fallback"}

		client := NewClient(primary, fallback)
		ch, err := client.SynthesizeStream(ctx, "hello", SynthesisConfig{})

		if err != nil {
			t.Errorf("expected success after fallback, got error: %v", err)
		}
		if ch == nil {
			t.Error("expected channel, got nil")
		}
		if primary.calls != 1 {
			t.Errorf("primary.calls = %d, want 1", primary.calls)
		}
		if fallback.calls != 1 {
			t.Errorf("fallback.calls = %d, want 1", fallback.calls)
		}
	})

	t.Run("no fallback on retryable error", func(t *testing.T) {
		primary := &mockProvider{
			name: "primary",
			streamErr: resilience.NewProviderError("primary", "SynthesizeStream", errors.New("server error"), resilience.ErrorInfo{
				Category:  resilience.CategoryServer,
				Retryable: true,
			}),
		}
		fallback := &mockProvider{name: "fallback"}

		client := NewClient(primary, fallback)
		_, err := client.SynthesizeStream(ctx, "hello", SynthesisConfig{})

		if err == nil {
			t.Error("expected error, got nil")
		}
		if primary.calls != 1 {
			t.Errorf("primary.calls = %d, want 1", primary.calls)
		}
		if fallback.calls != 0 {
			t.Errorf("fallback.calls = %d, want 0 (should not fallback on retryable)", fallback.calls)
		}
	})
}

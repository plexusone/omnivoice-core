package stt

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
	transcribeErr error
	streamErr     error
	calls         int
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Transcribe(ctx context.Context, audio []byte, config TranscriptionConfig) (*TranscriptionResult, error) {
	m.calls++
	if m.transcribeErr != nil {
		return nil, m.transcribeErr
	}
	return &TranscriptionResult{Text: "hello"}, nil
}

func (m *mockProvider) TranscribeFile(ctx context.Context, filePath string, config TranscriptionConfig) (*TranscriptionResult, error) {
	return m.Transcribe(ctx, nil, config)
}

func (m *mockProvider) TranscribeURL(ctx context.Context, url string, config TranscriptionConfig) (*TranscriptionResult, error) {
	return m.Transcribe(ctx, nil, config)
}

// mockStreamingProvider implements StreamingProvider for testing.
type mockStreamingProvider struct {
	mockProvider
}

func (m *mockStreamingProvider) TranscribeStream(ctx context.Context, config TranscriptionConfig) (io.WriteCloser, <-chan StreamEvent, error) {
	m.calls++
	if m.streamErr != nil {
		return nil, nil, m.streamErr
	}
	ch := make(chan StreamEvent, 1)
	ch <- StreamEvent{Type: EventTranscript, IsFinal: true}
	close(ch)
	return &nopWriteCloser{}, ch, nil
}

type nopWriteCloser struct{}

func (n *nopWriteCloser) Write(p []byte) (int, error) { return len(p), nil }
func (n *nopWriteCloser) Close() error                { return nil }

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

func TestClient_SmartFallback_Transcribe(t *testing.T) {
	ctx := context.Background()

	t.Run("fallback on permanent error", func(t *testing.T) {
		primary := &mockProvider{
			name: "primary",
			transcribeErr: resilience.NewProviderError("primary", "Transcribe", errors.New("auth"), resilience.ErrorInfo{
				Category:  resilience.CategoryAuth,
				Retryable: false,
			}),
		}
		fallback := &mockProvider{name: "fallback"}

		client := NewClient(primary, fallback)
		_, err := client.Transcribe(ctx, []byte("audio"), TranscriptionConfig{})

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
			transcribeErr: resilience.NewProviderError("primary", "Transcribe", errors.New("rate limit"), resilience.ErrorInfo{
				Category:  resilience.CategoryRateLimit,
				Retryable: true,
			}),
		}
		fallback := &mockProvider{name: "fallback"}

		client := NewClient(primary, fallback)
		_, err := client.Transcribe(ctx, []byte("audio"), TranscriptionConfig{})

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

func TestClient_SmartFallback_TranscribeStream(t *testing.T) {
	ctx := context.Background()

	t.Run("fallback on permanent error", func(t *testing.T) {
		primary := &mockStreamingProvider{
			mockProvider: mockProvider{
				name: "primary",
				streamErr: resilience.NewProviderError("primary", "TranscribeStream", errors.New("auth"), resilience.ErrorInfo{
					Category:  resilience.CategoryAuth,
					Retryable: false,
				}),
			},
		}
		fallback := &mockStreamingProvider{
			mockProvider: mockProvider{name: "fallback"},
		}

		client := NewClient(primary, fallback)
		w, ch, err := client.TranscribeStream(ctx, TranscriptionConfig{})

		if err != nil {
			t.Errorf("expected success after fallback, got error: %v", err)
		}
		if w == nil || ch == nil {
			t.Error("expected writer and channel, got nil")
		}
		if primary.calls != 1 {
			t.Errorf("primary.calls = %d, want 1", primary.calls)
		}
		if fallback.calls != 1 {
			t.Errorf("fallback.calls = %d, want 1", fallback.calls)
		}
	})

	t.Run("no fallback on retryable error", func(t *testing.T) {
		primary := &mockStreamingProvider{
			mockProvider: mockProvider{
				name: "primary",
				streamErr: resilience.NewProviderError("primary", "TranscribeStream", errors.New("server error"), resilience.ErrorInfo{
					Category:  resilience.CategoryServer,
					Retryable: true,
				}),
			},
		}
		fallback := &mockStreamingProvider{
			mockProvider: mockProvider{name: "fallback"},
		}

		client := NewClient(primary, fallback)
		_, _, err := client.TranscribeStream(ctx, TranscriptionConfig{})

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

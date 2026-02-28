package providertest

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/plexusone/omnivoice/stt"
)

// Verify MockProvider implements the interfaces.
var (
	_ stt.Provider          = (*MockProvider)(nil)
	_ stt.StreamingProvider = (*MockProvider)(nil)
)

// MockProvider is a minimal STT provider for testing the test suite itself.
// It returns fake transcription results and can be configured to simulate various behaviors.
type MockProvider struct {
	name string

	// TranscribeFunc allows customizing Transcribe behavior for testing.
	// If nil, returns default fake transcription.
	TranscribeFunc func(ctx context.Context, audio []byte, config stt.TranscriptionConfig) (*stt.TranscriptionResult, error)
}

// NewMockProvider creates a new mock STT provider.
func NewMockProvider() *MockProvider {
	return &MockProvider{
		name: "mock",
	}
}

// Name returns the provider name.
func (m *MockProvider) Name() string {
	return m.name
}

// Transcribe returns fake transcription for the given audio.
func (m *MockProvider) Transcribe(ctx context.Context, audio []byte, config stt.TranscriptionConfig) (*stt.TranscriptionResult, error) {
	if m.TranscribeFunc != nil {
		return m.TranscribeFunc(ctx, audio, config)
	}

	// Check context
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Return fake transcription
	return &stt.TranscriptionResult{
		Text:     "Hello, this is a test transcription.",
		Language: "en-US",
		Duration: time.Duration(len(audio)) * time.Microsecond / 32, // Approximate duration
		Segments: []stt.Segment{
			{
				Text:       "Hello, this is a test transcription.",
				StartTime:  0,
				EndTime:    time.Second * 2,
				Confidence: 0.95,
				Language:   "en-US",
				Words: []stt.Word{
					{Text: "Hello", StartTime: 0, EndTime: 300 * time.Millisecond, Confidence: 0.98},
					{Text: "this", StartTime: 350 * time.Millisecond, EndTime: 500 * time.Millisecond, Confidence: 0.95},
					{Text: "is", StartTime: 550 * time.Millisecond, EndTime: 650 * time.Millisecond, Confidence: 0.99},
					{Text: "a", StartTime: 700 * time.Millisecond, EndTime: 750 * time.Millisecond, Confidence: 0.97},
					{Text: "test", StartTime: 800 * time.Millisecond, EndTime: 1100 * time.Millisecond, Confidence: 0.96},
					{Text: "transcription", StartTime: 1150 * time.Millisecond, EndTime: 2000 * time.Millisecond, Confidence: 0.94},
				},
			},
		},
		LanguageConfidence: 0.99,
	}, nil
}

// TranscribeFile returns fake transcription for a file.
func (m *MockProvider) TranscribeFile(ctx context.Context, filePath string, config stt.TranscriptionConfig) (*stt.TranscriptionResult, error) {
	// Check context
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Delegate to Transcribe with fake audio
	return m.Transcribe(ctx, []byte("fake-audio-from-file"), config)
}

// TranscribeURL returns fake transcription for a URL.
func (m *MockProvider) TranscribeURL(ctx context.Context, url string, config stt.TranscriptionConfig) (*stt.TranscriptionResult, error) {
	// Check context
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Delegate to Transcribe with fake audio
	return m.Transcribe(ctx, []byte("fake-audio-from-url"), config)
}

// TranscribeStream returns a streaming transcription session.
func (m *MockProvider) TranscribeStream(ctx context.Context, config stt.TranscriptionConfig) (io.WriteCloser, <-chan stt.StreamEvent, error) {
	// Check context
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}

	events := make(chan stt.StreamEvent, 20)
	writer := &mockStreamWriter{
		ctx:    ctx,
		events: events,
	}

	return writer, events, nil
}

// mockStreamWriter simulates streaming audio input and generating events.
type mockStreamWriter struct {
	ctx       context.Context
	events    chan stt.StreamEvent
	mu        sync.Mutex
	closed    bool
	totalData int
}

func (w *mockStreamWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return 0, stt.ErrStreamClosed
	}

	select {
	case <-w.ctx.Done():
		return 0, w.ctx.Err()
	default:
	}

	// Track data received
	firstWrite := w.totalData == 0
	w.totalData += len(p)

	// Emit speech start on first data
	if firstWrite {
		select {
		case w.events <- stt.StreamEvent{
			Type:          stt.EventSpeechStart,
			SpeechStarted: true,
		}:
		case <-w.ctx.Done():
			return 0, w.ctx.Err()
		}

		// Emit interim result
		select {
		case w.events <- stt.StreamEvent{
			Type:       stt.EventTranscript,
			Transcript: "Hello",
			IsFinal:    false,
		}:
		case <-w.ctx.Done():
			return 0, w.ctx.Err()
		}
	}

	return len(p), nil
}

func (w *mockStreamWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}
	w.closed = true

	// Emit final result
	select {
	case w.events <- stt.StreamEvent{
		Type:       stt.EventTranscript,
		Transcript: "Hello, this is a test transcription.",
		IsFinal:    true,
		Segment: &stt.Segment{
			Text:       "Hello, this is a test transcription.",
			StartTime:  0,
			EndTime:    2 * time.Second,
			Confidence: 0.95,
		},
	}:
	default:
	}

	// Emit speech end
	select {
	case w.events <- stt.StreamEvent{
		Type:        stt.EventSpeechEnd,
		SpeechEnded: true,
	}:
	default:
	}

	// Close events channel
	close(w.events)
	return nil
}

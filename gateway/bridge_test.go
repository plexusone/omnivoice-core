package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/plexusone/omnivoice-core/audio/format"
	"github.com/plexusone/omnivoice-core/realtime"
)

// mockRealtimeProvider implements realtime.Provider for testing.
type mockRealtimeProvider struct {
	name         string
	audioOut     chan realtime.AudioChunk
	transcriptCh chan realtime.Transcript
	closed       bool
}

func newMockRealtimeProvider() *mockRealtimeProvider {
	return &mockRealtimeProvider{
		name:         "mock",
		audioOut:     make(chan realtime.AudioChunk, 100),
		transcriptCh: make(chan realtime.Transcript, 100),
	}
}

func (m *mockRealtimeProvider) ProcessAudioStream(
	ctx context.Context,
	audioIn <-chan []byte,
	config realtime.ProcessConfig,
) (<-chan realtime.AudioChunk, <-chan realtime.Transcript, error) {
	// Echo back audio with a simple transformation
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case audio, ok := <-audioIn:
				if !ok {
					return
				}
				// Echo audio back
				m.audioOut <- realtime.AudioChunk{Audio: audio}
				// Send transcript
				m.transcriptCh <- realtime.Transcript{Text: "test", IsInput: false}
			}
		}
	}()

	return m.audioOut, m.transcriptCh, nil
}

func (m *mockRealtimeProvider) Name() string {
	return m.name
}

func (m *mockRealtimeProvider) Close() error {
	m.closed = true
	close(m.audioOut)
	close(m.transcriptCh)
	return nil
}

func TestNewRealtimeBridge(t *testing.T) {
	provider := newMockRealtimeProvider()
	bridge := NewRealtimeBridge(BridgeConfig{
		Provider:   provider,
		FromFormat: format.Twilio,
		ToFormat:   format.OpenAI,
	})

	if bridge == nil {
		t.Fatal("expected non-nil bridge")
	}
	if bridge.provider != provider {
		t.Error("provider not set correctly")
	}
}

func TestRealtimeBridge_StartAndClose(t *testing.T) {
	provider := newMockRealtimeProvider()
	bridge := NewRealtimeBridge(BridgeConfig{
		Provider:   provider,
		FromFormat: format.Twilio,
		ToFormat:   format.OpenAI,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := bridge.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected error starting bridge: %v", err)
	}

	// Check that events channel is working
	select {
	case event := <-bridge.Events():
		if event.Type != EventSessionStarted {
			t.Errorf("expected EventSessionStarted, got %v", event.Type)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for session started event")
	}

	err = bridge.Close()
	if err != nil {
		t.Fatalf("unexpected error closing bridge: %v", err)
	}

	if !provider.closed {
		t.Error("expected provider to be closed")
	}
}

func TestRealtimeBridge_SendAudio(t *testing.T) {
	provider := newMockRealtimeProvider()
	bridge := NewRealtimeBridge(BridgeConfig{
		Provider:   provider,
		FromFormat: format.OpenAI, // Use same format to avoid conversion complexity in test
		ToFormat:   format.OpenAI,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := bridge.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected error starting bridge: %v", err)
	}
	defer bridge.Close()

	// Drain the session started event
	<-bridge.Events()

	// Send audio
	testAudio := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	err = bridge.SendAudio(testAudio)
	if err != nil {
		t.Fatalf("unexpected error sending audio: %v", err)
	}

	// Should receive audio back from mock provider
	select {
	case audio := <-bridge.AudioOut():
		if len(audio) == 0 {
			t.Error("expected non-empty audio output")
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for audio output")
	}
}

func TestRealtimeBridge_Metrics(t *testing.T) {
	provider := newMockRealtimeProvider()
	bridge := NewRealtimeBridge(BridgeConfig{
		Provider:   provider,
		FromFormat: format.OpenAI,
		ToFormat:   format.OpenAI,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := bridge.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected error starting bridge: %v", err)
	}
	defer bridge.Close()

	// Drain the session started event
	<-bridge.Events()

	// Send some audio
	testAudio := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	_ = bridge.SendAudio(testAudio)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	metrics := bridge.Metrics()
	if metrics.SessionDurationMs == 0 {
		t.Error("expected non-zero session duration")
	}
	if metrics.AudioBytesReceived == 0 {
		t.Error("expected non-zero audio bytes received")
	}
}

func TestRealtimeBridge_Interrupt(t *testing.T) {
	provider := newMockRealtimeProvider()
	bridge := NewRealtimeBridge(BridgeConfig{
		Provider:   provider,
		FromFormat: format.OpenAI,
		ToFormat:   format.OpenAI,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := bridge.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected error starting bridge: %v", err)
	}
	defer bridge.Close()

	// Drain the session started event
	<-bridge.Events()

	// Trigger interrupt
	bridge.Interrupt()

	// Check that interruption event was sent
	select {
	case event := <-bridge.Events():
		if event.Type != EventInterruption {
			t.Errorf("expected EventInterruption, got %v", event.Type)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for interruption event")
	}

	metrics := bridge.Metrics()
	if metrics.InterruptionCount != 1 {
		t.Errorf("expected 1 interruption, got %d", metrics.InterruptionCount)
	}
}

func TestRealtimeBridge_Transcript(t *testing.T) {
	provider := newMockRealtimeProvider()
	bridge := NewRealtimeBridge(BridgeConfig{
		Provider:   provider,
		FromFormat: format.OpenAI,
		ToFormat:   format.OpenAI,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := bridge.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected error starting bridge: %v", err)
	}
	defer bridge.Close()

	// Initially empty
	transcript := bridge.Transcript()
	if len(transcript) != 0 {
		t.Errorf("expected empty transcript, got %d turns", len(transcript))
	}
}

func TestNewRealtimeBridgeForTwilio(t *testing.T) {
	provider := newMockRealtimeProvider()
	bridge := NewRealtimeBridgeForTwilio(provider, realtime.ProcessConfig{
		Instructions: "Test instructions",
	})

	if bridge == nil {
		t.Fatal("expected non-nil bridge")
	}
	if bridge.fromFormat != format.Twilio {
		t.Errorf("expected Twilio format, got %v", bridge.fromFormat)
	}
	if bridge.toFormat != format.OpenAI {
		t.Errorf("expected OpenAI format, got %v", bridge.toFormat)
	}
}

func TestNewRealtimeBridgeForTwilioGemini(t *testing.T) {
	provider := newMockRealtimeProvider()
	bridge := NewRealtimeBridgeForTwilioGemini(provider, realtime.ProcessConfig{
		Instructions: "Test instructions",
	})

	if bridge == nil {
		t.Fatal("expected non-nil bridge")
	}
	if bridge.fromFormat != format.Twilio {
		t.Errorf("expected Twilio format, got %v", bridge.fromFormat)
	}
	if bridge.toFormat != format.GeminiInput {
		t.Errorf("expected GeminiInput format, got %v", bridge.toFormat)
	}
}

func TestBridgeConfig_DefaultBufferSize(t *testing.T) {
	provider := newMockRealtimeProvider()
	bridge := NewRealtimeBridge(BridgeConfig{
		Provider:   provider,
		FromFormat: format.Twilio,
		ToFormat:   format.OpenAI,
		// BufferSize not set - should default to 100
	})

	if cap(bridge.audioIn) != 100 {
		t.Errorf("expected buffer size 100, got %d", cap(bridge.audioIn))
	}
}

func TestBridgeConfig_CustomBufferSize(t *testing.T) {
	provider := newMockRealtimeProvider()
	bridge := NewRealtimeBridge(BridgeConfig{
		Provider:   provider,
		FromFormat: format.Twilio,
		ToFormat:   format.OpenAI,
		BufferSize: 50,
	})

	if cap(bridge.audioIn) != 50 {
		t.Errorf("expected buffer size 50, got %d", cap(bridge.audioIn))
	}
}

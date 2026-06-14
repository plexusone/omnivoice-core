package realtime

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/plexusone/omnivoice-core/audio/format"
)

// mockProvider is a mock implementation for testing.
type mockProvider struct {
	name     string
	audioCh  chan AudioChunk
	transCh  chan Transcript
	closeErr error
}

func newMockProvider(name string) *mockProvider {
	return &mockProvider{
		name:    name,
		audioCh: make(chan AudioChunk, 10),
		transCh: make(chan Transcript, 10),
	}
}

func (p *mockProvider) ProcessAudioStream(ctx context.Context, audioIn <-chan []byte, config ProcessConfig) (<-chan AudioChunk, <-chan Transcript, error) {
	return p.audioCh, p.transCh, nil
}

func (p *mockProvider) Name() string {
	return p.name
}

func (p *mockProvider) Close() error {
	close(p.audioCh)
	close(p.transCh)
	return p.closeErr
}

// Verify mockProvider implements Provider
var _ Provider = (*mockProvider)(nil)

func TestNewClient(t *testing.T) {
	p1 := newMockProvider("provider1")
	p2 := newMockProvider("provider2")

	client := NewClient(p1, p2)

	// Primary should be first provider
	primary, ok := client.Primary()
	if !ok {
		t.Fatal("expected primary provider")
	}
	if primary.Name() != "provider1" {
		t.Errorf("expected primary 'provider1', got %q", primary.Name())
	}

	// Can get specific provider
	p, ok := client.Provider("provider2")
	if !ok {
		t.Fatal("expected provider2")
	}
	if p.Name() != "provider2" {
		t.Errorf("expected 'provider2', got %q", p.Name())
	}
}

func TestSetPrimary(t *testing.T) {
	p1 := newMockProvider("provider1")
	p2 := newMockProvider("provider2")

	client := NewClient(p1, p2)
	client.SetPrimary("provider2")

	primary, _ := client.Primary()
	if primary.Name() != "provider2" {
		t.Errorf("expected primary 'provider2', got %q", primary.Name())
	}
}

func TestProcessConfig(t *testing.T) {
	params, _ := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"location": map[string]any{"type": "string"},
		},
	})

	config := ProcessConfig{
		Instructions: "You are a helpful assistant.",
		Voice:        "alloy",
		Temperature:  0.8,
		Functions: []FunctionDeclaration{
			{
				Name:        "get_weather",
				Description: "Get the current weather",
				Parameters:  params,
			},
		},
		OnFunctionCall: func(id, name, args string) (any, error) {
			return map[string]any{"temp": 72}, nil
		},
	}

	if config.Instructions != "You are a helpful assistant." {
		t.Error("Instructions not set correctly")
	}
	if len(config.Functions) != 1 {
		t.Errorf("expected 1 function, got %d", len(config.Functions))
	}
	if config.Functions[0].Name != "get_weather" {
		t.Errorf("expected function 'get_weather', got %q", config.Functions[0].Name)
	}
}

func TestAudioChunk(t *testing.T) {
	chunk := AudioChunk{
		Audio:   []byte{0x00, 0x01, 0x02},
		IsFinal: true,
	}

	if len(chunk.Audio) != 3 {
		t.Errorf("expected 3 bytes, got %d", len(chunk.Audio))
	}
	if !chunk.IsFinal {
		t.Error("expected IsFinal to be true")
	}
}

func TestTranscript(t *testing.T) {
	// User input
	userTranscript := Transcript{
		Text:    "Hello",
		IsFinal: true,
		IsInput: true,
		ItemID:  "item1",
	}

	if userTranscript.Role() != "user" {
		t.Errorf("expected role 'user', got %q", userTranscript.Role())
	}

	// Assistant output
	assistantTranscript := Transcript{
		Text:    "Hi there!",
		IsFinal: true,
		IsInput: false,
		ItemID:  "item2",
	}

	if assistantTranscript.Role() != "assistant" {
		t.Errorf("expected role 'assistant', got %q", assistantTranscript.Role())
	}
}

func TestAudioFormat(t *testing.T) {
	if format.PCM16_24kHz.SampleRate != 24000 {
		t.Errorf("expected 24000, got %d", format.PCM16_24kHz.SampleRate)
	}
	if format.PCM16_16kHz.SampleRate != 16000 {
		t.Errorf("expected 16000, got %d", format.PCM16_16kHz.SampleRate)
	}
}

func TestShouldFallback(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{ErrConnectionFailed, true},
		{ErrProviderUnavailable, true},
		{ErrRateLimited, true},
		{ErrInvalidConfig, false},
		{ErrAuthenticationFailed, false},
	}

	for _, tt := range tests {
		if got := shouldFallback(tt.err); got != tt.expected {
			t.Errorf("shouldFallback(%v) = %v, want %v", tt.err, got, tt.expected)
		}
	}
}

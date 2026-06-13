package bargein

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/plexusone/omnivoice-core/gateway"
	"github.com/plexusone/omnivoice-core/stt"
)

// mockTTSController implements TTSController for testing.
type mockTTSController struct {
	mu      sync.Mutex
	active  bool
	stopped int
}

func (m *mockTTSController) IsActive() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.active
}

func (m *mockTTSController) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.active = false
	m.stopped++
}

func (m *mockTTSController) SetActive(active bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.active = active
}

func (m *mockTTSController) StopCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopped
}

func TestDetector_ImmediateMode(t *testing.T) {
	events := make(chan stt.StreamEvent, 10)
	tts := &mockTTSController{active: true}

	var interrupted bool
	var interruptMu sync.Mutex

	detector := NewDetector(Config{
		Mode:                ModeImmediate,
		MinSpeechDurationMs: 0, // Immediate for testing
		MinAgentSpeechMs:    0,
		CooldownMs:          0,
	})

	detector.AttachTTS(tts)
	detector.AttachSTTEvents(events)
	detector.OnInterrupt(func(event gateway.Event) {
		interruptMu.Lock()
		interrupted = true
		interruptMu.Unlock()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := detector.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !detector.IsRunning() {
		t.Error("expected detector to be running")
	}

	// Simulate agent speaking
	detector.SetAgentSpeaking(true)

	// Simulate user speech start
	events <- stt.StreamEvent{Type: stt.EventSpeechStart}

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	interruptMu.Lock()
	wasInterrupted := interrupted
	interruptMu.Unlock()

	if !wasInterrupted {
		t.Error("expected interrupt to be triggered in immediate mode")
	}

	if tts.StopCount() != 1 {
		t.Errorf("expected TTS Stop() to be called once, got %d", tts.StopCount())
	}

	detector.Stop()

	if detector.IsRunning() {
		t.Error("expected detector to be stopped")
	}
}

func TestDetector_DisabledMode(t *testing.T) {
	events := make(chan stt.StreamEvent, 10)
	tts := &mockTTSController{active: true}

	var interrupted bool

	detector := NewDetector(Config{
		Mode: ModeDisabled,
	})

	detector.AttachTTS(tts)
	detector.AttachSTTEvents(events)
	detector.OnInterrupt(func(event gateway.Event) {
		interrupted = true
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// In disabled mode, Start should return nil but not actually run
	if err := detector.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	detector.SetAgentSpeaking(true)
	events <- stt.StreamEvent{Type: stt.EventSpeechStart}

	time.Sleep(50 * time.Millisecond)

	if interrupted {
		t.Error("expected no interrupt in disabled mode")
	}

	if tts.StopCount() != 0 {
		t.Errorf("expected TTS Stop() not to be called, got %d", tts.StopCount())
	}
}

func TestDetector_NoAgentSpeaking(t *testing.T) {
	events := make(chan stt.StreamEvent, 10)
	tts := &mockTTSController{active: true}

	var interrupted bool

	detector := NewDetector(Config{
		Mode:                ModeImmediate,
		MinSpeechDurationMs: 0,
		MinAgentSpeechMs:    0,
		CooldownMs:          0,
	})

	detector.AttachTTS(tts)
	detector.AttachSTTEvents(events)
	detector.OnInterrupt(func(event gateway.Event) {
		interrupted = true
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := detector.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Do NOT set agent speaking
	events <- stt.StreamEvent{Type: stt.EventSpeechStart}

	time.Sleep(50 * time.Millisecond)

	if interrupted {
		t.Error("expected no interrupt when agent is not speaking")
	}

	detector.Stop()
}

func TestDetector_MinSpeechDuration(t *testing.T) {
	events := make(chan stt.StreamEvent, 10)
	tts := &mockTTSController{active: true}

	var interrupted bool
	var interruptMu sync.Mutex

	detector := NewDetector(Config{
		Mode:                ModeImmediate,
		MinSpeechDurationMs: 100, // 100ms minimum
		MinAgentSpeechMs:    0,
		CooldownMs:          0,
	})

	detector.AttachTTS(tts)
	detector.AttachSTTEvents(events)
	detector.OnInterrupt(func(event gateway.Event) {
		interruptMu.Lock()
		interrupted = true
		interruptMu.Unlock()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := detector.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	detector.SetAgentSpeaking(true)

	// Simulate speech start
	events <- stt.StreamEvent{Type: stt.EventSpeechStart}

	// Check immediately - should not interrupt yet
	time.Sleep(20 * time.Millisecond)

	interruptMu.Lock()
	wasInterrupted := interrupted
	interruptMu.Unlock()

	if wasInterrupted {
		t.Error("expected no interrupt before min speech duration")
	}

	// Wait past min duration and send transcript (which checks again)
	time.Sleep(100 * time.Millisecond)
	events <- stt.StreamEvent{Type: stt.EventTranscript, Transcript: "hello"}

	time.Sleep(50 * time.Millisecond)

	interruptMu.Lock()
	wasInterrupted = interrupted
	interruptMu.Unlock()

	if !wasInterrupted {
		t.Error("expected interrupt after min speech duration")
	}

	detector.Stop()
}

func TestDetector_Cooldown(t *testing.T) {
	events := make(chan stt.StreamEvent, 10)
	tts := &mockTTSController{}

	var interruptCount int
	var mu sync.Mutex

	detector := NewDetector(Config{
		Mode:                ModeImmediate,
		MinSpeechDurationMs: 0,
		MinAgentSpeechMs:    0,
		CooldownMs:          100, // 100ms cooldown
	})

	detector.AttachTTS(tts)
	detector.AttachSTTEvents(events)
	detector.OnInterrupt(func(event gateway.Event) {
		mu.Lock()
		interruptCount++
		mu.Unlock()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := detector.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// First interrupt
	detector.SetAgentSpeaking(true)
	tts.SetActive(true)
	events <- stt.StreamEvent{Type: stt.EventSpeechStart}
	time.Sleep(20 * time.Millisecond)

	// Try to trigger again immediately (should be blocked by cooldown)
	detector.SetAgentSpeaking(true)
	tts.SetActive(true)
	events <- stt.StreamEvent{Type: stt.EventSpeechStart}
	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	count := interruptCount
	mu.Unlock()

	if count != 1 {
		t.Errorf("expected 1 interrupt due to cooldown, got %d", count)
	}

	// Wait for cooldown and try again
	time.Sleep(100 * time.Millisecond)
	detector.SetAgentSpeaking(true)
	tts.SetActive(true)
	events <- stt.StreamEvent{Type: stt.EventSpeechStart}
	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	count = interruptCount
	mu.Unlock()

	if count != 2 {
		t.Errorf("expected 2 interrupts after cooldown, got %d", count)
	}

	detector.Stop()
}

func TestDetector_StartWithoutEvents(t *testing.T) {
	detector := NewDetector(DefaultConfig())

	ctx := context.Background()
	err := detector.Start(ctx)

	if err != ErrNoSTTEvents {
		t.Errorf("expected ErrNoSTTEvents, got %v", err)
	}
}

func TestConfig_Validate(t *testing.T) {
	// Empty config should preserve zeros (explicit "no minimum")
	// but should get default mode
	empty := Config{}
	cfg := empty.Validate()

	if cfg.Mode != ModeImmediate {
		t.Errorf("expected default mode ModeImmediate, got %q", cfg.Mode)
	}

	// Zero values should be preserved (not replaced with defaults)
	if cfg.MinSpeechDurationMs != 0 {
		t.Errorf("expected MinSpeechDurationMs 0 to be preserved, got %d", cfg.MinSpeechDurationMs)
	}

	// Negative values should get defaults
	negative := Config{
		MinSpeechDurationMs: -1,
		SilenceThresholdMs:  -1,
		CooldownMs:          -1,
		MinAgentSpeechMs:    -1,
	}
	cfg = negative.Validate()

	if cfg.MinSpeechDurationMs != 200 {
		t.Errorf("expected negative MinSpeechDurationMs to become 200, got %d", cfg.MinSpeechDurationMs)
	}
	if cfg.SilenceThresholdMs != 500 {
		t.Errorf("expected negative SilenceThresholdMs to become 500, got %d", cfg.SilenceThresholdMs)
	}
	if cfg.CooldownMs != 300 {
		t.Errorf("expected negative CooldownMs to become 300, got %d", cfg.CooldownMs)
	}
	if cfg.MinAgentSpeechMs != 500 {
		t.Errorf("expected negative MinAgentSpeechMs to become 500, got %d", cfg.MinAgentSpeechMs)
	}
}

func TestConfig_DurationHelpers(t *testing.T) {
	cfg := Config{
		MinSpeechDurationMs: 200,
		SilenceThresholdMs:  500,
		CooldownMs:          300,
		MinAgentSpeechMs:    500,
	}

	if cfg.MinSpeechDuration() != 200*time.Millisecond {
		t.Error("MinSpeechDuration helper incorrect")
	}
	if cfg.SilenceThreshold() != 500*time.Millisecond {
		t.Error("SilenceThreshold helper incorrect")
	}
	if cfg.Cooldown() != 300*time.Millisecond {
		t.Error("Cooldown helper incorrect")
	}
	if cfg.MinAgentSpeech() != 500*time.Millisecond {
		t.Error("MinAgentSpeech helper incorrect")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Mode != ModeImmediate {
		t.Error("default mode should be ModeImmediate")
	}
	if cfg.MinSpeechDurationMs != 200 {
		t.Errorf("expected MinSpeechDurationMs 200, got %d", cfg.MinSpeechDurationMs)
	}
	if cfg.SilenceThresholdMs != 500 {
		t.Errorf("expected SilenceThresholdMs 500, got %d", cfg.SilenceThresholdMs)
	}
	if cfg.CooldownMs != 300 {
		t.Errorf("expected CooldownMs 300, got %d", cfg.CooldownMs)
	}
	if cfg.MinAgentSpeechMs != 500 {
		t.Errorf("expected MinAgentSpeechMs 500, got %d", cfg.MinAgentSpeechMs)
	}
}

func TestDetector_SpeechEndResetsState(t *testing.T) {
	events := make(chan stt.StreamEvent, 10)
	tts := &mockTTSController{active: true}

	detector := NewDetector(Config{
		Mode:                ModeImmediate,
		MinSpeechDurationMs: 0,
		MinAgentSpeechMs:    0,
		CooldownMs:          0,
	})

	detector.AttachTTS(tts)
	detector.AttachSTTEvents(events)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := detector.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Speech start and end
	events <- stt.StreamEvent{Type: stt.EventSpeechStart}
	time.Sleep(10 * time.Millisecond)
	events <- stt.StreamEvent{Type: stt.EventSpeechEnd}
	time.Sleep(10 * time.Millisecond)

	// Check internal state (via behavior)
	// After speech end, the detector should have userSpeaking = false
	// We verify this indirectly by checking that no interrupt happens
	// when agent starts speaking after user stopped
	detector.SetAgentSpeaking(true)
	time.Sleep(20 * time.Millisecond)

	// TTS should still be active (no interrupt triggered)
	if !tts.IsActive() {
		t.Error("TTS should not have been stopped when user is not speaking")
	}

	detector.Stop()
}

package bargein

import (
	"context"
	"sync"
	"time"

	"github.com/plexusone/omnivoice-core/gateway"
	"github.com/plexusone/omnivoice-core/stt"
)

// TTSController is the interface for controlling TTS playback.
// This is satisfied by pipeline.TTSPipeline.
type TTSController interface {
	// IsActive returns whether TTS is currently playing.
	IsActive() bool

	// Stop stops any active TTS playback.
	Stop()
}

// InterruptHandler is called when barge-in is triggered.
type InterruptHandler func(event gateway.Event)

// Detector monitors STT events and triggers barge-in when appropriate.
type Detector struct {
	config Config

	mu                sync.Mutex
	ttsController     TTSController
	sttEvents         <-chan stt.StreamEvent
	onInterrupt       InterruptHandler
	userSpeaking      bool
	agentSpeaking     bool
	speechStartTime   time.Time
	agentSpeechStart  time.Time
	lastInterruptTime time.Time

	running  bool
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
}

// NewDetector creates a new barge-in detector with the given configuration.
func NewDetector(cfg Config) *Detector {
	return &Detector{
		config: cfg.Validate(),
	}
}

// AttachTTS sets the TTS controller to stop when barge-in is triggered.
func (d *Detector) AttachTTS(tts TTSController) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ttsController = tts
}

// AttachSTTEvents sets the STT event channel to monitor.
func (d *Detector) AttachSTTEvents(events <-chan stt.StreamEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sttEvents = events
}

// OnInterrupt sets the handler called when barge-in is triggered.
func (d *Detector) OnInterrupt(handler InterruptHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.onInterrupt = handler
}

// SetAgentSpeaking notifies the detector that the agent has started or stopped speaking.
func (d *Detector) SetAgentSpeaking(speaking bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.agentSpeaking = speaking
	if speaking {
		d.agentSpeechStart = time.Now()
	}
}

// Start begins monitoring STT events for barge-in.
// Call Stop() to end monitoring.
func (d *Detector) Start(ctx context.Context) error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return nil
	}
	if d.sttEvents == nil {
		d.mu.Unlock()
		return ErrNoSTTEvents
	}
	if d.config.Mode == ModeDisabled {
		d.mu.Unlock()
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	d.cancelFn = cancel
	d.running = true
	d.mu.Unlock()

	d.wg.Add(1)
	go d.monitor(ctx)

	return nil
}

// Stop ends monitoring and releases resources.
func (d *Detector) Stop() {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return
	}
	d.running = false
	if d.cancelFn != nil {
		d.cancelFn()
	}
	d.mu.Unlock()

	d.wg.Wait()
}

// IsRunning returns whether the detector is actively monitoring.
func (d *Detector) IsRunning() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.running
}

// monitor is the main loop that processes STT events.
func (d *Detector) monitor(ctx context.Context) {
	defer d.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-d.sttEvents:
			if !ok {
				return
			}
			d.handleSTTEvent(event)
		}
	}
}

// handleSTTEvent processes a single STT event.
func (d *Detector) handleSTTEvent(event stt.StreamEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()

	switch event.Type {
	case stt.EventSpeechStart:
		d.userSpeaking = true
		d.speechStartTime = time.Now()

		// Check if we should interrupt
		if d.shouldInterrupt() {
			d.triggerInterrupt()
		}

	case stt.EventSpeechEnd:
		d.userSpeaking = false

	case stt.EventTranscript:
		// For immediate mode, any transcript while agent speaking triggers interrupt
		if d.config.Mode == ModeImmediate && d.agentSpeaking && event.Transcript != "" {
			if d.shouldInterrupt() {
				d.triggerInterrupt()
			}
		}
	}
}

// shouldInterrupt determines if barge-in should be triggered.
// Must be called with d.mu held.
func (d *Detector) shouldInterrupt() bool {
	// Mode check
	if d.config.Mode == ModeDisabled {
		return false
	}

	// Must be agent speaking to interrupt
	if !d.agentSpeaking {
		return false
	}

	// Check TTS is actually active
	if d.ttsController != nil && !d.ttsController.IsActive() {
		return false
	}

	// Cooldown check
	if time.Since(d.lastInterruptTime) < d.config.Cooldown() {
		return false
	}

	// Minimum agent speech duration check
	if time.Since(d.agentSpeechStart) < d.config.MinAgentSpeech() {
		return false
	}

	// Minimum user speech duration check
	if d.config.Mode == ModeImmediate {
		// For immediate mode, check if user has been speaking long enough
		if time.Since(d.speechStartTime) < d.config.MinSpeechDuration() {
			return false
		}
	}

	return true
}

// triggerInterrupt executes the barge-in.
// Must be called with d.mu held.
func (d *Detector) triggerInterrupt() {
	d.lastInterruptTime = time.Now()
	d.agentSpeaking = false

	// Stop TTS
	if d.ttsController != nil {
		d.ttsController.Stop()
	}

	// Call interrupt handler
	if d.onInterrupt != nil {
		event := gateway.Event{
			Type:      gateway.EventInterruption,
			Timestamp: time.Now(),
			Data: map[string]any{
				"mode":                    string(d.config.Mode),
				"user_speech_duration_ms": time.Since(d.speechStartTime).Milliseconds(),
			},
		}
		// Call handler without holding lock
		handler := d.onInterrupt
		d.mu.Unlock()
		handler(event)
		d.mu.Lock()
	}
}

// Package observability provides instrumentation interfaces for voice operations.
package observability

import (
	"context"
	"time"
)

// EventType identifies the type of voice event.
type EventType string

// CallSystem lifecycle events.
const (
	// EventCallInitiated is emitted when a call is started (outbound) or received (inbound).
	EventCallInitiated EventType = "call.initiated"

	// EventCallRinging is emitted when an outbound call starts ringing.
	EventCallRinging EventType = "call.ringing"

	// EventCallAnswered is emitted when a call is answered.
	EventCallAnswered EventType = "call.answered"

	// EventCallEnded is emitted when a call ends (normal hangup).
	EventCallEnded EventType = "call.ended"

	// EventCallFailed is emitted when a call fails.
	EventCallFailed EventType = "call.failed"

	// EventCallBusy is emitted when the line is busy.
	EventCallBusy EventType = "call.busy"

	// EventCallNoAnswer is emitted when there's no answer.
	EventCallNoAnswer EventType = "call.no_answer"
)

// Media streaming events.
const (
	// EventMediaConnected is emitted when media streaming connects.
	EventMediaConnected EventType = "media.connected"

	// EventMediaDisconnected is emitted when media streaming disconnects.
	EventMediaDisconnected EventType = "media.disconnected"

	// EventMediaError is emitted on media streaming errors.
	EventMediaError EventType = "media.error"
)

// DTMF events.
const (
	// EventDTMFReceived is emitted when DTMF tones are received.
	EventDTMFReceived EventType = "dtmf.received"
)

// VoiceEvent represents an observable event in the voice system.
type VoiceEvent struct {
	// Type identifies the event type.
	Type EventType

	// Timestamp is when the event occurred.
	Timestamp time.Time

	// CallID is the unique identifier for the call (if applicable).
	CallID string

	// Provider is the name of the provider (e.g., "twilio", "telnyx").
	Provider string

	// Direction is "inbound" or "outbound" for call events.
	Direction string

	// From is the caller ID.
	From string

	// To is the called number.
	To string

	// Duration is the call duration (for ended events).
	Duration time.Duration

	// Error contains error details (for failed/error events).
	Error error

	// Metadata holds additional provider-specific data.
	Metadata map[string]any
}

// VoiceObserver receives voice events for observability.
// Implementations should be non-blocking and handle errors internally.
type VoiceObserver interface {
	// OnEvent is called when a voice event occurs.
	// Implementations must be safe for concurrent use.
	OnEvent(ctx context.Context, event VoiceEvent)
}

// VoiceObserverFunc is a function adapter for VoiceObserver.
type VoiceObserverFunc func(ctx context.Context, event VoiceEvent)

// OnEvent implements VoiceObserver.
func (f VoiceObserverFunc) OnEvent(ctx context.Context, event VoiceEvent) {
	f(ctx, event)
}

// Observable is implemented by types that emit voice events.
type Observable interface {
	// AddObserver registers an observer to receive events.
	AddObserver(observer VoiceObserver)

	// RemoveObserver unregisters an observer.
	RemoveObserver(observer VoiceObserver)
}

// MultiObserver fans out events to multiple observers.
type MultiObserver struct {
	observers []VoiceObserver
}

// NewMultiObserver creates a MultiObserver with the given observers.
func NewMultiObserver(observers ...VoiceObserver) *MultiObserver {
	return &MultiObserver{observers: observers}
}

// Add adds an observer.
func (m *MultiObserver) Add(observer VoiceObserver) {
	m.observers = append(m.observers, observer)
}

// OnEvent dispatches the event to all observers.
func (m *MultiObserver) OnEvent(ctx context.Context, event VoiceEvent) {
	for _, obs := range m.observers {
		obs.OnEvent(ctx, event)
	}
}

// EmitEvent is a helper to emit events with common fields populated.
func EmitEvent(ctx context.Context, observer VoiceObserver, eventType EventType, callID, provider string, opts ...EventOption) {
	if observer == nil {
		return
	}

	event := VoiceEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		CallID:    callID,
		Provider:  provider,
		Metadata:  make(map[string]any),
	}

	for _, opt := range opts {
		opt(&event)
	}

	observer.OnEvent(ctx, event)
}

// EventOption configures a VoiceEvent.
type EventOption func(*VoiceEvent)

// WithDirection sets the call direction.
func WithDirection(dir string) EventOption {
	return func(e *VoiceEvent) {
		e.Direction = dir
	}
}

// WithFrom sets the caller ID.
func WithFrom(from string) EventOption {
	return func(e *VoiceEvent) {
		e.From = from
	}
}

// WithTo sets the called number.
func WithTo(to string) EventOption {
	return func(e *VoiceEvent) {
		e.To = to
	}
}

// WithDuration sets the call duration.
func WithDuration(d time.Duration) EventOption {
	return func(e *VoiceEvent) {
		e.Duration = d
	}
}

// WithError sets the error.
func WithError(err error) EventOption {
	return func(e *VoiceEvent) {
		e.Error = err
	}
}

// WithMetadata adds metadata to the event.
func WithMetadata(key string, value any) EventOption {
	return func(e *VoiceEvent) {
		if e.Metadata == nil {
			e.Metadata = make(map[string]any)
		}
		e.Metadata[key] = value
	}
}

// Package callsystem provides integrations with telephony and meeting platforms.
package callsystem

import (
	"context"
	"time"

	"github.com/plexusone/omnivoice-core/agent"
	"github.com/plexusone/omnivoice-core/observability"
	"github.com/plexusone/omnivoice-core/transport"
)

// CallDirection indicates inbound or outbound call.
type CallDirection string

const (
	// Inbound is an incoming call.
	Inbound CallDirection = "inbound"

	// Outbound is an outgoing call.
	Outbound CallDirection = "outbound"
)

// CallStatus represents the call state.
type CallStatus string

const (
	// StatusRinging indicates the call is ringing.
	StatusRinging CallStatus = "ringing"

	// StatusAnswered indicates the call is connected.
	StatusAnswered CallStatus = "answered"

	// StatusEnded indicates the call has ended.
	StatusEnded CallStatus = "ended"

	// StatusFailed indicates the call failed.
	StatusFailed CallStatus = "failed"

	// StatusBusy indicates the line was busy.
	StatusBusy CallStatus = "busy"

	// StatusNoAnswer indicates no answer.
	StatusNoAnswer CallStatus = "no_answer"
)

// Call represents a phone or video call.
type Call interface {
	// ID returns the call identifier.
	ID() string

	// Direction returns inbound or outbound.
	Direction() CallDirection

	// Status returns the current call status.
	Status() CallStatus

	// From returns the caller ID.
	From() string

	// To returns the called number.
	To() string

	// StartTime returns when the call started.
	StartTime() time.Time

	// Duration returns the call duration.
	Duration() time.Duration

	// Answer answers an inbound call.
	Answer(ctx context.Context) error

	// Hangup ends the call.
	Hangup(ctx context.Context) error

	// Transport returns the underlying transport connection.
	Transport() transport.Connection

	// AttachAgent attaches a voice agent to handle the call.
	AttachAgent(ctx context.Context, session agent.Session) error

	// DetachAgent detaches the voice agent.
	DetachAgent(ctx context.Context) error
}

// CallHandler is called when a new call arrives.
type CallHandler func(call Call) error

// CallSystemConfig configures a call system integration.
type CallSystemConfig struct {
	// AccountSID is the account identifier (Twilio, etc).
	AccountSID string

	// AuthToken is the authentication token.
	AuthToken string //nolint:gosec // G117: field intentionally stores credential

	// APIKey is an API key (alternative to AuthToken).
	APIKey string //nolint:gosec // G117: field intentionally stores credential

	// APISecret is the API secret.
	APISecret string //nolint:gosec // G117: field intentionally stores credential

	// WebhookURL is the URL for incoming webhooks.
	WebhookURL string

	// PhoneNumber is the default outbound caller ID.
	PhoneNumber string

	// Region is the service region.
	Region string

	// Observer receives voice events for observability.
	// If nil, no events are emitted.
	Observer observability.VoiceObserver
}

// CallSystem defines the interface for telephony/meeting integrations.
type CallSystem interface {
	// Name returns the call system name.
	Name() string

	// Configure configures the call system.
	Configure(config CallSystemConfig) error

	// OnIncomingCall sets the handler for incoming calls.
	OnIncomingCall(handler CallHandler)

	// MakeCall initiates an outbound call.
	MakeCall(ctx context.Context, to string, opts ...CallOption) (Call, error)

	// GetCall retrieves a call by ID.
	GetCall(ctx context.Context, callID string) (Call, error)

	// ListCalls lists active calls.
	ListCalls(ctx context.Context) ([]Call, error)

	// Close shuts down the call system.
	Close() error
}

// CallOption configures an outbound call.
type CallOption func(*CallOptions)

// CallOptions holds parsed options for MakeCall.
// Exported so provider implementations can access option values.
type CallOptions struct {
	From           string
	Timeout        time.Duration
	MachineDetect  bool
	Record         bool
	Whisper        string
	AgentConfig    *agent.Config
	StatusCallback string
	Observer       observability.VoiceObserver
}

// WithFrom sets the outbound caller ID.
func WithFrom(from string) CallOption {
	return func(o *CallOptions) {
		o.From = from
	}
}

// WithTimeout sets the call timeout.
func WithTimeout(timeout time.Duration) CallOption {
	return func(o *CallOptions) {
		o.Timeout = timeout
	}
}

// WithMachineDetection enables answering machine detection.
func WithMachineDetection() CallOption {
	return func(o *CallOptions) {
		o.MachineDetect = true
	}
}

// WithRecording enables call recording.
func WithRecording() CallOption {
	return func(o *CallOptions) {
		o.Record = true
	}
}

// WithWhisper sets a whisper message for the agent.
func WithWhisper(message string) CallOption {
	return func(o *CallOptions) {
		o.Whisper = message
	}
}

// WithAgent automatically attaches a voice agent to the call.
func WithAgent(config *agent.Config) CallOption {
	return func(o *CallOptions) {
		o.AgentConfig = config
	}
}

// WithStatusCallback sets a webhook URL for status updates.
func WithStatusCallback(url string) CallOption {
	return func(o *CallOptions) {
		o.StatusCallback = url
	}
}

// WithObserver sets a voice observer for call events.
func WithObserver(observer observability.VoiceObserver) CallOption {
	return func(o *CallOptions) {
		o.Observer = observer
	}
}

// ObservableCallSystem extends CallSystem with observability support.
// Providers that support observability should implement this interface.
type ObservableCallSystem interface {
	CallSystem
	observability.Observable
}

// MeetingSystem defines the interface for meeting platform integrations.
type MeetingSystem interface {
	// Name returns the meeting system name.
	Name() string

	// JoinMeeting joins an existing meeting.
	JoinMeeting(ctx context.Context, meetingID string, opts ...MeetingOption) (Meeting, error)

	// LeaveMeeting leaves a meeting.
	LeaveMeeting(ctx context.Context, meetingID string) error

	// ListMeetings lists active meetings.
	ListMeetings(ctx context.Context) ([]Meeting, error)
}

// Meeting represents a video/audio meeting.
type Meeting interface {
	// ID returns the meeting identifier.
	ID() string

	// Title returns the meeting title.
	Title() string

	// Participants returns current participants.
	Participants() []Participant

	// Transport returns the transport connection.
	Transport() transport.Connection

	// AttachAgent attaches a voice agent to the meeting.
	AttachAgent(ctx context.Context, session agent.Session) error

	// DetachAgent detaches the voice agent.
	DetachAgent(ctx context.Context) error

	// Leave leaves the meeting.
	Leave(ctx context.Context) error
}

// Participant represents a meeting participant.
type Participant struct {
	// ID is the participant identifier.
	ID string

	// Name is the participant display name.
	Name string

	// IsMuted indicates if audio is muted.
	IsMuted bool

	// IsBot indicates if this is a bot participant.
	IsBot bool
}

// MeetingOption configures meeting join behavior.
type MeetingOption func(*meetingOptions)

type meetingOptions struct {
	displayName string
	muted       bool
	agentConfig *agent.Config
}

// WithDisplayName sets the bot display name.
func WithDisplayName(name string) MeetingOption {
	return func(o *meetingOptions) {
		o.displayName = name
	}
}

// WithMuted joins with audio muted.
func WithMuted() MeetingOption {
	return func(o *meetingOptions) {
		o.muted = true
	}
}

// WithMeetingAgent attaches a voice agent to the meeting.
func WithMeetingAgent(config *agent.Config) MeetingOption {
	return func(o *meetingOptions) {
		o.agentConfig = config
	}
}

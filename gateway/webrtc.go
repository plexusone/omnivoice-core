package gateway

import (
	"context"
	"time"
)

// WebRTCGateway defines the interface for WebRTC-based voice gateways.
// Unlike PSTN gateways that handle phone calls, WebRTC gateways handle
// browser and mobile app connections via rooms and participants.
//
// Implementations:
//   - github.com/plexusone/omni-livekit/omnivoice/gateway
//
// Key differences from Gateway (PSTN):
//   - No phone numbers; uses room names and participant identities
//   - No webhooks; direct WebRTC signaling
//   - Agent joins a room and waits for participants (vs answering calls)
//   - Lower latency (<200ms vs 500ms+ for PSTN)
type WebRTCGateway interface {
	// Name returns the provider name.
	Name() ProviderName

	// Start connects to the room and starts handling participants.
	Start(ctx context.Context) error

	// Stop disconnects from the room and cleans up.
	Stop() error

	// OnParticipantJoined sets the handler for when participants join.
	// Return nil to accept the participant, or an error to ignore them.
	OnParticipantJoined(handler ParticipantHandler)

	// JoinRoom joins a specific room. Some implementations may join
	// automatically on Start() based on configuration.
	JoinRoom(ctx context.Context, roomName string) error

	// LeaveRoom leaves the current room.
	LeaveRoom() error

	// CurrentRoom returns the name of the currently joined room, or empty string.
	CurrentRoom() string

	// GetSession retrieves an active session by participant identity.
	GetSession(participantID string) (WebRTCSession, bool)

	// ListSessions returns all active sessions.
	ListSessions() []WebRTCSession

	// GenerateClientToken creates a JWT token for a client to join a room.
	GenerateClientToken(roomName, identity, displayName string) (string, error)
}

// ParticipantInfo contains information about a participant.
type ParticipantInfo struct {
	// Identity is the unique identifier for the participant.
	Identity string

	// DisplayName is the human-readable name.
	DisplayName string

	// RoomName is the room the participant joined.
	RoomName string

	// Metadata is optional JSON metadata attached to the participant.
	Metadata string

	// JoinedAt is when the participant joined.
	JoinedAt time.Time
}

// ParticipantHandler is called when a participant joins the room.
// Return nil to accept the participant, or an error to ignore them.
type ParticipantHandler func(participant *ParticipantInfo) error

// WebRTCSession represents an active voice conversation with a WebRTC participant.
// This is similar to Session but with WebRTC-specific semantics.
type WebRTCSession interface {
	// ID returns the session identifier (typically participant identity).
	ID() string

	// Participant returns information about the remote participant.
	Participant() *ParticipantInfo

	// RoomName returns the room this session is in.
	RoomName() string

	// AgentIdentity returns the identity of the AI agent.
	AgentIdentity() string

	// StartTime returns when the session started.
	StartTime() time.Time

	// Duration returns the session duration.
	Duration() time.Duration

	// Events returns a channel for session events.
	// Uses the same Event type as PSTN sessions.
	Events() <-chan Event

	// Transcript returns the conversation transcript.
	Transcript() []Turn

	// Metrics returns session performance metrics.
	Metrics() Metrics

	// SendText sends text input to the agent (bypasses STT).
	SendText(text string) error

	// SendAudio sends PCM16 audio samples to the participant.
	// This is the raw audio output method for WebRTC.
	SendAudio(samples []int16) error

	// Interrupt stops the current agent speech.
	Interrupt()

	// Close ends the session.
	Close() error
}

// RoomInfo contains information about a WebRTC room.
type RoomInfo struct {
	// Name is the room identifier.
	Name string

	// ParticipantCount is the number of participants in the room.
	ParticipantCount int

	// CreatedAt is when the room was created.
	CreatedAt time.Time

	// Metadata is optional JSON metadata attached to the room.
	Metadata string
}

// WebRTCConfig provides common configuration for WebRTC gateways.
type WebRTCConfig struct {
	// Server URL (e.g., "wss://your-app.livekit.cloud")
	ServerURL string

	// API credentials
	APIKey    string
	APISecret string

	// Room configuration
	RoomName      string
	AgentIdentity string
	AgentName     string

	// Audio configuration
	SampleRate int // 16000 or 24000 (default: 24000)
	Channels   int // 1 for mono (default: 1)

	// Voice pipeline configuration
	STTProvider string
	STTAPIKey   string
	STTModel    string
	STTLanguage string

	TTSProvider string
	TTSAPIKey   string
	TTSVoiceID  string
	TTSModel    string

	LLMProvider     string
	LLMAPIKey       string
	LLMModel        string
	LLMSystemPrompt string

	// Session configuration
	MaxSessionDuration time.Duration
	InterruptionMode   string // "immediate", "after_sentence", "disabled"
}

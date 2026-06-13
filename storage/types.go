package storage

import (
	"time"
)

// SessionStatus represents the current state of a voice session.
type SessionStatus string

const (
	// StatusPending indicates the session is waiting to be established.
	StatusPending SessionStatus = "pending"

	// StatusActive indicates the session is currently in progress.
	StatusActive SessionStatus = "active"

	// StatusOnHold indicates the session is on hold.
	StatusOnHold SessionStatus = "on_hold"

	// StatusTransferring indicates the session is being transferred.
	StatusTransferring SessionStatus = "transferring"

	// StatusEnded indicates the session has concluded.
	StatusEnded SessionStatus = "ended"
)

// Direction represents the direction of a call.
type Direction string

const (
	// DirectionInbound indicates an incoming call.
	DirectionInbound Direction = "inbound"

	// DirectionOutbound indicates an outgoing call.
	DirectionOutbound Direction = "outbound"
)

// SessionState represents the persistent state of a voice session.
type SessionState struct {
	// ID uniquely identifies this session.
	ID string `json:"id"`

	// CreatedAt is when the session was first created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the session was last modified.
	UpdatedAt time.Time `json:"updated_at"`

	// Call metadata
	CallID    string        `json:"call_id,omitempty"`
	Provider  string        `json:"provider"`
	Direction Direction     `json:"direction"`
	From      string        `json:"from"`
	To        string        `json:"to"`
	Status    SessionStatus `json:"status"`

	// Conversation history
	History []Turn `json:"history"`

	// Metrics for the session
	Metrics SessionMetrics `json:"metrics"`

	// RecoveryData holds provider-specific data needed to recover a session.
	RecoveryData map[string]any `json:"recovery_data,omitempty"`
}

// Turn represents a single conversational turn.
type Turn struct {
	// Role is the speaker: "user" or "agent".
	Role string `json:"role"`

	// Content is the transcript of what was said.
	Content string `json:"content"`

	// Timestamp is when this turn occurred.
	Timestamp time.Time `json:"timestamp"`

	// DurationMs is the speech duration in milliseconds.
	DurationMs int `json:"duration_ms"`
}

// SessionMetrics tracks statistics for a voice session.
type SessionMetrics struct {
	// TurnCount is the total number of conversational turns.
	TurnCount int `json:"turn_count"`

	// InterruptionCount is how many times the user interrupted the agent.
	InterruptionCount int `json:"interruption_count"`

	// UserSpeechDurationMs is total user speech time in milliseconds.
	UserSpeechDurationMs int `json:"user_speech_duration_ms"`

	// AgentSpeechDurationMs is total agent speech time in milliseconds.
	AgentSpeechDurationMs int `json:"agent_speech_duration_ms"`

	// FirstResponseMs is latency to first agent response in milliseconds.
	FirstResponseMs int `json:"first_response_ms,omitempty"`

	// AverageResponseMs is average agent response latency in milliseconds.
	AverageResponseMs int `json:"average_response_ms,omitempty"`
}

// NewSessionState creates a new session state with initialized fields.
func NewSessionState(id string) *SessionState {
	now := time.Now()
	return &SessionState{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		Status:    StatusPending,
		History:   make([]Turn, 0),
		Metrics:   SessionMetrics{},
	}
}

// AddTurn appends a conversational turn and updates metrics.
func (s *SessionState) AddTurn(role, content string, durationMs int) {
	s.History = append(s.History, Turn{
		Role:       role,
		Content:    content,
		Timestamp:  time.Now(),
		DurationMs: durationMs,
	})

	s.Metrics.TurnCount++
	if role == "user" {
		s.Metrics.UserSpeechDurationMs += durationMs
	} else if role == "agent" {
		s.Metrics.AgentSpeechDurationMs += durationMs
	}

	s.UpdatedAt = time.Now()
}

// RecordInterruption increments the interruption counter.
func (s *SessionState) RecordInterruption() {
	s.Metrics.InterruptionCount++
	s.UpdatedAt = time.Now()
}

// Duration returns the total session duration.
func (s *SessionState) Duration() time.Duration {
	if s.Status == StatusEnded {
		return s.UpdatedAt.Sub(s.CreatedAt)
	}
	return time.Since(s.CreatedAt)
}

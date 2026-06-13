package storage

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a session does not exist.
var ErrNotFound = errors.New("session not found")

// ErrStoreClosed is returned when operating on a closed store.
var ErrStoreClosed = errors.New("store is closed")

// SessionStore defines the interface for session persistence.
type SessionStore interface {
	// Save persists or updates a session state.
	// If the session already exists, it will be overwritten.
	Save(ctx context.Context, session *SessionState) error

	// Load retrieves a session state by ID.
	// Returns ErrNotFound if the session does not exist.
	Load(ctx context.Context, sessionID string) (*SessionState, error)

	// Delete removes a session state.
	// Returns ErrNotFound if the session does not exist.
	Delete(ctx context.Context, sessionID string) error

	// ListActive returns IDs of all active sessions.
	// A session is considered active if its status is not StatusEnded.
	ListActive(ctx context.Context) ([]string, error)

	// UpdateHeartbeat updates the session's timestamp to indicate activity.
	// This is used to track session liveness and enable cleanup of stale sessions.
	UpdateHeartbeat(ctx context.Context, sessionID string) error

	// Close releases any resources held by the store.
	Close() error
}

// Namer is an optional interface for stores that have a name.
type Namer interface {
	Name() string
}

// HealthChecker is an optional interface for stores that support health checks.
type HealthChecker interface {
	// Ping checks if the store is healthy and reachable.
	Ping(ctx context.Context) error
}

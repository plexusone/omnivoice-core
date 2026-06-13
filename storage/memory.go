package storage

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// Compile-time interface checks.
var (
	_ SessionStore  = (*MemoryStore)(nil)
	_ Namer         = (*MemoryStore)(nil)
	_ HealthChecker = (*MemoryStore)(nil)
)

// MemoryStore is an in-memory implementation of SessionStore.
// It is suitable for development, testing, and single-instance deployments.
// Note: Data is not persisted across restarts.
type MemoryStore struct {
	sessions map[string]*SessionState
	mu       sync.RWMutex
	closed   bool
}

// NewMemoryStore creates a new in-memory session store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]*SessionState),
	}
}

// Name returns the store implementation name.
func (m *MemoryStore) Name() string {
	return "memory"
}

// Ping checks if the store is healthy.
func (m *MemoryStore) Ping(_ context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return ErrStoreClosed
	}
	return nil
}

// Save persists or updates a session state.
func (m *MemoryStore) Save(_ context.Context, session *SessionState) error {
	if session == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStoreClosed
	}

	// Deep copy to avoid external mutations
	copied, err := deepCopy(session)
	if err != nil {
		return err
	}
	copied.UpdatedAt = time.Now()

	m.sessions[session.ID] = copied
	return nil
}

// Load retrieves a session state by ID.
func (m *MemoryStore) Load(_ context.Context, sessionID string) (*SessionState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrStoreClosed
	}

	session, ok := m.sessions[sessionID]
	if !ok {
		return nil, ErrNotFound
	}

	// Return a deep copy to prevent external mutations
	return deepCopy(session)
}

// Delete removes a session state.
func (m *MemoryStore) Delete(_ context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStoreClosed
	}

	if _, ok := m.sessions[sessionID]; !ok {
		return ErrNotFound
	}

	delete(m.sessions, sessionID)
	return nil
}

// ListActive returns IDs of all active sessions.
func (m *MemoryStore) ListActive(_ context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrStoreClosed
	}

	var ids []string
	for id, session := range m.sessions {
		if session.Status != StatusEnded {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// UpdateHeartbeat updates the session's timestamp.
func (m *MemoryStore) UpdateHeartbeat(_ context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStoreClosed
	}

	session, ok := m.sessions[sessionID]
	if !ok {
		return ErrNotFound
	}

	session.UpdatedAt = time.Now()
	return nil
}

// Close releases the store's resources.
func (m *MemoryStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true
	m.sessions = nil
	return nil
}

// Count returns the number of stored sessions (useful for testing).
func (m *MemoryStore) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// deepCopy creates a deep copy of a SessionState via JSON round-trip.
func deepCopy(src *SessionState) (*SessionState, error) {
	data, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	var dst SessionState
	if err := json.Unmarshal(data, &dst); err != nil {
		return nil, err
	}
	return &dst, nil
}

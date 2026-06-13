package storage

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStore(t *testing.T) {
	runStoreTests(t, NewMemoryStore())
}

// runStoreTests runs a suite of conformance tests for any SessionStore implementation.
func runStoreTests(t *testing.T, store SessionStore) {
	t.Helper()

	ctx := context.Background()

	t.Run("SaveAndLoad", func(t *testing.T) {
		session := NewSessionState("test-session-1")
		session.Provider = "twilio"
		session.From = "+15551234567"
		session.To = "+15559876543"
		session.Direction = DirectionInbound
		session.Status = StatusActive

		// Save
		if err := store.Save(ctx, session); err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		// Load
		loaded, err := store.Load(ctx, "test-session-1")
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		if loaded.ID != session.ID {
			t.Errorf("ID mismatch: got %q, want %q", loaded.ID, session.ID)
		}
		if loaded.Provider != session.Provider {
			t.Errorf("Provider mismatch: got %q, want %q", loaded.Provider, session.Provider)
		}
		if loaded.From != session.From {
			t.Errorf("From mismatch: got %q, want %q", loaded.From, session.From)
		}
		if loaded.Status != session.Status {
			t.Errorf("Status mismatch: got %q, want %q", loaded.Status, session.Status)
		}
	})

	t.Run("LoadNotFound", func(t *testing.T) {
		_, err := store.Load(ctx, "nonexistent-session")
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		session := NewSessionState("test-session-delete")
		session.Status = StatusActive

		if err := store.Save(ctx, session); err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		if err := store.Delete(ctx, "test-session-delete"); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, err := store.Load(ctx, "test-session-delete")
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound after delete, got %v", err)
		}
	})

	t.Run("DeleteNotFound", func(t *testing.T) {
		err := store.Delete(ctx, "nonexistent-session")
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("ListActive", func(t *testing.T) {
		// Create active session
		active := NewSessionState("test-session-active")
		active.Status = StatusActive
		if err := store.Save(ctx, active); err != nil {
			t.Fatalf("Save active session failed: %v", err)
		}

		// Create ended session
		ended := NewSessionState("test-session-ended")
		ended.Status = StatusEnded
		if err := store.Save(ctx, ended); err != nil {
			t.Fatalf("Save ended session failed: %v", err)
		}

		ids, err := store.ListActive(ctx)
		if err != nil {
			t.Fatalf("ListActive failed: %v", err)
		}

		// Should contain active, not ended
		foundActive := false
		foundEnded := false
		for _, id := range ids {
			if id == "test-session-active" {
				foundActive = true
			}
			if id == "test-session-ended" {
				foundEnded = true
			}
		}

		if !foundActive {
			t.Error("active session should be in ListActive result")
		}
		if foundEnded {
			t.Error("ended session should not be in ListActive result")
		}
	})

	t.Run("UpdateHeartbeat", func(t *testing.T) {
		session := NewSessionState("test-session-heartbeat")
		session.Status = StatusActive
		originalUpdate := session.UpdatedAt

		if err := store.Save(ctx, session); err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		// Wait a bit to ensure time difference
		time.Sleep(10 * time.Millisecond)

		if err := store.UpdateHeartbeat(ctx, "test-session-heartbeat"); err != nil {
			t.Fatalf("UpdateHeartbeat failed: %v", err)
		}

		loaded, err := store.Load(ctx, "test-session-heartbeat")
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		if !loaded.UpdatedAt.After(originalUpdate) {
			t.Error("UpdatedAt should be updated after heartbeat")
		}
	})

	t.Run("UpdateHeartbeatNotFound", func(t *testing.T) {
		err := store.UpdateHeartbeat(ctx, "nonexistent-heartbeat")
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("SaveNil", func(t *testing.T) {
		// Should not error on nil
		if err := store.Save(ctx, nil); err != nil {
			t.Errorf("Save(nil) should not error, got %v", err)
		}
	})

	t.Run("ConversationHistory", func(t *testing.T) {
		session := NewSessionState("test-session-history")
		session.Status = StatusActive

		session.AddTurn("user", "Hello, I need help", 1500)
		session.AddTurn("agent", "Hi there! How can I assist you today?", 2200)
		session.AddTurn("user", "What's my account balance?", 1800)

		if err := store.Save(ctx, session); err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		loaded, err := store.Load(ctx, "test-session-history")
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		if len(loaded.History) != 3 {
			t.Errorf("expected 3 turns, got %d", len(loaded.History))
		}

		if loaded.Metrics.TurnCount != 3 {
			t.Errorf("expected turn count 3, got %d", loaded.Metrics.TurnCount)
		}

		if loaded.Metrics.UserSpeechDurationMs != 3300 { // 1500 + 1800
			t.Errorf("expected user speech 3300ms, got %d", loaded.Metrics.UserSpeechDurationMs)
		}

		if loaded.Metrics.AgentSpeechDurationMs != 2200 {
			t.Errorf("expected agent speech 2200ms, got %d", loaded.Metrics.AgentSpeechDurationMs)
		}
	})

	t.Run("RecoveryData", func(t *testing.T) {
		session := NewSessionState("test-session-recovery")
		session.Status = StatusActive
		session.RecoveryData = map[string]any{
			"call_sid":   "CA123456",
			"stream_sid": "MZ789012",
			"sequence":   42,
		}

		if err := store.Save(ctx, session); err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		loaded, err := store.Load(ctx, "test-session-recovery")
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		if loaded.RecoveryData == nil {
			t.Fatal("RecoveryData should not be nil")
		}

		if loaded.RecoveryData["call_sid"] != "CA123456" {
			t.Errorf("call_sid mismatch: got %v", loaded.RecoveryData["call_sid"])
		}
	})

	// Clean up
	if err := store.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestSessionState_AddTurn(t *testing.T) {
	s := NewSessionState("test")

	s.AddTurn("user", "Hello", 1000)
	s.AddTurn("agent", "Hi there!", 1500)

	if len(s.History) != 2 {
		t.Errorf("expected 2 turns, got %d", len(s.History))
	}

	if s.Metrics.TurnCount != 2 {
		t.Errorf("expected turn count 2, got %d", s.Metrics.TurnCount)
	}

	if s.Metrics.UserSpeechDurationMs != 1000 {
		t.Errorf("expected user speech 1000, got %d", s.Metrics.UserSpeechDurationMs)
	}

	if s.Metrics.AgentSpeechDurationMs != 1500 {
		t.Errorf("expected agent speech 1500, got %d", s.Metrics.AgentSpeechDurationMs)
	}
}

func TestSessionState_RecordInterruption(t *testing.T) {
	s := NewSessionState("test")

	s.RecordInterruption()
	s.RecordInterruption()

	if s.Metrics.InterruptionCount != 2 {
		t.Errorf("expected 2 interruptions, got %d", s.Metrics.InterruptionCount)
	}
}

func TestSessionState_Duration(t *testing.T) {
	s := NewSessionState("test")
	s.Status = StatusActive

	// Active session duration should be time since creation
	time.Sleep(10 * time.Millisecond)
	d := s.Duration()
	if d < 10*time.Millisecond {
		t.Errorf("expected duration >= 10ms for active session, got %v", d)
	}

	// Ended session duration should be fixed
	s.Status = StatusEnded
	s.UpdatedAt = s.CreatedAt.Add(5 * time.Second)
	d = s.Duration()
	if d != 5*time.Second {
		t.Errorf("expected duration 5s for ended session, got %v", d)
	}
}

func TestMemoryStore_Count(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	if store.Count() != 0 {
		t.Errorf("expected count 0, got %d", store.Count())
	}

	_ = store.Save(ctx, NewSessionState("a"))
	_ = store.Save(ctx, NewSessionState("b"))

	if store.Count() != 2 {
		t.Errorf("expected count 2, got %d", store.Count())
	}

	_ = store.Delete(ctx, "a")

	if store.Count() != 1 {
		t.Errorf("expected count 1, got %d", store.Count())
	}

	store.Close()
}

func TestMemoryStore_ClosedOperations(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	store.Close()

	if err := store.Save(ctx, NewSessionState("test")); err != ErrStoreClosed {
		t.Errorf("expected ErrStoreClosed on Save, got %v", err)
	}

	if _, err := store.Load(ctx, "test"); err != ErrStoreClosed {
		t.Errorf("expected ErrStoreClosed on Load, got %v", err)
	}

	if err := store.Delete(ctx, "test"); err != ErrStoreClosed {
		t.Errorf("expected ErrStoreClosed on Delete, got %v", err)
	}

	if _, err := store.ListActive(ctx); err != ErrStoreClosed {
		t.Errorf("expected ErrStoreClosed on ListActive, got %v", err)
	}

	if err := store.UpdateHeartbeat(ctx, "test"); err != ErrStoreClosed {
		t.Errorf("expected ErrStoreClosed on UpdateHeartbeat, got %v", err)
	}

	if namer, ok := SessionStore(store).(Namer); ok {
		if namer.Name() != "memory" {
			t.Errorf("expected name 'memory', got %q", namer.Name())
		}
	}
}

func TestMemoryStore_IsolatedCopies(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	session := NewSessionState("test-isolation")
	session.Status = StatusActive
	session.From = "original"

	_ = store.Save(ctx, session)

	// Modify original
	session.From = "modified"

	// Load should return original value
	loaded, _ := store.Load(ctx, "test-isolation")
	if loaded.From != "original" {
		t.Error("store should return isolated copies")
	}

	// Modify loaded
	loaded.From = "also-modified"

	// Load again should still return original
	loaded2, _ := store.Load(ctx, "test-isolation")
	if loaded2.From != "original" {
		t.Error("store should return isolated copies on each load")
	}

	store.Close()
}

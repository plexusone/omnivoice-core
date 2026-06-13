// Package storage provides session state persistence for voice calls.
//
// The package defines a SessionStore interface for storing and retrieving
// session state, enabling features like call recovery and session resumption.
//
// # Available Implementations
//
//   - MemoryStore: In-memory storage for development and testing
//   - RedisStore: Redis-backed storage for production deployments
//
// # Usage
//
//	store := storage.NewMemoryStore()
//
//	// Save session state
//	state := &storage.SessionState{
//	    ID:       "session-123",
//	    Provider: "twilio",
//	    From:     "+15551234567",
//	    To:       "+15559876543",
//	    Status:   storage.StatusActive,
//	}
//	err := store.Save(ctx, state)
//
//	// Load session state
//	loaded, err := store.Load(ctx, "session-123")
//
// # Redis Configuration
//
// For production deployments, use RedisStore with connection pooling:
//
//	store, err := storage.NewRedisStore("redis://localhost:6379",
//	    storage.WithPrefix("voicecall:"),
//	    storage.WithTTL(24 * time.Hour),
//	)
package storage

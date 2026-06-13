package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Compile-time interface checks.
var (
	_ SessionStore  = (*RedisStore)(nil)
	_ Namer         = (*RedisStore)(nil)
	_ HealthChecker = (*RedisStore)(nil)
)

// RedisStore is a Redis-backed implementation of SessionStore.
// It is suitable for production deployments requiring persistence
// and multi-instance coordination.
type RedisStore struct {
	client       *redis.Client
	prefix       string
	ttl          time.Duration
	closed       bool
	activeSetKey string
}

// RedisOption configures a RedisStore.
type RedisOption func(*RedisStore)

// WithPrefix sets the key prefix for all session keys.
func WithPrefix(prefix string) RedisOption {
	return func(r *RedisStore) {
		r.prefix = prefix
	}
}

// WithTTL sets the time-to-live for session keys.
func WithTTL(ttl time.Duration) RedisOption {
	return func(r *RedisStore) {
		r.ttl = ttl
	}
}

// NewRedisStore creates a new Redis-backed session store.
// The addr should be in the form "redis://host:port" or "host:port".
func NewRedisStore(addr string, opts ...RedisOption) (*RedisStore, error) {
	// Parse Redis URL or address
	opt, err := parseRedisAddr(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid redis address: %w", err)
	}

	client := redis.NewClient(opt)

	store := &RedisStore{
		client: client,
		prefix: "voicesession:",
		ttl:    24 * time.Hour,
	}

	for _, o := range opts {
		o(store)
	}

	store.activeSetKey = store.prefix + "active"

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return store, nil
}

// NewRedisStoreFromClient creates a RedisStore from an existing client.
func NewRedisStoreFromClient(client *redis.Client, opts ...RedisOption) *RedisStore {
	store := &RedisStore{
		client: client,
		prefix: "voicesession:",
		ttl:    24 * time.Hour,
	}

	for _, o := range opts {
		o(store)
	}

	store.activeSetKey = store.prefix + "active"
	return store
}

// Name returns the store implementation name.
func (r *RedisStore) Name() string {
	return "redis"
}

// Ping checks if the store is healthy.
func (r *RedisStore) Ping(ctx context.Context) error {
	if r.closed {
		return ErrStoreClosed
	}
	return r.client.Ping(ctx).Err()
}

// Save persists or updates a session state.
func (r *RedisStore) Save(ctx context.Context, session *SessionState) error {
	if session == nil {
		return nil
	}
	if r.closed {
		return ErrStoreClosed
	}

	session.UpdatedAt = time.Now()

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := r.sessionKey(session.ID)

	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()
	pipe.Set(ctx, key, data, r.ttl)

	// Track in active set if not ended
	if session.Status != StatusEnded {
		pipe.SAdd(ctx, r.activeSetKey, session.ID)
	} else {
		pipe.SRem(ctx, r.activeSetKey, session.ID)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// Load retrieves a session state by ID.
func (r *RedisStore) Load(ctx context.Context, sessionID string) (*SessionState, error) {
	if r.closed {
		return nil, ErrStoreClosed
	}

	key := r.sessionKey(sessionID)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	var session SessionState
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// Delete removes a session state.
func (r *RedisStore) Delete(ctx context.Context, sessionID string) error {
	if r.closed {
		return ErrStoreClosed
	}

	key := r.sessionKey(sessionID)

	// Check if exists
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to check session existence: %w", err)
	}
	if exists == 0 {
		return ErrNotFound
	}

	// Delete key and remove from active set
	pipe := r.client.Pipeline()
	pipe.Del(ctx, key)
	pipe.SRem(ctx, r.activeSetKey, sessionID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// ListActive returns IDs of all active sessions.
func (r *RedisStore) ListActive(ctx context.Context) ([]string, error) {
	if r.closed {
		return nil, ErrStoreClosed
	}

	ids, err := r.client.SMembers(ctx, r.activeSetKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list active sessions: %w", err)
	}

	return ids, nil
}

// UpdateHeartbeat updates the session's timestamp.
func (r *RedisStore) UpdateHeartbeat(ctx context.Context, sessionID string) error {
	if r.closed {
		return ErrStoreClosed
	}

	// Load, update, save
	session, err := r.Load(ctx, sessionID)
	if err != nil {
		return err
	}

	session.UpdatedAt = time.Now()
	return r.Save(ctx, session)
}

// Close releases the store's resources.
func (r *RedisStore) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	return r.client.Close()
}

// sessionKey returns the Redis key for a session ID.
func (r *RedisStore) sessionKey(sessionID string) string {
	return r.prefix + sessionID
}

// parseRedisAddr parses a Redis address or URL into options.
func parseRedisAddr(addr string) (*redis.Options, error) {
	// Handle redis:// URLs
	if strings.HasPrefix(addr, "redis://") || strings.HasPrefix(addr, "rediss://") {
		return redis.ParseURL(addr)
	}

	// Plain host:port
	return &redis.Options{
		Addr: addr,
	}, nil
}

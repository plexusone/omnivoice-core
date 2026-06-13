# Session Storage

The `storage` package provides session state persistence for voice calls, enabling recovery after restarts and metrics tracking across conversations.

## Overview

Session storage is critical for:

- **Call recovery** - Resume interrupted calls after server restart
- **Conversation history** - Track turns, transcripts, and durations
- **Metrics collection** - Monitor turn counts, interruptions, speech durations
- **Multi-instance coordination** - Share session state across server instances

## Interfaces

### SessionStore

The core interface for session persistence:

```go
type SessionStore interface {
    Save(ctx context.Context, session *SessionState) error
    Load(ctx context.Context, sessionID string) (*SessionState, error)
    Delete(ctx context.Context, sessionID string) error
    ListActive(ctx context.Context) ([]string, error)
    UpdateHeartbeat(ctx context.Context, sessionID string) error
    Close() error
}
```

### SessionState

The session state type tracks call metadata and conversation history:

```go
type SessionState struct {
    ID           string        `json:"id"`
    CreatedAt    time.Time     `json:"created_at"`
    UpdatedAt    time.Time     `json:"updated_at"`

    // Call metadata
    CallID       string        `json:"call_id,omitempty"`
    Provider     string        `json:"provider"`
    Direction    string        `json:"direction"`
    From         string        `json:"from"`
    To           string        `json:"to"`
    Status       SessionStatus `json:"status"`

    // Conversation
    History      []Turn        `json:"history"`
    Metrics      SessionMetrics `json:"metrics"`

    // Provider-specific recovery data
    RecoveryData map[string]any `json:"recovery_data,omitempty"`
}
```

## Implementations

### MemoryStore (Default)

In-memory storage for single-instance deployments:

```go
import "github.com/plexusone/omnivoice-core/storage"

store := storage.NewMemoryStore()
defer store.Close()

session := storage.NewSessionState("call-123")
session.Provider = "twilio"
session.From = "+15551234567"
session.Status = storage.StatusActive

if err := store.Save(ctx, session); err != nil {
    log.Fatal(err)
}
```

### RedisStore

Redis-backed storage for multi-instance deployments:

```go
import "github.com/plexusone/omnivoice-core/storage"

store, err := storage.NewRedisStore("localhost:6379",
    storage.WithPrefix("voice:sessions:"),
    storage.WithTTL(24 * time.Hour),
)
if err != nil {
    log.Fatal(err)
}
defer store.Close()
```

## Conversation History

Track conversation turns with speech duration metrics:

```go
session := storage.NewSessionState("call-123")

// Add conversation turns
session.AddTurn("user", "Hello, I need help with my account", 2500)
session.AddTurn("agent", "I'd be happy to help. What's your account number?", 3200)
session.AddTurn("user", "It's 12345", 1000)

// Metrics are automatically updated
fmt.Printf("Turns: %d\n", session.Metrics.TurnCount)
fmt.Printf("User speech: %dms\n", session.Metrics.UserSpeechDurationMs)
fmt.Printf("Agent speech: %dms\n", session.Metrics.AgentSpeechDurationMs)
```

## Session Recovery

Store provider-specific data for call recovery:

```go
session := storage.NewSessionState("call-123")
session.RecoveryData = map[string]any{
    "call_sid":    "CA123456",
    "stream_sid":  "MZ789012",
    "sequence":    42,
}

if err := store.Save(ctx, session); err != nil {
    log.Fatal(err)
}

// After restart, recover sessions
ids, _ := store.ListActive(ctx)
for _, id := range ids {
    session, err := store.Load(ctx, id)
    if err != nil {
        continue
    }
    // Reconnect using recovery data
    callSID := session.RecoveryData["call_sid"].(string)
}
```

## Session Lifecycle

```
StatusConnecting → StatusActive → StatusEnded
                        ↓
                  StatusFailed
```

Sessions move through states:

1. **Connecting** - Call is being established
2. **Active** - Call is in progress
3. **Ended** - Call completed normally
4. **Failed** - Call failed due to error

## Best Practices

1. **Use Redis for production** - MemoryStore is for development/testing only
2. **Set appropriate TTL** - Clean up old sessions automatically
3. **Update heartbeats** - Call `UpdateHeartbeat` periodically for active sessions
4. **Store recovery data** - Save provider-specific data needed to reconnect

package v2

import (
	"context"
	"encoding/json"
	"time"

	"drayya-core/internal/database"
)

// Memory is the Redis-backed conversation store. One key per (caller,
// session_id); value is a JSON-encoded ring buffer of the last
// MaxMemoryTurns messages. TTL refreshes on every write so active sessions
// stay alive, idle ones evict automatically.
//
// We keep a small fixed window (10 turns by default) rather than persisting
// the full history because:
//   - LLM input cost scales linearly with history size
//   - older context drifts from the current intent
//   - PII bloat — most messages are uninteresting after 1h
//
// Persistent records (symptom logs, bookings, audit) go to Postgres via
// individual tool implementations, not here.
type Memory struct {
	maxTurns int
	ttl      time.Duration
}

func NewMemory(maxTurns int, ttl time.Duration) *Memory {
	if maxTurns <= 0 {
		maxTurns = MaxMemoryTurns
	}
	if ttl <= 0 {
		ttl = ConversationTTL
	}
	return &Memory{maxTurns: maxTurns, ttl: ttl}
}

// Load returns the conversation history for this session. Returns an empty
// slice (never nil) on cache miss or Redis unavailable — the agent should
// always be able to make progress even when memory is gone.
func (m *Memory) Load(ctx context.Context, persona Persona, sessionID string) []Message {
	if database.Rdb == nil {
		return []Message{}
	}
	raw, err := database.Rdb.Get(ctx, persona.SessionKey(sessionID)).Result()
	if err != nil || raw == "" {
		return []Message{}
	}
	var msgs []Message
	if err := json.Unmarshal([]byte(raw), &msgs); err != nil {
		return []Message{}
	}
	return msgs
}

// Append adds a turn and trims to the most recent maxTurns. TTL is reset on
// every write so an active conversation stays alive indefinitely.
func (m *Memory) Append(ctx context.Context, persona Persona, sessionID string, msg Message) {
	if database.Rdb == nil {
		return
	}
	history := m.Load(ctx, persona, sessionID)
	if msg.TimestampMS == 0 {
		msg.TimestampMS = time.Now().UnixMilli()
	}
	history = append(history, msg)
	if len(history) > m.maxTurns {
		history = history[len(history)-m.maxTurns:]
	}
	data, err := json.Marshal(history)
	if err != nil {
		return
	}
	_ = database.Rdb.Set(ctx, persona.SessionKey(sessionID), data, m.ttl).Err()
}

// Clear wipes the session — called on explicit "start over" or when the
// safety layer ends a conversation (e.g. emergency routed → user reaches
// SOS, anything they say next is a fresh conversation).
func (m *Memory) Clear(ctx context.Context, persona Persona, sessionID string) {
	if database.Rdb == nil {
		return
	}
	_ = database.Rdb.Del(ctx, persona.SessionKey(sessionID)).Err()
}

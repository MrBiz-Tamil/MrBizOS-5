package v2

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"drayya-core/internal/database"
)

// Confirmation gate runtime.
//
// Destructive tools (Confirm=true) cannot execute on first request. Instead
// they return a structured "pending confirmation" result and the runtime
// persists the proposed action in Redis with a 60s TTL. The LLM, seeing the
// pending result, asks the user to confirm in plain language.
//
// On the next turn the agent checks BEFORE the LLM call whether:
//  1. There's a pending action for this session
//  2. The user's message looks like a YES/NO confirmation
//
// If both: skip LLM, execute (or cancel) the pending action, feed the
// result to the LLM as a function result for the next turn.
//
// This design has two important properties:
//   - The LLM never sees raw "do you confirm?" tokens that could be
//     manipulated; the runtime decides what counts as confirmation.
//   - If the user pivots ("actually no, find a different doctor"), the
//     LLM handles it naturally because the pending action is dropped
//     after TTL or explicit cancel.

const PendingTTL = 60 * time.Second

// PendingAction is what we serialise to Redis between turns.
type PendingAction struct {
	ToolName  string         `json:"tool"`
	Args      map[string]any `json:"args"`
	Summary   string         `json:"summary"`  // human-readable, fed back to user
	CreatedAt int64          `json:"created_at"`
}

// pendingKey returns the Redis key holding any pending action for this session.
// Keyed by session, not user — confirmations are session-local.
func pendingKey(persona Persona, sessionID string) string {
	if persona.UserID != nil {
		return "devi:v2:pending:user:" + uintStr(*persona.UserID) + ":" + sessionID
	}
	return "devi:v2:pending:guest:" + persona.GuestBrowserID + ":" + sessionID
}

// savePending stores a pending action with TTL. Best-effort: if Redis is
// down we return nothing and the runtime will treat the lack of a pending
// record as "nothing to confirm" on the next turn, which is the safe
// failure mode.
func savePending(ctx context.Context, persona Persona, sessionID string, act PendingAction) {
	if database.Rdb == nil {
		return
	}
	act.CreatedAt = time.Now().Unix()
	data, _ := json.Marshal(act)
	_ = database.Rdb.Set(ctx, pendingKey(persona, sessionID), data, PendingTTL).Err()
}

// loadPending returns the pending action (if any) for this session. Returns
// nil when there's nothing waiting, Redis is down, or the entry expired.
func loadPending(ctx context.Context, persona Persona, sessionID string) *PendingAction {
	if database.Rdb == nil {
		return nil
	}
	raw, err := database.Rdb.Get(ctx, pendingKey(persona, sessionID)).Result()
	if err != nil || raw == "" {
		return nil
	}
	var act PendingAction
	if err := json.Unmarshal([]byte(raw), &act); err != nil {
		return nil
	}
	return &act
}

// clearPending removes the pending action — called after execution OR cancel.
func clearPending(ctx context.Context, persona Persona, sessionID string) {
	if database.Rdb == nil {
		return
	}
	_ = database.Rdb.Del(ctx, pendingKey(persona, sessionID)).Err()
}

// ─── User-message confirmation detection ──────────────────────────────────

// Matches anything that looks like a clear YES from the user. We're
// intentionally narrow — "yes please", "okay go ahead", "confirm", "do it".
// Anything ambiguous ("maybe", "sounds good but…") is treated as NOT a
// confirmation; the LLM has to handle it as normal conversation.
var confirmYesPattern = regexp.MustCompile(
	`(?i)^\s*(yes|y|yeah|yep|yup|ok|okay|sure|confirm(ed)?|do it|go ahead|please do|proceed|go for it)\s*[.!]*\s*$`,
)

// Matches a clear NO. Similarly narrow.
var confirmNoPattern = regexp.MustCompile(
	`(?i)^\s*(no|nope|cancel|stop|don.?t|skip|never mind|nevermind)\s*[.!]*\s*$`,
)

// ConfirmIntent is the agent's interpretation of the user message when a
// pending action exists.
type ConfirmIntent int

const (
	ConfirmIntentNone   ConfirmIntent = iota // not a confirmation reply; resume LLM
	ConfirmIntentYes                          // execute the pending action
	ConfirmIntentNo                           // cancel it
)

// DetectConfirm reads a user message and decides whether it's a yes/no/none.
// Whitespace and trailing punctuation are forgiven.
func DetectConfirm(msg string) ConfirmIntent {
	trimmed := strings.TrimSpace(msg)
	if confirmYesPattern.MatchString(trimmed) {
		return ConfirmIntentYes
	}
	if confirmNoPattern.MatchString(trimmed) {
		return ConfirmIntentNo
	}
	return ConfirmIntentNone
}

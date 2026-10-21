// Package v2 is the Devi V2 agent runtime.
//
// V2 is an LLM-first conversational assistant for patients. Where V1 used a
// rule-based intent router with the LLM as a 5% fallback, V2 inverts that:
// every message goes through Gemini 2.0 Flash with a tool catalog, and the
// LLM decides whether to reply directly or call one or more tools.
//
// Three layers (in order):
//
//  1. Safety rails — rule-based emergency/crisis detection. Runs first,
//     bypasses LLM if matched. See safety.go.
//
//  2. Limits — sliding-window rate limits + input sanitization. Run after
//     safety; reject hostile or excessive requests before paying for the
//     LLM call. See limits.go.
//
//  3. Agent loop — system prompt + conversation memory + tool schema →
//     Gemini call → execute requested tools → loop until no more calls →
//     return final reply. See agent.go.
//
// V1 (in the parent package) continues running unchanged. V2 routes are
// mounted at /devi/v2/* so traffic can be split during the A/B rollout.
package v2

import (
	"context"
	"encoding/json"
)

// ─── Persona ──────────────────────────────────────────────────────────────

// PersonaRole is who the LLM is talking to. The agent runtime resolves this
// once per request from the JWT/session and uses it to:
//   - filter the tool catalog visible to the LLM
//   - pick the right system prompt module
//   - apply per-tier rate limits
type PersonaRole string

const (
	RoleGuest    PersonaRole = "guest"    // no account; soft-gate writes
	RolePatient  PersonaRole = "patient"  // full patient toolset
	RoleDoctor   PersonaRole = "doctor"   // Devi defers; no tools
	RoleAdmin    PersonaRole = "admin"    // Devi defers; no tools
	RoleStaff    PersonaRole = "staff"    // other staff roles; Devi defers
)

// Persona is the resolved identity of the caller for one request.
type Persona struct {
	Role           PersonaRole
	UserID         *uint   // nil for guests
	GuestBrowserID string  // empty for authenticated users
	FamilyMemberID *uint   // set when user is "in family context"
	Tier           string  // "free" | "premium" | "" (guest)
	Language       string  // "en" | "ta" | "ml" | "hi" | "" (auto-detect)
	IsNew          bool    // patient created < 24h ago — triggers onboarding nudges
	HasProfile     bool    // PatientProfile present and non-empty
}

// SessionKey returns the Redis namespace key for this caller's session state.
// Authenticated users share state across devices via user_id; guests get a
// per-browser key.
func (p Persona) SessionKey(sessionID string) string {
	if p.UserID != nil {
		return "devi:v2:session:user:" + uintStr(*p.UserID) + ":" + sessionID
	}
	if p.GuestBrowserID != "" {
		return "devi:v2:session:guest:" + p.GuestBrowserID + ":" + sessionID
	}
	return "devi:v2:session:anon:" + sessionID
}

// LimitKey returns the Redis key family used for per-caller rate counters.
// Same key shape regardless of which window (minute/hour/day) the counter
// belongs to — the window suffix is appended by the limiter.
func (p Persona) LimitKey() string {
	if p.UserID != nil {
		return "devi:v2:limit:user:" + uintStr(*p.UserID)
	}
	if p.GuestBrowserID != "" {
		return "devi:v2:limit:guest:" + p.GuestBrowserID
	}
	return "devi:v2:limit:anon"
}

// IsDeferred returns true when Devi should politely step aside instead of
// engaging. Doctors and staff have their own surfaces; Devi exists for
// patients and guests.
func (p Persona) IsDeferred() bool {
	return p.Role == RoleDoctor || p.Role == RoleAdmin || p.Role == RoleStaff
}

// ─── Conversation messages ────────────────────────────────────────────────

// MessageRole maps to Gemini's role taxonomy. We mirror Gemini's wire format
// so persistence ↔ LLM call is a no-op transform.
type MessageRole string

const (
	RoleUser     MessageRole = "user"
	RoleModel    MessageRole = "model"
	RoleFunction MessageRole = "function" // tool result
)

// Message is one turn of conversation. Stored in Redis (last 10), sent to
// Gemini on every call.
type Message struct {
	Role        MessageRole     `json:"role"`
	Content     string          `json:"content,omitempty"`     // user text or model reply
	ToolName    string          `json:"tool_name,omitempty"`   // set when Role=function
	ToolResult  json.RawMessage `json:"tool_result,omitempty"` // tool output
	TimestampMS int64           `json:"ts"`
}

// ─── Tools ────────────────────────────────────────────────────────────────

// Tool is a capability the LLM can request. Every Tool maps to a Go function
// that reads or writes a narrowly-scoped slice of platform data. The runtime
// is responsible for:
//
//   - Filtering the tool catalog by Persona.Role before showing it to the LLM
//     (tool.Allowed[role] must be true)
//   - Enforcing the per-tool rate counters (MaxPerSession, MaxPerMinute)
//   - Gating destructive tools behind a confirmation handshake (Confirm=true)
//   - Recording usage to limits + audit
//
// Tools NEVER trust the LLM. The runtime resolves the caller identity from
// the JWT and injects user_id into tool args itself. The LLM cannot pass an
// arbitrary user_id to read someone else's data.
type Tool struct {
	// Name is what the LLM sees and what it sends back to request a call.
	Name string

	// Description tells the LLM when to use this tool. Keep crisp — Gemini
	// reads every description on every call. Avoid examples here; they bloat
	// the prompt. Examples belong in the system prompt.
	Description string

	// Parameters is the JSON schema the LLM must produce to call this tool.
	// Use object {type, properties{name:{type,description}}, required:[...]}.
	Parameters map[string]any

	// Allowed lists the personas that may invoke this tool. The runtime
	// filters by this set BEFORE serializing the tool to Gemini, so the LLM
	// never sees tools it can't call.
	Allowed []PersonaRole

	// Confirm = true means the tool is destructive: the runtime will not
	// execute it on the first request. Instead it asks the user to confirm
	// (text-based "reply YES to confirm") and stores a pending action in
	// Redis for 60 seconds.
	Confirm bool

	// MaxPerSession caps how many times this tool may run in one session.
	// 0 = unlimited (don't use 0 for write tools).
	MaxPerSession int

	// MaxPerMinute is a sliding-window per-minute cap.
	MaxPerMinute int

	// Execute is invoked by the runtime after all guards pass. ctx carries
	// the persona via persona.FromContext; args is the LLM-supplied JSON
	// blob already unmarshalled into the schema. Return value is JSON-
	// marshalled and fed back to the LLM as a function result.
	Execute func(ctx context.Context, args map[string]any) (any, error)
}

// AllowedFor reports whether the given role may invoke this tool.
func (t Tool) AllowedFor(role PersonaRole) bool {
	for _, r := range t.Allowed {
		if r == role {
			return true
		}
	}
	return false
}

// ─── Registry ────────────────────────────────────────────────────────────

// Registry is the canonical catalog of tools available to the agent. Build
// it once at process startup via NewRegistry() then mutate only by calls to
// Register. The runtime reads it on every request — keep lookups O(1).
type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool. Panics on duplicate name — that's a programmer
// error, not a runtime condition.
func (r *Registry) Register(t Tool) {
	if _, dup := r.tools[t.Name]; dup {
		panic("devi v2: duplicate tool registration: " + t.Name)
	}
	r.tools[t.Name] = t
}

// Get returns the tool by name, plus whether it exists. Cheap to call.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// ForPersona returns the subset of tools this persona may call. Sorted by
// name so the prompt is deterministic (helps cache hits).
func (r *Registry) ForPersona(p Persona) []Tool {
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		if t.AllowedFor(p.Role) {
			out = append(out, t)
		}
	}
	// Stable order — sort by name. Avoids cache misses from map iteration.
	sortToolsByName(out)
	return out
}

// All returns every registered tool. Used by admin endpoints, not the LLM.
func (r *Registry) All() []Tool {
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	sortToolsByName(out)
	return out
}

// ─── Agent request / response envelopes ──────────────────────────────────

// ChatRequest is the wire shape accepted by POST /devi/v2/chat.
type ChatRequest struct {
	Message        string `json:"message"`
	SessionID      string `json:"session_id"`
	GuestBrowserID string `json:"guest_browser_id,omitempty"`
}

// ChatResponse is what the agent returns. The Reply is what the user sees;
// the other fields are debugging affordances surfaced to authenticated
// admins via a `?debug=1` flag (not implemented in Phase 0).
type ChatResponse struct {
	Reply       string   `json:"reply"`
	SessionID   string   `json:"session_id"`
	Done        bool     `json:"done"`         // Conversation closed (e.g. emergency routed)
	ToolsCalled []string `json:"tools_called,omitempty"`
	LimitedBy   string   `json:"limited_by,omitempty"` // "rate" | "cost" | "size" | "" if not limited
	LatencyMS   int      `json:"latency_ms,omitempty"`
}

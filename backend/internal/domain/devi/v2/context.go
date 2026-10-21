package v2

import "context"

// This file retains the request-scoped context helpers for the AI agent. The
// Fiber-coupled request plumbing that used to live alongside them was removed
// with the REST server; the persona/session context values below are pure Go
// context and are threaded by the agent into each tool invocation.

type personaCtxKey struct{}

// WithPersona returns a derived context carrying the resolved Persona.
// Called by agent.executeTool before invoking the tool.
func WithPersona(ctx context.Context, p Persona) context.Context {
	return context.WithValue(ctx, personaCtxKey{}, p)
}

// PersonaFromContext returns the persona stored on the context, or a
// zero-value Persona if none was set. Tools should call this once and
// branch on Role / UserID rather than asking the LLM for identity.
func PersonaFromContext(ctx context.Context) Persona {
	if p, ok := ctx.Value(personaCtxKey{}).(Persona); ok {
		return p
	}
	return Persona{}
}

// ─── Session-id-in-context ────────────────────────────────────────────────
// Some tools (switch_family_context, confirmation gates) need to mutate
// the session itself. The agent threads the session ID into context too.

type sessionIDCtxKey struct{}

func WithSessionID(ctx context.Context, sid string) context.Context {
	return context.WithValue(ctx, sessionIDCtxKey{}, sid)
}

func SessionIDFromContext(ctx context.Context) string {
	sid, _ := ctx.Value(sessionIDCtxKey{}).(string)
	return sid
}

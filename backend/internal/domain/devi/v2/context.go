package v2

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
)

// contextWithTimeout returns a derived context that inherits the Fiber
// request's UserContext (so audit metadata propagates to GORM) but with a
// hard deadline. Cancel via the returned func to release resources.
func contextWithTimeout(c *fiber.Ctx, d time.Duration) (context.Context, context.CancelFunc) {
	parent := c.UserContext()
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, d)
}

// ─── Persona-in-context (used by tools to scope by user_id) ───────────────
//
// Tools NEVER trust the LLM to supply a user_id. The runtime injects the
// resolved Persona into the request context before calling Execute; tools
// read it via PersonaFromContext.

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

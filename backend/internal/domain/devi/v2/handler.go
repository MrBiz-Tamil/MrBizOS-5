package v2

import (
	"strings"
	"time"

	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handler is the Fiber-facing wrapper around the agent. Construct once at
// startup and mount via RegisterRoutes.
type Handler struct {
	agent *Agent
}

func NewHandler(agent *Agent) *Handler {
	return &Handler{agent: agent}
}

// RegisterRoutes mounts the V2 endpoints. V1 is untouched and continues to
// serve /devi/chat. V2 lives at /devi/v2/* during the parallel run.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	g := router.Group("/devi/v2")
	// Chat works with OR without auth — the persona resolver branches on
	// whether the JWT middleware set user_id. So we don't gate with
	// Protected(); we attach AuditCtx so any DB mutations triggered by
	// tools carry the right actor_id.
	g.Use(middleware.AuditCtx())
	g.Post("/chat", h.chat)

	// Debug — returns the resolved tool catalog for the current persona.
	// Useful for confirming RBAC scoping during rollout.
	g.Get("/tools", h.listTools)
}

// HandleChat is the public entry point that V1 can call when bucketing
// routes a request to V2. Identical to the /devi/v2/chat handler — kept
// separate name so the rollout shim is explicit at the call site.
func (h *Handler) HandleChat(c *fiber.Ctx) error {
	return h.chat(c)
}

// Agent exposes the underlying agent for advanced callers (e.g. WhatsApp
// adapters). Most code should use HandleChat or POST /devi/v2/chat.
func (h *Handler) Agent() *Agent {
	return h.agent
}

func (h *Handler) chat(c *fiber.Ctx) error {
	var req ChatRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}
	if strings.TrimSpace(req.Message) == "" {
		return response.Error(c, fiber.StatusBadRequest, "EMPTY_MESSAGE", "Message cannot be empty")
	}
	if req.SessionID == "" {
		req.SessionID = uuid.NewString()
	}

	persona := ResolvePersona(c, req.GuestBrowserID)
	// 20s upper bound regardless of LLM timeouts — Fiber's default is
	// long. Pass our own context so callers see a clean timeout if the
	// agent stalls.
	ctx, cancel := contextWithTimeout(c, 25*time.Second)
	defer cancel()
	resp := h.agent.Run(ctx, persona, req.SessionID, req.Message)
	return response.JSON(c, fiber.StatusOK, resp, "")
}

func (h *Handler) listTools(c *fiber.Ctx) error {
	persona := ResolvePersona(c, c.Query("guest_browser_id"))
	tools := h.agent.registry.ForPersona(persona)
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		out = append(out, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"confirm":     t.Confirm,
		})
	}
	return response.JSON(c, fiber.StatusOK, map[string]any{
		"role":  persona.Role,
		"tools": out,
	}, "")
}

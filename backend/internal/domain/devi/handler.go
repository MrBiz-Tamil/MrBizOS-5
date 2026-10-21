package devi

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"drayya-core/internal/middleware"
	notifSvc "drayya-core/internal/services/notification"
	"drayya-core/internal/tokens"
	"drayya-core/internal/utils/response"
)

type HubCollector interface {
	BroadcastMessage(roomID string, data interface{})
}

type Handler struct {
	svc Service
	hub HubCollector
	db  *gorm.DB
	rdb *redis.Client
}

func NewHandler(svc Service, hub HubCollector) *Handler {
	return &Handler{svc: svc, hub: hub}
}

// SetInfra injects the database and Redis handles used by admin endpoints and
// rate-limiting. Called from server/app.go immediately after NewHandler.
func (h *Handler) SetInfra(db *gorm.DB, rdb *redis.Client) {
	h.db = db
	h.rdb = rdb
}

// ─── HTTP Chat Endpoint ───────────────────────────────────────────────────────

type ChatRequest struct {
	// Auth fields (populated by middleware for logged-in callers)
	UserID         *uint `json:"user_id,omitempty"`
	FamilyMemberID *uint `json:"family_member_id,omitempty"`

	// Guest field (populated by frontend UUID stored in localStorage)
	GuestBrowserID string `json:"guest_browser_id,omitempty"`

	Message string `json:"message"`
}

type ChatResponse struct {
	Reply      string          `json:"reply"`
	Actions    []ChatAction    `json:"actions,omitempty"`
	Components []ChatComponent `json:"components,omitempty"`
	Done       bool            `json:"done"`
	SessionID  string          `json:"session_id"`
	IsGuest    bool            `json:"is_guest"`
}

func (h *Handler) DeviChat(c *fiber.Ctx) error {
	var req ChatRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if req.Message == "" {
		return response.Error(c, fiber.StatusBadRequest, "EMPTY_MESSAGE", "Message cannot be empty")
	}

	if uid := optionalJWTUserID(c); uid != nil {
		req.UserID = uid
	} else {
		req.UserID = nil
		req.FamilyMemberID = nil
		if req.GuestBrowserID == "" {
			req.GuestBrowserID = "anonymous"
		}
	}

	uc := BuildUserContext(req.UserID, req.FamilyMemberID, req.GuestBrowserID)
	start := time.Now()
	result := h.svc.Process(c.Context(), uc, req.Message)
	elapsed := int(time.Since(start).Milliseconds())

	// Stamp the response time onto the symptom log row this turn just created,
	// so the admin Avg Response stat reflects real measurement (not a constant).
	// Best-effort — failures are silently swallowed so chat responses are never
	// delayed by a stats-only side effect.
	if h.db != nil && uc.SessionKey != "" && elapsed > 0 {
		h.db.Exec(`
			UPDATE devi_symptom_logs SET response_ms = ?
			WHERE id = (
				SELECT id FROM devi_symptom_logs
				WHERE session_id = ? AND response_ms = 0
				ORDER BY id DESC LIMIT 1
			)`, elapsed, uc.SessionKey)
	}

	return response.JSON(c, fiber.StatusOK, ChatResponse{
		Reply:      result.Message,
		Actions:    result.Actions,
		Components: result.Components,
		Done:       result.Done,
		SessionID:  uc.SessionKey,
		IsGuest:    uc.IsGuest(),
	}, "")
}

func optionalJWTUserID(c *fiber.Ctx) *uint {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return nil
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil
	}
	claims, err := tokens.ValidateToken(parts[1])
	if err != nil || claims.Type == "refresh" || claims.Type == "mfa" || claims.UserID == "" {
		return nil
	}
	id, err := strconv.ParseUint(claims.UserID, 10, 32)
	if err != nil || id == 0 {
		return nil
	}
	uid := uint(id)
	return &uid
}

// ─── WhatsApp Webhook ─────────────────────────────────────────────────────────

func (h *Handler) WhatsAppWebhook(c *fiber.Ctx) error {
	if c.Method() == "GET" {
		challenge := c.Query("hub.challenge")
		return c.SendString(challenge)
	}

	var body struct {
		Entry []struct {
			Changes []struct {
				Value struct {
					Messages []struct {
						From string `json:"from"`
						Text struct {
							Body string `json:"body"`
						} `json:"text"`
					} `json:"messages"`
				} `json:"value"`
			} `json:"changes"`
		} `json:"entry"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.SendStatus(200)
	}

	for _, entry := range body.Entry {
		for _, change := range entry.Changes {
			for _, wMsg := range change.Value.Messages {
				phone := wMsg.From
				text := wMsg.Text.Body
				if text == "" {
					continue
				}

				// Rate limit: max 10 messages per phone per 60s window.
				// Silently drop and return 200 so Meta doesn't retry.
				if h.rdb != nil {
					rlKey := fmt.Sprintf("wa:rl:%s", phone)
					count, _ := h.rdb.Incr(c.Context(), rlKey).Result()
					if count == 1 {
						h.rdb.Expire(c.Context(), rlKey, 60*time.Second)
					}
					if count > 10 {
						log.Printf("[WhatsApp] rate-limited phone=%s (count=%d)", phone, count)
						continue
					}
				}

				replyText := h.svc.ProcessWhatsApp(c.Context(), phone, "wa-"+phone, text)

				if replyText != "" {
					go func(to, replyText string) {
						if err := notifSvc.SendText(to, replyText); err != nil {
							log.Printf("[WhatsApp] Failed to send reply to %s: %v", to, err)
						}
					}(phone, replyText)
				}
			}
		}
	}
	return c.SendStatus(200)
}

// ─── Memory Endpoints (Moved from handlers/devi.go) ───────────────────────────

func (h *Handler) GetHistory(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	limit := c.QueryInt("limit", 20)
	history, err := h.svc.GetSymptomHistory(userID, limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, history, "Symptom history fetched")
}

func (h *Handler) FindSimilar(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	q := c.Query("q")
	if q == "" {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_QUERY", "q param required")
	}
	limit := c.QueryInt("limit", 5)
	results, err := h.svc.FindSimilarSymptoms(userID, q, limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SEARCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, results, "Similar symptom logs")
}

func (h *Handler) DeleteHistory(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	if err := h.svc.DeleteSymptomHistory(userID, 0); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Symptom history deleted")
}

func (h *Handler) GetFamilyHistory(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	fmID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil || fmID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid family member ID")
	}
	limit := c.QueryInt("limit", 20)
	history, err := h.svc.GetFamilyMemberHistory(userID, uint(fmID), limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, history, "Family member symptom history fetched")
}

func (h *Handler) DeleteFamilyHistory(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	fmID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil || fmID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid family member ID")
	}
	if err := h.svc.DeleteSymptomHistory(userID, uint(fmID)); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Family member symptom history deleted")
}

// ─── Admin: Stats ─────────────────────────────────────────────────────────────

func (h *Handler) AdminGetStats(c *fiber.Ctx) error {
	if h.db == nil {
		return response.Error(c, fiber.StatusServiceUnavailable, "NO_DB", "Database not available")
	}
	stats, err := GetAdminStats(h.db)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "STATS_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, stats, "Stats fetched")
}

func (h *Handler) AdminGetLogs(c *fiber.Ctx) error {
	if h.db == nil {
		return response.Error(c, fiber.StatusServiceUnavailable, "NO_DB", "Database not available")
	}
	limit := c.QueryInt("limit", 50)
	if limit > 200 {
		limit = 200
	}
	logs, err := GetRecentSymptomLogs(h.db, limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "LOGS_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, logs, "Recent logs fetched")
}

// ─── Admin: Config ────────────────────────────────────────────────────────────

func (h *Handler) AdminGetConfig(c *fiber.Ctx) error {
	if h.db == nil {
		return response.Error(c, fiber.StatusServiceUnavailable, "NO_DB", "Database not available")
	}
	var configs []DeviConfig
	if err := h.db.Find(&configs).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CONFIG_FAILED", err.Error())
	}
	// Convert to map for easy frontend consumption
	out := make(map[string]string, len(configs))
	for _, c := range configs {
		out[c.Key] = c.Value
	}
	return response.JSON(c, fiber.StatusOK, out, "Config fetched")
}

func (h *Handler) AdminUpdateConfig(c *fiber.Ctx) error {
	if h.db == nil {
		return response.Error(c, fiber.StatusServiceUnavailable, "NO_DB", "Database not available")
	}
	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := c.BodyParser(&req); err != nil || req.Key == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "key and value are required")
	}
	cfg := DeviConfig{Key: req.Key, Value: req.Value, UpdatedAt: time.Now()}
	if err := h.db.Save(&cfg).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SAVE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, cfg, "Config updated")
}

func (h *Handler) AdminGetHistory(c *fiber.Ctx) error {
	idStr := c.Params("user_id")
	uid, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || uid == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid user_id")
	}
	history, err := h.svc.GetSymptomHistory(uint(uid), 50)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, history, "User symptom history fetched")
}

func (h *Handler) AdminGetPendingIntents(c *fiber.Ctx) error {
	if h.db == nil {
		return response.Error(c, fiber.StatusServiceUnavailable, "NO_DB", "Database not available")
	}
	var intents []LearnedIntent
	if err := h.db.Where("status = ?", "pending").Order("created_at desc").Find(&intents).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, intents, "Pending intents fetched")
}

func (h *Handler) AdminApproveIntent(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid intent ID")
	}
	if err := h.svc.ApproveLearnedIntent(c.Context(), uint(id)); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "APPROVE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Intent approved and injected into router")
}

func (h *Handler) AdminRejectIntent(c *fiber.Ctx) error {
	if h.db == nil {
		return response.Error(c, fiber.StatusServiceUnavailable, "NO_DB", "Database not available")
	}
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid intent ID")
	}
	if err := h.db.Model(&LearnedIntent{}).Where("id = ?", id).Update("status", "rejected").Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "REJECT_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Intent rejected")
}

// ─── Routes ───────────────────────────────────────────────────────────────────

func (h *Handler) RegisterRoutes(router fiber.Router) {
	deviGroup := router.Group("/devi")

	// Chat endpoints
	deviGroup.Post("/chat", h.DeviChat)
	deviGroup.Post("/whatsapp", h.WhatsAppWebhook)
	deviGroup.Get("/whatsapp", h.WhatsAppWebhook)

	// Memory endpoints
	deviGroup.Get("/history", middleware.Protected(), h.GetHistory)
	deviGroup.Get("/history/similar", middleware.Protected(), h.FindSimilar)
	deviGroup.Delete("/history", middleware.Protected(), h.DeleteHistory)
	deviGroup.Get("/family/:id/history", middleware.Protected(), h.GetFamilyHistory)
	deviGroup.Delete("/family/:id/history", middleware.Protected(), h.DeleteFamilyHistory)

	// Admin endpoints
	adminGroup := router.Group("/admin/devi", middleware.Protected(), middleware.AdminOnly())
	adminGroup.Get("/history/:user_id", h.AdminGetHistory)
	adminGroup.Get("/stats", h.AdminGetStats)
	adminGroup.Get("/logs", h.AdminGetLogs)
	adminGroup.Get("/config", h.AdminGetConfig)
	adminGroup.Patch("/config", h.AdminUpdateConfig)

	// Self-Learning Endpoints
	adminGroup.Get("/intents/pending", h.AdminGetPendingIntents)
	adminGroup.Post("/intents/:id/approve", h.AdminApproveIntent)
	adminGroup.Post("/intents/:id/reject", h.AdminRejectIntent)
}


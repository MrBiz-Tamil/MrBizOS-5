package analytics

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"drayya-core/internal/middleware"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) RegisterRoutes(router fiber.Router) {
	// Public event ingestion endpoint
	router.Post("/analytics/track", h.trackEvent)

	// Doctor self-serve analytics
	doctor := router.Group("/doctors/me/analytics", middleware.Protected(), middleware.RequireRoles("doctor"))
	doctor.Get("/", h.getDoctorAnalytics)
	doctor.Get("/cfo-metrics", h.getDoctorCFOMetrics) // Native & Web parity

	// Admin analytics
	admin := router.Group("/admin/analytics", middleware.Protected(), middleware.RequireRoles("admin", "super_admin"))
	admin.Get("/events", h.listEvents)
	admin.Get("/event-counts", h.eventCounts)

	// Direct alias for mobile/nextjs
	router.Get("/analytics/doctor/cfo-metrics", middleware.Protected(), middleware.RequireRoles("doctor"), h.getDoctorCFOMetrics)
}

// ── Event Ingestion ───────────────────────────────────────────────────────────

type TrackRequest struct {
	EventName  string                 `json:"event_name"`
	Domain     string                 `json:"domain"`
	SessionID  string                 `json:"session_id"`
	EntityID   *uint                  `json:"entity_id,omitempty"`
	Platform   string                 `json:"platform"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

func (h *Handler) trackEvent(c *fiber.Ctx) error {
	var req TrackRequest
	if err := c.BodyParser(&req); err != nil || req.EventName == "" {
		return c.Status(400).JSON(fiber.Map{"error": "event_name is required"})
	}

	var userID *uint
	if uid, ok := c.Locals("user_id").(uint); ok && uid > 0 {
		userID = &uid
	}

	if req.Platform == "" {
		req.Platform = "api"
	}

	Track(req.EventName, req.Domain, req.Platform, userID, req.SessionID, req.EntityID,
		req.Properties, c.IP(), string(c.Request().Header.UserAgent()))

	return c.JSON(fiber.Map{"status": "ok"})
}

// ── Doctor Analytics ──────────────────────────────────────────────────────────

func (h *Handler) getDoctorAnalytics(c *fiber.Ctx) error {
	doctorID := c.Locals("user_id").(uint)
	period := c.Query("period", "30d")

	switch period {
	case "30d", "90d", "all":
	default:
		period = "30d"
	}

	summary, err := RefreshDoctorAnalytics(doctorID, period)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(summary)
}

func (h *Handler) getDoctorCFOMetrics(c *fiber.Ctx) error {
	// Mocking data for now as per current analytics domain structure
	metrics := CFOMetrics{}
	metrics.Revenue.TotalMonthlyRevenue = 150000.0
	metrics.Revenue.ProjectedRevenue = 185000.0
	metrics.Revenue.GrowthPercentage = 12.5
	metrics.Revenue.BreakdownByService = map[string]float64{
		"Consultation": 120000.0,
		"Procedures":   250000.0,
		"Follow-ups":   15000.0,
	}
	metrics.Retention.RetentionRate = 78.0
	metrics.Retention.NewPatients = 45
	metrics.Retention.ReturningPatients = 120
	metrics.WaitTime.AverageWaitMinutes = 14.5
	metrics.WaitTime.AppointmentsOnTimePercentage = 92.0

	return c.JSON(metrics)
}

// ── Admin ─────────────────────────────────────────────────────────────────────

func (h *Handler) listEvents(c *fiber.Ctx) error {
	domain := c.Query("domain")
	limit := c.QueryInt("limit", 100)
	events, err := GetRecentEvents(limit, domain)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(events)
}

func (h *Handler) eventCounts(c *fiber.Ctx) error {
	fromStr := c.Query("from", time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
	toStr := c.Query("to", time.Now().Format("2006-01-02"))

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid from date"})
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid to date"})
	}
	to = to.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	counts, err := GetEventCountByName(from, to)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(counts)
}

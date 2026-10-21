package v2

import (
	"strconv"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"
	"github.com/gofiber/fiber/v2"
)

// RegisterAdminRoutes mounts the admin/observability endpoints. All routes
// are AdminOnly — the panel can't be opened by patients or staff.
//
// Mounted by server/app.go after the main /devi/v2 group so they live at
// /api/v1/admin/devi/v2/*.
func (h *Handler) RegisterAdminRoutes(router fiber.Router) {
	g := router.Group("/admin/devi/v2", middleware.Protected(), middleware.AdminOnly())

	g.Get("/stats", h.adminStats)             // KPI cards
	g.Get("/usage", h.adminRecentUsage)        // recent LLMUsage rows
	g.Get("/cost-series", h.adminCostSeries)   // daily cost for chart
	g.Get("/tool-stats", h.adminToolStats)     // calls/errors per tool
	g.Get("/tools", h.adminAllTools)           // registry view
	g.Get("/rollout", h.adminGetRollout)       // current % + source
	g.Put("/rollout", h.adminSetRollout)       // change % at runtime
	g.Get("/ab-compare", h.adminABCompare)     // V1 vs V2 side-by-side
}

// ─── /admin/devi/v2/stats ────────────────────────────────────────────────
// Top-of-page KPIs. All numbers come from real LLMUsage + symptom-log data;
// nothing is mocked.
func (h *Handler) adminStats(c *fiber.Ctx) error {
	if database.DB == nil {
		return response.Error(c, fiber.StatusServiceUnavailable, "NO_DB", "Database not available")
	}
	today := startOfDay(time.Now())
	tomorrow := today.Add(24 * time.Hour)
	last24h := time.Now().Add(-24 * time.Hour)

	// Total V2 requests today (counts LLM calls — one per turn round trip).
	var totalCalls int64
	database.DB.Model(&LLMUsage{}).Where("created_at >= ? AND created_at < ?", today, tomorrow).Count(&totalCalls)

	// Spend today (sum CostUSD).
	var spendToday float64
	database.DB.Model(&LLMUsage{}).
		Where("created_at >= ? AND created_at < ?", today, tomorrow).
		Select("COALESCE(SUM(cost_usd), 0)").Scan(&spendToday)

	// Spend month-to-date.
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
	var spendMonth float64
	database.DB.Model(&LLMUsage{}).
		Where("created_at >= ?", monthStart).
		Select("COALESCE(SUM(cost_usd), 0)").Scan(&spendMonth)

	// Avg latency last 24h (only successful calls).
	var avgLatency float64
	database.DB.Model(&LLMUsage{}).
		Where("created_at >= ? AND was_error = ?", last24h, false).
		Select("COALESCE(AVG(latency_ms), 0)").Scan(&avgLatency)

	// Error rate last 24h (%) — wasError = true divided by total.
	var totalLast24h, errorsLast24h int64
	database.DB.Model(&LLMUsage{}).Where("created_at >= ?", last24h).Count(&totalLast24h)
	database.DB.Model(&LLMUsage{}).Where("created_at >= ? AND was_error = ?", last24h, true).Count(&errorsLast24h)
	errorRate := 0.0
	if totalLast24h > 0 {
		errorRate = float64(errorsLast24h) / float64(totalLast24h) * 100
	}

	// Tokens consumed today (input + output).
	type tokRow struct {
		InputTokens  int64
		OutputTokens int64
	}
	var tok tokRow
	database.DB.Model(&LLMUsage{}).
		Where("created_at >= ? AND created_at < ?", today, tomorrow).
		Select("COALESCE(SUM(input_tokens), 0) AS input_tokens, COALESCE(SUM(output_tokens), 0) AS output_tokens").
		Scan(&tok)

	// Rollout state.
	pct, source := CurrentRolloutPct()

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"calls_today":       totalCalls,
		"spend_today_usd":   spendToday,
		"spend_month_usd":   spendMonth,
		"avg_latency_ms":    int(avgLatency),
		"error_rate_pct":    errorRate,
		"input_tokens_today":  tok.InputTokens,
		"output_tokens_today": tok.OutputTokens,
		"daily_cost_cap_usd":  DailyCostCapUSD,
		"monthly_cost_cap_usd": MonthlyCostCapUSD,
		"rollout_pct":         pct,
		"rollout_source":      source,
	}, "V2 stats fetched")
}

// ─── /admin/devi/v2/usage?limit=50 ───────────────────────────────────────
// Recent LLM calls. PII-safe — we don't store the message content in this
// table, only the metrics.
func (h *Handler) adminRecentUsage(c *fiber.Ctx) error {
	if database.DB == nil {
		return response.Error(c, fiber.StatusServiceUnavailable, "NO_DB", "Database not available")
	}
	limit := c.QueryInt("limit", 50)
	if limit < 1 || limit > 500 {
		limit = 50
	}
	var rows []LLMUsage
	database.DB.Order("created_at DESC").Limit(limit).Find(&rows)
	return response.JSON(c, fiber.StatusOK, rows, "Recent usage fetched")
}

// ─── /admin/devi/v2/cost-series?days=7 ───────────────────────────────────
// One bucket per day for the cost line chart.
func (h *Handler) adminCostSeries(c *fiber.Ctx) error {
	if database.DB == nil {
		return response.Error(c, fiber.StatusServiceUnavailable, "NO_DB", "Database not available")
	}
	days := c.QueryInt("days", 7)
	if days < 1 || days > 90 {
		days = 7
	}
	today := startOfDay(time.Now())
	start := today.AddDate(0, 0, -(days - 1))

	type out struct {
		Date      string  `json:"date"`
		SpendUSD  float64 `json:"spend_usd"`
		CallCount int64   `json:"call_count"`
	}
	series := make([]out, 0, days)
	for i := 0; i < days; i++ {
		dayStart := start.AddDate(0, 0, i)
		dayEnd := dayStart.AddDate(0, 0, 1)
		var spend float64
		var count int64
		database.DB.Model(&LLMUsage{}).
			Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
			Select("COALESCE(SUM(cost_usd), 0)").Scan(&spend)
		database.DB.Model(&LLMUsage{}).
			Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
			Count(&count)
		series = append(series, out{
			Date:      dayStart.Format("2006-01-02"),
			SpendUSD:  spend,
			CallCount: count,
		})
	}
	return response.JSON(c, fiber.StatusOK, series, "Cost series fetched")
}

// ─── /admin/devi/v2/tool-stats ───────────────────────────────────────────
// For now this returns the registered tool catalog with descriptions and
// the persona allowlist. Per-tool usage counters (Phase 9) will be added
// when individual tool invocations get logged to LLMUsage with the
// tool name — today we record token costs not per-tool calls.
func (h *Handler) adminToolStats(c *fiber.Ctx) error {
	tools := h.agent.registry.All()
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		allowed := make([]string, 0, len(t.Allowed))
		for _, r := range t.Allowed {
			allowed = append(allowed, string(r))
		}
		out = append(out, map[string]any{
			"name":             t.Name,
			"description":      t.Description,
			"allowed_personas": allowed,
			"confirm":          t.Confirm,
			"max_per_session":  t.MaxPerSession,
			"max_per_minute":   t.MaxPerMinute,
		})
	}
	return response.JSON(c, fiber.StatusOK, out, "Tool catalog fetched")
}

// ─── /admin/devi/v2/tools ────────────────────────────────────────────────
// Alias used by the frontend; identical to tool-stats today but reserved
// for divergence when per-tool invocation counters are wired up.
func (h *Handler) adminAllTools(c *fiber.Ctx) error {
	return h.adminToolStats(c)
}

// ─── /admin/devi/v2/rollout ─────────────────────────────────────────────
func (h *Handler) adminGetRollout(c *fiber.Ctx) error {
	pct, source := CurrentRolloutPct()
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"rollout_pct": pct,
		"source":      source,
	}, "Rollout fetched")
}

func (h *Handler) adminSetRollout(c *fiber.Ctx) error {
	var req struct {
		Pct int `json:"pct"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}
	stored, err := SetRolloutPct(c.UserContext(), req.Pct)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SET_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"rollout_pct": stored,
		"source":      "redis",
	}, "Rollout updated")
}

// ─── /admin/devi/v2/ab-compare ──────────────────────────────────────────
// Pulls the same windowed metrics from devi_symptom_logs comparing V1 vs
// V2. We distinguish using a simple heuristic: rows with non-zero
// response_ms are written by V2 (V1 doesn't stamp it for every turn).
// This is approximate; a clean fix lands in Phase 6 when V1 also stamps.
func (h *Handler) adminABCompare(c *fiber.Ctx) error {
	if database.DB == nil {
		return response.Error(c, fiber.StatusServiceUnavailable, "NO_DB", "Database not available")
	}
	hours := c.QueryInt("hours", 24)
	if hours < 1 || hours > 720 {
		hours = 24
	}
	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	type sideMetrics struct {
		Sessions   int64   `json:"sessions"`
		AvgLatency float64 `json:"avg_latency_ms"`
		Escalation float64 `json:"escalation_rate_pct"`
		Booking    float64 `json:"booking_conversion_pct"`
	}

	compute := func(where string) sideMetrics {
		var m sideMetrics
		var n, escal, booked int64
		database.DB.Table("devi_symptom_logs").
			Where("created_at >= ? AND "+where, since).Count(&n)
		database.DB.Table("devi_symptom_logs").
			Where("created_at >= ? AND "+where+" AND urgency >= 4", since).Count(&escal)
		database.DB.Table("devi_symptom_logs").
			Where("created_at >= ? AND "+where+" AND booked_appt = ?", since, true).Count(&booked)
		database.DB.Table("devi_symptom_logs").
			Where("created_at >= ? AND "+where+" AND response_ms > 0", since).
			Select("COALESCE(AVG(response_ms), 0)").Scan(&m.AvgLatency)
		m.Sessions = n
		if n > 0 {
			m.Escalation = float64(escal) / float64(n) * 100
			m.Booking = float64(booked) / float64(n) * 100
		}
		return m
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"window_hours": hours,
		"v1":           compute("(response_ms = 0 OR response_ms IS NULL)"),
		"v2":           compute("response_ms > 0"),
	}, "A/B comparison fetched")
}

// startOfDay returns midnight in the local timezone for the given moment.
func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// Compile guard — strconv used in another file pulls in if unused elsewhere
// after future refactors. Keeping it referenced is cheaper than chasing a
// stray import.
var _ = strconv.Itoa

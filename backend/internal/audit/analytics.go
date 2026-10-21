package audit

import (
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// RegisterAnalyticsRoutes mounts read-only aggregation endpoints over
// audit_logs. All require admin auth. Charts on the /admin/activity-logs
// → 📈 Analytics tab consume these.
func RegisterAnalyticsRoutes(router fiber.Router) {
	g := router.Group("/admin/audit-logs/analytics",
		middleware.Protected(), middleware.AdminOnly())
	g.Get("/by-action", AnalyticsByAction)
	g.Get("/by-role", AnalyticsByRole)
	g.Get("/by-table", AnalyticsByTable)
	g.Get("/top-actors", AnalyticsTopActors)
	g.Get("/daily-trend", AnalyticsDailyTrend)
}

// daysWindow parses ?days=N from the request, default 30, cap 365.
func daysWindow(c *fiber.Ctx) time.Time {
	days := c.QueryInt("days", 30)
	if days < 1 || days > 365 {
		days = 30
	}
	return time.Now().Add(-time.Duration(days) * 24 * time.Hour)
}

// AnalyticsByAction returns counts of CREATE / UPDATE / DELETE in the window.
func AnalyticsByAction(c *fiber.Ctx) error {
	since := daysWindow(c)
	type row struct {
		Action string `json:"action"`
		Count  int64  `json:"count"`
	}
	var rows []row
	database.DB.Table("audit_logs").
		Select("action, COUNT(*) AS count").
		Where("created_at >= ?", since).
		Group("action").
		Order("count DESC").
		Scan(&rows)
	if rows == nil {
		rows = []row{}
	}
	return response.JSON(c, fiber.StatusOK, rows, "By-action analytics")
}

// AnalyticsByRole groups by actor_role.
func AnalyticsByRole(c *fiber.Ctx) error {
	since := daysWindow(c)
	type row struct {
		Role  string `json:"role" gorm:"column:actor_role"`
		Count int64  `json:"count"`
	}
	var rows []row
	database.DB.Table("audit_logs").
		Select("actor_role, COUNT(*) AS count").
		Where("created_at >= ? AND actor_role != ''", since).
		Group("actor_role").
		Order("count DESC").
		Scan(&rows)
	if rows == nil {
		rows = []row{}
	}
	return response.JSON(c, fiber.StatusOK, rows, "By-role analytics")
}

// AnalyticsByTable returns the top 10 most-mutated tables.
func AnalyticsByTable(c *fiber.Ctx) error {
	since := daysWindow(c)
	type row struct {
		Table string `json:"table" gorm:"column:table_name"`
		Count int64  `json:"count"`
	}
	var rows []row
	database.DB.Table("audit_logs").
		Select("table_name, COUNT(*) AS count").
		Where("created_at >= ?", since).
		Group("table_name").
		Order("count DESC").
		Limit(10).
		Scan(&rows)
	if rows == nil {
		rows = []row{}
	}
	return response.JSON(c, fiber.StatusOK, rows, "By-table analytics")
}

// AnalyticsTopActors returns top 10 most-active human users (actor_id > 0,
// excludes system jobs which always have actor_id = 0).
func AnalyticsTopActors(c *fiber.Ctx) error {
	since := daysWindow(c)
	type row struct {
		ActorID   uint   `json:"actor_id"`
		ActorRole string `json:"actor_role"`
		Count     int64  `json:"count"`
	}
	var rows []row
	database.DB.Table("audit_logs").
		Select("actor_id, actor_role, COUNT(*) AS count").
		Where("created_at >= ? AND actor_id > 0", since).
		Group("actor_id, actor_role").
		Order("count DESC").
		Limit(10).
		Scan(&rows)
	if rows == nil {
		rows = []row{}
	}
	return response.JSON(c, fiber.StatusOK, rows, "Top-actors analytics")
}

// AnalyticsDailyTrend returns one row per day in the window with the total
// number of audit events. Used to render a sparkline / line chart.
func AnalyticsDailyTrend(c *fiber.Ctx) error {
	since := daysWindow(c)
	type row struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}

	// Cross-DB date truncation: DATE() works on both SQLite and PostgreSQL.
	var rows []row
	database.DB.Table("audit_logs").
		Select("DATE(created_at) AS date, COUNT(*) AS count").
		Where("created_at >= ?", since).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&rows)
	if rows == nil {
		rows = []row{}
	}
	return response.JSON(c, fiber.StatusOK, rows, "Daily trend")
}

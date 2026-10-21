package anomaly

import (
	"strconv"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes mounts admin anomaly endpoints under /admin/audit-anomalies.
func RegisterRoutes(router fiber.Router) {
	g := router.Group("/admin/audit-anomalies", middleware.Protected(), middleware.HRManagerOnly())
	g.Get("/", ListAnomalies)
	g.Get("/summary", Summary)
	g.Post("/:id/resolve", ResolveAnomaly)
	g.Post("/scan", TriggerScan)
}

// ListAnomalies returns a paginated list of anomalies, newest first.
//
// Query params:
//   status     new | resolved | all  (default: new)
//   severity   info | warning | critical
//   page       1
//   limit      50 (max 200)
func ListAnomalies(c *fiber.Ctx) error {
	status := c.Query("status", StatusNew)
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}
	limit := c.QueryInt("limit", 50)
	if limit < 1 || limit > 200 {
		limit = 50
	}

	q := database.DB.Model(&Anomaly{})
	if status != "all" {
		q = q.Where("status = ?", status)
	}
	if sev := c.Query("severity"); sev != "" {
		q = q.Where("severity = ?", sev)
	}

	var total int64
	q.Count(&total)

	var rows []Anomaly
	if err := q.Order("id desc").
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	if rows == nil {
		rows = []Anomaly{}
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"anomalies": rows,
		"total":     total,
		"page":      page,
		"limit":     limit,
	}, "Anomalies fetched")
}

// Summary returns counts by severity for the new-anomaly badge in the UI.
func Summary(c *fiber.Ctx) error {
	type row struct {
		Severity string
		N        int64
	}
	var rows []row
	database.DB.Model(&Anomaly{}).
		Select("severity, COUNT(*) AS n").
		Where("status = ?", StatusNew).
		Group("severity").
		Scan(&rows)

	out := map[string]int64{
		SeverityCritical: 0,
		SeverityWarning:  0,
		SeverityInfo:     0,
		"total":          0,
	}
	for _, r := range rows {
		out[r.Severity] = r.N
		out["total"] += r.N
	}
	return response.JSON(c, fiber.StatusOK, out, "Summary fetched")
}

// ResolveAnomaly marks an anomaly as reviewed/resolved.
//
// Body: { "notes": "investigated, was a legitimate bulk import" }
func ResolveAnomaly(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid anomaly ID")
	}
	var body struct {
		Notes string `json:"notes"`
	}
	_ = c.BodyParser(&body)

	now := time.Now()
	resolvedBy, _ := c.Locals("user_id").(uint)
	if err := database.DB.Model(&Anomaly{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      StatusResolved,
			"resolved_at": &now,
			"resolved_by": resolvedBy,
			"notes":       body.Notes,
		}).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Anomaly resolved")
}

// TriggerScan kicks off an immediate scan instead of waiting for the next
// 15-minute tick. Useful after an admin action they want to verify.
func TriggerScan(c *fiber.Ctx) error {
	go scan(30 * time.Minute) // wide window for manual triggers
	return response.JSON(c, fiber.StatusAccepted, fiber.Map{
		"message": "Scan triggered — refresh in 5 seconds to see results",
	}, "Scan triggered")
}

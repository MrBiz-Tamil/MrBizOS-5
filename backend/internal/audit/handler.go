package audit

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes mounts the admin audit-log query endpoints.
// All routes require an authenticated admin (middleware.AdminOnly).
func RegisterRoutes(router fiber.Router) {
	g := router.Group("/admin/audit-logs", middleware.Protected(), middleware.AdminOnly())
	g.Get("/", ListAuditLogs)
	g.Get("/verify-chain", VerifyChainHandler)
	g.Get("/export.csv", ExportCSV)
	g.Get("/user/:user_id", LogsForUser)
	g.Get("/table/:table/:row_id", LogsForRow)
	g.Get("/:id", GetAuditLog)
}

// ListAuditLogs returns a paginated, filtered list of audit log entries.
//
// Query params:
//   actor_id      filter by actor user_id
//   action        filter by CREATE | UPDATE | DELETE
//   table         filter by exact table name
//   from, to      ISO-8601 date range
//   search        substring match across row_id, ip, path
//   page, limit   pagination (defaults: page=1, limit=50, max limit=200)
func ListAuditLogs(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}
	limit := c.QueryInt("limit", 50)
	if limit < 1 || limit > 200 {
		limit = 50
	}

	q := database.DB.Model(&AuditLog{})

	if actorID := c.QueryInt("actor_id", 0); actorID > 0 {
		q = q.Where("actor_id = ?", actorID)
	}
	if action := strings.ToUpper(c.Query("action")); action != "" {
		q = q.Where("action = ?", action)
	}
	if table := c.Query("table"); table != "" {
		q = q.Where("table_name = ?", table)
	}
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			q = q.Where("created_at >= ?", t)
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			q = q.Where("created_at <= ?", t)
		}
	}
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		needle := "%" + search + "%"
		q = q.Where(
			"row_id LIKE ? OR ip_address LIKE ? OR path LIKE ? OR request_id LIKE ?",
			needle, needle, needle, needle,
		)
	}

	var total int64
	q.Count(&total)

	var rows []AuditLog
	if err := q.Order("id desc").
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	if rows == nil {
		rows = []AuditLog{}
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"logs":  rows,
		"total": total,
		"page":  page,
		"limit": limit,
	}, "Audit logs fetched")
}

// GetAuditLog returns a single audit log row with full before/after JSON.
func GetAuditLog(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid audit log ID")
	}
	var log AuditLog
	if err := database.DB.First(&log, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Audit log not found")
	}
	return response.JSON(c, fiber.StatusOK, log, "Audit log fetched")
}

// LogsForUser returns every audit log row produced by a given user, newest first.
func LogsForUser(c *fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("user_id"), 10, 64)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid user_id")
	}
	limit := c.QueryInt("limit", 100)
	if limit < 1 || limit > 500 {
		limit = 100
	}
	var rows []AuditLog
	if err := database.DB.Where("actor_id = ?", userID).
		Order("id desc").Limit(limit).
		Find(&rows).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	if rows == nil {
		rows = []AuditLog{}
	}
	return response.JSON(c, fiber.StatusOK, rows, "User audit history fetched")
}

// LogsForRow returns the full history of a single row in a given table.
// Useful when investigating "who changed prescription #5543".
func LogsForRow(c *fiber.Ctx) error {
	table := c.Params("table")
	rowID := c.Params("row_id")
	if table == "" || rowID == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_PARAMS", "table and row_id are required")
	}
	var rows []AuditLog
	if err := database.DB.Where("table_name = ? AND row_id = ?", table, rowID).
		Order("id asc").
		Find(&rows).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	if rows == nil {
		rows = []AuditLog{}
	}
	return response.JSON(c, fiber.StatusOK, rows, "Row history fetched")
}

// VerifyChainHandler runs the chain verification across the entire table.
// Returns the row count and the ID of the first broken row (0 = intact).
func VerifyChainHandler(c *fiber.Ctx) error {
	total, broken, err := VerifyChain(database.DB)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "VERIFY_FAILED", err.Error())
	}
	status := "INTACT"
	if broken > 0 {
		status = "BROKEN"
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"status":              status,
		"total_rows":          total,
		"first_broken_row_id": broken,
	}, "Chain verification complete")
}

// ExportCSV streams the filtered audit log set as a CSV file. Same filters
// as ListAuditLogs but no pagination — capped at 50,000 rows to prevent
// runaway exports.
func ExportCSV(c *fiber.Ctx) error {
	q := database.DB.Model(&AuditLog{})

	if actorID := c.QueryInt("actor_id", 0); actorID > 0 {
		q = q.Where("actor_id = ?", actorID)
	}
	if action := strings.ToUpper(c.Query("action")); action != "" {
		q = q.Where("action = ?", action)
	}
	if table := c.Query("table"); table != "" {
		q = q.Where("table_name = ?", table)
	}
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			q = q.Where("created_at >= ?", t)
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			q = q.Where("created_at <= ?", t)
		}
	}

	var rows []AuditLog
	if err := q.Order("id desc").Limit(50_000).Find(&rows).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="audit_logs_%s.csv"`, time.Now().Format("20060102_150405")))

	w := csv.NewWriter(c.Response().BodyWriter())
	defer w.Flush()

	// Header
	w.Write([]string{
		"id", "created_at", "actor_id", "actor_role", "action",
		"table", "row_id", "request_id", "ip", "path", "method", "hash_chain",
	})
	for _, r := range rows {
		w.Write([]string{
			strconv.FormatUint(r.ID, 10),
			r.CreatedAt.Format(time.RFC3339),
			strconv.FormatUint(uint64(r.ActorID), 10),
			r.ActorRole,
			r.Action,
			r.Table,
			r.RowID,
			r.RequestID,
			r.IPAddress,
			r.Path,
			r.Method,
			r.HashChain,
		})
	}
	return nil
}

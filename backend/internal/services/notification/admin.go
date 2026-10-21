package notification

// admin.go — HTTP handler for the notification management admin panel.
//
// Endpoints (all require Protected + AdminOnly middleware):
//
//	GET  /admin/notifications          — paginated job list (filterable)
//	GET  /admin/notifications/stats    — counts by status + channel, dead job list
//	POST /admin/notifications/:id/retry — requeue a dead/failed job
//	DELETE /admin/notifications/:id    — hard-delete a dead job

import (
	"fmt"
	"strconv"
	"time"

	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// AdminHandler exposes notification management endpoints for the admin panel.
type AdminHandler struct {
	db *gorm.DB
}

// NewAdminHandler returns an AdminHandler backed by the given DB.
func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

// RegisterAdminRoutes registers all notification admin routes onto router.
// The caller is expected to wrap router with Protected + AdminOnly middleware.
func (h *AdminHandler) RegisterAdminRoutes(router fiber.Router) {
	g := router.Group("/notifications")
	g.Get("/", h.ListJobs)
	g.Get("/stats", h.Stats)
	g.Post("/:id/retry", h.RetryJob)
	g.Delete("/:id", h.DeleteJob)
}

// ── handlers ─────────────────────────────────────────────────────────────────

// ListJobs godoc
// GET /admin/notifications
// Query params: status, channel, job_type, limit (default 50), offset (default 0)
func (h *AdminHandler) ListJobs(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	q := h.db.Model(&NotificationJob{}).Order("created_at DESC")

	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	if ch := c.Query("channel"); ch != "" {
		q = q.Where("channel = ?", ch)
	}
	if jt := c.Query("job_type"); jt != "" {
		q = q.Where("job_type = ?", jt)
	}

	var total int64
	q.Count(&total)

	var jobs []NotificationJob
	if err := q.Limit(limit).Offset(offset).Find(&jobs).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"total":  total,
		"limit":  limit,
		"offset": offset,
		"jobs":   safeJobs(jobs),
	}, "Notification jobs fetched")
}

// Stats godoc
// GET /admin/notifications/stats
// Returns counts broken down by status and channel, plus the full dead-job list.
func (h *AdminHandler) Stats(c *fiber.Ctx) error {
	type statusCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	type channelCount struct {
		Channel Channel `json:"channel"`
		Count   int64   `json:"count"`
	}

	var byStatus []statusCount
	h.db.Model(&NotificationJob{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&byStatus)

	var byChannel []channelCount
	h.db.Model(&NotificationJob{}).
		Select("channel, count(*) as count").
		Group("channel").
		Scan(&byChannel)

	// Dead jobs in full so admins can see exactly what failed
	var dead []NotificationJob
	h.db.Model(&NotificationJob{}).
		Where("status = 'dead'").
		Order("created_at DESC").
		Limit(100).
		Find(&dead)

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"by_status":  byStatus,
		"by_channel": byChannel,
		"dead_jobs":  safeJobs(dead),
	}, "Notification stats")
}

// RetryJob godoc
// POST /admin/notifications/:id/retry
// Resets a dead or failed job back to pending so the worker picks it up.
// Also bumps MaxAttempts by 3 (adds headroom for the fresh retry run).
func (h *AdminHandler) RetryJob(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid job ID")
	}

	var job NotificationJob
	if err := h.db.First(&job, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Job %d not found", id))
	}

	if job.Status != "dead" && job.Status != "pending" {
		return response.Error(c, fiber.StatusConflict, "INVALID_STATE",
			fmt.Sprintf("Job %d is in status %q — only dead or stuck-pending jobs can be retried", id, job.Status))
	}

	if err := h.db.Model(&job).Updates(map[string]interface{}{
		"status":          "pending",
		"next_attempt_at": time.Now(),
		"last_error":      "",
		// Give extra attempts headroom on manual retry
		"max_attempts": job.Attempts + 3,
	}).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{"id": id, "status": "pending"}, "Job re-queued")
}

// DeleteJob godoc
// DELETE /admin/notifications/:id
// Hard-deletes a notification job. Only dead jobs should be deleted to avoid
// losing in-flight deliveries.
func (h *AdminHandler) DeleteJob(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid job ID")
	}

	var job NotificationJob
	if err := h.db.First(&job, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Job %d not found", id))
	}

	if job.Status != "dead" {
		return response.Error(c, fiber.StatusConflict, "INVALID_STATE",
			fmt.Sprintf("Job %d has status %q — only dead jobs can be deleted", id, job.Status))
	}

	if err := h.db.Delete(&NotificationJob{}, id).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{"id": id}, "Job deleted")
}

// ── helpers ───────────────────────────────────────────────────────────────────

func parseID(c *fiber.Ctx) (uint, error) {
	n, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || n == 0 {
		return 0, err
	}
	return uint(n), nil
}

// safeJob is the admin-facing projection of a NotificationJob.
// Body is omitted (can be large HTML) unless the status is dead/failed.
type safeJob struct {
	ID            uint      `json:"id"`
	Channel       Channel   `json:"channel"`
	JobType       string    `json:"job_type"`
	Recipient     string    `json:"recipient"`
	Subject       string    `json:"subject,omitempty"`
	TemplateID    string    `json:"template_id,omitempty"`
	Status        string    `json:"status"`
	Attempts      int       `json:"attempts"`
	MaxAttempts   int       `json:"max_attempts"`
	LastError     string    `json:"last_error,omitempty"`
	NextAttemptAt time.Time `json:"next_attempt_at"`
	SentAt        *time.Time `json:"sent_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func safeJobs(jobs []NotificationJob) []safeJob {
	out := make([]safeJob, len(jobs))
	for i, j := range jobs {
		out[i] = safeJob{
			ID:            j.ID,
			Channel:       j.Channel,
			JobType:       j.JobType,
			Recipient:     j.Recipient,
			Subject:       j.Subject,
			TemplateID:    j.TemplateID,
			Status:        j.Status,
			Attempts:      j.Attempts,
			MaxAttempts:   j.MaxAttempts,
			LastError:     j.LastError,
			NextAttemptAt: j.NextAttemptAt,
			SentAt:        j.SentAt,
			CreatedAt:     j.CreatedAt,
		}
	}
	return out
}

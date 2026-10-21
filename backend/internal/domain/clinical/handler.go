package clinical

import (
	"strconv"

	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler { return &Handler{db: db} }

// RegisterRoutes mounts the clinical-oversight endpoints. The group is open to
// the head doctor (chief_doctor) and the network manager; admin/super_admin
// bypass via RequireRoles. Pod mutations are further restricted to manager/admin.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	g := router.Group("/clinical", middleware.RequireRoles("chief_doctor", "hospital_head_doctor", "manager", "admin"))
	g.Get("/dashboard", h.Dashboard)
	g.Get("/stats", h.Stats)
	g.Get("/review-queue", h.ReviewQueue)
	g.Get("/reviews", h.ListReviews)
	g.Post("/consultations/:appointmentId/review", h.SubmitReview)

	// Doctor onboarding pipeline (literal segments before /:id).
	g.Post("/onboarding/start", h.StartOnboarding)
	g.Get("/onboarding/candidates", h.OnboardingCandidates)
	// High-throughput verification queue (worklist + KPIs + bulk actions).
	g.Get("/onboarding/queue", h.VerificationQueue)
	g.Get("/onboarding/queue/summary", h.QueueSummary)
	g.Post("/onboarding/bulk", h.BulkAction)
	g.Get("/onboarding", h.ListOnboarding)
	g.Get("/onboarding/:id/application", h.OnboardingApplication)
	g.Post("/onboarding/:id/clinical-review", h.ClinicalReview)
	g.Patch("/onboarding/:id/checklist", h.UpdateChecklist)
	g.Post("/onboarding/:id/activate", h.ActivateDoctor)

	// Specialty pods (org structure).
	h.registerPodRoutes(g)

	// Scorecard & tiers (Phase 2).
	h.registerScorecardRoutes(g)

	// Compliance monitor (Phase 3) — separately gated for HR access.
	h.registerComplianceRoutes(router)

	// Manager console — cross-pod rollup + supply/demand (Phase 4).
	h.registerNetworkRoutes(g)
}

// queueRow is one un-reviewed completed consultation awaiting QA.
type queueRow struct {
	AppointmentID    uint   `json:"appointment_id"`
	DoctorID         uint   `json:"doctor_id"`
	DoctorName       string `json:"doctor_name"`
	PatientName      string `json:"patient_name"`
	StartTime        string `json:"start_time"`
	ConsultationMode string `json:"consultation_mode"`
	AppointmentType  string `json:"appointment_type"`
	PatientScore     *int   `json:"patient_score"`   // patient's own rating, if any
	PatientComment   string `json:"patient_comment"`
}

// ReviewQueue returns completed consultations that have not yet been QA-reviewed,
// newest first, with the patient's own rating attached for context.
//
// GET /clinical/review-queue?limit=&offset=&flagged_only=
func (h *Handler) ReviewQueue(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 25)
	offset := c.QueryInt("offset", 0)
	if limit > 100 {
		limit = 100
	}

	q := h.db.Table("appointments AS a").
		Select(`a.id AS appointment_id, a.doctor_id AS doctor_id,
			du.full_name AS doctor_name, pu.full_name AS patient_name,
			a.start_time AS start_time, a.consultation_mode AS consultation_mode,
			a.appointment_type AS appointment_type,
			r.score AS patient_score, r.comment AS patient_comment`).
		Joins(`JOIN "user" du ON du.id = a.doctor_id`).
		Joins(`JOIN "user" pu ON pu.id = a.patient_id`).
		Joins(`LEFT JOIN ratings r ON r.appointment_id = a.id AND r.is_hidden = false`).
		Where("a.status = ?", "completed").
		Where("NOT EXISTS (SELECT 1 FROM consultation_reviews cr WHERE cr.appointment_id = a.id)")
	// Pod scoping — a head doctor only sees their pod's consultations.
	if ids, scoped := h.scopedDoctorIDs(c); scoped {
		q = q.Where("a.doctor_id IN ?", ids)
	}
	var rows []queueRow
	err := q.Order("a.start_time DESC").Limit(limit).Offset(offset).Scan(&rows).Error
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "QUEUE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, rows, "Review queue fetched")
}

// SubmitReviewRequest is the body for POST /clinical/consultations/:id/review.
type SubmitReviewRequest struct {
	QualityScore int    `json:"quality_score"`
	Notes        string `json:"notes"`
	Flagged      bool   `json:"flagged"`
	Outcome      string `json:"outcome"` // ok | coaching | warning
}

// SubmitReview records a chief doctor's QA assessment of a completed consultation.
//
// POST /clinical/consultations/:appointmentId/review
func (h *Handler) SubmitReview(c *fiber.Ctx) error {
	reviewerID, _ := c.Locals("user_id").(uint)
	apptID, err := strconv.Atoi(c.Params("appointmentId"))
	if err != nil || apptID <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid appointment ID")
	}
	var req SubmitReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	if req.QualityScore < 1 || req.QualityScore > 5 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_SCORE", "quality_score must be 1–5")
	}
	switch req.Outcome {
	case "", "ok":
		req.Outcome = "ok"
	case "coaching", "warning":
	default:
		return response.Error(c, fiber.StatusBadRequest, "INVALID_OUTCOME", "outcome must be ok, coaching, or warning")
	}

	// Resolve the reviewed doctor + guard that the appointment is actually a
	// completed consultation (can't review something that didn't happen).
	var appt struct {
		DoctorID *uint
		Status   string
	}
	if err := h.db.Table("appointments").Select("doctor_id, status").
		Where("id = ?", apptID).Scan(&appt).Error; err != nil || appt.DoctorID == nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
	}
	if appt.Status != "completed" {
		return response.Error(c, fiber.StatusUnprocessableEntity, "NOT_COMPLETED", "Only completed consultations can be reviewed")
	}

	// One review per appointment.
	var exists int64
	h.db.Model(&ConsultationReview{}).Where("appointment_id = ?", apptID).Count(&exists)
	if exists > 0 {
		return response.Error(c, fiber.StatusConflict, "ALREADY_REVIEWED", "This consultation has already been reviewed")
	}

	review := ConsultationReview{
		AppointmentID: uint(apptID),
		DoctorID:      *appt.DoctorID,
		ReviewerID:    reviewerID,
		QualityScore:  req.QualityScore,
		Notes:         req.Notes,
		Flagged:       req.Flagged,
		Outcome:       req.Outcome,
	}
	// WithContext so the audit log attributes who reviewed the consultation.
	if err := h.db.WithContext(c.UserContext()).Create(&review).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, review, "Consultation reviewed")
}

// reviewRow is one past QA review with doctor name for the history view.
type reviewRow struct {
	ConsultationReview
	DoctorName string `json:"doctor_name"`
}

// ListReviews returns past QA reviews, newest first.
//
// GET /clinical/reviews?doctor_id=&flagged_only=&limit=&offset=
func (h *Handler) ListReviews(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)
	if limit > 200 {
		limit = 200
	}
	q := h.db.Table("consultation_reviews AS cr").
		Select(`cr.*, du.full_name AS doctor_name`).
		Joins(`JOIN "user" du ON du.id = cr.doctor_id`)
	if did := c.QueryInt("doctor_id", 0); did > 0 {
		q = q.Where("cr.doctor_id = ?", did)
	}
	if c.QueryBool("flagged_only", false) {
		q = q.Where("cr.flagged = ?", true)
	}
	var rows []reviewRow
	if err := q.Order("cr.created_at DESC").Limit(limit).Offset(offset).Scan(&rows).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, rows, "Reviews fetched")
}

// Stats powers the oversight header KPIs.
//
// GET /clinical/stats
func (h *Handler) Stats(c *fiber.Ctx) error {
	ids, scoped := h.scopedDoctorIDs(c)

	pendingQ := h.db.Table("appointments AS a").
		Where("a.status = ?", "completed").
		Where("NOT EXISTS (SELECT 1 FROM consultation_reviews cr WHERE cr.appointment_id = a.id)")
	reviewedQ := h.db.Model(&ConsultationReview{})
	flaggedQ := h.db.Model(&ConsultationReview{}).Where("flagged = ?", true)
	avgQ := h.db.Model(&ConsultationReview{}).Select("COALESCE(AVG(quality_score), 0)")
	if scoped {
		pendingQ = pendingQ.Where("a.doctor_id IN ?", ids)
		reviewedQ = reviewedQ.Where("doctor_id IN ?", ids)
		flaggedQ = flaggedQ.Where("doctor_id IN ?", ids)
		avgQ = avgQ.Where("doctor_id IN ?", ids)
	}
	var pending, reviewed, flagged int64
	var avgQuality float64
	pendingQ.Count(&pending)
	reviewedQ.Count(&reviewed)
	flaggedQ.Count(&flagged)
	avgQ.Scan(&avgQuality)

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"pending_review": pending,
		"reviewed_total": reviewed,
		"flagged_total":  flagged,
		"avg_quality":    avgQuality,
	}, "Stats fetched")
}

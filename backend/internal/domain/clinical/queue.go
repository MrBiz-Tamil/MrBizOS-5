package clinical

// queue.go — the high-throughput Verification Queue for online (freelance) doctor
// onboarding. The platform hires 100+ AYUSH freelancers a day; the per-record
// modal flow in onboarding.go does not scale to that with a 20-person team. This
// file adds three read/act endpoints on top of the SAME doctor_onboardings record:
//
//   GET  /clinical/onboarding/queue          — a worklist (SLA-aged, pod-scoped)
//   GET  /clinical/onboarding/queue/summary  — KPI strip (new today, overdue, …)
//   POST /clinical/onboarding/bulk           — recommend/activate many at once
//
// All date math is done in Go (not SQL) so the same code runs on SQLite (dev) and
// PostgreSQL (prod) without DB-specific date functions.

import (
	"log"
	"time"

	"drayya-core/internal/services/email"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// slaHours is the age (since last update) past which a record is "overdue" — the
// thing the team should attack first.
const slaHours = 24

// markDoctorVerified is the single source of truth for taking an onboarded doctor
// live. It flips BOTH is_verified flags — doctorprofile (drives search visibility
// and the head-doctor pending list) and user (drives the admin dashboard's
// verified/pending counters) — so every activation path stays consistent, and
// notifies the doctor. Run inside the caller's transaction for the DB writes;
// the email is fire-and-forget.
func markDoctorVerified(tx *gorm.DB, doctorID uint) error {
	if err := tx.Table("doctorprofile").Where("user_id = ?", doctorID).
		Update("is_verified", true).Error; err != nil {
		return err
	}
	return tx.Table("user").Where("id = ?", doctorID).
		Update("is_verified", true).Error
}

// notifyDoctorVerified emails the doctor that they're live and can start
// consulting. Best-effort — never blocks or fails the activation.
func notifyDoctorVerified(db *gorm.DB, doctorID uint) {
	var u struct {
		Email    string
		FullName string
	}
	if err := db.Table("user").Select("email, full_name").
		Where("id = ?", doctorID).Scan(&u).Error; err != nil || u.Email == "" {
		return
	}
	if err := email.SendSimple(u.Email,
		"Your Dr.Ayya Profile Has Been Verified ✅",
		email.DoctorProfileVerifiedEmail(u.FullName),
	); err != nil {
		log.Printf("[clinical] verification email failed for doctor %d: %v", doctorID, err)
	}
}

// queueItem is one onboarding record enriched with the context a reviewer needs
// to act without opening it: who, which AYUSH pod, how stale, and document
// readiness. Stage gates are included so the UI can show progress and the bulk
// endpoint can be driven from the same payload.
type queueItem struct {
	ID                     uint   `json:"id"`
	DoctorID               uint   `json:"doctor_id"`
	DoctorName             string `json:"doctor_name"`
	Email                  string `json:"email"`
	Role                   string `json:"role"`
	Stage                  string `json:"stage"`
	ClinicalRecommendation string `json:"clinical_recommendation"`

	LicenseVerified  bool `json:"license_verified"`
	IdentityVerified bool `json:"identity_verified"`
	AgreementSigned  bool `json:"agreement_signed"`
	OrientationDone  bool `json:"orientation_done"`
	TestConsultDone  bool `json:"test_consult_done"`

	AYUSHSystem string `json:"ayush_system"` // catalog DoctorType name (Siddha, Ayurveda, …)
	PodID       *uint  `json:"pod_id"`
	PodName     string `json:"pod_name"`

	DocsTotal    int `json:"docs_total"`
	DocsVerified int `json:"docs_verified"`
	DocsPending  int `json:"docs_pending"`

	// LicenseMismatch is 1 when the license number on the doctor's profile differs
	// from the one on their original application — a credentialing red flag the
	// head doctor should resolve before going live. Kept as int (0/1) so the CASE
	// expression scans cleanly on both SQLite and PostgreSQL.
	LicenseMismatch int `json:"license_mismatch"`

	AppliedAt *time.Time `json:"applied_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	// Derived (filled in Go).
	AgeHours int  `json:"age_hours"`
	Overdue  bool `json:"overdue"`
}

// VerificationQueue returns the onboarding worklist, oldest-first by default so
// the team always works the SLA backlog. Pod-scoped for head doctors.
//
// GET /clinical/onboarding/queue?stage=&ayush=&overdue=1&q=&limit=
func (h *Handler) VerificationQueue(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 100)
	if limit > 500 {
		limit = 500
	}

	q := h.db.Table("doctor_onboardings AS o").
		Select(`o.*,
			du.full_name AS doctor_name, du.email AS email, du.role AS role,
			COALESCE(dt.name, dp.specialization, '') AS ayush_system,
			dp.pod_id AS pod_id, COALESCE(p.name, '') AS pod_name,
			ja.applied_at AS applied_at,
			CASE WHEN COALESCE(ja.license_number, '') <> '' AND COALESCE(ja.license_number, '') <> COALESCE(dp.license_number, '') THEN 1 ELSE 0 END AS license_mismatch,
			(SELECT COUNT(*) FROM documents d WHERE d.owner_type = 'doctor' AND d.owner_id = o.doctor_id) AS docs_total,
			(SELECT COUNT(*) FROM documents d WHERE d.owner_type = 'doctor' AND d.owner_id = o.doctor_id AND d.status = 'verified') AS docs_verified,
			(SELECT COUNT(*) FROM documents d WHERE d.owner_type = 'doctor' AND d.owner_id = o.doctor_id AND d.status = 'pending') AS docs_pending`).
		Joins(`JOIN "user" du ON du.id = o.doctor_id`).
		Joins(`LEFT JOIN doctorprofile dp ON dp.user_id = o.doctor_id`).
		Joins(`LEFT JOIN doctor_types dt ON dt.id = dp.doctor_type_id`).
		Joins(`LEFT JOIN pods p ON p.id = dp.pod_id`).
		Joins(`LEFT JOIN job_applications ja ON ja.onboarded_user_id = o.doctor_id`)

	if stage := c.Query("stage"); stage != "" {
		q = q.Where("o.stage = ?", stage)
	}
	if ayush := c.Query("ayush"); ayush != "" {
		q = q.Where("dt.slug = ? OR dt.name = ?", ayush, ayush)
	}
	if term := c.Query("q"); term != "" {
		like := "%" + term + "%"
		q = q.Where("LOWER(du.full_name) LIKE LOWER(?) OR LOWER(du.email) LIKE LOWER(?)", like, like)
	}
	if ids, scoped := h.scopedDoctorIDs(c); scoped {
		q = q.Where("o.doctor_id IN ?", ids)
	}

	var rows []queueItem
	if err := q.Order("o.updated_at ASC").Limit(limit).Scan(&rows).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "QUEUE_FAILED", err.Error())
	}

	// Derive age + overdue in Go, and apply the overdue filter here (kept out of
	// SQL so the cutoff math is DB-agnostic).
	now := time.Now()
	overdueOnly := c.QueryBool("overdue", false)
	out := make([]queueItem, 0, len(rows))
	for i := range rows {
		r := rows[i]
		r.AgeHours = int(now.Sub(r.UpdatedAt).Hours())
		r.Overdue = r.AgeHours >= slaHours && r.Stage != StageActive && r.Stage != StageRejected
		if overdueOnly && !r.Overdue {
			continue
		}
		out = append(out, r)
	}
	return response.JSON(c, fiber.StatusOK, out, "Verification queue fetched")
}

// QueueSummary powers the KPI strip: pipeline volume + the SLA signals the team
// is graded on. Pod-scoped for head doctors.
//
// GET /clinical/onboarding/queue/summary
func (h *Handler) QueueSummary(c *fiber.Ctx) error {
	ids, scoped := h.scopedDoctorIDs(c)
	scopeIn := func(db *gorm.DB) *gorm.DB {
		if scoped {
			return db.Where("doctor_id IN ?", ids)
		}
		return db
	}

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	overdueCutoff := now.Add(-slaHours * time.Hour)

	byStage := map[string]int64{}
	type sc struct {
		Stage string
		N     int64
	}
	var counts []sc
	scopeIn(h.db.Model(&DoctorOnboarding{})).Select("stage, COUNT(*) AS n").Group("stage").Scan(&counts)
	var inPipeline int64
	for _, c := range counts {
		byStage[c.Stage] = c.N
		if c.Stage != StageActive && c.Stage != StageRejected {
			inPipeline += c.N
		}
	}

	var newToday, overdue, readyToActivate, activatedToday int64
	scopeIn(h.db.Model(&DoctorOnboarding{})).Where("created_at >= ?", startOfDay).Count(&newToday)
	scopeIn(h.db.Model(&DoctorOnboarding{})).
		Where("updated_at < ? AND stage NOT IN ?", overdueCutoff, []string{StageActive, StageRejected}).
		Count(&overdue)
	scopeIn(h.db.Model(&DoctorOnboarding{})).Where("stage = ?", StageGoLive).Count(&readyToActivate)
	scopeIn(h.db.Model(&DoctorOnboarding{})).Where("stage = ? AND activated_at >= ?", StageActive, startOfDay).Count(&activatedToday)

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"by_stage":          byStage,
		"in_pipeline":       inPipeline,
		"new_today":         newToday,
		"overdue":           overdue,
		"ready_to_activate": readyToActivate,
		"activated_today":   activatedToday,
	}, "Summary fetched")
}

// BulkAction applies one action to many onboarding records in a single request,
// turning "open 100 modals" into "select and clear". It NEVER weakens a gate:
// activate still refuses any record not at go_live (reusing the same rule as the
// single-record ActivateDoctor). Returns a per-id result so the UI can report
// exactly what advanced and what was skipped.
//
// POST /clinical/onboarding/bulk  { ids: [], action: "recommend" | "activate", notes }
func (h *Handler) BulkAction(c *fiber.Ctx) error {
	actorID, _ := c.Locals("user_id").(uint)
	var req struct {
		IDs    []uint `json:"ids"`
		Action string `json:"action"`
		Notes  string `json:"notes"`
	}
	if err := c.BodyParser(&req); err != nil || len(req.IDs) == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "ids and action are required")
	}
	if req.Action != "recommend" && req.Action != "activate" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ACTION", "action must be recommend or activate")
	}

	// Pod scope — a head doctor may only act on doctors in their pod.
	scopeIDs, scoped := h.scopedDoctorIDs(c)
	allowed := map[uint]bool{}
	if scoped {
		for _, id := range scopeIDs {
			allowed[id] = true
		}
	}

	type result struct {
		ID      uint   `json:"id"`
		OK      bool   `json:"ok"`
		Stage   string `json:"stage,omitempty"`
		Skipped string `json:"skipped,omitempty"`
	}
	results := make([]result, 0, len(req.IDs))
	okCount := 0

	for _, id := range req.IDs {
		var rec DoctorOnboarding
		if err := h.db.First(&rec, id).Error; err != nil {
			results = append(results, result{ID: id, Skipped: "not found"})
			continue
		}
		if scoped && !allowed[rec.DoctorID] {
			results = append(results, result{ID: id, Skipped: "out of pod scope"})
			continue
		}
		if rec.Stage == StageRejected {
			results = append(results, result{ID: id, Skipped: "rejected"})
			continue
		}

		switch req.Action {
		case "recommend":
			if rec.ClinicalRecommendation == "recommended" {
				results = append(results, result{ID: id, OK: true, Stage: rec.Stage, Skipped: "already recommended"})
				continue
			}
			rec.ClinicalRecommendation = "recommended"
			rec.ClinicalReviewerID = &actorID
			if req.Notes != "" {
				rec.ClinicalNotes = req.Notes
			}
			rec.Stage = computeStage(&rec)
			if err := h.db.WithContext(c.UserContext()).Save(&rec).Error; err != nil {
				results = append(results, result{ID: id, Skipped: "save failed"})
				continue
			}
			okCount++
			results = append(results, result{ID: id, OK: true, Stage: rec.Stage})

		case "activate":
			if rec.Stage == StageActive {
				results = append(results, result{ID: id, OK: true, Stage: rec.Stage, Skipped: "already active"})
				continue
			}
			if computeStage(&rec) != StageGoLive {
				results = append(results, result{ID: id, Skipped: "gates incomplete"})
				continue
			}
			now := time.Now()
			rec.Stage = StageActive
			rec.ActivatedAt = &now
			rec.ActivatedBy = &actorID
			err := h.db.WithContext(c.UserContext()).Transaction(func(tx *gorm.DB) error {
				if e := tx.Save(&rec).Error; e != nil {
					return e
				}
				return markDoctorVerified(tx, rec.DoctorID)
			})
			if err != nil {
				results = append(results, result{ID: id, Skipped: "activate failed"})
				continue
			}
			go notifyDoctorVerified(h.db, rec.DoctorID)
			okCount++
			results = append(results, result{ID: id, OK: true, Stage: rec.Stage})
		}
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"succeeded": okCount,
		"total":     len(req.IDs),
		"results":   results,
	}, "Bulk action applied")
}

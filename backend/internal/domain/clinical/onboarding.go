package clinical

// onboarding.go — doctor onboarding pipeline, owned by the chief doctor.
//
// This is the clinical/go-live side of bringing a new online doctor on board —
// distinct from HR's generic staff onboarding (/hr/onboarding) which handles
// documents, payroll, and orientation for ALL staff. Here the chief doctor runs
// the clinical interview round, credentialing, and the final go-live activation
// that flips the doctor's profile to verified.
//
// Stage flow:
//   clinical_review → credentialing → orientation → go_live → active
//                  ↘ rejected
import (
	"log"
	"strconv"
	"strings"
	"time"

	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Onboarding stages.
const (
	StageClinicalReview = "clinical_review"
	StageCredentialing  = "credentialing"
	StageOrientation    = "orientation"
	StageGoLive         = "go_live"
	StageActive         = "active"
	StageRejected       = "rejected"
)

// DoctorOnboarding tracks one new doctor from clinical interview to go-live.
type DoctorOnboarding struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	DoctorID uint   `gorm:"uniqueIndex;not null" json:"doctor_id"` // user id of the doctor
	Stage    string `gorm:"default:'clinical_review';index" json:"stage"`

	// Clinical interview round (head-doctor recommendation).
	ClinicalRecommendation string `gorm:"default:'pending'" json:"clinical_recommendation"` // pending | recommended | rejected
	ClinicalReviewerID     *uint  `json:"clinical_reviewer_id,omitempty"`
	ClinicalNotes          string `gorm:"type:text" json:"clinical_notes"`

	// Credentialing checklist.
	LicenseVerified  bool `gorm:"default:false" json:"license_verified"`
	IdentityVerified bool `gorm:"default:false" json:"identity_verified"`
	AgreementSigned  bool `gorm:"default:false" json:"agreement_signed"`

	// Orientation checklist.
	OrientationDone bool `gorm:"default:false" json:"orientation_done"`
	TestConsultDone bool `gorm:"default:false" json:"test_consult_done"`

	// Go-live.
	ActivatedAt *time.Time `json:"activated_at,omitempty"`
	ActivatedBy *uint      `json:"activated_by,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (DoctorOnboarding) TableName() string { return "doctor_onboardings" }

// computeStage derives the current stage from the checklist, so the pipeline
// advances automatically as boxes get ticked. Never downgrades a rejected or
// already-active record.
func computeStage(o *DoctorOnboarding) string {
	if o.Stage == StageRejected || o.Stage == StageActive {
		return o.Stage
	}
	if o.ClinicalRecommendation != "recommended" {
		return StageClinicalReview
	}
	if !(o.LicenseVerified && o.IdentityVerified && o.AgreementSigned) {
		return StageCredentialing
	}
	if !(o.OrientationDone && o.TestConsultDone) {
		return StageOrientation
	}
	return StageGoLive
}

// onboardingRow is a list item with the doctor's name attached.
type onboardingRow struct {
	DoctorOnboarding
	DoctorName string `json:"doctor_name"`
	Email      string `json:"email"`
	Role       string `json:"role"`
}

// StartOnboarding opens an onboarding record for a doctor.
//
// POST /clinical/onboarding/start  { doctor_id }
func (h *Handler) StartOnboarding(c *fiber.Ctx) error {
	var req struct {
		DoctorID uint `json:"doctor_id"`
	}
	if err := c.BodyParser(&req); err != nil || req.DoctorID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "doctor_id is required")
	}
	// Must be a real doctor user.
	var u struct{ Role string }
	if err := h.db.Table("user").Select("role").Where("id = ?", req.DoctorID).Scan(&u).Error; err != nil || u.Role == "" {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "User not found")
	}
	if u.Role != "doctor" && u.Role != "chief_doctor" && u.Role != "hospital_head_doctor" {
		return response.Error(c, fiber.StatusUnprocessableEntity, "NOT_A_DOCTOR", "Onboarding is for doctor/head doctor accounts")
	}
	var existing int64
	h.db.Model(&DoctorOnboarding{}).Where("doctor_id = ?", req.DoctorID).Count(&existing)
	if existing > 0 {
		return response.Error(c, fiber.StatusConflict, "ALREADY_STARTED", "Onboarding already exists for this doctor")
	}
	rec := DoctorOnboarding{DoctorID: req.DoctorID, Stage: StageClinicalReview}
	if err := h.db.WithContext(c.UserContext()).Create(&rec).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, rec, "Onboarding started")
}

// OnboardingCandidates returns doctor accounts that are NOT yet in the clinical
// onboarding pipeline — searchable by name/email. Unverified doctors surface
// first since they're the ones who still need to go live. This is the natural
// hire→onboard bridge: a doctor created by the career/HR hiring flow shows up
// here automatically.
//
// GET /clinical/onboarding/candidates?q=
func (h *Handler) OnboardingCandidates(c *fiber.Ctx) error {
	type candidate struct {
		ID         uint   `json:"id"`
		Name       string `json:"name"`
		Email      string `json:"email"`
		IsVerified bool   `json:"is_verified"`
		Role       string `json:"role"`
	}
	// NOTE: "user" must be double-quoted in PostgreSQL because it is a reserved
	// keyword. Without quotes the alias 'u' fails with "column u.id does not exist".
	q := h.db.Table(`"user" AS u`).
		Select(`u.id AS id, u.full_name AS name, u.email AS email, u.role AS role, COALESCE(dp.is_verified, false) AS is_verified`).
		Joins(`LEFT JOIN doctorprofile dp ON dp.user_id = u.id`).
		Where("u.role IN ?", []string{"doctor", "chief_doctor", "hospital_head_doctor"}).
		Where(`NOT EXISTS (SELECT 1 FROM doctor_onboardings o WHERE o.doctor_id = u.id)`)
	if term := strings.TrimSpace(c.Query("q")); term != "" {
		like := "%" + strings.ToLower(term) + "%"
		q = q.Where("LOWER(u.full_name) LIKE ? OR LOWER(u.email) LIKE ?", like, like)
	}
	if ids, scoped := h.scopedDoctorIDs(c); scoped {
		q = q.Where("u.id IN ?", ids)
	}
	var rows []candidate
	if err := q.Order("is_verified ASC, u.full_name ASC").Limit(20).Scan(&rows).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, rows, "Candidates fetched")
}

// OnboardingApplication returns the original job application behind an onboarding
// record, so the head doctor has the clinical context (resume, education,
// treatment approach, AYUSH system, experience, license) needed to make the
// clinical-review decision. This is the read side of SEAM 1 — the application is
// owned by HR but surfaced read-only to the chief doctor for the review round.
//
// GET /clinical/onboarding/:id/application
func (h *Handler) OnboardingApplication(c *fiber.Ctx) error {
	rec, err := h.loadOnboarding(c)
	if err != nil {
		return err
	}
	// Pod scoping — a head doctor only sees applications for doctors in their pod.
	if ids, scoped := h.scopedDoctorIDs(c); scoped {
		allowed := false
		for _, id := range ids {
			if id == rec.DoctorID {
				allowed = true
				break
			}
		}
		if !allowed {
			return response.Error(c, fiber.StatusForbidden, "OUT_OF_SCOPE", "Doctor is not in your pod")
		}
	}
	type appView struct {
		ID                uint   `json:"id"`
		FullName          string `json:"full_name"`
		Email             string `json:"email"`
		PhoneNumber       string `json:"phone_number"`
		Position          string `json:"position"`
		ExperienceYears   int    `json:"experience_years"`
		LicenseNumber     string `json:"license_number"`
		AYUSHSystem       string `json:"ayush_system"`
		TreatmentApproach string `json:"treatment_approach"`
		EducationDetails  string `json:"education_details"`
		ResumeURL         string `json:"resume_url"`
		CoverLetter       string `json:"cover_letter"`
		Status            string `json:"status"`
		InterviewNotes    string `json:"interview_notes"`
		AppliedAt         string `json:"applied_at"`
	}
	var view appView
	err = h.db.Table("job_applications").
		Select(`id, full_name, email, phone_number, position, experience_years,
			license_number, ayush_system, treatment_approach, education_details,
			resume_url, cover_letter, status, interview_notes, applied_at`).
		Where("onboarded_user_id = ?", rec.DoctorID).
		Order("updated_at DESC").
		Limit(1).
		Scan(&view).Error
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	if view.ID == 0 {
		return response.Error(c, fiber.StatusNotFound, "NO_APPLICATION",
			"No source application linked to this onboarding record")
	}
	return response.JSON(c, fiber.StatusOK, view, "Application fetched")
}

// ListOnboarding returns the pipeline, optionally filtered by stage.
//
// GET /clinical/onboarding?stage=
func (h *Handler) ListOnboarding(c *fiber.Ctx) error {
	q := h.db.Table("doctor_onboardings AS o").
		Select(`o.*, du.full_name AS doctor_name, du.email AS email, du.role AS role`).
		Joins(`JOIN "user" du ON du.id = o.doctor_id`)
	if stage := c.Query("stage"); stage != "" {
		q = q.Where("o.stage = ?", stage)
	}
	if ids, scoped := h.scopedDoctorIDs(c); scoped {
		q = q.Where("o.doctor_id IN ?", ids)
	}
	var rows []onboardingRow
	if err := q.Order("o.updated_at DESC").Limit(200).Scan(&rows).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	// Pipeline counts for the board header.
	type sc struct {
		Stage string
		N     int64
	}
	var counts []sc
	h.db.Model(&DoctorOnboarding{}).Select("stage, COUNT(*) AS n").Group("stage").Scan(&counts)
	byStage := map[string]int64{}
	for _, c := range counts {
		byStage[c.Stage] = c.N
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"items": rows, "counts": byStage}, "Onboarding fetched")
}

// ClinicalReview records the head-doctor's interview decision (the clinical
// round after HR's screen). Recommending advances to credentialing; rejecting
// closes the record.
//
// POST /clinical/onboarding/:id/clinical-review  { recommendation, notes }
func (h *Handler) ClinicalReview(c *fiber.Ctx) error {
	reviewerID, _ := c.Locals("user_id").(uint)
	rec, err := h.loadOnboarding(c)
	if err != nil {
		return err
	}
	var req struct {
		Recommendation string `json:"recommendation"` // recommended | rejected
		Notes          string `json:"notes"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	if req.Recommendation != "recommended" && req.Recommendation != "rejected" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_RECOMMENDATION", "recommendation must be recommended or rejected")
	}
	rec.ClinicalRecommendation = req.Recommendation
	rec.ClinicalReviewerID = &reviewerID
	rec.ClinicalNotes = req.Notes
	if req.Recommendation == "rejected" {
		rec.Stage = StageRejected
	} else {
		rec.Stage = computeStage(rec)
	}
	if err := h.db.WithContext(c.UserContext()).Save(rec).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, rec, "Clinical review recorded")
}

// UpdateChecklist ticks credentialing/orientation boxes and recomputes the stage.
//
// PATCH /clinical/onboarding/:id/checklist
func (h *Handler) UpdateChecklist(c *fiber.Ctx) error {
	rec, err := h.loadOnboarding(c)
	if err != nil {
		return err
	}
	if rec.Stage == StageRejected || rec.Stage == StageActive {
		return response.Error(c, fiber.StatusUnprocessableEntity, "LOCKED", "Onboarding is closed for edits")
	}
	var req struct {
		LicenseVerified  *bool `json:"license_verified"`
		IdentityVerified *bool `json:"identity_verified"`
		AgreementSigned  *bool `json:"agreement_signed"`
		OrientationDone  *bool `json:"orientation_done"`
		TestConsultDone  *bool `json:"test_consult_done"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	if req.LicenseVerified != nil {
		rec.LicenseVerified = *req.LicenseVerified
	}
	if req.IdentityVerified != nil {
		rec.IdentityVerified = *req.IdentityVerified
	}
	if req.AgreementSigned != nil {
		rec.AgreementSigned = *req.AgreementSigned
	}
	if req.OrientationDone != nil {
		rec.OrientationDone = *req.OrientationDone
	}
	if req.TestConsultDone != nil {
		rec.TestConsultDone = *req.TestConsultDone
	}
	rec.Stage = computeStage(rec)
	if err := h.db.WithContext(c.UserContext()).Save(rec).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, rec, "Checklist updated")
}

// ActivateDoctor takes a fully-onboarded doctor live: verifies their profile so
// they can appear and consult. Requires every gate to be complete.
//
// POST /clinical/onboarding/:id/activate
func (h *Handler) ActivateDoctor(c *fiber.Ctx) error {
	activatorID, _ := c.Locals("user_id").(uint)
	rec, err := h.loadOnboarding(c)
	if err != nil {
		return err
	}
	if rec.Stage == StageActive {
		return response.Error(c, fiber.StatusConflict, "ALREADY_ACTIVE", "Doctor is already active")
	}
	if computeStage(rec) != StageGoLive {
		return response.Error(c, fiber.StatusUnprocessableEntity, "NOT_READY",
			"Complete the clinical review, credentialing, and orientation before going live")
	}
	now := time.Now()
	rec.Stage = StageActive
	rec.ActivatedAt = &now
	rec.ActivatedBy = &activatorID

	// Flip both is_verified flags (profile + user) so search visibility and the
	// admin dashboard counters agree, and notify the doctor. markDoctorVerified is
	// shared with the bulk path so single and bulk activation never diverge. Done
	// in the request context so the audit log attributes it.
	err = h.db.WithContext(c.UserContext()).Transaction(func(tx *gorm.DB) error {
		if e := tx.Save(rec).Error; e != nil {
			return e
		}
		return markDoctorVerified(tx, rec.DoctorID)
	})
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ACTIVATE_FAILED", err.Error())
	}
	go notifyDoctorVerified(h.db, rec.DoctorID)
	return response.JSON(c, fiber.StatusOK, rec, "Doctor activated — now live")
}

// BackfillOnboarding sweeps existing unverified doctors who predate the clinical
// pipeline into the verification queue, so they're not stranded invisible after
// the one-click verify bypass was closed. It opens a clinical_review record for
// every role='doctor' user whose profile is unverified and who has no onboarding
// record yet. Idempotent (the NOT EXISTS guard) so it is safe to run on every
// startup; verified/live doctors are left alone.
func BackfillOnboarding(db *gorm.DB) {
	// "user" is a reserved word — quote it (works on SQLite and PostgreSQL).
	// Booleans rely on the table's GORM column defaults (false); timestamps are
	// set explicitly since GORM auto-timestamps don't apply to raw INSERT…SELECT.
	if err := db.Exec(`
		INSERT INTO doctor_onboardings (doctor_id, stage, clinical_recommendation, clinical_notes, created_at, updated_at)
		SELECT u.id, ?, 'pending', 'Backfilled from existing unverified doctor.', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		FROM "user" u
		JOIN doctorprofile dp ON dp.user_id = u.id
		WHERE u.role = 'doctor' AND dp.is_verified = false
		AND NOT EXISTS (SELECT 1 FROM doctor_onboardings o WHERE o.doctor_id = u.id)
	`, StageClinicalReview).Error; err != nil {
		// Non-fatal — log and continue; the queue still works for new hires.
		log.Printf("[clinical] BackfillOnboarding failed: %v", err)
	}
}

// loadOnboarding fetches the onboarding record from the :id param.
func (h *Handler) loadOnboarding(c *fiber.Ctx) (*DoctorOnboarding, error) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return nil, response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid onboarding ID")
	}
	var rec DoctorOnboarding
	if err := h.db.First(&rec, id).Error; err != nil {
		return nil, response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Onboarding record not found")
	}
	return &rec, nil
}

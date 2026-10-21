package handlers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"drayya-core/internal/auth"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/career"
	"drayya-core/internal/domain/clinical"
	"drayya-core/internal/domain/document"
	"drayya-core/internal/domain/hr"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/middleware"
	"drayya-core/internal/services/email"
	"drayya-core/internal/services/notification"
	"drayya-core/internal/utils/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

var (
	emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	phoneRe = regexp.MustCompile(`^\+?[0-9\s\-]{7,15}$`)
)

type CareerHandler struct {
	service career.Service
}

func NewCareerHandler(service career.Service) *CareerHandler {
	return &CareerHandler{service: service}
}

// GET /api/v1/careers  — public list of active job openings
func (h *CareerHandler) GetCareers(c *fiber.Ctx) error {
	jobs, err := h.service.ListJobs()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch careers", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, jobs, "Job openings fetched")
}

// GET /api/v1/careers/:id — public details of a specific job opening
func (h *CareerHandler) GetJob(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid ID", err.Error())
	}
	job, err := h.service.GetJobByID(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Job not found", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, job, "Job fetched")
}

// POST /api/v1/careers/apply  or  POST /api/v1/careers/:id/apply
// Accepts the full doctor/staff application payload from the frontend form.
func (h *CareerHandler) ApplyForJob(c *fiber.Ctx) error {
	var app career.JobApplication
	if err := c.BodyParser(&app); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body", err.Error())
	}

	// Server-side format validation
	app.Email = strings.TrimSpace(app.Email)
	app.PhoneNumber = strings.TrimSpace(app.PhoneNumber)
	if !emailRe.MatchString(app.Email) {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_EMAIL", "Please enter a valid email address")
	}
	if app.PhoneNumber != "" && !phoneRe.MatchString(app.PhoneNumber) {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_PHONE", "Please enter a valid phone number")
	}
	if app.WhatsappNumber == "" {
		app.WhatsappNumber = app.PhoneNumber
	}

	// URL param takes priority; fallback to body's job_id
	if idRaw := c.Params("id"); idRaw != "" {
		if id, err := strconv.Atoi(idRaw); err == nil {
			uid := uint(id)
			app.JobID = &uid
		}
	}
	// Also accept ?job_id= query param (from the landing page's Apply button)
	if app.JobID == nil {
		if idRaw := c.Query("job_id"); idRaw != "" {
			if id, err := strconv.Atoi(idRaw); err == nil {
				uid := uint(id)
				app.JobID = &uid
			}
		}
	}

	// Attach logged-in user if available (not required — public route)
	if userID, ok := c.Locals("user_id").(uint); ok && userID > 0 {
		app.UserID = &userID
	}

	if err := h.service.Apply(&app); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Application failed", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, fiber.Map{
		"id":         app.ID,
		"position":   app.Position,
		"applied_at": app.AppliedAt,
		"status":     app.Status,
	}, "Application submitted successfully")
}

// GET /api/v1/careers/my-applications  — requires auth
func (h *CareerHandler) GetMyApplications(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", "Login required")
	}
	apps, err := h.service.MyApplications(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch applications", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, apps, "My applications fetched")
}

// GET /api/v1/careers/admin/applications  — admin only
func (h *CareerHandler) AdminListApplications(c *fiber.Ctx) error {
	apps, err := h.service.ListAllApplications()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch applications", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, apps, "All applications fetched")
}

// PATCH /api/v1/careers/admin/applications/:id/status  — admin only
func (h *CareerHandler) AdminUpdateStatus(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid ID", err.Error())
	}
	var body struct {
		Status        string `json:"status"`
		InterviewDate string `json:"interview_date"`
		Notes         string `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil || body.Status == "" {
		return response.Error(c, fiber.StatusBadRequest, "status field required", "")
	}
	valid := map[string]bool{"applied": true, "reviewing": true, "interviewing": true, "offered": true, "rejected": true}
	if !valid[body.Status] {
		return response.Error(c, fiber.StatusBadRequest, "Invalid status value", "")
	}
	if err := h.service.UpdateApplicationStatusWithNotes(uint(id), body.Status, body.InterviewDate, body.Notes); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Update failed", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"id": id, "status": body.Status}, "Status updated")
}

// POST /api/v1/careers/admin/jobs  — admin only
func (h *CareerHandler) AdminCreateJob(c *fiber.Ctx) error {
	var job career.JobOpening
	if err := c.BodyParser(&job); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body", err.Error())
	}
	if err := h.service.CreateJob(&job); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to create job", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, job, "Job created")
}

// PATCH /api/v1/careers/admin/jobs/:id  — admin only
func (h *CareerHandler) AdminUpdateJob(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid ID", err.Error())
	}
	var job career.JobOpening
	if err := c.BodyParser(&job); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body", err.Error())
	}
	job.ID = uint(id)
	if err := h.service.UpdateJob(&job); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to update job", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, job, "Job updated")
}

// POST /api/v1/careers/admin/applications/:id/onboard  — admin only
// Creates a doctor user account from the application, pre-populates the doctor and
// staff profiles, and emails the new hire their login credentials.
func (h *CareerHandler) AdminOnboardDoctor(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid ID", err.Error())
	}

	app, err := h.service.GetApplicationByID(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Application not found", err.Error())
	}
	if app.OnboardedUserID != nil {
		return response.Error(c, fiber.StatusConflict, "Already onboarded", fmt.Sprintf("linked user_id: %d", *app.OnboardedUserID))
	}
	if app.Status != "offered" {
		return response.Error(c, fiber.StatusBadRequest, "Application must be in 'offered' status to onboard", "current status: "+app.Status)
	}

	// The applicant may already have an account (commonly their own PATIENT
	// account, created when they applied while logged in). Handle both:
	//   - existing patient → promote that account to a doctor (don't duplicate);
	//   - existing non-patient (doctor/staff/admin) → clear conflict, can't convert;
	//   - no existing account → create a fresh doctor account.
	// Without this, RegisterUser hit the user.email UNIQUE constraint and leaked a
	// cryptic 500.
	tempPassword := generateTempPassword()
	var created *identity.User
	promoted := false

	var existingUser identity.User
	if err := database.DB.Where("email = ?", app.Email).First(&existingUser).Error; err == nil {
		if existingUser.Role != string(identity.RolePatient) {
			return response.Error(c, fiber.StatusConflict, "EMAIL_IN_USE",
				fmt.Sprintf("A user already exists with email %s (role: %s). Only a patient account can be promoted to a doctor — resolve this account before onboarding.", app.Email, existingUser.Role))
		}
		// Promote the applicant's existing patient account to a doctor.
		u, err := promoteToDoctor(c.UserContext(), &existingUser)
		if err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "Failed to promote existing account", err.Error())
		}
		created, promoted = u, true
	} else {
		newUser := identity.User{
			FullName:           app.FullName,
			Email:              app.Email,
			PhoneNumber:        app.PhoneNumber,
			WhatsAppNumber:     app.WhatsappNumber,
			Address:            app.PermanentAddress,
			Gender:             app.Gender,
			Role:               string(identity.RoleDoctor),
			IsActive:           true,
			MustChangePassword: true,
		}
		u, err := auth.RegisterUser(c.UserContext(), newUser, tempPassword)
		if err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "Failed to create user account", err.Error())
		}
		created = u
	}

	var edu struct {
		UG struct {
			College string `json:"college"`
		} `json:"ug"`
	}
	collegeName := ""
	if app.EducationDetails != "" {
		_ = json.Unmarshal([]byte(app.EducationDetails), &edu)
		collegeName = edu.UG.College
	}

	// Pre-populate DoctorProfile from application data so the doctor doesn't re-enter it.
	profileUpdates := map[string]interface{}{
		"license_number":      app.LicenseNumber,
		"years_of_experience": app.ExperienceYears,
		"specialization":      ayushSystemLabel(app.AYUSHSystem),
		"college_name":        collegeName,
	}
	// SEAM 4 — link the doctor to the specialization catalog. The application's
	// ayush_system slug (siddha/ayurveda/yoga_naturopathy) maps directly to a
	// DoctorType. Setting doctor_type_id is what makes the doctor appear in
	// patient search/filtering by AYUSH system, and scopes which sub-specialties
	// the head doctor can later assign. We also mirror the catalog's canonical
	// name into the free-text specialization so the two never drift.
	if app.AYUSHSystem != "" {
		var dt identity.DoctorType
		if err := database.DB.Where("slug = ?", app.AYUSHSystem).First(&dt).Error; err == nil {
			profileUpdates["doctor_type_id"] = dt.ID
			profileUpdates["specialization"] = dt.Name
			// Route the doctor into their AYUSH system's pod so the matching head
			// doctor (the pod's lead) sees them in scoped oversight/onboarding views.
			var podID uint
			if err := database.DB.Table("pods").
				Where("doctor_type_id = ? AND is_active = ?", dt.ID, true).
				Select("id").Scan(&podID).Error; err == nil && podID > 0 {
				profileUpdates["pod_id"] = podID
			}
		}
	}
	database.DB.WithContext(c.UserContext()).Model(&identity.DoctorProfile{}).
		Where("user_id = ?", created.ID).
		Updates(profileUpdates)

	// Create HR OnboardingRecord so the doctor immediately appears in the
	// admin Onboarding Kanban. Previously the recruitment flow created a
	// User + DoctorProfile + StaffProfile but skipped this step, leaving
	// newly-onboarded doctors invisible to HR until manually inserted.
	// Lookup the auto-created StaffProfile to get its ID — that's the FK.
	var staffProfile hr.StaffProfile
	if err := database.DB.Where("user_id = ?", created.ID).First(&staffProfile).Error; err == nil {
		onboardRow := hr.OnboardingRecord{
			StaffID:          staffProfile.ID,
			Status:           "pending_docs",
			ProbationEndDate: time.Now().AddDate(0, 3, 0), // 3-month probation default
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		if err := database.DB.WithContext(c.UserContext()).Create(&onboardRow).Error; err != nil {
			log.Printf("[onboard] warn: OnboardingRecord creation failed for staff %d: %v", staffProfile.ID, err)
		}
	}

	// SEAM 2 — hand the doctor off to the head doctor's clinical pipeline.
	// Onboarding creates the account; the clinical go-live (verification) is the
	// head doctor's job. We open a DoctorOnboarding record at the clinical_review
	// stage so the new hire appears automatically in the chief doctor's onboarding
	// board — previously HR-onboarded doctors were invisible there until manually
	// added. The account stays unverified until the chief doctor activates it.
	clinicalRow := clinical.DoctorOnboarding{
		DoctorID:      created.ID,
		Stage:         clinical.StageClinicalReview,
		ClinicalNotes: fmt.Sprintf("Auto-started from application #%d (%s).", app.ID, app.Position),
	}
	if err := database.DB.WithContext(c.UserContext()).Create(&clinicalRow).Error; err != nil {
		log.Printf("[onboard] warn: clinical onboarding start failed for doctor %d: %v", created.ID, err)
	}

	// Move the applicant's uploaded files (photo, CV) from the public inbox into
	// the new doctor's document vault, so the head doctor can view them during
	// credentialing. Tokens are embedded in the stored /careers/documents/<token>
	// URLs the apply form saved.
	document.ReownToDoctor(database.DB, extractDocTokens(app.PassportPhotoURL, app.ResumeURL), created.ID)

	// Set the correct designation on the auto-created StaffProfile.
	database.DB.WithContext(c.UserContext()).Model(&hr.StaffProfile{}).
		Where("user_id = ?", created.ID).
		Update("designation", app.Position)

	// Initialize basic consultation fees (INR 300 / 30,000 paise standard fee) automatically
	database.DB.WithContext(c.UserContext()).Model(&identity.DoctorProfile{}).
		Where("user_id = ?", created.ID).
		Update("consultation_fees", `{"INR":30000}`)

	// Allocate default leave balances automatically for the current calendar year
	_ = hr.AllocateLeaveBalances(created.ID, time.Now().Year())

	// Atomically mark the application as onboarded and link the new user.
	database.DB.WithContext(c.UserContext()).Model(&career.JobApplication{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":            "onboarded",
			"onboarded_user_id": created.ID,
			"user_id":           created.ID,
			"updated_at":        time.Now(),
		})

	go func() {
		// A promoted patient keeps their existing password — don't send a temp one.
		emailSubject := "Welcome to Dr.Ayya — Your Account is Ready"
		var emailBody string
		if promoted {
			emailSubject = "Your Dr.Ayya account is now a Doctor account"
			emailBody = email.WrapInLayout("Doctor Account Activated", "Welcome to the Dr.Ayya doctor network",
				fmt.Sprintf("<p>Hi %s,</p><p>Your existing Dr.Ayya account (<b>%s</b>) has been upgraded to a <b>Doctor</b> account for the <b>%s</b> role. Sign in with your current password — no new credentials needed.</p><p>Our head doctor will review your credentials and take you live shortly.</p>", app.FullName, app.Email, app.Position))
		} else {
			emailBody = email.DoctorOnboardingWelcomeEmail(email.JobApplicationEmailData{
				FullName: app.FullName,
				Position: app.Position,
				Email:    app.Email,
			}, tempPassword)
		}

		if err := email.SendSimple(app.Email, emailSubject, emailBody); err != nil {
			log.Printf("[onboard] credential email failed for %s: %v", app.Email, err)
		}

		whatsappPhone := app.WhatsappNumber
		if whatsappPhone == "" {
			whatsappPhone = app.PhoneNumber
		}
		if whatsappPhone != "" {
			// Send welcome template
			notification.WhatsAppWelcome(whatsappPhone, app.FullName, "Doctor")

			if promoted {
				_ = notification.SendText(whatsappPhone, fmt.Sprintf("Welcome to Dr.Ayya!\nYour existing account (%s) is now a Doctor account. Sign in with your current password.", app.Email))
				return
			}
			// Send text credentials
			msg := fmt.Sprintf("Welcome to Dr.Ayya!\nYour doctor account is ready.\nLogin Email: %s\nTemporary Password: %s\nPlease change your password on first login.", app.Email, tempPassword)
			
			var waTmpl career.RecruitmentTemplate
			if err := database.DB.Where("type = ? AND title = ?", "WhatsApp", "Doctor Onboarding Welcome").First(&waTmpl).Error; err == nil {
				msg = strings.ReplaceAll(waTmpl.Content, "{{email}}", app.Email)
				msg = strings.ReplaceAll(msg, "{{password}}", tempPassword)
			}
			
			_ = notification.SendText(whatsappPhone, msg)
		}
	}()

	msg := "Doctor onboarded successfully — credentials sent by email"
	if promoted {
		msg = "Existing account promoted to doctor — they sign in with their current password"
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"user_id":     created.ID,
		"employee_id": fmt.Sprintf("DOC-%d", created.ID),
		"email":       created.Email,
		"role":        created.Role,
		"promoted":    promoted,
	}, msg)
}

// extractDocTokens pulls the vault access token out of each /careers/documents/
// <token> URL the apply form stored. Non-vault URLs (legacy pasted links) yield
// nothing and are simply ignored.
func extractDocTokens(urls ...string) []string {
	const marker = "/careers/documents/"
	var tokens []string
	for _, u := range urls {
		i := strings.Index(u, marker)
		if i < 0 {
			continue
		}
		tok := u[i+len(marker):]
		if j := strings.IndexAny(tok, "/?#"); j >= 0 {
			tok = tok[:j]
		}
		if tok != "" {
			tokens = append(tokens, tok)
		}
	}
	return tokens
}

// promoteToDoctor converts an existing patient account into a doctor: flips the
// role and ensures the DoctorProfile + StaffProfile exist (the same artifacts
// RegisterUser creates for a brand-new doctor). The user keeps their existing
// password. Idempotent on the profiles.
func promoteToDoctor(ctx context.Context, u *identity.User) (*identity.User, error) {
	if err := database.DB.WithContext(ctx).Model(&identity.User{}).
		Where("id = ?", u.ID).Update("role", string(identity.RoleDoctor)).Error; err != nil {
		return nil, err
	}
	u.Role = string(identity.RoleDoctor)

	var dpCount int64
	database.DB.Model(&identity.DoctorProfile{}).Where("user_id = ?", u.ID).Count(&dpCount)
	if dpCount == 0 {
		// Temp unique license number to avoid the UNIQUE constraint on empty values.
		tempLic := fmt.Sprintf("TEMP-LIC-%d-%d", u.ID, time.Now().UnixNano())
		if err := database.DB.WithContext(ctx).Create(&identity.DoctorProfile{
			UserID: u.ID, IsVerified: false, AdminBoostScore: 1.0, LicenseNumber: tempLic,
		}).Error; err != nil {
			return nil, err
		}
	}

	var spCount int64
	database.DB.Model(&hr.StaffProfile{}).Where("user_id = ?", u.ID).Count(&spCount)
	if spCount == 0 {
		if err := database.DB.WithContext(ctx).Create(&hr.StaffProfile{
			UserID: u.ID, EmployeeID: fmt.Sprintf("DOC-%d", u.ID), Department: "Medical",
			Designation: "Doctor", JoiningDate: time.Now(), EmploymentStatus: "active",
		}).Error; err != nil {
			log.Printf("[onboard] warn: StaffProfile creation failed for promoted user %d: %v", u.ID, err)
		}
	}
	return u, nil
}

// ayushSystemLabel maps the stored enum slug to a display-friendly label.
func ayushSystemLabel(slug string) string {
	labels := map[string]string{
		"siddha":            "Siddha (BSMS)",
		"ayurveda":          "Ayurveda (BAMS)",
		"unani":             "Unani (BUMS)",
		"yoga_naturopathy":  "Yoga & Naturopathy (BNYS)",
		"homeopathy":        "Homeopathy (BHMS)",
		"sowa_rigpa":        "Sowa-Rigpa (BSRMS)",
	}
	if label, ok := labels[slug]; ok {
		return label
	}
	return slug
}

// generateTempPassword returns a 12-character random alphanumeric password.
func generateTempPassword() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789"
	b := make([]byte, 12)
	rand.Read(b) //nolint:errcheck — crypto/rand.Read never returns an error on supported platforms
	for i, v := range b {
		b[i] = chars[int(v)%len(chars)]
	}
	return string(b)
}

// GET /api/v1/careers/admin/jobs  — admin only (all jobs including inactive)
func (h *CareerHandler) AdminGetAllJobs(c *fiber.Ctx) error {
	jobs, err := h.service.GetAllJobs()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch jobs", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, jobs, "All jobs fetched")
}

// GET /api/v1/careers/admin/templates
func (h *CareerHandler) AdminListTemplates(c *fiber.Ctx) error {
	templates, err := h.service.ListTemplates()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch templates", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, templates, "All templates fetched")
}

// GET /api/v1/admin/email-templates  — returns only Type="Email" or Type="WhatsApp" templates
func (h *CareerHandler) ListEmailTemplates(c *fiber.Ctx) error {
	allTemplates, err := h.service.ListTemplates()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch templates", err.Error())
	}
	var resTemplates []career.RecruitmentTemplate
	for _, t := range allTemplates {
		if t.Type == "Email" || t.Type == "WhatsApp" {
			resTemplates = append(resTemplates, t)
		}
	}
	if resTemplates == nil {
		resTemplates = []career.RecruitmentTemplate{}
	}
	return response.JSON(c, fiber.StatusOK, resTemplates, "Templates fetched")
}

// GET /api/v1/admin/email-templates/:id  — get a single template
func (h *CareerHandler) GetEmailTemplate(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid ID", err.Error())
	}
	t, err := h.service.GetTemplateByID(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Template not found", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, t, "Template fetched")
}

// PUT /api/v1/admin/email-templates/:id  — update subject/content
func (h *CareerHandler) UpdateEmailTemplate(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid ID", err.Error())
	}
	var body struct {
		Content string `json:"content"`
		Title   string `json:"title"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body", err.Error())
	}
	t, err := h.service.GetTemplateByID(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Template not found", err.Error())
	}
	if body.Content != "" {
		t.Content = body.Content
	}
	if err := h.service.UpdateTemplate(t); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to update template", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, t, "Template saved")
}

// POST /api/v1/admin/email-templates/:id/preview  — render template with sample vars, returns HTML string
func (h *CareerHandler) PreviewEmailTemplate(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid ID", err.Error())
	}
	t, err := h.service.GetTemplateByID(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Template not found", err.Error())
	}
	// Accept custom sample vars from the request body
	var sampleVars map[string]string
	if err := c.BodyParser(&sampleVars); err != nil || sampleVars == nil {
		// Use default sample vars based on the template's Variables field
		sampleVars = defaultSampleVars(t.Variables)
	}
	text := renderTemplate(t.Content, sampleVars)
	var html string
	if t.Type == "WhatsApp" {
		formattedText := strings.ReplaceAll(text, "\n", "<br/>")
		html = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
  <style>
    body {
      margin: 0;
      padding: 0;
      background-color: #efeae2;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      display: flex;
      justify-content: center;
      align-items: center;
      min-height: 100vh;
    }
    .chat-container {
      width: 100%%;
      max-width: 450px;
      padding: 24px;
      box-sizing: border-box;
    }
    .message-bubble {
      background-color: #ffffff;
      border-radius: 8px;
      padding: 8px 12px 24px 12px;
      box-shadow: 0 1px 0.5px rgba(0,0,0,0.13);
      position: relative;
      font-size: 14.2px;
      line-height: 1.45;
      color: #111b21;
      word-wrap: break-word;
      max-width: 90%%;
    }
    .message-bubble::before {
      content: "";
      position: absolute;
      left: -8px;
      top: 0;
      width: 8px;
      height: 13px;
      background: linear-gradient(135deg, #ffffff 50%%, transparent 50%%);
    }
    .message-time {
      position: absolute;
      right: 8px;
      bottom: 4px;
      font-size: 11px;
      color: #667781;
      display: flex;
      align-items: center;
      gap: 3px;
    }
    .ticks {
      color: #53bdeb;
      font-weight: bold;
    }
  </style>
</head>
<body>
  <div class="chat-container">
    <div class="message-bubble">
      %s
      <div class="message-time">
        12:00 PM
        <span class="ticks">✓✓</span>
      </div>
    </div>
  </div>
</body>
</html>
`, formattedText)
	} else {
		html = email.WrapInLayout(t.Title, t.Title, text)
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"html": html}, "Preview rendered")
}

// POST /api/v1/admin/email-templates/:id/send-test  — send a real test email or whatsapp message
func (h *CareerHandler) SendTestEmail(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid ID", err.Error())
	}
	var body struct {
		To         string            `json:"to"`
		SampleVars map[string]string `json:"sample_vars"`
	}
	if err := c.BodyParser(&body); err != nil || body.To == "" {
		return response.Error(c, fiber.StatusBadRequest, "Destination (to) is required", "")
	}
	t, err := h.service.GetTemplateByID(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Template not found", err.Error())
	}
	if body.SampleVars == nil {
		body.SampleVars = defaultSampleVars(t.Variables)
	}

	if t.Type == "WhatsApp" {
		varParts := strings.Split(t.Variables, ",")
		params := make([]string, 0, len(varParts))
		for _, rv := range varParts {
			vName := strings.TrimSpace(rv)
			if val, ok := body.SampleVars[vName]; ok {
				params = append(params, val)
			} else {
				params = append(params, "")
			}
		}
		result, err := notification.SendTemplate(body.To, t.Title, params)
		if err != nil {
			log.Printf("[email-templates] test send whatsapp failed: %v", err)
			return response.Error(c, fiber.StatusInternalServerError, "Failed to send test WhatsApp: "+err.Error(), "")
		}
		messageID := ""
		if result != nil {
			messageID = result.MessageID
		}
		return response.JSON(c, fiber.StatusOK, fiber.Map{"sent_to": body.To, "message_id": messageID}, "Test WhatsApp message sent")
	}

	html := email.WrapInLayout(t.Title, "[TEST] "+t.Title, renderTemplate(t.Content, body.SampleVars))
	if err := email.SendSimple(body.To, "[TEST] "+t.Title, html); err != nil {
		log.Printf("[email-templates] test send failed: %v", err)
		return response.Error(c, fiber.StatusInternalServerError, "Failed to send test email: "+err.Error(), "")
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"sent_to": body.To}, "Test email sent")
}

// defaultSampleVars generates placeholder sample data from a comma-separated variable list.
func defaultSampleVars(variables string) map[string]string {
	sampleMap := map[string]string{
		// common
		"name":             "Ashok Kumar",
		"email":            "ashok@example.com",
		"role":             "Doctor",
		// OTP
		"otp":             "847291",
		"time":            "2:30 PM",
		// Auth
		"resetURL":        "https://medihub.mrbiz.in/reset?token=sample123",
		"verifyURL":       "https://medihub.mrbiz.in/verify?token=sample123",
		// Careers
		"position":        "Senior Doctor",
		"experience":      "5",
		"license":         "MCI-123456",
		"whatsapp":        "+91 98765 43210",
		"status_label":    "Shortlisted",
		"status_color":    "#059669",
		"message":         "Congratulations! You've been shortlisted for the next round.",
		"password":        "Temp@1234",
		"phone":           "+91 98765 43210",
		"dob":             "15 Aug 1990",
		"gender":          "Male",
		"resume":          "https://medihub.mrbiz.in/resume/sample.pdf",
		"approach":        "Patient-first, holistic care approach.",
		"education":       "MBBS - AIIMS Delhi\nMD - Cardiology",
		// Appointments
		"patientName":     "Priya Sharma",
		"doctorName":      "Dr. Ramesh",
		"date":            "Mon, 16 Jun 2026",
		"mode":            "📹 Video Consultation",
		"fee":             "500",
		"appointmentID":   "1234",
		// Finance
		"amount":          "1200.00",
		"description":     "Dr.Ayya Wallet Top-up",
		"txnID":           "TXN-2026-987654",
		"doctorName2":     "Dr. Ramesh",
		"appointmentCount": "8",
		"payoutID":        "PAY-2026-001",
		// Medical
		"diagnosis":       "Viral Fever — Mild",
		// HR / IT
		"employee_id":     "EMP-9001",
		"department":      "ICU",
		"designation":     "Senior Nurse",
		"message2":        "Please provision Adobe CC license and email access.",
	}
	// If a variable list was given, only keep those variables but use our defaults as values
	if variables == "" {
		return sampleMap
	}
	result := make(map[string]string)
	for _, v := range strings.Split(variables, ",") {
		v = strings.TrimSpace(v)
		if val, ok := sampleMap[v]; ok {
			result[v] = val
		} else {
			result[v] = "[" + v + "]"
		}
	}
	return result
}

// renderTemplate applies {{variable}} substitution on a raw HTML body string.
func renderTemplate(content string, vars map[string]string) string {
	for k, v := range vars {
		content = strings.ReplaceAll(content, "{{"+k+"}}", v)
	}
	return content
}

// POST /api/v1/careers/admin/templates
func (h *CareerHandler) AdminCreateTemplate(c *fiber.Ctx) error {
	var t career.RecruitmentTemplate
	if err := c.BodyParser(&t); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body", err.Error())
	}
	if err := h.service.CreateTemplate(&t); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to create template", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, t, "Template created")
}

// PUT /api/v1/careers/admin/templates/:id
func (h *CareerHandler) AdminUpdateTemplate(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid ID", err.Error())
	}
	var t career.RecruitmentTemplate
	if err := c.BodyParser(&t); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body", err.Error())
	}
	t.ID = uint(id)
	if err := h.service.UpdateTemplate(&t); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to update template", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, t, "Template updated")
}

// DELETE /api/v1/careers/admin/templates/:id
func (h *CareerHandler) AdminDeleteTemplate(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid ID", err.Error())
	}
	if err := h.service.DeleteTemplate(uint(id)); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to delete template", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Template deleted")
}

func (h *CareerHandler) RegisterRoutes(router fiber.Router) {
	g := router.Group("/careers")

	// Public routes
	g.Get("/", h.GetCareers)
	g.Get("/positions", h.GetCareers)
	g.Get("/:id<int>", h.GetJob)
	g.Post("/apply", h.ApplyForJob)
	g.Post("/:id<int>/apply", h.ApplyForJob)

	// Authenticated routes
	g.Get("/my-applications", middleware.Protected(), h.GetMyApplications)

	// Admin routes — applications
	g.Get("/admin/applications", middleware.Protected(), middleware.HRManagerOnly(), h.AdminListApplications)
	g.Patch("/admin/applications/:id/status", middleware.Protected(), middleware.HRManagerOnly(), h.AdminUpdateStatus)
	g.Post("/admin/applications/:id/onboard", middleware.Protected(), middleware.HRManagerOnly(), h.AdminOnboardDoctor)

	// Admin routes — job openings
	g.Post("/admin/jobs", middleware.Protected(), middleware.HRManagerOnly(), h.AdminCreateJob)
	g.Patch("/admin/jobs/:id", middleware.Protected(), middleware.HRManagerOnly(), h.AdminUpdateJob)
	g.Get("/admin/jobs", middleware.Protected(), middleware.HRManagerOnly(), h.AdminGetAllJobs)

	// Admin routes — templates (career)
	g.Get("/admin/templates", middleware.Protected(), middleware.HRManagerOnly(), h.AdminListTemplates)
	g.Post("/admin/templates", middleware.Protected(), middleware.HRManagerOnly(), h.AdminCreateTemplate)
	g.Put("/admin/templates/:id", middleware.Protected(), middleware.HRManagerOnly(), h.AdminUpdateTemplate)
	g.Delete("/admin/templates/:id", middleware.Protected(), middleware.HRManagerOnly(), h.AdminDeleteTemplate)

	// Centralized Email Template Hub routes
	emailTmpl := router.Group("/admin/email-templates", middleware.Protected(), middleware.AdminOnly())
	emailTmpl.Get("/", h.ListEmailTemplates)
	emailTmpl.Get("/:id", h.GetEmailTemplate)
	emailTmpl.Put("/:id", h.UpdateEmailTemplate)
	emailTmpl.Post("/:id/preview", h.PreviewEmailTemplate)
	emailTmpl.Post("/:id/send-test", h.SendTestEmail)
}

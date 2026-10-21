package career_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/career"
	"drayya-core/internal/domain/clinical"
	"drayya-core/internal/domain/hr"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/server"
	"drayya-core/internal/tokens"
)

func setupCareerFullFlowApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("SKIP_LIMITER", "true")
	os.Setenv("RESEND_API_KEY", "")

	dsn := "file:career_full_flow_test?mode=memory&cache=shared&_foreign_keys=off"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	database.DB = db

	require.NoError(t, identity.AutoMigrate(db))
	require.NoError(t, career.AutoMigrate(db))
	require.NoError(t, hr.AutoMigrate(db))
	require.NoError(t, clinical.AutoMigrate(db))

	identity.SeedDoctorTypes(db)
	clinical.SeedAYUSHPods(db)
	require.NoError(t, hr.SeedDefaultLeaveTypes())

	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS job_applications")
		db.Exec("DROP TABLE IF EXISTS job_openings")
		db.Exec("DROP TABLE IF EXISTS staff_profiles")
		db.Exec("DROP TABLE IF EXISTS doctorprofile")
		db.Exec("DROP TABLE IF EXISTS user")
		db.Exec("DROP TABLE IF EXISTS doctor_onboardings")
		db.Exec("DROP TABLE IF EXISTS consultation_reviews")
		db.Exec("DROP TABLE IF EXISTS pods")
		db.Exec("DROP TABLE IF EXISTS leave_types")
		db.Exec("DROP TABLE IF EXISTS leave_balances")
		db.Exec("DROP TABLE IF EXISTS hr_onboarding_records")
	})

	return server.New(), db
}

func makeUserToken(t *testing.T, db *gorm.DB, email, role string) string {
	t.Helper()
	user := identity.User{
		Email:        email,
		PasswordHash: "x",
		Role:         role,
		FullName:     "Test " + role,
		IsActive:     true,
	}
	require.NoError(t, db.Create(&user).Error)
	tok, err := tokens.CreateAccessToken(strconv.FormatUint(uint64(user.ID), 10), user.Role, user.HospitalID, nil)
	require.NoError(t, err)
	return tok
}

func TestFullCareerRecruitmentAndOnboardingFlow(t *testing.T) {
	app, db := setupCareerFullFlowApp(t)

	// Create tokens for roles
	hrToken := makeUserToken(t, db, "hr@drayya.com", "hr_manager")
	chiefDocToken := makeUserToken(t, db, "chief@drayya.com", "chief_doctor")

	var applicationID int
	var onboardedUserID uint
	var staffProfileID uint
	var clinicalOnboardID uint
	var hrOnboardID uint

	// ── 1. HR Manager creates a job opening ───────────────────────────────────
	t.Run("HR creates job opening", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/careers/admin/jobs", body(map[string]any{
			"title":          "Resident Siddha Doctor",
			"department":     "Medical",
			"location":       "Coimbatore",
			"type":           "full-time",
			"specialization": "General",
			"description":    "Looking for a passionate Siddha practitioner.",
			"is_active":      true,
		}))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+hrToken)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		data := readJSON(t, resp.Body)["data"].(map[string]any)
		assert.Equal(t, "Resident Siddha Doctor", data["title"])
	})

	// ── 2. Public applicant submits application ──────────────────────────────
	t.Run("Public submits job application", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/careers/apply", body(map[string]any{
			"job_id":             1,
			"full_name":          "Dr. Kannan Kumar",
			"email":              "kannan.kumar@example.com",
			"phone_number":       "9876501234",
			"whatsapp_number":    "9876501234",
			"date_of_birth":      "1988-10-12",
			"gender":             "male",
			"father_name":        "Kumar S",
			"mother_name":        "Lakshmi K",
			"permanent_address":  "45, West Street, Coimbatore",
			"living_address":     "45, West Street, Coimbatore",
			"position":           "doctor",
			"experience_years":   6,
			"license_number":     "CO-SID-2014-9988",
			"ayush_system":       "siddha",
			"treatment_approach": "Traditional Siddha Varmam and herbal preparations",
			"resume_url":         "https://example.com/kannan_resume.pdf",
			"education_details":  `{"ug":{"college":"Govt Siddha College Palayamkottai","degree":"BSMS","year":"2014"}}`,
		}))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		data := readJSON(t, resp.Body)["data"].(map[string]any)
		assert.Equal(t, "applied", data["status"])
		applicationID = int(data["id"].(float64))
		assert.Greater(t, applicationID, 0)
	})

	// ── 3. HR screens applicant and extends an offer ─────────────────────────
	t.Run("HR screens and offers application", func(t *testing.T) {
		// HR reviews application
		url := fmt.Sprintf("/api/v1/careers/admin/applications/%d/status", applicationID)
		req := httptest.NewRequest("PATCH", url, body(map[string]any{"status": "reviewing"}))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+hrToken)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// HR interviews applicant
		req = httptest.NewRequest("PATCH", url, body(map[string]any{"status": "interviewing", "interview_date": "2026-06-20", "notes": "Friendly candidate"}))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+hrToken)
		resp, err = app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// HR makes offer
		req = httptest.NewRequest("PATCH", url, body(map[string]any{"status": "offered"}))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+hrToken)
		resp, err = app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var appRow career.JobApplication
		db.First(&appRow, applicationID)
		assert.Equal(t, "offered", appRow.Status)
	})

	// ── 4. HR onboards the doctor ─────────────────────────────────────────────
	t.Run("HR initiates onboarding process", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/careers/admin/applications/%d/onboard", applicationID)
		req := httptest.NewRequest("POST", url, nil)
		req.Header.Set("Authorization", "Bearer "+hrToken)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		data := readJSON(t, resp.Body)["data"].(map[string]any)
		onboardedUserID = uint(data["user_id"].(float64))
		assert.Greater(t, onboardedUserID, uint(0))

		// Check if doctor profile was created
		var docProfile identity.DoctorProfile
		err = db.Where("user_id = ?", onboardedUserID).First(&docProfile).Error
		require.NoError(t, err)
		assert.Equal(t, "CO-SID-2014-9988", docProfile.LicenseNumber)
		assert.False(t, docProfile.IsVerified)

		// Check if staff profile was created
		var staffProfile hr.StaffProfile
		err = db.Where("user_id = ?", onboardedUserID).First(&staffProfile).Error
		require.NoError(t, err)
		staffProfileID = staffProfile.ID
		assert.Greater(t, staffProfileID, uint(0))

		// Check that user is created and active
		var u identity.User
		err = db.First(&u, onboardedUserID).Error
		require.NoError(t, err)
		assert.Equal(t, "doctor", u.Role)
		assert.True(t, u.IsActive)
	})

	// ── 5. Chief Doctor conducts clinical onboarding & verification ───────────
	t.Run("Chief Doctor clinical review and go-live", func(t *testing.T) {
		// Fetch the clinical onboarding record
		req := httptest.NewRequest("GET", "/api/v1/clinical/onboarding", nil)
		req.Header.Set("Authorization", "Bearer "+chiefDocToken)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var listResp struct {
			Data struct {
				Items []map[string]any `json:"items"`
			} `json:"data"`
		}
		raw, _ := io.ReadAll(resp.Body)
		fmt.Printf("DEBUG: raw clinical onboarding list: %s\n", string(raw))
		json.Unmarshal(raw, &listResp)

		found := false
		for _, item := range listResp.Data.Items {
			if item["doctor_id"] != nil {
				fmt.Printf("DEBUG: comparing doctor_id: %v with onboardedUserID: %d\n", item["doctor_id"], onboardedUserID)
				if uint(item["doctor_id"].(float64)) == onboardedUserID {
					clinicalOnboardID = uint(item["id"].(float64))
					found = true
					break
				}
			}
		}
		require.True(t, found, "should find clinical onboarding record for the doctor")

		// 1. Submit clinical review (recommendation)
		reviewURL := fmt.Sprintf("/api/v1/clinical/onboarding/%d/clinical-review", clinicalOnboardID)
		req = httptest.NewRequest("POST", reviewURL, body(map[string]any{
			"recommendation": "recommended",
			"notes":          "Excellent domain knowledge in traditional Siddha formulations.",
		}))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+chiefDocToken)
		resp, err = app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var onboardRec clinical.DoctorOnboarding
		db.First(&onboardRec, clinicalOnboardID)
		assert.Equal(t, "recommended", onboardRec.ClinicalRecommendation)
		assert.Equal(t, "credentialing", onboardRec.Stage)

		// 2. Update Checklist: Credentialing
		checklistURL := fmt.Sprintf("/api/v1/clinical/onboarding/%d/checklist", clinicalOnboardID)
		req = httptest.NewRequest("PATCH", checklistURL, body(map[string]any{
			"license_verified":  true,
			"identity_verified": true,
			"agreement_signed":  true,
		}))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+chiefDocToken)
		resp, err = app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		db.First(&onboardRec, clinicalOnboardID)
		assert.Equal(t, "orientation", onboardRec.Stage)

		// 3. Update Checklist: Orientation & Training
		req = httptest.NewRequest("PATCH", checklistURL, body(map[string]any{
			"orientation_done": true,
			"test_consult_done": true,
		}))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+chiefDocToken)
		resp, err = app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		db.First(&onboardRec, clinicalOnboardID)
		assert.Equal(t, "go_live", onboardRec.Stage)

		// 4. Activate Doctor (Go-Live)
		activateURL := fmt.Sprintf("/api/v1/clinical/onboarding/%d/activate", clinicalOnboardID)
		req = httptest.NewRequest("POST", activateURL, nil)
		req.Header.Set("Authorization", "Bearer "+chiefDocToken)
		resp, err = app.Test(req, -1)
		require.NoError(t, err)
		if resp.StatusCode != fiber.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			fmt.Printf("DEBUG: activation failed with status %d, body: %s\n", resp.StatusCode, string(bodyBytes))
		}
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		db.First(&onboardRec, clinicalOnboardID)
		assert.Equal(t, "active", onboardRec.Stage)

		// Check if DoctorProfile and User are marked verified
		var docProfile identity.DoctorProfile
		db.Where("user_id = ?", onboardedUserID).First(&docProfile)
		assert.True(t, docProfile.IsVerified)

		var u identity.User
		db.First(&u, onboardedUserID)
		assert.True(t, u.IsVerified)
	})

	// ── 6. HR Manager verifies documents & completes onboarding record ───────
	t.Run("HR verifies documents and completes onboarding", func(t *testing.T) {
		// Fetch HR onboarding record list
		req := httptest.NewRequest("GET", "/api/v1/hr/onboarding/", nil)
		req.Header.Set("Authorization", "Bearer "+hrToken)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var items []map[string]any
		raw, _ := io.ReadAll(resp.Body)
		json.Unmarshal(raw, &items)

		found := false
		for _, item := range items {
			if uint(item["staff_id"].(float64)) == staffProfileID {
				hrOnboardID = uint(item["id"].(float64))
				found = true
				break
			}
		}
		require.True(t, found, "should find HR onboarding record for the staff profile")

		// 1. Verify documents (certificate, aadhar_id, pan_card, bank_account, passport_photo)
		docURL := fmt.Sprintf("/api/v1/hr/onboarding/%d/documents", hrOnboardID)
		docs := []string{"certificate", "aadhar_id", "pan_card", "bank_account", "passport_photo"}
		for _, dt := range docs {
			req = httptest.NewRequest("PUT", docURL, body(map[string]any{
				"doc_type": dt,
				"status":   "verified",
			}))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+hrToken)
			resp, err = app.Test(req, -1)
			require.NoError(t, err)
			assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		}

		// 2. Complete tasks (system_access_granted)
		taskURL := fmt.Sprintf("/api/v1/hr/onboarding/%d/tasks", hrOnboardID)
		req = httptest.NewRequest("PUT", taskURL, body(map[string]any{
			"task_type":   "system_access_granted",
			"is_complete": true,
		}))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+hrToken)
		resp, err = app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// 3. Move status to in_training then completed
		statusURL := fmt.Sprintf("/api/v1/hr/onboarding/%d/status", hrOnboardID)
		req = httptest.NewRequest("PUT", statusURL, body(map[string]any{"status": "in_training"}))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+hrToken)
		resp, err = app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		req = httptest.NewRequest("PUT", statusURL, body(map[string]any{"status": "completed"}))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+hrToken)
		resp, err = app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Verify final HR onboarding status is completed
		var hrRec hr.OnboardingRecord
		db.First(&hrRec, hrOnboardID)
		assert.Equal(t, "completed", hrRec.Status)
		assert.Equal(t, "verified", hrRec.Certificate)
		assert.Equal(t, "verified", hrRec.AadharID)
		assert.Equal(t, "verified", hrRec.PanCard)
		assert.Equal(t, "verified", hrRec.BankAccount)
		assert.Equal(t, "verified", hrRec.PassportPhoto)
		assert.True(t, hrRec.SystemAccess)
	})
}

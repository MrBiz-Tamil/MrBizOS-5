// Package career_test contains end-to-end integration tests for the doctor
// career application flow:
//
//  1. Admin creates a job opening
//  2. Applicant submits a public application
//  3. Admin advances status: applied → reviewing → interviewing → offered
//  4. Admin onboards the doctor (creates user account, pre-populates profiles,
//     links application)
//  5. Admin verifies the doctor (sets is_verified=true on DoctorProfile)
//
// Guard cases (duplicate onboard, wrong status) are also covered.
package career_test

import (
	"bytes"
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

// ─── Test setup ───────────────────────────────────────────────────────────────

func setupCareerApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()

	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("SKIP_LIMITER", "true")
	// Suppress real email delivery — SendSimple returns an error immediately
	// when the key is empty, which is caught by the non-blocking goroutine.
	os.Setenv("RESEND_API_KEY", "")

	// Use a shared-cache named in-memory DB so every goroutine / connection
	// within this test sees the same data (plain ":memory:" creates a separate
	// database per connection which breaks GORM's pool).
	dsn := "file:career_test?mode=memory&cache=shared&_foreign_keys=off"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Pin to one connection so SQLite WAL serialises all reads/writes to the
	// same in-memory database instance.
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	database.DB = db

	require.NoError(t, identity.AutoMigrate(db))
	require.NoError(t, career.AutoMigrate(db))
	require.NoError(t, hr.AutoMigrate(db))
	require.NoError(t, clinical.AutoMigrate(db))

	t.Cleanup(func() {
		// Drop all tables so the shared in-memory DB is clean for the next run.
		db.Exec("DROP TABLE IF EXISTS job_applications")
		db.Exec("DROP TABLE IF EXISTS job_openings")
		db.Exec("DROP TABLE IF EXISTS staff_profiles")
		db.Exec("DROP TABLE IF EXISTS doctorprofile")
		db.Exec("DROP TABLE IF EXISTS user")
		db.Exec("DROP TABLE IF EXISTS doctor_onboardings")
		db.Exec("DROP TABLE IF EXISTS consultation_reviews")
		db.Exec("DROP TABLE IF EXISTS pods")
	})

	return server.New(), db
}

func makeAdminToken(t *testing.T, db *gorm.DB) string {
	t.Helper()
	admin := identity.User{
		Email:        "admin@career-test.com",
		PasswordHash: "x",
		Role:         "admin",
		FullName:     "Test Admin",
		IsActive:     true,
	}
	require.NoError(t, db.Create(&admin).Error)
	tok, err := tokens.CreateAccessToken(strconv.FormatUint(uint64(admin.ID), 10), admin.Role, admin.HospitalID, nil)
	require.NoError(t, err)
	return tok
}

func body(v any) *bytes.Buffer {
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

func readJSON(t *testing.T, r io.Reader) map[string]any {
	t.Helper()
	raw, _ := io.ReadAll(r)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m), "response: %s", string(raw))
	return m
}

// ─── Full end-to-end test ─────────────────────────────────────────────────────

func TestCareerFlowE2E(t *testing.T) {
	app, db := setupCareerApp(t)
	tok := makeAdminToken(t, db)

	var applicationID int
	var onboardedUserID int

	// ── 1. Admin creates a job opening ────────────────────────────────────────
	t.Run("admin_create_job", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/careers/admin/jobs", body(map[string]any{
			"title":          "Senior Ayurveda Doctor",
			"department":     "Medical",
			"location":       "Chennai",
			"type":           "full-time",
			"specialization": "Panchakarma",
			"description":    "Join our Ayurveda team",
			"is_active":      true,
		}))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tok)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		data := readJSON(t, resp.Body)["data"].(map[string]any)
		assert.Equal(t, "Senior Ayurveda Doctor", data["title"])
		assert.Equal(t, true, data["is_active"])
	})

	// ── 2. Public listing includes the new job ─────────────────────────────────
	t.Run("public_list_jobs", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/careers", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		data := readJSON(t, resp.Body)["data"].([]any)
		assert.Len(t, data, 1)
		job := data[0].(map[string]any)
		assert.Equal(t, "Senior Ayurveda Doctor", job["title"])
	})

	// ── 3. Applicant submits application (no auth) ─────────────────────────────
	t.Run("public_apply", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/careers/apply", body(map[string]any{
			"job_id":           1,
			"full_name":        "Dr. Priya Rajan",
			"email":            "priya.rajan@example.com",
			"phone_number":     "9876543210",
			"whatsapp_number":  "9876543210",
			"date_of_birth":    "1990-05-15",
			"gender":           "female",
			"father_name":      "Rajan K",
			"mother_name":      "Meena R",
			"permanent_address": "12, MG Road, Chennai",
			"living_address":    "34, Anna Nagar, Chennai",
			"position":          "senior_doctor",
			"experience_years":  8,
			"license_number":    "TN-AYU-2015-4421",
			"ayush_system":      "ayurveda",
			"treatment_approach": "Classical Panchakarma and herbal formulations",
			"resume_url":        "https://example.com/resume.pdf",
			"education_details": `{"school":{"name":"Chennai School","board":"CBSE","year":"2006"},"ug":{"college":"SASTRA University","degree":"BAMS","year":"2012"}}`,
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

	// Duplicate application for the same job+email must be rejected.
	t.Run("duplicate_apply_rejected", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/careers/apply", body(map[string]any{
			"job_id":    1,
			"full_name": "Dr. Priya Rajan",
			"email":     "priya.rajan@example.com",
			"position":  "senior_doctor",
		}))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	// ── 4. Admin lists applications ────────────────────────────────────────────
	t.Run("admin_list_applications", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/careers/admin/applications", nil)
		req.Header.Set("Authorization", "Bearer "+tok)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		apps := readJSON(t, resp.Body)["data"].([]any)
		assert.Len(t, apps, 1)
		first := apps[0].(map[string]any)
		assert.Equal(t, "applied", first["status"])
		assert.Equal(t, "Dr. Priya Rajan", first["full_name"])
	})

	// Unauthenticated access must be denied.
	t.Run("admin_list_applications_unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/careers/admin/applications", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	// ── 5. Admin advances status through the pipeline ──────────────────────────
	for _, status := range []string{"reviewing", "interviewing", "offered"} {
		s := status
		t.Run("status_"+s, func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/careers/admin/applications/%d/status", applicationID)
			req := httptest.NewRequest("PATCH", url, body(map[string]any{"status": s}))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+tok)

			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			assert.Equal(t, fiber.StatusOK, resp.StatusCode, "set status=%s failed", s)

			var row career.JobApplication
			db.First(&row, applicationID)
			assert.Equal(t, s, row.Status)
		})
	}

	// Invalid status value must be rejected.
	t.Run("status_invalid_rejected", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/careers/admin/applications/%d/status", applicationID)
		req := httptest.NewRequest("PATCH", url, body(map[string]any{"status": "hired"}))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tok)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	// ── 6. Onboarding a non-offered application is rejected ────────────────────
	t.Run("onboard_non_offered_rejected", func(t *testing.T) {
		// Create a fresh application stuck at "reviewing".
		draft := career.JobApplication{
			FullName: "Another Doc",
			Email:    "another@example.com",
			Position: "junior_doctor",
			Status:   "reviewing",
		}
		require.NoError(t, db.Create(&draft).Error)

		url := fmt.Sprintf("/api/v1/careers/admin/applications/%d/onboard", draft.ID)
		req := httptest.NewRequest("POST", url, body(map[string]any{}))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tok)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	// ── 7. Admin onboards the offered applicant ────────────────────────────────
	t.Run("admin_onboard_doctor", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/careers/admin/applications/%d/onboard", applicationID)
		req := httptest.NewRequest("POST", url, body(map[string]any{}))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tok)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode, "onboard response: %s", string(raw))

		var result map[string]any
		require.NoError(t, json.Unmarshal(raw, &result))
		data := result["data"].(map[string]any)

		onboardedUserID = int(data["user_id"].(float64))
		assert.Greater(t, onboardedUserID, 0)
		assert.Equal(t, "doctor", data["role"])
		assert.Equal(t, "priya.rajan@example.com", data["email"])
		assert.Equal(t, fmt.Sprintf("DOC-%d", onboardedUserID), data["employee_id"])

		// Application must be linked and marked onboarded.
		var app career.JobApplication
		db.First(&app, applicationID)
		assert.Equal(t, "onboarded", app.Status)
		require.NotNil(t, app.OnboardedUserID)
		assert.Equal(t, uint(onboardedUserID), *app.OnboardedUserID)
		// Gap 6: user_id on the application must be linked to the created user.
		require.NotNil(t, app.UserID, "Gap 6: application.user_id must not be nil after onboarding")
		assert.Equal(t, uint(onboardedUserID), *app.UserID, "Gap 6: application.user_id must equal the onboarded user's ID")

		// Doctor user account must exist with correct fields.
		var user identity.User
		require.NoError(t, db.First(&user, onboardedUserID).Error)
		assert.Equal(t, "doctor", user.Role)
		assert.Equal(t, "priya.rajan@example.com", user.Email)
		assert.Equal(t, "Dr. Priya Rajan", user.FullName)
		assert.True(t, user.IsActive)

		// DoctorProfile must exist with application data pre-populated and not yet verified.
		var profile identity.DoctorProfile
		require.NoError(t, db.Where("user_id = ?", onboardedUserID).First(&profile).Error)
		assert.False(t, profile.IsVerified, "newly onboarded doctor must not be verified yet")
		assert.Equal(t, "TN-AYU-2015-4421", profile.LicenseNumber)
		assert.Equal(t, 8, profile.YearsOfExperience)
		assert.Equal(t, "Ayurveda (BAMS)", profile.Specialization)

		// StaffProfile must be auto-created and linked to the new user.
		var staff hr.StaffProfile
		require.NoError(t, db.Where("user_id = ?", onboardedUserID).First(&staff).Error)
		assert.Equal(t, fmt.Sprintf("DOC-%d", onboardedUserID), staff.EmployeeID)
		assert.Equal(t, "Medical", staff.Department)
		assert.Equal(t, "senior_doctor", staff.Designation)
		assert.Equal(t, "active", staff.EmploymentStatus)
	})

	// ── 8. Duplicate onboard on same application is rejected ──────────────────
	t.Run("onboard_duplicate_rejected", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/careers/admin/applications/%d/onboard", applicationID)
		req := httptest.NewRequest("POST", url, body(map[string]any{}))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tok)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
	})

	// ── 9a. Gap 7: pending queue shows no mismatch when licenses match ────────
	t.Run("gap7_pending_license_match", func(t *testing.T) {
		// Delete the clinical onboarding record to simulate a legacy/non-pipeline doctor
		// so they appear in the admin pending verification queue.
		require.NoError(t, db.Exec("DELETE FROM doctor_onboardings WHERE doctor_id = ?", onboardedUserID).Error)

		req := httptest.NewRequest("GET", "/api/v1/admin/doctors", nil)
		req.Header.Set("Authorization", "Bearer "+tok)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode, "pending doctors response: %s", string(raw))

		var pending []map[string]any
		require.NoError(t, json.Unmarshal(raw, &pending))

		var found bool
		for _, p := range pending {
			if p["email"] == "priya.rajan@example.com" {
				found = true
				// Application license was "TN-AYU-2015-4421" and onboarding copied it
				// to the profile — no mismatch expected.
				assert.Equal(t, "TN-AYU-2015-4421", p["application_license_number"],
					"Gap 7: application_license_number must be surfaced in pending queue")
				assert.Equal(t, false, p["license_mismatch"],
					"Gap 7: license_mismatch must be false when application and profile licenses match")
			}
		}
		assert.True(t, found, "onboarded-but-unverified doctor must appear in the pending queue")
	})

	// ── 9b. Gap 7: pending queue flags mismatch when profile license is changed ─
	t.Run("gap7_pending_license_mismatch", func(t *testing.T) {
		// Simulate the doctor updating their profile with a different license number.
		require.NoError(t, db.Table("doctorprofile").
			Where("user_id = ?", onboardedUserID).
			Update("license_number", "TN-AYU-2015-XXXX").Error)

		req := httptest.NewRequest("GET", "/api/v1/admin/doctors", nil)
		req.Header.Set("Authorization", "Bearer "+tok)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var pending []map[string]any
		require.NoError(t, json.Unmarshal(raw, &pending))

		for _, p := range pending {
			if p["email"] == "priya.rajan@example.com" {
				assert.Equal(t, true, p["license_mismatch"],
					"Gap 7: license_mismatch must be true when profile license differs from application")
				assert.Equal(t, "TN-AYU-2015-4421", p["application_license_number"])
				assert.Equal(t, "TN-AYU-2015-XXXX", p["license_number"])
			}
		}
	})

	// ── 9. Admin verifies the onboarded doctor ─────────────────────────────────
	// Profile license was changed to "TN-AYU-2015-XXXX" in gap7_pending_license_mismatch,
	// so the verify response must include a license_mismatch warning.
	t.Run("admin_verify_doctor", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/admin/doctors/%d/verify", onboardedUserID)
		req := httptest.NewRequest("POST", url, nil)
		req.Header.Set("Authorization", "Bearer "+tok)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		raw, _ := io.ReadAll(resp.Body)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode, "verify response: %s", string(raw))

		var result map[string]any
		require.NoError(t, json.Unmarshal(raw, &result))

		// Gap 7: mismatch must be surfaced in the verify response so the admin can
		// take manual action before granting the doctor access to patient appointments.
		assert.Equal(t, true, result["license_mismatch"],
			"Gap 7: verify response must flag license_mismatch when profile != application")
		assert.Equal(t, "TN-AYU-2015-4421", result["application_license_number"])
		assert.Equal(t, "TN-AYU-2015-XXXX", result["profile_license_number"])
		assert.NotEmpty(t, result["warning"])

		// DoctorProfile must now be verified.
		var profile identity.DoctorProfile
		db.Where("user_id = ?", onboardedUserID).First(&profile)
		assert.True(t, profile.IsVerified)

		// User.is_verified must be in sync.
		var user identity.User
		db.First(&user, onboardedUserID)
		assert.True(t, user.IsVerified)
	})

	// ── 10. Verified doctor no longer appears in pending queue ─────────────────
	t.Run("verified_doctor_not_in_pending_queue", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/admin/doctors", nil)
		req.Header.Set("Authorization", "Bearer "+tok)

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		raw, _ := io.ReadAll(resp.Body)
		var pending []map[string]any
		json.Unmarshal(raw, &pending)

		for _, p := range pending {
			assert.NotEqual(t, "priya.rajan@example.com", p["email"],
				"verified doctor must not appear in the pending queue")
		}
	})
}

package admin

import (
	"log"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/services/email"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

func GetDoctorMetrics(c *fiber.Ctx) error {
	db := database.DB

	type Result struct {
		DoctorID           uint    `json:"doctor_id"`
		FullName           string  `json:"full_name"`
		Specialization     string  `json:"specialization"`
		AverageRating      float32 `json:"average_rating"`
		TotalConsultations int     `json:"total_consultations"`
		CompletionRate     float64 `json:"completion_rate"`
	}

	var results []Result
	// Join User with DoctorProfile and calculate completion rate from appointments
	err := db.Table("doctorprofile").
		Select(`
			doctorprofile.user_id as doctor_id,
			"user".full_name,
			doctorprofile.specialization,
			doctorprofile.average_rating,
			doctorprofile.total_consultations,
			(CAST((SELECT COUNT(*) FROM appointments WHERE appointments.doctor_id = doctorprofile.user_id AND appointments.status = 'completed') AS FLOAT) / 
			 NULLIF((SELECT COUNT(*) FROM appointments WHERE appointments.doctor_id = doctorprofile.user_id AND appointments.status IN ('completed', 'no_show', 'cancelled')), 0) * 100) as completion_rate
		`).
		Joins(`JOIN "user" ON "user".id = doctorprofile.user_id`).
		Where("doctorprofile.is_verified = ?", true).
		Scan(&results).Error

	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "QUERY_FAILED", err.Error())
	}

	// Handle null completion rates (for doctors with no appointments)
	for i := range results {
		if results[i].CompletionRate == 0 && results[i].TotalConsultations == 0 {
			results[i].CompletionRate = 100 // default for new verified doctors
		}
	}

	return c.JSON(results)
}

func GetPendingDoctors(c *fiber.Ctx) error {
	db := database.DB

	type PendingDoc struct {
		ID                       uint      `json:"id"`
		UserID                   uint      `json:"user_id"`
		FullName                 string    `json:"full_name"`
		Email                    string    `json:"email"`
		Specialization           string    `json:"specialization"`
		LicenseNumber            string    `json:"license_number"`
		Qualification            string    `json:"qualification"`
		YearsOfExperience        int       `json:"years_of_experience"`
		CreatedAt                time.Time `json:"created_at"`
		ApplicationLicenseNumber string    `json:"application_license_number"`
		LicenseMismatch          bool      `json:"license_mismatch"`
	}

	var pending []PendingDoc
	err := db.Table("doctorprofile").
		Select(`
			doctorprofile.id,
			doctorprofile.user_id,
			"user".full_name,
			"user".email,
			doctorprofile.specialization,
			doctorprofile.license_number,
			doctorprofile.qualification,
			doctorprofile.years_of_experience,
			doctorprofile.created_at,
			COALESCE(ja.license_number, '') as application_license_number,
			CASE
				WHEN ja.license_number IS NOT NULL AND ja.license_number != '' AND ja.license_number != doctorprofile.license_number THEN true
				ELSE false
			END as license_mismatch
		`).
		Joins(`JOIN "user" ON "user".id = doctorprofile.user_id`).
		Joins(`LEFT JOIN job_applications ja ON ja.onboarded_user_id = doctorprofile.user_id`).
		Where("doctorprofile.is_verified = ?", false).
		// Exclude doctors who are in the governed clinical onboarding pipeline
		// (an open, non-terminal doctor_onboardings record). Those are activated
		// by the head doctor via ActivateDoctor after credentialing/orientation —
		// quick-verify here would bypass those gates. Legacy doctors with no
		// onboarding record still appear so they can be verified directly.
		Where(`NOT EXISTS (SELECT 1 FROM doctor_onboardings o
			WHERE o.doctor_id = doctorprofile.user_id AND o.stage NOT IN ('active','rejected'))`).
		Order("doctorprofile.created_at ASC").
		Scan(&pending).Error

	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "QUERY_FAILED", err.Error())
	}

	return c.JSON(pending)
}

func GetVerifiedDoctors(c *fiber.Ctx) error {
	db := database.DB
	var docs []identity.User
	db.Table("\"user\"").
		Select("\"user\".*").
		Joins("JOIN doctorprofile dp ON dp.user_id = \"user\".id").
		Where("dp.is_verified = ?", true).
		Find(&docs)
	
	for i := range docs { docs[i].PasswordHash = "" }
	return c.JSON(docs)
}

type DoctorAuditRecord struct {
	UserID            uint      `json:"user_id"`
	FullName          string    `json:"full_name"`
	Email             string    `json:"email"`
	PhoneNumber       string    `json:"phone_number"`
	Specialization    string    `json:"specialization"`
	CollegeName       string    `json:"college_name"`
	LicenseNumber     string    `json:"license_number"`
	Qualification     string    `json:"qualification"`
	YearsOfExperience int       `json:"years_of_experience"`
	IsVerified        bool      `json:"is_verified"`
	CreatedAt         time.Time `json:"created_at"`
}

func GetDoctorAuditDirectory(c *fiber.Ctx) error {
	db := database.DB
	var records []DoctorAuditRecord

	err := db.Table("\"user\"").
		Select(`
			"user".id as user_id,
			"user".full_name,
			"user".email,
			"user".phone_number,
			dp.specialization,
			dp.college_name,
			dp.license_number,
			dp.qualification,
			dp.years_of_experience,
			dp.is_verified,
			dp.created_at
		`).
		Joins("JOIN doctorprofile dp ON dp.user_id = \"user\".id").
		Where("\"user\".role IN ?", []string{"doctor", "chief_doctor", "hospital_head_doctor"}).
		Order("dp.college_name ASC, \"user\".full_name ASC").
		Scan(&records).Error

	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "QUERY_FAILED", err.Error())
	}

	return c.JSON(records)
}

func GetDoctorManagementStats(c *fiber.Ctx) error {
	db := database.DB

	var totalDoctors, pendingVerifications, onlineDoctors int64
	var avgRating float64

	db.Model(&identity.DoctorProfile{}).Count(&totalDoctors)
	db.Model(&identity.DoctorProfile{}).Where("is_verified = ?", false).Count(&pendingVerifications)
	db.Model(&identity.DoctorProfile{}).Where("is_online = ? AND is_verified = ?", true, true).Count(&onlineDoctors)

	db.Model(&identity.DoctorProfile{}).
		Where("is_verified = ?", true).
		Select("COALESCE(AVG(average_rating), 0)").
		Scan(&avgRating)

	return c.JSON(fiber.Map{
		"total_doctors":         totalDoctors,
		"pending_verifications": pendingVerifications,
		"online_doctors":        onlineDoctors,
		"average_rating":        avgRating,
	})
}

func VerifyDoctor(c *fiber.Ctx) error {
	db := database.DB
	doctorID, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_DOCTOR_ID", "Invalid doctor ID")
	}

	// Gate enforcement — a freelance doctor with an open clinical onboarding
	// record can ONLY be verified by completing the pipeline (clinical review,
	// credentialing, orientation) and activating from the go_live stage. This
	// closes the dashboard "Verify" shortcut that previously skipped every gate.
	// An admin/super_admin may still force-verify (e.g. legacy data fix) by
	// passing ?force=true, which is audit-logged via the request context.
	var ob struct{ Stage string }
	obFound := db.Table("doctor_onboardings").Select("stage").
		Where("doctor_id = ?", doctorID).Scan(&ob)
	if obFound.RowsAffected > 0 && ob.Stage != "go_live" && ob.Stage != "active" {
		role, _ := c.Locals("role").(string)
		isAdmin := role == "admin" || role == "super_admin"
		if !(isAdmin && c.QueryBool("force", false)) {
			return response.Error(c, fiber.StatusConflict, "NOT_READY",
				"This doctor is still in clinical onboarding. Complete the verification queue gates before taking them live.")
		}
	}

	// Update is_verified in doctorprofile
	result := db.Table("doctorprofile").Where("user_id = ?", doctorID).Update("is_verified", true)
	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "VERIFY_FAILED", result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return response.Error(c, fiber.StatusNotFound, "DOCTOR_PROFILE_NOT_FOUND", "Doctor profile not found")
	}

	// Also update IsVerified in user table for consistency
	db.Model(&identity.User{}).Where("id = ?", doctorID).Update("is_verified", true)

	// Keep the clinical onboarding pipeline in sync. If this doctor has an open
	// onboarding record (e.g. an admin quick-verified them directly), close it as
	// active so the head doctor's board doesn't keep showing them mid-onboarding
	// while they're already live. No-op for doctors with no onboarding record.
	db.Table("doctor_onboardings").
		Where("doctor_id = ? AND stage NOT IN ('active','rejected')", doctorID).
		Updates(map[string]interface{}{
			"stage":        "active",
			"activated_at": time.Now(),
			"updated_at":   time.Now(),
		})

	// Cross-check license number against the onboarding application (Gap 7).
	licenseMismatch := false
	var appLicenseNumber, profileLicenseNumber string
	type licenseRow struct {
		AppLicense     string
		ProfileLicense string
	}
	var lr licenseRow
	if err := db.Table("doctorprofile").
		Select("COALESCE(ja.license_number, '') as app_license, COALESCE(doctorprofile.license_number, '') as profile_license").
		Joins("LEFT JOIN job_applications ja ON ja.onboarded_user_id = doctorprofile.user_id").
		Where("doctorprofile.user_id = ?", doctorID).
		Scan(&lr).Error; err == nil {
		appLicenseNumber = lr.AppLicense
		profileLicenseNumber = lr.ProfileLicense
		if appLicenseNumber != "" && appLicenseNumber != profileLicenseNumber {
			licenseMismatch = true
			log.Printf("[admin] license mismatch for doctor %d: application=%q profile=%q", doctorID, appLicenseNumber, profileLicenseNumber)
		}
	}

	// Notify the doctor so they know they can start accepting appointments.
	go func() {
		var user identity.User
		if err := db.Select("email, full_name").First(&user, doctorID).Error; err != nil || user.Email == "" {
			return
		}
		if err := email.SendSimple(
			user.Email,
			"Your Dr.Ayya Profile Has Been Verified ✅",
			email.DoctorProfileVerifiedEmail(user.FullName),
		); err != nil {
			log.Printf("[admin] verification email failed for user %d: %v", doctorID, err)
		}
	}()

	resp := fiber.Map{"message": "Doctor verified successfully"}
	if licenseMismatch {
		resp["license_mismatch"] = true
		resp["application_license_number"] = appLicenseNumber
		resp["profile_license_number"] = profileLicenseNumber
		resp["warning"] = "License number on profile differs from the onboarding application — please review before approving access"
	}
	return c.JSON(resp)
}

func PromotePremiumDoctor(c *fiber.Ctx) error {
	db := database.DB
	doctorID, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_DOCTOR_ID", "Invalid doctor ID")
	}

	var body struct {
		IsPremium bool `json:"is_premium"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	// Update is_verified_premium in User table
	err = db.Model(&identity.User{}).Where("id = ?", doctorID).Update("is_verified_premium", body.IsPremium).Error
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}

	status := "promoted to premium"
	if !body.IsPremium {
		status = "removed from premium"
	}

	return c.JSON(fiber.Map{"message": "Doctor " + status})
}

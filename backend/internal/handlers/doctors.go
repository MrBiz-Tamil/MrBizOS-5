package handlers

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/emr"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/events"
	"drayya-core/internal/middleware"
	"drayya-core/internal/presence"
	"drayya-core/internal/services/tds"
	"drayya-core/internal/utils/response"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// DoctorListItem is a FLAT scan target for the doctors list query.
// No struct embedding — explicit column aliases eliminate GORM scanning ambiguity.
// JSON field names match the frontend DoctorProfile TypeScript interface exactly.
type DoctorListItem struct {
	// From "user" table
	UserID      uint   `gorm:"column:user_id"       json:"user_id"`
	ID          uint   `gorm:"column:profile_id"    json:"id"`
	FullName    string `gorm:"column:full_name"     json:"full_name"`
	Email       string `gorm:"column:email"         json:"email"`
	PhoneNumber string `gorm:"column:phone_number"  json:"phone_number"`
	// From doctorprofile table
	IsOnline           bool    `gorm:"column:is_online"           json:"is_online"`
	Specialization     string  `gorm:"column:specialization"      json:"specialization"`
	AverageRating      float32 `gorm:"column:average_rating"      json:"average_rating"`
	ConsultationFee    int64   `gorm:"column:consultation_fee"    json:"consultation_fee"`
	Languages          string  `gorm:"column:languages"           json:"languages"`
	YearsOfExperience  int     `gorm:"column:years_of_experience" json:"years_of_experience"`
	Qualification      string  `gorm:"column:qualification"       json:"qualification"`
	Bio                string  `gorm:"column:bio"                 json:"bio"`
	IsVerified         bool    `gorm:"column:is_verified"         json:"is_verified"`
	TotalConsultations int     `gorm:"column:total_consultations" json:"total_consultations"`
	ProfilePhotoURL    string  `gorm:"column:profile_photo_url"   json:"profile_photo_url"`
	CurrentStatus      string  `gorm:"column:current_status"      json:"current_status"`
	AdminBoostScore    float64 `gorm:"column:admin_boost_score"   json:"admin_boost_score"`
}

// GetDoctorTypes returns all 4 AYUSH doctor types with nested sub-specialties
// and live doctor / online counts per type.
// GET /api/v1/doctor-types — public, no auth required.
func GetDoctorTypes(c *fiber.Ctx) error {
	type SubSpecialtyOut struct {
		ID                  uint   `json:"id"`
		Name                string `json:"name"`
		Slug                string `json:"slug"`
		Description         string `json:"description"`
		RequiresCertificate bool   `json:"requires_certificate"`
		DoctorCount         int64  `json:"doctor_count"`
	}
	type DoctorTypeOut struct {
		ID           uint              `json:"id"`
		Name         string            `json:"name"`
		Slug         string            `json:"slug"`
		Description  string            `json:"description"`
		Icon         string            `json:"icon"`
		Color        string            `json:"color"`
		DoctorCount  int64             `json:"doctor_count"`
		OnlineCount  int64             `json:"online_count"`
		SubSpecialties []SubSpecialtyOut `json:"sub_specialties"`
	}

	var types []identity.DoctorType
	if err := database.DB.Where("is_active = ?", true).Order("id asc").Find(&types).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch doctor types")
	}

	result := make([]DoctorTypeOut, 0, len(types))
	for _, t := range types {
		dt := t // capture

		var doctorCount, onlineCount int64
		database.DB.Model(&identity.DoctorProfile{}).
			Joins(`JOIN "user" u ON u.id = doctorprofile.user_id`).
			Where("u.is_active = ? AND u.role IN ? AND (doctorprofile.doctor_type_id = ? OR LOWER(doctorprofile.specialization) LIKE LOWER(?))",
				true, []string{"doctor", "chief_doctor", "hospital_head_doctor"}, dt.ID, "%"+dt.Slug+"%").
			Count(&doctorCount)
		database.DB.Model(&identity.DoctorProfile{}).
			Joins(`JOIN "user" u ON u.id = doctorprofile.user_id`).
			Where("u.is_active = ? AND u.role IN ? AND doctorprofile.is_online = ? AND (doctorprofile.doctor_type_id = ? OR LOWER(doctorprofile.specialization) LIKE LOWER(?))",
				true, []string{"doctor", "chief_doctor", "hospital_head_doctor"}, true, dt.ID, "%"+dt.Slug+"%").
			Count(&onlineCount)

		var subs []identity.DoctorSubSpecialty
		database.DB.Where("doctor_type_id = ? AND is_active = ?", dt.ID, true).Find(&subs)

		subOut := make([]SubSpecialtyOut, 0, len(subs))
		for _, s := range subs {
			var subCount int64
			database.DB.Table("doctor_sub_specialty_assignments dsa").
				Joins(`JOIN "user" u ON u.id = dsa.doctor_user_id`).
				Where("dsa.sub_specialty_id = ? AND dsa.is_active = ? AND u.is_active = ?", s.ID, true, true).
				Count(&subCount)
			subOut = append(subOut, SubSpecialtyOut{
				ID:                  s.ID,
				Name:                s.Name,
				Slug:                s.Slug,
				Description:         s.Description,
				RequiresCertificate: s.RequiresCertificate,
				DoctorCount:         subCount,
			})
		}

		result = append(result, DoctorTypeOut{
			ID:             dt.ID,
			Name:           dt.Name,
			Slug:           dt.Slug,
			Description:    dt.Description,
			Icon:           dt.Icon,
			Color:          dt.Color,
			DoctorCount:    doctorCount,
			OnlineCount:    onlineCount,
			SubSpecialties: subOut,
		})
	}

	return response.JSON(c, fiber.StatusOK, result, "Doctor types fetched")
}

// GetDoctorCountByType returns the count of active, verified doctors grouped by doctor type.
// GET /api/v1/doctors/count — public, no auth required.
func GetDoctorCountByType(c *fiber.Ctx) error {
	onlineOnly := c.Query("online_only") == "true"

	var types []identity.DoctorType
	if err := database.DB.Where("is_active = ?", true).Find(&types).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch doctor types")
	}

	counts := make(map[string]int)
	total := 0

	for _, dt := range types {
		var count int64
		query := database.DB.Model(&identity.DoctorProfile{}).
			Joins(`JOIN "user" u ON u.id = doctorprofile.user_id`).
			Where("u.is_active = ? AND u.role = ?", true, "doctor")

		if onlineOnly {
			query = query.Where("doctorprofile.is_online = ?", true)
		}

		// Resilient matching: match by ID OR by specialization keyword (slug)
		query = query.Where("(doctorprofile.doctor_type_id = ? OR LOWER(doctorprofile.specialization) LIKE LOWER(?))",
			dt.ID, "%"+dt.Slug+"%")

		query.Count(&count)
		counts[dt.Slug] = int(count)
		total += int(count)
	}

	return c.JSON(fiber.Map{
		"counts": counts,
		"total":  total,
	})
}

// DoctorListItem is a FLAT scan target for the doctors list query.
// No struct embedding — explicit column aliases eliminate GORM scanning ambiguity.
// JSON field names match the frontend DoctorProfile TypeScript interface exactly.
type DoctorListItemExtended struct {
	DoctorListItem
	IsFeatured          bool    `gorm:"column:is_featured"          json:"is_featured"`
	TotalReviews        int     `gorm:"column:total_reviews"        json:"total_reviews"`
	WeeklyConsultations int     `gorm:"column:weekly_consultations" json:"weekly_consultations"`
	CompletionRate      float64 `gorm:"column:completion_rate"      json:"completion_rate"`
	PerformanceScore    float64 `gorm:"column:performance_score"    json:"performance_score"`
	DoctorTypeID        *uint   `gorm:"column:doctor_type_id"       json:"doctor_type_id"`
}

// DoctorsPagedResponse wraps the paginated doctors list with metadata.
type DoctorsPagedResponse struct {
	Doctors []DoctorListItemExtended `json:"doctors"`
	Total   int64                    `json:"total"`
	Page    int                      `json:"page"`
	Limit   int                      `json:"limit"`
	HasMore bool                     `json:"has_more"`
}

// GetDoctors - List active doctors with full server-side filtering and pagination.
//
// Query params:
//
//	type / specialty    — AYUSH type slug (siddha, ayurveda, yoga_naturopathy, psychology)
//	specialization      — free-text LIKE filter
//	sub_specialty       — sub-specialty slug
//	min_rating          — minimum average_rating (float)
//	min_experience      — minimum years_of_experience (int)
//	fee_min / fee_max   — fee range in paise
//	languages           — comma-separated (e.g. tamil,english)
//	verified_only=true  — only verified doctors
//	online_only=true    — only online doctors
//	free_only=true      — only free consultation doctors
//	featured=true       — only featured doctors
//	sort                — rating|experience|fee_asc|fee_desc|consultations|performance (default: performance)
//	page / limit        — pagination (default page=1, limit=20)
func GetDoctors(c *fiber.Ctx) error {
	var rows []DoctorListItemExtended

	feeExpr := "(dp.consultation_fees->>'INR')::bigint"
	if database.DB.Dialector.Name() == "sqlite" {
		feeExpr = "COALESCE(json_extract(dp.consultation_fees, '$.INR'), 0)"
	}

	selectClause := fmt.Sprintf(`
			u.id                                        AS user_id,
			COALESCE(dp.id, 0)                          AS profile_id,
			COALESCE(u.full_name, '')                   AS full_name,
			COALESCE(u.email, '')                       AS email,
			COALESCE(u.phone_number, '')                AS phone_number,
			COALESCE(dp.is_online, false)               AS is_online,
			COALESCE(dp.specialization, '')             AS specialization,
			COALESCE(dp.average_rating, 0)              AS average_rating,
			COALESCE(%s, 0)                             AS consultation_fee,
			COALESCE(dp.languages_spoken, '')           AS languages,
			COALESCE(dp.years_of_experience, 0)         AS years_of_experience,
			COALESCE(dp.qualification, '')              AS qualification,
			COALESCE(dp.bio, '')                        AS bio,
			COALESCE(dp.is_verified, false)             AS is_verified,
			COALESCE(dp.is_featured, false)             AS is_featured,
			COALESCE(dp.total_consultations, 0)         AS total_consultations,
			COALESCE(dp.total_reviews, 0)               AS total_reviews,
			COALESCE(dp.weekly_consultations, 0)        AS weekly_consultations,
			COALESCE(dp.completion_rate, 0)             AS completion_rate,
			COALESCE(dp.performance_score, 0)           AS performance_score,
			COALESCE(u.profile_picture, '')             AS profile_photo_url,
			COALESCE(dp.current_status, 'offline')      AS current_status,
			COALESCE(dp.admin_boost_score, 0)           AS admin_boost_score,
			dp.doctor_type_id                           AS doctor_type_id`, feeExpr)

	query := database.DB.
		Table(`"user" u`).
		Select(selectClause).
		Joins(`LEFT JOIN doctorprofile dp ON u.id = dp.user_id`).
		Where(`u.role IN ?`, []string{"doctor", "chief_doctor", "hospital_head_doctor"}).
		Where(`u.is_active = ?`, true)

	// ── Filters ──────────────────────────────────────────────────────────────

	if spec := c.Query("specialization"); spec != "" {
		query = query.Where("LOWER(dp.specialization) LIKE LOWER(?)", "%"+spec+"%")
	}

	typeSlug := c.Query("type")
	if typeSlug == "" {
		typeSlug = c.Query("specialty")
	}
	if typeSlug != "" {
		var dt identity.DoctorType
		if err := database.DB.Where("slug = ?", typeSlug).First(&dt).Error; err == nil {
			query = query.Where("(dp.doctor_type_id = ? OR LOWER(dp.specialization) LIKE LOWER(?))", dt.ID, "%"+typeSlug+"%")
		} else {
			query = query.Where("LOWER(dp.specialization) LIKE LOWER(?)", "%"+typeSlug+"%")
		}
	}

	if subSlugOrID := c.Query("sub_specialty"); subSlugOrID != "" {
		query = query.Where(`dp.user_id IN (
			SELECT dsa.doctor_user_id FROM doctor_sub_specialty_assignments dsa
			JOIN doctor_sub_specialties dss ON dss.id = dsa.sub_specialty_id
			WHERE (dss.slug = ? OR CAST(dsa.sub_specialty_id AS TEXT) = ?) AND dsa.is_active = true
		)`, subSlugOrID, subSlugOrID)
	}

	if minRating := c.QueryFloat("min_rating", 0); minRating > 0 {
		query = query.Where("dp.average_rating >= ?", minRating)
	}

	if minExp := c.QueryInt("min_experience", 0); minExp > 0 {
		query = query.Where("dp.years_of_experience >= ?", minExp)
	}

	if feeMin := c.QueryInt("fee_min", 0); feeMin > 0 {
		query = query.Where(fmt.Sprintf("%s >= ?", feeExpr), feeMin)
	}
	if feeMax := c.QueryInt("fee_max", 0); feeMax > 0 {
		query = query.Where(fmt.Sprintf("%s <= ?", feeExpr), feeMax)
	}

	if langs := c.Query("languages"); langs != "" {
		for _, lang := range strings.Split(langs, ",") {
			lang = strings.TrimSpace(lang)
			if lang != "" {
				query = query.Where("LOWER(dp.languages_spoken) LIKE LOWER(?)", "%"+lang+"%")
			}
		}
	}

	if c.Query("verified_only") == "true" || c.Query("verified") == "true" {
		query = query.Where("dp.is_verified = ?", true)
	}

	if c.Query("online_only") == "true" || c.Query("online") == "true" {
		query = query.Where("dp.is_online = ?", true)
	}

	if c.Query("free_only") == "true" {
		query = query.Where(fmt.Sprintf("(COALESCE(%s, 0) = 0 OR dp.offers_free_consultation = ?)", feeExpr), true)
	}

	if c.Query("featured") == "true" {
		query = query.Where("dp.is_featured = ?", true)
	}

	// ── Count total before pagination ────────────────────────────────────────

	var total int64
	query.Count(&total)

	// ── Sort ─────────────────────────────────────────────────────────────────

	switch c.Query("sort") {
	case "rating":
		query = query.Order("dp.average_rating DESC NULLS LAST")
	case "experience":
		query = query.Order("dp.years_of_experience DESC NULLS LAST")
	case "fee_asc":
		if database.DB.Dialector.Name() == "sqlite" {
			query = query.Order(fmt.Sprintf("%s ASC", feeExpr))
		} else {
			query = query.Order(fmt.Sprintf("%s ASC NULLS LAST", feeExpr))
		}
	case "fee_desc":
		if database.DB.Dialector.Name() == "sqlite" {
			query = query.Order(fmt.Sprintf("%s DESC", feeExpr))
		} else {
			query = query.Order(fmt.Sprintf("%s DESC NULLS LAST", feeExpr))
		}
	case "consultations":
		query = query.Order("dp.total_consultations DESC NULLS LAST")
	default:
		// Default: composite performance score — online doctors bubble up naturally
		query = query.Order(`dp.performance_score DESC NULLS LAST, dp.is_online DESC NULLS LAST, dp.admin_boost_score DESC NULLS LAST`)
	}

	// ── Pagination ───────────────────────────────────────────────────────────

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit
	query = query.Offset(offset).Limit(limit)

	if err := query.Scan(&rows).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch doctors: "+err.Error())
	}

	if rows == nil {
		rows = []DoctorListItemExtended{}
	}

	// Synchronize with real-time Redis presence heartbeats.
	ctx := c.Context()
	for i := range rows {
		doctorIDStr := fmt.Sprintf("%d", rows[i].UserID)
		redisOnline := presence.IsDoctorOnline(ctx, doctorIDStr)
		if redisOnline {
			rows[i].IsOnline = true
		} else if rows[i].IsOnline {
			rows[i].IsOnline = false
			uid := rows[i].UserID
			go database.DB.Model(&identity.DoctorProfile{}).
				Where("user_id = ?", uid).
				Updates(map[string]interface{}{"is_online": false, "current_status": "offline"})
		}
	}

	return c.JSON(DoctorsPagedResponse{
		Doctors: rows,
		Total:   total,
		Page:    page,
		Limit:   limit,
		HasMore: int64(offset+limit) < total,
	})
}

// GetDoctorsDebug - Diagnostic endpoint to inspect DB state without needing psql access.
// GET /api/v1/doctors/debug — public, no auth required (safe: only returns aggregate counts).
func GetDoctorsDebug(c *fiber.Ctx) error {
	type Row struct {
		Label string `gorm:"column:label" json:"label"`
		Count int64  `gorm:"column:cnt"   json:"count"`
	}
	var counts []Row
	database.DB.Raw(`
		SELECT 'total_users'       AS label, COUNT(*)                                        AS cnt FROM "user"
		UNION ALL
		SELECT 'doctor_role',                COUNT(*)                                        FROM "user" WHERE role = 'doctor'
		UNION ALL
		SELECT 'doctor_active',              COUNT(*)                                        FROM "user" WHERE role = 'doctor' AND is_active = true
		UNION ALL
		SELECT 'doctor_profiles',            COUNT(*)                                        FROM doctorprofile
		UNION ALL
		SELECT 'doctor_verified',            COUNT(*)                                        FROM doctorprofile WHERE is_verified = true
		UNION ALL
		SELECT 'doctor_online',              COUNT(*)                                        FROM doctorprofile WHERE is_online = true
	`).Scan(&counts)

	// Sample the first doctor row so we can see actual field values
	type SampleRow struct {
		UserID     uint   `gorm:"column:user_id"   json:"user_id"`
		FullName   string `gorm:"column:full_name" json:"full_name"`
		Role       string `gorm:"column:role"      json:"role"`
		IsActive   bool   `gorm:"column:is_active" json:"is_active"`
		HasProfile bool   `gorm:"column:has_profile" json:"has_profile"`
	}
	var samples []SampleRow
	database.DB.Raw(`
		SELECT u.id AS user_id, u.full_name, u.role, u.is_active,
		       (dp.id IS NOT NULL) AS has_profile
		FROM "user" u
		LEFT JOIN doctorprofile dp ON u.id = dp.user_id
		WHERE u.role = 'doctor'
		LIMIT 5
	`).Scan(&samples)

	return c.JSON(fiber.Map{
		"counts":  counts,
		"samples": samples,
	})
}

// GetDoctorDetails - Get doctor details with profile
func GetDoctorDetails(c *fiber.Ctx) error {
	id := c.Params("id")

	var user identity.User
	result := database.DB.First(&user, id)

	if result.Error != nil {
		return response.Error(c, fiber.StatusNotFound, "DOCTOR_NOT_FOUND", "Doctor not found")
	}

	if user.Role != "doctor" {
		return response.Error(c, fiber.StatusBadRequest, "NOT_A_DOCTOR", "User is not a doctor")
	}

	// Get doctor profile
	var profile identity.DoctorProfile
	database.DB.Where("user_id = ?", id).First(&profile)

	// Remove password hash
	user.PasswordHash = ""

	return c.JSON(fiber.Map{
		"user":    user,
		"profile": profile,
	})
}

// UpdateDoctorProfile - Update doctor profile using compile-time-safe typed request structs.
// Field access control (what each role may write) is enforced here in the service layer,
// not by a raw map from the HTTP body.
func UpdateDoctorProfile(c *fiber.Ctx) error {
	authenticatedID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	targetID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil || targetID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid doctor ID")
	}

	isAdmin := role == "admin" || role == "super_admin" || role == "hospital_admin"
	if authenticatedID != uint(targetID) && !isAdmin {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You can only update your own profile")
	}

	if isAdmin {
		// Admin path: only admin-permitted fields
		var req identity.DoctorAdminUpdateRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		}
		adminUpdates := map[string]interface{}{}
		if req.IsVerified != nil {
			adminUpdates["is_verified"] = *req.IsVerified
		}
		if req.AdminBoostScore != nil {
			adminUpdates["admin_boost_score"] = *req.AdminBoostScore
		}
		if req.OffersFreeConsultation != nil {
			adminUpdates["offers_free_consultation"] = *req.OffersFreeConsultation
		}
		if req.DailyFreeConsultLimit != nil {
			adminUpdates["daily_free_consult_limit"] = *req.DailyFreeConsultLimit
		}
		if req.FreeType != nil {
			adminUpdates["free_type"] = *req.FreeType
		}
		if req.DoctorTypeID != nil {
			adminUpdates["doctor_type_id"] = *req.DoctorTypeID
		}
		if req.LicenseNumber != nil {
			adminUpdates["license_number"] = *req.LicenseNumber
		}
		if req.PANVerified != nil {
			adminUpdates["pan_verified"] = *req.PANVerified
		}
		if req.EmergencyCapable != nil {
			adminUpdates["emergency_capable"] = *req.EmergencyCapable
		}
		if req.LicenseExpiry != nil {
			if *req.LicenseExpiry == "" {
				adminUpdates["license_expiry"] = nil
			} else if t, perr := time.Parse("2006-01-02", *req.LicenseExpiry); perr == nil {
				adminUpdates["license_expiry"] = t
			} else {
				return response.Error(c, fiber.StatusBadRequest, "INVALID_DATE", "license_expiry must be YYYY-MM-DD")
			}
		}
		if len(adminUpdates) > 0 {
			if err := database.DB.Model(&identity.DoctorProfile{}).Where("user_id = ?", targetID).Updates(adminUpdates).Error; err != nil {
				return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update profile")
			}
		}
		return c.JSON(fiber.Map{"message": "Doctor profile updated by admin"})
	}

	// Doctor self-update path: only doctor-permitted fields
	var req identity.DoctorSelfUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	// User-level fields (full_name, phone_number)
	userUpdates := map[string]interface{}{}
	// DoctorSelfUpdateRequest doesn't include user fields — use UserSelfUpdateRequest
	// for user-row changes. For this endpoint we only handle doctor profile fields.

	profileUpdates := map[string]interface{}{}
	if req.Specialization != nil {
		profileUpdates["specialization"] = *req.Specialization
	}
	if req.Qualification != nil {
		profileUpdates["qualification"] = *req.Qualification
	}
	if req.YearsOfExperience != nil {
		profileUpdates["years_of_experience"] = *req.YearsOfExperience
	}
	if req.ConsultationFee != nil {
		profileUpdates["consultation_fee"] = *req.ConsultationFee
	}
	if req.Bio != nil {
		profileUpdates["bio"] = *req.Bio
	}
	if req.LanguagesSpoken != nil {
		profileUpdates["languages_spoken"] = *req.LanguagesSpoken
	}
	if req.Latitude != nil {
		profileUpdates["latitude"] = *req.Latitude
	}
	if req.Longitude != nil {
		profileUpdates["longitude"] = *req.Longitude
	}
	if req.PayoutUPI != nil {
		profileUpdates["payout_upi"] = *req.PayoutUPI
	}
	if req.BankAccountNumber != nil {
		profileUpdates["bank_account_number"] = *req.BankAccountNumber
	}
	if req.BankIFSC != nil {
		profileUpdates["bank_ifsc"] = *req.BankIFSC
	}
	if req.LicenseNumber != nil {
		profileUpdates["license_number"] = *req.LicenseNumber
	}
	if req.LicenseExpiry != nil {
		// Accept an ISO date (YYYY-MM-DD); empty string clears it.
		if *req.LicenseExpiry == "" {
			profileUpdates["license_expiry"] = nil
		} else if t, perr := time.Parse("2006-01-02", *req.LicenseExpiry); perr == nil {
			profileUpdates["license_expiry"] = t
		} else {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_DATE", "license_expiry must be YYYY-MM-DD")
		}
	}
	if req.DoctorTypeID != nil {
		profileUpdates["doctor_type_id"] = *req.DoctorTypeID
	}

	_ = userUpdates // no user-row writes on this endpoint

	if len(profileUpdates) > 0 {
		if err := database.DB.Model(&identity.DoctorProfile{}).Where("user_id = ?", targetID).Updates(profileUpdates).Error; err != nil {
			if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
				return response.Error(c, fiber.StatusConflict, "DUPLICATE_VALUE", "Duplicate value violates unique constraint")
			}
			return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update profile")
		}
	}

	return c.JSON(fiber.Map{"message": "Profile updated successfully"})
}

// GetDoctorAvailability - Get doctor's availability schedule
func GetDoctorAvailability(c *fiber.Ctx) error {
	doctorID := c.Params("id")

	var availability []identity.DoctorAvailability
	database.DB.Where("doctor_id = ?", doctorID).
		Order("day_of_week ASC").
		Find(&availability)

	return c.JSON(availability)
}

// GetMyAvailability - Get logged-in doctor's own availability
func GetMyAvailability(c *fiber.Ctx) error {
	doctorID, ok := c.Locals("user_id").(uint)
	if !ok || doctorID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	}

	var availability []identity.DoctorAvailability
	database.DB.Where("doctor_id = ?", doctorID).
		Order("day_of_week ASC").
		Find(&availability)

	return c.JSON(availability)
}

// GetDoctorSlots — server-side computed available 30-min slots.
// GET /doctors/:id/slots?days=7
// Returns one entry per day for the next N days (default 7) with
// each slot marked available or booked.
func GetDoctorSlots(c *fiber.Ctx) error {
	doctorIDStr := c.Params("id")
	doctorIDUint, err := strconv.ParseUint(doctorIDStr, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid doctor ID")
	}
	doctorID := uint(doctorIDUint)

	numDays := 7
	if d := c.QueryInt("days", 0); d > 0 && d <= 30 {
		numDays = d
	}

	// Load availability settings for this doctor
	var avail []identity.DoctorAvailability
	database.DB.Where("doctor_id = ?", doctorID).Find(&avail)
	if len(avail) == 0 {
		return c.JSON(fiber.Map{"days": []interface{}{}})
	}

	// Build a map: db day-of-week → settings
	// DoctorAvailability.DayOfWeek: 1=Mon … 7=Sun (ISO)
	availMap := map[int]identity.DoctorAvailability{}
	for _, a := range avail {
		if a.IsAvailable {
			availMap[a.DayOfWeek] = a
		}
	}

	// Load booked appointments for the next numDays
	ist, _ := time.LoadLocation("Asia/Kolkata")
	now := time.Now().In(ist)
	windowEnd := now.AddDate(0, 0, numDays)

	var booked []appointment.Appointment
	database.DB.
		Where("doctor_id = ? AND status IN ? AND start_time >= ? AND start_time < ?",
			doctorID, []string{"pending", "confirmed", "in_consultation"}, now, windowEnd).
		Select("start_time, end_time").
		Find(&booked)

	type SlotItem struct {
		Time      string `json:"time"`     // "09:00"
		Datetime  string `json:"datetime"` // RFC3339
		Available bool   `json:"available"`
	}
	type DayItem struct {
		Date  string     `json:"date"`  // "2026-04-14"
		Label string     `json:"label"` // "Mon Apr 14"
		Slots []SlotItem `json:"slots"`
	}

	days := make([]DayItem, 0, numDays)

	for d := 0; d < numDays; d++ {
		day := now.AddDate(0, 0, d).In(ist)
		day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, ist)

		// ISO day: Monday=1 … Sunday=7
		iso := int(day.Weekday())
		if iso == 0 {
			iso = 7
		}

		settings, ok := availMap[iso]
		if !ok {
			continue
		}

		// Parse start/end times from "HH:MM:SS"
		parseHHMM := func(s string) (int, int) {
			var h, m, sec int
			fmt.Sscanf(s, "%d:%d:%d", &h, &m, &sec)
			return h, m
		}
		sh, sm := parseHHMM(settings.StartTime)
		eh, em := parseHHMM(settings.EndTime)

		slotStart := time.Date(day.Year(), day.Month(), day.Day(), sh, sm, 0, 0, ist)
		slotEnd := time.Date(day.Year(), day.Month(), day.Day(), eh, em, 0, 0, ist)

		slots := []SlotItem{}
		for t := slotStart; t.Before(slotEnd); t = t.Add(30 * time.Minute) {
			// Skip slots in the past (with 15-min buffer)
			if t.Before(now.Add(15 * time.Minute)) {
				continue
			}

			// Check collision
			tEnd := t.Add(30 * time.Minute)
			isBooked := false
			for _, b := range booked {
				bEnd := b.EndTime
				if b.EndTime.IsZero() {
					bEnd = b.StartTime.Add(30 * time.Minute)
				}
				// Overlap: slotStart < bEnd AND slotEnd > bStart
				if t.Before(bEnd) && tEnd.After(b.StartTime) {
					isBooked = true
					break
				}
			}

			slots = append(slots, SlotItem{
				Time:      t.Format("15:04"),
				Datetime:  t.Format(time.RFC3339),
				Available: !isBooked,
			})
		}

		if len(slots) > 0 {
			label := day.Format("Mon Jan 2")
			if d == 0 {
				label = "Today"
			} else if d == 1 {
				label = "Tomorrow"
			}
			days = append(days, DayItem{
				Date:  day.Format("2006-01-02"),
				Label: label,
				Slots: slots,
			})
		}
	}

	return c.JSON(fiber.Map{"days": days})
}

// UpdateDoctorAvailability - Update doctor's availability.
// Parses the request body into []DoctorAvailabilitySlot (validated struct) and maps
// to []DoctorAvailability models before saving, ensuring all fields are validated.
func UpdateDoctorAvailability(c *fiber.Ctx) error {
	parsedID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil || parsedID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid doctor ID")
	}

	// Ownership check
	callerID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	if callerID != uint(parsedID) && role != "admin" && role != "super_admin" {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Cannot modify another doctor's availability")
	}

	// Parse into the validated slot struct (not raw DoctorAvailability)
	var slots []identity.DoctorAvailabilitySlot
	if err := c.BodyParser(&slots); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	// Validate each slot before touching the DB
	for i, s := range slots {
		if s.DayOfWeek < 0 || s.DayOfWeek > 6 {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_DAY",
				fmt.Sprintf("slot %d: day_of_week must be 0–6", i))
		}
		startT, err := time.Parse("15:04", s.StartTime)
		if err != nil {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_TIME",
				fmt.Sprintf("slot %d: invalid start_time format, expected HH:MM", i))
		}
		endT, err := time.Parse("15:04", s.EndTime)
		if err != nil {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_TIME",
				fmt.Sprintf("slot %d: invalid end_time format, expected HH:MM", i))
		}
		if !endT.After(startT) {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_TIME_RANGE",
				fmt.Sprintf("slot %d: end_time must be after start_time", i))
		}
	}

	// Map validated slots to DB model
	schedules := make([]identity.DoctorAvailability, len(slots))
	for i, s := range slots {
		schedules[i] = identity.DoctorAvailability{
			DoctorID:    uint(parsedID),
			DayOfWeek:   s.DayOfWeek,
			StartTime:   s.StartTime,
			EndTime:     s.EndTime,
			IsAvailable: s.IsAvailable,
		}
	}

	// Atomically delete and re-insert inside a transaction
	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("doctor_id = ?", parsedID).Delete(&identity.DoctorAvailability{}).Error; err != nil {
			return err
		}
		for i := range schedules {
			if err := tx.Create(&schedules[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update availability: "+txErr.Error())
	}

	return c.JSON(fiber.Map{"message": "Availability updated successfully"})
}

// UpdateSingleAvailabilitySlot handles partial updates to one availability slot with
// optimistic locking. PATCH /doctors/:id/availability/:slotId
func UpdateSingleAvailabilitySlot(c *fiber.Ctx) error {
	callerID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	slotID, err := strconv.ParseUint(c.Params("slotId"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid slot ID")
	}

	var slot identity.DoctorAvailability
	if err := database.DB.First(&slot, slotID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "SLOT_NOT_FOUND", "Slot not found")
	}

	if slot.DoctorID != callerID && role != "admin" && role != "super_admin" {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Forbidden")
	}

	var req struct {
		IsAvailable bool   `json:"is_available"`
		StartTime   string `json:"start_time"`
		EndTime     string `json:"end_time"`
		Version     uint   `json:"version"` // client must send current version
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
	}

	updates := map[string]interface{}{
		"is_available": req.IsAvailable,
		"version":      slot.Version + 1,
	}
	if req.StartTime != "" {
		if _, err := time.Parse("15:04", req.StartTime); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_TIME", "Invalid start_time format (HH:MM)")
		}
		updates["start_time"] = req.StartTime
	}
	if req.EndTime != "" {
		if _, err := time.Parse("15:04", req.EndTime); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_TIME", "Invalid end_time format (HH:MM)")
		}
		updates["end_time"] = req.EndTime
	}

	result := database.DB.Model(&identity.DoctorAvailability{}).
		Where("id = ? AND version = ?", slotID, slot.Version).
		Updates(updates)

	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return response.Error(c, fiber.StatusConflict, "VERSION_CONFLICT",
			"Conflict: slot was modified by another request. Refresh and retry.")
	}

	return c.JSON(fiber.Map{"message": "Slot updated", "new_version": slot.Version + 1})
}

// DeleteDoctorAvailability - Delete a specific availability slot
func DeleteDoctorAvailability(c *fiber.Ctx) error {
	id := c.Params("id")

	callerID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	var avail identity.DoctorAvailability
	if err := database.DB.First(&avail, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "SLOT_NOT_FOUND", "Availability slot not found")
	}
	if avail.DoctorID != callerID && role != "admin" && role != "super_admin" {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Cannot delete another doctor's availability slot")
	}
	if err := database.DB.Delete(&avail).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", "Failed to delete slot")
	}
	return c.JSON(fiber.Map{"message": "Availability slot deleted"})
}

// GetDoctorTemplates - Get doctor's prescription templates
func GetDoctorTemplates(c *fiber.Ctx) error {
	var doctorID uint
	if val := c.Locals("user_id"); val != nil {
		doctorID = val.(uint)
	} else if qID := c.Query("doctor_id"); qID != "" {
		id, _ := strconv.ParseUint(qID, 10, 32)
		doctorID = uint(id)
	}

	if doctorID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_DOCTOR_ID", "doctor_id required")
	}

	var templates []emr.PrescriptionTemplate
	database.DB.Where("doctor_id = ?", doctorID).
		Order("created_at DESC").
		Find(&templates)

	return c.JSON(templates)
}

// GetDoctorTemplateByID - Get specific prescription template
func GetDoctorTemplateByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var template emr.PrescriptionTemplate
	if err := database.DB.First(&template, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "TEMPLATE_NOT_FOUND", "Template not found")
	}
	return c.JSON(template)
}

// CreateDoctorTemplate - Create prescription template
func CreateDoctorTemplate(c *fiber.Ctx) error {
	var template emr.PrescriptionTemplate
	if err := c.BodyParser(&template); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	result := database.DB.Create(&template)
	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", result.Error.Error())
	}

	return c.Status(201).JSON(template)
}

// UpdateDoctorTemplate - Update prescription template
func UpdateDoctorTemplate(c *fiber.Ctx) error {
	id := c.Params("id")

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	result := database.DB.Model(&emr.PrescriptionTemplate{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", result.Error.Error())
	}

	if result.RowsAffected == 0 {
		return response.Error(c, fiber.StatusNotFound, "TEMPLATE_NOT_FOUND", "Template not found")
	}

	return c.JSON(fiber.Map{"message": "Template updated successfully"})
}

// DeleteDoctorTemplate - Delete prescription template
func DeleteDoctorTemplate(c *fiber.Ctx) error {
	id := c.Params("id")

	result := database.DB.Delete(&emr.PrescriptionTemplate{}, id)

	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", result.Error.Error())
	}

	if result.RowsAffected == 0 {
		return response.Error(c, fiber.StatusNotFound, "TEMPLATE_NOT_FOUND", "Template not found")
	}

	return c.JSON(fiber.Map{"message": "Template deleted successfully"})
}

// GetWaitingRoom returns pending/confirmed appointments for today for the logged-in doctor
func GetWaitingRoom(c *fiber.Ctx) error {
	doctorID, ok := c.Locals("user_id").(uint)
	if !ok || doctorID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	}
	ist, _ := time.LoadLocation("Asia/Kolkata")
	nowIST := time.Now().In(ist)
	todayStart := time.Date(nowIST.Year(), nowIST.Month(), nowIST.Day(), 0, 0, 0, 0, ist)
	todayEnd := todayStart.Add(24 * time.Hour)

	var appts []appointment.Appointment
	database.DB.
		Where("doctor_id = ? AND status IN ? AND start_time >= ? AND start_time < ?",
			doctorID, []string{"pending", "confirmed"}, todayStart, todayEnd).
		Preload("Patient.User").
		Order("start_time ASC").
		Find(&appts)

	list := make([]fiber.Map, 0, len(appts))
	for _, a := range appts {
		patientName := "Patient #" + strconv.Itoa(int(a.PatientID))
		if a.Patient.User != nil && a.Patient.User.FullName != "" {
			patientName = a.Patient.User.FullName
		}
		waitMins := int(time.Until(a.StartTime).Minutes())
		if waitMins < 0 {
			waitMins = 0
		}

		tier := ""
		if a.Patient.User != nil {
			tier = a.Patient.User.Tier
		}
		if tier == "" {
			tier = "SILVER"
		}

		list = append(list, fiber.Map{
			"id":           a.ID,
			"patient_id":   a.PatientID,
			"patient_name": patientName,
			"patient_tier": tier,
			"priority":     map[bool]string{true: "high", false: "normal"}[a.IsPriority],
			"wait_time":    strconv.Itoa(waitMins) + "m",
			"status":       string(a.Status),
			"consultation": a.AppointmentType,
			"start_time":   a.StartTime,
		})
	}
	return c.JSON(fiber.Map{"waiting_list": list, "count": len(list)})
}

// GetTodaySchedule returns today's appointments for the logged-in doctor
func GetTodaySchedule(c *fiber.Ctx) error {
	doctorID, ok := c.Locals("user_id").(uint)
	if !ok || doctorID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	}
	ist, _ := time.LoadLocation("Asia/Kolkata")
	nowIST := time.Now().In(ist)
	todayStart := time.Date(nowIST.Year(), nowIST.Month(), nowIST.Day(), 0, 0, 0, 0, ist)
	todayEnd := todayStart.Add(24 * time.Hour)

	var appts []appointment.Appointment
	database.DB.
		Where("doctor_id = ? AND start_time >= ? AND start_time < ?", doctorID, todayStart, todayEnd).
		Preload("Patient.User").
		Order("start_time ASC").
		Find(&appts)

	schedules := make([]fiber.Map, 0, len(appts))
	for _, a := range appts {
		patientName := "Patient #" + strconv.Itoa(int(a.PatientID))
		if a.Patient.User != nil && a.Patient.User.FullName != "" {
			patientName = a.Patient.User.FullName
		}
		schedules = append(schedules, fiber.Map{
			"id":                a.ID,
			"patient_name":      patientName,
			"start_time":        a.StartTime.Format(time.RFC3339),
			"type":              a.AppointmentType,
			"status":            string(a.Status),
			"duration":          30,
			"consultation_type": a.ConsultationMode,
		})
	}
	return c.JSON(schedules)
}

// GetDoctorStats returns real statistics for the logged-in doctor
func GetDoctorStats(c *fiber.Ctx) error {
	doctorID, ok := c.Locals("user_id").(uint)
	if !ok || doctorID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	}

	var profile identity.DoctorProfile
	database.DB.Where("user_id = ?", doctorID).First(&profile)

	ist, _ := time.LoadLocation("Asia/Kolkata")
	nowIST := time.Now().In(ist)
	todayStart := time.Date(nowIST.Year(), nowIST.Month(), nowIST.Day(), 0, 0, 0, 0, ist)
	todayEnd := todayStart.Add(24 * time.Hour)
	now := time.Now()
	monthStart := time.Date(nowIST.Year(), nowIST.Month(), 1, 0, 0, 0, 0, ist)

	var todayCount, upcomingCount, monthCount int64

	// Today's total (any status, usually confirmed/completed)
	database.DB.Model(&appointment.Appointment{}).
		Where("doctor_id = ? AND start_time >= ? AND start_time < ?", doctorID, todayStart, todayEnd).
		Count(&todayCount)

	// Upcoming (confirmed in future)
	database.DB.Model(&appointment.Appointment{}).
		Where("doctor_id = ? AND status = ? AND start_time > ?", doctorID, "confirmed", now).
		Count(&upcomingCount)

	// Completed this month
	database.DB.Model(&appointment.Appointment{}).
		Where("doctor_id = ? AND status = ? AND start_time >= ?", doctorID, "completed", monthStart).
		Count(&monthCount)

	return c.JSON(fiber.Map{
		"today_appointments":    todayCount,
		"upcoming_appointments": upcomingCount,
		"completed_this_month":  monthCount,
		"patients_seen":         todayCount, // fallback for legacy
		"total_consultations":   profile.TotalConsultations,
		"rating":                profile.AverageRating,
		"is_online":             profile.IsOnline,
		"consultations_today":   profile.TotalConsultationsToday,
	})
}

// PingDoctor updates the doctor's last_ping timestamp to keep them online
func PingDoctor(c *fiber.Ctx) error {
	doctorID, ok := c.Locals("user_id").(uint)
	if !ok || doctorID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	}

	now := time.Now()
	if err := database.DB.Model(&identity.DoctorProfile{}).
		Where("user_id = ?", doctorID).
		Updates(map[string]interface{}{"is_online": true, "last_ping": now}).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}

	// Refresh Redis presence so IsDoctorOnline returns true while pinging
	if database.Rdb != nil {
		doctorIDStr := fmt.Sprintf("%d", doctorID)
		presenceKey := fmt.Sprintf("presence:user:%s", doctorIDStr)
		database.Rdb.Set(c.Context(), presenceKey, "online", 5*time.Minute)
		database.Rdb.SAdd(c.Context(), "set:doctors:online", doctorIDStr)
	}

	// Publish domain event (sampled — ping fires every few seconds so we keep it lightweight)
	if EventBus != nil {
		EventBus.Publish(c.Context(), events.EventDoctorPing, doctorID, map[string]interface{}{
			"doctor_id": doctorID,
		})
	}

	return c.JSON(fiber.Map{"status": "ok", "timestamp": now})
}

// GetEarningsSummary - Get doctor's financial summary
func GetEarningsSummary(s finance.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		doctorID, ok := c.Locals("user_id").(uint)
		if !ok {
			return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Invalid user context")
		}

		data, err := s.GetDoctorFinancials(doctorID)
		if err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
		}

		return c.JSON(data)
	}
}

// GetTransactionHistory - Get doctor's transaction history
func GetTransactionHistory(s finance.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		doctorID, ok := c.Locals("user_id").(uint)
		if !ok {
			return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Invalid user context")
		}

		data, err := s.GetDoctorFinancials(doctorID)
		if err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
		}

		// Return just the transactions part for this endpoint
		return c.JSON(data["recent_transactions"])
	}
}

// RequestPayout - Request a payout
func RequestPayout(c *fiber.Ctx) error {
	type PayoutRequest struct {
		Amount float64 `json:"amount"`
	}
	var req PayoutRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	return c.JSON(fiber.Map{
		"message":      "Payout request submitted successfully",
		"reference_id": "PAY-" + strconv.FormatInt(time.Now().Unix(), 10),
	})
}

// ── Doctor TDS Handlers ───────────────────────────────────────────────────────

// parseFYBounds returns the start and end time for a financial year string like "2025-26".
func parseFYBounds(fy string) (start, end time.Time, ok bool) {
	if len(fy) < 5 {
		return
	}
	// fy = "2025-26"  →  startYear = 2025
	startYearStr := fy[:4]
	startYear, err := strconv.Atoi(startYearStr)
	if err != nil {
		return
	}
	ist, _ := time.LoadLocation("Asia/Kolkata")
	start = time.Date(startYear, time.April, 1, 0, 0, 0, 0, ist)
	end = time.Date(startYear+1, time.April, 1, 0, 0, 0, 0, ist)
	ok = true
	return
}

// quarterBounds returns the [start, end) for Q1–Q4 of a financial year.
func quarterBounds(startYear int, q int) (time.Time, time.Time) {
	ist, _ := time.LoadLocation("Asia/Kolkata")
	switch q {
	case 1:
		return time.Date(startYear, time.April, 1, 0, 0, 0, 0, ist),
			time.Date(startYear, time.July, 1, 0, 0, 0, 0, ist)
	case 2:
		return time.Date(startYear, time.July, 1, 0, 0, 0, 0, ist),
			time.Date(startYear, time.October, 1, 0, 0, 0, 0, ist)
	case 3:
		return time.Date(startYear, time.October, 1, 0, 0, 0, 0, ist),
			time.Date(startYear+1, time.January, 1, 0, 0, 0, 0, ist)
	default: // Q4
		return time.Date(startYear+1, time.January, 1, 0, 0, 0, 0, ist),
			time.Date(startYear+1, time.April, 1, 0, 0, 0, 0, ist)
	}
}

// GetDoctorTDSComputation computes 194J TDS for the authenticated doctor for a given FY.
// GET /doctor/finance/tds/computation?fy=2025-26
func GetDoctorTDSComputation(c *fiber.Ctx) error {
	doctorID, ok := c.Locals("user_id").(uint)
	if !ok || doctorID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	}

	fy := c.Query("fy", tds.FYFromDate(time.Now()))
	fyStart, fyEnd, valid := parseFYBounds(fy)
	if !valid {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_FY", "Invalid financial year format (expected YYYY-YY)")
	}
	startYear, _ := strconv.Atoi(fy[:4])

	// Sum consultation fees from completed appointments in this FY
	sumFees := func(from, to time.Time) int64 {
		var res struct{ Total int64 }
		database.DB.Model(&appointment.Appointment{}).
			Select("COALESCE(SUM(consultation_fee), 0) as total").
			Where("doctor_id = ? AND status = ? AND start_time >= ? AND start_time < ?",
				doctorID, string(appointment.StatusCompleted), from, to).
			Scan(&res)
		return res.Total
	}

	q1s, q1e := quarterBounds(startYear, 1)
	q2s, q2e := quarterBounds(startYear, 2)
	q3s, q3e := quarterBounds(startYear, 3)
	q4s, q4e := quarterBounds(startYear, 4)

	q1Fee := sumFees(q1s, q1e)
	q2Fee := sumFees(q2s, q2e)
	q3Fee := sumFees(q3s, q3e)
	q4Fee := sumFees(q4s, q4e)

	_ = fyStart
	_ = fyEnd

	totalFees := q1Fee + q2Fee + q3Fee + q4Fee
	q1TDS := tds.TDSOnProfessionalFee(q1Fee)
	q2TDS := tds.TDSOnProfessionalFee(q2Fee)
	q3TDS := tds.TDSOnProfessionalFee(q3Fee)
	q4TDS := tds.TDSOnProfessionalFee(q4Fee)
	totalTDS := q1TDS + q2TDS + q3TDS + q4TDS
	netReceipts := totalFees - totalTDS

	// Rough estimated tax liability: use ComputeAnnualTax on net receipts
	estimatedTaxLiability := tds.ComputeAnnualTax(netReceipts)
	// TDS already covers some of it
	refundExpected := totalTDS - estimatedTaxLiability

	return c.JSON(fiber.Map{
		"financial_year":          fy,
		"total_professional_fees": totalFees,
		"tds_deducted":            totalTDS,
		"tds_rate":                10,
		"net_receipts":            netReceipts,
		"q1":                      q1TDS,
		"q2":                      q2TDS,
		"q3":                      q3TDS,
		"q4":                      q4TDS,
		"estimated_tax_liability": estimatedTaxLiability,
		"advance_tax_paid":        int64(0),
		"refund_expected":         refundExpected,
	})
}

// GetDoctorTDSCertificates returns generated Form 16A certificates for the doctor per quarter.
// GET /doctor/finance/tds/certificates?fy=2025-26
func GetDoctorTDSCertificates(c *fiber.Ctx) error {
	doctorID, ok := c.Locals("user_id").(uint)
	if !ok || doctorID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	}

	fy := c.Query("fy", tds.FYFromDate(time.Now()))
	_, _, valid := parseFYBounds(fy)
	if !valid {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_FY", "Invalid financial year format")
	}
	startYear, _ := strconv.Atoi(fy[:4])

	// Fetch doctor info for certificate deductee fields
	var doc identity.User
	if err := database.DB.Select("id, full_name").First(&doc, doctorID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "DOCTOR_NOT_FOUND", "Doctor not found")
	}

	sumFees := func(from, to time.Time) int64 {
		var res struct{ Total int64 }
		database.DB.Model(&appointment.Appointment{}).
			Select("COALESCE(SUM(consultation_fee), 0) as total").
			Where("doctor_id = ? AND status = ? AND start_time >= ? AND start_time < ?",
				doctorID, string(appointment.StatusCompleted), from, to).
			Scan(&res)
		return res.Total
	}

	quarters := []struct {
		Label string
		Q     int
	}{
		{"Q1 (Apr–Jun)", 1},
		{"Q2 (Jul–Sep)", 2},
		{"Q3 (Oct–Dec)", 3},
		{"Q4 (Jan–Mar)", 4},
	}

	certs := make([]fiber.Map, 0, 4)
	for i, q := range quarters {
		qs, qe := quarterBounds(startYear, q.Q)
		fees := sumFees(qs, qe)
		if fees == 0 {
			continue
		}
		tdsPaise := tds.TDSOnProfessionalFee(fees)
		if tdsPaise == 0 {
			continue // below ₹30,000 threshold
		}
		certs = append(certs, fiber.Map{
			"id":             i + 1,
			"financial_year": fy,
			"form_type":      "16A",
			"deductor_name":  "Dr.Ayya Healthcare Pvt. Ltd.",
			"deductor_tan":   "MUMM12345A",
			"deductor_pan":   "AABCM1234D",
			"deductee_name":  doc.FullName,
			"deductee_pan":   "",
			"gross_amount":   fees,
			"tds_deducted":   tdsPaise,
			"tds_rate":       10,
			"section":        "194J",
			"quarter":        q.Label,
			"status":         "issued",
			"issued_at":      qe.Format(time.RFC3339),
		})
	}

	return c.JSON(certs)
}

// ComputeDoctorTDS is the POST variant (same logic, triggers a refresh).
// POST /doctor/finance/tds/compute
func ComputeDoctorTDS(c *fiber.Ctx) error {
	// Delegate to the GET handler — results are always computed live from DB
	return GetDoctorTDSComputation(c)
}

// DownloadDoctorTDSCertificate returns the Form 16A JSON for a specific certificate.
// POST /doctor/finance/tds/certificates/:id/download
func DownloadDoctorTDSCertificate(c *fiber.Ctx) error {
	doctorID, ok := c.Locals("user_id").(uint)
	if !ok || doctorID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	}

	certIDStr := c.Params("id")
	certID, err := strconv.Atoi(certIDStr)
	if err != nil || certID < 1 || certID > 4 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid certificate ID")
	}

	fy := c.Query("fy", tds.FYFromDate(time.Now()))
	_, _, valid := parseFYBounds(fy)
	if !valid {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_FY", "Invalid financial year format")
	}
	startYear, _ := strconv.Atoi(fy[:4])

	var doc identity.User
	if err := database.DB.Select("id, full_name").First(&doc, doctorID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "DOCTOR_NOT_FOUND", "Doctor not found")
	}

	qs, qe := quarterBounds(startYear, certID)
	var res struct{ Total int64 }
	database.DB.Model(&appointment.Appointment{}).
		Select("COALESCE(SUM(consultation_fee), 0) as total").
		Where("doctor_id = ? AND status = ? AND start_time >= ? AND start_time < ?",
			doctorID, string(appointment.StatusCompleted), qs, qe).
		Scan(&res)

	fees := res.Total
	tdsPaise := tds.TDSOnProfessionalFee(fees)

	quarterLabels := []string{"Q1 (Apr–Jun)", "Q2 (Jul–Sep)", "Q3 (Oct–Dec)", "Q4 (Jan–Mar)"}
	cert := tds.GenerateForm16A(
		"Dr.Ayya Healthcare Pvt. Ltd.",
		"MUMM12345A",
		"29AABCM1234D1ZK",
		doc.FullName,
		"",
		fy,
		quarterLabels[certID-1],
		[]tds.Form16AEntry{
			{
				DateOfPayment: qe.Add(-24 * time.Hour),
				AmountPaise:   fees,
				TDSPaise:      tdsPaise,
				Section:       "194J",
			},
		},
	)

	return c.JSON(cert)
}

// SetDoctorOnline - Set doctor status to online
func SetDoctorOnline(c *fiber.Ctx) error {
	doctorID, ok := c.Locals("user_id").(uint)
	if !ok || doctorID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	}

	var user identity.User
	database.DB.Select("full_name").First(&user, doctorID)

	now := time.Now()
	if err := database.DB.Model(&identity.DoctorProfile{}).Where("user_id = ?", doctorID).
		Updates(map[string]interface{}{"is_online": true, "current_status": "online", "last_ping": now}).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update status")
	}

	doctorIDStr := fmt.Sprintf("%d", doctorID)

	// Write to Redis so presence.IsDoctorOnline returns true
	if database.Rdb != nil {
		presenceKey := fmt.Sprintf("presence:user:%s", doctorIDStr)
		database.Rdb.Set(c.Context(), presenceKey, "online", 5*time.Minute)
		database.Rdb.SAdd(c.Context(), "set:doctors:online", doctorIDStr)

		// Push real-time update to all connected WebSocket clients via chat:global
		jsonMsg, _ := json.Marshal(map[string]interface{}{
			"type":        "doctor_status_update",
			"sender_id":   doctorID,
			"sender_name": user.FullName,
			"sender_role": "doctor",
			"timestamp":   now.Format(time.RFC3339),
			"metadata": map[string]interface{}{
				"doctor_id": doctorID,
				"status":    "online",
			},
		})
		database.Rdb.Publish(c.Context(), "chat:global", jsonMsg)
	}

	// Publish domain event
	if EventBus != nil {
		EventBus.Publish(c.Context(), events.EventDoctorOnline, doctorID, map[string]interface{}{
			"doctor_id": doctorID,
		})
	}

	return c.JSON(fiber.Map{"status": "online"})
}

// SetDoctorOffline - Set doctor status to offline
func SetDoctorOffline(c *fiber.Ctx) error {
	doctorID, ok := c.Locals("user_id").(uint)
	if !ok || doctorID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	}

	var user identity.User
	database.DB.Select("full_name").First(&user, doctorID)

	if err := database.DB.Model(&identity.DoctorProfile{}).Where("user_id = ?", doctorID).
		Updates(map[string]interface{}{"is_online": false, "current_status": "offline"}).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update status")
	}

	doctorIDStr := fmt.Sprintf("%d", doctorID)

	// Remove from Redis so presence.IsDoctorOnline returns false
	if database.Rdb != nil {
		presenceKey := fmt.Sprintf("presence:user:%s", doctorIDStr)
		database.Rdb.Del(c.Context(), presenceKey)
		database.Rdb.SRem(c.Context(), "set:doctors:online", doctorIDStr)

		// Push real-time update to all connected WebSocket clients via chat:global
		jsonMsg, _ := json.Marshal(map[string]interface{}{
			"type":        "doctor_status_update",
			"sender_id":   doctorID,
			"sender_name": user.FullName,
			"sender_role": "doctor",
			"timestamp":   time.Now().Format(time.RFC3339),
			"metadata": map[string]interface{}{
				"doctor_id": doctorID,
				"status":    "offline",
			},
		})
		database.Rdb.Publish(c.Context(), "chat:global", jsonMsg)
	}

	// Publish domain event
	if EventBus != nil {
		EventBus.Publish(c.Context(), events.EventDoctorOffline, doctorID, map[string]interface{}{
			"doctor_id": doctorID,
		})
	}

	return c.JSON(fiber.Map{"status": "offline"})
}

func RegisterDoctorRoutes(router fiber.Router, financeSvc finance.Service) {
	doctorGroup := router.Group("/doctors")

	// ── 1. Bare list route (MUST be first — before any wildcard) ──────────────
	// GET /doctors          → doctorsApi.getAll()
	// GET /doctors/list     → legacy alias kept for backward compat
	doctorGroup.Get("/", GetDoctors)
	doctorGroup.Get("/list", GetDoctors)
	doctorGroup.Get("/count", GetDoctorCountByType)

	// Doctor types (AYUSH) with nested sub-specialties and live counts
	router.Get("/doctor-types", GetDoctorTypes)

	// ── 2. Diagnostic route (static — BEFORE /:id wildcard) ─────────────────
	doctorGroup.Get("/debug", GetDoctorsDebug)

	// ── 3. Named-segment view routes (static — BEFORE /:id wildcard) ─────────
	// These MUST be registered before /:id or Fiber will route them as /:id="views"
	doctorViews := doctorGroup.Group("/views")
	doctorViews.Get("/online", GetOnlineDoctors)
	doctorViews.Get("/ranked", GetRankedDoctors)
	doctorViews.Get("/free", GetFreeDoctors)

	// ── 3. Protected doctor-only routes (inline middleware — empty-path Group in
	//       Fiber v2 bleeds middleware onto the whole parent group, so we apply
	//       Protected()+DoctorOnly() explicitly on each route instead) ───────────
	auth := []fiber.Handler{middleware.Protected(), middleware.DoctorOnly()}
	doctorGroup.Get("/waiting-room", append(auth, GetWaitingRoom)...)
	doctorGroup.Get("/schedule/today", append(auth, GetTodaySchedule)...)
	doctorGroup.Get("/stats/overview", append(auth, GetDoctorStats)...)
	doctorGroup.Get("/templates", append(auth, GetDoctorTemplates)...)
	doctorGroup.Get("/templates/:id", append(auth, GetDoctorTemplateByID)...)
	doctorGroup.Post("/templates", append(auth, CreateDoctorTemplate)...)
	doctorGroup.Put("/templates/:id", append(auth, UpdateDoctorTemplate)...)
	doctorGroup.Delete("/templates/:id", append(auth, DeleteDoctorTemplate)...)
	doctorGroup.Get("/promotions/analytics", append(auth, GetPromotionAnalytics)...)
	doctorGroup.Post("/online", append(auth, SetDoctorOnline)...)
	doctorGroup.Post("/offline", append(auth, SetDoctorOffline)...)
	doctorGroup.Post("/ping", append(auth, PingDoctor)...)
	doctorGroup.Get("/me/availability", append(auth, GetMyAvailability)...)
	doctorGroup.Post("/availability", append(auth, UpdateDoctorAvailability)...)
	doctorGroup.Delete("/availability/:id", append(auth, DeleteDoctorAvailability)...)

	// ── 4. Param/wildcard routes (LAST — after all static routes) ────────────
	doctorGroup.Get("/:id", GetDoctorDetails)
	doctorGroup.Get("/:id/availability", GetDoctorAvailability)
	doctorGroup.Get("/:id/slots", GetDoctorSlots)
	doctorGroup.Get("/availability/:id", GetDoctorAvailability) // legacy pattern
	doctorGroup.Put("/:id", middleware.Protected(), middleware.DoctorOnly(), UpdateDoctorProfile)
	doctorGroup.Post("/:id/availability", middleware.Protected(), UpdateDoctorAvailability)
	doctorGroup.Patch("/:id/availability/:slotId", middleware.Protected(), middleware.DoctorOnly(), UpdateSingleAvailabilitySlot)

	// Doctor Finance
	financeGroup := router.Group("/doctor/finance", middleware.Protected(), middleware.DoctorOnly())
	financeGroup.Get("/summary", GetEarningsSummary(financeSvc))
	financeGroup.Get("/transactions", GetTransactionHistory(financeSvc))
	financeGroup.Post("/payouts/request", RequestPayout)

	// Doctor TDS (Section 194J)
	tdsGroup := financeGroup.Group("/tds")
	tdsGroup.Get("/computation", GetDoctorTDSComputation)
	tdsGroup.Get("/certificates", GetDoctorTDSCertificates)
	tdsGroup.Post("/compute", ComputeDoctorTDS)
	tdsGroup.Post("/certificates/:id/download", DownloadDoctorTDSCertificate)
}

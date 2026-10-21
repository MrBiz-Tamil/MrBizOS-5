package admin

import (
	"strconv"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/rating"
	"drayya-core/internal/services/visibility"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// AdminGetRatings returns all ratings with optional filters for admin audit.
// GET /admin/ratings?doctor_id=&min_score=&max_score=&page=&limit=
func AdminGetRatings(c *fiber.Ctx) error {
	repo := rating.NewRepository(database.DB)
	svc := rating.NewService(repo)

	doctorID, _ := strconv.ParseUint(c.Query("doctor_id", "0"), 10, 64)
	minScore, _ := strconv.Atoi(c.Query("min_score", "0"))
	maxScore, _ := strconv.Atoi(c.Query("max_score", "0"))
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}

	ratings, total, err := svc.GetAllForAdmin(uint(doctorID), minScore, maxScore, page, limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"ratings": ratings,
		"total":   total,
		"page":    page,
		"limit":   limit,
		"has_more": int64(page*limit) < total,
	}, "Ratings fetched")
}

// AdminHideRating hides a review from the public listing.
// PATCH /admin/ratings/:id/hide  Body: { "reason": "string" }
func AdminHideRating(c *fiber.Ctx) error {
	ratingID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid rating ID")
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil || req.Reason == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "reason is required")
	}

	adminID := c.Locals("user_id").(uint)

	repo := rating.NewRepository(database.DB)
	svc := rating.NewService(repo)

	if err := svc.HideRating(uint(ratingID), adminID, req.Reason); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Rating hidden")
}

// AdminRestoreRating restores a previously hidden review.
// PATCH /admin/ratings/:id/restore
func AdminRestoreRating(c *fiber.Ctx) error {
	ratingID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid rating ID")
	}

	repo := rating.NewRepository(database.DB)
	svc := rating.NewService(repo)

	if err := svc.RestoreRating(uint(ratingID)); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Rating restored")
}

// AdminUpdateDoctor updates admin-controlled fields on a doctor profile.
// PATCH /admin/doctors/:id
// Permitted fields: admin_boost_score, is_featured, is_verified, doctor_type_id,
//
//	offers_free_consultation, daily_free_consult_limit, free_type, total_consultations
func AdminUpdateDoctor(c *fiber.Ctx) error {
	doctorID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid doctor ID")
	}

	var req struct {
		AdminBoostScore         *float64 `json:"admin_boost_score"`
		IsFeatured              *bool    `json:"is_featured"`
		IsVerified              *bool    `json:"is_verified"`
		DoctorTypeID            *uint    `json:"doctor_type_id"`
		OffersFreeConsultation  *bool    `json:"offers_free_consultation"`
		DailyFreeConsultLimit   *int     `json:"daily_free_consult_limit"`
		FreeType                *string  `json:"free_type"`
		TotalConsultations      *int     `json:"total_consultations"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	updates := map[string]interface{}{}
	if req.AdminBoostScore != nil {
		updates["admin_boost_score"] = *req.AdminBoostScore
	}
	if req.IsFeatured != nil {
		updates["is_featured"] = *req.IsFeatured
	}
	if req.IsVerified != nil {
		updates["is_verified"] = *req.IsVerified
	}
	if req.DoctorTypeID != nil {
		updates["doctor_type_id"] = *req.DoctorTypeID
	}
	if req.OffersFreeConsultation != nil {
		updates["offers_free_consultation"] = *req.OffersFreeConsultation
	}
	if req.DailyFreeConsultLimit != nil {
		updates["daily_free_consult_limit"] = *req.DailyFreeConsultLimit
	}
	if req.FreeType != nil {
		updates["free_type"] = *req.FreeType
	}
	if req.TotalConsultations != nil {
		updates["total_consultations"] = *req.TotalConsultations
	}

	if len(updates) == 0 {
		return response.Error(c, fiber.StatusBadRequest, "NO_FIELDS", "No updatable fields provided")
	}

	result := database.DB.Model(&identity.DoctorProfile{}).
		Where("user_id = ?", doctorID).
		Updates(updates)
	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Doctor not found")
	}

	// Log admin action
	adminID, _ := c.Locals("user_id").(uint)
	database.DB.Create(&identity.ActivityLog{
		UserID:    adminID,
		Action:    "admin_update_doctor",
		Entity:    "doctor_profile",
		EntityID:  strconv.FormatUint(doctorID, 10),
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	})

	return response.JSON(c, fiber.StatusOK, nil, "Doctor profile updated")
}

// AdminRecalcDoctorScore triggers an immediate score recalculation for all doctors.
// POST /admin/doctors/:id/recalc-score  (pass "all" as :id to recalculate all)
func AdminRecalcDoctorScore(c *fiber.Ctx) error {
	idParam := c.Params("id")
	if idParam == "all" {
		if err := visibility.RecalculateAllScores(); err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "RECALC_FAILED", err.Error())
		}
		return response.JSON(c, fiber.StatusOK, nil, "All doctor scores recalculated")
	}

	doctorID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid doctor ID")
	}

	const scoreExpr = `(
		(COALESCE(average_rating, 0) / 5.0) * 40.0 +
		LEAST(COALESCE(total_consultations, 0) / 200.0, 1.0) * 20.0 +
		COALESCE(completion_rate, 0) * 20.0 +
		CASE WHEN is_online = true THEN 10.0 ELSE 0.0 END +
		CASE WHEN is_verified = true THEN 5.0 ELSE 0.0 END +
		COALESCE(admin_boost_score, 0) * 5.0
	)`

	if err := database.DB.Model(&identity.DoctorProfile{}).
		Where("user_id = ?", doctorID).
		Update("performance_score", gorm.Expr(scoreExpr)).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "RECALC_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Doctor score recalculated")
}

// GetDoctorPerformanceTable returns all doctor metrics in a sortable table for admin.
// GET /admin/doctors/performance?sort=rating|consultations|completion|score&page=&limit=
func GetDoctorPerformanceTable(c *fiber.Ctx) error {
	type DoctorPerf struct {
		UserID             uint    `json:"user_id"`
		FullName           string  `json:"full_name"`
		Specialization     string  `json:"specialization"`
		DoctorTypeID       *uint   `json:"doctor_type_id"`
		IsVerified         bool    `json:"is_verified"`
		IsFeatured         bool    `json:"is_featured"`
		IsOnline           bool    `json:"is_online"`
		AverageRating      float64 `json:"average_rating"`
		TotalReviews       int     `json:"total_reviews"`
		TotalConsultations int     `json:"total_consultations"`
		WeeklyConsultations int    `json:"weekly_consultations"`
		CompletionRate     float64 `json:"completion_rate"`
		PerformanceScore   float64 `json:"performance_score"`
		AdminBoostScore    float64 `json:"admin_boost_score"`
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 { page = 1 }
	if limit < 1 || limit > 100 { limit = 20 }

	sortMap := map[string]string{
		"rating":       "dp.average_rating DESC",
		"consultations": "dp.total_consultations DESC",
		"completion":   "dp.completion_rate DESC",
		"score":        "dp.performance_score DESC",
		"weekly":       "dp.weekly_consultations DESC",
	}
	sortSQL, ok := sortMap[c.Query("sort", "score")]
	if !ok {
		sortSQL = "dp.performance_score DESC"
	}

	var perfs []DoctorPerf
	var total int64

	q := database.DB.Table("doctorprofile dp").
		Select(`dp.user_id, u.full_name, dp.specialization, dp.doctor_type_id,
			dp.is_verified, dp.is_featured, dp.is_online,
			dp.average_rating, dp.total_reviews, dp.total_consultations,
			dp.weekly_consultations, dp.completion_rate,
			dp.performance_score, dp.admin_boost_score`).
		Joins(`JOIN "user" u ON u.id = dp.user_id`).
		Where(`u.is_active = ? AND u.role IN ?`, true, []string{"doctor", "chief_doctor", "hospital_head_doctor"})

	if search := c.Query("search"); search != "" {
		q = q.Where("LOWER(u.full_name) LIKE LOWER(?) OR LOWER(dp.specialization) LIKE LOWER(?)",
			"%"+search+"%", "%"+search+"%")
	}

	q.Count(&total)

	err := q.Order(sortSQL).Offset((page - 1) * limit).Limit(limit).Scan(&perfs).Error
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	if perfs == nil {
		perfs = []DoctorPerf{}
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"doctors": perfs,
		"total":   total,
		"page":    page,
		"limit":   limit,
		"has_more": int64(page*limit) < total,
	}, "Doctor performance table fetched")
}


package visibility

import (
	"context"
	"fmt"
	"log"

	"drayya-core/internal/audit"
	"drayya-core/internal/database"
	"drayya-core/internal/presence"
	"gorm.io/gorm"
)

// DoctorView represents a doctor profile with additional computed fields.
// JSON field names match the frontend DoctorProfile TypeScript interface exactly.
type DoctorView struct {
	ID                  uint    `json:"id"`
	UserID              uint    `json:"user_id"`
	FullName            string  `json:"full_name"`
	Specialization      string  `json:"specialization"`
	ConsultationFee     int     `json:"consultation_fee"`
	AverageRating       float32 `json:"average_rating"`
	IsOnline            bool    `json:"is_online"`
	IsVerified          bool    `json:"is_verified"`
	IsFeatured          bool    `json:"is_featured"`
	CurrentStatus       string  `json:"current_status"`
	TotalConsultations  int     `json:"total_consultations"`
	TotalReviews        int     `json:"total_reviews"`
	WeeklyConsultations int     `json:"weekly_consultations"`
	CompletionRate      float64 `json:"completion_rate"`
	PerformanceScore    float64 `json:"performance_score,omitempty"`
	ProfilePhotoURL     string  `json:"profile_photo_url"`
	Languages           string  `json:"languages"`
	YearsOfExperience   int     `json:"years_of_experience"`
	Qualification       string  `json:"qualification"`
	DoctorTypeID        *uint   `json:"doctor_type_id"`
}

// getSafeSelect returns the safe column select clause with dialect-aware consultation fee extraction.
func getSafeSelect() string {
	feeExpr := "(doctorprofile.consultation_fees->>'INR')::bigint"
	if database.DB != nil && database.DB.Dialector.Name() == "sqlite" {
		feeExpr = "COALESCE(json_extract(doctorprofile.consultation_fees, '$.INR'), 0)"
	}
	return fmt.Sprintf(`doctorprofile.id, doctorprofile.user_id, doctorprofile.specialization,
		COALESCE(%s, 0) AS consultation_fee, doctorprofile.is_verified, doctorprofile.is_online,
		doctorprofile.is_featured, doctorprofile.current_status, doctorprofile.average_rating,
		doctorprofile.total_consultations, doctorprofile.total_reviews,
		doctorprofile.weekly_consultations, doctorprofile.completion_rate,
		doctorprofile.performance_score, doctorprofile.doctor_type_id,
		COALESCE("user".profile_picture, '')           AS profile_photo_url,
		COALESCE(doctorprofile.languages_spoken, '')   AS languages,
		COALESCE(doctorprofile.years_of_experience, 0) AS years_of_experience,
		COALESCE(doctorprofile.qualification, '')       AS qualification,
		"user".full_name`, feeExpr)
}

// getPerformanceExpr returns the composite score SQL expression with dialect-aware min/least cap.
// Weights: 40% quality (rating), 20% volume (capped), 20% completion rate,
//          10% online bonus, 5% verified bonus, up to 50 admin boost.
func getPerformanceExpr() string {
	leastExpr := "LEAST(COALESCE(doctorprofile.total_consultations, 0) / 200.0, 1.0)"
	if database.DB != nil && database.DB.Dialector.Name() == "sqlite" {
		leastExpr = "MIN(COALESCE(doctorprofile.total_consultations, 0) / 200.0, 1.0)"
	}
	return fmt.Sprintf(`(
		(COALESCE(doctorprofile.average_rating, 0) / 5.0) * 40.0 +
		%s * 20.0 +
		COALESCE(doctorprofile.completion_rate, 0) * 20.0 +
		CASE WHEN doctorprofile.is_online = true THEN 10.0 ELSE 0.0 END +
		CASE WHEN doctorprofile.is_verified = true THEN 5.0 ELSE 0.0 END +
		COALESCE(doctorprofile.admin_boost_score, 0) * 5.0
	)`, leastExpr)
}

// baseQuery returns a GORM query joining doctorprofile + user.
func baseQuery() *gorm.DB {
	return database.DB.Table("doctorprofile").
		Select(getSafeSelect()).
		Joins(`JOIN "user" ON "user".id = doctorprofile.user_id`).
		Where(`"user".role IN ?`, []string{"doctor", "chief_doctor", "hospital_head_doctor"}).
		Where(`"user".is_active = ?`, true)
}

// GetOnlineDoctors returns all currently online doctors.
// Uses Redis heartbeat for real-time presence; DB is_online is the fallback.
func GetOnlineDoctors() ([]DoctorView, error) {
	var doctors []DoctorView

	err := baseQuery().
		Where("doctorprofile.is_online = ?", true).
		Order(getPerformanceExpr() + " DESC").
		Scan(&doctors).Error

	if err != nil {
		log.Printf("Error fetching online doctors: %v", err)
		return nil, err
	}

	ctx := context.Background()
	onlineList := make([]DoctorView, 0, len(doctors))
	for _, doc := range doctors {
		if presence.IsDoctorOnline(ctx, fmt.Sprintf("%d", doc.UserID)) {
			onlineList = append(onlineList, doc)
		}
	}

	if len(onlineList) == 0 && len(doctors) > 0 {
		log.Printf("    Redis has 0 heartbeats — falling back to DB is_online flag (%d doctors)", len(doctors))
		return doctors, nil
	}

	log.Printf("    Found %d online doctors (real-time)", len(onlineList))
	return onlineList, nil
}

// GetFreeDoctors returns doctors offering free consultations.
func GetFreeDoctors() ([]DoctorView, error) {
	var doctors []DoctorView

	feeExpr := "(doctorprofile.consultation_fees->>'INR')::bigint"
	if database.DB != nil && database.DB.Dialector.Name() == "sqlite" {
		feeExpr = "COALESCE(json_extract(doctorprofile.consultation_fees, '$.INR'), 0)"
	}

	err := baseQuery().
		Where(fmt.Sprintf("%s = 0 OR doctorprofile.offers_free_consultation = ?", feeExpr), true).
		Order(getPerformanceExpr() + " DESC").
		Scan(&doctors).Error

	if err != nil {
		log.Printf("Error fetching free doctors: %v", err)
		return nil, err
	}

	log.Printf("    Found %d free consultation doctors", len(doctors))
	return doctors, nil
}

// GetRankedDoctors returns active doctors sorted by composite performance score.
func GetRankedDoctors(limit int) ([]DoctorView, error) {
	var doctors []DoctorView

	err := database.DB.Table("doctorprofile").
		Select(getSafeSelect()+",\n\t"+getPerformanceExpr()+" AS performance_score").
		Joins(`JOIN "user" ON "user".id = doctorprofile.user_id`).
		Where(`"user".role IN ?`, []string{"doctor", "chief_doctor", "hospital_head_doctor"}).
		Where(`"user".is_active = ?`, true).
		Order(getPerformanceExpr() + " DESC").
		Limit(limit).
		Scan(&doctors).Error

	if err != nil {
		log.Printf("Error fetching ranked doctors: %v", err)
		return nil, err
	}

	log.Printf("    Found %d ranked doctors", len(doctors))
	return doctors, nil
}

// GetDoctorsBySpecialization returns active doctors filtered by specialization.
func GetDoctorsBySpecialization(specialization string) ([]DoctorView, error) {
	var doctors []DoctorView

	err := baseQuery().
		Where("LOWER(doctorprofile.specialization) = LOWER(?)", specialization).
		Order(getPerformanceExpr() + " DESC").
		Scan(&doctors).Error

	if err != nil {
		log.Printf("Error fetching doctors by specialization: %v", err)
		return nil, err
	}

	return doctors, nil
}

// RecalculateAllScores updates performance_score and completion_rate for every active doctor.
// Called nightly by the cron job and can also be triggered by admin.
func RecalculateAllScores() error {
	// Mark every write inside this function as coming from the nightly recalc
	// job so audit_logs rows show actor_role="system:cron:nightly_score_recalc"
	// instead of the meaningless "actor #0 system" placeholder.
	ctx := audit.ForSystem(context.Background(), "cron:nightly_score_recalc")
	db := database.DB.WithContext(ctx)

	// Update completion_rate from appointments table
	err := db.Exec(`
		UPDATE doctorprofile dp
		SET completion_rate = sub.rate
		FROM (
			SELECT
				doctor_id,
				CASE
					WHEN COUNT(*) FILTER (WHERE status IN ('completed','cancelled','no_show','declined')) = 0 THEN 0
					ELSE ROUND(
						COUNT(*) FILTER (WHERE status = 'completed')::numeric /
						NULLIF(COUNT(*) FILTER (WHERE status IN ('completed','cancelled','no_show','declined')), 0), 4
					)
				END AS rate
			FROM appointments
			WHERE doctor_id IS NOT NULL
			GROUP BY doctor_id
		) sub
		WHERE dp.user_id = sub.doctor_id
	`).Error
	if err != nil {
		return fmt.Errorf("completion_rate update failed: %w", err)
	}

	// Update performance_score using the composite formula
	err = db.Exec(`
		UPDATE doctorprofile
		SET performance_score = ` + getPerformanceExpr()).Error
	if err != nil {
		return fmt.Errorf("performance_score update failed: %w", err)
	}

	log.Println("    [Scores] performance_score and completion_rate recalculated for all doctors")
	return nil
}

package rating

import (
	"drayya-core/internal/domain/identity"
	"gorm.io/gorm"
)

type Repository interface {
	CreateRating(rating *Rating) error
	GetByDoctorID(doctorID uint) ([]Rating, error)
	GetByUserID(userID uint) ([]Rating, error)
	HasRatedAppointment(userID, appointmentID uint) (bool, error)
	UpdateDoctorAverageRating(doctorID uint) error
	HideRating(ratingID uint, hiddenBy uint, reason string) error
	RestoreRating(ratingID uint) error
	GetAllForAdmin(doctorID uint, minScore, maxScore int, page, limit int) ([]Rating, int64, error)
	GetByID(ratingID uint) (*Rating, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateRating(rating *Rating) error {
	return r.db.Create(rating).Error
}

func (r *repository) GetByDoctorID(doctorID uint) ([]Rating, error) {
	var ratings []Rating
	err := r.db.Where("doctor_id = ?", doctorID).Order("created_at desc").Find(&ratings).Error
	return ratings, err
}

func (r *repository) GetByUserID(userID uint) ([]Rating, error) {
	var ratings []Rating
	err := r.db.Where("user_id = ?", userID).Order("created_at desc").Find(&ratings).Error
	return ratings, err
}

func (r *repository) HasRatedAppointment(userID, appointmentID uint) (bool, error) {
	var count int64
	err := r.db.Model(&Rating{}).
		Where("user_id = ? AND appointment_id = ?", userID, appointmentID).
		Count(&count).Error
	return count > 0, err
}

// UpdateDoctorAverageRating recalculates average using recency weighting:
// reviews from the last 30 days get 2x weight, 30–90 days get 1.5x, older get 1x.
// Hidden reviews are excluded from the calculation.
func (r *repository) UpdateDoctorAverageRating(doctorID uint) error {
	var result struct {
		WeightedAvg float64
		TotalCount  int64
	}
	var query string
	if r.db.Dialector.Name() == "sqlite" {
		query = `
			SELECT
				ROUND(
					CAST(SUM(score * CASE
						WHEN created_at > datetime('now', '-30 days') THEN 2.0
						WHEN created_at > datetime('now', '-90 days') THEN 1.5
						ELSE 1.0 END
					) AS float) /
					NULLIF(SUM(CASE
						WHEN created_at > datetime('now', '-30 days') THEN 2.0
						WHEN created_at > datetime('now', '-90 days') THEN 1.5
						ELSE 1.0 END
					), 0), 2
				) AS weighted_avg,
				COUNT(*) AS total_count
			FROM ratings
			WHERE doctor_id = ? AND is_hidden = false
		`
	} else {
		query = `
			SELECT
				ROUND(
					CAST(SUM(score * CASE
						WHEN created_at > NOW() - INTERVAL '30 days' THEN 2.0
						WHEN created_at > NOW() - INTERVAL '90 days' THEN 1.5
						ELSE 1.0 END
					) AS numeric) /
					NULLIF(SUM(CASE
						WHEN created_at > NOW() - INTERVAL '30 days' THEN 2.0
						WHEN created_at > NOW() - INTERVAL '90 days' THEN 1.5
						ELSE 1.0 END
					), 0), 2
				) AS weighted_avg,
				COUNT(*) AS total_count
			FROM ratings
			WHERE doctor_id = ? AND is_hidden = false
		`
	}
	r.db.Raw(query, doctorID).Scan(&result)

	return r.db.Model(&identity.DoctorProfile{}).
		Where("user_id = ?", doctorID).
		Updates(map[string]interface{}{
			"average_rating": result.WeightedAvg,
			"total_reviews":  result.TotalCount,
		}).Error
}

func (r *repository) HideRating(ratingID uint, hiddenBy uint, reason string) error {
	return r.db.Model(&Rating{}).Where("id = ?", ratingID).
		Updates(map[string]interface{}{
			"is_hidden":     true,
			"hidden_by":     hiddenBy,
			"hidden_reason": reason,
		}).Error
}

func (r *repository) RestoreRating(ratingID uint) error {
	return r.db.Model(&Rating{}).Where("id = ?", ratingID).
		Updates(map[string]interface{}{
			"is_hidden":     false,
			"hidden_by":     nil,
			"hidden_reason": "",
		}).Error
}

func (r *repository) GetByID(ratingID uint) (*Rating, error) {
	var rating Rating
	err := r.db.First(&rating, ratingID).Error
	return &rating, err
}

func (r *repository) GetAllForAdmin(doctorID uint, minScore, maxScore int, page, limit int) ([]Rating, int64, error) {
	query := r.db.Model(&Rating{})
	if doctorID > 0 {
		query = query.Where("doctor_id = ?", doctorID)
	}
	if minScore > 0 {
		query = query.Where("score >= ?", minScore)
	}
	if maxScore > 0 {
		query = query.Where("score <= ?", maxScore)
	}
	var total int64
	query.Count(&total)

	var ratings []Rating
	offset := (page - 1) * limit
	err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&ratings).Error
	return ratings, total, err
}

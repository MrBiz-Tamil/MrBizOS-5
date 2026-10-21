package handlers

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// GetPromotionAnalytics - Calculate ROI metrics for a doctor's free consultation campaigns
func GetPromotionAnalytics(c *fiber.Ctx) error {
	// In a real app, we get doctorID from the JWT token locals
	var doctorID uint
	if val := c.Locals("user_id"); val != nil {
		doctorID = val.(uint)
	} else {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized: No user_id in context")
	}

	// 1. Total Free Consults Given
	var totalFree int64
	database.DB.Model(&appointment.Appointment{}).
		Where("doctor_id = ? AND consultation_fee = 0 AND status = ?", doctorID, appointment.StatusCompleted).
		Count(&totalFree)

	// 2. Returning Paid Patients
	// Logic: Patients who had at least one completed free consult AND at least one subsequent paid consult with the same doctor
	var returningPaid int64

	// We use a raw-ish query via GORM to handle the self-join efficiently for conversion tracking
	database.DB.Table("appointment as a1").
		Joins("JOIN appointment as a2 ON a1.patient_id = a2.patient_id AND a1.doctor_id = a2.doctor_id").
		Where("a1.doctor_id = ? AND a1.consultation_fee = 0 AND a1.status = ? AND a2.consultation_fee > 0 AND a2.status = ? AND a2.start_time > a1.start_time",
			doctorID, appointment.StatusCompleted, appointment.StatusCompleted).
		Distinct("a1.patient_id").
		Count(&returningPaid)

	conversionRate := 0.0
	if totalFree > 0 {
		conversionRate = (float64(returningPaid) / float64(totalFree)) * 100
	}

	// Mock rating boost until medical review logic is fully integrated
	ratingBoost := 0.0
	if totalFree > 5 {
		ratingBoost = 0.4
	} else if totalFree > 0 {
		ratingBoost = 0.1
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"total_free_given": int(totalFree),
		"returning_paid":   int(returningPaid),
		"conversion_rate":  conversionRate,
		"rating_boost":     ratingBoost,
	}, "Fetched")
}

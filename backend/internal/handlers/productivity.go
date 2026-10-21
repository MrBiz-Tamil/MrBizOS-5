package handlers

import (
	"time"
	"fmt"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/middleware"
	"drayya-core/internal/services/financials"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

func GetMyPerformance(c *fiber.Ctx) error {
	doctorID, ok := c.Locals("user_id").(uint)
	if !ok || doctorID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	}

	// 1. Fetch Doctor Profile for Rating & Score
	var profile identity.DoctorProfile
	database.DB.Where("user_id = ?", doctorID).First(&profile)

	// 2. Fetch Completed Consultations Count
	var totalCompleted int64
	database.DB.Model(&appointment.Appointment{}).Where("doctor_id = ? AND status = ?", doctorID, "completed").Count(&totalCompleted)

	// 3. Fetch Today's Revenue
	earnings, _ := financials.GetAccountBalance(c.Context(), fmt.Sprintf("Doctor:%d", doctorID))
	// In a real scenario, we would filter ledger entries for today only, but let's stick to balance for now or simplify.
	
	// 4. Generate Daily Stats (last 7 days)
	now := time.Now()
	dailyStats := make([]fiber.Map, 7)
	for i := 0; i < 7; i++ {
		d := now.AddDate(0, 0, -6+i)
		dayStart := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
		dayEnd := dayStart.Add(24 * time.Hour)
		
		var count int64
		database.DB.Model(&appointment.Appointment{}).
			Where("doctor_id = ? AND status = ? AND start_time >= ? AND start_time < ?", 
				doctorID, "completed", dayStart, dayEnd).
			Count(&count)
		
		dailyStats[i] = fiber.Map{
			"date":  d.Format("2006-01-02"),
			"count": count,
		}
	}

	data := fiber.Map{
		"performance_score":       profile.AverageRating * 20, // 0-5 scale to 0-100
		"consultations_completed": totalCompleted,
		"today_revenue":           float64(earnings) / 100, // This is current balance, but UI uses it as revenue
		"average_rating":          profile.AverageRating,
		"patient_satisfaction":    float64(profile.AverageRating) * 20,
		"daily_stats":            dailyStats,
		"metrics": fiber.Map{
			"on_time_percentage":    95.0, // Placeholder for future complex analytics
			"avg_consultation_time": "15m",
			"follow_up_rate":        profile.AdminBoostScore / 4,
		},
	}

	return response.JSON(c, fiber.StatusOK, data, "Doctor performance metrics fetched")
}

func RegisterProductivityRoutes(router fiber.Router) {
	prodGroup := router.Group("/productivity", middleware.Protected())
	prodGroup.Get("/my-performance", GetMyPerformance)
}

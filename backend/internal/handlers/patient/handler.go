package patient

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// GetProfile returns the patient profile for the authenticated user.
func GetProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var profile identity.PatientProfile

	err := database.DB.Where("user_id = ?", userID).First(&profile).Error
	if err == gorm.ErrRecordNotFound {
		return response.Error(c, fiber.StatusNotFound, "PROFILE_NOT_FOUND", "Profile not found")
	}
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DB_ERROR", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, profile, "Fetched")
}

// CreateOrUpdateProfile creates or updates the patient profile for the authenticated user.
func CreateOrUpdateProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var req identity.PatientProfile
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	req.UserID = userID

	var existing identity.PatientProfile
	err := database.DB.Where("user_id = ?", userID).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		if err := database.DB.Create(&req).Error; err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "CREATE_PROFILE_FAILED", "Failed to create profile")
		}
		return response.JSON(c, fiber.StatusCreated, req, "Created")
	}
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DB_ERROR", err.Error())
	}

	req.ID = existing.ID
	if err := database.DB.Save(&req).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_PROFILE_FAILED", "Failed to update profile")
	}

	return response.JSON(c, fiber.StatusOK, req, "Updated")
}

// RegisterPatientRoutes wires patient profile routes onto the given router.
func RegisterPatientRoutes(router fiber.Router) {
	patientGroup := router.Group("/patients", middleware.Protected())
	patientGroup.Get("/profile", GetProfile)
	patientGroup.Post("/profile", CreateOrUpdateProfile)
	patientGroup.Put("/profile", CreateOrUpdateProfile)
	patientGroup.Get("/dashboard/summary", GetDashboardSummary)
}

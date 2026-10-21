//go:build ignore

package emr

import (
	"drayya-core/internal/database"
	"drayya-core/internal/models"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// Router /emr/diagnosis [post]
func CreateDiagnosis(c *fiber.Ctx) error {
	var diag models.Diagnosis
	if err := c.BodyParser(&diag); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	if err := database.DB.Create(&diag).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SAVE_DIAGNOSIS_FAILED", "Failed to save diagnosis")
	}

	return response.JSON(c, fiber.StatusCreated, diag, "Created")
}

// Router /emr/diagnosis [get]
func GetDiagnosisHistory(c *fiber.Ctx) error {
	patientID := c.Query("patient_id")
	if patientID == "" {
		return response.Error(c, fiber.StatusBadRequest, "PATIENT_ID_REQUIRED", "patient_id required")
	}

	var history []models.Diagnosis
	database.DB.Where("patient_id = ?", patientID).Order("created_at desc").Find(&history)

	return response.JSON(c, fiber.StatusOK, history, "Fetched")
}

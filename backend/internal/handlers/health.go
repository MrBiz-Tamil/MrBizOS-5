package handlers

import (
	"fmt"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/document"
	"drayya-core/internal/domain/emr"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

// GetLatestVitals returns the most recent vitals for the authenticated patient
func GetLatestVitals(c *fiber.Ctx) error {
	val := c.Locals("user_id")
	if val == nil {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "User ID not found in context")
	}

	userID, ok := val.(uint)
	if !ok {
		return response.Error(c, fiber.StatusInternalServerError, "TYPE_ERROR", "User ID type mismatch")
	}

	var vitals emr.PatientVitals

	result := database.DB.Where("patient_id = ?", userID).Order("recorded_at desc").First(&vitals)
	if result.Error != nil {
		return response.JSON(c, fiber.StatusOK, nil, "No vitals found")
	}

	return response.JSON(c, fiber.StatusOK, vitals, "Latest vitals fetched")
}

// LogVitals saves a new vitals record
func LogVitals(c *fiber.Ctx) error {
	val := c.Locals("user_id")
	if val == nil {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "User ID not found in context")
	}

	userID := val.(uint)
	var record emr.PatientVitals

	if err := c.BodyParser(&record); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	record.PatientID = userID
	if err := database.DB.Create(&record).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to log vitals")
	}

	// Create timeline event
	event := emr.HealthTimelineEvent{
		PatientID:   userID,
		Type:        "vitals",
		Title:       "Vitals Logged",
		Description: fmt.Sprintf("Logged new vitals: BP %.0f/%.0f, HR %.0f", record.SystolicBP, record.DiastolicBP, record.HeartRate),
		ReferenceID: record.ID,
		Date:        time.Now(),
	}
	database.DB.Create(&event)

	return response.JSON(c, fiber.StatusCreated, record, "Vitals logged successfully")
}

// GetHealthTimeline returns the patient's health event timeline
func GetHealthTimeline(c *fiber.Ctx) error {
	val := c.Locals("user_id")
	if val == nil {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "User ID not found in context")
	}

	userID := val.(uint)
	limitStr := c.Query("limit", "20")
	limit, _ := strconv.Atoi(limitStr)

	var events []emr.HealthTimelineEvent
	database.DB.Where("patient_id = ?", userID).Order("date desc").Limit(limit).Find(&events)

	if events == nil {
		events = []emr.HealthTimelineEvent{}
	}

	return response.JSON(c, fiber.StatusOK, events, "Health timeline fetched")
}

// GetVitalsHistory returns history of vitals
func GetVitalsHistory(c *fiber.Ctx) error {
	val := c.Locals("user_id")
	if val == nil {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "User ID not found in context")
	}

	userID := val.(uint)
	limitStr := c.Query("limit", "10")
	limit, _ := strconv.Atoi(limitStr)

	var history []emr.PatientVitals
	database.DB.Where("patient_id = ?", userID).Order("recorded_at desc").Limit(limit).Find(&history)

	if history == nil {
		history = []emr.PatientVitals{}
	}

	return response.JSON(c, fiber.StatusOK, history, "Vitals history fetched")
}

// UploadHealthDocument handles uploading of health documents
func UploadHealthDocument(c *fiber.Ctx) error {
	val := c.Locals("user_id")
	if val == nil {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "User ID not found in context")
	}
	userID := val.(uint)

	file, err := c.FormFile("document")
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "FILE_MISSING", "Document file is required")
	}

	// Store the file for real in the private document vault (owner = this patient)
	// instead of the old mock that discarded the bytes and returned a fake URL.
	category := c.FormValue("category")
	if category == "" {
		category = "other"
	}
	doc, err := document.SavePatientFile(c.UserContext(), file, category, userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPLOAD_FAILED", err.Error())
	}

	// Record a timeline event referencing the stored document.
	event := emr.HealthTimelineEvent{
		PatientID:   userID,
		Type:        "document",
		Title:       "Document Uploaded",
		Description: fmt.Sprintf("Uploaded health document: %s", doc.OriginalName),
		Date:        time.Now(),
		ReferenceID: doc.ID,
	}
	database.DB.Create(&event)

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"document_id": doc.ID,
		"filename":    doc.OriginalName,
		"event_id":    event.ID,
	}, "Health document uploaded successfully")
}

func RegisterHealthRoutes(router fiber.Router) {
	healthGroup := router.Group("/health")
	healthGroup.Get("/vitals/latest", middleware.Protected(), GetLatestVitals)
	healthGroup.Get("/vitals/history", middleware.Protected(), GetVitalsHistory)
	healthGroup.Post("/vitals", middleware.Protected(), LogVitals)
	healthGroup.Get("/timeline", middleware.Protected(), GetHealthTimeline)
	healthGroup.Post("/documents/upload", middleware.Protected(), UploadHealthDocument)
}

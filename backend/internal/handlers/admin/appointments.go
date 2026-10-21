package admin

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// GetAllAppointments - Get all appointments (admin)
func GetAllAppointments(c *fiber.Ctx) error {
	var appointments []appointment.Appointment

	// Get query parameters for filtering
	status := c.Query("status")
	limit := c.QueryInt("limit", 100)

	query := database.DB.Preload("Patient").Preload("Doctor")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Order("start_time DESC").
		Limit(limit).
		Find(&appointments)

	return c.JSON(appointments)
}

// GetAppointmentDetails - Get appointment details (admin)
func GetAppointmentDetails(c *fiber.Ctx) error {
	id := c.Params("id")

	var appointment appointment.Appointment
	result := database.DB.Preload("Patient").Preload("Doctor").First(&appointment, id)

	if result.Error != nil {
		return response.Error(c, fiber.StatusNotFound, "APPOINTMENT_NOT_FOUND", "Appointment not found")
	}

	return c.JSON(appointment)
}

// UpdateAppointment - Update appointment (admin)
func UpdateAppointment(c *fiber.Ctx) error {
	id := c.Params("id")

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	result := database.DB.Model(&appointment.Appointment{}).Where("id = ?", id).Updates(updates)

	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", result.Error.Error())
	}

	if result.RowsAffected == 0 {
		return response.Error(c, fiber.StatusNotFound, "APPOINTMENT_NOT_FOUND", "Appointment not found")
	}

	return c.JSON(fiber.Map{"message": "Appointment updated successfully"})
}

// CancelAppointment - Cancel appointment (admin)
func CancelAppointment(c *fiber.Ctx) error {
	id := c.Params("id")

	result := database.DB.Model(&appointment.Appointment{}).
		Where("id = ?", id).
		Update("status", "cancelled")

	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CANCEL_FAILED", result.Error.Error())
	}

	if result.RowsAffected == 0 {
		return response.Error(c, fiber.StatusNotFound, "APPOINTMENT_NOT_FOUND", "Appointment not found")
	}

	return c.JSON(fiber.Map{"message": "Appointment cancelled successfully"})
}

// ReassignAppointments - Bulk reassign appointments from one doctor to another on a specific date
func ReassignAppointments(c *fiber.Ctx) error {
	var req struct {
		FromDoctorID uint   `json:"from_doctor_id"`
		ToDoctorID   uint   `json:"to_doctor_id"`
		Date         string `json:"date"` // Format "YYYY-MM-DD"
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	// Simple date prefix matching logic e.g., "2024-05-10%"
	datePrefix := req.Date + "%"

	result := database.DB.Model(&appointment.Appointment{}).
		Where("doctor_id = ? AND start_time LIKE ?", req.FromDoctorID, datePrefix).
		Update("doctor_id", req.ToDoctorID)

	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "REASSIGN_FAILED", result.Error.Error())
	}

	return c.JSON(fiber.Map{
		"message":       "Appointments reassigned successfully",
		"rows_affected": result.RowsAffected,
	})
}

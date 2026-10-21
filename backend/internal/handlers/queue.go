package handlers

import (
	"log"
	"drayya-core/internal/queue"

	"github.com/gofiber/fiber/v2"
)

func EnqueueAppointment(c *fiber.Ctx) error {
	var meta queue.AppointmentMetadata
	if err := c.BodyParser(&meta); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Validate (Basic)
	if meta.AppointmentID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "AppointmentID is required"})
	}

	// Enqueue
	if err := queue.Enqueue(c.Context(), meta); err != nil {
		log.Printf("Enqueue error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to enqueue appointment"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Appointment enqueued successfully",
		"score":   meta.InitialScore,
	})
}

func GetQueueStats(c *fiber.Ctx) error {
	// Replicate Python's get_queue_stats logic if needed
	// For now, simple count
	// This requires implemention in queue/manager.go
	return c.JSON(fiber.Map{"message": "Stats not implemented yet"})
}

package handlers

import (
	"drayya-core/internal/services/visibility"

	"github.com/gofiber/fiber/v2"
)

// GetOnlineDoctors returns all online doctors
func GetOnlineDoctors(c *fiber.Ctx) error {
	doctors, err := visibility.GetOnlineDoctors()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":  "Failed to fetch online doctors",
			"detail": err.Error(),
		})
	}

	if doctors == nil {
		doctors = []visibility.DoctorView{}
	}

	return c.JSON(doctors)
}

// GetFreeDoctors returns doctors offering free consultations
func GetFreeDoctors(c *fiber.Ctx) error {
	doctors, err := visibility.GetFreeDoctors()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":  "Failed to fetch free doctors",
			"detail": err.Error(),
		})
	}

	if doctors == nil {
		doctors = []visibility.DoctorView{}
	}

	return c.JSON(doctors)
}

// GetRankedDoctors returns doctors sorted by performance score
func GetRankedDoctors(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 20)

	doctors, err := visibility.GetRankedDoctors(limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":  "Failed to fetch ranked doctors",
			"detail": err.Error(),
		})
	}

	if doctors == nil {
		doctors = []visibility.DoctorView{}
	}

	return c.JSON(doctors)
}

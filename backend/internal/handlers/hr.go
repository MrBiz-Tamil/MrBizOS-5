//go:build ignore
// +build ignore

package handlers

import (
	"drayya-core/internal/hr"
	"drayya-core/internal/utils/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func ClockIn(c *fiber.Ctx) error {
	var req struct {
		UserID  uint `json:"user_id"`
		ShiftID uint `json:"shift_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	err := hr.ClockIn(req.UserID, req.ShiftID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CLOCK_IN_FAILED", err.Error())
	}

	return c.SendStatus(200)
}

func ClockOut(c *fiber.Ctx) error {
	attendanceID, _ := strconv.Atoi(c.Params("attendanceId"))

	var req struct {
		UserID uint `json:"user_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	err := hr.ClockOut(req.UserID, uint(attendanceID))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CLOCK_OUT_FAILED", err.Error())
	}

	return c.SendStatus(200)
}

func CalculatePayroll(c *fiber.Ctx) error {
	userID, _ := strconv.Atoi(c.Params("userId"))
	month, _ := strconv.Atoi(c.Query("month"))
	year, _ := strconv.Atoi(c.Query("year"))

	result, err := hr.CalculateMonthlyPayroll(uint(userID), month, year)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PAYROLL_CALCULATION_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, result, "Fetched")
}

func GetHRRoster(c *fiber.Ctx) error {
	hospitalID, _ := strconv.Atoi(c.Query("hospitalId"))

	roster, err := hr.GetStaffRoster(uint(hospitalID))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ROSTER_FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, roster, "Fetched")
}

func RegisterHRRoutes(router fiber.Router) {
	hrGroup := router.Group("/hr")

	hrGroup.Post("/attendance/clock-in", ClockIn)
	hrGroup.Post("/attendance/clock-out/:attendanceId", ClockOut)
	hrGroup.Get("/payroll/:userId", CalculatePayroll)
	hrGroup.Get("/roster", GetHRRoster)
}

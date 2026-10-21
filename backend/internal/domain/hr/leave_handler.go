package hr

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"drayya-core/internal/middleware"
)

// RegisterLeaveRoutes wires all leave-management endpoints under the HR group.
func RegisterLeaveRoutes(router fiber.Router) {
	leave := router.Group("/hr/leave", middleware.Protected())

	// Leave types (admin manages; staff reads)
	leave.Get("/types", listLeaveTypes)
	leave.Post("/types", middleware.HRManagerOnly(), createLeaveType)

	// Self-service: employee
	leave.Post("/apply", applyLeave)
	leave.Get("/my", getMyLeaveRequests)
	leave.Get("/balance", getMyLeaveBalance)
	leave.Post("/:id/cancel", cancelLeaveRequest)

	// Manager / HR approval
	leave.Get("/pending", middleware.HRManagerOnly(), getPendingLeaveRequests)
	leave.Post("/:id/approve", middleware.HRManagerOnly(), approveLeaveRequest)
	leave.Post("/:id/reject", middleware.HRManagerOnly(), rejectLeaveRequest)

	// Holiday calendar
	leave.Get("/holidays", getHolidays)
	leave.Post("/holidays", middleware.HRManagerOnly(), createHoliday)
}

// ── Leave Type handlers ───────────────────────────────────────────────────────

func listLeaveTypes(c *fiber.Ctx) error {
	types, err := GetAllLeaveTypes()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(types)
}

func createLeaveType(c *fiber.Ctx) error {
	var lt LeaveType
	if err := c.BodyParser(&lt); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if err := CreateLeaveType(&lt); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(lt)
}

// ── Employee self-service ─────────────────────────────────────────────────────

type ApplyLeaveRequest struct {
	LeaveTypeID uint   `json:"leave_type_id"`
	StartDate   string `json:"start_date"` // "2026-03-10"
	EndDate     string `json:"end_date"`
	HalfDay     bool   `json:"half_day"`
	Reason      string `json:"reason"`
}

func applyLeave(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var req ApplyLeaveRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	const layout = "2006-01-02"
	start, err := time.Parse(layout, req.StartDate)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid start_date format (YYYY-MM-DD)"})
	}
	end, err := time.Parse(layout, req.EndDate)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid end_date format (YYYY-MM-DD)"})
	}

	result, err := ApplyLeave(userID, req.LeaveTypeID, start, end, req.HalfDay, req.Reason)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(result)
}

func getMyLeaveRequests(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	reqs, err := GetLeaveRequestsByUser(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(reqs)
}

func getMyLeaveBalance(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	year := time.Now().Year()
	if y := c.QueryInt("year", 0); y > 0 {
		year = y
	}
	balances, err := GetLeaveBalances(userID, year)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(balances)
}

func cancelLeaveRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	id, _ := strconv.Atoi(c.Params("id"))
	if err := CancelLeave(uint(id), userID); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "leave request cancelled"})
}

// ── Manager approval ──────────────────────────────────────────────────────────

func getPendingLeaveRequests(c *fiber.Ctx) error {
	reqs, err := GetPendingLeaveRequests()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(reqs)
}

func approveLeaveRequest(c *fiber.Ctx) error {
	approverID := c.Locals("user_id").(uint)
	id, _ := strconv.Atoi(c.Params("id"))
	result, err := ApproveLeave(uint(id), approverID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

func rejectLeaveRequest(c *fiber.Ctx) error {
	rejectorID := c.Locals("user_id").(uint)
	id, _ := strconv.Atoi(c.Params("id"))
	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.BodyParser(&body)
	result, err := RejectLeave(uint(id), rejectorID, body.Reason)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// ── Holiday Calendar ──────────────────────────────────────────────────────────

func getHolidays(c *fiber.Ctx) error {
	year := time.Now().Year()
	if y := c.QueryInt("year", 0); y > 0 {
		year = y
	}
	holidays, err := GetHolidaysByYear(year)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(holidays)
}

func createHoliday(c *fiber.Ctx) error {
	var h HolidayCalendar
	if err := c.BodyParser(&h); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	h.Year = h.Date.Year()
	if err := CreateHoliday(&h); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(h)
}

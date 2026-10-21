package hr

import (
	"drayya-core/internal/database"
	"drayya-core/internal/utils/response"
	"time"

	"github.com/gofiber/fiber/v2"
)

// validAttendanceStatuses are the states HR can record for a day.
var validAttendanceStatuses = map[string]bool{
	"present": true, "absent": true, "half_day": true, "leave": true, "late": true,
}

// MarkAttendance upserts a single staff member's attendance for a day. This is the
// input that drives payroll loss-of-pay: a day marked "absent" reduces paid days.
//
// POST /admin/hr/attendance/mark
// body: { user_id: number, date: "YYYY-MM-DD", status: string }
//
// @Summary Mark Attendance
// @Description Record or update a staff member's attendance status for a date.
// @Tags HR
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} response.StandardResponse "Attendance marked"
// @Router /api/v1/admin/hr/attendance/mark [post]
func MarkAttendance(c *fiber.Ctx) error {
	var req struct {
		UserID uint   `json:"user_id"`
		Date   string `json:"date"`
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil || req.UserID == 0 || req.Date == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "user_id and date are required")
	}
	if !validAttendanceStatuses[req.Status] {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_STATUS", "status must be one of present, absent, half_day, leave, late")
	}
	day, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_DATE", "date must be YYYY-MM-DD")
	}

	var existing StaffAttendance
	err = database.DB.Where("user_id = ? AND date = ?", req.UserID, day.Format("2006-01-02")).First(&existing).Error
	if err != nil {
		rec := StaffAttendance{UserID: req.UserID, Date: day, Status: req.Status, IsVerified: true}
		if cerr := database.DB.Create(&rec).Error; cerr != nil {
			return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", cerr.Error())
		}
		return response.JSON(c, fiber.StatusOK, rec, "Attendance marked")
	}
	existing.Status = req.Status
	existing.IsVerified = true
	if uerr := database.DB.Save(&existing).Error; uerr != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", uerr.Error())
	}
	return response.JSON(c, fiber.StatusOK, existing, "Attendance updated")
}

// GetAttendanceSheet returns a staff member's attendance records for a month, plus
// a summary (present/absent/leave counts) that mirrors what payroll will use.
//
// GET /admin/hr/attendance?user_id=&month=&year=
//
// @Summary Get Attendance Sheet
// @Description Monthly attendance records + summary for one staff member.
// @Tags HR
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} response.StandardResponse "Attendance sheet"
// @Router /api/v1/admin/hr/attendance [get]
func GetAttendanceSheet(c *fiber.Ctx) error {
	userID := c.QueryInt("user_id")
	month := c.QueryInt("month")
	year := c.QueryInt("year")
	if userID <= 0 || month < 1 || month > 12 || year < 2000 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "user_id, month (1-12) and year are required")
	}

	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	var records []StaffAttendance
	database.DB.Where("user_id = ? AND date >= ? AND date < ?", userID, start.Format("2006-01-02"), end.Format("2006-01-02")).
		Order("date ASC").Find(&records)

	summary := map[string]int{"present": 0, "absent": 0, "half_day": 0, "leave": 0, "late": 0}
	for _, r := range records {
		summary[r.Status]++
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"user_id": userID,
		"month":   month,
		"year":    year,
		"records": records,
		"summary": summary,
	}, "Attendance sheet fetched")
}

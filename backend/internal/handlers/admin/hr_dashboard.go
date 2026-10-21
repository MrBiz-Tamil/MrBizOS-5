package admin

import (
	"fmt"
	"time"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

type HRDashboardOverview struct {
	TotalStaff        int64            `json:"total_staff"`
	StaffByDepartment map[string]int64 `json:"staff_by_department"`
	PendingLeaves     int64            `json:"pending_leaves"`
	ActiveRecruitment int64            `json:"active_recruitment"`
	AttendanceToday   float64          `json:"attendance_today"`
}

func GetHRDashboardOverview(c *fiber.Ctx) error {
	db := database.DB
	var overview HRDashboardOverview

	// 1. Total Staff (Non-patient users)
	db.Model(&identity.User{}).Where("role != ?", "patient").Count(&overview.TotalStaff)

	// 2. Staff by Role
	rows, err := db.Model(&identity.User{}).
		Where("role != ?", "patient").
		Select("role, count(*)").
		Group("role").Rows()

	if err == nil {
		overview.StaffByDepartment = make(map[string]int64)
		for rows.Next() {
			var role string
			var count int64
			rows.Scan(&role, &count)
			overview.StaffByDepartment[role] = count
		}
		rows.Close()
	}

	// 3. Real Active Recruitment Count (Job openings that are active)
	db.Table("job_openings").Where("is_active = ?", true).Count(&overview.ActiveRecruitment)

	// 4. Real Pending Leaves Count
	db.Table("leave_requests").Where("status = ?", "pending").Count(&overview.PendingLeaves)

	// 5. Attendance Calculation (Based on active users with 'doctor' or 'nurse' or 'bpo' roles)
	var activeStaff int64
	db.Model(&identity.User{}).
		Where("role != ? AND last_login >= ?", "patient", time.Now().Add(-24*time.Hour)).
		Count(&activeStaff)

	if overview.TotalStaff > 0 {
		overview.AttendanceToday = (float64(activeStaff) / float64(overview.TotalStaff)) * 100
		if overview.AttendanceToday > 100 {
			overview.AttendanceToday = 100
		}
	} else {
		overview.AttendanceToday = 0
	}

	return c.JSON(overview)
}

func UpdateStaffStatus(c *fiber.Ctx) error {
	db := database.DB
	staffID, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_STAFF_ID", "Invalid staff ID")
	}

	var body struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	err = db.Model(&identity.User{}).Where("id = ?", staffID).Update("is_active", body.IsActive).Error
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}

	status := "enabled"
	if !body.IsActive {
		status = "disabled"
	}

	return c.JSON(fiber.Map{"message": "Staff account " + status})
}

func GetStaffDirectory(c *fiber.Ctx) error {
	var users []identity.User
	database.DB.Where("role != ?", "patient").Find(&users)

	// Remove password hashes
	for i := range users {
		users[i].PasswordHash = ""
	}

	return c.JSON(users)
}

func ExportPayroll(c *fiber.Ctx) error {
	db := database.DB
	var staff []identity.User
	db.Where("role != ?", "patient").Find(&staff)

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=payroll_export.csv")

	csvContent := "Employee ID,Full Name,Email,Role,Base Salary (Estimated)\n"
	for _, s := range staff {
		// Base salary estimation logic based on role
		salary := 25000
		if s.Role == "doctor" {
			salary = 75000
		} else if s.Role == "admin" || s.Role == "super_admin" {
			salary = 45000
		}
		csvContent += fmt.Sprintf("%d,%s,%s,%s,%d\n", s.ID, s.FullName, s.Email, s.Role, salary)
	}

	return c.SendString(csvContent)
}

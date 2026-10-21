package hr

import (
	"time"

	"drayya-core/internal/database"
)

// ── Leave Types ──────────────────────────────────────────────────────────────

func CreateLeaveType(lt *LeaveType) error {
	return database.DB.Create(lt).Error
}

func GetAllLeaveTypes() ([]LeaveType, error) {
	var types []LeaveType
	return types, database.DB.Find(&types).Error
}

func GetLeaveTypeByID(id uint) (*LeaveType, error) {
	var lt LeaveType
	return &lt, database.DB.First(&lt, id).Error
}

// ── Leave Balances ────────────────────────────────────────────────────────────

func GetLeaveBalances(userID uint, year int) ([]LeaveBalance, error) {
	var balances []LeaveBalance
	err := database.DB.Where("user_id = ? AND year = ?", userID, year).
		Preload("LeaveType").Find(&balances).Error
	return balances, err
}

func GetLeaveBalance(userID, leaveTypeID uint, year int) (*LeaveBalance, error) {
	var b LeaveBalance
	err := database.DB.Where("user_id = ? AND leave_type_id = ? AND year = ?", userID, leaveTypeID, year).
		Preload("LeaveType").First(&b).Error
	return &b, err
}

func UpsertLeaveBalance(b *LeaveBalance) error {
	var existing LeaveBalance
	err := database.DB.Where("user_id = ? AND leave_type_id = ? AND year = ?",
		b.UserID, b.LeaveTypeID, b.Year).First(&existing).Error
	if err != nil {
		return database.DB.Create(b).Error
	}
	b.ID = existing.ID
	return database.DB.Save(b).Error
}

func SaveLeaveBalance(b *LeaveBalance) error {
	return database.DB.Save(b).Error
}

// ── Leave Requests ────────────────────────────────────────────────────────────

func CreateLeaveRequest(req *LeaveRequest) error {
	return database.DB.Create(req).Error
}

func GetLeaveRequestByID(id uint) (*LeaveRequest, error) {
	var r LeaveRequest
	err := database.DB.Preload("LeaveType").First(&r, id).Error
	return &r, err
}

func UpdateLeaveRequest(req *LeaveRequest) error {
	return database.DB.Save(req).Error
}

func GetLeaveRequestsByUser(userID uint) ([]LeaveRequest, error) {
	var reqs []LeaveRequest
	err := database.DB.Where("user_id = ?", userID).
		Preload("LeaveType").Order("created_at DESC").Find(&reqs).Error
	return reqs, err
}

func GetPendingLeaveRequests() ([]LeaveRequest, error) {
	var reqs []LeaveRequest
	err := database.DB.Where("status = 'pending'").
		Preload("LeaveType").Order("created_at ASC").Find(&reqs).Error
	return reqs, err
}

func GetLeaveRequestsBetween(userID uint, start, end time.Time) ([]LeaveRequest, error) {
	var reqs []LeaveRequest
	err := database.DB.Where("user_id = ? AND status = 'approved' AND start_date >= ? AND end_date <= ?",
		userID, start, end).Find(&reqs).Error
	return reqs, err
}

// ── Holiday Calendar ──────────────────────────────────────────────────────────

func CreateHoliday(h *HolidayCalendar) error {
	return database.DB.Create(h).Error
}

func GetHolidaysByYear(year int) ([]HolidayCalendar, error) {
	var holidays []HolidayCalendar
	err := database.DB.Where("year = ?", year).Order("date ASC").Find(&holidays).Error
	return holidays, err
}

func IsHoliday(date time.Time) bool {
	var count int64
	database.DB.Model(&HolidayCalendar{}).
		Where("date = ?", date.Truncate(24*time.Hour)).Count(&count)
	return count > 0
}

// CountWorkingDays calculates business days between two dates excluding weekends and holidays.
func CountWorkingDays(start, end time.Time) float64 {
	var days float64
	current := start
	for !current.After(end) {
		if current.Weekday() != time.Saturday && current.Weekday() != time.Sunday && !IsHoliday(current) {
			days++
		}
		current = current.AddDate(0, 0, 1)
	}
	return days
}

//go:build ignore
// +build ignore

package hr

import (
	"fmt"
	"drayya-core/internal/database"
	"drayya-core/internal/models"
	"time"
)

func ClockIn(userID uint, shiftID uint) error {
	now := time.Now()
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	attendance := models.StaffAttendance{
		UserID:  userID,
		ShiftID: shiftID,
		Date:    date,
		CheckIn: &now,
		Status:  "present",
	}

	// Check if shift timing is late (simplified)
	var shift models.StaffShift
	if err := database.DB.First(&shift, shiftID).Error; err == nil {
		// Logically check if now > shift.StartTime + 15mins
	}

	return database.DB.Create(&attendance).Error
}

func ClockOut(userID uint, attendanceID uint) error {
	now := time.Now()
	return database.DB.Model(&models.StaffAttendance{}).
		Where("id = ? AND user_id = ?", attendanceID, userID).
		Update("check_out", &now).Error
}

func CalculateMonthlyPayroll(userID uint, month int, year int) (*models.PayrollRecord, error) {
	var profile models.StaffProfile
	if err := database.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return nil, fmt.Errorf("staff profile not found")
	}

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	endDate := startDate.AddDate(0, 1, 0)

	var attendanceRecords []models.StaffAttendance
	err := database.DB.Where("user_id = ? AND date >= ? AND date < ?", userID, startDate, endDate).
		Find(&attendanceRecords).Error
	if err != nil {
		return nil, err
	}

	// Simple logic: Base - (DailyRate * days_missed)
	daysInMonth := time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.Local).Day()
	daysPresent := len(attendanceRecords)
	dailyRate := profile.BaseSalary / int64(daysInMonth)

	deductions := int64(daysInMonth-daysPresent) * dailyRate
	netAmount := profile.BaseSalary - deductions

	payroll := models.PayrollRecord{
		UserID:     userID,
		Month:      month,
		Year:       year,
		BaseAmount: profile.BaseSalary,
		Deductions: deductions,
		NetAmount:  netAmount,
		Status:     "processed",
	}

	if err := database.DB.Create(&payroll).Error; err != nil {
		return nil, err
	}

	return &payroll, nil
}

func GetStaffRoster(hospitalID uint) ([]models.StaffAttendance, error) {
	var roster []models.StaffAttendance
	err := database.DB.Preload("User").Order("date desc").Limit(100).Find(&roster).Error
	return roster, err
}

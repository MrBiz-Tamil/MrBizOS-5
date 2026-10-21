package main

import (
	"log"
	"drayya-core/internal/domain/hr"
	"drayya-core/internal/domain/identity"
	"time"

	"gorm.io/gorm"
)

func seedHR(db *gorm.DB) {
	// Get staff users (nurses, lab tech, pharmacist)
	var staffUsers []identity.User
	db.Where("role IN ?", []string{"nurse", "lab_technician", "pharmacist"}).Find(&staffUsers)

	for i, u := range staffUsers {
		var existing hr.StaffProfile
		if db.Where("user_id = ?", u.ID).First(&existing).Error == nil {
			continue
		}
		dept := "Nursing"
		desig := "Staff Nurse"
		salary := int64(3500000) // 35,000 INR
		if u.Role == "lab_technician" {
			dept = "Laboratory"
			desig = "Lab Technician"
			salary = 3000000
		} else if u.Role == "pharmacist" {
			dept = "Pharmacy"
			desig = "Pharmacist"
			salary = 4000000
		}
		db.Create(&hr.StaffProfile{
			UserID:           u.ID,
			EmployeeID:       "EMP-2026-" + string(rune('A'+i)) + "001",
			Department:       dept,
			Designation:      desig,
			JoiningDate:      time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			BaseSalary:       salary,
			EmploymentStatus: "active",
		})
	}

	// Shifts
	shifts := []hr.StaffShift{
		{HospitalID: 1, Name: "Morning Shift", StartTime: "06:00", EndTime: "14:00", IsActive: true},
		{HospitalID: 1, Name: "Afternoon Shift", StartTime: "14:00", EndTime: "22:00", IsActive: true},
		{HospitalID: 1, Name: "Night Shift", StartTime: "22:00", EndTime: "06:00", IsActive: true},
	}
	for i := range shifts {
		db.FirstOrCreate(&shifts[i], "hospital_id = ? AND name = ?", shifts[i].HospitalID, shifts[i].Name)
	}

	// Attendance for today and yesterday
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	yesterday := today.Add(-24 * time.Hour)

	for _, u := range staffUsers {
		for _, date := range []time.Time{yesterday, today} {
			var existing hr.StaffAttendance
			if db.Where("user_id = ? AND date = ?", u.ID, date).First(&existing).Error == nil {
				continue
			}
			checkIn := date.Add(6 * time.Hour)
			checkOut := date.Add(14 * time.Hour)
			att := hr.StaffAttendance{
				UserID:     u.ID,
				ShiftID:    shifts[0].ID,
				Date:       date,
				CheckIn:    &checkIn,
				Status:     "present",
				IsVerified: true,
			}
			if date.Equal(yesterday) {
				att.CheckOut = &checkOut
			}
			db.Create(&att)
		}
	}

	// Payroll for last month
	lastMonth := now.AddDate(0, -1, 0)
	for _, u := range staffUsers {
		var profile hr.StaffProfile
		db.Where("user_id = ?", u.ID).First(&profile)
		if profile.ID == 0 {
			continue
		}
		var existing hr.PayrollRecord
		if db.Where("user_id = ? AND month = ? AND year = ?", u.ID, int(lastMonth.Month()), lastMonth.Year()).First(&existing).Error == nil {
			continue
		}
		paidAt := time.Date(lastMonth.Year(), lastMonth.Month(), 28, 0, 0, 0, 0, time.UTC)
		db.Create(&hr.PayrollRecord{
			UserID:      u.ID,
			Month:       int(lastMonth.Month()),
			Year:        lastMonth.Year(),
			BaseAmount:  profile.BaseSalary,
			Bonus:       0,
			Deductions:  profile.BaseSalary / 10, // 10% deduction
			NetAmount:   profile.BaseSalary - profile.BaseSalary/10,
			Status:      "paid",
			PaymentDate: &paidAt,
		})
	}

	log.Printf("       Created HR profiles, shifts, attendance, payroll for %d staff", len(staffUsers))
}


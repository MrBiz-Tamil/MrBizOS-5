package hr

import (
	"fmt"
	"time"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ClockIn(userID uint, shiftID uint) error {
	now := time.Now()
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	attendance := &StaffAttendance{
		UserID:  userID,
		ShiftID: shiftID,
		Date:    date,
		CheckIn: &now,
		Status:  "present",
	}

	// In a real system, we'd fetch the shift and check if it's late.
	// For now, mirroring the simplified logic in the monolith.

	return s.repo.RecordAttendance(attendance)
}

func (s *Service) ClockOut(userID uint, attendanceID uint) error {
	now := time.Now()

	// Note: The specific attendanceID is usually verified by the repo/DB query.
	// This mirrors the monolith logic of updating based on ID and UserID.
	// We can add a helper to the repo if needed, but for now using generic update.

	// For a more DDD approach, we'd fetch the entity, modify it, then save.
	// But to keep it simple and avoid massive changes for now:
	attendance := &StaffAttendance{
		ID:       attendanceID,
		UserID:   userID,
		CheckOut: &now,
	}

	return s.repo.UpdateAttendance(attendance)
}

// CalculateMonthlyPayroll runs a salaried employee's monthly payroll through the
// statutory compliance engine (ComputePayrollCompliance): attendance determines
// loss-of-pay, the LOP-adjusted gross drives PF / ESI / Professional Tax, a
// Payslip is generated, and a PayrollRecord is written for the payroll list.
//
// This is now the single payroll engine — the earlier naive "base − absent days"
// formula (which produced no statutory deductions and could not reconcile against
// the payslip) has been retired. Freelancers are rejected outright.
func (s *Service) CalculateMonthlyPayroll(userID uint, month int, year int) (*PayrollRecord, error) {
	profile, err := s.repo.GetStaffProfileByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("staff profile not found")
	}

	// Freelance panel doctors are paid via consultation revenue-share, not payroll.
	// Running salaried payroll on them would invent a fictitious salary and statutory
	// deductions they are not subject to. Their earnings live in the payout ledger.
	if profile.IsFreelancer() {
		return nil, fmt.Errorf("user %d is a freelancer (revenue-share); payroll does not apply — use the payout/earnings ledger", userID)
	}

	// Terminated staff are not paid for months beyond their last working day; final
	// settlement is handled separately at offboarding.
	if profile.IsTerminated() {
		monthEnd := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local).AddDate(0, 1, -1)
		if profile.LastWorkingDay == nil || profile.LastWorkingDay.Before(time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)) {
			_ = monthEnd
			return nil, fmt.Errorf("user %d is terminated; no payroll for this period", userID)
		}
	}

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	endDate := startDate.AddDate(0, 1, 0)

	attendanceRecords, err := s.repo.GetAttendanceInRange(userID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Loss-of-pay: a salaried employee is paid the full month unless there are
	// explicitly recorded absent days. (Treating "no clock-in row" as absence would
	// zero out anyone whose attendance simply isn't tracked.)
	daysInMonth := time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.Local).Day()
	absentDays := 0
	for _, a := range attendanceRecords {
		if a.Status == "absent" {
			absentDays++
		}
	}
	if absentDays > daysInMonth {
		absentDays = daysInMonth
	}
	paidDays := daysInMonth - absentDays
	lopDays := float64(absentDays)
	paidGross := profile.BaseSalary * int64(paidDays) / int64(daysInMonth)

	// Statutory compute + persist (upsert by user+month).
	monthStr := fmt.Sprintf("%04d-%02d", year, month)
	pc := ComputePayrollCompliance(userID, monthStr, paidGross, 0.5, 0)
	if err := SavePayrollCompliance(pc); err != nil {
		return nil, err
	}

	// Generate the payslip once per month (skip if it already exists so re-runs
	// don't duplicate).
	if _, perr := GetPayslip(userID, monthStr); perr != nil {
		_, fullName, _ := s.repo.GetUserEmailByID(userID)
		_, _ = GeneratePayslip(pc, fullName, profile.Designation, "", "", "", paidDays, lopDays)
	}

	totalDeductions := pc.PFEmployee + pc.ESIEmployee + pc.ProfTax + pc.IncomeTaxTDS

	// Make re-runs idempotent: clear any prior record for this user+month.
	_ = s.repo.DeletePayrollRecords(userID, year, month)
	payroll := &PayrollRecord{
		UserID:     userID,
		Month:      month,
		Year:       year,
		BaseAmount: paidGross,
		Deductions: totalDeductions,
		NetAmount:  pc.NetSalary,
		Status:     "processed",
	}
	if err := s.repo.CreatePayrollRecord(payroll); err != nil {
		return nil, err
	}

	return payroll, nil
}

func (s *Service) GetStaffRoster(hospitalID uint) ([]StaffAttendance, error) {
	// For roster, we return recent attendance.
	return s.repo.GetAttendanceInRange(0, time.Now().AddDate(0, 0, -7), time.Now())
}

func (s *Service) GetStaffProfile(id uint) (*StaffProfile, error) {
	return s.repo.GetStaffProfile(id)
}

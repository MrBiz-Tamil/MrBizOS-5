package hr

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"fmt"
	"time"
)

// FinalSettlement is the full-and-final packet computed when a staff member is
// offboarded. The two employment tracks settle differently:
//
//   - Salaried: pro-rated salary up to the last working day + encashment of unused
//     paid leave, less any deductions.
//   - Freelancer: the accrued-but-unpaid consultation net still owed, with the TDS
//     that was withheld on it shown separately.
//
// All amounts are paise.
type FinalSettlement struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UserID         uint      `gorm:"index;not null" json:"user_id"`
	EmploymentType string    `json:"employment_type"`
	LastWorkingDay time.Time `json:"last_working_day"`

	// Salaried components.
	ProratedSalaryPaise  int64   `json:"prorated_salary_paise"`
	LeaveEncashedDays    float64 `json:"leave_encashed_days"`
	LeaveEncashmentPaise int64   `json:"leave_encashment_paise"`
	YearsOfService       float64 `json:"years_of_service"`
	GratuityEligible     bool    `json:"gratuity_eligible"`
	GratuityPaise        int64   `json:"gratuity_paise"`

	// Freelancer components.
	PendingPayoutPaise int64 `json:"pending_payout_paise"`
	PendingTDSPaise    int64 `json:"pending_tds_paise"`

	DeductionsPaise int64  `json:"deductions_paise"`
	NetPayablePaise int64  `json:"net_payable_paise"`
	Status          string `gorm:"default:'pending'" json:"status"` // pending | paid

	// Exit clearance checklist — settlement is not released until all are cleared.
	ITCleared      bool `json:"it_cleared"`       // accounts/access revoked
	AssetsReturned bool `json:"assets_returned"`  // laptop/badge/devices
	FinanceCleared bool `json:"finance_cleared"`  // advances/dues settled
	HandoverDone   bool `json:"handover_done"`    // knowledge transfer complete

	Notes string `gorm:"type:text" json:"notes"`
}

// ClearanceComplete reports whether every exit-checklist item is signed off.
func (s FinalSettlement) ClearanceComplete() bool {
	return s.ITCleared && s.AssetsReturned && s.FinanceCleared && s.HandoverDone
}

func (FinalSettlement) TableName() string { return "final_settlements" }

// ComputeFinalSettlement builds (but does not persist) the settlement for a user
// as of their last working day.
func ComputeFinalSettlement(userID uint, lastWorkingDay time.Time) (*FinalSettlement, error) {
	db := database.DB
	var profile StaffProfile
	if err := db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return nil, fmt.Errorf("staff profile not found")
	}

	s := &FinalSettlement{
		UserID:         userID,
		EmploymentType: profile.EmploymentType,
		LastWorkingDay: lastWorkingDay,
		Status:         "pending",
	}

	if profile.IsFreelancer() {
		// Accrued-but-unpaid consultation net, grouped by fee for an exact split.
		type bucket struct {
			ConsultationFee int64
			Cnt             int
		}
		var buckets []bucket
		db.Table("appointments").
			Select("consultation_fee, COUNT(*) AS cnt").
			Where("doctor_id = ? AND status = ? AND payout_status = ?", userID, "completed", "pending").
			Group("consultation_fee").Scan(&buckets)
		for _, b := range buckets {
			split := finance.ComputeConsultationSplit(b.ConsultationFee)
			s.PendingPayoutPaise += split.DoctorNet * int64(b.Cnt)
			s.PendingTDSPaise += split.TDS * int64(b.Cnt)
		}
		s.NetPayablePaise = s.PendingPayoutPaise
		s.Notes = "Freelance final settlement: accrued consultation net pending payout."
		return s, nil
	}

	// Salaried: pro-rate the final month up to the last working day.
	daysInMonth := time.Date(lastWorkingDay.Year(), lastWorkingDay.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	daysWorked := lastWorkingDay.Day()
	s.ProratedSalaryPaise = profile.BaseSalary * int64(daysWorked) / int64(daysInMonth)

	// Encash unused paid leave for the current year. Daily rate = monthly base / 30.
	dailyRate := profile.BaseSalary / 30
	var balances []LeaveBalance
	db.Joins("JOIN leave_types lt ON lt.id = leave_balances.leave_type_id").
		Where("leave_balances.user_id = ? AND leave_balances.year = ? AND lt.is_paid = ?",
			userID, lastWorkingDay.Year(), true).
		Find(&balances)
	var encashDays float64
	for _, b := range balances {
		if r := b.Remaining(); r > 0 {
			encashDays += r
		}
	}
	s.LeaveEncashedDays = encashDays
	s.LeaveEncashmentPaise = int64(encashDays * float64(dailyRate))

	// Gratuity (Payment of Gratuity Act): payable after 5 years of continuous
	// service. Formula = last drawn monthly wage × 15/26 × completed years, where
	// a final year over 6 months rounds up. Capped at the statutory ₹20,00,000.
	s.GratuityPaise, s.GratuityEligible, s.YearsOfService = computeGratuity(profile.BaseSalary, profile.JoiningDate, lastWorkingDay)

	s.NetPayablePaise = s.ProratedSalaryPaise + s.LeaveEncashmentPaise + s.GratuityPaise - s.DeductionsPaise
	s.Notes = "Salaried final settlement: pro-rated salary + leave encashment" +
		map[bool]string{true: " + gratuity", false: ""}[s.GratuityEligible] +
		". Statutory deductions (PF/ESI/PT) are processed via the regular payroll run."
	return s, nil
}

// gratuityCapPaise is the statutory maximum gratuity (₹20,00,000).
const gratuityCapPaise int64 = 2000000 * 100

// computeGratuity returns (gratuity paise, eligible, years of service). Returns
// zero when the joining date is unknown or service is under 5 years.
func computeGratuity(monthlyWagePaise int64, joining, lastWorkingDay time.Time) (int64, bool, float64) {
	if joining.IsZero() || joining.Year() < 1900 || !lastWorkingDay.After(joining) {
		return 0, false, 0
	}
	years := lastWorkingDay.Sub(joining).Hours() / (24 * 365.25)
	if years < 5 {
		return 0, false, years
	}
	// Completed years, rounding up a final stub over six months.
	full := int64(years)
	if years-float64(full) > 0.5 {
		full++
	}
	gratuity := monthlyWagePaise * 15 / 26 * full
	if gratuity > gratuityCapPaise {
		gratuity = gratuityCapPaise
	}
	return gratuity, true, years
}

// SaveFinalSettlement upserts the settlement for a user (one active settlement per
// offboarding).
func SaveFinalSettlement(s *FinalSettlement) error {
	var existing FinalSettlement
	if err := database.DB.Where("user_id = ?", s.UserID).First(&existing).Error; err == nil {
		s.ID = existing.ID
		return database.DB.Save(s).Error
	}
	return database.DB.Create(s).Error
}

package hr

import (
	"time"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&StaffProfile{},
		&StaffShift{},
		&StaffAttendance{},
		&PayrollRecord{},
		&LeaveType{},
		&LeaveBalance{},
		&LeaveRequest{},
		&HolidayCalendar{},
		&PayrollCompliance{},
		&Payslip{},
		&PerformanceReview{},
		&OnboardingRecord{},
		&TDSRemittance{},
		&FinalSettlement{},
	)
}

// EmploymentType forks the entire HR/payroll workflow.
//
//   - EmploymentSalaried  → true employee. Runs through ComputePayrollCompliance
//     (PF + ESI + PT + TDS), gets a Payslip, attendance-based pay, Form 16.
//   - EmploymentFreelancer → panel doctor / vendor. NEVER touches payroll. Paid
//     via consultation revenue-share (finance.ComputeConsultationSplit), earnings
//     accrue to a payout ledger, month-end artifact is a Form 16A / TDS certificate.
//
// BaseSalary, PF/ESI and attendance deductions are meaningless for freelancers and
// must be ignored when EmploymentType == EmploymentFreelancer.
const (
	EmploymentSalaried   = "salaried"
	EmploymentFreelancer = "freelancer"
)

type StaffProfile struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	OrganizationID   uint      `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID   uint      `gorm:"index;default:2" json:"business_unit_id"`
	UserID           uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	EmployeeID       string    `gorm:"uniqueIndex;not null" json:"employee_id"`
	EmploymentType   string    `gorm:"default:'salaried';index" json:"employment_type"` // salaried | freelancer
	Department       string    `json:"department"`
	Designation      string    `json:"designation"`
	JoiningDate      time.Time `json:"joining_date"`
	BaseSalary       int64     `json:"base_salary"` // In cents — salaried only; ignored for freelancers
	ManagerID        *uint     `json:"manager_id"`
	EmploymentStatus string    `gorm:"default:'active'" json:"employment_status"` // active, on_leave, terminated
	// Offboarding
	LastWorkingDay    *time.Time `json:"last_working_day,omitempty"`
	TerminationReason string     `json:"termination_reason,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// IsTerminated reports whether the staff member has been offboarded.
func (p StaffProfile) IsTerminated() bool {
	return p.EmploymentStatus == "terminated"
}

// IsFreelancer reports whether this profile is a revenue-share panel doctor
// rather than a salaried employee. Defaults to salaried when unset.
func (p StaffProfile) IsFreelancer() bool {
	return p.EmploymentType == EmploymentFreelancer
}

// BackfillEmploymentTypes is a one-time, idempotent classifier run at startup.
// When the EmploymentType column is first introduced, every existing StaffProfile
// row defaults to 'salaried' — including doctors, who are revenue-share panel
// members and must not be on payroll. This flips existing doctor profiles to
// 'freelancer'.
//
// It is gated on "no freelancer rows exist yet": once the fork is populated (by
// this backfill or by any later manual classification) it never runs again, so a
// doctor an admin deliberately moves onto salary will not be clobbered on restart.
func BackfillEmploymentTypes(db *gorm.DB) {
	var freelancerCount int64
	db.Model(&StaffProfile{}).Where("employment_type = ?", EmploymentFreelancer).Count(&freelancerCount)
	if freelancerCount > 0 {
		return // already populated — do not clobber existing classifications
	}
	// Quote "user" — it is a reserved word and the table is named literally `user`.
	db.Exec(`
		UPDATE staff_profiles SET employment_type = ?
		WHERE user_id IN (SELECT id FROM "user" WHERE role = 'doctor')
	`, EmploymentFreelancer)
}

func (StaffProfile) TableName() string {
	return "staff_profiles"
}

type StaffShift struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	OrganizationID uint      `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID uint      `gorm:"index;default:2" json:"business_unit_id"`
	HospitalID     uint      `gorm:"index" json:"hospital_id"`
	Name       string    `json:"name"`       // e.g. "Morning Shift", "Night Shift"
	StartTime  string    `json:"start_time"` // "08:00"
	EndTime    string    `json:"end_time"`   // "16:00"
	IsActive   bool      `gorm:"default:true" json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (StaffShift) TableName() string {
	return "staff_shifts"
}

type StaffAttendance struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"index;not null" json:"user_id"`
	ShiftID    uint       `gorm:"index" json:"shift_id"`
	Date       time.Time  `gorm:"type:date" json:"date"`
	CheckIn    *time.Time `json:"check_in"`
	CheckOut   *time.Time `json:"check_out"`
	Status     string     `gorm:"default:'present'" json:"status"` // present, late, absent, half_day
	IsVerified bool       `gorm:"default:false" json:"is_verified"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (StaffAttendance) TableName() string {
	return "staff_attendance"
}

type PayrollRecord struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	UserID        uint       `gorm:"index;not null" json:"user_id"`
	Month         int        `json:"month"`
	Year          int        `json:"year"`
	BaseAmount    int64      `json:"base_amount"`
	Bonus         int64      `json:"bonus"`
	Deductions    int64      `json:"deductions"`
	NetAmount     int64      `json:"net_amount"`
	Status        string     `gorm:"default:'pending'" json:"status"` // pending, processed, paid, failed
	PaymentDate   *time.Time `json:"payment_date"`
	TransactionID string     `json:"transaction_id"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (PayrollRecord) TableName() string {
	return "payroll_records"
}

// StaffShiftAssignment is one staff member's shift on one specific date. The
// existing StaffShift is just a hospital-wide TEMPLATE (e.g. "Morning 08:00-16:00");
// this is the actual roster entry: who works which shift on which day. Listed
// admin-side under /admin/hr/shifts.
type StaffShiftAssignment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Date      time.Time `gorm:"index;not null" json:"date"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Notes     string    `json:"notes"`
	Status    string    `gorm:"size:16;default:'scheduled';index" json:"status"` // scheduled | active | completed | cancelled
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (StaffShiftAssignment) TableName() string {
	return "staff_shift_assignments"
}

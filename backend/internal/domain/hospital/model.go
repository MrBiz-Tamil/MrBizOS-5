package hospital

import (
	"time"

	"drayya-core/internal/domain/franchise"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/subscription"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Hospital struct {
	ID             uint                           `gorm:"primaryKey" json:"id"`
	OrganizationID uint                           `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID uint                           `gorm:"index;default:2" json:"business_unit_id"`
	Name           string                         `gorm:"not null" json:"name"`
	Address        string                         `json:"address"`
	ZoneID         *uint                          `gorm:"index" json:"zone_id,omitempty"` // unified geographic zone
	ContactInfo    string                         `json:"contact_info"`
	Type           string                         `gorm:"default:'General'" json:"type"`        // e.g., General, Specialized
	HospitalType   string                         `gorm:"default:'owned'" json:"hospital_type"` // owned, franchise, or saas
	FranchiseID    *uint                          `json:"franchise_id,omitempty"`
	Franchise      *franchise.Franchise           `gorm:"foreignKey:FranchiseID" json:"franchise,omitempty"`
	SubscriptionID *uint                          `json:"subscription_id,omitempty"`
	Subscription   *subscription.UserSubscription `gorm:"foreignKey:SubscriptionID" json:"subscription,omitempty"`
	IsActive       bool                           `gorm:"default:true" json:"is_active"`
	CreatedAt      time.Time                      `json:"created_at"`
	UpdatedAt      time.Time                      `json:"updated_at"`
}

func (Hospital) TableName() string {
	return "hospitals"
}

type HospitalDepartment struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	HospitalID  uint      `gorm:"index" json:"hospital_id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func (HospitalDepartment) TableName() string {
	return "hospital_departments"
}

type Bed struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	HospitalID    uint      `gorm:"index" json:"hospital_id"`
	WardID        *uint     `gorm:"index" json:"ward_id,omitempty"`
	DepartmentID  uint      `gorm:"index" json:"department_id"`
	RoomNumber    string    `json:"room_number"`
	BedNumber     string    `json:"bed_number"`
	BedType       string    `gorm:"default:'standard'" json:"bed_type"`
	Status        string    `gorm:"default:'available'" json:"status"` // available, occupied, maintenance
	PricePerNight float64   `json:"price_per_night"`
	DailyRate     float64   `gorm:"-" json:"daily_rate"` // computed alias for PricePerNight
	WardName      string    `gorm:"-" json:"ward_name,omitempty"`
	PatientName   string    `gorm:"-" json:"patient_name,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Bed) TableName() string {
	return "hospital_beds"
}

// AfterFind populates the computed DailyRate alias so the frontend type matches.
func (b *Bed) AfterFind(tx *gorm.DB) error {
	b.DailyRate = b.PricePerNight
	return nil
}

// Ward represents a physical ward (section of the hospital with beds).
type Ward struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	OrganizationID uint      `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID uint      `gorm:"index;default:2" json:"business_unit_id"`
	HospitalID     uint      `gorm:"index" json:"hospital_id"`
	Name           string    `gorm:"not null" json:"name"`
	WardType    string    `gorm:"default:'general'" json:"ward_type"`
	Floor       int       `json:"floor"`
	Capacity    int       `json:"capacity"`
	Description string    `json:"description"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	// Computed when listing
	CurrentOccupancy int `gorm:"-" json:"current_occupancy"`
	AvailableBeds    int `gorm:"-" json:"available_beds"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (Ward) TableName() string {
	return "hospital_wards"
}

// NursingTask tracks tasks assigned to nursing staff.
type NursingTask struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	HospitalID   uint       `gorm:"index" json:"hospital_id"`
	PatientID    *uint      `json:"patient_id,omitempty"`
	BedID        *uint      `json:"bed_id,omitempty"`
	Title        string     `gorm:"not null" json:"title"`
	Description  string     `json:"description"`
	Priority     string     `gorm:"default:'medium'" json:"priority"` // low | medium | high | urgent
	Status       string     `gorm:"default:'pending'" json:"status"`  // pending | in_progress | completed | cancelled
	DueAt        *time.Time `json:"due_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Notes        string     `json:"notes"`
	AssignedToID *uint      `json:"assigned_to_id,omitempty"`
	CreatedByID  uint       `json:"created_by_id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (NursingTask) TableName() string {
	return "hospital_nursing_tasks"
}

// ShiftSchedule tracks staff shift assignments.
type ShiftSchedule struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	HospitalID uint      `gorm:"index" json:"hospital_id"`
	StaffID    uint      `gorm:"index" json:"staff_id"`
	StaffName  string    `json:"staff_name"`
	StaffRole  string    `json:"staff_role"` // doctor | nurse | technician
	ShiftDate  time.Time `gorm:"index" json:"shift_date"`
	ShiftType  string    `json:"shift_type"` // morning | afternoon | night
	StartTime  string    `json:"start_time"` // "07:00"
	EndTime    string    `json:"end_time"`   // "15:00"
	WardID     *uint     `json:"ward_id,omitempty"`
	Status     string    `gorm:"default:'scheduled'" json:"status"` // scheduled | active | completed | missed
	Notes      string    `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
}

func (ShiftSchedule) TableName() string {
	return "hospital_shift_schedules"
}

// WardServiceRequest represents an in-hospital order for medicines or supplies from a Nurse to the Pharmacy
type WardServiceRequest struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	HospitalID    uint           `gorm:"index" json:"hospital_id"`
	PatientID     *uint          `json:"patient_id"`
	RequestedByID uint           `json:"requested_by_id"` // UserID of the Nurse
	BedID         *uint          `json:"bed_id"`
	Items         datatypes.JSON `json:"items"`                            // JSON [{"medicine_id": 1, "qty": 2}]
	Status        string         `gorm:"default:'pending'" json:"status"`  // pending, processing, dispatched, delivered
	Priority      string         `gorm:"default:'normal'" json:"priority"` // normal, urgent, stat
	Notes         string         `json:"notes"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func (WardServiceRequest) TableName() string {
	return "hospital_ward_service_requests"
}

type IPDAdmission struct {
	ID               uint          `gorm:"primaryKey" json:"id"`
	HospitalID       uint          `gorm:"index" json:"hospital_id"`
	PatientID        uint          `gorm:"index" json:"patient_id"`
	Patient          identity.User `gorm:"foreignKey:PatientID" json:"patient"`
	DoctorID         uint          `gorm:"index" json:"doctor_id"`
	Doctor           identity.User `gorm:"foreignKey:DoctorID" json:"doctor"`
	BedID            uint          `gorm:"index" json:"bed_id"`
	Bed              Bed           `gorm:"foreignKey:BedID" json:"bed"`
	AdmissionDate    time.Time     `json:"admission_date"`
	DischargeDate    *time.Time    `json:"discharge_date"`
	Status           string        `gorm:"default:'admitted'" json:"status"` // admitted, stable, critical, discharged
	DischargeSummary string        `json:"discharge_summary"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

func (IPDAdmission) TableName() string {
	return "hospital_ipd_admissions"
}

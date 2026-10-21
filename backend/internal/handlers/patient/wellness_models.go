package patient

import (
	"time"

	"gorm.io/gorm"
)

// HealthGoal represents a patient's personal health goal
type HealthGoal struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	PatientID    uint           `gorm:"index;not null" json:"patient_id"`
	Title        string         `gorm:"not null" json:"title"`
	Category     string         `gorm:"default:'fitness'" json:"category"` // fitness|nutrition|mental|sleep|weight
	TargetValue  float64        `json:"target_value"`
	CurrentValue float64        `json:"current_value"`
	Unit         string         `json:"unit"`
	Deadline     string         `json:"deadline"`
	Notes        string         `gorm:"type:text" json:"notes"`
	Status       string         `gorm:"default:'active'" json:"status"` // active|achieved|paused
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (HealthGoal) TableName() string { return "patient_health_goals" }

// MedicationReminder represents a patient's medication schedule
type MedicationReminder struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	PatientID    uint           `gorm:"index;not null" json:"patient_id"`
	MedicineName string         `gorm:"not null;column:medicine_name" json:"medicine_name"`
	Dosage       string         `json:"dosage"`
	TimeOfDay    string         `gorm:"type:text" json:"time_of_day"`  // JSON array stored as text
	DaysOfWeek   string         `gorm:"type:text" json:"days_of_week"` // JSON array stored as text
	Notes        string         `gorm:"type:text" json:"notes"`
	Active       bool           `gorm:"default:true" json:"active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (MedicationReminder) TableName() string { return "patient_medication_reminders" }

// SecondOpinion represents a patient's request for a second medical opinion
type SecondOpinion struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	PatientID          uint           `gorm:"index;not null" json:"patient_id"`
	Question           string         `gorm:"type:text;not null" json:"question"`
	Specialty          string         `json:"specialty"`
	Urgency            string         `gorm:"default:'low'" json:"urgency"` // low|medium|high
	CurrentDiagnosis   string         `gorm:"type:text;column:current_diagnosis" json:"current_diagnosis"`
	CurrentMedications string         `gorm:"type:text;column:current_medications" json:"current_medications"`
	Status             string         `gorm:"default:'pending'" json:"status"` // pending|answered|closed
	Response           string         `gorm:"type:text" json:"response,omitempty"`
	RespondingDoctor   string         `gorm:"column:responding_doctor" json:"responding_doctor,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

func (SecondOpinion) TableName() string { return "patient_second_opinions" }

// VaccinationRecord represents a patient's vaccination entry
type VaccinationRecord struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	PatientID   uint           `gorm:"index;not null" json:"patient_id"`
	VaccineName string         `gorm:"not null;column:vaccine_name" json:"vaccine_name"`
	DoseNumber  int            `gorm:"default:1" json:"dose_number"`
	DateGiven   string         `json:"date_given"`
	NextDueDate string         `json:"next_due_date"`
	Provider    string         `json:"provider"`
	BatchNumber string         `json:"batch_number"`
	Notes       string         `gorm:"type:text" json:"notes"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (VaccinationRecord) TableName() string { return "patient_vaccinations" }

// HealthTimelineEvent represents a patient's health history event
type HealthTimelineEvent struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	PatientID uint           `gorm:"index;not null" json:"patient_id"`
	EventType string         `gorm:"not null" json:"event_type"` // appointment|lab_result|medication|vaccination|diagnosis
	Title     string         `gorm:"not null" json:"title"`
	Date      string         `json:"date"`
	Notes     string         `gorm:"type:text" json:"notes"`
	RefID     uint           `json:"ref_id,omitempty"` // optional: linked appointment/lab ID
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (HealthTimelineEvent) TableName() string { return "patient_health_timeline" }

// PatientLabBooking is a patient-initiated lab test booking (distinct from
// clinical lab orders created by doctors via /lab/requests).
type PatientLabBooking struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	PatientID      uint           `gorm:"index;not null" json:"patient_id"`
	TestID         uint           `json:"test_id"`
	TestName       string         `json:"test_name"`
	Date           string         `json:"date"`
	Time           string         `json:"time"`
	HomeCollection bool           `gorm:"default:false" json:"home_collection"`
	AddressID      *uint          `json:"address_id,omitempty"`
	Status         string         `gorm:"default:'booked'" json:"status"` // booked|sample_collected|processing|completed|cancelled
	ReportURL      string         `json:"report_url,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (PatientLabBooking) TableName() string { return "patient_lab_bookings" }

// MigrateWellness runs AutoMigrate for all wellness-related models.
func MigrateWellness(db *gorm.DB) {
	db.AutoMigrate(
		&HealthGoal{},
		&MedicationReminder{},
		&SecondOpinion{},
		&VaccinationRecord{},
		&HealthTimelineEvent{},
		&PatientLabBooking{},
	)
}

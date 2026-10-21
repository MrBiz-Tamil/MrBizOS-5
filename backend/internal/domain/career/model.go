package career

import (
	"time"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&JobOpening{},
		&JobApplication{},
		&RecruitmentTemplate{},
	)
}

type JobOpening struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Title        string    `gorm:"not null" json:"title"`
	Department   string    `json:"department"`
	Location     string    `json:"location"`
	JobType        string `json:"type"` // full-time, part-time, contract
	Specialization string `json:"specialization"` // e.g., "Panchakarma", "General Consultation"
	Description    string `gorm:"type:text" json:"description"`
	Requirements string    `gorm:"type:text" json:"requirements"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (JobOpening) TableName() string {
	return "job_openings"
}

// JobApplication stores the full doctor/staff application form.
// All fields submitted by the frontend apply page are persisted here.
type JobApplication struct {
	ID        uint       `gorm:"primaryKey"        json:"id"`
	JobID     *uint      `gorm:"index"             json:"job_id"`
	Job       JobOpening `gorm:"foreignKey:JobID"  json:"job,omitempty"`
	UserID    *uint      `gorm:"index"             json:"user_id"`

	// ── Contact & Identity ───────────────────────────────────────────────────
	FullName        string `json:"full_name"`
	Email           string `json:"email"`
	PhoneNumber     string `json:"phone_number"`
	WhatsappNumber  string `json:"whatsapp_number"`
	DateOfBirth     string `json:"date_of_birth"`
	Gender          string `json:"gender"`
	FatherName      string `json:"father_name"`
	MotherName      string `json:"mother_name"`
	PassportPhotoURL string `json:"passport_photo_url"`

	// ── Address ──────────────────────────────────────────────────────────────
	PermanentAddress string `gorm:"type:text" json:"permanent_address"`
	LivingAddress    string `gorm:"type:text" json:"living_address"`

	// ── Position & Professional ───────────────────────────────────────────────
	Position          string `json:"position"`           // junior_doctor, senior_doctor, nurse, …
	ExperienceYears   int    `json:"experience_years"`
	LicenseNumber     string `json:"license_number"`
	AYUSHSystem       string `json:"ayush_system"` // siddha, ayurveda, unani, yoga_naturopathy, homeopathy, sowa_rigpa
	TreatmentApproach string `gorm:"type:text" json:"treatment_approach"`

	// ── Education (JSON blob from frontend) ──────────────────────────────────
	EducationDetails string `gorm:"type:text" json:"education_details"` // serialised JSON

	// ── Documents ────────────────────────────────────────────────────────────
	ResumeURL   string `json:"resume_url"`
	CoverLetter string `gorm:"type:text" json:"cover_letter"`

	// ── Workflow ─────────────────────────────────────────────────────────────
	Status          string    `gorm:"default:'applied'" json:"status"` // applied, reviewing, interviewing, offered, onboarded, rejected
	InterviewDate   string    `json:"interview_date"`   // ISO date string set when status = interviewing
	InterviewNotes  string    `gorm:"type:text" json:"interview_notes"`
	AppliedAt       time.Time `json:"applied_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	// Set by AdminOnboardDoctor to link this application to the created user account.
	OnboardedUserID *uint `gorm:"index" json:"onboarded_user_id"`
}

// BeforeCreate auto-sets AppliedAt so it is never zero.
func (a *JobApplication) BeforeCreate(tx *gorm.DB) error {
	if a.AppliedAt.IsZero() {
		a.AppliedAt = time.Now()
	}
	return nil
}

func (JobApplication) TableName() string {
	return "job_applications"
}

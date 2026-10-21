package emr

import (
	"drayya-core/internal/domain/identity"
	"time"
)

// ClinicalReferral represents a doctor-to-doctor or doctor-to-specialist referral
type ClinicalReferral struct {
	ID              uint          `gorm:"primaryKey" json:"id"`
	PatientID       uint          `gorm:"index;not null" json:"patient_id"`
	ReferringDoctorID uint        `gorm:"index;not null" json:"referring_doctor_id"`
	ReferredDoctorID  *uint       `gorm:"index" json:"referred_doctor_id,omitempty"` // nil if open referral
	Specialty       string        `json:"specialty"` // e.g. "Cardiology", "Neurology"
	Urgency         string        `gorm:"default:'routine'" json:"urgency"` // routine, urgent, emergency
	Reason          string        `json:"reason"`
	ClinicalSummary string        `gorm:"type:text" json:"clinical_summary"`
	Notes           string        `gorm:"type:text" json:"notes,omitempty"`
	DiagnosisCodes  string        `json:"diagnosis_codes,omitempty"` // ICD-10 codes CSV
	Status          string        `gorm:"default:'pending'" json:"status"` // pending, accepted, completed, declined
	AppointmentID   *uint         `json:"appointment_id,omitempty"` // linked appointment after acceptance
	AttachmentURL   string        `json:"attachment_url,omitempty"`
	ExpiresAt       *time.Time    `json:"expires_at,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`

	// Relations
	Patient        identity.User `gorm:"foreignKey:PatientID" json:"patient,omitempty"`
	ReferringDoctor identity.User `gorm:"foreignKey:ReferringDoctorID" json:"referring_doctor,omitempty"`
	ReferredDoctor *identity.User `gorm:"foreignKey:ReferredDoctorID" json:"referred_doctor,omitempty"`
}

func (ClinicalReferral) TableName() string {
	return "clinical_referrals"
}

// ReferralNote is a progress note added by the referred specialist after consultation
type ReferralNote struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ReferralID uint      `gorm:"index;not null" json:"referral_id"`
	DoctorID   uint      `gorm:"index;not null" json:"doctor_id"`
	Note       string    `gorm:"type:text" json:"note"`
	Outcome    string    `json:"outcome"` // ongoing_treatment, discharged, admitted, follow_up
	CreatedAt  time.Time `json:"created_at"`

	Doctor identity.User `gorm:"foreignKey:DoctorID" json:"doctor,omitempty"`
}

func (ReferralNote) TableName() string {
	return "referral_notes"
}

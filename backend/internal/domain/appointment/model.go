package appointment

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"drayya-core/internal/domain/identity"
	"strings"
	"time"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Appointment{},
	)
}

type AppointmentStatus string

const (
	StatusPending       AppointmentStatus = "pending"
	StatusConfirmed     AppointmentStatus = "confirmed"
	StatusInConsult     AppointmentStatus = "in_consultation"
	StatusCompleted     AppointmentStatus = "completed"
	StatusCancelled     AppointmentStatus = "cancelled"
	StatusNoShow        AppointmentStatus = "no_show"
	StatusRescheduled   AppointmentStatus = "rescheduled"
	StatusDeclined      AppointmentStatus = "declined"
	StatusDoctorEnRoute AppointmentStatus = "doctor_en_route"
	StatusDoctorArrived AppointmentStatus = "doctor_arrived"
)

type Appointment struct {
	ID               uint              `gorm:"primaryKey" json:"id"`
	BookingID        string            `gorm:"uniqueIndex;type:varchar(20)" json:"booking_id"`
	PatientID        uint              `gorm:"index;not null" json:"patient_id"`
	DoctorID         *uint             `gorm:"index;index:idx_appointment_doctor_status_time,priority:1" json:"doctor_id"`
	StartTime        time.Time         `gorm:"index:idx_appointment_doctor_status_time,priority:3" json:"start_time"`
	EndTime          time.Time         `json:"end_time"`
	Status           AppointmentStatus `gorm:"index;index:idx_appointment_doctor_status_time,priority:2;default:'pending'" json:"status"`
	AppointmentType  string            `json:"appointment_type"`  // consultation, emergency, follow_up
	ConsultationMode string            `json:"consultation_mode"` // video, chat, in_person
	Notes            string            `json:"notes"`
	ConsultationFee  int64             `json:"consultation_fee"`
	IsPriority       bool              `json:"is_priority"`
	FamilyMemberID   *uint             `json:"family_member_id"`
	Latitude         float64           `gorm:"default:0" json:"latitude"`
	Longitude        float64           `gorm:"default:0" json:"longitude"`
	// Payment & payout tracking
	// Indexed: hot path. The payout summary and panel-doctor earnings aggregations
	// filter status='completed' AND payout_status='pending' across the entire
	// appointments table — at 10k+ doctors that becomes a full scan without this.
	PayoutStatus     string            `gorm:"default:'pending';index" json:"payout_status"` // pending | processed
	ReminderSent     bool              `gorm:"default:false" json:"-"`
	// EscalatedAt is stamped by the tiered connect sweep when a live (instant/SOS)
	// request goes unanswered past its lane threshold and is re-broadcast. nil =
	// not yet escalated; prevents re-escalating the same request every tick.
	EscalatedAt      *time.Time        `gorm:"index" json:"escalated_at,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	Version          uint              `gorm:"default:1;not null" json:"version"`

	// Relationships
	Patient PatientProfile `gorm:"foreignKey:PatientID;references:UserID" json:"patient"`
	Doctor  DoctorProfile  `gorm:"foreignKey:DoctorID;references:UserID" json:"doctor"`
}

// Aliases for convenience from identity domain
type PatientProfile = identity.PatientProfile
type DoctorProfile = identity.DoctorProfile

func (a *Appointment) BeforeCreate(tx *gorm.DB) error {
	if a.StartTime.IsZero() {
		a.StartTime = time.Now()
	}
	if a.EndTime.IsZero() {
		a.EndTime = a.StartTime.Add(30 * time.Minute)
	}
	if a.BookingID == "" {
		bytes := make([]byte, 3)
		if _, err := rand.Read(bytes); err == nil {
			a.BookingID = fmt.Sprintf("APT-%s-%s", time.Now().Format("060102"), strings.ToUpper(hex.EncodeToString(bytes)))
		} else {
			a.BookingID = fmt.Sprintf("APT-%s-%d", time.Now().Format("060102"), time.Now().UnixNano()%10000)
		}
	}
	return nil
}

func (a *Appointment) AfterSave(tx *gorm.DB) error {
	// If the table emr_access_grants does not exist, skip the hook gracefully (common in unit tests of other domains)
	if !tx.Migrator().HasTable("emr_access_grants") {
		return nil
	}

	// If appointment status is cancelled or declined, revoke the grant
	if a.Status == "cancelled" || a.Status == "declined" {
		return tx.Exec("DELETE FROM emr_access_grants WHERE source_type = ? AND source_id = ?", "appointment", a.ID).Error
	}

	// If a doctor is assigned, update/upsert the 7-day access grant
	if a.DoctorID != nil {
		expiresAt := a.StartTime.Add(7 * 24 * time.Hour)
		// Delete outdated grants for this appointment first
		tx.Exec("DELETE FROM emr_access_grants WHERE source_type = ? AND source_id = ?", "appointment", a.ID)
		// Insert new grant
		return tx.Exec("INSERT INTO emr_access_grants (doctor_id, patient_id, source_type, source_id, granted_at, expires_at) VALUES (?, ?, ?, ?, ?, ?)",
			*a.DoctorID, a.PatientID, "appointment", a.ID, time.Now(), expiresAt).Error
	}

	// If doctor is unassigned, delete the grant
	return tx.Exec("DELETE FROM emr_access_grants WHERE source_type = ? AND source_id = ?", "appointment", a.ID).Error
}


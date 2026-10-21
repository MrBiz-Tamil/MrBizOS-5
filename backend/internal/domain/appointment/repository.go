package appointment

import (
	"context"
	"strings"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"

	"gorm.io/gorm"
)

type Repository interface {
	CreateAppointment(appt *Appointment) error
	GetAppointmentByID(id string) (*Appointment, error)
	GetAppointmentByIDUint(id uint) (*Appointment, error)
	GetAppointmentsByPatientOrDoctor(userID uint, includePast bool, limit int) ([]Appointment, error)
	GetPendingInstantAppointments() ([]Appointment, error)
	// GetInstantQueueForDoctor returns all unassigned pending instant/emergency appointments
	// PLUS any confirmed instant/emergency appointments that were dispatch-assigned to this
	// specific doctor in the last 30 minutes. This lets the WaitingCallDashboard show
	// appointments even after the dispatch worker has pre-confirmed them.
	GetInstantQueueForDoctor(doctorID uint) ([]Appointment, error)
	GetUpcomingAppointmentsForDoctor(doctorID uint) ([]Appointment, error)
	GetAppointmentHistoryCombo(patientID, doctorID *uint) ([]Appointment, error)
	UpdateAppointment(id string, updates map[string]interface{}) (int64, error)
	UpdateAppointmentAtomic(id string, fromStatus string, updates map[string]interface{}) (int64, error)
	UpdateAppointmentStatus(id string, status string) (int64, error)
	ExecuteTransaction(ctx context.Context, fn func(txCtx context.Context, txRepo Repository, tx *gorm.DB) error) error
	// CheckDoctorAvailability returns true if the doctor has no overlapping appointment
	// (within 30 minutes of scheduledAt). excludeID can be used to exclude a specific
	// appointment from the check (e.g. when rescheduling).
	CheckDoctorAvailability(doctorUserID uint, scheduledAt time.Time, excludeID uint) (bool, error)

	// External dependencies that should eventually logically sit behind an interface
	GetPatientInfo(patientID uint) (*identity.User, error)
	GetDoctorProfile(doctorID uint) (*identity.DoctorProfile, error)
	IncrementPatientFreeConsult(patientID uint) error
	IncrementDoctorFreeConsultGiven(doctorID uint) error
	IncrementDoctorConsultations(doctorUserID uint) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func NewDefaultRepository() Repository {
	return &repository{db: database.DB}
}

func appointmentQuery(db *gorm.DB, id string) *gorm.DB {
	if strings.HasPrefix(id, "APT-") {
		return db.Where("booking_id = ?", id)
	}
	return db.Where("id = ?", id)
}

func (r *repository) CreateAppointment(appt *Appointment) error {
	return r.db.Create(appt).Error
}

func (r *repository) GetAppointmentByID(id string) (*Appointment, error) {
	var appt Appointment
	query := appointmentQuery(r.db.Preload("Patient").Preload("Doctor"), id)
	if err := query.First(&appt).Error; err != nil {
		return nil, err
	}
	return &appt, nil
}

func (r *repository) GetAppointmentByIDUint(id uint) (*Appointment, error) {
	var appt Appointment
	if err := r.db.First(&appt, id).Error; err != nil {
		return nil, err
	}
	return &appt, nil
}

func (r *repository) GetAppointmentsByPatientOrDoctor(userID uint, includePast bool, limit int) ([]Appointment, error) {
	var appointments []Appointment
	now := time.Now()
	query := r.db.Where("patient_id = ? OR doctor_id = ?", userID, userID).
		Preload("Patient").
		Preload("Doctor")

	if includePast {
		query = query.Where("start_time < ?", now).Order("start_time DESC").Limit(limit)
	} else {
		query = query.Where("status IN ? AND start_time > ?", []string{"confirmed", "pending"}, now).
			Order("start_time ASC")
	}

	if err := query.Find(&appointments).Error; err != nil {
		return nil, err
	}
	return appointments, nil
}

func (r *repository) GetUpcomingAppointmentsForDoctor(doctorID uint) ([]Appointment, error) {
	var appointments []Appointment
	err := r.db.Where("doctor_id = ? AND status IN ? AND start_time > ?", doctorID, []string{"confirmed", "pending"}, time.Now()).
		Order("start_time ASC").
		Preload("Patient").
		Find(&appointments).Error
	return appointments, err
}

func (r *repository) GetPendingInstantAppointments() ([]Appointment, error) {
	var appointments []Appointment
	err := r.db.Where("appointment_type IN ? AND status = ?", []string{"instant", "emergency"}, "pending").
		Order("created_at DESC").
		Preload("Patient").
		Limit(20).
		Find(&appointments).Error
	return appointments, err
}

// GetInstantQueueForDoctor returns:
// 1. All unassigned (pending) instant/emergency appointments — visible to every online doctor.
// 2. Confirmed instant/emergency appointments that were dispatch-pre-assigned to THIS doctor
//
//	within the last 30 minutes — ensures the assigned doctor still sees the call even after
//	the dispatch worker has already moved the status to 'confirmed'.
func (r *repository) GetInstantQueueForDoctor(doctorID uint) ([]Appointment, error) {
	cutoff := time.Now().Add(-30 * time.Minute)
	var appointments []Appointment
	err := r.db.Where(
		"(appointment_type IN ? AND status = ?) OR "+
			"(appointment_type IN ? AND status = ? AND doctor_id = ? AND created_at >= ?)",
		[]string{"instant", "emergency"}, "pending",
		[]string{"instant", "emergency"}, "confirmed", doctorID, cutoff,
	).
		Order("created_at DESC").
		Preload("Patient").
		Limit(20).
		Find(&appointments).Error
	return appointments, err
}

func (r *repository) GetAppointmentHistoryCombo(patientID, doctorID *uint) ([]Appointment, error) {
	var history []Appointment
	if patientID == nil || doctorID == nil {
		return history, nil
	}
	err := r.db.Where("patient_id = ? AND doctor_id = ?", *patientID, *doctorID).
		Order("start_time DESC").
		Limit(10).
		Find(&history).Error
	return history, err
}

func (r *repository) UpdateAppointment(id string, updates map[string]interface{}) (int64, error) {
	query := appointmentQuery(r.db.Model(&Appointment{}), id)
	result := query.Updates(updates)
	return result.RowsAffected, result.Error
}

func (r *repository) UpdateAppointmentAtomic(id string, fromStatus string, updates map[string]interface{}) (int64, error) {
	// Read the current version to enforce optimistic locking.
	var appt Appointment
	query := appointmentQuery(r.db.Select("id, version"), id)
	if err := query.First(&appt).Error; err != nil {
		return 0, err
	}
	updates["version"] = appt.Version + 1

	queryUpdate := appointmentQuery(r.db.Model(&Appointment{}), id).
		Where("status = ? AND version = ?", fromStatus, appt.Version)
	result := queryUpdate.Updates(updates)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (r *repository) UpdateAppointmentStatus(id string, status string) (int64, error) {
	query := appointmentQuery(r.db.Model(&Appointment{}), id)
	result := query.Update("status", status)
	return result.RowsAffected, result.Error
}

func (r *repository) ExecuteTransaction(ctx context.Context, fn func(txCtx context.Context, txRepo Repository, tx *gorm.DB) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := &repository{db: tx}
		txCtx := context.WithValue(ctx, database.TxKey{}, tx)
		return fn(txCtx, txRepo, tx)
	})
}

// User methods temporarily placed here to satisfy the business logic dependencies
func (r *repository) GetPatientInfo(patientID uint) (*identity.User, error) {
	var user identity.User
	err := r.db.First(&user, patientID).Error
	return &user, err
}

func (r *repository) GetDoctorProfile(doctorID uint) (*identity.DoctorProfile, error) {
	var profile identity.DoctorProfile
	err := r.db.Where("user_id = ?", doctorID).First(&profile).Error
	return &profile, err
}

func (r *repository) IncrementPatientFreeConsult(patientID uint) error {
	return r.db.Model(&identity.User{}).Where("id = ?", patientID).
		Update("free_consults_used", gorm.Expr("free_consults_used + ?", 1)).Error
}

func (r *repository) IncrementDoctorFreeConsultGiven(doctorID uint) error {
	return r.db.Model(&identity.DoctorProfile{}).Where("user_id = ?", doctorID).
		Update("free_consults_given_today", gorm.Expr("free_consults_given_today + ?", 1)).Error
}

func (r *repository) IncrementDoctorConsultations(doctorUserID uint) error {
	return r.db.Model(&identity.DoctorProfile{}).Where("user_id = ?", doctorUserID).
		Updates(map[string]interface{}{
			"total_consultations":       gorm.Expr("total_consultations + ?", 1),
			"total_consultations_today": gorm.Expr("total_consultations_today + ?", 1),
			"weekly_consultations":      gorm.Expr("weekly_consultations + ?", 1),
		}).Error
}

// CheckDoctorAvailability returns true if the doctor has no conflicting appointment
// within a 30-minute window around scheduledAt. excludeID != 0 skips that appointment
// (useful when rescheduling an existing one).
func (r *repository) CheckDoctorAvailability(doctorUserID uint, scheduledAt time.Time, excludeID uint) (bool, error) {
	windowStart := scheduledAt.Add(-30 * time.Minute)
	windowEnd := scheduledAt.Add(30 * time.Minute)

	query := r.db.Model(&Appointment{}).
		Where("doctor_id = ?", doctorUserID).
		Where("status NOT IN ?", []string{"cancelled", "no_show", "declined"}).
		Where("start_time >= ? AND start_time < ?", windowStart, windowEnd)

	if excludeID != 0 {
		query = query.Where("id != ?", excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}

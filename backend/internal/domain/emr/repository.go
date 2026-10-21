package emr

import (
	"context"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Prescription{},
		&MedicalRecord{},
		&PrescriptionTemplate{},
		&Diagnosis{},
		&PatientVitals{},
		&HealthTimelineEvent{},
		&ClinicalReferral{},
		&ReferralNote{},
		&EMRAccessGrant{},
		&PrescriptionItem{},
	)
}

type Repository interface {
	GetDB() *gorm.DB
	CreatePrescription(ctx context.Context, p *Prescription) error
	GetPrescriptionByID(ctx context.Context, id uint) (*Prescription, error)
	GetPrescriptionByAppointmentID(ctx context.Context, appointmentID uint) (*Prescription, error)
	GetPrescriptionsByDoctor(ctx context.Context, doctorID uint) ([]Prescription, error)
	GetPrescriptionsByPatient(ctx context.Context, patientID uint) ([]Prescription, error)
	UpdatePrescription(ctx context.Context, p *Prescription) error

	CreateMedicalRecord(ctx context.Context, r *MedicalRecord) error
	GetMedicalRecordByID(ctx context.Context, id uint) (*MedicalRecord, error)
	GetMedicalRecordsByPatient(ctx context.Context, patientID uint) ([]MedicalRecord, error)
	GetMedicalRecordsByDoctor(ctx context.Context, doctorID uint) ([]MedicalRecord, error)
	UpdateMedicalRecord(ctx context.Context, id uint, updates map[string]interface{}) error
	DeleteMedicalRecord(ctx context.Context, id uint) error

	CreateDiagnosis(ctx context.Context, d *Diagnosis) error
	GetDiagnosisHistory(ctx context.Context, patientID uint) ([]Diagnosis, error)
}

type defaultRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &defaultRepository{db: db}
}

func (r *defaultRepository) GetDB() *gorm.DB {
	return r.db
}

func (r *defaultRepository) CreatePrescription(ctx context.Context, p *Prescription) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *defaultRepository) GetPrescriptionByID(ctx context.Context, id uint) (*Prescription, error) {
	var p Prescription
	err := r.db.WithContext(ctx).Preload("Doctor").Preload("Patient").First(&p, id).Error
	return &p, err
}

func (r *defaultRepository) GetPrescriptionByAppointmentID(ctx context.Context, appointmentID uint) (*Prescription, error) {
	var p Prescription
	err := r.db.WithContext(ctx).Preload("Doctor").Preload("Patient").
		Where("appointment_id = ?", appointmentID).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *defaultRepository) GetPrescriptionsByDoctor(ctx context.Context, doctorID uint) ([]Prescription, error) {
	var p []Prescription
	err := r.db.WithContext(ctx).Where("doctor_id = ?", doctorID).Order("created_at desc").Find(&p).Error
	return p, err
}

func (r *defaultRepository) GetPrescriptionsByPatient(ctx context.Context, patientID uint) ([]Prescription, error) {
	var p []Prescription
	err := r.db.WithContext(ctx).Where("user_id = ?", patientID).Order("created_at desc").Find(&p).Error
	return p, err
}

func (r *defaultRepository) UpdatePrescription(ctx context.Context, p *Prescription) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *defaultRepository) CreateMedicalRecord(ctx context.Context, rec *MedicalRecord) error {
	return r.db.WithContext(ctx).Create(rec).Error
}

func (r *defaultRepository) GetMedicalRecordByID(ctx context.Context, id uint) (*MedicalRecord, error) {
	var rec MedicalRecord
	err := r.db.WithContext(ctx).Preload("Doctor").Preload("Patient").First(&rec, id).Error
	return &rec, err
}

func (r *defaultRepository) GetMedicalRecordsByPatient(ctx context.Context, patientID uint) ([]MedicalRecord, error) {
	var rec []MedicalRecord
	err := r.db.WithContext(ctx).Where("patient_id = ?", patientID).Order("created_at desc").Find(&rec).Error
	return rec, err
}

func (r *defaultRepository) GetMedicalRecordsByDoctor(ctx context.Context, doctorID uint) ([]MedicalRecord, error) {
	var rec []MedicalRecord
	err := r.db.WithContext(ctx).Where("doctor_id = ?", doctorID).Order("created_at desc").Find(&rec).Error
	return rec, err
}

func (r *defaultRepository) UpdateMedicalRecord(ctx context.Context, id uint, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&MedicalRecord{}).Where("id = ?", id).Updates(updates).Error
}

func (r *defaultRepository) DeleteMedicalRecord(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&MedicalRecord{}, id).Error
}

func (r *defaultRepository) CreateDiagnosis(ctx context.Context, d *Diagnosis) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *defaultRepository) GetDiagnosisHistory(ctx context.Context, patientID uint) ([]Diagnosis, error) {
	var history []Diagnosis
	err := r.db.WithContext(ctx).Where("patient_id = ?", patientID).Order("created_at desc").Find(&history).Error
	return history, err
}

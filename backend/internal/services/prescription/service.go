package prescription

import (
	"encoding/json"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/emr"
	"drayya-core/internal/utils/privacy"
	"time"

	"gorm.io/datatypes"
)

type PrescriptionMedicine struct {
	Name     string `json:"name"`
	Dosage   string `json:"dosage"`
	Duration string `json:"duration"`
}

type CreatePrescriptionRequest struct {
	AppointmentID uint                   `json:"appointment_id"`
	DoctorID      uint                   `json:"doctor_id"`
	PatientID     uint                   `json:"patient_id"`
	Medications   []PrescriptionMedicine `json:"medications"`
	Instructions  string                 `json:"instructions"`
	Diagnosis     string                 `json:"diagnosis"`
}

func CreatePrescription(req CreatePrescriptionRequest) (*emr.Prescription, error) {
	medsJson, err := json.Marshal(req.Medications)
	if err != nil {
		return nil, err
	}

	prescription := &emr.Prescription{
		AppointmentID: req.AppointmentID,
		DoctorID:      req.DoctorID,
		PatientID:     req.PatientID,
		Diagnosis:     privacy.EncryptedString(req.Diagnosis),
		Instructions:  privacy.EncryptedString(req.Instructions),
		Medicines:     datatypes.JSON(medsJson),
		CreatedAt:     time.Now(),
	}

	if err := database.DB.Create(prescription).Error; err != nil {
		return nil, err
	}

	return prescription, nil
}

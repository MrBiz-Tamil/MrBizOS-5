package main

import (
	"encoding/json"
	"log"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/emr"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func seedEMR(db *gorm.DB) {
	// Get completed appointments
	var completed []appointment.Appointment
	db.Where("status = ?", "completed").Find(&completed)

	for _, apt := range completed {
		if apt.DoctorID == nil {
			continue
		}
		// Prescription
		var existingRx emr.Prescription
		if db.Where("appointment_id = ?", apt.ID).First(&existingRx).Error == nil {
			continue
		}
		medsJSON, _ := json.Marshal([]map[string]interface{}{
			{"name": "Dolo 650mg", "dosage": "1 tablet", "frequency": "3 times/day", "duration": "5 days"},
			{"name": "Cetirizine 10mg", "dosage": "1 tablet", "frequency": "Once at night", "duration": "7 days"},
		})
		db.Create(&emr.Prescription{
			AppointmentID: apt.ID,
			DoctorID:      *apt.DoctorID,
			PatientID:     apt.PatientID,
			Diagnosis:     "Upper respiratory infection",
			Symptoms:      "Fever, cough, runny nose",
			Medicines:     datatypes.JSON(medsJSON),
			Instructions:  "Take medicines after food. Drink plenty of water. Rest for 2 days.",
		})

		// Diagnosis
		db.Create(&emr.Diagnosis{
			PatientID:     apt.PatientID,
			DoctorID:      *apt.DoctorID,
			AppointmentID: &apt.ID,
			Condition:     "Acute Upper Respiratory Infection",
			Severity:      "mild",
			Notes:         "Patient presents with typical viral URI symptoms. No signs of bacterial infection.",
		})

		// Medical Record
		db.Create(&emr.MedicalRecord{
			PatientID: apt.PatientID,
			DoctorID:  *apt.DoctorID,
			Diagnosis: "Upper respiratory infection - viral etiology",
			Notes:     "Physical exam: throat mildly inflamed, no lymphadenopathy. Advised symptomatic treatment.",
		})
	}

	// Vitals for first patient
	var firstPatient appointment.Appointment
	db.Where("status = ?", "completed").First(&firstPatient)
	if firstPatient.ID != 0 {
		now := time.Now()
		vitals := []emr.PatientVitals{
			{PatientID: firstPatient.PatientID, SystolicBP: 120, DiastolicBP: 80, HeartRate: 72, RecordedAt: now.Add(-72 * time.Hour)},
			{PatientID: firstPatient.PatientID, SystolicBP: 118, DiastolicBP: 78, HeartRate: 70, RecordedAt: now.Add(-24 * time.Hour)},
			{PatientID: firstPatient.PatientID, SystolicBP: 122, DiastolicBP: 82, HeartRate: 75, RecordedAt: now},
		}
		for _, v := range vitals {
			db.FirstOrCreate(&v, "patient_id = ? AND recorded_at = ?", v.PatientID, v.RecordedAt)
		}
	}

	log.Printf("       Created prescriptions, diagnoses, records for %d completed appointments", len(completed))
}


package main

import (
	"fmt"
	"log"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/lab"
	"time"

	"gorm.io/gorm"
)

func seedLab(db *gorm.DB) {
	// Lab Tests catalog
	tests := []lab.LabTest{
		{Name: "Complete Blood Count (CBC)", Category: "Hematology", Description: "Measures blood cells", ReferenceRange: "WBC: 4000-11000/  L", Unit: "cells/  L", BasePrice: 30000, IsActive: true},
		{Name: "Lipid Profile", Category: "Biochemistry", Description: "Cholesterol and triglycerides", ReferenceRange: "Total Cholesterol: <200 mg/dL", Unit: "mg/dL", BasePrice: 50000, IsActive: true},
		{Name: "Blood Glucose Fasting", Category: "Biochemistry", Description: "Fasting blood sugar", ReferenceRange: "70-100 mg/dL", Unit: "mg/dL", BasePrice: 15000, IsActive: true},
		{Name: "HbA1c", Category: "Biochemistry", Description: "3-month average blood sugar", ReferenceRange: "<5.7%", Unit: "%", BasePrice: 40000, IsActive: true},
		{Name: "Thyroid Panel (TSH, T3, T4)", Category: "Endocrinology", Description: "Thyroid function test", ReferenceRange: "TSH: 0.4-4.0 mIU/L", Unit: "mIU/L", BasePrice: 60000, IsActive: true},
		{Name: "Liver Function Test (LFT)", Category: "Biochemistry", Description: "Liver enzyme levels", ReferenceRange: "ALT: 7-56 U/L", Unit: "U/L", BasePrice: 45000, IsActive: true},
		{Name: "Kidney Function Test (KFT)", Category: "Biochemistry", Description: "Kidney markers", ReferenceRange: "Creatinine: 0.7-1.3 mg/dL", Unit: "mg/dL", BasePrice: 45000, IsActive: true},
		{Name: "Urine Routine", Category: "Urinalysis", Description: "Urine analysis", ReferenceRange: "pH: 4.5-8.0", Unit: "-", BasePrice: 20000, IsActive: true},
		{Name: "Vitamin D Level", Category: "Biochemistry", Description: "25-OH Vitamin D", ReferenceRange: "30-100 ng/mL", Unit: "ng/mL", BasePrice: 55000, IsActive: true},
		{Name: "ECG", Category: "Cardiology", Description: "Electrocardiogram", ReferenceRange: "Normal sinus rhythm", Unit: "-", BasePrice: 30000, IsActive: true},
	}
	for i := range tests {
		db.FirstOrCreate(&tests[i], "name = ?", tests[i].Name)
	}

	// Lab requests
	var doctors []identity.User
	db.Where("role = ?", "doctor").Find(&doctors)
	var patients []identity.User
	db.Where("role = ?", "patient").Find(&patients)
	var labTech identity.User
	db.Where("email = ?", "labtech@Dr.Ayya.com").First(&labTech)

	if len(doctors) < 2 || len(patients) < 3 {
		log.Println("          Not enough doctors/patients for lab requests")
		return
	}

	now := time.Now()
	requests := []lab.LabRequest{
		{HospitalID: 1, PatientID: patients[0].ID, DoctorID: doctors[0].ID, Status: "completed", Priority: "normal", OrderedAt: now.Add(-48 * time.Hour)},
		{HospitalID: 1, PatientID: patients[2].ID, DoctorID: doctors[0].ID, Status: "completed", Priority: "urgent", OrderedAt: now.Add(-24 * time.Hour)},
		{HospitalID: 1, PatientID: patients[1].ID, DoctorID: doctors[1].ID, Status: "in_progress", Priority: "normal", OrderedAt: now.Add(-2 * time.Hour)},
		{HospitalID: 1, PatientID: patients[3].ID, DoctorID: doctors[0].ID, Status: "pending", Priority: "normal", OrderedAt: now},
		{HospitalID: 1, PatientID: patients[0].ID, DoctorID: doctors[1].ID, Status: "pending", Priority: "urgent", OrderedAt: now.Add(-1 * time.Hour)},
	}
	for i := range requests {
		var existing lab.LabRequest
		if db.Where("patient_id = ? AND ordered_at = ?", requests[i].PatientID, requests[i].OrderedAt).First(&existing).Error == nil {
			continue
		}
		db.Create(&requests[i])
	}

	// Results for completed requests
	var completedReqs []lab.LabRequest
	db.Where("status = ?", "completed").Find(&completedReqs)
	for _, req := range completedReqs {
		refMin := 70.0
		refMax := 100.0
		var existing lab.LabResult
		if db.Where("request_id = ?", req.ID).First(&existing).Error == nil {
			continue
		}
		db.Create(&lab.LabResult{
			RequestID:    req.ID,
			TestID:       tests[0].ID,
			Value:        "8500",
			Unit:         "cells/  L",
			ReferenceMin: &refMin,
			ReferenceMax: &refMax,
			Flag:         "normal",
			Comments:     "All values within normal range",
			TechnicianID: labTech.ID,
		})
	}

	// Specimens
	for i, req := range completedReqs {
		collectedAt := req.OrderedAt.Add(30 * time.Minute)
		var existing lab.Specimen
		if db.Where("request_id = ?", req.ID).First(&existing).Error == nil {
			continue
		}
		db.Create(&lab.Specimen{
			RequestID:     req.ID,
			Type:          "blood",
			Barcode:       fmt.Sprintf("SPM-2026-%04d", i+1),
			Status:        "processed",
			CollectedAt:   collectedAt,
			CollectedByID: labTech.ID,
			Notes:         "Sample collected and processed successfully",
		})
	}

	log.Printf("       Created %d tests, %d requests", len(tests), len(requests))
}


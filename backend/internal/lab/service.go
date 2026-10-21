//go:build ignore

package lab

import (
	"drayya-core/internal/database"
	"drayya-core/internal/models"
	"time"
)

func CreateLabRequest(hospitalID uint, patientID uint, doctorID uint, testIDs []uint, priority string) (*models.LabRequest, error) {
	tx := database.DB.Begin()

	var tests []models.LabTest
	if err := tx.Where("id IN ?", testIDs).Find(&tests).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	req := models.LabRequest{
		HospitalID: hospitalID,
		PatientID:  patientID,
		DoctorID:   doctorID,
		Status:     "pending",
		Priority:   priority,
		OrderedAt:  time.Now(),
		Tests:      tests,
	}

	if err := tx.Create(&req).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	return &req, tx.Commit().Error
}

func UpdateSpecimenStatus(requestID uint, specimenType string, barcode string, status string, collectorID uint) error {
	specimen := models.Specimen{
		RequestID:     requestID,
		Type:          specimenType,
		Barcode:       barcode,
		Status:        status,
		CollectedAt:   time.Now(),
		CollectedByID: collectorID,
	}

	tx := database.DB.Begin()
	if err := tx.Create(&specimen).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Update LabRequest status to 'collected' if all specimens are collected (simplified: just update on first specimen)
	if err := tx.Model(&models.LabRequest{}).Where("id = ?", requestID).Updates(map[string]interface{}{
		"status":       "collected",
		"collected_at": time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func SubmitLabResults(requestID uint, technicianID uint, resultData []map[string]interface{}) error {
	tx := database.DB.Begin()

	for _, res := range resultData {
		testID := uint(res["test_id"].(float64))
		value := res["value"].(string)

		var test models.LabTest
		if err := tx.First(&test, testID).Error; err != nil {
			tx.Rollback()
			return err
		}

		labRes := models.LabResult{
			RequestID:    requestID,
			TestID:       testID,
			Value:        value,
			Unit:         test.Unit,
			TechnicianID: technicianID,
			Flag:         "Normal", // Simple flag logic, can be expanded
		}

		if err := tx.Create(&labRes).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Model(&models.LabRequest{}).Where("id = ?", requestID).Update("status", "completed").Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func GetLabRequests(hospitalID uint, status string) ([]models.LabRequest, error) {
	var requests []models.LabRequest
	query := database.DB.Preload("Patient").Preload("Doctor").Preload("Tests")
	if hospitalID != 0 {
		query = query.Where("hospital_id = ?", hospitalID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Find(&requests).Error
	return requests, err
}

func GetLabTests() ([]models.LabTest, error) {
	var tests []models.LabTest
	err := database.DB.Where("is_active = ?", true).Find(&tests).Error
	return tests, err
}

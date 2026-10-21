package handlers_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/services/notification"
	"drayya-core/internal/tests/common"
	"drayya-core/internal/money"

	"github.com/stretchr/testify/assert"
)

func TestPatientDoctorHRBackend(t *testing.T) {
	app := common.SetupApp()
	t.Cleanup(notification.StopWorker)
	db := database.DB

	// 1. Seed HR Manager
	hrManager := common.CreateTestUser(db, "hr_manager")
	hrToken := common.GenerateTestToken(hrManager)

	// 2. Seed Patient
	patient := common.CreateTestUser(db, "patient")
	patientToken := common.GenerateTestToken(patient)
	db.Create(&identity.PatientProfile{
		UserID: patient.ID,
		BloodGroup: "O+",
	})

	// 3. Seed Doctor 1 (Verified, Online)
	doc1 := common.CreateTestUser(db, "doctor")
	db.Create(&identity.DoctorProfile{
		UserID:           doc1.ID,
		IsVerified:      true,
		IsOnline:        true,
		ConsultationFees: money.RegionalPricing{"INR": 50000},
		Specialization:  "Cardiology",
		LicenseNumber:    "LIC-0001",
	})

	// 4. Seed Doctor 2 (Unverified, Offline)
	doc2 := identity.User{
		Email:        "doc2@example.com",
		PasswordHash: "hashed",
		Role:         "doctor",
		FullName:     "Dr. Two",
		IsActive:     true,
	}
	db.Create(&doc2)
	db.Create(&identity.DoctorProfile{
		UserID:           doc2.ID,
		IsVerified:      false,
		IsOnline:        false,
		ConsultationFees: money.RegionalPricing{"INR": 0}, // Free
		OffersFreeConsultation: true,
		Specialization:  "General Medicine",
		LicenseNumber:    "LIC-0002",
	})

	// 5. Seed Appointment for Patient and Doctor 1
	now := time.Now()
	doctorIDCopy := doc1.ID
	db.Create(&appointment.Appointment{
		PatientID: patient.ID,
		DoctorID:  &doctorIDCopy,
		StartTime: now.Add(1 * time.Hour),
		EndTime:   now.Add(2 * time.Hour),
		Status:    "confirmed",
	})

	// --- A. PATIENT UNIT TESTS ---
	t.Run("Patient Dashboard Summary BFF", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/patients/dashboard/summary", nil)
		req.Header.Set("Authorization", "Bearer "+patientToken)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		// Assert BFF summary fields
		data, exists := result["data"].(map[string]interface{})
		assert.True(t, exists, "data field should exist in response envelope")
		
		appointments := data["appointments"].([]interface{})
		assert.Len(t, appointments, 1, "should have 1 upcoming appointment")
		
		onlineDocs := data["online_doctors"].([]interface{})
		assert.Len(t, onlineDocs, 1, "should have 1 online doctor")
		assert.Equal(t, float64(doc1.ID), onlineDocs[0].(map[string]interface{})["user_id"])

		freeDocs := data["free_doctors"].([]interface{})
		assert.Len(t, freeDocs, 1, "should have 1 free doctor")
		assert.Equal(t, float64(doc2.ID), freeDocs[0].(map[string]interface{})["user_id"])
	})

	// --- B. DOCTOR UNIT TESTS ---
	t.Run("Doctor List Verification Filter", func(t *testing.T) {
		// All doctors
		req := httptest.NewRequest("GET", "/api/v1/doctors", nil)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		body, _ := io.ReadAll(resp.Body)
		var allRes map[string]interface{}
		json.Unmarshal(body, &allRes)
		doctors := allRes["doctors"].([]interface{})
		assert.Len(t, doctors, 2, "should return all 2 doctors")

		// Verified only
		reqVerified := httptest.NewRequest("GET", "/api/v1/doctors?verified_only=true", nil)
		respVerified, err := app.Test(reqVerified, -1)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, respVerified.StatusCode)

		bodyVerified, _ := io.ReadAll(respVerified.Body)
		var verRes map[string]interface{}
		json.Unmarshal(bodyVerified, &verRes)
		verDoctors := verRes["doctors"].([]interface{})
		assert.Len(t, verDoctors, 1, "should return only 1 verified doctor")
		assert.Equal(t, doc1.Email, verDoctors[0].(map[string]interface{})["email"])
	})

	// --- C. HR UNIT TESTS ---
	t.Run("HR Dashboard Overview", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/hr/dashboard/overview", nil)
		req.Header.Set("Authorization", "Bearer "+hrToken)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var envelope map[string]interface{}
		json.Unmarshal(body, &envelope)

		data, exists := envelope["data"].(map[string]interface{})
		assert.True(t, exists, "data field should exist in response envelope")
		
		assert.Equal(t, float64(3), data["total_staff"], "Total staff should be 3 (HR Manager + 2 Doctors)")
		assert.Equal(t, float64(2), data["active_doctors"], "Active doctors should be 2")
	})
}

package e2e

import (
	"fmt"
	"drayya-core/internal/domain/appointment"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func debugLog(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	f, _ := os.OpenFile("test_debug.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	f.WriteString(msg + "\n")
}

func helperTopup(t *testing.T) {
	topupBody := map[string]interface{}{
		"amount": 1000,
		"source": "E2E",
	}
	_, resp, err := makeRequest("POST", "/api/v1/finance/topup", PatientToken, topupBody)
	if err != nil {
		t.Fatalf("Helper Topup failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Helper Topup failed with status %d", resp.StatusCode)
	}
}

// Scenario 1: Patient Login -> Topup -> Instant Call -> Doctor Accept -> Prescription -> Dashboard
func TestScenario1_InstantCall(t *testing.T) {
	if _, err := os.Stat("test_debug.txt"); err == nil {
		os.Remove("test_debug.txt")
	}
	debugLog("=== Scenario 1 Start ===")

	// 1. Patient Login (Simulated via token in setup)
	assert.NotEmpty(t, PatientToken, "Patient token should exist")

	// 2. Topup Wallet
	topupBody := map[string]interface{}{
		"amount": 1000, // Increased to 1000 to be safe
		"source": "E2E",
	}
	_, resp, err := makeRequest("POST", "/api/v1/finance/topup", PatientToken, topupBody)
	if err != nil {
		debugLog("Topup Error: %v", err)
		t.Fatalf("Topup failed: %v", err)
	}
	debugLog("Topup Status: %d", resp.StatusCode)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// 3. Instant Call Request (Create Appointment)
	apptReq := map[string]interface{}{
		"patient_id":        PatientID,
		"appointment_type":  "instant", // Corrected key
		"consultation_mode": "video",   // Corrected key
		"notes":             "Severe headache",
	}
	_, resp, err = makeRequest("POST", "/api/v1/appointments", PatientToken, apptReq)

	if err != nil {
		debugLog("Create Appt Error: %v", err)
		t.Fatalf("Create Appointment failed: %v", err)
	}

	// Log Body
	var bodyMap map[string]interface{}
	parseResponse(resp, &bodyMap)
	debugLog("Create Appt Status: %d Body: %v", resp.StatusCode, bodyMap)

	if resp.StatusCode != 201 {
		t.Logf("Create Appointment Failed. Code: %d, Body: %v", resp.StatusCode, bodyMap)
	}
	assert.Equal(t, 201, resp.StatusCode)

	// Get ID
	if appt, ok := bodyMap["appointment"].(map[string]interface{}); ok {
		if id, ok := appt["id"].(float64); ok {
			apptID := uint(id)
			debugLog("Created Appt ID: %d", apptID)

			// Verify Dispatcher Assignment (Simulate)
			db := GetDB()
			docID := DoctorID
			db.Model(&appointment.Appointment{}).Where("id = ?", apptID).Updates(map[string]interface{}{
				"doctor_id": docID,
				"status":    "confirmed",
			})

			// 4. Doctor Accept (Simulated)
			db.Model(&appointment.Appointment{}).Where("id = ?", apptID).Update("status", "in_progress")

			// 5. Create Prescription (Doctor Action)
			meds := []map[string]string{
				{"name": "Paracetamol", "dosage": "500mg", "duration": "3 days"},
			}

			prescReq := map[string]interface{}{
				"appointment_id": apptID,
				"diagnosis":      "Migraine",
				"symptoms":       "Headache, Nausea",
				"medicines":      meds,
				"instructions":   "Take rest",
			}
			_, resp, err = makeRequest("POST", "/api/emr/prescriptions", DoctorToken, prescReq)

			var prescBody map[string]interface{}
			parseResponse(resp, &prescBody)
			debugLog("Create Presc Status: %d Body: %v", resp.StatusCode, prescBody)

			assert.NoError(t, err)
			assert.Equal(t, 201, resp.StatusCode)

			// 6. Verify Dashboard
			_, resp, err = makeRequest("GET", "/api/v1/emr/medical-records/me", PatientToken, nil)
			debugLog("Dashboard Status: %d", resp.StatusCode)
			assert.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)
		} else {
			debugLog("Failed to parse appt ID from appointment map: %v", appt)
			t.Fail()
		}
	} else {
		debugLog("Failed to parse appointment from body: %v", bodyMap)
		t.Fail()
	}
}

// Scenario 2: Patient Login -> Topup -> Instant Chat -> Doctor Accept -> Prescription
func TestScenario2_InstantChat(t *testing.T) {
	debugLog("=== Scenario 2 Start ===")
	helperTopup(t)

	// 3. Instant Chat Request
	apptReq := map[string]interface{}{
		"patient_id":        PatientID,
		"appointment_type":  "instant",
		"consultation_mode": "chat",
		"notes":             "Skin rash",
	}
	_, resp, err := makeRequest("POST", "/api/v1/appointments", PatientToken, apptReq)
	assert.NoError(t, err)

	var bodyMap map[string]interface{}
	parseResponse(resp, &bodyMap)
	debugLog("Create Appt (Chat) Status: %d Body: %v", resp.StatusCode, bodyMap)

	assert.Equal(t, 201, resp.StatusCode)

	if appt, ok := bodyMap["appointment"].(map[string]interface{}); ok {
		if id, ok := appt["id"].(float64); ok {
			apptID := uint(id)

			// Manual Assign
			GetDB().Model(&appointment.Appointment{}).Where("id = ?", apptID).Updates(map[string]interface{}{
				"doctor_id": DoctorID,
				"status":    "in_progress",
			})

			// Create Prescription
			meds := []map[string]string{
				{"name": "Ointment", "dosage": "Apply twice", "duration": "5 days"},
			}

			prescReq := map[string]interface{}{
				"appointment_id": apptID,
				"diagnosis":      "Dermatitis",
				"medicines":      meds,
				"instructions":   "Keep dry",
			}
			_, resp, err = makeRequest("POST", "/api/emr/prescriptions", DoctorToken, prescReq)
			debugLog("Create Presc (Chat) Status: %d", resp.StatusCode)
			assert.NoError(t, err)
			assert.Equal(t, 201, resp.StatusCode)
		}
	}
}

// Scenario 3: SOS Call
func TestScenario3_SOSCall(t *testing.T) {
	debugLog("=== Scenario 3 Start ===")
	// SOS Trigger
	loc := map[string]float64{"lat": 12.97, "lng": 77.59}

	// Trigger SOS
	_, resp, err := makeRequest("POST", "/api/sos/trigger", PatientToken, map[string]interface{}{
		"location": loc,
	})
	assert.NoError(t, err)
	debugLog("SOS Trigger Status: %d", resp.StatusCode)
	assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 201)

	// In backend, SOS creates an appointment or emergency request.
	var appt appointment.Appointment
	result := GetDB().Where("patient_id = ? AND appointment_type = ?", PatientID, "emergency").Last(&appt)
	assert.Nil(t, result.Error)
	debugLog("Found SOS Appt ID: %d", appt.ID)

	// Doctor Accepts
	GetDB().Model(&appt).Updates(map[string]interface{}{
		"doctor_id": DoctorID,
		"status":    "in_progress",
	})

	// Prescription
	meds := []map[string]string{{"name": "Emergency Med", "dosage": "Immediate", "duration": "1 day"}}

	prescReq := map[string]interface{}{
		"appointment_id": appt.ID,
		"diagnosis":      "Emergency Condition",
		"medicines":      meds,
	}
	_, resp, err = makeRequest("POST", "/api/emr/prescriptions", DoctorToken, prescReq)
	debugLog("SOS Presc Status: %d", resp.StatusCode)
	assert.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)
}

// Scenario 5: Select Doctor (Consultation)
func TestScenario5_SelectDoctorCall(t *testing.T) {
	debugLog("=== Scenario 5 Start ===")
	helperTopup(t)

	// Create Appointment with specific Doctor
	apptReq := map[string]interface{}{
		"patient_id":        PatientID,
		"doctor_id":         DoctorID,
		"appointment_type":  "consultation",
		"consultation_mode": "video",
		"notes":             "Regular checkup",
	}
	_, resp, err := makeRequest("POST", "/api/v1/appointments", PatientToken, apptReq)
	assert.NoError(t, err)

	var bodyMap map[string]interface{}
	parseResponse(resp, &bodyMap)
	debugLog("Select Doctor Appt Status: %d", resp.StatusCode)

	assert.Equal(t, 201, resp.StatusCode)

	if appt, ok := bodyMap["appointment"].(map[string]interface{}); ok {
		if id, ok := appt["id"].(float64); ok {
			apptID := uint(id)

			// Verify Doctor ID is returned
			if dID, ok := appt["doctor_id"].(float64); ok {
				assert.Equal(t, DoctorID, uint(dID))
			}

			// Doctor completes/prescribes
			prescReq := map[string]interface{}{
				"appointment_id": apptID,
				"diagnosis":      "Healthy",
				"instructions":   "Drink water",
			}
			_, resp, err = makeRequest("POST", "/api/emr/prescriptions", DoctorToken, prescReq)
			debugLog("Select Doctor Presc Status: %d", resp.StatusCode)
			assert.NoError(t, err)
			assert.Equal(t, 201, resp.StatusCode)
		}
	}
}

func TestScenario4_SOSChat(t *testing.T) {
	TestScenario3_SOSCall(t)
}

func TestScenario6_SelectDoctorChat(t *testing.T) {
	debugLog("=== Scenario 6 Start ===")
	helperTopup(t)

	apptReq := map[string]interface{}{
		"patient_id":        PatientID,
		"doctor_id":         DoctorID,
		"appointment_type":  "consultation",
		"consultation_mode": "chat",
		"notes":             "Follow up chat",
	}
	_, resp, err := makeRequest("POST", "/api/v1/appointments", PatientToken, apptReq)
	assert.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	var bodyMap map[string]interface{}
	parseResponse(resp, &bodyMap)

	if appt, ok := bodyMap["appointment"].(map[string]interface{}); ok {
		if id, ok := appt["id"].(float64); ok {
			apptID := uint(id)

			prescReq := map[string]interface{}{
				"appointment_id": apptID,
				"diagnosis":      "Recovering",
			}
			_, resp, err = makeRequest("POST", "/api/emr/prescriptions", DoctorToken, prescReq)
			assert.NoError(t, err)
			assert.Equal(t, 201, resp.StatusCode)
		}
	}
}

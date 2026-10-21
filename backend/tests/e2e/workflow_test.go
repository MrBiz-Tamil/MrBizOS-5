package e2e

import (
	"fmt"
	"io"
	"drayya-core/internal/domain/pharmacy"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFullPatientJourney_Instant(t *testing.T) {
	runWorkflowTest(t, "instant", "instant")
}

func TestFullPatientJourney_SOS(t *testing.T) {
	runWorkflowTest(t, "sos", "sos")
}

func TestFullPatientJourney_Direct(t *testing.T) {
	runWorkflowTest(t, "consultation", "chat")
}

func runWorkflowTest(t *testing.T, apptType, chatType string) {
	// 1. Doctor Login
	docLoginReq := map[string]string{
		"email":    TestDoctorEmail,
		"password": TestPassword,
	}
	_, resp, err := makeRequest("POST", "/api/v1/auth/login", "", docLoginReq)
	require.NoError(t, err)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Doctor login failed: status %d, body: %s", resp.StatusCode, string(body))
	}
	var docLoginResp LoginResponse
	parseResponse(resp, &docLoginResp)
	DoctorToken = docLoginResp.AccessToken
	DoctorID = docLoginResp.User.ID

	// 2. Patient Login
	loginReq := map[string]string{
		"email":    TestPatientEmail,
		"password": TestPassword,
	}
	_, resp, err = makeRequest("POST", "/api/v1/auth/login", "", loginReq)
	require.NoError(t, err)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Patient login failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var loginResp LoginResponse
	parseResponse(resp, &loginResp)
	PatientToken = loginResp.AccessToken
	PatientID = loginResp.User.ID

	// 3. Wallet Top-up
	topupReq := map[string]interface{}{
		"amount": 1000.0,
		"source": "Stripe",
		"txn_id": "TEST-TXN-123",
	}
	_, resp, err = makeRequest("POST", "/api/v1/finance/topup", PatientToken, topupReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// 4. Call/Chat - Create Room
	chatReq := map[string]interface{}{
		"doctor_id":  DoctorID,
		"patient_id": PatientID,
		"chat_type":  chatType,
	}
	_, resp, err = makeRequest("POST", "/api/v1/communication/chat/rooms", PatientToken, chatReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// 5. End of consultation - Doctor creates a prescription
	apptReq := map[string]interface{}{
		"doctor_id":         DoctorID,
		"appointment_type":  apptType,
		"consultation_mode": chatType,
	}
	_, resp, err = makeRequest("POST", "/api/v1/appointments", PatientToken, apptReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var apptResp struct {
		Appointment struct {
			ID uint `json:"id"`
		} `json:"appointment"`
	}
	err = parseResponse(resp, &apptResp)
	require.NoError(t, err)
	apptID := apptResp.Appointment.ID

	prescReq := PrescriptionRequest{
		AppointmentID: apptID,
		Medications: []MedicationItem{
			{Name: "Paracetamol", Dosage: "500mg", Duration: "3 days", Quantity: 1},
		},
		Instructions: "Take after meals",
	}
	_, resp, err = makeRequest("POST", "/api/v1/emr/prescriptions", DoctorToken, prescReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var prescResp struct {
		ID uint `json:"id"`
	}
	err = parseResponse(resp, &prescResp)
	require.NoError(t, err)

	// 6. Patient gives a doctors rating
	ratingReq := map[string]interface{}{
		"doctor_id":      DoctorID,
		"appointment_id": apptID,
		"score":          5,
	}
	_, resp, err = makeRequest("POST", "/api/v1/ratings", PatientToken, ratingReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// 7. Doctor share a prescription, patient dashboard and pharmacy (Convert to order)
	pharmacyID := uint(1)
	GetDB().FirstOrCreate(&pharmacy.Pharmacy{ID: pharmacyID, Name: "E2E Pharmacy", IsActive: true})

	// H-8: Rx status must be "verified" before conversion.
	GetDB().Table("prescriptions").Where("id = ?", prescResp.ID).Update("status", "verified")

	convertReq := map[string]interface{}{
		"prescription_id":  prescResp.ID,
		"pharmacy_id":      pharmacyID,
		"shipping_address": "123 Test St, BLR",
		"phone":            "9876543210",
	}
	_, resp, err = makeRequest("POST", "/api/v1/pharmacy/prescriptions/convert", PatientToken, convertReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var orderResp struct {
		ID uint `json:"id"`
	}
	err = parseResponse(resp, &orderResp)
	require.NoError(t, err)

	// 8. Patient complete a payment pharmacy
	_, resp, err = makeRequest("POST", fmt.Sprintf("/api/v1/pharmacy/orders/%d/pay", orderResp.ID), PatientToken, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// 9. Pharmacy dispatch a medicine (Shipping update)
	delivery := pharmacy.CourierDelivery{
		ID:        uint(time.Now().UnixNano() % 10000),
		OrderID:   orderResp.ID,
		CourierID: DoctorID,
		Status:    "assigned",
	}
	GetDB().FirstOrCreate(&delivery)

	dispatchReq := map[string]interface{}{
		"status": "picked_up",
	}
	_, resp, err = makeRequest("PUT", fmt.Sprintf("/api/v1/courier/delivery/%d/status", delivery.ID), DoctorToken, dispatchReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	fmt.Println("--- WORKFLOW TEST COMPLETED SUCCESSFULLY ---")
}

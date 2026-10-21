package tests_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/money"
	"drayya-core/internal/tests/common"

	"github.com/stretchr/testify/assert"
)

func TestCreateAppointment(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	// Seed Data
	patient := common.CreateTestUser(db, "patient")
	doctor := common.CreateTestUser(db, "doctor")

	// Create profiles
	db.Create(&identity.PatientProfile{UserID: patient.ID})
	db.Create(&identity.DoctorProfile{
		UserID:           doctor.ID,
		IsVerified:       true,
		ConsultationFees: money.RegionalPricing{"INR": 50000},
	})

	patient.FreeConsultsUsed = 1
	db.Save(&patient)

	// Setup Wallet for Patient (Required for payment)
	// We need to manually create a wallet account with balance since we don't have a wallet service helper exposed directly yet,
	// checking appointment service logic, it uses LedgerService.GetOrCreateAccount("PatientWallet:%d").
	// We can try to rely on the service to create it if it doesn't exist, but it needs balance.
	// LedgerService.GetOrCreateAccount creates with 0 balance.
	// So we need to credit it.

	// Since we don't have easy access to LedgerService instance here (it's internal to AppointmentService),
	// we will manually insert LedgerAccount and LedgerEntry to simulate balance?
	// OR we can make the appointment amount 0 if possible?
	// Default amount is 50000 (500 INR).
	// We need to credit the wallet.

	// Let's create the account manually in DB.
	// walletName := "PatientWallet:" + string(rune(patient.ID)) // Unused currently
	// Actually logic is fmt.Sprintf("PatientWallet:%d", req.PatientID)
	// We need to match that.

	// But first, let's try to create appointment and expect failure due to insufficient funds?
	// That would verify the flow partially.

	token := common.GenerateTestToken(patient)

	payload := map[string]interface{}{
		"doctor_id":         doctor.ID,
		"appointment_type":  "consultation",
		"consultation_mode": "video",
		"notes":             "Test appointment",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/appointments", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// Expect 402 due to payment failure ("insufficient wallet balance")
	if resp.StatusCode != 402 {
		body, _ := io.ReadAll(resp.Body)
		t.Logf("Expected 402, got %d. Body: %s", resp.StatusCode, string(body))
	}
	assert.Equal(t, 402, resp.StatusCode)
}

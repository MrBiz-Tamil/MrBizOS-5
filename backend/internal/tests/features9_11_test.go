package tests_test

// Tests for features 9–11:
//   9.  pgvector + Devi symptom memory
//  10.  Doctor payout API
//  11.  Admin revenue dashboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	devdomain "drayya-core/internal/domain/devi"
	"drayya-core/internal/domain/identity"
	payoutdomain "drayya-core/internal/domain/payout"
	"drayya-core/internal/tests/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// Feature 9: Devi Symptom Memory
// ─────────────────────────────────────────────────────────────────────────────

func TestDeviHistory_Empty(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	patient := common.CreateTestUser(db, "patient")
	token := common.GenerateTestToken(patient)

	req := httptest.NewRequest("GET", "/api/v1/devi/history", nil)
	req.Header.Set("Authorization", authHeader(token))

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	// data should be an array (empty)
	data, _ := body["data"].([]interface{})
	assert.Equal(t, 0, len(data))
}

func TestDeviHistory_WithSeededLogs(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	patient := common.CreateTestUser(db, "patient")
	token := common.GenerateTestToken(patient)

	// Seed 3 symptom logs directly
	for i := 0; i < 3; i++ {
		db.Create(&devdomain.SymptomLog{
			UserID:       patient.ID,
			SessionID:    fmt.Sprintf("sess-%d", i),
			SymptomsText: fmt.Sprintf("fever and cough iteration %d", i),
			DeptMatched:  "General Medicine",
			Urgency:      3,
		})
	}

	req := httptest.NewRequest("GET", "/api/v1/devi/history?limit=10", nil)
	req.Header.Set("Authorization", authHeader(token))

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data, _ := body["data"].([]interface{})
	assert.Equal(t, 3, len(data), "should return all 3 seeded logs")
}

func TestDeviHistory_IsolatedPerUser(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	alice := common.CreateTestUser(db, "patient")
	bob := common.CreateTestUser(db, "patient")
	token := common.GenerateTestToken(alice)

	// Seed logs for Alice and Bob
	db.Create(&devdomain.SymptomLog{UserID: alice.ID, SymptomsText: "Alice headache", DeptMatched: "Neurology", Urgency: 2})
	db.Create(&devdomain.SymptomLog{UserID: bob.ID, SymptomsText: "Bob back pain", DeptMatched: "Ortho", Urgency: 2})

	req := httptest.NewRequest("GET", "/api/v1/devi/history", nil)
	req.Header.Set("Authorization", authHeader(token))

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data, _ := body["data"].([]interface{})
	assert.Equal(t, 1, len(data), "Alice should only see her own log")
}

func TestDeviHistory_Delete(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	patient := common.CreateTestUser(db, "patient")
	token := common.GenerateTestToken(patient)

	db.Create(&devdomain.SymptomLog{UserID: patient.ID, SymptomsText: "knee pain", Urgency: 3})

	delReq := httptest.NewRequest("DELETE", "/api/v1/devi/history", nil)
	delReq.Header.Set("Authorization", authHeader(token))
	resp, err := app.Test(delReq, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Confirm deleted
	var count int64
	db.Model(&devdomain.SymptomLog{}).Where("user_id = ?", patient.ID).Count(&count)
	assert.Equal(t, int64(0), count, "all logs should be erased after DELETE")
}

func TestDeviHistory_SimilarSearch_MissingQ(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	patient := common.CreateTestUser(db, "patient")
	token := common.GenerateTestToken(patient)

	req := httptest.NewRequest("GET", "/api/v1/devi/history/similar", nil) // no q
	req.Header.Set("Authorization", authHeader(token))

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestDeviHistory_SimilarSearch_NoAPIKey_FallsBack(t *testing.T) {
	// Without GEMINI_API_KEY, similar search should fall back to recency
	app := common.SetupApp()
	db := database.DB

	patient := common.CreateTestUser(db, "patient")
	token := common.GenerateTestToken(patient)

	db.Create(&devdomain.SymptomLog{UserID: patient.ID, SymptomsText: "chest pain", DeptMatched: "Cardiology", Urgency: 8})

	req := httptest.NewRequest("GET", "/api/v1/devi/history/similar?q=heart+pain", nil)
	req.Header.Set("Authorization", authHeader(token))

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode) // should not error
}

func TestDeviHistory_AdminView(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	adminUser := common.CreateTestUser(db, "admin")
	patient := common.CreateTestUser(db, "patient")
	adminToken := common.GenerateTestToken(adminUser)

	db.Create(&devdomain.SymptomLog{UserID: patient.ID, SymptomsText: "rash", DeptMatched: "Dermatology", Urgency: 2})

	url := fmt.Sprintf("/api/v1/admin/devi/history/%d", patient.ID)
	req := httptest.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", authHeader(adminToken))

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

// ─────────────────────────────────────────────────────────────────────────────
// Feature 9b: Devi symptom repository unit tests (no HTTP)
// ─────────────────────────────────────────────────────────────────────────────

func TestDeviRepository_SaveAndGetHistory(t *testing.T) {
	common.SetupApp()
	db := database.DB

	patient := common.CreateTestUser(db, "patient")
	repo := devdomain.NewRepository(db)

	require.NoError(t, repo.Save(&devdomain.SymptomLog{
		UserID:       patient.ID,
		SessionID:    "s1",
		SymptomsText: "sore throat and fever",
		DeptMatched:  "ENT",
		Urgency:      4,
	}))

	logs, err := repo.GetUserHistory(patient.ID, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, len(logs))
	assert.Equal(t, "sore throat and fever", logs[0].SymptomsText)
}

func TestDeviRepository_FindSimilar_InProcess(t *testing.T) {
	// In-process cosine similarity (no GEMINI_API_KEY, so embedding is empty)
	common.SetupApp()
	db := database.DB

	patient := common.CreateTestUser(db, "patient")
	repo := devdomain.NewRepository(db)

	// Seed logs without embeddings — FindSimilar should fall back to GetUserHistory
	repo.Save(&devdomain.SymptomLog{UserID: patient.ID, SymptomsText: "cold", Urgency: 2})
	repo.Save(&devdomain.SymptomLog{UserID: patient.ID, SymptomsText: "flu", Urgency: 3})

	results, err := repo.FindSimilar(patient.ID, nil, 5) // nil embedding
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results), 5)
}

// ─────────────────────────────────────────────────────────────────────────────
// Feature 10: Doctor Payout API
// ─────────────────────────────────────────────────────────────────────────────

func TestListPayouts_Empty(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	adminUser := common.CreateTestUser(db, "admin")
	token := common.GenerateTestToken(adminUser)

	req := httptest.NewRequest("GET", "/api/v1/admin/payouts", nil)
	req.Header.Set("Authorization", authHeader(token))

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestListPayouts_WithRecords(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	adminUser := common.CreateTestUser(db, "admin")
	doctor := common.CreateTestUser(db, "doctor")
	token := common.GenerateTestToken(adminUser)

	db.Create(&payoutdomain.Record{
		DoctorUserID:     doctor.ID,
		AmountPaise:      40000, // ₹400
		RazorpayPayoutID: "pay_TESTPAYOUT1",
		RefID:            fmt.Sprintf("PAYOUT-TEST-%d", time.Now().UnixNano()),
		Status:           "processed",
		Mode:             "IMPS",
		AppointmentCount: 2,
	})

	req := httptest.NewRequest("GET", "/api/v1/admin/payouts", nil)
	req.Header.Set("Authorization", authHeader(token))

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data := body["data"].(map[string]interface{})
	total := data["total"].(float64)
	assert.GreaterOrEqual(t, total, float64(1))
}

func TestListPayouts_FilterByStatus(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	adminUser := common.CreateTestUser(db, "admin")
	doctor := common.CreateTestUser(db, "doctor")
	token := common.GenerateTestToken(adminUser)

	db.Create(&payoutdomain.Record{
		DoctorUserID: doctor.ID, AmountPaise: 20000,
		RefID: fmt.Sprintf("PAYOUT-FAIL-%d", time.Now().UnixNano()),
		Status: "failed",
	})

	req := httptest.NewRequest("GET", "/api/v1/admin/payouts?status=failed", nil)
	req.Header.Set("Authorization", authHeader(token))

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestPayoutSummary(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	adminUser := common.CreateTestUser(db, "admin")
	doctor := common.CreateTestUser(db, "doctor")
	token := common.GenerateTestToken(adminUser)

	db.Create(&payoutdomain.Record{
		DoctorUserID:     doctor.ID,
		AmountPaise:      80000,
		RefID:            fmt.Sprintf("PAYOUT-SUM-%d", time.Now().UnixNano()),
		RazorpayPayoutID: "pay_SUM001",
		Status:           "processed",
	})

	// Seed a pending appointment for same doctor
	docID := doctor.ID
	db.Create(&identity.PatientProfile{UserID: doctor.ID}) // avoid FK error
	patient := common.CreateTestUser(db, "patient")
	db.Create(&identity.PatientProfile{UserID: patient.ID})
	db.Create(&appointment.Appointment{
		PatientID:       patient.ID,
		DoctorID:        &docID,
		ConsultationFee: 50000,
		Status:          appointment.StatusCompleted,
		PayoutStatus:    "pending",
	})

	req := httptest.NewRequest("GET", "/api/v1/admin/payouts/summary", nil)
	req.Header.Set("Authorization", authHeader(token))

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data, _ := body["data"].([]interface{})
	assert.Greater(t, len(data), 0, "should contain at least one doctor in summary")
}

func TestDoctorPayoutHistory(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	adminUser := common.CreateTestUser(db, "admin")
	doctor := common.CreateTestUser(db, "doctor")
	token := common.GenerateTestToken(adminUser)

	db.Create(&payoutdomain.Record{
		DoctorUserID:     doctor.ID,
		AmountPaise:      30000,
		RefID:            fmt.Sprintf("PAYOUT-HIST-%d", time.Now().UnixNano()),
		Status:           "processed",
	})

	url := fmt.Sprintf("/api/v1/admin/payouts/doctor/%d", doctor.ID)
	req := httptest.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", authHeader(token))

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data, _ := body["data"].([]interface{})
	assert.Equal(t, 1, len(data))
}

func TestTriggerPayout_NoPendingAppointments(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	adminUser := common.CreateTestUser(db, "admin")
	token := common.GenerateTestToken(adminUser)

	req := httptest.NewRequest("POST", "/api/v1/admin/payouts/trigger",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", authHeader(token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["processed"])
}

func TestUpdatePayoutDetails_Success(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	doctor := common.CreateTestUser(db, "doctor")
	// Create doctor profile first
	db.Create(&identity.DoctorProfile{UserID: doctor.ID, LicenseNumber: fmt.Sprintf("LIC%d", doctor.ID)})
	token := common.GenerateTestToken(doctor)

	payload := map[string]interface{}{
		"payout_upi": "doctor@ybl",
	}
	req := httptest.NewRequest("PUT", "/api/v1/doctors/me/payout-details",
		mustJSON(t, payload))
	req.Header.Set("Authorization", authHeader(token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify in DB
	var profile identity.DoctorProfile
	db.Where("user_id = ?", doctor.ID).First(&profile)
	assert.Equal(t, "doctor@ybl", profile.PayoutUPI)
}

func TestUpdatePayoutDetails_MissingFields(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	doctor := common.CreateTestUser(db, "doctor")
	token := common.GenerateTestToken(doctor)

	// Neither UPI nor bank account provided
	req := httptest.NewRequest("PUT", "/api/v1/doctors/me/payout-details",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", authHeader(token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

// ─────────────────────────────────────────────────────────────────────────────
// Feature 11: Admin Revenue Dashboard
// ─────────────────────────────────────────────────────────────────────────────

func TestRevenueDashboard(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	adminUser := common.CreateTestUser(db, "admin")
	token := common.GenerateTestToken(adminUser)

	req := httptest.NewRequest("GET", "/api/v1/admin/revenue/dashboard", nil)
	req.Header.Set("Authorization", authHeader(token))

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data := body["data"].(map[string]interface{})
	// Check top-level keys
	assert.Contains(t, data, "summary")
	assert.Contains(t, data, "payouts")
	assert.Contains(t, data, "counts")
}

func TestRevenueDashboard_WithData(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	adminUser := common.CreateTestUser(db, "admin")
	doctor := common.CreateTestUser(db, "doctor")
	patient := common.CreateTestUser(db, "patient")
	db.Create(&identity.PatientProfile{UserID: patient.ID})
	token := common.GenerateTestToken(adminUser)

	docID := doctor.ID
	// Seed completed appointments with fee
	db.Create(&appointment.Appointment{
		PatientID:       patient.ID,
		DoctorID:        &docID,
		ConsultationFee: 50000, // ₹500
		Status:          appointment.StatusCompleted,
	})
	db.Create(&appointment.Appointment{
		PatientID:       patient.ID,
		DoctorID:        &docID,
		ConsultationFee: 75000, // ₹750
		Status:          appointment.StatusCompleted,
	})

	req := httptest.NewRequest("GET", "/api/v1/admin/revenue/dashboard", nil)
	req.Header.Set("Authorization", authHeader(token))

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data := body["data"].(map[string]interface{})
	summary := data["summary"].(map[string]interface{})
	consultations := summary["consultations"].(map[string]interface{})
	allTime := consultations["all_time"].(float64)
	assert.Equal(t, 1250.0, allTime, "₹500 + ₹750 = ₹1250 all-time consultation revenue")
}

func TestRevenueTrends(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	adminUser := common.CreateTestUser(db, "admin")
	token := common.GenerateTestToken(adminUser)

	req := httptest.NewRequest("GET", "/api/v1/admin/revenue/trends?days=7", nil)
	req.Header.Set("Authorization", authHeader(token))

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data, _ := body["data"].([]interface{})
	assert.Equal(t, 7, len(data), "should return 7 day entries")

	// Verify each entry has the required fields
	day0 := data[0].(map[string]interface{})
	assert.Contains(t, day0, "date")
	assert.Contains(t, day0, "revenue_inr")
	assert.Contains(t, day0, "consultation_count")
}

func TestTopDoctors(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	adminUser := common.CreateTestUser(db, "admin")
	doctor := common.CreateTestUser(db, "doctor")
	patient := common.CreateTestUser(db, "patient")
	db.Create(&identity.PatientProfile{UserID: patient.ID})
	token := common.GenerateTestToken(adminUser)

	docID := doctor.ID
	db.Create(&appointment.Appointment{
		PatientID:       patient.ID,
		DoctorID:        &docID,
		ConsultationFee: 60000,
		Status:          appointment.StatusCompleted,
	})

	req := httptest.NewRequest("GET", "/api/v1/admin/revenue/doctors?limit=5", nil)
	req.Header.Set("Authorization", authHeader(token))

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	data, _ := body["data"].([]interface{})
	assert.Greater(t, len(data), 0)

	top := data[0].(map[string]interface{})
	assert.Contains(t, top, "gross_revenue_inr")
	assert.Contains(t, top, "doctor_share_inr")
	assert.Contains(t, top, "platform_fee_inr")
	assert.Contains(t, top, "consultations")

	grossRev := top["gross_revenue_inr"].(float64)
	assert.Equal(t, 600.0, grossRev)
	platformFee := top["platform_fee_inr"].(float64)
	assert.Equal(t, 120.0, platformFee, "20% of ₹600 = ₹120")
}

func TestRevenueDashboard_Unauthorized(t *testing.T) {
	app := common.SetupApp()

	req := httptest.NewRequest("GET", "/api/v1/admin/revenue/dashboard", nil)
	// No token

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)
}

package hr_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strconv"
	"testing"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/hr"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/tests/common"

	"github.com/stretchr/testify/assert"
)

// TestOnboardingFlow tests the full onboarding lifecycle:
//   1. Create an HR Manager token
//   2. Seed a StaffProfile with an OnboardingRecord
//   3. GET /hr/onboarding   → list should include the record
//   4. GET /hr/onboarding/:id → verify doc defaults
//   5. PUT /hr/onboarding/:id/documents → mark certificate as "verified"
//   6. PUT /hr/onboarding/:id/tasks    → mark system_access_granted = true
//   7. PUT /hr/onboarding/:id/status   → move to "in_training" (triggers auto IT email goroutine)
//   8. PUT /hr/onboarding/:id/status   → move to "completed"
func TestOnboardingFlow(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	// --- Seed: HR Manager user + token ---
	hrManager := common.CreateTestUser(db, "hr_manager")
	token := common.GenerateTestToken(hrManager)

	// --- Seed: Identity user for the employee ---
	employee := identity.User{
		Email:        "emp_onboard@drtest.com",
		PasswordHash: "hashed",
		Role:         "nurse",
		FullName:     "New Nurse",
		IsActive:     true,
	}
	db.Create(&employee)

	// --- Seed: StaffProfile for the employee ---
	profile := hr.StaffProfile{
		UserID:           employee.ID,
		EmployeeID:       "EMP-9001",
		Department:       "ICU",
		Designation:      "Senior Nurse",
		EmploymentStatus: "active",
	}
	db.Create(&profile)

	baseURL := "/api/v1/hr/onboarding"

	// ── Step 0: POST /hr/onboarding/initiate (Initiate onboarding) ────────────
	initiateBody := bytes.NewBufferString(fmt.Sprintf(`{"staff_id":%d}`, profile.ID))
	req := httptest.NewRequest("POST", baseURL+"/initiate", initiateBody)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 5000)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode, "initiate onboarding")

	var record hr.OnboardingRecord
	json.NewDecoder(resp.Body).Decode(&record)
	assert.Equal(t, profile.ID, record.StaffID)
	assert.Equal(t, "pending_docs", record.Status)

	recordIDStr := strconv.Itoa(int(record.ID))

	// ── Step 1: GET /hr/onboarding (list) ─────────────────────────────────────
	req = httptest.NewRequest("GET", baseURL+"/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req, 5000)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode, "list onboarding records")

	var records []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&records)
	assert.NotEmpty(t, records, "should have at least 1 onboarding record")

	// ── Step 2: GET /hr/onboarding/:id ────────────────────────────────────────
	req = httptest.NewRequest("GET", baseURL+"/"+recordIDStr, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req, 5000)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode, "get onboarding by ID")

	var detail map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&detail)
	assert.Equal(t, "pending_docs", detail["status"], "default status should be pending_docs")
	assert.Equal(t, "pending", detail["certificate"], "certificate should default to pending")

	// ── Step 2.5: POST /hr/onboarding/:id/document-reminder ───────────────────
	req = httptest.NewRequest("POST", baseURL+"/"+recordIDStr+"/document-reminder", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req, 5000)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode, "send document reminder")

	var reminderResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&reminderResp)
	assert.Equal(t, "Document reminder email sent successfully", reminderResp["message"])

	// ── Step 3: PUT /hr/onboarding/:id/documents → verify certificate ─────────
	body := bytes.NewBufferString(`{"doc_type":"certificate","status":"verified"}`)
	req = httptest.NewRequest("PUT", baseURL+"/"+recordIDStr+"/documents", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, 5000)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode, "update document status")

	// confirm it persisted
	var updated hr.OnboardingRecord
	db.First(&updated, record.ID)
	assert.Equal(t, "verified", updated.Certificate, "certificate should now be verified")

	// ── Step 4: PUT /hr/onboarding/:id/tasks → grant system access ───────────
	body = bytes.NewBufferString(`{"task_type":"system_access_granted","is_complete":true}`)
	req = httptest.NewRequest("PUT", baseURL+"/"+recordIDStr+"/tasks", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, 5000)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode, "update task status")

	db.First(&updated, record.ID)
	assert.True(t, updated.SystemAccess, "system_access_granted should be true")

	// ── Step 5: Move status → in_training (triggers auto IT email goroutine) ──
	body = bytes.NewBufferString(`{"status":"in_training"}`)
	req = httptest.NewRequest("PUT", baseURL+"/"+recordIDStr+"/status", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, 5000)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode, "update status to in_training")

	db.First(&updated, record.ID)
	assert.Equal(t, "in_training", updated.Status, "status should be in_training")

	// ── Step 6: POST /hr/onboarding/:id/it-request (manual IT email) ─────────
	body = bytes.NewBufferString(`{"message":"Please provision Adobe CC license for EMP-9001"}`)
	req = httptest.NewRequest("POST", baseURL+"/"+recordIDStr+"/it-request", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, 5000)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode, "manual IT request should succeed")

	var itResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&itResp)
	assert.Equal(t, "IT Request sent", itResp["message"])

	// ── Step 7: Move status → completed ───────────────────────────────────────
	body = bytes.NewBufferString(`{"status":"completed"}`)
	req = httptest.NewRequest("PUT", baseURL+"/"+recordIDStr+"/status", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, 5000)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode, "update status to completed")

	db.First(&updated, record.ID)
	assert.Equal(t, "completed", updated.Status, "final status should be completed")

	fmt.Println("✅ HR Onboarding end-to-end test PASSED")
}

// TestOnboardingBadDocType verifies that an invalid doc_type returns an error
func TestOnboardingBadDocType(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	hrManager := common.CreateTestUser(db, "hr_manager")
	token := common.GenerateTestToken(hrManager)

	employee := identity.User{Email: "emp2@drtest.com", PasswordHash: "x", Role: "nurse", IsActive: true}
	db.Create(&employee)
	profile := hr.StaffProfile{UserID: employee.ID, EmployeeID: "EMP-9002", EmploymentStatus: "active"}
	db.Create(&profile)
	record := hr.OnboardingRecord{StaffID: profile.ID, Status: "pending_docs"}
	db.Create(&record)

	body := bytes.NewBufferString(`{"doc_type":"invalid_field","status":"submitted"}`)
	req := httptest.NewRequest("PUT", "/api/v1/hr/onboarding/"+strconv.Itoa(int(record.ID))+"/documents", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 5000)
	assert.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode, "invalid doc_type should return 500")
}

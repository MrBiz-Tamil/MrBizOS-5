package hospital_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/billing"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/franchise"
	"drayya-core/internal/domain/hospital"
	"drayya-core/internal/tests/common"

	"github.com/stretchr/testify/assert"
)

func TestGetHospitalDashboard(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	// Seed beds (no ward counted from real data — service uses CountWards)
	db.Create(&hospital.Bed{Status: "occupied", HospitalID: 1})
	db.Create(&hospital.Bed{Status: "available", HospitalID: 1})

	req := httptest.NewRequest("GET", "/api/v1/hospital/dashboard", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var envelope map[string]interface{}
	json.Unmarshal(body, &envelope)
	result := envelope["data"].(map[string]interface{})

	assert.Equal(t, float64(2), result["total_beds"])
	assert.Equal(t, float64(1), result["occupied_beds"])
	assert.Equal(t, float64(0), result["total_wards"]) // No wards seeded
}

func TestListWards(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	db.Create(&hospital.Ward{HospitalID: 1, Name: "General A", WardType: "general", Floor: 1, Capacity: 20, IsActive: true})
	db.Create(&hospital.Ward{HospitalID: 1, Name: "ICU", WardType: "icu", Floor: 2, Capacity: 8, IsActive: true})

	req := httptest.NewRequest("GET", "/api/v1/hospital/wards?hospital_id=1", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var envelope map[string]interface{}
	json.Unmarshal(body, &envelope)
	wards := envelope["data"].([]interface{})
	assert.Len(t, wards, 2)
}

func TestListBeds_FilterByWard(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	w1 := hospital.Ward{HospitalID: 1, Name: "General", WardType: "general", Capacity: 5, IsActive: true}
	db.Create(&w1)
	w2 := hospital.Ward{HospitalID: 1, Name: "ICU", WardType: "icu", Capacity: 3, IsActive: true}
	db.Create(&w2)

	wid1 := w1.ID
	wid2 := w2.ID
	db.Create(&hospital.Bed{HospitalID: 1, WardID: &wid1, BedNumber: "B01", Status: "available"})
	db.Create(&hospital.Bed{HospitalID: 1, WardID: &wid1, BedNumber: "B02", Status: "available"})
	db.Create(&hospital.Bed{HospitalID: 1, WardID: &wid2, BedNumber: "B03", Status: "occupied"})

	// Filter by ward 1 → 2 beds
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/hospital/beds?ward_id=%d", wid1), nil)
	resp, _ := app.Test(req, -1)
	assert.Equal(t, 200, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	var envelope map[string]interface{}
	json.Unmarshal(body, &envelope)
	assert.Len(t, envelope["data"].([]interface{}), 2)

	// No filter → all 3 beds
	req2 := httptest.NewRequest("GET", "/api/v1/hospital/beds?hospital_id=1", nil)
	resp2, _ := app.Test(req2, -1)
	body2, _ := io.ReadAll(resp2.Body)
	var envelope2 map[string]interface{}
	json.Unmarshal(body2, &envelope2)
	assert.Len(t, envelope2["data"].([]interface{}), 3)
}

func TestCallNextPatient_NoWaiting(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	nurse := common.CreateTestUser(db, "nurse")
	token := common.GenerateTestToken(nurse)

	req := httptest.NewRequest("POST", "/api/v1/hospitals/1/opd/call-next", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode) // No waiting patient → 404
}

func TestCompleteTask_Lifecycle(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	nurse := common.CreateTestUser(db, "nurse")
	token := common.GenerateTestToken(nurse)

	task := hospital.NursingTask{
		HospitalID:  1,
		Title:       "Check vitals — Patient #5",
		Status:      "pending",
		Priority:    "high",
		CreatedByID: nurse.ID,
	}
	db.Create(&task)

	body := bytes.NewBufferString(`{"notes":"Vitals normal"}`)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/hospital/tasks/%d/complete", task.ID), body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify DB
	var updated hospital.NursingTask
	db.First(&updated, task.ID)
	assert.Equal(t, "completed", updated.Status)
	assert.NotNil(t, updated.CompletedAt)
	assert.Equal(t, "Vitals normal", updated.Notes)
}

func TestCompleteTask_AlreadyCompleted(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	nurse := common.CreateTestUser(db, "nurse")
	token := common.GenerateTestToken(nurse)

	task := hospital.NursingTask{
		HospitalID:  1,
		Title:       "Done task",
		Status:      "completed",
		CreatedByID: nurse.ID,
	}
	db.Create(&task)

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/hospital/tasks/%d/complete", task.ID), bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req, -1)
	assert.Equal(t, 400, resp.StatusCode) // Cannot complete already-completed task
}

func TestListTasks_FilterByStatus(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	nurse := common.CreateTestUser(db, "nurse")
	token := common.GenerateTestToken(nurse)

	db.Create(&hospital.NursingTask{HospitalID: 1, Title: "Task 1", Status: "pending", CreatedByID: nurse.ID})
	db.Create(&hospital.NursingTask{HospitalID: 1, Title: "Task 2", Status: "in_progress", CreatedByID: nurse.ID})
	db.Create(&hospital.NursingTask{HospitalID: 1, Title: "Task 3", Status: "completed", CreatedByID: nurse.ID})

	// Without filter → returns active only (excludes completed)
	req := httptest.NewRequest("GET", "/api/v1/hospital/tasks?hospital_id=1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ := app.Test(req, -1)
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var envelope map[string]interface{}
	json.Unmarshal(body, &envelope)
	tasks := envelope["data"].([]interface{})
	assert.Len(t, tasks, 2) // pending + in_progress
}

func TestListShifts_Empty(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	nurse := common.CreateTestUser(db, "nurse")
	token := common.GenerateTestToken(nurse)

	req := httptest.NewRequest("GET", "/api/v1/hospital/shifts?hospital_id=1", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, _ := app.Test(req, -1)
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var envelope map[string]interface{}
	json.Unmarshal(body, &envelope)
	shifts := envelope["data"].([]interface{})
	assert.Len(t, shifts, 0)
}

func TestEnforceTenancy_CrossTenantBlocked(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	// Create user belonging to Hospital 2
	nurse := common.CreateTestUser(db, "nurse")
	hospID := uint(2)
	nurse.HospitalID = &hospID
	db.Save(&nurse)
	token := common.GenerateTestToken(nurse)

	// Attempting to access Hospital 1's tasks or beds should be blocked with 403 Forbidden
	req := httptest.NewRequest("GET", "/api/v1/hospital/tasks?hospital_id=1", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, _ := app.Test(req, -1)
	assert.Equal(t, 403, resp.StatusCode)
}

func TestDynamicFranchiseRoyalty(t *testing.T) {
	_ = common.SetupApp()
	db := database.DB

	// Ensure franchise table auto-migrated
	db.AutoMigrate(&franchise.Franchise{}, &franchise.FranchiseRoyaltyLog{})

	// 1. Create a Franchise Partner and link it to Hospital 3
	franOwner := common.CreateTestUser(db, "admin")
	fran := franchise.Franchise{
		ID:          uint(100),
		OwnerID:     franOwner.ID,
		Name:        "MediHub Franchise Delhi",
		RoyaltyRate: 12.5,
		IsActive:    true,
	}
	db.Create(&fran)

	franHosp := hospital.Hospital{
		ID:           uint(3),
		Name:         "MediHub Franchise Hospital Delhi",
		HospitalType: "franchise",
		FranchiseID:  &fran.ID,
		IsActive:     true,
	}
	db.Create(&franHosp)

	// 2. Create doctor and patient
	doctor := common.CreateTestUser(db, "doctor")
	hospID := uint(3)
	doctor.HospitalID = &hospID
	db.Save(&doctor)

	patient := common.CreateTestUser(db, "patient")

	// 3. Create appointment & invoice
	appt := appointment.Appointment{
		PatientID:       patient.ID,
		DoctorID:        &doctor.ID,
		Status:          appointment.StatusPending,
		ConsultationFee: 100000, // 1000 INR (100,000 Paise)
		StartTime:       time.Now(),
		EndTime:         time.Now().Add(30 * time.Minute),
	}
	db.Create(&appt)

	invoice := billing.Billing{
		AppointmentID: &appt.ID,
		PatientID:     patient.ID,
		Amount:        100000,
		PaymentStatus: "pending",
	}
	db.Create(&invoice)

	// 4. Trigger Billing Payment Mock (which processes dynamic royalty)
	billingSvc := billing.NewService(billing.NewRepository(db), finance.NewService(finance.NewRepository(db)))
	payment, err := billingSvc.ProcessPaymentMock(invoice.ID, "upi")
	assert.NoError(t, err)
	assert.NotNil(t, payment)

	// Allow goroutine to process royalty split
	time.Sleep(100 * time.Millisecond)

	// 5. Verify FranchiseRoyaltyLog entry is created with correct royalty amount (12.5% of 100,000 paise = 12,500 paise)
	var royaltyLog franchise.FranchiseRoyaltyLog
	err = db.Where("franchise_id = ?", fran.ID).First(&royaltyLog).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(100000), royaltyLog.TotalAmount)
	assert.Equal(t, int64(12500), royaltyLog.RoyaltyAmount)
}

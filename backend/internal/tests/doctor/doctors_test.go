package doctor_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/tests/common"

	"github.com/stretchr/testify/assert"
)

// pagedDoctorsResponse mirrors the direct paged response from GetDoctors (no wrapper envelope).
type pagedDoctorsResponse struct {
	Doctors []map[string]interface{} `json:"doctors"`
	Total   int64                    `json:"total"`
	Page    int                      `json:"page"`
	Limit   int                      `json:"limit"`
	HasMore bool                     `json:"has_more"`
}

func TestGetDoctors(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	// Seed Data
	d1 := common.CreateTestUser(db, "doctor")
	db.Create(&identity.DoctorProfile{
		UserID:     d1.ID,
		IsVerified: true,
	})

	d2 := identity.User{
		Email:        "test_doctor_2@example.com",
		PasswordHash: "hashed_secret",
		Role:         "doctor",
		FullName:     "Test doctor 2",
		IsActive:     true,
	}
	db.Create(&d2)
	db.Create(&identity.DoctorProfile{
		UserID:     d2.ID,
		IsVerified: false,
	})

	common.CreateTestUser(db, "patient")

	// Test 1: Get All Doctors
	req := httptest.NewRequest("GET", "/api/v1/doctors", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("ALL DOCTORS:", string(body))
	var result pagedDoctorsResponse
	json.Unmarshal(body, &result)
	assert.Len(t, result.Doctors, 2)

	// Test 2: Get Verified Doctors — filter param is now verified_only=true
	reqVerified := httptest.NewRequest("GET", "/api/v1/doctors?verified_only=true", nil)
	respVerified, err := app.Test(reqVerified, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, respVerified.StatusCode)

	bodyVerified, _ := io.ReadAll(respVerified.Body)
	fmt.Println("VERIFIED DOCTORS:", string(bodyVerified))
	var resultVerified pagedDoctorsResponse
	json.Unmarshal(bodyVerified, &resultVerified)
	if len(resultVerified.Doctors) > 0 {
		assert.Equal(t, d1.Email, resultVerified.Doctors[0]["email"])
	}
	assert.Len(t, resultVerified.Doctors, 1)
}

func TestGetWaitingRoom(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	doctor := common.CreateTestUser(db, "doctor")
	token := common.GenerateTestToken(doctor)

	// Seed two appointments for today
	ist, _ := time.LoadLocation("Asia/Kolkata")
	nowIST := time.Now().In(ist)
	todayStart := time.Date(nowIST.Year(), nowIST.Month(), nowIST.Day(), 0, 0, 0, 0, ist)

	patient1 := common.CreateTestUser(db, "patient")
	// Make sure patient profile exists
	db.Create(&identity.PatientProfile{
		UserID: patient1.ID,
	})

	doctorIDCopy := doctor.ID
	db.Create(&appointment.Appointment{
		PatientID: patient1.ID,
		DoctorID:  &doctorIDCopy,
		StartTime: todayStart.Add(2 * time.Hour),
		EndTime:   todayStart.Add(2 * time.Hour).Add(30 * time.Minute),
		Status:    "confirmed",
	})
	db.Create(&appointment.Appointment{
		PatientID: patient1.ID,
		DoctorID:  &doctorIDCopy,
		StartTime: todayStart.Add(4 * time.Hour),
		EndTime:   todayStart.Add(4 * time.Hour).Add(30 * time.Minute),
		Status:    "pending",
	})

	req := httptest.NewRequest("GET", "/api/doctors/waiting-room", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	// Check counts
	assert.Equal(t, float64(2), result["count"])
}

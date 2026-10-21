package patient_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/tests/common"

	"github.com/stretchr/testify/assert"
)

func TestPatientProfile(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	patient := common.CreateTestUser(db, "patient")
	token := common.GenerateTestToken(patient)

	// 1. Create profile
	profileData := map[string]interface{}{
		"gender":            "Male",
		"blood_group":       "O+",
		"height":            180,
		"weight":            75,
		"address":           "123 Test St",
		"emergency_contact": "9876543210",
	}
	body, _ := json.Marshal(profileData)

	req := httptest.NewRequest("POST", "/api/v1/patients/profile", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	// Verify in DB directly
	var dbProfile identity.PatientProfile
	err = db.Where("user_id = ?", patient.ID).First(&dbProfile).Error
	assert.NoError(t, err)
	assert.Equal(t, "O+", dbProfile.BloodGroup)

	// 2. Get profile — response.JSON wraps in {"success":true,"data":{...}}
	reqGet := httptest.NewRequest("GET", "/api/v1/patients/profile", nil)
	reqGet.Header.Set("Authorization", "Bearer "+token)

	respGet, err := app.Test(reqGet, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, respGet.StatusCode)

	bodyGet, _ := io.ReadAll(respGet.Body)
	var envelope map[string]interface{}
	json.Unmarshal(bodyGet, &envelope)
	data := envelope["data"].(map[string]interface{})
	assert.Equal(t, "O+", data["blood_group"])
}

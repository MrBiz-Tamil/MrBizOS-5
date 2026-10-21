package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/emr"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/server"
	"drayya-core/internal/tokens"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type EMRGrantIntegrationTestSuite struct {
	suite.Suite
	App *fiber.App
	DB  *gorm.DB
}

func (suite *EMRGrantIntegrationTestSuite) SetupSuite() {
	os.Setenv("USE_SQLITE", "true")
	os.Setenv("DB_URL", "file:memdb_emr_grant?mode=memory&cache=shared")
	os.Setenv("JWT_SECRET", "test_secret_key")

	db, err := gorm.Open(sqlite.Open("file:memdb_emr_grant?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}
	suite.DB = db
	database.DB = db

	// Migrate required models
	identity.AutoMigrate(db)
	appointment.AutoMigrate(db)
	emr.AutoMigrate(db)

	suite.App = server.New()
}

func (suite *EMRGrantIntegrationTestSuite) Request(method, path string, body interface{}, token string) *http.Response {
	var bodyReader *bytes.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(jsonBody)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, _ := suite.App.Test(req, -1)
	return resp
}

func (suite *EMRGrantIntegrationTestSuite) TestEMRAccessGrants() {
	db := suite.DB

	// 1. Create Users
	patient := identity.User{
		Email:    "john@test.com",
		FullName: "Patient John",
		Role:     "patient",
	}
	db.Create(&patient)

	doctorA := identity.User{
		Email:    "doctorA@test.com",
		FullName: "Doctor A",
		Role:     "doctor",
	}
	db.Create(&doctorA)
	doctorAToken, _ := tokens.CreateAccessToken(fmt.Sprintf("%d", doctorA.ID), "doctor", nil, nil)

	doctorB := identity.User{
		Email:    "doctorB@test.com",
		FullName: "Doctor B",
		Role:     "doctor",
	}
	db.Create(&doctorB)
	doctorBToken, _ := tokens.CreateAccessToken(fmt.Sprintf("%d", doctorB.ID), "doctor", nil, nil)

	doctorC := identity.User{
		Email:    "doctorC@test.com",
		FullName: "Doctor C",
		Role:     "doctor",
	}
	db.Create(&doctorC)
	doctorCToken, _ := tokens.CreateAccessToken(fmt.Sprintf("%d", doctorC.ID), "doctor", nil, nil)

	// 2. Doctor A treats John (3 months ago)
	apptA := appointment.Appointment{
		PatientID:        patient.ID,
		DoctorID:         &doctorA.ID,
		StartTime:        time.Now().Add(-90 * 24 * time.Hour),
		EndTime:          time.Now().Add(-90 * 24 * time.Hour).Add(30 * time.Minute),
		Status:           appointment.StatusCompleted,
		AppointmentType:  "consultation",
		ConsultationMode: "chat",
	}
	db.Create(&apptA)

	// Create Prescription from Doctor A
	prescA := emr.Prescription{
		AppointmentID: apptA.ID,
		DoctorID:      doctorA.ID,
		PatientID:     patient.ID,
		Diagnosis:     "High blood pressure",
		Symptoms:      "Headache",
		Medicines:     []byte(`[{"name":"Medicine X","dosage":"10mg","duration":"30 days"}]`),
		Instructions:  "Take daily",
		Status:        "issued",
		PdfURL:        "storage/prescriptions/test.pdf",
	}
	db.Create(&prescA)

	// Create Medical Record from Doctor A
	medRecA := emr.MedicalRecord{
		PatientID: patient.ID,
		DoctorID:  doctorA.ID,
		Diagnosis: "High blood pressure",
		Notes:     "Patient John has hypertension.",
	}
	db.Create(&medRecA)

	// Create Diagnosis from Doctor A
	diagA := emr.Diagnosis{
		PatientID:     patient.ID,
		DoctorID:      doctorA.ID,
		AppointmentID: &apptA.ID,
		Condition:     "Hypertension",
		Severity:      "Moderate",
		Notes:         "Monitor daily",
	}
	db.Create(&diagA)

	// 3. Verify Doctor A can access all these records by ID
	// Prescription by ID
	resp := suite.Request("GET", fmt.Sprintf("/api/v1/emr/prescriptions/%d", prescA.ID), nil, doctorAToken)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	// Prescription by Appointment ID
	resp = suite.Request("GET", fmt.Sprintf("/api/v1/emr/prescriptions/appointment/%d", apptA.ID), nil, doctorAToken)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	// Medical Record Details
	resp = suite.Request("GET", fmt.Sprintf("/api/v1/emr/medical-records/%d", medRecA.ID), nil, doctorAToken)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	// Diagnosis by Appointment
	resp = suite.Request("GET", fmt.Sprintf("/api/v1/emr/diagnosis/appointment/%d", apptA.ID), nil, doctorAToken)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	// 4. Verify Doctor B (no appointment today yet) CANNOT access the records
	resp = suite.Request("GET", fmt.Sprintf("/api/v1/emr/prescriptions/%d", prescA.ID), nil, doctorBToken)
	assert.Equal(suite.T(), 403, resp.StatusCode)

	resp = suite.Request("GET", fmt.Sprintf("/api/v1/emr/prescriptions/appointment/%d", apptA.ID), nil, doctorBToken)
	assert.Equal(suite.T(), 403, resp.StatusCode)

	resp = suite.Request("GET", fmt.Sprintf("/api/v1/emr/medical-records/%d", medRecA.ID), nil, doctorBToken)
	assert.Equal(suite.T(), 403, resp.StatusCode)

	resp = suite.Request("GET", fmt.Sprintf("/api/v1/emr/diagnosis/appointment/%d", apptA.ID), nil, doctorBToken)
	assert.Equal(suite.T(), 403, resp.StatusCode)

	// 5. Today: John books an appointment with Doctor B
	apptB := appointment.Appointment{
		PatientID:        patient.ID,
		DoctorID:         &doctorB.ID,
		StartTime:        time.Now(),
		EndTime:          time.Now().Add(30 * time.Minute),
		Status:           appointment.StatusConfirmed,
		AppointmentType:  "consultation",
		ConsultationMode: "video",
	}
	db.Create(&apptB)

	// Verify an EMR Access Grant was automatically generated for Doctor B
	var grant emr.EMRAccessGrant
	err := db.Where("doctor_id = ? AND patient_id = ? AND source_id = ?", doctorB.ID, patient.ID, apptB.ID).First(&grant).Error
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "appointment", grant.SourceType)
	assert.True(suite.T(), grant.ExpiresAt.After(time.Now()))

	// 6. Verify Doctor B CAN now access Doctor A's historical records!
	// Prescription by ID
	resp = suite.Request("GET", fmt.Sprintf("/api/v1/emr/prescriptions/%d", prescA.ID), nil, doctorBToken)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	// Prescription by Appointment
	resp = suite.Request("GET", fmt.Sprintf("/api/v1/emr/prescriptions/appointment/%d", apptA.ID), nil, doctorBToken)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	// Medical Record Details
	resp = suite.Request("GET", fmt.Sprintf("/api/v1/emr/medical-records/%d", medRecA.ID), nil, doctorBToken)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	// Diagnosis by Appointment
	resp = suite.Request("GET", fmt.Sprintf("/api/v1/emr/diagnosis/appointment/%d", apptA.ID), nil, doctorBToken)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	// 7. Verify Doctor C (unrelated) still gets 403 Forbidden
	resp = suite.Request("GET", fmt.Sprintf("/api/v1/emr/prescriptions/%d", prescA.ID), nil, doctorCToken)
	assert.Equal(suite.T(), 403, resp.StatusCode)

	// 8. Revocation test: Patient cancels appointment with Doctor B
	apptB.Status = appointment.StatusCancelled
	db.Save(&apptB)

	// Verify grant is deleted
	var count int64
	db.Model(&emr.EMRAccessGrant{}).Where("doctor_id = ? AND patient_id = ?", doctorB.ID, patient.ID).Count(&count)
	assert.Equal(suite.T(), int64(0), count)

	// Verify Doctor B is now blocked again (403)
	resp = suite.Request("GET", fmt.Sprintf("/api/v1/emr/prescriptions/%d", prescA.ID), nil, doctorBToken)
	assert.Equal(suite.T(), 403, resp.StatusCode)
}

func TestEMRGrantIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(EMRGrantIntegrationTestSuite))
}

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/career"
	"drayya-core/internal/domain/communication"
	"drayya-core/internal/domain/emr"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/hr"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/loyalty"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/domain/store"
	"drayya-core/internal/services/notification"
	"drayya-core/internal/money"
	"drayya-core/internal/server"
	"drayya-core/internal/tokens"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type PatientDoctorHRIntegrationTestSuite struct {
	suite.Suite
	App *fiber.App
	DB  *gorm.DB
}

func (suite *PatientDoctorHRIntegrationTestSuite) SetupSuite() {
	os.Setenv("USE_SQLITE", "true")
	os.Setenv("DB_URL", "file:memdb_pd_hr?mode=memory&cache=shared")
	os.Setenv("JWT_SECRET", "test_secret_key")

	db, err := gorm.Open(sqlite.Open("file:memdb_pd_hr?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}
	suite.DB = db
	database.DB = db

	// AutoMigrate all required models
	identity.AutoMigrate(db)
	finance.AutoMigrate(db)
	appointment.AutoMigrate(db)
	hr.AutoMigrate(db)
	emr.AutoMigrate(db)
	communication.AutoMigrate(db)
	notification.AutoMigrateOutbox(db)
	career.AutoMigrate(db)
	pharmacy.AutoMigrate(db)
	store.AutoMigrate(db)
	loyalty.AutoMigrate(db)

	finance.SeedTaxRules(db)

	suite.App = server.New()
}

func (suite *PatientDoctorHRIntegrationTestSuite) TearDownSuite() {
	notification.StopWorker()
}

func (suite *PatientDoctorHRIntegrationTestSuite) Request(method, path string, body interface{}, token string) *http.Response {
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

func (suite *PatientDoctorHRIntegrationTestSuite) TestPatientDoctorHRFlow() {
	db := suite.DB

	// 1. Register HR Manager and get token
	hrManager := identity.User{
		Email:    "hr@test.com",
		FullName: "HR Manager",
		Role:     "hr_manager",
		IsActive: true,
	}
	db.Create(&hrManager)
	hrToken, _ := tokens.CreateAccessToken(fmt.Sprintf("%d", hrManager.ID), "hr_manager", nil, nil)

	// 2. Register Doctor (via API)
	regResp := suite.Request("POST", "/api/v1/auth/register", map[string]interface{}{
		"email":     "doctor_new@test.com",
		"password":  "password123",
		"full_name": "Dr. Onboarded",
		"role":      "doctor",
	}, "")
	assert.Equal(suite.T(), 201, regResp.StatusCode)

	var loginResp struct {
		Data map[string]interface{} `json:"data"`
	}
	json.NewDecoder(regResp.Body).Decode(&loginResp)
	docUserMap := loginResp.Data["user"].(map[string]interface{})
	docUserID := uint(docUserMap["id"].(float64))

	// Find the automatically created DoctorProfile for this doctor
	var docProfile identity.DoctorProfile
	err := db.Where("user_id = ?", docUserID).First(&docProfile).Error
	assert.NoError(suite.T(), err)

	// Update doctor profile fees and specialization and unique LicenseNumber
	db.Model(&docProfile).Updates(map[string]interface{}{
		"consultation_fees": money.RegionalPricing{"INR": 60000},
		"specialization":    "Cardiology",
		"license_number":    "LIC-INT-001",
		"is_online":         true, // Set online for patient visibility
	})

	// 3. HR Manager initiates doctor onboarding
	// Create StaffProfile first
	staffProfile := hr.StaffProfile{
		UserID:           docUserID,
		EmployeeID:       "DOC-EMP-100",
		Department:       "Cardiology",
		Designation:      "Chief Cardiologist",
		EmploymentStatus: "active",
	}
	db.Create(&staffProfile)

	// Initiate onboarding
	onbResp := suite.Request("POST", "/api/v1/hr/onboarding/initiate", map[string]interface{}{
		"staff_id": staffProfile.ID,
	}, hrToken)
	assert.Equal(suite.T(), 200, onbResp.StatusCode)

	var onbRecord hr.OnboardingRecord
	json.NewDecoder(onbResp.Body).Decode(&onbRecord)
	assert.Equal(suite.T(), "pending_docs", onbRecord.Status)

	// Update documents and status to completed
	recID := fmt.Sprintf("%d", onbRecord.ID)
	verifyResp := suite.Request("PUT", "/api/v1/hr/onboarding/"+recID+"/documents", map[string]interface{}{
		"doc_type": "certificate",
		"status":   "verified",
	}, hrToken)
	assert.Equal(suite.T(), 200, verifyResp.StatusCode)

	taskResp := suite.Request("PUT", "/api/v1/hr/onboarding/"+recID+"/tasks", map[string]interface{}{
		"task_type":   "system_access_granted",
		"is_complete": true,
	}, hrToken)
	assert.Equal(suite.T(), 200, taskResp.StatusCode)

	statusResp := suite.Request("PUT", "/api/v1/hr/onboarding/"+recID+"/status", map[string]interface{}{
		"status": "completed",
	}, hrToken)
	assert.Equal(suite.T(), 200, statusResp.StatusCode)

	// Verify onboarding is complete in database
	db.First(&onbRecord, onbRecord.ID)
	assert.Equal(suite.T(), "completed", onbRecord.Status)

	// 4. Verify HR Dashboard Overview shows correct numbers
	hrOverviewResp := suite.Request("GET", "/api/v1/hr/dashboard/overview", nil, hrToken)
	assert.Equal(suite.T(), 200, hrOverviewResp.StatusCode)
	
	var hrData struct {
		Data map[string]interface{} `json:"data"`
	}
	json.NewDecoder(hrOverviewResp.Body).Decode(&hrData)
	assert.Equal(suite.T(), float64(2), hrData.Data["total_staff"], "Total staff should be 2 (HR Manager + 1 Doctor)")
	assert.Equal(suite.T(), float64(1), hrData.Data["active_doctors"], "Active doctors should be 1")

	// 5. Patient registers and fetches consolidated summary dashboard BFF
	patResp := suite.Request("POST", "/api/v1/auth/register", map[string]interface{}{
		"email":     "patient_new@test.com",
		"password":  "password123",
		"full_name": "New Patient",
		"role":      "patient",
	}, "")
	assert.Equal(suite.T(), 201, patResp.StatusCode)

	var patLogin struct {
		Data map[string]interface{} `json:"data"`
	}
	json.NewDecoder(patResp.Body).Decode(&patLogin)
	patToken := patLogin.Data["access_token"].(string)

	// Fetch Patient summary BFF
	summaryResp := suite.Request("GET", "/api/v1/patients/dashboard/summary", nil, patToken)
	assert.Equal(suite.T(), 200, summaryResp.StatusCode)

	var summaryData struct {
		Data map[string]interface{} `json:"data"`
	}
	json.NewDecoder(summaryResp.Body).Decode(&summaryData)

	// Assert the new onboarded doctor is visible in the online doctors list
	onlineDocs := summaryData.Data["online_doctors"].([]interface{})
	assert.Len(suite.T(), onlineDocs, 1)
	assert.Equal(suite.T(), float64(docUserID), onlineDocs[0].(map[string]interface{})["user_id"])
}

func TestPatientDoctorHRIntegrationSuite(t *testing.T) {
	suite.Run(t, new(PatientDoctorHRIntegrationTestSuite))
}

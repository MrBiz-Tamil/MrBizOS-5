package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/communication"
	"drayya-core/internal/domain/emr"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/hr"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/loyalty"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/domain/store"
	"drayya-core/internal/services/notification"
	"drayya-core/internal/server"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type WorkflowTestSuite struct {
	suite.Suite
	App *fiber.App
	DB  *gorm.DB
}

func (suite *WorkflowTestSuite) SetupSuite() {
	os.Setenv("USE_SQLITE", "true")
	os.Setenv("DB_URL", "file:memdb_workflow?mode=memory&cache=shared")
	os.Setenv("JWT_SECRET", "test_secret_key")

	db, err := gorm.Open(sqlite.Open("file:memdb_workflow?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		fmt.Printf("DATABASE CONNECTION ERROR: %v\n", err)
		panic("failed to connect database: " + err.Error())
	}
	suite.DB = db
	database.DB = db

	// AutoMigrate all models
	identity.AutoMigrate(db)
	finance.AutoMigrate(db)
	appointment.AutoMigrate(db)
	pharmacy.AutoMigrate(db)
	store.AutoMigrate(db)
	loyalty.AutoMigrate(db)
	hr.AutoMigrate(db)
	emr.AutoMigrate(db)
	communication.AutoMigrate(db)
	notification.AutoMigrateOutbox(db)

	suite.App = server.New()
}

func (suite *WorkflowTestSuite) TearDownSuite() {
	notification.StopWorker()
	os.Remove("Dr.Ayya_test.db")
}

func (suite *WorkflowTestSuite) Request(method, path string, body interface{}, token string) *http.Response {
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

	resp, _ := suite.App.Test(req)
	return resp
}

var (
	patientToken string
	doctorToken  string
	patientID    uint
	doctorID     uint
	apptID       uint
)

func (suite *WorkflowTestSuite) Test01_RegisterUsers() {
	// 1. Register Patient
	resp := suite.Request("POST", "/api/v1/auth/register", map[string]interface{}{
		"email":    "patient@test.com",
		"password": "password123",
		"full_name": "Test Patient",
		"role":     "patient",
	}, "")
	assert.Equal(suite.T(), 201, resp.StatusCode)

	var loginResp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		fmt.Printf("Decode Error: %v\n", err)
	}

	if loginResp.Data != nil {
		if token, ok := loginResp.Data["access_token"].(string); ok {
			patientToken = token
		} else if token, ok := loginResp.Data["token"].(string); ok {
			patientToken = token
		}

		if data, ok := loginResp.Data["user"].(map[string]interface{}); ok {
			patientID = uint(data["id"].(float64))
			suite.DB.FirstOrCreate(&identity.PatientProfile{UserID: patientID})
		}
	}
	assert.NotEmpty(suite.T(), patientToken, "Patient token should not be empty")

	// 2. Register Doctor
	resp = suite.Request("POST", "/api/v1/auth/register", map[string]interface{}{
		"email":    "doctor@test.com",
		"password": "password123",
		"full_name": "Dr. Test",
		"role":     "doctor",
	}, "")
	assert.Equal(suite.T(), 201, resp.StatusCode)

	json.NewDecoder(resp.Body).Decode(&loginResp)
	if loginResp.Data != nil {
		if token, ok := loginResp.Data["access_token"].(string); ok {
			doctorToken = token
		} else if token, ok := loginResp.Data["token"].(string); ok {
			doctorToken = token
		}

		if data, ok := loginResp.Data["user"].(map[string]interface{}); ok {
			doctorID = uint(data["id"].(float64))
			suite.DB.Model(&identity.User{}).Where("id = ?", doctorID).Update("is_verified", true)
			suite.DB.Model(&identity.DoctorProfile{}).Where("user_id = ?", doctorID).Update("is_verified", true)
		}
		if data, ok := loginResp.Data["user"].(map[string]interface{}); ok {
			doctorID = uint(data["id"].(float64))
			suite.DB.FirstOrCreate(&identity.DoctorProfile{UserID: doctorID})
		}
	}
	assert.NotEmpty(suite.T(), doctorToken, "Doctor token should not be empty")
}

func (suite *WorkflowTestSuite) Test02_WalletTopUp() {
	// 1. Top up wallet
	resp := suite.Request("POST", "/api/v1/finance/topup", map[string]interface{}{
		"amount": 500.0, // 500 INR
		"txn_id": "TXN_12345",
	}, patientToken)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	// 2. Verify balance
	resp = suite.Request("GET", "/api/v1/finance/balance", nil, patientToken)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	var balanceResp struct {
		Data map[string]interface{} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&balanceResp)
	balance := balanceResp.Data["balance"].(float64)
	assert.Equal(suite.T(), 500.0, balance)
}

func (suite *WorkflowTestSuite) Test03_CreateAppointment() {
	if doctorID == 0 {
		suite.T().Skip("Doctor ID missing")
	}

	// Create Appointment
	resp := suite.Request("POST", "/api/v1/appointments", map[string]interface{}{
		"doctor_id":         doctorID,
		"appointment_type":  "consultation",
		"consultation_mode": "video",
		"notes":             "I have a headache",
	}, patientToken)
	if resp.StatusCode != 201 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		fmt.Printf("Create Appointment Failed: %s\n", buf.String())
	}
	assert.Equal(suite.T(), 201, resp.StatusCode)

	var env struct {
		Data struct {
			Appointment appointment.Appointment `json:"appointment"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&env)
	apptID = env.Data.Appointment.ID
	assert.NotZero(suite.T(), apptID)
}

func (suite *WorkflowTestSuite) Test05_CompleteConsultationAndPrescription() {
	if apptID == 0 {
		suite.T().Skip("Appointment ID missing")
	}

	// 1. Start Consultation (as doctor)
	resp := suite.Request("POST", "/api/v1/appointments/"+fmt.Sprintf("%d", apptID)+"/start", nil, doctorToken)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	// 2. Add Prescription
	resp = suite.Request("POST", "/api/v1/emr/prescriptions", map[string]interface{}{
		"appointment_id": apptID,
		"patient_id":     patientID,
		"medicines": []map[string]interface{}{
			{
				"name":     "Paracetamol",
				"dosage":   "500mg",
				"duration": "3 days",
				"notes":    "After food",
			},
		},
		"notes": "Rest well",
	}, doctorToken)
	assert.Equal(suite.T(), 201, resp.StatusCode)

	// 3. Start Appointment
	resp = suite.Request("POST", "/api/v1/appointments/"+fmt.Sprintf("%d", apptID)+"/start", nil, doctorToken)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	// 4. Complete Appointment
	resp = suite.Request("POST", "/api/v1/appointments/"+fmt.Sprintf("%d", apptID)+"/complete", nil, doctorToken)
	if resp.StatusCode != 200 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		fmt.Printf("Complete Appointment Failed: %s\n", buf.String())
	}
	assert.Equal(suite.T(), 200, resp.StatusCode)
}

func (suite *WorkflowTestSuite) Test06_VerifyPatientTimeline() {
	resp := suite.Request("GET", "/api/v1/emr/medical-records/timeline/"+fmt.Sprintf("%d", patientID), nil, patientToken)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	var env struct {
		Data []map[string]interface{} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&env)
	assert.NotEmpty(suite.T(), env.Data)
}

func (suite *WorkflowTestSuite) Test08_LiveKitAndSOS() {
	// 1. Get LiveKit Token
	resp := suite.Request("POST", "/api/v1/livekit/token", map[string]interface{}{
		"room_name": fmt.Sprintf("appointment_%d", apptID),
		"user_name": "Patient User",
		"role":      "patient",
	}, patientToken)
	// LiveKit might be disabled in test environment, so we accept 200 or 503
	assert.True(suite.T(), resp.StatusCode == 200 || resp.StatusCode == 503, "LiveKit token should return 200 or 503")

	// 2. SOS Appointment
	resp = suite.Request("POST", "/api/v1/appointments", map[string]interface{}{
		"appointment_type":  "emergency",
		"consultation_mode": "chat",
		"notes":             "SOS: Chest pain",
		"is_priority":       true,
	}, patientToken)
	if resp.StatusCode != 201 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		fmt.Printf("SOS Appointment Failed: %s\n", buf.String())
	}
	assert.Equal(suite.T(), 201, resp.StatusCode)

	var env struct {
		Data struct {
			Appointment struct {
				ID uint `json:"id"`
			} `json:"appointment"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&env)
	sosApptID := env.Data.Appointment.ID
	assert.NotZero(suite.T(), sosApptID, "SOS Appointment ID should not be zero")
	fmt.Printf("DEBUG: SOS Appt ID: %d, Doctor ID: %d\n", sosApptID, doctorID)

	// 2.5 Assign Doctor and Confirm Status (since SOS needs to be accepted/assigned before call)
	res := suite.DB.Exec("UPDATE appointments SET doctor_id = ?, status = ? WHERE id = ?", doctorID, "confirmed", sosApptID)
	if res.Error != nil {
		fmt.Printf("DEBUG: Update Error: %v\n", res.Error)
	}
	fmt.Printf("DEBUG: Rows affected: %d\n", res.RowsAffected)

	// 3. Start SOS Call
	resp = suite.Request("POST", "/api/v1/livekit/call/start", map[string]interface{}{
		"appointment_id": sosApptID,
		"doctor_id":      doctorID,
		"patient_id":     patientID,
		"call_type":      "emergency",
	}, doctorToken)
	if resp.StatusCode != 200 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		fmt.Printf("Start Call Failed: %s\n", buf.String())
	}
	assert.Equal(suite.T(), 200, resp.StatusCode)
}

func (suite *WorkflowTestSuite) Test10_PharmacyWorkflow() {
	// 1. Seed Pharmacy (Hub)
	hub := pharmacy.Pharmacy{
		Name:               "Test Central Hub",
		City:               "Bangalore",
		IsCentralWarehouse: true,
		IsActive:           true,
	}
	suite.DB.Create(&hub)

	// 2. Seed Store Product
	product := store.WellnessProduct{
		Name:         "Organic Honey",
		Category:     "Wellness",
		SellingPrice: 50000, // 500.00
		IsActive:     true,
	}
	suite.DB.Create(&product)
	assert.NotZero(suite.T(), product.ID, "Product ID should not be zero")

	// 3. Seed Inventory at Hub
	inv := store.WellnessInventory{
		ProductID:     product.ID,
		PharmacyID:    hub.ID,
		StockQuantity: 100,
	}
	suite.DB.Create(&inv)

	// 4. Create Order
	resp := suite.Request("POST", "/api/v1/store/orders", map[string]interface{}{
		"delivery_address": "123 Test St, Bangalore",
		"pharmacy_id":      hub.ID,
		"items": []map[string]interface{}{
			{
				"product_id": product.ID,
				"quantity":   2,
			},
		},
	}, patientToken)

	if resp.StatusCode != 201 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		fmt.Printf("Store Order Failed: %s\n", buf.String())
	}
	assert.Equal(suite.T(), 201, resp.StatusCode)
}

func (suite *WorkflowTestSuite) Test11_Dispatch() {
	// 1. List Orders
	resp := suite.Request("GET", "/api/v1/store/orders?user_id="+fmt.Sprintf("%d", patientID), nil, patientToken)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	var env struct {
		Data []map[string]interface{} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&env)
	assert.NotEmpty(suite.T(), env.Data)

	lastOrder := env.Data[0]
	assert.Contains(suite.T(), []string{"pending", "processing"}, lastOrder["status"])
}

func TestWorkflowSuite(t *testing.T) {
	suite.Run(t, new(WorkflowTestSuite))
}

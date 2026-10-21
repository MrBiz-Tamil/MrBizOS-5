package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/emr"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/handlers"
	"drayya-core/internal/server"
	"drayya-core/internal/tokens"
	"drayya-core/internal/ws"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PharmacyFullTestSuite struct {
	suite.Suite
	App *fiber.App
	DB  *gorm.DB
}

func (suite *PharmacyFullTestSuite) SetupSuite() {
	os.Setenv("USE_SQLITE", "true")
	os.Setenv("DB_URL", "file:memdb_pharmacy_full?mode=memory&cache=shared")
	os.Setenv("JWT_SECRET", "test_secret_key")
	database.Connect()
	suite.DB = database.DB

	// Migrate domains
	identity.AutoMigrate(suite.DB)
	pharmacy.AutoMigrate(suite.DB)
	emr.AutoMigrate(suite.DB)
	finance.AutoMigrate(suite.DB)

	handlers.Hub = ws.NewHub(nil)
	suite.App = server.New()
}

func (suite *PharmacyFullTestSuite) TestPharmacyEndToEnd() {
	// 1. Setup Data
	adminToken, _ := tokens.CreateAccessToken("100", "admin", nil, nil)

	// Create User with matching ID to avoid 401 logic if it checks DB
	adminUser := identity.User{ID: 100, FullName: "Admin", Role: "admin", Email: "admin@Dr.Ayya.test"}
	suite.DB.Create(&adminUser)

	// Create Pharmacy
	p := pharmacy.Pharmacy{
		Name:               "Testing Main Pharmacy",
		Address:            "456 Test Blvd",
		City:               "Bangalore",
		IsActive:           true,
		IsCentralWarehouse: true,
	}
	suite.DB.Create(&p)

	// Create Supplier (Necessary for restock trigger)
	supplier := pharmacy.Supplier{
		Name:         "Test Supplier",
		ContactEmail: "supplier@test.com",
		IsActive:     true,
	}
	suite.DB.Create(&supplier)

	medicine := pharmacy.Medicine{
		Name:     "Amoxicillin 500mg",
		Category: "Antibiotics",
	}
	suite.DB.Create(&medicine)

	inventory := pharmacy.PharmacyInventory{
		PharmacyID:    p.ID,
		MedicineID:    medicine.ID,
		StockQuantity: 50,
		ReorderLevel:  10,
		IsActive:      true,
	}
	suite.DB.Create(&inventory)

	// 2. Test Conversion & Reservation
	// Create a mock prescription
	patient := identity.User{FullName: "John Doe", Role: "patient", Email: "john@pharmacy.test"}
	suite.DB.Create(&patient)

	prescription := emr.Prescription{
		PatientID: patient.ID,
		Medicines: datatypes.JSON([]byte(`[{"name": "Amoxicillin 500mg", "dosage": "1-0-1", "duration": "5 days"}]`)),
		Status:    "verified",
	}
	suite.DB.Create(&prescription)

	// Convert to Order
	resp := suite.Request("POST", "/api/pharmacy/prescriptions/convert", map[string]interface{}{
		"prescription_id":  prescription.ID,
		"pharmacy_id":      p.ID,
		"shipping_address": "789 Delivery Rd",
		"phone":            "9876543210",
	}, adminToken)

	if !assert.Equal(suite.T(), 201, resp.StatusCode) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		fmt.Printf("DEBUG: Convert Response: %s\n", buf.String())
		suite.T().FailNow()
	}

	var convertRes struct {
		Data pharmacy.PharmacyOrder `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&convertRes)
	order := convertRes.Data
	assert.NotZero(suite.T(), order.ID)
	assert.Equal(suite.T(), "pending_rx_verification", order.Status)

	// Set order amount and medicines (pharmacist verification simulation)
	order.Amount = 50000 // 500 INR
	order.Items = datatypes.JSON([]byte(`[{"medicine_id": 1, "qty": 2}]`))
	suite.DB.Save(&order)

	var updatedInv pharmacy.PharmacyInventory
	suite.DB.First(&updatedInv, inventory.ID)

	// 3. Test Payment & Ledger Splits
	suite.DB.Exec(`
		INSERT INTO ledger_account_v2 (id, name, type, currency, balance, created_at, updated_at)
		VALUES ('123e4567-e89b-12d3-a456-426614174000', 'PatientWallet:101', 'liability', 'INR', 100000, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`)

	resp = suite.Request("POST", fmt.Sprintf("/api/pharmacy/orders/%d/pay", order.ID), nil, adminToken)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	// Verify Ledger Entries (3 entries: 1 debit for patient, 2 credits for splits)
	var entries []finance.LedgerEntry
	suite.DB.Find(&entries)
	assert.GreaterOrEqual(suite.T(), len(entries), 3, "There should be at least 3 ledger entries for the payment split")

	// 4. Test Stock Disposal
	initialStock := updatedInv.StockQuantity
	resp = suite.Request("POST", "/api/pharmacy/inventory/dispose", map[string]interface{}{
		"inventory_id": updatedInv.ID,
		"quantity":     5,
		"reason":       "Damaged seal",
	}, adminToken)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	suite.DB.First(&updatedInv, inventory.ID)
	assert.Equal(suite.T(), initialStock-5, updatedInv.StockQuantity)

	// 5. Test Restock Trigger
	// Drop stock below threshold
	suite.DB.Model(&updatedInv).Update("stock_quantity", 5)

	resp = suite.Request("POST", "/api/pharmacy/admin/trigger-restock", nil, adminToken)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	var restockRes struct {
		Data map[string]interface{} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&restockRes)
	assert.NotNil(suite.T(), restockRes.Data["count"])

	// 6. Test Financials Report
	resp = suite.Request("GET", fmt.Sprintf("/api/pharmacy/financials?pharmacy_id=%d", p.ID), nil, adminToken)
	assert.Equal(suite.T(), 200, resp.StatusCode)

	var financialsRes struct {
		Data map[string]interface{} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&financialsRes)
	financials := financialsRes.Data
	assert.NotNil(suite.T(), financials["total_sales"])
	assert.NotNil(suite.T(), financials["platform_royalty"])
}

func (suite *PharmacyFullTestSuite) Request(method, path string, body interface{}, token string) *http.Response {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, _ := suite.App.Test(req, -1)
	return resp
}

func TestPharmacyFullSuite(t *testing.T) {
	suite.Run(t, new(PharmacyFullTestSuite))
}

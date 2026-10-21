package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/hr"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/domain/store"
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
	"gorm.io/gorm"
)

type ArchitectureTestSuite struct {
	suite.Suite
	App *fiber.App
	DB  *gorm.DB
}

func (suite *ArchitectureTestSuite) SetupSuite() {
	os.Setenv("USE_SQLITE", "true")
	os.Setenv("DB_URL", "file:memdb_arch?mode=memory&cache=shared")
	os.Setenv("JWT_SECRET", "test_secret_key")
	database.Connect()
	suite.DB = database.DB

	identity.AutoMigrate(suite.DB)
	finance.AutoMigrate(suite.DB)
	hr.AutoMigrate(suite.DB)
	pharmacy.AutoMigrate(suite.DB)
	store.AutoMigrate(suite.DB)
	finance.SeedTaxRules(suite.DB)

	handlers.Hub = ws.NewHub(nil)
	suite.App = server.New()
}

func (suite *ArchitectureTestSuite) TestHubFallbackRouting() {
	// 1. Create Hub and Spoke
	hub := pharmacy.Pharmacy{
		ID:                 1,
		Name:               "Central Hub Bangalore",
		City:               "Bangalore",
		IsCentralWarehouse: true,
		IsActive:           true,
	}
	suite.DB.Create(&hub)

	spoke := pharmacy.Pharmacy{
		ID:                 2,
		Name:               "Spoke Chennai",
		City:               "Chennai",
		IsCentralWarehouse: false,
		ParentHubID:        &hub.ID,
		IsActive:           true,
	}
	suite.DB.Create(&spoke)

	// 2. Create Product and Stock only at Hub
	product := store.WellnessProduct{
		Name:         "Organic Vitamin C",
		SellingPrice: 1000,
		IsActive:     true,
	}
	suite.DB.Create(&product)

	hubInv := store.WellnessInventory{
		ProductID:     product.ID,
		PharmacyID:    hub.ID,
		StockQuantity: 10,
	}
	suite.DB.Create(&hubInv)

	// Verify Spoke has NO stock
	var spokeInv store.WellnessInventory
	err := suite.DB.Where("product_id = ? AND pharmacy_id = ?", product.ID, spoke.ID).First(&spokeInv).Error
	assert.Error(suite.T(), err, "Spoke should not have inventory record")

	// 3. Create Order via Spoke (ID 2)
	patient := identity.User{
		ID:            2,
		FullName:      "Chennai Patient",
		Role:          "patient",
		IsActive:      true,
	}
	suite.DB.Create(&patient)
	patientToken, _ := tokens.CreateAccessToken("2", "patient", nil, nil)

	// Top up patient's wallet first
	topupResp := suite.Request("POST", "/api/v1/finance/topup", map[string]interface{}{
		"amount": 100.0, // 100 INR = 10000 paise
		"txn_id": "TXN_CHENN_01",
	}, patientToken)
	assert.Equal(suite.T(), 200, topupResp.StatusCode)

	resp := suite.Request("POST", "/api/v1/store/orders", map[string]interface{}{
		"delivery_address": "Chennai Main Road",
		"pharmacy_id":      spoke.ID,
		"items": []map[string]interface{}{
			{
				"product_id": product.ID,
				"quantity":   2,
			},
		},
	}, patientToken)

	assert.Equal(suite.T(), 201, resp.StatusCode)

	var res struct {
		Data store.WellnessOrder `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&res)
	order := res.Data

	// 4. Verify Routing: Order should be fulfilled by Hub (ID 1)
	assert.Equal(suite.T(), hub.ID, order.PharmacyID, "Order should be routed to Central Hub")
	
	// 5. Pay from wallet to trigger stock deduction
	payResp := suite.Request("POST", fmt.Sprintf("/api/v1/store/orders/%d/pay/wallet", order.ID), nil, patientToken)
	assert.Equal(suite.T(), 200, payResp.StatusCode)

	suite.DB.First(&hubInv, hubInv.ID)
	assert.Equal(suite.T(), 8, hubInv.StockQuantity, "Stock should be deducted from Hub (10 - 2 = 8)")
}

func (suite *ArchitectureTestSuite) Request(method, path string, body interface{}, token string) *http.Response {
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

func TestArchitectureSuite(t *testing.T) {
	suite.Run(t, new(ArchitectureTestSuite))
}

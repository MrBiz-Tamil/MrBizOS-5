package pharmacy_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/money"
	"drayya-core/internal/tests/common"

	"github.com/stretchr/testify/assert"
)

func TestGetInventory(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	// Seed Data
	med := pharmacy.Medicine{
		Name:         "Paracetamol",
		Manufacturer: "GSK",
		Prices:       money.RegionalPricing{"INR": 1000},
	}
	db.Create(&med)

	db.Create(&pharmacy.PharmacyInventory{
		PharmacyID:    1,
		MedicineID:    med.ID,
		StockQuantity: 100,
		ReorderLevel:  20,
		ExpiryDate:    time.Now().AddDate(0, 6, 0),
		IsActive:      true,
	})

	// Route requires PharmacyManagerOnly — use a pharmacist token
	pharmacist := common.CreateTestUser(db, "pharmacist")
	token := common.GenerateTestToken(pharmacist)

	req := httptest.NewRequest("GET", "/api/v1/pharmacy/inventory", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)

	// response.JSON wraps in {"success":true,"data":{"data":[...],"page":1,...}}
	var envelope map[string]interface{}
	json.Unmarshal(body, &envelope)
	payload := envelope["data"].(map[string]interface{})
	items := payload["data"].([]interface{})

	assert.Len(t, items, 1)
	item := items[0].(map[string]interface{})
	assert.Equal(t, "Paracetamol", item["Medicine"].(map[string]interface{})["name"])
}

func TestGetStockAlerts(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	// Seed Low Stock Item
	med := pharmacy.Medicine{Name: "LowStockMed", Prices: money.RegionalPricing{"INR": 500}}
	db.Create(&med)
	db.Create(&pharmacy.PharmacyInventory{
		MedicineID:    med.ID,
		StockQuantity: 5,
		ReorderLevel:  10,
		IsActive:      true,
		ExpiryDate:    time.Now().AddDate(1, 0, 0),
	})

	// Route requires PharmacyManagerOnly
	pharmacist := common.CreateTestUser(db, "pharmacist")
	token := common.GenerateTestToken(pharmacist)

	req := httptest.NewRequest("GET", "/api/v1/pharmacy/alerts", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var envelope map[string]interface{}
	json.Unmarshal(body, &envelope)
	result := envelope["data"].(map[string]interface{})

	lowStock := result["low_stock"].([]interface{})
	assert.Len(t, lowStock, 1)
}

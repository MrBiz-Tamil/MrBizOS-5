package handlers_test

import (
	"encoding/json"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/tests/common"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestShippingHandler_PublicTrack(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	// 1. Create a Shipment with 3PL data
	shipment := pharmacy.Shipment{
		TrackingNumber: "MH-3PL-123",
		OrderID:        1,
		OrderType:      "pharmacy",
		ProviderName:   "BlueDart",
		Status:         "in_transit",
		Destination:    "110001",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	db.Create(&shipment)

	// 2. Test Public Track Endpoint
	req := httptest.NewRequest("GET", "/api/v1/retail/track/MH-3PL-123", nil)
	resp, _ := app.Test(req)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Data pharmacy.Shipment `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, "MH-3PL-123", result.Data.TrackingNumber)
	assert.Equal(t, "BlueDart", result.Data.ProviderName)
	assert.Equal(t, "in_transit", result.Data.Status)
}

func TestShippingHandler_AdminDashboard(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	// 1. Create dummy data
	db.Create(&pharmacy.Shipment{
		TrackingNumber: "AWB1",
		ProviderName:   "Shiprocket",
		Status:         "in_transit",
		CreatedAt:      time.Now(),
	})

	// 2. Test Admin Courier Dashboard
	req := httptest.NewRequest("GET", "/api/v1/admin/dashboard/courier", nil)
	// We need a token for admin access
	admin := common.CreateTestUser(db, "admin")
	token := common.GenerateTestToken(admin)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, _ := app.Test(req)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	activeShipments := result["active_shipments"].([]interface{})
	assert.NotEmpty(t, activeShipments)
	
	first := activeShipments[0].(map[string]interface{})
	assert.Equal(t, "AWB1", first["id"])
	assert.Equal(t, "Shiprocket", first["provider_name"])
}

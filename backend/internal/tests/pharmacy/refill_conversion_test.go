package pharmacy_test

import (
	"encoding/json"
	"fmt"
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

func TestConvertRefillToOrder_StockCheckAndTraceability(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	// Seed medicine
	med := pharmacy.Medicine{
		Name:         "Ashwagandha Churna",
		Manufacturer: "AYUSH Labs",
		Prices:       money.RegionalPricing{"INR": 15000}, // ₹150.00
		IsActive:     true,
	}
	db.Create(&med)

	// Seed pharmacy
	pharm := pharmacy.Pharmacy{
		Name:     "Central Siddha Pharmacy",
		IsActive: true,
	}
	db.Create(&pharm)

	// Seed active inventory batch (expiry 6 months from now)
	expDate := time.Now().AddDate(0, 6, 0).Truncate(24 * time.Hour)
	batch := pharmacy.PharmacyInventory{
		PharmacyID:    pharm.ID,
		MedicineID:    med.ID,
		BatchNumber:   "ASH-B01",
		StockQuantity: 10,
		IsActive:      true,
		ExpiryDate:    expDate,
	}
	db.Create(&batch)

	// Seed patient
	patient := common.CreateTestUser(db, "patient")
	token := common.GenerateTestToken(patient)

	// Create parent order that is a refill source
	itemsJSON := fmt.Sprintf(`[{"medicine_id":%d,"quantity":2,"unit_price":15000}]`, med.ID)
	parentOrder := pharmacy.PharmacyOrder{
		OrderNumber:     "ORD-PARENT-123",
		UserID:          patient.ID,
		PharmacyID:      pharm.ID,
		Items:           []byte(itemsJSON),
		Amount:          30000,
		Currency:        "INR",
		Status:          "delivered",
		PaymentStatus:     "paid",
		PaymentMethod:   "online",
		DeliveryMode:    pharmacy.ModeStandard,
		ShippingAddress: "123 Test Street",
		DeliveryPhone:   "9999999999",
	}
	db.Create(&parentOrder)

	// Route: POST /api/v1/pharmacy/refills/:id/convert
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/pharmacy/refills/%d/convert", parentOrder.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 201 {
		t.Fatalf("Status is %d, body: %s", resp.StatusCode, string(body))
	}
	var envelope map[string]interface{}
	json.Unmarshal(body, &envelope)
	data := envelope["data"].(map[string]interface{})

	assert.Equal(t, float64(parentOrder.ID), data["parent_order_id"])

	// Verify stock was decremented: 10 - 2 = 8
	var updatedBatch pharmacy.PharmacyInventory
	db.First(&updatedBatch, batch.ID)
	assert.Equal(t, 8, updatedBatch.StockQuantity)

	// Verify that a new refill order was created in database
	var newOrder pharmacy.PharmacyOrder
	err = db.Where("parent_order_id = ? AND order_number LIKE ?", parentOrder.ID, "RFL-%").First(&newOrder).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(30000), newOrder.Amount)
	assert.Equal(t, "pending", newOrder.Status)

	// Verify that corresponding PharmacyOrderItem was created for recall tracking
	var orderItem pharmacy.PharmacyOrderItem
	err = db.Where("order_id = ?", newOrder.ID).First(&orderItem).Error
	assert.NoError(t, err)
	assert.Equal(t, med.ID, orderItem.MedicineID)
	assert.Equal(t, 2, orderItem.Quantity)
	assert.Equal(t, "ASH-B01", orderItem.BatchNumber)
}

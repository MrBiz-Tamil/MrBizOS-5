package shipping_test

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/shipping"
	"drayya-core/internal/tests/common"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDispatchDock_DispatchOrder_Local(t *testing.T) {
	common.SetupApp()
	db := database.DB

	// 1. Setup Pharmacy
	p := pharmacy.Pharmacy{Name: "Test Pharmacy", IsActive: true}
	db.Create(&p)

	// 2. Setup Available Courier
	courier := pharmacy.CourierProfile{
		UserID:      100,
		IsAvailable: true,
		Rating:      5.0,
	}
	db.Create(&courier)

	// 3. Setup Order
	order := pharmacy.PharmacyOrder{
		OrderNumber:  "ORD-QUICK-1",
		PharmacyID:   p.ID,
		Status:       "paid",
		DeliveryMode: pharmacy.ModeQuick,
	}
	db.Create(&order)

	// 4. Test Dispatch
	err := shipping.GlobalDispatchDock.DispatchOrder(order.ID)
	assert.NoError(t, err)

	// 5. Verify Results
	var updatedOrder pharmacy.PharmacyOrder
	db.First(&updatedOrder, order.ID)
	// Status could be 'assigned' (if auto-assign worked) or 'dispatching' (if broadcast fallback)
	assert.Contains(t, []string{"assigned", "dispatching"}, updatedOrder.Status)

	if updatedOrder.Status == "assigned" {
		var delivery pharmacy.CourierDelivery
		err = db.Where("order_id = ?", order.ID).First(&delivery).Error
		assert.NoError(t, err)
		assert.Equal(t, courier.UserID, delivery.CourierID)
		assert.Equal(t, "assigned", delivery.Status)
	}
}

func TestDispatchDock_DispatchOrder_Standard(t *testing.T) {
	common.SetupApp()
	db := database.DB

	// 1. Setup Pharmacy
	p := pharmacy.Pharmacy{Name: "Test Hub", Address: "560001", IsActive: true}
	db.Create(&p)

	// 2. Setup Order
	order := pharmacy.PharmacyOrder{
		OrderNumber:  "ORD-STD-1",
		PharmacyID:   p.ID,
		Status:       "paid",
		DeliveryMode: pharmacy.ModeStandard,
	}
	db.Create(&order)

	// 3. Test Dispatch
	err := shipping.GlobalDispatchDock.DispatchOrder(order.ID)
	assert.NoError(t, err)

	// 4. Verify Results
	var updatedOrder pharmacy.PharmacyOrder
	db.First(&updatedOrder, order.ID)
	assert.Equal(t, "shipped", updatedOrder.Status)
	assert.NotEmpty(t, updatedOrder.TrackingID)
}


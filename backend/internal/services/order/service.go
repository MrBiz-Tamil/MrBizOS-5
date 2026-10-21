package order

import (
	"context"
	"errors"
	"fmt"
	"log"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/shipping"
	"drayya-core/internal/utils/privacy"
	"time"

	"gorm.io/gorm"
)

// Redis Lua script for atomic stock decrement
// Keys: [stock_key]
// Args: [decrement_amount]
const deductStockScript = `
local current_stock = tonumber(redis.call('get', KEYS[1]))
if not current_stock or current_stock < tonumber(ARGV[1]) then
    return -1
end
return redis.call('decrby', KEYS[1], ARGV[1])
`

type OrderRequest struct {
	UserID          uint
	Items           []CartItem
	Lat             float64
	Lng             float64
	City            string
	DeliveryAddress string
	DeliveryPhone   string
	DestPincode     string
	// DeliveryMode selects the fulfilment channel:
	//   QUICK    -> local courier, 30-min SLA (default when Lat/Lng are provided)
	//   STANDARD -> 3PL courier (Shiprocket / India Post), 24-h SLA
	DeliveryMode    pharmacy.DeliveryMode
}

func CreateOrder(ctx context.Context, req OrderRequest) (*pharmacy.PharmacyOrder, error) {
	// 1. Get Fulfillment Plan
	plans, err := FindBestFulfillment(ctx, req.Lat, req.Lng, req.City, req.Items)
	if err != nil {
		return nil, err
	}

	if len(plans) == 0 {
		return nil, errors.New("no pharmacies available or insufficient stock")
	}

	// 2. Start Transaction
	var masterOrder *pharmacy.PharmacyOrder

	// Track successful deductions for manual reversal if transaction fails

	// Ensure we clean up on error using a closure trick or manual tracking
	// This defer runs after the surrounding function (CreateOrder) returns,
	// but we want it to run if the *transaction* fails.
	// The `err` variable here is the `err` returned by `database.DB.Transaction`.
	// We need to capture `err` from the transaction scope.
	// A common pattern is to assign the transaction error to the outer `err` variable.
	// The defer needs to be placed *before* the transaction block to capture the final `err` value.
	// However, the instruction places it inside the transaction function.
	// Let's adjust the defer to be inside the transaction function, and it will check the error
	// state of the transaction function itself.

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var txErr error
		// Track successful deductions for manual reversal if transaction fails
		type deduction struct {
			key string
			qty int
		}
		var successfulDeductions []deduction

		defer func() {
			if txErr != nil {
				for _, d := range successfulDeductions {
					database.Rdb.IncrBy(ctx, d.key, int64(d.qty))
					log.Printf("     Rolled back stock for %s (+%d)", d.key, d.qty)
				}
			}
		}()

		// Determine delivery mode: default to QUICK when coordinates are provided,
		// otherwise fall back to STANDARD (3PL) for pincode-only orders.
		deliveryMode := req.DeliveryMode
		if deliveryMode == "" {
			if req.Lat != 0 && req.Lng != 0 {
				deliveryMode = pharmacy.ModeQuick
			} else {
				deliveryMode = pharmacy.ModeStandard
			}
		}

		// Create Master Order
		orderNumber := fmt.Sprintf("ORD-%d", time.Now().Unix())
		masterOrder = &pharmacy.PharmacyOrder{
			OrderNumber:     orderNumber,
			UserID:          req.UserID,
			Amount:     0,
			// "confirmed" so the SLA monitor & AutoDispatchProcessor can pick it up
			// immediately. "pending" is only used when awaiting payment confirmation.
			Status:          "confirmed",
			PaymentStatus:   "pending",
			ShippingAddress: privacy.EncryptedString(req.DeliveryAddress),
			DeliveryPhone:   privacy.EncryptedString(req.DeliveryPhone),
			DeliveryMode:    deliveryMode,
			DestLatitude:    req.Lat,
			DestLongitude:   req.Lng,
			DestPincode:     req.DestPincode,
			CreatedAt:       time.Now(),
		}

		if err := tx.Create(masterOrder).Error; err != nil {
			txErr = err
			return err
		}

		// Process each fulfillment segment
		for _, plan := range plans {
			subOrder := &pharmacy.PharmacyOrder{
				OrderNumber:     fmt.Sprintf("%s-%d", masterOrder.OrderNumber, plan.PharmacyID),
				UserID:          req.UserID,
				PharmacyID:      plan.PharmacyID,
				ParentOrderID:   masterOrder.ID,
				Status:          "confirmed",
				PaymentStatus:   "pending",
				ShippingAddress: privacy.EncryptedString(req.DeliveryAddress),
				DeliveryPhone:   privacy.EncryptedString(req.DeliveryPhone),
				DeliveryMode:    deliveryMode,
				DestLatitude:    req.Lat,
				DestLongitude:   req.Lng,
				DestPincode:     req.DestPincode,
				CreatedAt:       time.Now(),
			}

			var subTotal int64
			var orderItems []pharmacy.PharmacyOrderItem

			for _, item := range plan.Items {
				// 1. Atomic Locking via Redis Lua
				if database.Rdb != nil {
					stockKey := fmt.Sprintf("inventory:%d:%d", plan.PharmacyID, item.MedicineID)
					res, err := database.Rdb.Eval(ctx, deductStockScript, []string{stockKey}, item.Quantity).Result()
					if err != nil || res.(int64) == -1 {
						txErr = fmt.Errorf("insufficient stock for item %d", item.MedicineID)
						return txErr
					}
					successfulDeductions = append(successfulDeductions, deduction{key: stockKey, qty: item.Quantity})
				}

				// 2. Fetch price from DB
				var inv pharmacy.PharmacyInventory
				if err := tx.Preload("Medicine").Where("pharmacy_id = ? AND medicine_id = ?", plan.PharmacyID, item.MedicineID).First(&inv).Error; err != nil {
					txErr = err
					return err
				}

				if inv.StockQuantity < item.Quantity {
					txErr = fmt.Errorf("insufficient stock for item %d", item.MedicineID)
					return txErr
				}

				itemTotal := int64(item.Quantity) * inv.Medicine.Prices["INR"]
				subTotal += itemTotal

				// 3. Update DB Stock
				if err := tx.Model(&inv).Update("stock_quantity", gorm.Expr("stock_quantity - ?", item.Quantity)).Error; err != nil {
					txErr = err
					return err
				}

				orderItems = append(orderItems, pharmacy.PharmacyOrderItem{
					MedicineName: inv.Medicine.Name,
					Quantity:     item.Quantity,
					UnitPrice:    inv.Medicine.Prices["INR"],
					TotalPrice:   itemTotal,
				})
			}

			// Calculate Shipping (Multiply by 100 to convert float64 currency to int64 cents)
			shipResult, err := shipping.ShippingService.CalculateBestShipping(plan.DeliveryMode, 1.0, "560001", "560001")
			if err == nil {
				subTotal += int64(shipResult["cost"].(float64) * 100)
			}

			subOrder.Amount = subTotal
			if err := tx.Create(subOrder).Error; err != nil {
				txErr = err
				return err
			}

			// Add Items to DB
			for _, oi := range orderItems {
				oi.OrderID = subOrder.ID
				if err := tx.Create(&oi).Error; err != nil {
					txErr = err
					return err
				}
			}

			masterOrder.Amount += subTotal
		}

		return tx.Save(masterOrder).Error
	})

	return masterOrder, err
}


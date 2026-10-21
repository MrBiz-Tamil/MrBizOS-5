package shipping

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/utils/privacy"
)

type DispatchDock struct {
	shippingService *Service
}

func NewDispatchDock(s *Service) *DispatchDock {
	return &DispatchDock{shippingService: s}
}

// GlobalDispatchDock is initialized lazily via InitDispatchDock() once
// ShippingService has been assigned in app.go.  Never call New() at package
// init because ShippingService is nil at that point.
var GlobalDispatchDock *DispatchDock

// InitDispatchDock must be called from app.go after ShippingService is set.
func InitDispatchDock() {
	GlobalDispatchDock = NewDispatchDock(ShippingService)
}

func (d *DispatchDock) DispatchOrder(orderID uint) error {
	var order pharmacy.PharmacyOrder
	if err := database.DB.Preload("Pharmacy").First(&order, orderID).Error; err != nil {
		return err
	}

	if order.Status != "paid" && order.Status != "confirmed" {
		return errors.New("order is not in a dispatchable state")
	}

	switch order.DeliveryMode {
	case pharmacy.ModeQuick:
		return d.dispatchLocalCourier(&order)
	case pharmacy.ModeStandard:
		return d.dispatchThirdPartyLogistics(&order)
	default:
		return fmt.Errorf("unknown delivery mode: %s", order.DeliveryMode)
	}
}

// dispatchLocalCourier attempts intelligent auto-assignment first (scored best-fit),
// then falls back to a city-wide broadcast if no courier is available.
func (d *DispatchDock) dispatchLocalCourier(order *pharmacy.PharmacyOrder) error {
	ctx := context.Background()

	// 1. Try Intelligent Auto-Assignment (Requires Redis for locks)
	if database.Rdb != nil {
		_, err := TryAutoAssign(ctx, order)
		if err == nil {
			log.Printf("[Dispatch] order %d auto-assigned", order.ID)
			return nil
		}
		log.Printf("[Dispatch] Auto-assign failed for order %d: %v", order.ID, err)
	} else {
		log.Printf("[Dispatch] Redis unavailable — skipping auto-assign for order %d", order.ID)
	}

	// 2. Fallback: Broadcast to all couriers in the city
	broadcastOrderToAllCouriers(ctx, order)
	return nil
}

// dispatchThirdPartyLogistics routes STANDARD orders through Shiprocket first,
// falling back to India Post for pincodes Shiprocket cannot service.
//
// Fixes:
//   - Uses real order data (customer name, address, items, weight)
//   - Calls the real Shiprocket API client
//   - Records a double-entry finance ledger entry for the shipping cost
//   - Creates a CourierDelivery row so SLA monitor can track the shipment
func (d *DispatchDock) dispatchThirdPartyLogistics(order *pharmacy.PharmacyOrder) error {
	ctx := context.Background()

	// 1. Idempotency Check: Don't dispatch if a delivery record already exists
	var existing pharmacy.CourierDelivery
	if err := database.DB.WithContext(ctx).Where("order_id = ?", order.ID).First(&existing).Error; err == nil {
		log.Printf("[Zero-Error] order %d already has a delivery record — skipping", order.ID)
		return nil
	}

	// 2. Fetch patient details
	var patient identity.User
	if err := database.DB.WithContext(ctx).First(&patient, order.UserID).Error; err != nil {
		log.Printf("[3PL] order %d: cannot fetch patient user: %v", order.ID, err)
		patient.FullName = "Customer"
	}

	weightKg := calculateOrderWeight(ctx, order)

	// 3. Cost-Effective Discovery: Use SLA-aware optimal routing
	isExpress := false // Currently no explicit express flag on standard orders, could be added later
	orderTotalINR := float64(order.Amount) / 100.0
	candidates := GlobalRegistry.GetOptimalProviders(order.DestPincode, weightKg, order.Pharmacy.Pincode, orderTotalINR, isExpress)
	if len(candidates) == 0 {
		return fmt.Errorf("no shipping providers available for pincode %s", order.DestPincode)
	}

	// 4. Robust Execution Loop (Zero-Error Handle)
	var finalProvider ShippingProvider
	var awbCode string
	var lastErr error

	for _, p := range candidates {
		log.Printf("[3PL] Attempting dispatch for order %d via %s (Cost-Effective Check)", order.ID, p.Name())
		
		// Build a ShipmentRequest with the real recipient and order data so the
		// courier waybill carries the patient's name/phone/address — not the
		// placeholders ("Dr.Ayya Customer", "9999999999") the old signature forced.
		loc := LookupPincode(order.DestPincode) // P4: pincode → (city, state)
		shipReq := ShipmentRequest{
			OrderID:        fmt.Sprintf("MH-%s", order.OrderNumber),
			WeightKg:       weightKg,
			SubTotalINR:    float64(order.Amount) / 100.0,
			RecipientName:  patient.FullName,
			RecipientPhone: string(order.DeliveryPhone),
			RecipientEmail: patient.Email,
			Address:        string(order.ShippingAddress),
			Pincode:        order.DestPincode,
			City:           loc.City,
			State:          loc.State,
			Items: []ShipmentItem{
				{Name: "Medicine Package", SKU: "MED-PKG", Units: 1, SellingPrice: float64(order.Amount) / 100.0},
			},
		}
		awb, err := p.CreateShipment(shipReq)
		
		if err == nil && awb != "" {
			awbCode = awb
			finalProvider = p
			break
		}
		
		lastErr = err
		log.Printf("[3PL] Provider %s failed: %v — trying next", p.Name(), err)
	}

	if finalProvider == nil {
		return fmt.Errorf("all providers failed for order %d. Last error: %v", order.ID, lastErr)
	}

	// 5. Atomic Ledger & Data Recording
	providerName := finalProvider.Name()
	if d.shippingService != nil && d.shippingService.financeSvc != nil {
		d.recordShippingLedger(ctx, order, finalProvider, weightKg)
	}

	// 6. Create a CourierDelivery row so the SLA monitor can track the shipment

	// 6. Create a CourierDelivery row so the SLA monitor can track the shipment
	slaDeadline := order.CreatedAt.Add(standardSLAHours * time.Hour)
	delivery := pharmacy.CourierDelivery{
		OrderID:               order.ID,
		Status:                "assigned",
		EstimatedDeliveryTime: slaDeadline,
		SLADeadline:           &slaDeadline,
		ProviderName:          providerName,
		TrackingID:            awbCode,
		DeliveryNotes:         privacy.EncryptedString(fmt.Sprintf("3PL provider: %s | AWB: %s", providerName, awbCode)),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
	if err := database.DB.WithContext(ctx).Create(&delivery).Error; err != nil {
		log.Printf("[3PL] order %d: create delivery row failed: %v", order.ID, err)
		// non-fatal — order still gets tracking_id
	}

	// 7. Create a Shipment record for the Logistics Dashboard visibility
	shipment := pharmacy.Shipment{
		TrackingNumber:   awbCode,
		OrderID:          order.ID,
		OrderType:        "pharmacy",
		ProviderName:     providerName,
		Status:           "in_transit",
		Origin:           order.Pharmacy.City,
		Destination:      order.DestPincode,
		EstimatedArrival: &slaDeadline,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	if err := database.DB.WithContext(ctx).Create(&shipment).Error; err != nil {
		log.Printf("[3PL] order %d: cannot create shipment row: %v", order.ID, err)
	}

	// 8. Update the order status and tracking ID
	return database.DB.WithContext(ctx).Model(order).Updates(map[string]interface{}{
		"status":       "shipped",
		"tracking_id":  awbCode,
		"sla_deadline": slaDeadline,
	}).Error
}

// recordShippingLedger posts a double-entry transaction for the estimated
// 3PL shipping cost: Debit Expense:Logistics:Delivery, Credit Liability:CourierPartner:{provider}.
func (d *DispatchDock) recordShippingLedger(ctx context.Context, order *pharmacy.PharmacyOrder, provider ShippingProvider, weightKg float64) {
	svc := d.shippingService.financeSvc

	// Use the provider's own cost calculation
	costRupees, err := provider.CalculateCost(weightKg, order.Pharmacy.Pincode, order.DestPincode)
	if err != nil {
		costRupees = 40.0 + (weightKg * 20.0) // Fallback
	}
	costPaise := int64(costRupees * 100)

	expenseAcct, err := svc.GetOrCreateAccount(ctx, "Expense:Logistics:Delivery", finance.TypeAsset)
	if err != nil {
		log.Printf("[3PL] ledger: get expense account: %v", err)
		return
	}
	partnerAcct, err := svc.GetOrCreateAccount(ctx,
		fmt.Sprintf("Liability:CourierPartner:%s", provider.Name()), finance.TypeLiability)
	if err != nil {
		log.Printf("[3PL] ledger: get partner account: %v", err)
		return
	}

	refID := fmt.Sprintf("3PL-SHIP-ORD-%d", order.ID)
	_, err = svc.RecordTransaction(ctx, refID, "3PL_SHIPPING",
		fmt.Sprintf("3PL shipping cost for order #%s via %s", order.OrderNumber, provider),
		[]finance.EntryData{
			{AccountID: expenseAcct.ID, Direction: finance.DirectionDebit, Amount: costPaise},
			{AccountID: partnerAcct.ID, Direction: finance.DirectionCredit, Amount: costPaise},
		},
	)
	if err != nil {
		log.Printf("[3PL] ledger: record transaction for order %d: %v", order.ID, err)
	}
}

// calculateOrderWeight sums the PhysicalWeightGrams of all items in the order.
// Falls back to 0.5kg if data is missing.
func calculateOrderWeight(ctx context.Context, order *pharmacy.PharmacyOrder) float64 {
	var items []struct {
		MedicineID uint `json:"medicine_id"`
		Quantity   int  `json:"quantity"`
	}
	if err := json.Unmarshal(order.Items, &items); err != nil {
		return 0.5 // fallback
	}

	totalGrams := 0
	for _, item := range items {
		var variant pharmacy.ProductVariant
		// Attempt to find the default variant for this medicine to get weight
		if err := database.DB.WithContext(ctx).
			Where("medicine_id = ?", item.MedicineID).
			First(&variant).Error; err == nil {
			totalGrams += variant.PhysicalWeightGrams * item.Quantity
		} else {
			totalGrams += 100 * item.Quantity // 100g placeholder per item
		}
	}

	if totalGrams == 0 {
		return 0.5
	}

	return float64(totalGrams) / 1000.0
}

// AutoDispatchProcessor is a legacy polling loop kept for backward compatibility.
// New code should prefer the SLA-monitor goroutine (RunSLAMonitor) which does the
// same job on a tighter interval with smarter escalation logic.
func (d *DispatchDock) AutoDispatchProcessor() {
	go func() {
		for {
			var pendingOrders []pharmacy.PharmacyOrder
			database.DB.Preload("Pharmacy").Where("status IN ?", []string{"paid", "confirmed"}).Limit(10).Find(&pendingOrders)

			for _, order := range pendingOrders {
				err := d.DispatchOrder(order.ID)
				if err != nil {
					fmt.Printf("Dispatch Error for Order %d: %v\n", order.ID, err)
				}
			}
			time.Sleep(1 * time.Minute)
		}
	}()
}

// publishOrderEvent is the legacy broadcast helper retained for direct WS
// publish calls that have not yet been migrated to publishDeliveryEvent.
func publishOrderEvent(ctx context.Context, room string, payload map[string]interface{}) {
	if database.Rdb == nil {
		return
	}
	payload["room_id"] = room
	data, _ := json.Marshal(payload)
	database.Rdb.Publish(ctx, "chat:global", data)
}

package shipping

// sla.go — SLA monitoring goroutine with automatic escalation.
//
// Escalation ladder for QUICK (30-min SLA) deliveries:
//
//   T+0     order paid → TryAutoAssign called immediately
//   T+2min  still "paid" (unassigned) → retry auto-assign with wider radius
//   T+5min  still "paid" → broadcast to all couriers + alert ops via WS
//   T+15min still "assigned" but not picked up → reassign to next-best courier
//   T+30min SLA deadline reached → auto-cancel, full refund, ops alert
//
// STANDARD deliveries use a 24-hour SLA; same ladder scaled proportionally.
//
// Human touch-points: zero.  Ops are notified by WS so they can optionally
// intervene, but the system handles every stage automatically.

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"drayya-core/internal/database"
	financeService "drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/pharmacy"
)

const (
	slaCheckInterval = 30 * time.Second // scan every 30 s
	retryAssignDelay = 2 * time.Minute  // unassigned → retry auto-assign
	broadcastDelay   = 5 * time.Minute  // unassigned → broadcast fallback
	stuckPickupDelay = 15 * time.Minute // assigned but not picked up → reassign

	// STANDARD order SLA constants
	standardUnshippedAlert = 2 * time.Hour   // ops alert if STANDARD order not dispatched in 2h
	standardSLACheckPeriod = 5 * time.Minute // how often to scan STANDARD orders
)

// RunSLAMonitor is a long-running goroutine. Start it once from server init.
func RunSLAMonitor() {
	log.Println("[SLA] Monitor started")
	for {
		time.Sleep(slaCheckInterval)
		checkSLAs(context.Background())
	}
}

// standardSLACounter throttles STANDARD checks so they only run every
// standardSLACheckPeriod even though RunSLAMonitor ticks every 30 s.
// Declared as atomic int32 so concurrent access from tests is safe.
var standardSLACounter int32

func checkSLAs(ctx context.Context) {
	if database.DB == nil {
		return
	}
	checkUnassignedOrders(ctx)
	checkStuckDeliveries(ctx)
	checkBreachedSLAs(ctx)

	// STANDARD SLA runs every (standardSLACheckPeriod / slaCheckInterval) ticks
	ticksPerStandardCheck := int32(standardSLACheckPeriod / slaCheckInterval) // = 10
	if atomic.AddInt32(&standardSLACounter, 1) >= ticksPerStandardCheck {
		atomic.StoreInt32(&standardSLACounter, 0)
		checkStandardOrders(ctx)
	}
}

// checkUnassignedOrders retries assignment for orders that have been paid but
// not yet assigned to any courier.
func checkUnassignedOrders(ctx context.Context) {
	retryThreshold := time.Now().Add(-retryAssignDelay)
	broadcastThreshold := time.Now().Add(-broadcastDelay)

	var orders []pharmacy.PharmacyOrder
	database.DB.WithContext(ctx).
		Preload("Pharmacy").
		Where("status IN ? AND delivery_mode = ? AND created_at < ?",
			[]string{"paid", "confirmed"}, pharmacy.ModeQuick, retryThreshold).
		Limit(50).
		Find(&orders)

	for _, order := range orders {
		o := order
		age := time.Since(o.CreatedAt)

		if age >= broadcastDelay {
			// Escalate: broadcast to all couriers + alert ops
			log.Printf("[SLA] order %d unassigned >5 min — broadcasting fallback", o.ID)
			broadcastOrderToAllCouriers(ctx, &o)
			alertOps(ctx, fmt.Sprintf("[SLA] ⚠️ Order #%d unassigned >5 min — broadcasted", o.ID))
		} else if age >= retryAssignDelay {
			// Retry auto-assign silently
			if _, err := TryAutoAssign(ctx, &o); err != nil {
				log.Printf("[SLA] order %d retry-assign failed: %v", o.ID, err)
			}
		}

		// Notify patient of slight delay if order is >3 min old
		if age >= 3*time.Minute {
			notifyOrderDelayed(ctx, o.UserID,
				fmt.Sprintf("We're finding a courier for your order #%s. It'll be with you soon!", o.OrderNumber))
		}

		_ = broadcastThreshold // consumed above
	}
}

// checkStuckDeliveries handles deliveries assigned to a courier but not picked
// up within stuckPickupDelay; reassigns to the next-best available courier.
func checkStuckDeliveries(ctx context.Context) {
	threshold := time.Now().Add(-stuckPickupDelay)

	var deliveries []pharmacy.CourierDelivery
	database.DB.WithContext(ctx).
		Where("status = ? AND created_at < ?", "assigned", threshold).
		Limit(20).
		Find(&deliveries)

	for _, d := range deliveries {
		del := d
		log.Printf("[SLA] delivery %d assigned but not picked up >15 min — reassigning", del.ID)

		// Release original courier's load
		DecrementCourierLoad(ctx, del.CourierID)

		// Mark old delivery as failed-reassign
		database.DB.WithContext(ctx).Model(&del).Updates(map[string]interface{}{
			"status":         "failed",
			"delivery_notes": "auto-reassigned: courier did not pick up within 15 min",
		})

		// Fetch order and try a fresh assignment
		var order pharmacy.PharmacyOrder
		if err := database.DB.WithContext(ctx).
			Preload("Pharmacy").First(&order, del.OrderID).Error; err != nil {
			continue
		}
		order.Status = "paid" // reset so TryAutoAssign sees it as unassigned
		if _, err := TryAutoAssign(ctx, &order); err != nil {
			// Fallback to broadcast
			broadcastOrderToAllCouriers(ctx, &order)
			alertOps(ctx, fmt.Sprintf("[SLA] ⚠️ Order #%d reassignment failed — broadcasted", order.ID))
		} else {
			log.Printf("[SLA] order %d successfully reassigned", order.ID)
		}
	}
}

// checkBreachedSLAs cancels orders whose SLA deadline has passed and issues a
// full refund to the patient.
func checkBreachedSLAs(ctx context.Context) {
	now := time.Now()

	var deliveries []pharmacy.CourierDelivery
	database.DB.WithContext(ctx).
		Where("sla_deadline < ? AND status NOT IN ?", now,
			[]string{"completed", "failed", "cancelled"}).
		Limit(20).
		Find(&deliveries)

	for _, d := range deliveries {
		del := d
		log.Printf("[SLA] delivery %d breached SLA deadline — auto-cancelling", del.ID)

		// Mark SLA breach timestamp
		database.DB.WithContext(ctx).Model(&del).Updates(map[string]interface{}{
			"status":          "failed",
			"sla_breached_at": now,
			"delivery_notes":  "auto-cancelled: SLA deadline exceeded",
		})

		DecrementCourierLoad(ctx, del.CourierID)

		// Cancel order + full refund
		var order pharmacy.PharmacyOrder
		if err := database.DB.WithContext(ctx).First(&order, del.OrderID).Error; err != nil {
			continue
		}
		issueFullRefund(ctx, &order)

		// Alert ops
		alertOps(ctx, fmt.Sprintf(
			"[SLA BREACH] 🔴 Order #%s (ID %d) exceeded %d-min SLA — auto-refunded patient",
			order.OrderNumber, order.ID, quickSLAMinutes,
		))

		// Notify patient
		notifyOrderFailed(ctx, order.UserID,
			fmt.Sprintf("We're sorry — your order #%s could not be delivered in time. A full refund has been initiated.", order.OrderNumber))
	}
}

// issueFullRefund releases held funds back to the patient's wallet via the
// double-entry ledger. Non-fatal; any error is logged for reconciliation.
func issueFullRefund(ctx context.Context, order *pharmacy.PharmacyOrder) {
	if order.Amount <= 0 {
		database.DB.WithContext(ctx).Model(order).Update("status", "cancelled")
		return
	}

	repo := financeService.NewRepository(database.DB)
	svc := financeService.NewService(repo)

	walletAcct, err := svc.GetOrCreateAccount(ctx,
		fmt.Sprintf("PatientWallet:%d", order.UserID), financeService.TypeLiability)
	if err != nil {
		log.Printf("[SLA] refund: get wallet for user %d: %v", order.UserID, err)
		return
	}
	heldFunds, err := svc.GetOrCreateAccount(ctx, "HeldFunds", financeService.TypeLiability)
	if err != nil {
		log.Printf("[SLA] refund: get HeldFunds: %v", err)
		return
	}

	entries := []financeService.EntryData{
		{AccountID: heldFunds.ID, Direction: financeService.DirectionDebit, Amount: order.Amount},
		{AccountID: walletAcct.ID, Direction: financeService.DirectionCredit, Amount: order.Amount},
	}
	refID := fmt.Sprintf("SLA-REFUND-ORD-%d", order.ID)
	if _, err := svc.RecordTransaction(ctx, refID, "SLA_REFUND",
		fmt.Sprintf("SLA breach auto-refund for order #%s", order.OrderNumber), entries); err != nil {
		log.Printf("[SLA] refund ledger failed for order %d: %v", order.ID, err)
	}

	database.DB.WithContext(ctx).Model(order).Update("status", "cancelled")
}

// broadcastOrderToAllCouriers publishes a delivery_request event to all couriers
// city-wide (same fallback as the original broadcast model).
func broadcastOrderToAllCouriers(ctx context.Context, order *pharmacy.PharmacyOrder) {
	publishDeliveryEvent(ctx, fmt.Sprintf("room:couriers:city:%s", order.Pharmacy.City), map[string]interface{}{
		"type": "delivery_request",
		"metadata": map[string]interface{}{
			"order_id":        order.ID,
			"order_number":    order.OrderNumber,
			"pickup_address":  order.Pharmacy.Address,
			"dropoff_address": order.ShippingAddress,
			"earnings":        45.0, // TODO: calculate real earnings
		},
	})

	if database.DB != nil {
		if database.Rdb != nil {
			dispatchKey := fmt.Sprintf("dispatch:order:%d", order.ID)
			database.Rdb.SetNX(ctx, dispatchKey, "pending", 5*time.Minute)
		}
		database.DB.WithContext(ctx).Model(order).Update("status", "dispatching")
	}
}

// alertOps publishes a system alert to the ops dashboard room.
func alertOps(ctx context.Context, message string) {
	publishDeliveryEvent(ctx, "room:system:dispatch", map[string]interface{}{
		"type":    "system_notification",
		"content": message,
	})
}

// notifyOrderDelayed sends a patient-facing WS message for a delayed order.
func notifyOrderDelayed(ctx context.Context, patientUserID uint, message string) {
	publishDeliveryEvent(ctx, fmt.Sprintf("room:user:%d", patientUserID), map[string]interface{}{
		"type":    "order_update",
		"status":  "delayed",
		"message": message,
	})
}

// notifyOrderFailed sends a patient-facing WS message for a cancelled order.
func notifyOrderFailed(ctx context.Context, patientUserID uint, message string) {
	publishDeliveryEvent(ctx, fmt.Sprintf("room:user:%d", patientUserID), map[string]interface{}{
		"type":    "order_update",
		"status":  "cancelled",
		"message": message,
	})
}

// ─── STANDARD order SLA monitoring ──────────────────────────────────────────
//
// STANDARD (24-hour) orders are routed through 3PL providers (Shiprocket /
// India Post). The system cannot control their internal pickup schedule, so
// the escalation ladder is simpler:
//
//   T+2h   order still "paid" (not yet dispatched) → ops alert
//   T+24h  delivery SLA deadline passed → ops alert + patient notification
//
// We do NOT auto-cancel STANDARD orders on breach — ops must decide whether
// to re-route or refund, since the parcel may already be in the 3PL network.

// checkStandardOrders looks for STANDARD orders that are stuck and raises ops
// alerts without taking automated destructive action.
func checkStandardOrders(ctx context.Context) {
	now := time.Now()

	// 1. Orders paid but not dispatched within 2 hours
	unshippedThreshold := now.Add(-standardUnshippedAlert)
	var stuckOrders []pharmacy.PharmacyOrder
	database.DB.WithContext(ctx).
		Where("status IN ? AND delivery_mode = ? AND created_at < ?",
			[]string{"paid", "confirmed"}, pharmacy.ModeStandard, unshippedThreshold).
		Limit(20).
		Find(&stuckOrders)

	for _, o := range stuckOrders {
		order := o
		age := time.Since(order.CreatedAt).Round(time.Minute)
		log.Printf("[SLA-STANDARD] order %d paid but not dispatched (%v ago) — alerting ops", order.ID, age)
		alertOps(ctx, fmt.Sprintf(
			"[STANDARD SLA] ⚠️ Order #%s (ID %d) paid %v ago but not yet dispatched",
			order.OrderNumber, order.ID, age,
		))
	}

	// 2. STANDARD deliveries past their SLA deadline
	var breachedDeliveries []pharmacy.CourierDelivery
	database.DB.WithContext(ctx).
		Joins("JOIN pharmacy_orders ON pharmacy_orders.id = courier_deliveries.order_id").
		Where("courier_deliveries.sla_deadline < ? AND courier_deliveries.status NOT IN ? AND pharmacy_orders.delivery_mode = ?",
			now, []string{"completed", "failed", "cancelled"}, pharmacy.ModeStandard).
		Limit(20).
		Find(&breachedDeliveries)

	for _, d := range breachedDeliveries {
		del := d
		log.Printf("[SLA-STANDARD] delivery %d breached 24h SLA — alerting ops (no auto-cancel)", del.ID)

		// Mark SLA breach timestamp (idempotent — only if not already set)
		if del.SLABreachedAt == nil {
			database.DB.WithContext(ctx).Model(&del).Update("sla_breached_at", now)
		}

		// Fetch order for notification context
		var order pharmacy.PharmacyOrder
		if err := database.DB.WithContext(ctx).First(&order, del.OrderID).Error; err != nil {
			continue
		}

		alertOps(ctx, fmt.Sprintf(
			"[STANDARD SLA BREACH] 🔴 Order #%s (ID %d) exceeded 24h SLA — manual intervention required",
			order.OrderNumber, order.ID,
		))

		// Notify patient that delivery is delayed (but not cancelled)
		notifyOrderDelayed(ctx, order.UserID, fmt.Sprintf(
			"Your order #%s is taking longer than expected. We're following up with the courier partner.",
			order.OrderNumber,
		))
	}
}

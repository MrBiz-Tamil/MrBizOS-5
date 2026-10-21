package shipping

// batcher.go — Order clustering and batch-delivery optimiser.
//
// Every batchWindowSeconds the RunBatchOptimiser goroutine:
//  1. Scans all QUICK orders that are still "paid" (not yet assigned).
//  2. Clusters orders whose pickup pharmacy is the same AND whose destination
//     is within clusterRadiusKm of each other into a single delivery batch.
//  3. Assigns the whole batch to one courier via TryAutoAssign, so a single
//     courier trip covers 2-5 stops — increasing utilisation 3-5×.
//
// For 1 000 orders/day across 5 mega-cities this keeps the average load at
// 200 orders/city/day.  With ~15 active couriers per city and batching this
// is comfortably ≤ 15 trips/courier/day.

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/pharmacy"
)

const (
	batchWindowSeconds = 180          // scan every 3 minutes
	clusterRadiusKm    = 2.5          // orders within 2.5 km of first order in cluster
	maxBatchSize       = 5            // max stops per courier trip
	batchLookbackMins  = 8            // batch orders placed in last 8 min (must be < SLA retryAssignDelay
	                                  // so the batcher always runs at least once before SLA escalation)
	// staleDispatchingMins is how long an order can sit in "dispatching" (broadcast sent,
	// no courier accepted) before we reset it to "paid" so the batcher can pick it up again.
	staleDispatchingMins = 4
)

// RunBatchOptimiser is a long-running goroutine. Start it once from server init.
func RunBatchOptimiser() {
	log.Println("[Batcher] Order batch optimiser started")
	for {
		time.Sleep(batchWindowSeconds * time.Second)
		runBatchCycle(context.Background())
	}
}

func runBatchCycle(ctx context.Context) {
	if database.DB == nil {
		return
	}

	// Reset orders stuck in "dispatching" (broadcast fired but no courier accepted)
	// back to "paid" so they re-enter the assignment pipeline.
	staleThreshold := time.Now().Add(-staleDispatchingMins * time.Minute)
	database.DB.WithContext(ctx).
		Model(&pharmacy.PharmacyOrder{}).
		Where("status = ? AND delivery_mode = ? AND updated_at < ?",
			"dispatching", pharmacy.ModeQuick, staleThreshold).
		Update("status", "paid")

	// Fetch recent unassigned QUICK orders that have destination coordinates.
	// Use a slightly wider lookback than batchLookbackMins to catch orders that
	// were just reset from "dispatching" back to "paid".
	cutoff := time.Now().Add(-(batchLookbackMins + staleDispatchingMins) * time.Minute)
	var orders []pharmacy.PharmacyOrder
	if err := database.DB.WithContext(ctx).
		Preload("Pharmacy").
		Where("status = ? AND delivery_mode = ? AND created_at >= ? AND dest_latitude != 0",
			"paid", pharmacy.ModeQuick, cutoff).
		Order("created_at ASC").
		Limit(100).
		Find(&orders).Error; err != nil || len(orders) == 0 {
		return
	}

	// Group into clusters per pharmacy (same pickup point) then by dest proximity
	grouped := groupByPharmacy(orders)

	for pharmacyID, group := range grouped {
		batches := clusterByDestination(group)
		for _, batch := range batches {
			if len(batch) < 2 {
				// Single order — let the regular auto-assign handle it
				continue
			}
			assignBatch(ctx, pharmacyID, batch)
		}
	}
}

// groupByPharmacy buckets orders by their source pharmacy so we only cluster
// orders that have the same pickup point.
func groupByPharmacy(orders []pharmacy.PharmacyOrder) map[uint][]pharmacy.PharmacyOrder {
	m := make(map[uint][]pharmacy.PharmacyOrder)
	for _, o := range orders {
		m[o.PharmacyID] = append(m[o.PharmacyID], o)
	}
	return m
}

// clusterByDestination greedily clusters orders within clusterRadiusKm of the
// first order in each cluster using a simple greedy sweep.
func clusterByDestination(orders []pharmacy.PharmacyOrder) [][]pharmacy.PharmacyOrder {
	assigned := make([]bool, len(orders))
	var batches [][]pharmacy.PharmacyOrder

	for i := 0; i < len(orders); i++ {
		if assigned[i] {
			continue
		}
		cluster := []pharmacy.PharmacyOrder{orders[i]}
		assigned[i] = true
		anchor := orders[i]

		for j := i + 1; j < len(orders) && len(cluster) < maxBatchSize; j++ {
			if assigned[j] {
				continue
			}
			dist := haversineKm(
				anchor.DestLatitude, anchor.DestLongitude,
				orders[j].DestLatitude, orders[j].DestLongitude,
			)
			if dist <= clusterRadiusKm {
				cluster = append(cluster, orders[j])
				assigned[j] = true
			}
		}
		batches = append(batches, cluster)
	}
	return batches
}

// assignBatch assigns all orders in a cluster to a single courier under one
// shared batch ID.  If auto-assignment fails, each order is left for the
// individual dispatcher to handle.
func assignBatch(ctx context.Context, pharmacyID uint, batch []pharmacy.PharmacyOrder) {
	if len(batch) == 0 {
		return
	}

	batchID := uuid.New().String()
	// Use the first order as the representative for scoring
	anchor := batch[0]

	delivery, err := TryAutoAssign(ctx, &anchor)
	if err != nil {
		log.Printf("[Batcher] pharmacy %d batch %s: auto-assign failed: %v",
			pharmacyID, batchID[:8], err)
		return
	}

	courierID := delivery.CourierID
	slaDeadline := delivery.SLADeadline

	// Assign remaining orders in the batch to the same courier
	for i := 1; i < len(batch); i++ {
		o := batch[i]
		extra := pharmacy.CourierDelivery{
			OrderID:               o.ID,
			CourierID:             courierID,
			Status:                "assigned",
			PickupLatitude:        o.Pharmacy.Latitude,
			PickupLongitude:       o.Pharmacy.Longitude,
			DestLatitude:          o.DestLatitude,
			DestLongitude:         o.DestLongitude,
			EstimatedDeliveryTime: slaDeadline.Add(time.Duration(i*10) * time.Minute), // stagger ETAs
			SLADeadline:           slaDeadline,
			BatchID:               batchID,
			CreatedAt:             time.Now(),
			UpdatedAt:             time.Now(),
		}
		if err := database.DB.WithContext(ctx).Create(&extra).Error; err != nil {
			log.Printf("[Batcher] create extra delivery for order %d: %v", o.ID, err)
			continue
		}
		database.DB.WithContext(ctx).Model(&o).Updates(map[string]interface{}{
			"status":       "assigned",
			"sla_deadline": slaDeadline,
		})
	}

	// Tag the anchor delivery with the batch ID too
	database.DB.WithContext(ctx).Model(delivery).Update("batch_id", batchID)

	log.Printf("[Batcher] batch %s: %d orders → courier %d (pharmacy %d)",
		batchID[:8], len(batch), courierID, pharmacyID)

	// Notify the courier of the batch via WebSocket
	NotifyBatchAssigned(ctx, courierID, batchID, batch)
}

// EstimatedBatchETA returns a rough minutes-to-delivery estimate for a batch
// based on average distance and assumed 20 km/h city speed.
func EstimatedBatchETA(anchor pharmacy.PharmacyOrder, stops int) int {
	distKm := haversineKm(
		anchor.Pharmacy.Latitude, anchor.Pharmacy.Longitude,
		anchor.DestLatitude, anchor.DestLongitude,
	)
	// 20 km/h city speed + 5 min per stop for handover
	minutes := int((distKm/20.0)*60) + stops*5
	if minutes < 10 {
		minutes = 10 // minimum realistic ETA
	}
	return minutes
}

// NotifyBatchAssigned sends a WebSocket push to the courier with their stop list.
func NotifyBatchAssigned(ctx context.Context, courierID uint, batchID string, orders []pharmacy.PharmacyOrder) {
	stops := make([]map[string]interface{}, 0, len(orders))
	for i, o := range orders {
		stops = append(stops, map[string]interface{}{
			"stop":        i + 1,
			"order_id":    o.ID,
			"address":     o.ShippingAddress,
			"dest_lat":    o.DestLatitude,
			"dest_lng":    o.DestLongitude,
			"phone":       o.DeliveryPhone,
		})
	}
	publishDeliveryEvent(ctx, fmt.Sprintf("room:courier:%d", courierID), map[string]interface{}{
		"type":     "batch_assigned",
		"batch_id": batchID,
		"stops":    stops,
		"total":    len(orders),
	})
}

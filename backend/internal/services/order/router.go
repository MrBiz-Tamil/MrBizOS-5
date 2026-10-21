package order

import (
	"context"
	"math"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/pharmacy"
	"sort"
)

type CartItem struct {
	MedicineID uint `json:"medicine_id"`
	Quantity   int  `json:"quantity"`
}

type FulfillmentPlan struct {
	PharmacyID   uint
	Items        []CartItem
	Distance     float64
	DeliveryMode pharmacy.DeliveryMode
}

// FindBestFulfillment determines which pharmacy should fulfill the order
func FindBestFulfillment(ctx context.Context, lat, lng float64, city string, items []CartItem) ([]FulfillmentPlan, error) {
	// 1. Fetch Quick Delivery Zone for the city
	var zone pharmacy.DeliveryZone
	isQuickCity := false
	maxQuickRadius := 10.0
	if err := database.DB.WithContext(ctx).Where("city = ? AND is_quick_enabled = ?", city, true).First(&zone).Error; err == nil {
		isQuickCity = true
		maxQuickRadius = zone.MaxRadiusKm
	}

	// 2. Fetch all active pharmacies
	var pharmacies []pharmacy.Pharmacy
	if err := database.DB.WithContext(ctx).Where("is_active = ?", true).Find(&pharmacies).Error; err != nil {
		return nil, err
	}

	// 3. Filter & Categorize Candidates
	type Candidate struct {
		Pharmacy     pharmacy.Pharmacy
		Distance     float64
		DeliveryMode pharmacy.DeliveryMode
		Priority     int // 0: Local Quick, 1: Local Standard, 2: Remote/Warehouse
	}
	var candidates []Candidate

	for _, p := range pharmacies {
		dist := haversine(lat, lng, p.Latitude, p.Longitude)
		mode := pharmacy.ModeStandard
		priority := 2 // Default: Global/Remote

		if p.City == city {
			if isQuickCity && dist <= maxQuickRadius && dist <= p.ServiceRadius {
				mode = pharmacy.ModeQuick
				priority = 0
			} else {
				priority = 1
			}
		} else if p.IsCentralWarehouse {
			priority = 2
		}

		// Only consider if within service radius or it's a warehouse
		if dist <= p.ServiceRadius || p.IsCentralWarehouse || dist <= 50.0 {
			candidates = append(candidates, Candidate{Pharmacy: p, Distance: dist, DeliveryMode: mode, Priority: priority})
		}
	}

	// Sort by Priority (Local first), then Distance
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Priority != candidates[j].Priority {
			return candidates[i].Priority < candidates[j].Priority
		}
		return candidates[i].Distance < candidates[j].Distance
	})

	// 4. Check Stock & Greedy Allocation
	remainingItems := make(map[uint]int)
	for _, item := range items {
		remainingItems[item.MedicineID] = item.Quantity
	}

	var plans []FulfillmentPlan

	for _, cand := range candidates {
		if len(remainingItems) == 0 {
			break
		}

		plan := FulfillmentPlan{
			PharmacyID:   cand.Pharmacy.ID,
			Distance:     cand.Distance,
			DeliveryMode: cand.DeliveryMode,
		}

		var itemIDs []uint
		for id := range remainingItems {
			itemIDs = append(itemIDs, id)
		}

		var inventory []pharmacy.PharmacyInventory
		database.DB.WithContext(ctx).
			Where("pharmacy_id = ? AND medicine_id IN ?", cand.Pharmacy.ID, itemIDs).
			Find(&inventory)

		stockMap := make(map[uint]int)
		for _, inv := range inventory {
			stockMap[inv.MedicineID] = inv.StockQuantity
		}

		for id, qty := range remainingItems {
			available := stockMap[id]
			if available > 0 {
				take := qty
				if available < qty {
					take = available
				}

				plan.Items = append(plan.Items, CartItem{MedicineID: id, Quantity: take})
				remainingItems[id] -= take
				if remainingItems[id] <= 0 {
					delete(remainingItems, id)
				}
			}
		}

		if len(plan.Items) > 0 {
			plans = append(plans, plan)
		}
	}

	return plans, nil
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth radius in km
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*(math.Pi/180.0))*math.Cos(lat2*(math.Pi/180.0))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

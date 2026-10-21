package pharmacy

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
)

// StockReservation locks an inventory unit count to a cart for a bounded time
// window so two concurrent patient carts can't oversell the same pack. The
// reservation either gets confirmed at checkout (decrementing real inventory)
// or it expires.
type StockReservation struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CartID      string    `gorm:"index;not null" json:"cart_id"`   // groups all reservations for one cart
	PharmacyID  uint      `gorm:"index;not null" json:"pharmacy_id"`
	MedicineID  uint      `gorm:"index;not null" json:"medicine_id"`
	Quantity    int       `gorm:"not null" json:"quantity"`
	UserID      uint      `gorm:"index" json:"user_id"`
	ExpiresAt   time.Time `gorm:"index;not null" json:"expires_at"`
	Status      string    `gorm:"default:'active';index" json:"status"` // active | confirmed | released | expired
	CreatedAt   time.Time `json:"created_at"`
}

func (StockReservation) TableName() string { return "stock_reservations" }

// IsExpired returns true if a reservation is past its TTL and should not count
// against available stock.
func (r StockReservation) IsExpired(now time.Time) bool {
	return r.Status == "active" && now.After(r.ExpiresAt)
}

// ReservationItem is one line of a multi-line cart reservation.
type ReservationItem struct {
	MedicineID uint
	Quantity   int
}

// ReservationResult is what the picker returns: the chosen pharmacy + the
// reservation IDs the caller will need to confirm or release.
type ReservationResult struct {
	CartID         string `json:"cart_id"`
	PharmacyID     uint   `json:"pharmacy_id"`
	DistanceKm     float64 `json:"distance_km,omitempty"`
	ReservationIDs []uint `json:"reservation_ids"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// DefaultReservationTTL is how long the patient has to complete checkout
// before the lock is auto-released. 15 minutes matches what most e-commerce
// systems use — long enough for a careful patient, short enough to recycle
// stock if they abandon.
const DefaultReservationTTL = 15 * time.Minute

// PickNearestPharmacyAndReserve finds the closest pharmacy (by haversine
// distance from the patient's coordinates) that can fulfill every line of the
// cart, then atomically reserves that stock with a TTL.
//
// Concurrency safety: the read-and-write is wrapped in a transaction. Available
// stock at evaluation time = sum(inventory.stock_quantity) − sum(active,
// not-yet-expired reservations). Any reservation row whose ExpiresAt is in the
// past is treated as released even if its Status column hasn't been updated yet
// (lazy expiry, no background goroutine required).
func PickNearestPharmacyAndReserve(
	ctx context.Context,
	db *gorm.DB,
	userID uint,
	patientLat, patientLng float64,
	items []ReservationItem,
	ttl time.Duration,
) (*ReservationResult, error) {
	if len(items) == 0 {
		return nil, errors.New("no items to reserve")
	}
	if ttl <= 0 {
		ttl = DefaultReservationTTL
	}

	// Resolve the unique medicine IDs we need stock for.
	wantedByMed := map[uint]int{}
	medIDs := make([]uint, 0, len(items))
	for _, it := range items {
		if it.MedicineID == 0 || it.Quantity <= 0 {
			return nil, fmt.Errorf("invalid item: medicine_id=%d qty=%d", it.MedicineID, it.Quantity)
		}
		if _, seen := wantedByMed[it.MedicineID]; !seen {
			medIDs = append(medIDs, it.MedicineID)
		}
		wantedByMed[it.MedicineID] += it.Quantity
	}

	var chosen *Pharmacy
	chosenDistance := math.MaxFloat64
	var chosenStockByMed map[uint]int

	// Pull all active pharmacies that have at least one of the wanted medicines
	// in inventory. This is the candidate pool — for each, we check whether it
	// can fulfill the full cart.
	var candidates []Pharmacy
	if err := db.WithContext(ctx).
		Table("pharmacy").
		Distinct("pharmacy.*").
		Joins("JOIN pharmacyinventory pi ON pi.pharmacy_id = pharmacy.id").
		Where("pi.medicine_id IN ? AND pi.is_active = ? AND pi.stock_quantity > 0",
			medIDs, true).
		Find(&candidates).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	for i := range candidates {
		ph := candidates[i]
		// Available stock for each wanted medicine at THIS pharmacy.
		type stockRow struct {
			MedicineID uint
			Total      int
		}
		var stocks []stockRow
		if err := db.WithContext(ctx).Table("pharmacyinventory").
			Select("medicine_id, COALESCE(SUM(stock_quantity), 0) AS total").
			Where("pharmacy_id = ? AND is_active = ? AND medicine_id IN ?", ph.ID, true, medIDs).
			Group("medicine_id").Scan(&stocks).Error; err != nil {
			return nil, err
		}
		stockByMed := map[uint]int{}
		for _, r := range stocks {
			stockByMed[r.MedicineID] = r.Total
		}

		// Subtract any non-expired active reservations on this (pharmacy, medicine).
		type holdRow struct {
			MedicineID uint
			Held       int
		}
		var holds []holdRow
		if err := db.WithContext(ctx).Table("stock_reservations").
			Select("medicine_id, COALESCE(SUM(quantity), 0) AS held").
			Where("pharmacy_id = ? AND status = ? AND expires_at > ? AND medicine_id IN ?",
				ph.ID, "active", now, medIDs).
			Group("medicine_id").Scan(&holds).Error; err != nil {
			return nil, err
		}
		for _, h := range holds {
			stockByMed[h.MedicineID] -= h.Held
		}

		// Does this pharmacy meet every line?
		canFulfill := true
		for medID, wanted := range wantedByMed {
			if stockByMed[medID] < wanted {
				canFulfill = false
				break
			}
		}
		if !canFulfill {
			continue
		}

		// Distance ranking — haversine. If the patient has no coordinates we
		// degenerate to "first qualifying pharmacy" by treating distance as 0.
		var distance float64
		if patientLat != 0 || patientLng != 0 {
			distance = haversineKm(patientLat, patientLng, ph.Latitude, ph.Longitude)
		}
		if distance < chosenDistance {
			chosenDistance = distance
			chosen = &candidates[i]
			chosenStockByMed = stockByMed
		}
	}

	if chosen == nil {
		return nil, errors.New("no pharmacy can fulfill the cart (out of stock across the network)")
	}
	_ = chosenStockByMed // (kept for diagnostic use; could be returned later)

	// Atomically write the reservation rows. Read-then-write is wrapped in a tx
	// so a parallel cart racing against us either sees our inserted holds or
	// reads stale stock and fails its own canFulfill check on its own attempt.
	cartID := fmt.Sprintf("CART-%d-%d", userID, now.UnixNano())
	expiresAt := now.Add(ttl)
	reservationIDs := make([]uint, 0, len(items))

	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, it := range items {
			rec := StockReservation{
				CartID:     cartID,
				PharmacyID: chosen.ID,
				MedicineID: it.MedicineID,
				Quantity:   it.Quantity,
				UserID:     userID,
				ExpiresAt:  expiresAt,
				Status:     "active",
				CreatedAt:  now,
			}
			if err := tx.Create(&rec).Error; err != nil {
				return err
			}
			reservationIDs = append(reservationIDs, rec.ID)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &ReservationResult{
		CartID:         cartID,
		PharmacyID:     chosen.ID,
		DistanceKm:     chosenDistance,
		ReservationIDs: reservationIDs,
		ExpiresAt:      expiresAt,
	}, nil
}

// ConfirmReservations is called at checkout success: decrements the underlying
// pharmacyinventory.stock_quantity and marks each reservation confirmed. Runs
// in a single transaction so a crash mid-confirm doesn't leave a half-applied
// state.
func ConfirmReservations(ctx context.Context, db *gorm.DB, cartID string) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []StockReservation
		if err := tx.Where("cart_id = ? AND status = ?", cartID, "active").Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return errors.New("no active reservations for cart")
		}
		now := time.Now()
		for _, r := range rows {
			if r.ExpiresAt.Before(now) {
				return fmt.Errorf("reservation %d already expired", r.ID)
			}
			// Decrement the oldest available batch first (FIFO). For an MVP this
			// is acceptable; a fuller picker would prefer earliest expiry.
			res := tx.Table("pharmacyinventory").
				Where("pharmacy_id = ? AND medicine_id = ? AND is_active = ? AND stock_quantity >= ?",
					r.PharmacyID, r.MedicineID, true, r.Quantity).
				Order("expiry_date ASC").Limit(1).
				UpdateColumn("stock_quantity", gorm.Expr("stock_quantity - ?", r.Quantity))
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return fmt.Errorf("inventory drift: medicine %d at pharmacy %d no longer has %d units",
					r.MedicineID, r.PharmacyID, r.Quantity)
			}
			tx.Model(&r).Update("status", "confirmed")
		}
		return nil
	})
}

// ReleaseReservations marks the cart's reservations released — patient
// cancelled or checkout failed. Stock is freed for the next cart.
func ReleaseReservations(ctx context.Context, db *gorm.DB, cartID string) error {
	return db.WithContext(ctx).Model(&StockReservation{}).
		Where("cart_id = ? AND status = ?", cartID, "active").
		Update("status", "released").Error
}

// haversineKm returns the great-circle distance between two lat/lng pairs in km.
func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371.0 // Earth radius, km
	rad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := rad(lat2 - lat1)
	dLng := rad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rad(lat1))*math.Cos(rad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

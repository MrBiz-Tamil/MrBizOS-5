package emr

import (
	"context"
	"time"

	"gorm.io/gorm"
)

const (
	RefillReminderWindow = 3 * 24 * time.Hour
	ReminderCooldown     = 24 * time.Hour
)

// AdherenceStats is one item's adherence picture: how many of the doctor's
// authorised refills were actually used and whether the latest supply is
// overdue. Used by the patient + doctor dashboards.
type AdherenceStats struct {
	ItemID           uint       `json:"item_id"`
	MedicineID       uint       `json:"medicine_id"`
	RefillsAllowed   int        `json:"refills_allowed"`
	RefillsUsed      int        `json:"refills_used"`
	AdherencePercent float64    `json:"adherence_percent"` // refills_used / refills_allowed × 100
	LastRefilledAt   *time.Time `json:"last_refilled_at,omitempty"`
	NextRefillDueAt  *time.Time `json:"next_refill_due_at,omitempty"`
	IsOverdue        bool       `json:"is_overdue"`
}

func computeAdherence(item PrescriptionItem, now time.Time) AdherenceStats {
	pct := 0.0
	if item.RefillsAllowed > 0 {
		pct = float64(item.RefillsUsed) / float64(item.RefillsAllowed) * 100
	}
	overdue := false
	if item.NextRefillDueAt != nil && item.NextRefillDueAt.Before(now) &&
		item.RefillsUsed < item.RefillsAllowed {
		overdue = true
	}
	return AdherenceStats{
		ItemID:           item.ID,
		MedicineID:       item.MedicineID,
		RefillsAllowed:   item.RefillsAllowed,
		RefillsUsed:      item.RefillsUsed,
		AdherencePercent: pct,
		LastRefilledAt:   item.LastRefilledAt,
		NextRefillDueAt:  item.NextRefillDueAt,
		IsOverdue:        overdue,
	}
}

// RefillDueItem is one row returned by the sweep — the worker side reads these
// and enqueues a WhatsApp/email nudge to the owning patient.
type RefillDueItem struct {
	ItemID         uint
	PrescriptionID uint
	PatientID      uint
	MedicineID     uint
	NextDueAt      time.Time
}

// SweepRefillsDue returns every PrescriptionItem whose next refill is within the
// reminder window and hasn't been nudged inside the cooldown. Designed to run on
// a daily cron.
func SweepRefillsDue(ctx context.Context, db *gorm.DB) ([]RefillDueItem, error) {
	now := time.Now()
	deadline := now.Add(RefillReminderWindow)
	cutoff := now.Add(-ReminderCooldown)

	type row struct {
		ItemID         uint
		PrescriptionID uint
		PatientID      uint
		MedicineID     uint
		NextDueAt      time.Time
	}
	var rows []row
	if err := db.WithContext(ctx).
		Table("prescription_items pi").
		Select(`pi.id AS item_id, pi.prescription_id, p.user_id AS patient_id,
		         pi.medicine_id, pi.next_refill_due_at AS next_due_at`).
		Joins("JOIN prescriptions p ON p.id = pi.prescription_id").
		Where(`pi.deleted_at IS NULL
			AND pi.refills_used < pi.refills_allowed
			AND pi.next_refill_due_at IS NOT NULL
			AND pi.next_refill_due_at <= ?
			AND (pi.last_reminder_sent_at IS NULL OR pi.last_reminder_sent_at <= ?)`,
			deadline, cutoff).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	// Stamp last_reminder_sent_at so a second sweep within the cooldown skips
	// these items — even if the caller hasn't dispatched the notifications yet.
	ids := make([]uint, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ItemID)
	}
	if err := db.WithContext(ctx).Table("prescription_items").
		Where("id IN ?", ids).Update("last_reminder_sent_at", now).Error; err != nil {
		return nil, err
	}

	out := make([]RefillDueItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, RefillDueItem(r))
	}
	return out, nil
}

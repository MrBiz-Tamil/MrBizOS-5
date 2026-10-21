package emr

import (
	"context"
	"drayya-core/internal/database"
	"drayya-core/internal/utils/response"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// RefillReminderWindow is how early we nudge the patient before their next dose
// run is due. 3 days lets a courier delivery arrive before the current supply
// runs out.
const RefillReminderWindow = 3 * 24 * time.Hour

// ReminderCooldown prevents the sweep from spamming the same line — once a
// reminder fires, we won't send another for this many hours regardless of
// status, so multiple sweeps per day stay safe.
const ReminderCooldown = 20 * time.Hour

// RecordRefillOrder is called when a patient places a follow-up order for an
// existing PrescriptionItem with refills remaining. It increments the used
// counter and bumps the next-due date forward by the original duration.
//
// POST /prescriptions/items/:id/refill
func RecordRefillOrder(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid prescription item id")
	}
	userID, _ := c.Locals("user_id").(uint)
	if userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	var item PrescriptionItem
	if err := database.DB.WithContext(c.Context()).First(&item, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Prescription item not found")
	}

	// Ownership check: the patient must own the parent prescription. Admins
	// bypass — they may dispense on behalf of the patient.
	role, _ := c.Locals("role").(string)
	if role != "admin" && role != "super_admin" && role != "pharmacist" && role != "pharmacy_admin" {
		var ownerID uint
		database.DB.Table("prescriptions").Where("id = ?", item.PrescriptionID).
			Select("user_id").Scan(&ownerID)
		if ownerID != userID {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Not your prescription")
		}
	}

	if item.RefillsUsed >= item.RefillsAllowed {
		return response.Error(c, fiber.StatusConflict, "NO_REFILLS",
			"No refills remaining on this prescription item; a new consultation is required")
	}

	now := time.Now()
	dur := time.Duration(item.DurationDays) * 24 * time.Hour
	if dur <= 0 {
		dur = 30 * 24 * time.Hour // fallback: month-long supply
	}
	next := now.Add(dur)
	updates := map[string]interface{}{
		"refills_used":      item.RefillsUsed + 1,
		"last_refilled_at":  now,
		"next_refill_due_at": next,
	}
	if err := database.DB.Model(&item).Updates(updates).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"id":                id,
		"refills_used":      item.RefillsUsed + 1,
		"refills_allowed":   item.RefillsAllowed,
		"next_refill_due_at": next,
	}, "Refill recorded")
}

// AdherenceStats is one item's adherence picture: how many of the doctor's
// authorised refills were actually used and whether the latest supply is
// overdue. Used by the patient + doctor dashboards.
type AdherenceStats struct {
	ItemID             uint       `json:"item_id"`
	MedicineID         uint       `json:"medicine_id"`
	RefillsAllowed     int        `json:"refills_allowed"`
	RefillsUsed        int        `json:"refills_used"`
	AdherencePercent   float64    `json:"adherence_percent"` // refills_used / refills_allowed × 100
	LastRefilledAt     *time.Time `json:"last_refilled_at,omitempty"`
	NextRefillDueAt    *time.Time `json:"next_refill_due_at,omitempty"`
	IsOverdue          bool       `json:"is_overdue"`
}

// GetItemAdherence returns adherence for a single prescription item.
// GET /prescriptions/items/:id/adherence
func GetItemAdherence(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid prescription item id")
	}
	var item PrescriptionItem
	if err := database.DB.WithContext(c.Context()).First(&item, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Prescription item not found")
	}
	stats := computeAdherence(item, time.Now())
	return response.JSON(c, fiber.StatusOK, stats, "Adherence fetched")
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
// a daily cron — multiple invocations on the same day are safe because of the
// LastReminderSentAt cooldown gate.
//
// Pure read + bulk update for the cooldown stamp; the caller fans out the
// notifications themselves so this function doesn't depend on the notification
// transport (keeps it unit-testable).
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

// SweepRefillsDueHandler exposes the sweep over HTTP so an external cron (or
// admin) can trigger it. Returns the items it stamped without sending the
// notifications — those go through the existing notification outbox.
//
// POST /admin/pharmacy/refills/sweep
func SweepRefillsDueHandler(c *fiber.Ctx) error {
	role, _ := c.Locals("role").(string)
	if role != "admin" && role != "super_admin" {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Admin only")
	}
	items, err := SweepRefillsDue(c.Context(), database.DB)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SWEEP_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"due_count": len(items),
		"items":     items,
	}, "Refill reminders ready to dispatch")
}

// Sanity helper used at startup: ensures the column exists before the sweep
// runs (older databases may not have migrated yet). Returns a non-nil error
// only if the migration didn't apply.
var ErrRefillColumnsMissing = errors.New("refill columns missing — run AutoMigrate")

// GetPrescriptionItems returns the structured PrescriptionItem rows for one
// prescription so the patient UI can render the refill / adherence panel
// without re-deriving anything from the legacy `medicines` JSON. Owner-gated:
// the patient who received the Rx, the issuing doctor, and admins/pharmacists.
//
// GET /prescriptions/:id/items
func GetPrescriptionItems(c *fiber.Ctx) error {
	idStr := c.Params("id")
	pid, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || pid == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid prescription id")
	}
	callerID, _ := c.Locals("user_id").(uint)
	if callerID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	role, _ := c.Locals("role").(string)

	// Owner check using the columns Prescription already carries.
	type owner struct {
		DoctorID  uint
		PatientID uint
	}
	var o owner
	if err := database.DB.Table("prescriptions").Where("id = ?", pid).
		Select("doctor_id, user_id AS patient_id").Scan(&o).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Prescription not found")
	}
	if role != "admin" && role != "super_admin" && role != "pharmacist" && role != "pharmacy_admin" &&
		callerID != o.DoctorID && callerID != o.PatientID {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Not your prescription")
	}

	var items []PrescriptionItem
	if err := database.DB.Where("prescription_id = ?", pid).Order("id").Find(&items).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, items, "Prescription items fetched")
}

package cron

import (
	"context"
	"log"

	"drayya-core/internal/audit"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/pharmacy"
)

// ResetDailyDoctorCounters resets free consultation counters for all doctors.
// Should be called daily at midnight.
func ResetDailyDoctorCounters() {
	log.Println("[Cron] Resetting daily doctor consultation counters...")
	ctx := audit.ForSystem(context.Background(), "cron:daily_doctor_reset")

	err := database.DB.WithContext(ctx).Model(&identity.DoctorProfile{}).
		Where("1 = 1").
		Updates(map[string]interface{}{
			"free_consults_given_today": 0,
			"total_consultations_today": 0,
		}).Error

	if err != nil {
		log.Printf("[Cron] ERROR: Failed to reset doctor counters: %v", err)
	} else {
		log.Println("[Cron] Successfully reset daily doctor counters.")
	}
}

// ResetDailyCourierCounters resets today_trips for all couriers at midnight and
// reconciles active_deliveries against the actual open delivery rows.
//
// Why reconciliation matters:
//   If a courier app crashes mid-delivery, UpdateDeliveryStatus("completed") is
//   never called, so active_deliveries stays elevated forever. The watchdog
//   cross-checks the counter against real open rows and corrects any drift.
func ResetDailyCourierCounters() {
	if database.DB == nil {
		return
	}
	log.Println("[Cron] Resetting daily courier trip counters + reconciling active_deliveries...")
	ctx := audit.ForSystem(context.Background(), "cron:daily_courier_reset")
	db := database.DB.WithContext(ctx)

	// 1. Reset today_trips for all couriers
	if err := db.Model(&pharmacy.CourierProfile{}).
		Where("1 = 1").
		Update("today_trips", 0).Error; err != nil {
		log.Printf("[Cron] ERROR: Failed to reset courier today_trips: %v", err)
	}

	// 2. Reconcile active_deliveries: set it to the actual count of open deliveries
	//    for each courier. Uses a correlated subquery so it's one round-trip.
	const reconcileSQL = `
		UPDATE courier_profiles cp
		SET active_deliveries = (
			SELECT COUNT(*)
			FROM courier_deliveries cd
			WHERE cd.courier_id = cp.user_id
			  AND cd.status NOT IN ('completed', 'failed', 'cancelled')
		)
	`
	if err := db.Exec(reconcileSQL).Error; err != nil {
		log.Printf("[Cron] ERROR: Failed to reconcile active_deliveries: %v", err)
	} else {
		log.Println("[Cron] Courier counters reset and active_deliveries reconciled.")
	}
}

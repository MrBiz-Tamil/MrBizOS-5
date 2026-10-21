package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/queue"
)

const (
	maxDispatchRetries = 3

	// Tiered connect policy — different urgency, different patience. When a live
	// request goes unanswered past its Escalate window it is re-broadcast (and,
	// for SOS, chiefs are alerted). Past its Final window it is auto-cancelled +
	// refunded as the last-resort safety net.
	sosEscalateAfter     = 30 * time.Second
	sosFinalTimeout      = 5 * time.Minute
	instantEscalateAfter = 90 * time.Second
	instantFinalTimeout  = 10 * time.Minute

	// Selected doctor: if a booked consultation is still pending more than this
	// past its scheduled start, the chosen doctor is a no-show — offer the patient
	// the next available doctor instead.
	selectedNoShowAfter = 2 * time.Minute

	// The escalation sweep must tick faster than the SOS window or 30s is meaningless.
	connectSweepInterval = 10 * time.Second
)

func StartWorker() {
	go runStreamWorker()
	go runConnectPolicySweep()
}

// runStreamWorker reads from the Redis stream and dispatches appointments.
// On failure it increments a retry counter; after maxDispatchRetries it pushes
// the item to the DLQ (appointment:dlq Redis list) and ACKs the stream entry
// so it does not block other messages indefinitely.
func runStreamWorker() {
	log.Println("[Dispatch] Autopilot started (Redis Streams)")

	ctx := context.Background()
	queue.EnsureStreamGroup(ctx)
	// Unique per-process so multiple worker replicas (or a quick restart) don't
	// share a consumer identity in the Redis Streams group, which would corrupt
	// the pending-entries list and silently swap ownership of in-flight messages.
	// HOSTNAME is set by Docker/k8s/ECS; we fall back to the PID for bare-metal.
	consumerName := os.Getenv("HOSTNAME")
	if consumerName == "" {
		consumerName = fmt.Sprintf("worker-%d", os.Getpid())
	}
	log.Printf("[Dispatch] consumer name: %s", consumerName)

	for {
		messages, err := queue.ReadStream(ctx, consumerName)
		if err != nil {
			log.Printf("[Dispatch] Stream read error: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		if messages == nil {
			continue
		}

		for _, msg := range messages {
			ok := ProcessStreamItem(ctx, msg)
			if ok {
				queue.AckStream(ctx, msg.ID)
				delete(retryMap, msg.ID) // clean up retry counter on success
			} else {
				retries := retryMap[msg.ID]
				if retries >= maxDispatchRetries {
					log.Printf("[Dispatch] Message %s exceeded %d retries → DLQ", msg.ID, maxDispatchRetries)
					if database.Rdb != nil {
						database.Rdb.LPush(ctx, "appointment:dlq", fmt.Sprintf("%v", msg.Values))
					}
					queue.AckStream(ctx, msg.ID)
					delete(retryMap, msg.ID)
				} else {
					retryMap[msg.ID] = retries + 1
					log.Printf("[Dispatch] Message %s failed (attempt %d/%d), will retry", msg.ID, retries+1, maxDispatchRetries)
					// Leave un-ACKed — stream re-delivers on next consumer group read
				}
			}
		}
	}
}

// retryMap tracks per-message retry counts (in-memory; sufficient for single-pod).
var retryMap = map[string]int{}

// runConnectPolicySweep enforces the tiered connect policy on live (instant/SOS)
// requests: re-broadcast unanswered ones past their escalate window, then
// auto-cancel + refund past their final window. Ticks fast so the 30s SOS
// window is real.
func runConnectPolicySweep() {
	log.Println("[Dispatch] Tiered connect-policy sweep started")
	for {
		time.Sleep(connectSweepInterval)
		ctx := context.Background()
		escalateUnanswered(ctx, "emergency", sosEscalateAfter, true)
		escalateUnanswered(ctx, "instant", instantEscalateAfter, false)
		offerFallbackForNoShow(ctx)
		cancelTimedOut(ctx, "emergency", sosFinalTimeout)
		cancelTimedOut(ctx, "instant", instantFinalTimeout)
	}
}

// offerFallbackForNoShow signals the patient when their chosen doctor hasn't
// accepted a booked consultation by selectedNoShowAfter past its scheduled
// start. The patient's screen can then offer "connect to the next available
// doctor now". We only signal (escalated_at) — we never auto-cancel a booking.
func offerFallbackForNoShow(ctx context.Context) {
	if database.DB == nil {
		return
	}
	cutoff := time.Now().Add(-selectedNoShowAfter)
	var appts []appointment.Appointment
	if err := database.DB.WithContext(ctx).
		Where("appointment_type = ? AND status = ? AND doctor_id IS NOT NULL AND escalated_at IS NULL AND start_time < ?",
			"consultation", "pending", cutoff).
		Limit(50).Find(&appts).Error; err != nil {
		log.Printf("[Dispatch] No-show query error: %v", err)
		return
	}
	now := time.Now()
	for _, appt := range appts {
		docID := uint(0)
		if appt.DoctorID != nil {
			docID = *appt.DoctorID
		}
		publishRoom(ctx, fmt.Sprintf("room:user:%d", appt.PatientID), map[string]interface{}{
			"type": "selected_doctor_unavailable",
			"metadata": map[string]interface{}{
				"appointment_id":    appt.ID,
				"doctor_id":         docID,
				"consultation_mode": appt.ConsultationMode,
			},
		})
		database.DB.WithContext(ctx).Model(&appointment.Appointment{}).
			Where("id = ?", appt.ID).Update("escalated_at", now)
		log.Printf("[Dispatch] Selected doctor no-show on appt #%d — offered fallback to patient", appt.ID)
	}
}

// escalateUnanswered re-broadcasts live requests that have waited past their
// escalate window and haven't been escalated yet. SOS also alerts the chiefs.
func escalateUnanswered(ctx context.Context, apptType string, after time.Duration, alertChiefs bool) {
	if database.DB == nil {
		return
	}
	cutoff := time.Now().Add(-after)
	var appts []appointment.Appointment
	if err := database.DB.WithContext(ctx).
		Where("appointment_type = ? AND status = ? AND escalated_at IS NULL AND created_at < ?",
			apptType, "pending", cutoff).
		Limit(50).Find(&appts).Error; err != nil {
		log.Printf("[Dispatch] Escalation query error (%s): %v", apptType, err)
		return
	}
	now := time.Now()
	for _, appt := range appts {
		// Re-broadcast louder to every online doctor — the WaitingCallDashboard
		// surfaces escalated requests at the top.
		publishRoom(ctx, "room:doctors:instant", map[string]interface{}{
			"type": "instant_call_escalated",
			"metadata": map[string]interface{}{
				"appointment_id":    appt.ID,
				"consultation_mode": appt.ConsultationMode,
				"type":              appt.AppointmentType,
				"waited_seconds":    int(now.Sub(appt.CreatedAt).Seconds()),
				"priority":          apptType == "emergency",
			},
		})
		if alertChiefs {
			notifyChiefs(ctx,
				fmt.Sprintf("[SOS] Emergency request #%d unanswered for %v — escalating to all doctors", appt.ID, after),
				map[string]interface{}{"appointment_id": appt.ID, "reason": "sos_unanswered"},
			)
		}
		database.DB.WithContext(ctx).Model(&appointment.Appointment{}).
			Where("id = ?", appt.ID).Update("escalated_at", now)
		log.Printf("[Dispatch] Escalated %s appt #%d (waited >%v)", apptType, appt.ID, after)
	}
}

// cancelTimedOut auto-cancels + refunds live requests still pending past their
// final window — the last-resort safety net so a patient's funds never stay
// locked when no doctor is ever found.
func cancelTimedOut(ctx context.Context, apptType string, after time.Duration) {
	if database.DB == nil {
		return
	}
	cutoff := time.Now().Add(-after)
	var stuck []appointment.Appointment
	if err := database.DB.WithContext(ctx).
		Where("appointment_type = ? AND status = ? AND created_at < ?", apptType, "pending", cutoff).
		Limit(20).Find(&stuck).Error; err != nil {
		log.Printf("[Dispatch] Final-timeout query error (%s): %v", apptType, err)
		return
	}
	for _, appt := range stuck {
		apptIDStr := fmt.Sprintf("%d", appt.ID)
		repo := appointment.NewDefaultRepository()
		financeRepo := finance.NewRepository(database.DB)
		financeSvc := finance.NewService(financeRepo)
		svc := appointment.NewService(repo, financeSvc)

		if err := svc.CancelAppointment(apptIDStr, "auto-cancelled: no doctor available within timeout"); err != nil {
			log.Printf("[Dispatch] Auto-cancel failed for %s appt #%d: %v", apptType, appt.ID, err)
			continue
		}
		log.Printf("[Dispatch] Auto-cancelled %s appt #%d after %v (no doctor)", apptType, appt.ID, after)
		// Tell the patient their request timed out so the waiting screen can react.
		publishRoom(ctx, fmt.Sprintf("room:user:%d", appt.PatientID), map[string]interface{}{
			"type":     "connect_timed_out",
			"metadata": map[string]interface{}{"appointment_id": appt.ID, "appointment_type": apptType},
		})
		notifyChiefs(ctx,
			fmt.Sprintf("[Alert] Auto-cancelled %s appt #%d — no doctor found in %v", apptType, appt.ID, after),
			map[string]interface{}{"appointment_id": appt.ID, "reason": "no_doctor_timeout"},
		)
	}
}

// publishRoom sends a WS event to a room via the shared chat:global Redis channel
// (the same mechanism the appointment handler uses).
func publishRoom(ctx context.Context, room string, payload map[string]interface{}) {
	if database.Rdb == nil {
		return
	}
	payload["room_id"] = room
	if data, err := json.Marshal(payload); err == nil {
		database.Rdb.Publish(ctx, "chat:global", data)
	}
}

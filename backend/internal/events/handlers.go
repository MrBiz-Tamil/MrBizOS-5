package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"drayya-core/internal/database"
	"drayya-core/internal/services/email"
	"drayya-core/internal/services/notification"
)

// DefaultHandler is the main event processor. Wire it to Subscribe in server startup.
func DefaultHandler(ctx context.Context, evt DomainEvent) error {
	log.Printf("[events] processing %s actor=%d id=%s", evt.Type, evt.ActorID, evt.ID)

	switch evt.Type {
	case EventDoctorOnline, EventDoctorOffline, EventDoctorPing:
		return handleDoctorPresence(ctx, evt)
	case EventAppointmentAccepted:
		return handleAppointmentAccepted(ctx, evt)
	case EventAppointmentCompleted:
		return handleAppointmentCompleted(ctx, evt)
	case EventAppointmentCancelled:
		return handleAppointmentCancelled(ctx, evt)
	case EventPayoutTriggered:
		return handlePayoutTriggered(ctx, evt)
	case EventUserRegistered:
		return handleUserRegistered(ctx, evt)
	default:
		// Unknown event types are logged and discarded (not retried)
		log.Printf("[events] unhandled event type: %s", evt.Type)
		return nil
	}
}

// userRow holds the minimal fields we fetch for push/email delivery.
type userRow struct {
	ID       uint
	Email    string
	FullName string
	FCMToken string
}

// lookupUser fetches minimal user fields from the DB without importing the identity package.
// Returns nil if the user is not found or ID is zero.
func lookupUser(id uint) *userRow {
	if id == 0 || database.DB == nil {
		return nil
	}
	var u userRow
	err := database.DB.
		Table("user").
		Select("id, email, full_name, fcm_token").
		Where("id = ?", id).
		Scan(&u).Error
	if err != nil || u.ID == 0 {
		return nil
	}
	return &u
}

// payloadUint extracts a uint from a map[string]interface{} payload field.
func payloadUint(p map[string]interface{}, key string) uint {
	v, ok := p[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return uint(n)
	case uint:
		return n
	case int:
		return uint(n)
	}
	return 0
}

func handleDoctorPresence(ctx context.Context, evt DomainEvent) error {
	doctorID := fmt.Sprintf("%v", evt.ActorID)
	log.Printf("[events] doctor presence change: doctorID=%s type=%s", doctorID, evt.Type)
	// Presence changes are operational signals — no patient push needed.
	// The real-time dashboard reads the doctor-online Redis key directly.
	return nil
}

func handleAppointmentAccepted(ctx context.Context, evt DomainEvent) error {
	apptID := payloadUint(evt.Payload, "appointment_id")
	patientID := payloadUint(evt.Payload, "patient_id")
	doctorName, _ := evt.Payload["doctor_name"].(string)
	startTime, _ := evt.Payload["start_time"].(string)

	log.Printf("[events] appointment accepted: appt=%d patient=%d", apptID, patientID)

	patient := lookupUser(patientID)
	if patient != nil && patient.FCMToken != "" {
		body := fmt.Sprintf("Your appointment has been confirmed with Dr. %s", doctorName)
		if startTime != "" {
			body = fmt.Sprintf("Your appointment with Dr. %s is confirmed for %s", doctorName, startTime)
		}
		go func() {
			if err := notification.SendToToken(patient.FCMToken, notification.FCMMessage{
				Title: "Appointment Confirmed ✅",
				Body:  body,
				Data: map[string]string{
					"type":           "appointment_accepted",
					"appointment_id": fmt.Sprintf("%d", apptID),
				},
			}); err != nil {
				log.Printf("[events] FCM appointment_accepted patient=%d: %v", patientID, err)
			}
		}()
	}

	return nil
}

func handleAppointmentCompleted(ctx context.Context, evt DomainEvent) error {
	apptID := payloadUint(evt.Payload, "appointment_id")
	patientID := payloadUint(evt.Payload, "patient_id")
	doctorName, _ := evt.Payload["doctor_name"].(string)

	log.Printf("[events] appointment completed: appt=%d", apptID)

	patient := lookupUser(patientID)
	if patient != nil && patient.FCMToken != "" {
		body := "How was your consultation? Rate your experience."
		if doctorName != "" {
			body = fmt.Sprintf("How was your consultation with Dr. %s? Tap to rate.", doctorName)
		}
		go func() {
			if err := notification.SendToToken(patient.FCMToken, notification.FCMMessage{
				Title: "Rate Your Consultation ⭐",
				Body:  body,
				Data: map[string]string{
					"type":           "rating_request",
					"appointment_id": fmt.Sprintf("%d", apptID),
				},
			}); err != nil {
				log.Printf("[events] FCM rating_request patient=%d: %v", patientID, err)
			}
		}()
	}

	return nil
}

func handleAppointmentCancelled(ctx context.Context, evt DomainEvent) error {
	apptID := payloadUint(evt.Payload, "appointment_id")
	patientID := payloadUint(evt.Payload, "patient_id")
	reason, _ := evt.Payload["reason"].(string)

	log.Printf("[events] appointment cancelled: appt=%d", apptID)

	patient := lookupUser(patientID)
	if patient != nil && patient.FCMToken != "" {
		body := "Your appointment has been cancelled."
		if reason != "" {
			body = fmt.Sprintf("Your appointment has been cancelled. Reason: %s", reason)
		}
		go func() {
			if err := notification.SendToToken(patient.FCMToken, notification.FCMMessage{
				Title: "Appointment Cancelled",
				Body:  body,
				Data: map[string]string{
					"type":           "appointment_cancelled",
					"appointment_id": fmt.Sprintf("%d", apptID),
				},
			}); err != nil {
				log.Printf("[events] FCM appointment_cancelled patient=%d: %v", patientID, err)
			}
		}()
	}

	return nil
}

func handlePayoutTriggered(ctx context.Context, evt DomainEvent) error {
	doctorID := payloadUint(evt.Payload, "doctor_id")
	amount := evt.Payload["amount"]
	txnRef, _ := evt.Payload["txn_ref"].(string)

	log.Printf("[events] payout triggered: doctor=%d amount=%v ref=%s", doctorID, amount, txnRef)

	// Write finance audit trail entry to activity_logs table (fire-and-forget)
	go func() {
		details, _ := json.Marshal(map[string]interface{}{
			"amount":  amount,
			"txn_ref": txnRef,
		})
		if database.DB == nil {
			return
		}
		if err := database.DB.Exec(
			`INSERT INTO activity_logs (user_id, action, entity, entity_id, details, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, NOW(), NOW())`,
			doctorID, "payout_triggered", "payout", txnRef, string(details),
		).Error; err != nil {
			log.Printf("[events] payout audit log write failed: %v", err)
		}
	}()

	return nil
}

func handleUserRegistered(ctx context.Context, evt DomainEvent) error {
	userID := payloadUint(evt.Payload, "user_id")
	role, _ := evt.Payload["role"].(string)

	log.Printf("[events] user registered: user=%d role=%s", userID, role)

	user := lookupUser(userID)
	if user != nil && user.Email != "" {
		name := user.FullName
		if name == "" {
			name = "there"
		}
		notification.EnqueueEmail(ctx,
			user.Email,
			"Welcome to Dr.Ayya! 🏥",
			email.WelcomeEmail(name, role),
		)
	}

	return nil
}

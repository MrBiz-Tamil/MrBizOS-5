package devi

// flow_emergency.go — SOS detection and emergency appointment dispatch.
// Extracted from engine.go; behaviour is identical.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"drayya-core/internal/queue"
)

// emergencyAppt mirrors the appointments table for direct GORM writes.
// Kept here because the appointment service does not expose an emergency-create
// path that bypasses the normal booking flow.
type emergencyAppt struct {
	ID               uint      `gorm:"primaryKey"`
	PatientID        uint      `gorm:"column:patient_id"`
	FamilyMemberID   *uint     `gorm:"column:family_member_id"`
	Status           string    `gorm:"column:status"`
	AppointmentType  string    `gorm:"column:appointment_type"`
	ConsultationMode string    `gorm:"column:consultation_mode"`
	IsPriority       bool      `gorm:"column:is_priority"`
	Notes            string    `gorm:"column:notes"`
	StartTime        time.Time `gorm:"column:start_time"`
	EndTime          time.Time `gorm:"column:end_time"`
}

func (emergencyAppt) TableName() string { return "appointments" }

func (e *Engine) emergencyStart(ctx context.Context, uc *UserContext) FlowResult {
	uid := uint(0)
	if uc.UserID != nil {
		uid = *uc.UserID
	}

	caseRef := ""
	if uid > 0 && e.svc.DB != nil {
		appt := emergencyAppt{
			PatientID:        uid,
			Status:           "pending",
			AppointmentType:  "emergency",
			ConsultationMode: "video",
			IsPriority:       true,
			Notes:            "SOS triggered via Devi chat assistant",
			StartTime:        time.Now(),
			EndTime:          time.Now().Add(30 * time.Minute),
		}
		if uc.FamilyMemberID != nil {
			appt.FamilyMemberID = uc.FamilyMemberID
		}
		if err := e.svc.DB.WithContext(ctx).Create(&appt).Error; err == nil {
			caseRef = fmt.Sprintf("%d", appt.ID)
			meta := queue.AppointmentMetadata{
				AppointmentID:   int(appt.ID),
				PatientID:       int(uid),
				AppointmentType: "emergency",
				AISeverityScore: 10.0,
			}
			if err := queue.EnqueueStream(ctx, meta); err != nil {
				log.Printf("[Devi] SOS enqueue failed for appt %d: %v", appt.ID, err)
			}
			e.broadcastSOSAlert(ctx, appt.ID, uid)
		} else {
			log.Printf("[Devi] SOS appointment creation failed for user %d: %v", uid, err)
		}
	}

	msg := "🚨 *Emergency alert sent!*\n\n" +
		"📞 *Call 112 (National Emergency) or 108 (Ambulance) immediately.*\n\n"
	if caseRef != "" {
		msg += fmt.Sprintf("🆔 Case #%s\n✅ A doctor has been notified and will contact you shortly.\n\n", caseRef)
	} else {
		msg += "🆘 *Tap the SOS button in the app to dispatch a doctor to your location.*\n\n"
	}
	msg += "⚡ *While you wait:*\n" +
		"• Stay with the patient — do NOT leave them alone\n" +
		"• Keep them still and calm\n" +
		"• Do NOT give food or water\n\n" +
		"💙 *Stay calm — help is on the way.*"

	var actions []ChatAction
	if caseRef != "" {
		actions = append(actions, linkBtn("Open Case #"+caseRef, "deep_link:appointment:"+caseRef))
	}
	return FlowResult{Message: msg, Actions: actions, Done: true}
}

func (e *Engine) broadcastSOSAlert(ctx context.Context, apptID, patientID uint) {
	if e.svc.Redis == nil {
		return
	}
	payload := map[string]interface{}{
		"type":           "sos_alert",
		"appointment_id": apptID,
		"patient_id":     patientID,
		"message":        fmt.Sprintf("🚨 Emergency SOS — Patient #%d needs immediate help! (Case #%d)", patientID, apptID),
		"priority":       "critical",
	}
	data, _ := json.Marshal(payload)
	if err := e.svc.Redis.Publish(ctx, "chat:global", data).Err(); err != nil {
		log.Printf("[Devi] SOS broadcast failed: %v", err)
	}
}

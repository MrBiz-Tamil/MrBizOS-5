package v2

import (
	"context"
	"errors"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
)

// createBookingTool creates a new Appointment row. Destructive — money will
// be charged at the next step, the doctor's calendar fills up, and the
// patient gets reminders. Confirm gate is mandatory.
//
// Patient-only. The runtime will stash the proposed booking and ask the
// user "Confirm booking with Dr X on Y at Z?" before actually inserting.
var createBookingTool = Tool{
	Name: "create_booking",
	Description: "Create an appointment booking. Confirms the slot, marks " +
		"the appointment pending payment, and notifies the doctor. THIS IS " +
		"DESTRUCTIVE — the user MUST be told the doctor, time, and fee " +
		"before this is called. The runtime will then ask the user to " +
		"confirm before executing.",
	Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"doctor_id": map[string]any{
				"type":        "integer",
				"description": "Doctor's user ID from search_doctors.",
			},
			"start_time": map[string]any{
				"type":        "string",
				"description": "Slot start time as ISO 8601 (e.g. '2026-06-08T10:00:00+05:30').",
			},
			"mode": map[string]any{
				"type":        "string",
				"description": "One of 'video', 'chat', 'in_person'.",
			},
			"notes": map[string]any{
				"type":        "string",
				"description": "Optional reason for visit.",
			},
		},
		"required": []string{"doctor_id", "start_time", "mode"},
	},
	Allowed:       []PersonaRole{RolePatient},
	Confirm:       true, // ⚠ DESTRUCTIVE — gated
	MaxPerSession: 3,
	MaxPerMinute:  1,
	Execute:       executeCreateBooking,
}

func executeCreateBooking(ctx context.Context, args map[string]any) (any, error) {
	persona := PersonaFromContext(ctx)
	if persona.UserID == nil {
		return map[string]any{"error": "must be signed in to book"}, nil
	}
	if database.DB == nil {
		return nil, errors.New("database unavailable")
	}

	var doctorID uint
	if v, ok := args["doctor_id"].(float64); ok && v > 0 {
		doctorID = uint(v)
	}
	if doctorID == 0 {
		return map[string]any{"error": "doctor_id is required"}, nil
	}

	startStr, _ := args["start_time"].(string)
	startTime, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return map[string]any{"error": "start_time must be ISO 8601, e.g. 2026-06-08T10:00:00+05:30"}, nil
	}
	if startTime.Before(time.Now()) {
		return map[string]any{"error": "start_time must be in the future"}, nil
	}

	mode, _ := args["mode"].(string)
	if mode != "video" && mode != "chat" && mode != "in_person" {
		return map[string]any{"error": "mode must be one of: video, chat, in_person"}, nil
	}

	notes, _ := args["notes"].(string)

	// Fetch the doctor's fee. consultation_fees lives in a jsonb on
	// DoctorProfile; for now we look up the raw int64 INR value.
	var feePaise int64
	database.DB.Raw(`SELECT
		COALESCE((consultation_fees->>'INR')::bigint, 0)
		FROM doctorprofile WHERE user_id = ?`, doctorID).Scan(&feePaise)

	doctorIDPtr := doctorID
	appt := appointment.Appointment{
		PatientID:        *persona.UserID,
		DoctorID:         &doctorIDPtr,
		StartTime:        startTime,
		EndTime:          startTime.Add(30 * time.Minute),
		Status:           appointment.StatusPending,
		AppointmentType:  "consultation",
		ConsultationMode: mode,
		Notes:            notes,
		ConsultationFee:  feePaise,
	}
	if err := database.DB.WithContext(ctx).Create(&appt).Error; err != nil {
		return nil, err
	}

	return map[string]any{
		"appointment_id":     appt.ID,
		"status":             string(appt.Status),
		"start_time":         appt.StartTime.Format(time.RFC3339),
		"consultation_fee":   feePaise,
		"payment_url":        "/appointments/" + uintStr(appt.ID) + "/pay",
		"message":            "Appointment created. Complete payment to confirm your slot.",
	}, nil
}

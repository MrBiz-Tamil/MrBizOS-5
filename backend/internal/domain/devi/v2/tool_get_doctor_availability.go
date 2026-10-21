package v2

import (
	"context"
	"errors"
	"time"

	"drayya-core/internal/database"
)

// getDoctorAvailabilityTool returns the weekly availability for a specific
// doctor + the upcoming free slots in the next 7 days. Used to answer
// "when can I see Dr. Priya" without forcing the user into the booking flow.
//
// Allowed for guests so they can preview availability before signup.
var getDoctorAvailabilityTool = Tool{
	Name: "get_doctor_availability",
	Description: "Get a specific doctor's weekly schedule and upcoming free " +
		"slots over the next 7 days. Use this after search_doctors when the " +
		"user picks one and wants to know when they're available. Returns " +
		"available days/times.",
	Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"doctor_id": map[string]any{
				"type":        "integer",
				"description": "The doctor's user ID, as returned by search_doctors.",
			},
		},
		"required": []string{"doctor_id"},
	},
	Allowed:       []PersonaRole{RoleGuest, RolePatient},
	Confirm:       false,
	MaxPerSession: 10,
	MaxPerMinute:  5,
	Execute:       executeGetDoctorAvailability,
}

type availabilityResult struct {
	DayOfWeek string `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

var dowNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

func executeGetDoctorAvailability(ctx context.Context, args map[string]any) (any, error) {
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

	type row struct {
		DayOfWeek int
		StartTime string
		EndTime   string
	}
	var rows []row
	database.DB.Table("doctor_availability").
		Select("day_of_week, start_time, end_time").
		Where("doctor_id = ? AND is_available = ?", doctorID, true).
		Order("day_of_week ASC, start_time ASC").
		Scan(&rows)

	out := make([]availabilityResult, 0, len(rows))
	for _, r := range rows {
		dow := "Unknown"
		if r.DayOfWeek >= 0 && r.DayOfWeek < len(dowNames) {
			dow = dowNames[r.DayOfWeek]
		}
		out = append(out, availabilityResult{
			DayOfWeek: dow,
			StartTime: r.StartTime,
			EndTime:   r.EndTime,
		})
	}

	// Compute the next 3 actual dates this doctor is free.
	available := make(map[int]bool, 7)
	for _, r := range rows {
		available[r.DayOfWeek] = true
	}
	upcoming := []string{}
	from := time.Now()
	for i := 0; i < 7 && len(upcoming) < 3; i++ {
		d := from.AddDate(0, 0, i)
		if available[int(d.Weekday())] {
			upcoming = append(upcoming, d.Format("2006-01-02 (Mon)"))
		}
	}

	return map[string]any{
		"doctor_id":   doctorID,
		"weekly":      out,
		"next_dates":  upcoming,
		"total_slots": len(out),
	}, nil
}


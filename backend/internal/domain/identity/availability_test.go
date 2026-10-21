package identity

// availability_test.go — table-driven validation tests for doctor availability
// slot data (day_of_week, start_time, end_time).
//
// The validateAvailabilitySlot function is defined below and mirrors the
// validation applied in the HTTP handler when a doctor sets availability.
// Keeping the logic here makes it easy to test exhaustively without spinning
// up a full HTTP stack.

import (
	"fmt"
	"testing"
	"time"
)

// validateAvailabilitySlot validates a single DoctorAvailability slot.
// Returns a non-nil error if any field is invalid:
//   - day_of_week must be in [0, 6]
//   - start_time and end_time must be non-empty and in "HH:MM" format
//   - end_time must be strictly after start_time (no midnight crossing)
func validateAvailabilitySlot(dayOfWeek int, startTime, endTime string) error {
	if dayOfWeek < 0 || dayOfWeek > 6 {
		return fmt.Errorf("day_of_week must be 0–6, got %d", dayOfWeek)
	}
	if startTime == "" || endTime == "" {
		return fmt.Errorf("start_time and end_time are required")
	}
	start, err := time.Parse("15:04", startTime)
	if err != nil {
		return fmt.Errorf("invalid start_time format %q: %v", startTime, err)
	}
	end, err := time.Parse("15:04", endTime)
	if err != nil {
		return fmt.Errorf("invalid end_time format %q: %v", endTime, err)
	}
	if !end.After(start) {
		return fmt.Errorf("end_time must be strictly after start_time")
	}
	return nil
}

// TestAvailabilitySlotValidation runs the full validation rule matrix.
func TestAvailabilitySlotValidation(t *testing.T) {
	tests := []struct {
		name      string
		dayOfWeek int
		startTime string
		endTime   string
		wantErr   bool
	}{
		// ── Valid cases ───────────────────────────────────────────────────────
		{"valid Monday 9-17", 1, "09:00", "17:00", false},
		{"valid Sunday (0) 8-12", 0, "08:00", "12:00", false},
		{"valid Saturday (6) 10-14", 6, "10:00", "14:00", false},
		{"valid Wednesday full-day 0-23:59", 3, "00:00", "23:59", false},
		{"valid one-hour slot", 2, "13:00", "14:00", false},
		{"valid minimum gap (1 min)", 4, "09:00", "09:01", false},

		// ── Invalid day_of_week ───────────────────────────────────────────────
		{"invalid day -1", -1, "09:00", "17:00", true},
		{"invalid day 7", 7, "09:00", "17:00", true},
		{"invalid day 100", 100, "09:00", "17:00", true},

		// ── Invalid time format ───────────────────────────────────────────────
		// Note: Go's time.Parse("15:04") accepts single-digit hours (e.g. "9:00").
		// Validation at the handler level uses the same parser, so "9:00" is valid.
		// We test formats that are genuinely unparsable instead.
		{"invalid start format single digit min", 1, "09:0", "17:00", true},
		{"invalid end format single digit min", 1, "09:00", "17:0", true},
		{"invalid start garbage", 1, "abc", "17:00", true},
		{"invalid end garbage", 1, "09:00", "xyz", true},
		{"invalid start non-numeric", 1, "ab:cd", "17:00", true},
		{"invalid end non-numeric", 1, "09:00", "xx:yy", true},

		// ── Empty / missing times ─────────────────────────────────────────────
		{"empty start time", 1, "", "17:00", true},
		{"empty end time", 1, "09:00", "", true},
		{"both times empty", 1, "", "", true},

		// ── Logical ordering errors ───────────────────────────────────────────
		{"start equals end", 1, "09:00", "09:00", true},
		{"start after end", 1, "17:00", "09:00", true},
		{"midnight crossing (23:00 to 01:00)", 1, "23:00", "01:00", true},
		{"start later by one minute", 5, "10:01", "10:00", true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateAvailabilitySlot(tt.dayOfWeek, tt.startTime, tt.endTime)
			if tt.wantErr && err == nil {
				t.Errorf("expected validation error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

// TestDoctorAvailabilityFields verifies that the DoctorAvailability model
// struct has the expected fields used by the validation logic.
func TestDoctorAvailabilityFields(t *testing.T) {
	da := DoctorAvailability{
		DoctorID:    42,
		DayOfWeek:   1,
		StartTime:   "09:00",
		EndTime:     "17:00",
		IsAvailable: true,
		Version:     1,
	}

	if da.DoctorID != 42 {
		t.Errorf("DoctorID: expected 42, got %d", da.DoctorID)
	}
	if da.DayOfWeek != 1 {
		t.Errorf("DayOfWeek: expected 1, got %d", da.DayOfWeek)
	}
	if da.StartTime != "09:00" {
		t.Errorf("StartTime: expected 09:00, got %s", da.StartTime)
	}
	if da.EndTime != "17:00" {
		t.Errorf("EndTime: expected 17:00, got %s", da.EndTime)
	}
	if !da.IsAvailable {
		t.Errorf("IsAvailable: expected true")
	}
	if da.Version != 1 {
		t.Errorf("Version: expected 1, got %d", da.Version)
	}

	// Validate that the struct-populated slot passes the validator.
	if err := validateAvailabilitySlot(da.DayOfWeek, da.StartTime, da.EndTime); err != nil {
		t.Errorf("valid DoctorAvailability failed validation: %v", err)
	}
}

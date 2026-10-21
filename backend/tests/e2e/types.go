package e2e

import (
	"encoding/json"
	"time"
)

// TestResponse is a generic wrapper for standardized API responses in tests
type TestResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Meta    interface{}     `json:"meta"`
}

// LoginResponse represents the structure of the login response data
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         struct {
		ID       uint   `json:"id"`
		Email    string `json:"email"`
		UserRole string `json:"role"`
	} `json:"user"`
}

// Global variable to hold tokens (simulating client state)
var (
	PatientToken string
	DoctorToken  string
	PatientID    uint
	DoctorID     uint
)

// Constants for test data
const (
	TestPatientEmail = "e2e_patient@example.com"
	TestDoctorEmail  = "e2e_doctor@example.com"
	TestPassword     = "password123"
)

// AppointmentResponse
type AppointmentResponse struct {
	ID        uint      `json:"id"`
	Status    string    `json:"status"`
	DoctorID  *uint     `json:"doctor_id"`
	CreatedAt time.Time `json:"created_at"`
}

// PrescriptionRequest
type PrescriptionRequest struct {
	AppointmentID uint             `json:"appointment_id"`
	Medications   []MedicationItem `json:"medications"`
	Instructions  string           `json:"instructions"`
}

type MedicationItem struct {
	Name     string `json:"name"`
	Dosage   string `json:"dosage"`
	Duration string `json:"duration"`
	Quantity int    `json:"quantity"`
}

package v2

import (
	"context"
	"errors"

	"drayya-core/internal/database"
)

// getMyPrescriptionsTool returns the caller's own prescriptions — never
// anyone else's. user_id is taken from the runtime persona; the LLM cannot
// pass an arbitrary user_id.
//
// Patient-only. Guests have no prescriptions by definition.
var getMyPrescriptionsTool = Tool{
	Name: "get_my_prescriptions",
	Description: "Fetch the current user's recent prescriptions issued by " +
		"doctors on the platform. Use when the user asks 'what did doctor X " +
		"prescribe', 'show my last prescription', or 'do I have any active " +
		"prescriptions'. Returns up to 5 most recent.",
	Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"limit": map[string]any{
				"type":        "integer",
				"description": "How many prescriptions to return. Defaults to 5; max 10.",
			},
		},
	},
	Allowed:       []PersonaRole{RolePatient},
	Confirm:       false,
	MaxPerSession: 5,
	MaxPerMinute:  3,
	Execute:       executeGetMyPrescriptions,
}

type prescriptionResult struct {
	ID         uint   `json:"id"`
	IssuedDate string `json:"issued_date"`
	Diagnosis  string `json:"diagnosis,omitempty"`
	DoctorName string `json:"doctor_name,omitempty"`
	Status     string `json:"status"`
	PDFURL     string `json:"pdf_url,omitempty"`
}

func executeGetMyPrescriptions(ctx context.Context, args map[string]any) (any, error) {
	persona := PersonaFromContext(ctx)
	if persona.UserID == nil {
		return map[string]any{"error": "must be signed in to view prescriptions"}, nil
	}
	if database.DB == nil {
		return nil, errors.New("database unavailable")
	}
	limit := 5
	if l, ok := args["limit"].(float64); ok && l > 0 && l <= 10 {
		limit = int(l)
	}

	type row struct {
		ID         uint
		Diagnosis  string
		Status     string
		DoctorName string
		FileURL    string
		CreatedAt  string
	}
	var rows []row
	database.DB.Table("prescriptions p").
		Select(`p.id, p.diagnosis, p.status, u.full_name AS doctor_name,
		         p.file_url, p.created_at`).
		Joins(`LEFT JOIN "user" u ON u.id = p.doctor_id`).
		Where(`p.user_id = ?`, *persona.UserID). // patient column is user_id (see Prescription model)
		Order("p.created_at DESC").Limit(limit).Scan(&rows)

	out := make([]prescriptionResult, 0, len(rows))
	for _, r := range rows {
		out = append(out, prescriptionResult{
			ID:         r.ID,
			IssuedDate: r.CreatedAt,
			Diagnosis:  r.Diagnosis,
			DoctorName: r.DoctorName,
			Status:     r.Status,
			PDFURL:     r.FileURL,
		})
	}
	return map[string]any{
		"prescriptions": out,
		"total_found":   len(out),
	}, nil
}

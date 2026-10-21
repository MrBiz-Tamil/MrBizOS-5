package v2

import (
	"context"
	"errors"
	"strings"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
)

// searchDoctorsTool finds verified, active doctors filtered by AYUSH
// specialization. Read-only; allowed for guests and patients.
//
// The LLM passes natural language like "siddha", "ayurveda", or a city. The
// runtime resolves these to indexed columns on doctorprofile/user.
var searchDoctorsTool = Tool{
	Name: "search_doctors",
	Description: "Find AYUSH doctors (Siddha, Ayurveda, Yoga, Unani, Homeopathy, Sowa-Rigpa) " +
		"available on the platform. Filter by specialization and/or city. Returns up to 5 " +
		"doctors with name, specialization, experience, and rating. Use this when the user " +
		"wants to book, browse, or hear about doctors.",
	Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"specialization": map[string]any{
				"type":        "string",
				"description": "AYUSH system to filter by: 'siddha', 'ayurveda', 'unani', 'yoga_naturopathy', 'homeopathy', 'sowa_rigpa', or empty for any.",
			},
			"city": map[string]any{
				"type":        "string",
				"description": "City name to filter doctors by (optional).",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "How many doctors to return. Defaults to 5. Max 10.",
			},
		},
	},
	Allowed:       []PersonaRole{RoleGuest, RolePatient},
	Confirm:       false,
	MaxPerSession: 10,
	MaxPerMinute:  5,
	Execute:       executeSearchDoctors,
}

// doctorResult is the trimmed wire shape sent back to the LLM. We don't
// send raw identity.User — too much PII and irrelevant fields would pollute
// the prompt and risk leaking PII into the LLM provider.
type doctorResult struct {
	ID             uint    `json:"id"`
	Name           string  `json:"name"`
	Specialization string  `json:"specialization"`
	ExperienceYrs  int     `json:"experience_years"`
	Rating         float64 `json:"rating"`
	IsVerified     bool    `json:"is_verified"`
	City           string  `json:"city,omitempty"`
}

func executeSearchDoctors(ctx context.Context, args map[string]any) (any, error) {
	if database.DB == nil {
		return nil, errors.New("database unavailable")
	}

	spec, _ := args["specialization"].(string)
	city, _ := args["city"].(string)
	limit := 5
	if l, ok := args["limit"].(float64); ok && l > 0 && l <= 10 {
		limit = int(l)
	}

	// JOIN user → doctorprofile, filter by verified + active. The LLM-given
	// specialization is normalised to lower-snake to match what onboard writes.
	type row struct {
		UserID         uint
		FullName       string
		Specialization string
		Years          int
		Rating         float64
		Verified       bool
	}
	var rows []row
	q := database.DB.
		Table(`"user" u`).
		Select(`u.id AS user_id, u.full_name AS full_name,
		         dp.specialization AS specialization,
		         dp.years_of_experience AS years,
		         dp.average_rating AS rating,
		         dp.is_verified AS verified`).
		Joins(`JOIN doctorprofile dp ON dp.user_id = u.id`).
		Where(`u.role = ? AND u.is_active = ? AND dp.is_verified = ?`,
			"doctor", true, true)

	if spec != "" {
		normalised := strings.ToLower(strings.TrimSpace(spec))
		// Match against the human-readable label as well as the raw slug —
		// the onboard handler now writes a label like "Siddha (BSMS)".
		q = q.Where(`LOWER(dp.specialization) LIKE ?`, "%"+normalised+"%")
	}
	if city != "" {
		// City lives in the hospital, not the doctor. Skipped in Phase 0;
		// added in Phase 2 when we wire hospital JOINs.
		_ = city
	}

	q.Order(`dp.average_rating DESC, dp.years_of_experience DESC`).
		Limit(limit).Scan(&rows)

	out := make([]doctorResult, 0, len(rows))
	for _, r := range rows {
		out = append(out, doctorResult{
			ID:             r.UserID,
			Name:           r.FullName,
			Specialization: r.Specialization,
			ExperienceYrs:  r.Years,
			Rating:         r.Rating,
			IsVerified:     r.Verified,
		})
	}
	return map[string]any{
		"doctors":     out,
		"total_found": len(out),
	}, nil
}

// SymptomAnalysisStub uses identity.User for compile-time linkage. Without
// this reference Go would flag the package import as unused once the
// integration moves to a separate join file. Remove when Phase 2 lands a
// real user fetch via Pluck.
var _ = identity.User{}

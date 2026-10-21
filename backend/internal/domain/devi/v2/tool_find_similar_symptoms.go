package v2

import (
	"context"
	"errors"

	"drayya-core/internal/database"
)

// findSimilarSymptomsTool searches the caller's OWN past symptom logs for
// entries similar to the current query. Phase 0 uses keyword fallback;
// Phase 2 will plug in pgvector cosine similarity using the existing
// embedding column.
//
// Patient-only: guests by definition have no history.
var findSimilarSymptomsTool = Tool{
	Name: "find_similar_symptoms",
	Description: "Look up the current user's past symptom-check conversations " +
		"that are similar to the query. Use when the user says 'when did I last " +
		"have this', 'is this the same as before', or to check continuity of " +
		"care. Returns up to 3 recent matching entries.",
	Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Description of current symptoms.",
			},
		},
		"required": []string{"query"},
	},
	Allowed:       []PersonaRole{RolePatient},
	Confirm:       false,
	MaxPerSession: 5,
	MaxPerMinute:  3,
	Execute:       executeFindSimilarSymptoms,
}

type symptomMatchResult struct {
	ID        uint   `json:"id"`
	Symptoms  string `json:"symptoms"`
	Urgency   int    `json:"urgency"`
	CreatedAt string `json:"created_at"`
}

func executeFindSimilarSymptoms(ctx context.Context, args map[string]any) (any, error) {
	persona := PersonaFromContext(ctx)
	if persona.UserID == nil {
		return map[string]any{"error": "must be signed in"}, nil
	}
	if database.DB == nil {
		return nil, errors.New("database unavailable")
	}
	query, _ := args["query"].(string)
	if query == "" {
		return map[string]any{"matches": []any{}, "total_found": 0}, nil
	}

	// Phase 0/1 — keyword search. Phase 2 will switch this to
	// embedding-based cosine similarity using the existing pgvector column.
	type row struct {
		ID        uint
		Symptoms  string
		Urgency   int
		CreatedAt string
	}
	var rows []row
	database.DB.Table("devi_symptom_logs").
		Select("id, symptoms_text AS symptoms, urgency, created_at").
		Where("user_id = ? AND LOWER(symptoms_text) LIKE ?",
			*persona.UserID, "%"+query+"%").
		Order("created_at DESC").
		Limit(3).Scan(&rows)

	out := make([]symptomMatchResult, 0, len(rows))
	for _, r := range rows {
		out = append(out, symptomMatchResult{
			ID:        r.ID,
			Symptoms:  r.Symptoms,
			Urgency:   r.Urgency,
			CreatedAt: r.CreatedAt,
		})
	}
	return map[string]any{
		"matches":     out,
		"total_found": len(out),
	}, nil
}

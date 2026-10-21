package v2

import (
	"context"
	"errors"
	"strings"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/utils/privacy"
)

// updatePatientProfileTool saves allergies / chronic conditions / language
// preference onto the caller's PatientProfile. Reversible (they can update
// again any time), so no confirmation gate.
//
// Patient-only. The LLM can't pass user_id — taken from the runtime persona.
var updatePatientProfileTool = Tool{
	Name: "update_patient_profile",
	Description: "Save the user's allergies, chronic conditions, or preferred " +
		"language to their profile. Use during onboarding ('what should I know " +
		"about your health?') or whenever the user mentions a new condition. " +
		"Pass only the fields the user provided; omit the rest.",
	Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"allergies": map[string]any{
				"type":        "string",
				"description": "Comma-separated list of allergies, or 'none'.",
			},
			"chronic_conditions": map[string]any{
				"type":        "string",
				"description": "Comma-separated list of chronic conditions, or 'none'.",
			},
			"preferred_language": map[string]any{
				"type":        "string",
				"description": "One of 'english', 'tamil', 'malayalam', 'hindi'.",
			},
		},
	},
	Allowed:       []PersonaRole{RolePatient},
	Confirm:       false,
	MaxPerSession: 5,
	MaxPerMinute:  3,
	Execute:       executeUpdatePatientProfile,
}

func executeUpdatePatientProfile(ctx context.Context, args map[string]any) (any, error) {
	persona := PersonaFromContext(ctx)
	if persona.UserID == nil {
		return map[string]any{"error": "must be signed in"}, nil
	}
	if database.DB == nil {
		return nil, errors.New("database unavailable")
	}

	updates := map[string]any{}
	if v, ok := args["allergies"].(string); ok && strings.TrimSpace(v) != "" {
		updates["allergies"] = v
	}
	if v, ok := args["chronic_conditions"].(string); ok && strings.TrimSpace(v) != "" {
		updates["chronic_conditions"] = v
	}
	if v, ok := args["preferred_language"].(string); ok && strings.TrimSpace(v) != "" {
		// preferred_language lives on the User table, not PatientProfile.
		// We handle it separately below.
	}

	// Ensure a PatientProfile row exists; AutoMigrate set it up but it's
	// possible there's no row yet for users created before the flow ran.
	var existing identity.PatientProfile
	err := database.DB.Where("user_id = ?", *persona.UserID).First(&existing).Error
	if err != nil {
		// Create.
		profile := identity.PatientProfile{UserID: *persona.UserID}
		for k, v := range updates {
			switch k {
			case "allergies":
				if s, ok := v.(string); ok {
					profile.Allergies = privacy.EncryptedString(s)
				}
			case "chronic_conditions":
				if s, ok := v.(string); ok {
					profile.ChronicConditions = privacy.EncryptedString(s)
				}
			}
		}
		if err := database.DB.WithContext(ctx).Create(&profile).Error; err != nil {
			return nil, err
		}
	} else if len(updates) > 0 {
		if err := database.DB.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	// Language goes on User.
	if v, ok := args["preferred_language"].(string); ok && strings.TrimSpace(v) != "" {
		_ = database.DB.WithContext(ctx).Model(&identity.User{}).
			Where("id = ?", *persona.UserID).
			Update("preferred_language", strings.ToLower(strings.TrimSpace(v))).Error
	}

	return map[string]any{
		"updated_fields": fieldsTouched(args),
		"message":        "Profile updated. Doctors will see this on your next consultation.",
	}, nil
}

func fieldsTouched(args map[string]any) []string {
	out := []string{}
	for _, k := range []string{"allergies", "chronic_conditions", "preferred_language"} {
		if v, ok := args[k].(string); ok && strings.TrimSpace(v) != "" {
			out = append(out, k)
		}
	}
	return out
}

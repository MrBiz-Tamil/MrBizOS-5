package v2

import (
	"context"
	"errors"

	"drayya-core/internal/database"
)

// switchFamilyContextTool changes the active patient context for the
// session. After this call, subsequent tools (book, prescription lookup,
// etc.) operate on the family member instead of the primary user.
//
// The actual storage is Redis-backed session state — we'll persist the
// chosen family_member_id under devi:v2:family_ctx:{session_id} with a
// matching TTL.
//
// Patient-only. The tool VERIFIES the family member belongs to the caller
// before activating — this is the critical guard, since the LLM gives us
// the family_member_id (admin operations can't be trusted to it).
var switchFamilyContextTool = Tool{
	Name: "switch_family_context",
	Description: "Switch the active profile for this conversation to a " +
		"family member you've added (mother, child, etc.) so subsequent " +
		"questions are scoped to them. Pass family_member_id from the user's " +
		"family list. Pass 0 to switch back to the primary user.",
	Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"family_member_id": map[string]any{
				"type":        "integer",
				"description": "The family member ID. Pass 0 to switch back to the primary user.",
			},
		},
		"required": []string{"family_member_id"},
	},
	Allowed:       []PersonaRole{RolePatient},
	Confirm:       false,
	MaxPerSession: 10,
	MaxPerMinute:  5,
	Execute:       executeSwitchFamilyContext,
}

func executeSwitchFamilyContext(ctx context.Context, args map[string]any) (any, error) {
	persona := PersonaFromContext(ctx)
	if persona.UserID == nil {
		return map[string]any{"error": "must be signed in"}, nil
	}
	sessionID := SessionIDFromContext(ctx)
	if sessionID == "" {
		return map[string]any{"error": "no session"}, nil
	}

	var fmID uint
	if v, ok := args["family_member_id"].(float64); ok && v >= 0 {
		fmID = uint(v)
	}

	// fmID == 0 → clear the family context (back to primary).
	if fmID == 0 {
		if database.Rdb != nil {
			_ = database.Rdb.Del(ctx, familyCtxKey(*persona.UserID, sessionID)).Err()
		}
		return map[string]any{
			"switched_to": "primary_user",
			"message":     "Switched back to your profile.",
		}, nil
	}

	// Critical guard — the family member MUST belong to this user. The LLM
	// can't be trusted to enforce this; we check the DB ourselves.
	if database.DB == nil {
		return nil, errors.New("database unavailable")
	}
	type row struct {
		FullName     string
		Relationship string
	}
	var r row
	tx := database.DB.Table("family_members").
		Select("full_name, relationship").
		Where("id = ? AND user_id = ?", fmID, *persona.UserID).
		Scan(&r)
	if tx.Error != nil || r.FullName == "" {
		return map[string]any{"error": "no such family member, or it doesn't belong to you"}, nil
	}

	if database.Rdb != nil {
		_ = database.Rdb.Set(ctx, familyCtxKey(*persona.UserID, sessionID), fmID, ConversationTTL).Err()
	}
	return map[string]any{
		"switched_to":       r.FullName,
		"relationship":      r.Relationship,
		"family_member_id":  fmID,
		"message":           "Switched to " + r.FullName + "'s profile for this conversation.",
	}, nil
}

func familyCtxKey(userID uint, sessionID string) string {
	return "devi:v2:family_ctx:" + uintStr(userID) + ":" + sessionID
}

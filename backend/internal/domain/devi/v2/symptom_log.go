package v2

import (
	"context"
	"strings"
	"time"

	"drayya-core/internal/database"
)

// Participant types — mirrored from internal/domain/devi to avoid a cyclic
// import (V1 doesn't import V2; if V2 imports V1, then the future shim that
// has V1 route to V2 would create the cycle).
const (
	participantPrimary      = "primary_user"
	participantFamilyMember = "family_member"
	participantGuest        = "guest"
)

// writeSymptomLog persists a row to devi_symptom_logs so the admin Devi
// dashboard sees V2 traffic alongside V1. Response timing is stamped onto
// the row so the "Avg Response" KPI (just rebuilt to be real) reflects V2
// latency.
//
// We DON'T log every turn — only ones that look like substantive symptom
// or platform questions. Greetings and single-word replies are skipped to
// avoid bloating the table and skewing intent metrics.
//
// Uses raw INSERT (not GORM struct) to avoid importing the V1 devi package.
// Best-effort: failures are swallowed so a Redis or DB blip never breaks
// the chat reply.
func writeSymptomLog(ctx context.Context, persona Persona, sessionID, userMessage, intent string, urgency, responseMS int) {
	if database.DB == nil {
		return
	}
	if !shouldLogTurn(userMessage) {
		return
	}

	var (
		userID         uint
		familyMemberID *uint
		guestBrowserID *string
		ptype          string
	)
	if persona.UserID != nil {
		userID = *persona.UserID
		ptype = participantPrimary
		if persona.FamilyMemberID != nil {
			familyMemberID = persona.FamilyMemberID
			ptype = participantFamilyMember
		}
	} else if persona.GuestBrowserID != "" {
		bid := persona.GuestBrowserID
		guestBrowserID = &bid
		ptype = participantGuest
	}

	go func() {
		_ = database.DB.WithContext(context.Background()).Exec(`
			INSERT INTO devi_symptom_logs
			    (user_id, family_member_id, guest_browser_id, participant_type,
			     session_id, intent, symptoms_text, urgency, booked_appt,
			     response_ms, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			userID, familyMemberID, guestBrowserID, ptype,
			sessionID, intent, userMessage, urgency, false,
			responseMS, time.Now(),
		).Error
	}()
}

// shouldLogTurn filters chatter that adds no signal to the symptom logs.
// Single-word affirmations, hellos, and clear confirmation replies are
// skipped. Everything else gets logged.
func shouldLogTurn(msg string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(msg))
	if trimmed == "" || len(trimmed) < 4 {
		return false
	}
	switch trimmed {
	case "hi", "hello", "hey", "thanks", "ok", "okay", "yes", "no", "sure":
		return false
	}
	return true
}

package devi

import (
	"fmt"
)

// UserContext is passed into every Devi engine call.
// It replaces the flat "userID string" that was used before, enabling proper
// session isolation across three participant types:
//
//   - Guest            – no account; sessions keyed by browser UUID
//   - Primary user     – authenticated; sessions keyed by user_id
//   - Family member    – authenticated parent chatting on behalf of a dependent
type UserContext struct {
	ParticipantType ParticipantType

	// Set for authenticated users (primary + family-member sessions)
	UserID *uint

	// Set only for family-member sessions
	FamilyMemberID   *uint
	FamilyMemberName string // display name of the dependent, filled by the caller

	// Set only for guest sessions
	GuestBrowserID string // UUID from cookie / localStorage

	// Computed Redis key for session state (30-min TTL for auth, 7-day for guests)
	SessionKey string
	// Computed Redis key for LLM conversation history
	HistoryKey string

	// Human-readable name used in Devi greeting responses
	DisplayName string
}

// BuildUserContext constructs a fully-scoped UserContext from request inputs.
// Call this in every HTTP handler and WS message handler before calling engine.Process.
//
//	userID         – nil for unauthenticated callers
//	familyMemberID – non-nil when the authenticated user is chatting for a dependent
//	guestBrowserID – UUID for unauthenticated callers (stored in browser localStorage)
func BuildUserContext(userID *uint, familyMemberID *uint, guestBrowserID string) *UserContext {
	uc := &UserContext{}

	switch {
	case userID == nil:
		// ── Guest ───────────────────────────────────────────────────────────────
		id := guestBrowserID
		if id == "" {
			id = "anon"
		}
		uc.ParticipantType = ParticipantGuest
		uc.GuestBrowserID  = id
		uc.SessionKey      = "devi:state:guest:" + id
		uc.HistoryKey      = "devi:history:guest:" + id
		uc.DisplayName     = "Guest"

	case familyMemberID != nil:
		// ── Family member (on behalf of) ────────────────────────────────────────
		uc.ParticipantType   = ParticipantFamilyMember
		uc.UserID            = userID
		uc.FamilyMemberID    = familyMemberID
		uc.SessionKey        = fmt.Sprintf("devi:state:user:%d:fm:%d", *userID, *familyMemberID)
		uc.HistoryKey        = fmt.Sprintf("devi:history:user:%d:fm:%d", *userID, *familyMemberID)
		uc.DisplayName       = "Family Member" // overwritten by caller after DB lookup

	default:
		// ── Authenticated primary user ───────────────────────────────────────────
		uc.ParticipantType = ParticipantPrimary
		uc.UserID          = userID
		uc.SessionKey      = fmt.Sprintf("devi:state:user:%d:primary", *userID)
		uc.HistoryKey      = fmt.Sprintf("devi:history:user:%d:primary", *userID)
		uc.DisplayName     = "there" // overwritten by caller if user name is known
	}

	return uc
}

// IsGuest returns true for unauthenticated sessions.
func (uc *UserContext) IsGuest() bool {
	return uc.ParticipantType == ParticipantGuest
}

// IsFamilyMember returns true when the caller is chatting on behalf of a dependent.
func (uc *UserContext) IsFamilyMember() bool {
	return uc.ParticipantType == ParticipantFamilyMember
}

// UserIDStr returns the primary user's ID as a string (empty for guests).
func (uc *UserContext) UserIDStr() string {
	if uc.UserID == nil {
		return ""
	}
	return fmt.Sprintf("%d", *uc.UserID)
}

// CanBook returns true when the participant has enough permissions to create
// appointments (guests cannot book — they must register first).
func (uc *UserContext) CanBook() bool {
	return !uc.IsGuest()
}

// CanAccessPharmacy returns true for authenticated participants.
func (uc *UserContext) CanAccessPharmacy() bool {
	return !uc.IsGuest()
}

// CanAccessLab returns true for authenticated participants.
func (uc *UserContext) CanAccessLab() bool {
	return !uc.IsGuest()
}

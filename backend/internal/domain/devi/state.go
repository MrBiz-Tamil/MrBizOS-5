package devi

// state.go — Redis session state for Devi conversations.
//
// Top-level envelope: sessionState (Intent + JSON payload).
// Each flow has its own typed payload struct that is marshalled into
// sessionState.Payload.  This replaces the old map[string]string approach
// and lets each flow evolve its own fields independently.

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// sessionState is the Redis envelope stored at uc.SessionKey.
type sessionState struct {
	Intent  string          `json:"intent"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// ── Per-flow typed payload structs ────────────────────────────────────────────

// BookingState tracks the multi-step appointment booking flow.
type BookingState struct {
	Step            int          `json:"step"`
	ConsultType     string       `json:"consult_type"`    // general|specialist|mental|emergency
	Specialization  string       `json:"specialization"`
	Symptoms        string       `json:"symptoms"`
	Doctors         []DoctorCard `json:"doctors"`         // options shown to user (max 3)
	SelectedDoctor  *DoctorCard  `json:"selected_doctor"`
	Slots           []SlotCard   `json:"slots"`           // time slots shown to user
	SelectedSlot    *SlotCard    `json:"selected_slot"`
	AppointmentID   uint         `json:"appointment_id"`  // set after DB create (step 4)
	FeePaise        int64        `json:"fee_paise"`
	ForFamilyMember bool         `json:"for_family_member"`
}

// PharmacyState tracks the medicine search → cart → checkout flow.
type PharmacyState struct {
	Step        int        `json:"step"`
	SearchQuery string     `json:"search_query"`
	Results     []MedCard  `json:"results"` // search results shown
	Cart        []CartItem `json:"cart"`
	CartTotal   int64      `json:"cart_total"` // paise
}

// PaymentState tracks the wallet top-up and in-chat payment flows.
type PaymentState struct {
	Step          int    `json:"step"`
	Action        string `json:"action"`         // topup | pay_consultation | balance
	AmountPaise   int64  `json:"amount_paise"`
	AppointmentID uint   `json:"appointment_id"`
}

// LabState tracks the lab-test booking flow.
type LabState struct {
	Step     int      `json:"step"`
	Query    string   `json:"query"`
	Results  []LabCard `json:"results"`
	Selected *LabCard  `json:"selected"`
}

// SupportState tracks the support-ticket creation flow.
type SupportState struct {
	Step        int    `json:"step"`
	IssueType   string `json:"issue_type"`
	Description string `json:"description"`
}

// SymptomState tracks the symptom-check flow.
type SymptomState struct {
	Step       int    `json:"step"`
	Symptoms   string `json:"symptoms"`
	Department string `json:"department"`
	DoctorID   string `json:"doctor_id"`
}

// ── Lightweight card types ────────────────────────────────────────────────────

// DoctorCard is a minimal doctor summary for display in chat.
type DoctorCard struct {
	UserID         uint    `json:"user_id"`
	ProfileID      uint    `json:"profile_id"`
	Name           string  `json:"name"`
	Specialization string  `json:"specialization"`
	FeePaise       int64   `json:"fee_paise"`
	Rating         float64 `json:"rating"`
	IsOnline       bool    `json:"is_online"`
}

// SlotCard is a bookable time slot.
type SlotCard struct {
	Label     string    `json:"label"`      // e.g. "Today 3:00 PM"
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// MedCard is a pharmacy product summary.
type MedCard struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Manufacturer string `json:"manufacturer"`
	PricePaise   int64  `json:"price_paise"`
	InStock      bool   `json:"in_stock"`
}

// CartItem is one line in the pharmacy cart.
type CartItem struct {
	MedID      uint   `json:"med_id"`
	Name       string `json:"name"`
	Qty        int    `json:"qty"`
	PricePaise int64  `json:"price_paise"` // per unit
}

// LabCard is a lab-test summary.
type LabCard struct {
	ID             uint    `json:"id"`
	Name           string  `json:"name"`
	PricePaise     int64   `json:"price_paise"`
	HomeCollection bool    `json:"home_collection"`
}

// ── Generic payload helpers ───────────────────────────────────────────────────

func marshalPayload(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func unmarshalBooking(s *sessionState) BookingState {
	var p BookingState
	_ = json.Unmarshal(s.Payload, &p)
	return p
}

func unmarshalPharmacy(s *sessionState) PharmacyState {
	var p PharmacyState
	_ = json.Unmarshal(s.Payload, &p)
	return p
}

func unmarshalPayment(s *sessionState) PaymentState {
	var p PaymentState
	_ = json.Unmarshal(s.Payload, &p)
	return p
}

func unmarshalLab(s *sessionState) LabState {
	var p LabState
	_ = json.Unmarshal(s.Payload, &p)
	return p
}

func unmarshalSupport(s *sessionState) SupportState {
	var p SupportState
	_ = json.Unmarshal(s.Payload, &p)
	return p
}

func unmarshalSymptom(s *sessionState) SymptomState {
	var p SymptomState
	_ = json.Unmarshal(s.Payload, &p)
	return p
}

// ── Redis state helpers ───────────────────────────────────────────────────────

func redisLoadState(ctx context.Context, rdb *redis.Client, key string) *sessionState {
	s := &sessionState{}
	if rdb == nil {
		return s
	}
	raw, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return s
	}
	_ = json.Unmarshal([]byte(raw), s)
	return s
}

func redisSaveState(ctx context.Context, rdb *redis.Client, key string, s *sessionState, ttl time.Duration) {
	if rdb == nil {
		return
	}
	b, _ := json.Marshal(s)
	rdb.Set(ctx, key, string(b), ttl)
}

func redisClearState(ctx context.Context, rdb *redis.Client, key string) {
	if rdb != nil {
		rdb.Del(ctx, key)
	}
}

// sessionTTL returns the appropriate Redis TTL for a user context.
func sessionTTL(uc *UserContext) time.Duration {
	if uc.IsGuest() {
		return 7 * 24 * time.Hour
	}
	return 30 * time.Minute
}

// paise converts rupees (as float64) to paise (int64).
func paise(rupees float64) int64 { return int64(rupees * 100) }

// rupees converts paise to rupees string for display.
func rupeesStr(p int64) string { return fmt.Sprintf("%.0f", float64(p)/100.0) }

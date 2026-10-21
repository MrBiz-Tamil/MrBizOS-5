package devi

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
)

// InsuranceState holds the state for the multi-step insurance flow.
type InsuranceState struct {
	Step       int    `json:"step"`
	PolicyType string `json:"policy_type"`
	AgeGroup   string `json:"age_group"`
	Phone      string `json:"phone"`
}

func (e *Engine) insuranceStart(ctx context.Context, uc *UserContext, msg string, state *sessionState) FlowResult {
	st := InsuranceState{Step: 0}
	state.Intent = string(intentInsurance)
	state.Payload = marshalPayload(st)

	return reply(
		"🛡️ *Dr.Ayya Health Insurance*\n\nGreat choice — let's find you the right plan! 💙\n\n"+
			"Which type of coverage are you looking for?\n\n"+
			"1️⃣  *Individual* — just for me\n"+
			"2️⃣  *Family Floater* — me + spouse + kids\n"+
			"3️⃣  *Senior Citizen* — for parents (60+)\n"+
			"4️⃣  *Critical Illness* — coverage for 30+ major diseases\n\n"+
			"Tap a button or reply with a number (1-4):",
		btn("1 Individual", "1"), btn("2 Family", "2"),
		btn("3 Senior", "3"), btn("4 Critical", "4"),
	)
}

func (e *Engine) insuranceContinue(ctx context.Context, uc *UserContext, state *sessionState, msg string) FlowResult {
	var st InsuranceState
	if err := json.Unmarshal([]byte(state.Payload), &st); err != nil {
		return e.greetingStart(uc)
	}

	switch st.Step {
	case 0:
		return e.insuranceStep0(ctx, uc, state, &st, msg)
	case 1:
		return e.insuranceStep1(ctx, uc, state, &st, msg)
	case 2:
		return e.insuranceStep2(ctx, uc, state, &st, msg)
	}
	return done(e.buildMainMenu(uc))
}

func (e *Engine) insuranceStep0(ctx context.Context, uc *UserContext, state *sessionState, st *InsuranceState, msg string) FlowResult {
	lower := strings.TrimSpace(msg)
	switch lower {
	case "1":
		st.PolicyType = "Individual"
	case "2":
		st.PolicyType = "Family Floater"
	case "3":
		st.PolicyType = "Senior Citizen"
	case "4":
		st.PolicyType = "Critical Illness"
	default:
		return reply("Please choose 1, 2, 3, or 4.",
			btn("1", "1"), btn("2", "2"), btn("3", "3"), btn("4", "4"))
	}

	st.Step = 1
	state.Payload = marshalPayload(st)

	return reply(
		fmt.Sprintf("Got it: *%s* plan.\n\nWhat is the age of the oldest member to be covered?\n\n"+
			"1️⃣  18–35 years\n"+
			"2️⃣  36–45 years\n"+
			"3️⃣  46–60 years\n"+
			"4️⃣  60+ years", st.PolicyType),
		btn("18-35", "1"), btn("36-45", "2"), btn("46-60", "3"), btn("60+", "4"),
	)
}

func (e *Engine) insuranceStep1(ctx context.Context, uc *UserContext, state *sessionState, st *InsuranceState, msg string) FlowResult {
	lower := strings.TrimSpace(msg)
	switch lower {
	case "1":
		st.AgeGroup = "18-35"
	case "2":
		st.AgeGroup = "36-45"
	case "3":
		st.AgeGroup = "46-60"
	case "4":
		st.AgeGroup = "60+"
	default:
		return reply("Please choose 1, 2, 3, or 4.")
	}

	st.Step = 2
	state.Payload = marshalPayload(st)

	return reply(
		"Perfect! 👍 Almost there.\n\n"+
			"Our expert advisor will call you with personalised quotes and tax-saving tips.\n\n"+
			"Is your registered number the best number to reach you on?\n"+
			"(Reply *YES* or type a different 10-digit mobile number)",
		btn("Yes, call me on this number", "yes"),
	)
}

func (e *Engine) insuranceStep2(ctx context.Context, uc *UserContext, state *sessionState, st *InsuranceState, msg string) FlowResult {
	lower := strings.ToLower(strings.TrimSpace(msg))
	phone := lower
	if isYes(lower) {
		// Mock: in real scenario, get from uc or identity
		phone = "User Phone"
	}

	// Capture Lead in DB (Mocked for now, but actionable)
	logInsuranceLead(st.PolicyType, st.AgeGroup, phone)

	refID := fmt.Sprintf("INS-%04d", rand.Intn(9000)+1000)
	return done(fmt.Sprintf(
		"✅ *Insurance enquiry received!* 🎉\n\n"+
			"🆔 Reference #%s\n"+
			"📋 Plan: *%s* | Age group: *%s*\n\n"+
			"📞 Our expert advisor will call *%s* within *30 minutes* with:\n"+
			"• Best quotes from HDFC ERGO, Star Health, Niva Bupa\n"+
			"• Tax-saving tips under Section 80D\n"+
			"• Help choosing the right coverage\n\n"+
			"💡 Have your age and any pre-existing conditions ready for the call."+
			proactiveFollowUp(),
		refID, st.PolicyType, st.AgeGroup, phone))
}

func logInsuranceLead(pType, age, phone string) {
	// Logic to save to 'insurance_leads' table would go here
	fmt.Printf("Insurance Lead: %s, %s, %s\n", pType, age, phone)
}

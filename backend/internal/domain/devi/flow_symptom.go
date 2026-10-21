package devi

// flow_symptom.go — Symptom check → department detection → doctor recommendation.
// State: SymptomState

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// symptomStart is the entry point when the user describes symptoms without
// an explicit booking intent.
func (e *Engine) symptomStart(ctx context.Context, uc *UserContext, msg string, state *sessionState) FlowResult {
	state.Intent = string(intentSymptom)

	// If the message is very short (just "fever", "pain" etc.), ask for more detail
	// so the analysis is more accurate.  Save initial text and wait for clarification.
	if len(strings.TrimSpace(msg)) < 10 {
		st := SymptomState{Step: 0, Symptoms: msg}
		state.Payload = marshalPayload(st)
		return reply(
			"I'm sorry you're not feeling well. 💙\n\n"+
				"Could you tell me a bit more so I can help you better?\n\n"+
				"• How long have you had this?\n"+
				"• Is it mild, moderate, or severe?\n"+
				"• Any other symptoms you've noticed?\n\n"+
				"Take your time — the more detail you share, the better I can guide you!",
		)
	}

	st := SymptomState{Step: 1, Symptoms: msg}
	state.Payload = marshalPayload(st)
	return e.processSymptomText(ctx, uc, state, st)
}

// symptomContinue handles both the clarification step (Step 0) and
// the "Would you like to book?" confirmation step (Step 2).
func (e *Engine) symptomContinue(ctx context.Context, uc *UserContext, state *sessionState, msg string) FlowResult {
	st := unmarshalSymptom(state)
	switch st.Step {
	case 0:
		// User provided clarification — combine with initial symptom and analyse
		combined := st.Symptoms
		if strings.TrimSpace(msg) != "" {
			combined = st.Symptoms + " " + msg
		}
		st.Symptoms = combined
		st.Step = 1
		state.Payload = marshalPayload(st)
		return e.processSymptomText(ctx, uc, state, st)
	case 2:
		lower := strings.ToLower(strings.TrimSpace(msg))
		if isYes(lower) || lower == "book" {
			if !uc.CanBook() {
				return done(e.guestRegisterPrompt("book this consultation"))
			}
			// Transition into booking flow with pre-filled specialization
			bookState := BookingState{
				Step:           1, // skip type selection; jump to show doctors
				ConsultType:    st.Department,
				Specialization: st.Department,
				Symptoms:       st.Symptoms,
			}
			newState := &sessionState{
				Intent:  string(intentBook),
				Payload: marshalPayload(bookState),
			}
			return e.showDoctors(ctx, uc, newState, &bookState)
		}
		return done("No problem! You can always book later. 😊\n\nType *menu* to explore other options." + proactiveFollowUp())
	}
	return done(e.buildMainMenu(uc))
}

// processSymptomText runs the analysis and builds the recommendation message.
func (e *Engine) processSymptomText(ctx context.Context, uc *UserContext, state *sessionState, st SymptomState) FlowResult {
	symptoms := st.Symptoms
	var result matchResult

	if os.Getenv("GEMINI_API_KEY") != "" {
		analysis, err := e.AnalyzeSymptomsLLM(ctx, symptoms, "unknown", "unknown")
		if err == nil && analysis != nil {
			urgency := 0
			switch analysis.UrgencyLevel {
			case "critical":
				urgency = 10
			case "high":
				urgency = 8
			case "moderate":
				urgency = 5
			default:
				urgency = 2
			}
			dept := analysis.RecommendedSpecialization
			if dept == "" && len(analysis.RecommendedSpecializations) > 0 {
				dept = analysis.RecommendedSpecializations[0]
			}
			result = matchResult{department: dept, urgency: urgency}
		} else {
			result = e.matchSymptoms(symptoms)
		}
	} else {
		result = e.matchSymptoms(symptoms)
	}

	// Async: persist symptom log + embedding
	go e.saveSymptomLog(ctx, uc, state.Intent, symptoms, result)

	if result.urgency >= 9 {
		return e.emergencyStart(ctx, uc)
	}

	dept := result.department
	urgencyMsg := ""
	switch {
	case result.urgency >= 7:
		urgencyMsg = "⚠️ *Please see a doctor today* — these symptoms need attention soon.\n\n"
	case result.urgency >= 4:
		urgencyMsg = "🟡 *Worth checking out* — I'd recommend seeing a doctor within 24–48 hours.\n\n"
	default:
		urgencyMsg = "🟢 *This looks routine* — no need to panic! A consultation this week should be fine.\n\n"
	}

	doctor := e.findAvailableDoctor(dept, uc.UserIDStr())
	doctorLine := ""
	doctorID := ""
	if doctor != nil {
		doctorID = fmt.Sprintf("%d", doctor.UserID)
		doctorLine = fmt.Sprintf("\n👨‍⚕️ *Best match:* Dr. profile #%s (%s)\n💰 ₹%s per session\n⭐ %.1f rating\n",
			doctorID, doctor.Specialization, rupeesStr(int64(doctor.ConsultationFees["INR"])*100), doctor.AverageRating)
	} else {
		doctorLine = fmt.Sprintf("\n👨‍⚕️ A verified *%s* specialist will be assigned.\n", dept)
	}

	// Advance state to step 2 (awaiting booking confirmation)
	st.Step = 2
	st.Department = dept
	st.DoctorID = doctorID
	state.Payload = marshalPayload(st)

	bookLine := "\nWould you like to *book a consultation* now?\nReply *YES* or tap the button below."
	if !uc.CanBook() {
		bookLine = "\n\n💙 *Register free* to book this consultation → type *register*."
	}

	msg := fmt.Sprintf("Based on what you've described, here's what I found:\n\n%s📋 *Recommended department:* %s\n%s%s",
		urgencyMsg, dept, doctorLine, bookLine)

	actions := []ChatAction{}
	if uc.CanBook() {
		actions = append(actions, btn("Book Consultation", "yes"), btn("Not Now", "menu"))
	} else {
		actions = append(actions, btn("Register Free", "register"))
	}
	return FlowResult{Message: msg, Actions: actions}
}

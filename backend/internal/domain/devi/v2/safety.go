package v2

import (
	"regexp"
	"strings"
)

// SafetyAction is the outcome of a rule-based safety check that runs BEFORE
// the LLM. When non-nil, the agent loop short-circuits — Reply is returned
// to the user and no LLM call is made.
//
// This is the fast path for cases where:
//   - We must respond instantly (emergency keywords)
//   - We refuse to engage at all (abuse, prompt injection)
//   - The persona itself disqualifies LLM use (doctors, admins)
//
// Sub-millisecond. Pure regex. No external calls.
type SafetyAction struct {
	Kind  string // "emergency" | "crisis" | "abuse" | "defer" | "injection" | "oversize"
	Reply string
	Done  bool // close the conversation after this reply
}

// Emergency keywords — chest pain, breathing difficulty, unconsciousness,
// stroke symptoms, severe bleeding, suicide ideation routed separately as
// "crisis". Keep this list tight; false positives interrupt legitimate
// conversations.
var emergencyPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(chest pain|crushing pain|heart attack|cardiac arrest)\b`),
	regexp.MustCompile(`(?i)\b(can.?t breathe|cannot breathe|gasping|suffocating)\b`),
	regexp.MustCompile(`(?i)\b(unconscious|passed out|fainted|collapsed)\b`),
	regexp.MustCompile(`(?i)\b(stroke|paralysis|face drooping|slurred speech)\b`),
	regexp.MustCompile(`(?i)\b(severe bleeding|won.?t stop bleeding|haemorrhage|hemorrhage)\b`),
	regexp.MustCompile(`(?i)\b(seizure|convulsion|fits)\b`),
	regexp.MustCompile(`(?i)\b(overdose|poisoning|swallowed.+pills)\b`),
}

// Crisis = self-harm or suicide ideation. Routed to a mental-health helpline,
// not the SOS doctor flow. Same fast path as emergency; different message.
var crisisPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(kill myself|end my life|suicide|don.?t want to live|want to die)\b`),
	regexp.MustCompile(`(?i)\b(self.?harm|cut myself|hurt myself)\b`),
}

// Prompt-injection markers — partial list. We do not try to be exhaustive;
// the goal is to drop the most obvious attacks cheaply. The LLM's system
// prompt has belt-and-suspenders instructions too.
var injectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)ignore (the |all )?(previous|prior|above) (instructions|rules|prompts)`),
	regexp.MustCompile(`(?i)you are (now |actually )?(not )?an? (different|new)? ?(ai|assistant|model)`),
	regexp.MustCompile(`(?i)disregard your (system|safety|prior) (prompt|instructions)`),
	regexp.MustCompile(`(?i)\bact as if you (have|are)\b`),
	regexp.MustCompile(`(?i)<\|im_start\|>|<\|im_end\|>|\[INST\]|\[/INST\]`),
}

// SafetyCheck runs ALL rule layers in order. Returns the first hit; nil if
// the message passes all checks and should proceed to the LLM.
func SafetyCheck(msg string, persona Persona) *SafetyAction {
	// Persona deferral runs before content checks — a doctor saying "chest
	// pain" probably means a patient symptom they want to discuss, not their
	// own emergency. Route them to the doctor dashboard instead.
	if persona.IsDeferred() {
		return deferReply(persona)
	}

	// Drop empties and pure whitespace.
	trimmed := strings.TrimSpace(msg)
	if trimmed == "" {
		return &SafetyAction{Kind: "abuse", Reply: "I didn't catch that — could you type your question?"}
	}

	// Oversize messages — reject without scanning further. 2KB is generous
	// for a chat message.
	if len(trimmed) > MaxInputChars {
		return &SafetyAction{
			Kind:  "oversize",
			Reply: "That's a lot of text! Could you summarise your question in a few lines? It helps me give you a better answer.",
		}
	}

	// Emergency — highest priority.
	for _, p := range emergencyPatterns {
		if p.MatchString(trimmed) {
			return &SafetyAction{
				Kind: "emergency",
				Done: true,
				Reply: "🚨 This sounds urgent. Please call emergency services NOW:\n\n" +
					"• Ambulance: 108\n" +
					"• Or use the SOS button in the app\n\n" +
					"If you're with the person: stay calm, keep them comfortable, " +
					"and don't give food or water until help arrives.",
			}
		}
	}

	// Crisis — self-harm / suicide. Direct to professional help.
	for _, p := range crisisPatterns {
		if p.MatchString(trimmed) {
			return &SafetyAction{
				Kind: "crisis",
				Done: true,
				Reply: "I'm really glad you reached out. What you're feeling sounds painful, " +
					"and you deserve to talk to someone who can help right now:\n\n" +
					"• iCall (free, multilingual): +91 9152987821\n" +
					"• Vandrevala Foundation: 1860 2662 345\n" +
					"• AASRA: +91 9820466726\n\n" +
					"You matter. Please call one of these now.",
			}
		}
	}

	// Prompt injection — refuse politely without engaging the substance.
	for _, p := range injectionPatterns {
		if p.MatchString(trimmed) {
			return &SafetyAction{
				Kind:  "injection",
				Reply: "I can't help with that. Want to talk about your health, book a doctor, or find a Siddha remedy instead?",
			}
		}
	}

	return nil
}

// deferReply produces the polite-step-aside response for non-patient roles.
// We don't engage with their query at all — Devi exists for patients.
func deferReply(persona Persona) *SafetyAction {
	switch persona.Role {
	case RoleDoctor:
		return &SafetyAction{
			Kind: "defer",
			Done: true,
			Reply: "Hi doctor 👋 Devi is built for patients. Your tools:\n\n" +
				"📋 Today's queue: /doctor/queue\n" +
				"📅 Schedule: /doctor/schedule\n" +
				"💊 Recent prescriptions: /doctor/prescriptions\n\n" +
				"For platform issues: support@drayya.in",
		}
	case RoleAdmin, RoleStaff:
		return &SafetyAction{
			Kind: "defer",
			Done: true,
			Reply: "Hi! For admin operations, please use Acharya (admin assistant). Devi serves patients only.",
		}
	}
	return nil
}

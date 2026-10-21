package devi

// intent.go — Intent type definitions, keyword detection, and the guest
// capability gate.  Extracted from engine.go so the engine file stays slim.

import "strings"

// intent represents a high-level user goal detected from the message text.
type intent string

const (
	intentGreeting     intent = "greeting"
	intentSymptom      intent = "symptom_check"
	intentBook         intent = "book_appointment"
	intentPharmacy     intent = "pharmacy"
	intentLab          intent = "lab_test"
	intentSupport      intent = "support"
	intentInsurance    intent = "insurance"
	intentPremium      intent = "premium"
	intentEmergency    intent = "emergency"
	intentRegister     intent = "register"
	intentMedicineInfo intent = "medicine_info" // drug info, dosage, interactions
	intentPayment      intent = "payment"       // wallet balance, top-up, pay
	intentMyAccount    intent = "my_account"    // appointments, orders, profile
	intentUnknown      intent = "unknown"
)

var intentKeywords = map[intent][]string{
	intentGreeting: {"hello", "hi", "hey", "start", "namaste", "help",
		"what can you do", "hii", "hey devi", "how are you",
		"good morning", "good evening", "good afternoon", "hai", "helo"},
	intentEmergency: {"emergency", "sos", "ambulance", "critical", "dying",
		"unconscious", "not breathing", "heart attack", "stroke",
		"can't breathe", "bleeding heavily", "chest pain severe",
		"seizure", "fainted", "collapsed", "overdose"},
	intentSymptom: {"symptom", "i have", "i am having", "suffering from", "pain",
		"fever", "headache", "cough", "vomit", "diarrhea", "dizzy", "rash",
		"bleed", "hurt", "ache", "sick", "ill", "chest", "breathe", "swollen",
		"nausea", "stomach ache", "body pain", "feeling",
		// Natural Indian-English additions
		"not feeling well", "feeling unwell", "i am sick", "i am not well",
		"not well", "having trouble", "since yesterday", "since morning",
		"unwell", "weakness", "fatigue", "tired", "sore throat", "throat pain",
		"cold", "flu", "runny nose", "body ache", "joint pain", "back pain",
		"chest tightness", "shortness of breath", "difficulty breathing"},
	intentBook: {"book", "appointment", "consult", "doctor", "schedule",
		"see a doctor", "meet doctor", "visit doctor", "consultation",
		"booking", "checkup", "see doctor",
		// Natural language additions
		"i want to see a doctor", "i need a consultation", "book me a doctor",
		"doctor appointment", "online doctor", "video call doctor", "consult now",
		"book doctor", "need a doctor", "find a doctor", "get doctor",
		"speak to a doctor", "talk to a doctor", "online consultation"},
	intentPharmacy: {
		// Core pharmacy nouns
		"medicine", "medicines", "pharmacy", "tablet", "tablets",
		"drug", "drugs", "capsule", "capsules", "syrup", "syrups",
		"meds", "medication", "medications", "prescription", "antibiotic",
		// Action phrases (natural language)
		"buy medicine", "buy medicines", "order medicine", "order medicines",
		"purchase medicine", "get medicine", "need medicine",
		"i want to buy", "i want to order", "i need to buy", "i need to order",
		"i would like to buy", "can i buy", "can i get",
		// Common generics / brands
		"paracetamol", "dolo", "crocin", "azithromycin", "metformin",
		"vitamin d", "vitamin c", "omega", "bp monitor", "glucometer",
		// Indian-English variations
		"pharmacy order", "medical store",
	},
	intentLab: {"lab", "test", "blood test", "urine test", "scan", "mri",
		"xray", "x-ray", "report", "sample", "diagnostic", "checkup",
		"blood work", "lab test", "blood sugar", "thyroid test", "lipid profile",
		"urine culture", "cbc", "complete blood count", "ecg", "ultrasound"},
	intentSupport: {"support", "complaint", "issue", "problem", "ticket",
		"refund", "billing", "payment issue", "appointment cancelled",
		"wrong charge", "not working", "help me",
		"complaint", "raise ticket", "i have a complaint", "something went wrong",
		"order not delivered", "didn't receive", "cancel order"},
	intentInsurance: {"insurance", "health plan", "coverage", "mediclaim",
		"health cover", "policy", "premium plan", "claim",
		"health insurance", "medical insurance", "family insurance",
		"critical illness", "term plan"},
	intentPremium: {"premium", "prime", "upgrade", "subscription", "plan",
		"unlimited", "vip", "membership",
		"go prime", "upgrade plan", "unlimited consultations"},
	intentRegister: {"register", "sign up", "signup", "create account",
		"join", "new account", "how to join",
		"open account", "create profile", "get started"},
	intentMedicineInfo: {"what is", "side effects", "dosage of", "dosage for",
		"how to take", "can i take", "interaction", "tablet for", "medicine for",
		"drug information", "mg dose", "when to take", "contraindication",
		"is it safe", "drug info", "medicine info", "tell me about",
		"information about", "uses of", "effects of",
		// Siddha / Ayurveda / Organic keywords
		"siddha", "ayurveda", "ayurvedic", "organic medicine", "herbal",
		"nilavembu", "kabasura", "ashwagandha", "triphala", "brahmi",
		"shatavari", "arjuna", "tulsi", "chyawanprash", "moringa", "murungai",
		"vendhayam", "fenugreek", "karpooravalli", "athimadhuram",
		"neem", "vembu", "giloy", "seendhil", "thirikatuku",
		"amla", "nelli", "bhringraj", "karisalanganni", "punarnava",
		"shankhpushpi", "turmeric", "manjal", "ginger", "inji",
		"aloe vera", "kathalai", "black seed", "karunseeragam",
		"cinnamon", "pattai", "coconut oil", "thengai ennai", "sesame oil", "nallennai",
		// Tamil script keywords
		"நிலவேம்பு", "கபசுர", "அஸ்வகந்தா", "திரிபலா", "பிரம்மி",
		"துளசி", "மஞ்சள்", "இஞ்சி", "கற்றாழை", "முருங்கை",
		// Malayalam script keywords
		"നിലവേമ്പ്", "അശ്വഗന്ധ", "ത്രിഫല", "ബ്രഹ്മി", "തുളസി",
		// Hindi script keywords
		"अश्वगंधा", "त्रिफला", "ब्राह्मी", "तुलसी", "हल्दी", "अदरक",
	},
	intentPayment: {"top up", "topup", "add money", "recharge", "wallet balance",
		"my balance", "pay for", "wallet", "how much in my wallet",
		"add funds", "pay appointment",
		// Natural language additions
		"check balance", "how much money", "add balance",
		"how much do i have", "my wallet", "check wallet",
		"wallet top up", "add to wallet"},
	intentMyAccount: {"my appointments", "my orders", "my profile",
		"my history", "upcoming appointment", "order status",
		"my prescriptions", "past appointment",
		// Natural language additions
		"my bookings", "show my appointments", "what appointments do i have",
		"order history", "my medicines", "my lab reports", "my account",
		"my records", "view appointments", "appointment history"},
}

// detectIntentLegacy scores each intent against the lowercased message and returns
// the highest-scoring match. Emergency always wins regardless of score.
func detectIntentLegacy(lower string) intent {
	// Emergency always wins — check first for safety
	for _, kw := range intentKeywords[intentEmergency] {
		if strings.Contains(lower, kw) {
			return intentEmergency
		}
	}

	scores := make(map[intent]int)
	for it, keywords := range intentKeywords {
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				scores[it]++
			}
		}
	}

	best, bestScore := intentUnknown, 0
	for it, score := range scores {
		if score > bestScore {
			bestScore = score
			best = it
		}
	}
	if bestScore == 0 {
		return intentUnknown
	}
	return best
}

// ── Guest capability gate ─────────────────────────────────────────────────────

// guestAllowedIntents lists the intents available to unauthenticated users.
var guestAllowedIntents = map[intent]bool{
	intentGreeting:     true,
	intentSymptom:      true, // info only — cannot book from symptom flow
	intentSupport:      true,
	intentInsurance:    true,
	intentRegister:     true,
	intentUnknown:      true,
	intentEmergency:    true, // always allowed — safety first
	intentMedicineInfo: true, // drug info is public
}

// guestGate returns a prompt and blocked=true when a guest tries a locked intent.
func guestGate(it intent) (msg string, blocked bool) {
	if guestAllowedIntents[it] {
		return "", false
	}
	return "🔒 To *" + string(it) + "*, you need a free Dr.Ayya account.\n\n" +
		"*Register in 30 seconds:*\n" +
		"👉 Visit *Dr.Ayya.in* → tap *Sign Up*\n" +
		"or type *register* and I'll guide you.\n\n" +
		"Already have an account? *Log in* to continue.", true
}

// isExitCommand returns true when the user wants to abort the current flow.
func isExitCommand(lower string) bool {
	switch lower {
	case "exit", "cancel", "menu", "restart", "0", "back", "home":
		return true
	}
	return false
}

// isYes returns true for common affirmative replies.
func isYes(lower string) bool {
	switch lower {
	case "yes", "y", "1", "ok", "sure", "confirm", "yeah", "yep", "haan", "ha":
		return true
	}
	return strings.Contains(lower, "yes") || strings.Contains(lower, "confirm")
}

// ── Pharmacy NLP helpers ──────────────────────────────────────────────────────

// pharmacyVerbPrefixes is ordered longest-first so greedy prefix matching works.
var pharmacyVerbPrefixes = []string{
	"i want to buy ", "i want to order ", "i would like to buy ",
	"i would like to order ", "i need to buy ", "i need to order ",
	"can i buy ", "can i get ", "can i order ",
	"please give me ", "give me ", "get me ",
	"buy me ", "order me ", "purchase ",
	"i need ", "i want ",
	"buy ", "order ", "get ", "need ",
}

// pharmacyGenericNouns are standalone words that represent a pharmacy intent
// but carry no specific product name.
var pharmacyGenericNouns = map[string]bool{
	"medicine": true, "medicines": true, "medication": true,
	"medications": true, "meds": true, "drugs": true,
	"tablets": true, "tablet": true, "capsule": true, "capsules": true,
	"syrup": true, "syrups": true, "pharmacy": true, "drug": true,
}

// pharmacyStopWords are filler words that add no product meaning.
var pharmacyStopWords = map[string]bool{
	"some": true, "any": true, "a": true, "an": true,
	"the": true, "please": true, "me": true, "my": true,
}

// extractProductQuery strips common action-verb phrases and generic intent
// nouns from a pharmacy-triggered message, returning the specific product
// name the user wants, or "" if the message is a generic "open pharmacy" phrase.
//
// Examples:
//
//	"I want to buy medicines"  → ""             (show pharmacy menu)
//	"buy paracetamol"          → "paracetamol"  (search)
//	"I need vitamin D tablets" → "vitamin d"    (search)
//	"order dolo 650"           → "dolo 650"     (search)
//	"medicine"                 → ""             (show pharmacy menu)
func extractProductQuery(msg string) string {
	lower := strings.ToLower(strings.TrimSpace(msg))

	// Strip leading verb phrase (longest-match wins)
	product := lower
	for _, prefix := range pharmacyVerbPrefixes {
		if strings.HasPrefix(lower, prefix) {
			product = strings.TrimSpace(lower[len(prefix):])
			break
		}
	}

	// Remove generic pharmacy nouns (e.g. "buy some medicines" → "some")
	words := strings.Fields(product)
	filtered := make([]string, 0, len(words))
	for _, w := range words {
		if !pharmacyGenericNouns[w] {
			filtered = append(filtered, w)
		}
	}

	// Remove filler / stop words that carry no product meaning
	clean := make([]string, 0, len(filtered))
	for _, w := range filtered {
		if !pharmacyStopWords[w] {
			clean = append(clean, w)
		}
	}

	product = strings.TrimSpace(strings.Join(clean, " "))
	return product // "" means generic intent → show menu; non-empty → search
}

// suggestExample returns a context-aware usage hint based on what the user typed.
// Used by unknownStart() to guide users who are stuck.
func suggestExample(msg string) string {
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "doctor") || strings.Contains(lower, "consult") || strings.Contains(lower, "appointment") {
		return "Try: *book a doctor* or *consult now*"
	}
	if strings.Contains(lower, "medicine") || strings.Contains(lower, "tablet") || strings.Contains(lower, "drug") || strings.Contains(lower, "pill") {
		return "Try: *buy paracetamol* or *pharmacy*"
	}
	if strings.Contains(lower, "pain") || strings.Contains(lower, "sick") || strings.Contains(lower, "fever") || strings.Contains(lower, "feel") || strings.Contains(lower, "hurt") {
		return "Try: *I have a headache* to check your symptoms"
	}
	if strings.Contains(lower, "lab") || strings.Contains(lower, "test") || strings.Contains(lower, "blood") || strings.Contains(lower, "report") {
		return "Try: *blood sugar test* or *lab tests*"
	}
	if strings.Contains(lower, "pay") || strings.Contains(lower, "money") || strings.Contains(lower, "wallet") || strings.Contains(lower, "balance") {
		return "Try: *check my wallet* or *top up*"
	}
	if strings.Contains(lower, "insur") || strings.Contains(lower, "cover") || strings.Contains(lower, "policy") {
		return "Try: *health insurance* or *family plan*"
	}
	return "Try: *symptom check*, *book a doctor*, or *pharmacy*"
}

package devi

// flow_medicine.go — Siddha, Ayurveda & Organic medicine information flow.
//
// Dr.Ayya sells ONLY Siddha, Ayurveda, and certified organic medicines.
// This file handles medicine info queries in four languages:
//   Tamil (தமிழ்) · Malayalam (മലയാളം) · Hindi (हिन्दी) · English
//
// Flow:
//   Step 0: Receive medicine name / health question
//         → Deterministic lookup (30+ medicines) → Formatted response
//         → LLM fallback for unlisted medicines
//
// Language detection: Unicode script ranges + transliteration keywords.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// MedicineInfoState tracks the medicine info flow.
type MedicineInfoState struct {
	Step     int    `json:"step"`
	MedName  string `json:"med_name"`
	Question string `json:"question"`
}

// ── Language support ──────────────────────────────────────────────────────────

// detectMessageLanguage returns the primary language of a message by checking
// Unicode script ranges (most reliable) and falling back to English.
func detectMessageLanguage(msg string) string {
	for _, r := range msg {
		if r >= 0x0B80 && r <= 0x0BFF {
			return "ta" // Tamil script
		}
		if r >= 0x0D00 && r <= 0x0D7F {
			return "ml" // Malayalam script
		}
		if r >= 0x0900 && r <= 0x097F {
			return "hi" // Devanagari (Hindi)
		}
	}
	// Transliterated Tamil / Malayalam keywords
	lower := strings.ToLower(msg)
	tamilHints := []string{"nilavembu", "kabasura", "vendhayam", "murungai",
		"karpooravalli", "athimadhuram", "nellikai", "thulasi", "inchi",
		"vembu", "manjal", "thengai", "nallennai"}
	for _, h := range tamilHints {
		if strings.Contains(lower, h) {
			return "ta"
		}
	}
	return "en"
}

// langLabel holds UI strings for one language.
type langLabel struct {
	Uses, SideEffects, Dosage, Notes, Interactions                string
	CategorySiddha, CategoryAyurveda, CategoryOrganic             string
	Disclaimer, BuyPrompt, ConsultPrompt                          string
}

// uiLabels maps language code → display strings.
var uiLabels = map[string]langLabel{
	"en": {
		Uses: "Uses", SideEffects: "Side Effects", Dosage: "Dosage",
		Notes: "Notes", Interactions: "Interactions",
		CategorySiddha:   "🌿 Siddha Medicine",
		CategoryAyurveda: "🌿 Ayurveda",
		CategoryOrganic:  "🌱 Organic / Natural",
		Disclaimer:       "⚕️ *Please consult a qualified Siddha or Ayurveda practitioner before use.*",
		BuyPrompt:        "Want to order",
		ConsultPrompt:    "Consult a doctor",
	},
	"ta": {
		Uses: "பயன்கள்", SideEffects: "பக்க விளைவுகள்", Dosage: "மருந்தளவு",
		Notes: "குறிப்பு", Interactions: "மருந்து இடைவினை",
		CategorySiddha:   "🌿 சித்த மருந்து",
		CategoryAyurveda: "🌿 ஆயுர்வேதம்",
		CategoryOrganic:  "🌱 இயற்கை மருந்து",
		Disclaimer:       "⚕️ *உபயோகிக்கும் முன் சித்த / ஆயுர்வேத மருத்துவரை அணுகவும்.*",
		BuyPrompt:        "ஆர்டர் செய்யவும்",
		ConsultPrompt:    "மருத்துவரை கலந்தாலோசிக்கவும்",
	},
	"ml": {
		Uses: "ഉപയോഗങ്ങൾ", SideEffects: "പാർശ്വഫലങ്ങൾ", Dosage: "അളവ്",
		Notes: "കുറിപ്പ്", Interactions: "ഇടപഴക്കം",
		CategorySiddha:   "🌿 സിദ്ധ ഔഷധം",
		CategoryAyurveda: "🌿 ആയുർവേദം",
		CategoryOrganic:  "🌱 ഓർഗാനിക്",
		Disclaimer:       "⚕️ *ഉപയോഗിക്കുന്നതിന് മുൻപ് സിദ്ധ / ആയുർവേദ വൈദ്യനെ കാണുക.*",
		BuyPrompt:        "ഓർഡർ ചെയ്യുക",
		ConsultPrompt:    "ഒരു ഡോക്ടറെ കാണുക",
	},
	"hi": {
		Uses: "उपयोग", SideEffects: "दुष्प्रभाव", Dosage: "खुराक",
		Notes: "नोट्स", Interactions: "इंटरेक्शन",
		CategorySiddha:   "🌿 सिद्ध औषधि",
		CategoryAyurveda: "🌿 आयुर्वेद",
		CategoryOrganic:  "🌱 जैविक / प्राकृतिक",
		Disclaimer:       "⚕️ *उपयोग से पहले सिद्ध / आयुर्वेद चिकित्सक से परामर्श करें।*",
		BuyPrompt:        "ऑर्डर करें",
		ConsultPrompt:    "डॉक्टर से सलाह लें",
	},
}

// ── Medicine database ─────────────────────────────────────────────────────────

// medEntry holds all data for one Siddha / Ayurveda / Organic medicine.
type medEntry struct {
	category     string   // "siddha" | "ayurveda" | "organic"
	nameEn       string   // Primary English name
	nameTa       string   // Tamil (Unicode)
	nameMl       string   // Malayalam (Unicode)
	nameHi       string   // Hindi (Unicode)
	keywords     []string // All searchable terms (English + transliterated + script)
	uses         string
	sideEffects  string
	dosage       string
	notes        string
	interactions string
}

// medicineDB is the Dr.Ayya Siddha / Ayurveda / Organic medicine catalogue.
var medicineDB = []medEntry{

	// ── SIDDHA MEDICINES ──────────────────────────────────────────────────────

	{
		category: "siddha",
		nameEn:   "Nilavembu Kudineer",
		nameTa:   "நிலவேம்பு குடிநீர்",
		nameMl:   "നിലവേമ്പ് കഷായം",
		nameHi:   "नीलवेम्बू काढ़ा",
		keywords: []string{
			"nilavembu", "nilavembu kudineer", "nilavembu kadineer",
			"நிலவேம்பு", "நிலவேம்பு குடிநீர்",
			"നിലവേമ്പ്", "नीलवेम्बू",
			"andrographis", "king of bitters", "dengue medicine", "fever kudineer",
		},
		uses:         "Fever, dengue, chikungunya, viral infections, malaria support, body pain, liver protection",
		sideEffects:  "Bitter taste; mild stomach upset if taken without food at first",
		dosage:       "30–60 ml twice daily (warm decoction), or as directed by your Siddha practitioner",
		notes:        "Best taken warm on an empty stomach. Highly effective during dengue and seasonal fevers. Do not use during pregnancy.",
		interactions: "May enhance the effect of blood-thinning medicines — monitor if on such medication.",
	},
	{
		category: "siddha",
		nameEn:   "Kabasura Kudineer",
		nameTa:   "கபசுர குடிநீர்",
		nameMl:   "കഫസൂര കഷായം",
		nameHi:   "कफसूर काढ़ा",
		keywords: []string{
			"kabasura", "kabasura kudineer", "kaba sura", "kabha sura",
			"கபசுர குடிநீர்", "கபசுர",
			"കഫസൂര", "कफसूर",
			"respiratory siddha", "immunity siddha", "covid siddha",
		},
		uses:         "Upper respiratory infections, cough, cold, flu, asthma support, immunity boosting; widely used post-COVID",
		sideEffects:  "Bitter taste; occasional mild stomach discomfort",
		dosage:       "60 ml twice daily after food, or as directed by practitioner",
		notes:        "Best taken warm. Avoid in pregnancy and lactation. Very popular during flu season and COVID recovery.",
		interactions: "No significant interactions reported. Consult doctor if on immunosuppressants.",
	},
	{
		category: "siddha",
		nameEn:   "Seendhil / Giloy (Siddha)",
		nameTa:   "சீந்தில் சர்க்கரை",
		nameMl:   "ചിറ്റമൃത്",
		nameHi:   "गिलोय / गुडूची",
		keywords: []string{
			"seendhil", "seenthil", "giloy", "guduchi", "guruchi", "guruge",
			"சீந்தில்", "ചിറ്റമൃത്", "गिलोय", "गुडूची",
			"tinospora cordifolia", "immunity booster siddha",
		},
		uses:         "Immunity booster, chronic fever, dengue recovery, diabetes support, liver detox, joint pain",
		sideEffects:  "Rare: mild stomach upset or low blood sugar in diabetics",
		dosage:       "5–10 ml juice or 1–2 g powder twice daily; decoction 30–60 ml; as directed",
		notes:        "One of the most powerful immunity herbs in Siddha and Ayurveda. Diabetics should monitor blood sugar.",
		interactions: "May lower blood sugar — caution if on diabetes medication.",
	},
	{
		category: "siddha",
		nameEn:   "Karpooravalli / Omavalli",
		nameTa:   "கற்பூரவல்லி",
		nameMl:   "ഓമവല്ലി / കർപ്പൂരവല്ലി",
		nameHi:   "पत्थरचूर / अजवाइन पत्ता",
		keywords: []string{
			"karpooravalli", "omavalli", "karpura valli", "omam leaf",
			"கற்பூரவல்லி", "ഓമവല്ലി", "पत्थरचूर",
			"plectranthus amboinicus", "indian borage", "country borage",
			"cough leaf", "cold leaf",
		},
		uses:         "Cough, cold, sore throat, nasal congestion, fever, respiratory infections in children and adults",
		sideEffects:  "Very safe; large amounts may cause mild stomach upset",
		dosage:       "Fresh leaf juice 5–10 ml with honey twice daily; or boiled in water as steam inhalation",
		notes:        "Excellent home remedy. Giving fresh leaf juice with honey is a traditional Tamil remedy for children's cough. Safe for all ages in moderate amounts.",
		interactions: "No significant interactions known.",
	},
	{
		category: "siddha",
		nameEn:   "Thirikatuku (Three Spices)",
		nameTa:   "திரிகடுகு",
		nameMl:   "ത്രികടുകം",
		nameHi:   "त्रिकटु",
		keywords: []string{
			"thirikatuku", "trikatu", "trikatuku", "three spices siddha",
			"திரிகடுகு", "ത്രികടുകം", "त्रिकटु",
			"ginger pepper pippali", "digestive siddha",
		},
		uses:         "Digestive disorders, indigestion, bloating, cough, cold, mucus, loss of appetite, metabolism boost",
		sideEffects:  "Hot/pungent sensation; avoid on an empty stomach. May cause heartburn in sensitive individuals.",
		dosage:       "250–500 mg powder twice daily with honey or warm water after food",
		notes:        "Contains dry ginger, black pepper, and long pepper (Pippali). Powerful digestive and respiratory tonic. Avoid in peptic ulcers.",
		interactions: "May enhance absorption of other medicines (increases bioavailability).",
	},
	{
		category: "siddha",
		nameEn:   "Athimadhuram / Licorice",
		nameTa:   "அதிமதுரம்",
		nameMl:   "അതിമധുരം / ഇരട്ടിമധുരം",
		nameHi:   "मुलेठी / यष्टिमधु",
		keywords: []string{
			"athimadhuram", "mulethi", "licorice", "liquorice", "yashtimadhur",
			"அதிமதுரம்", "അതിമധുരം", "मुलेठी",
			"glycyrrhiza glabra", "sweet root", "throat herb",
		},
		uses:         "Sore throat, cough, bronchitis, voice problems, acid reflux, stomach ulcers, skin health",
		sideEffects:  "Long-term high doses may raise blood pressure and cause water retention",
		dosage:       "1–3 g powder with honey twice daily; or decoction 30 ml twice daily",
		notes:        "Sweet-tasting medicine. A cornerstone throat remedy in Siddha. Do not use long-term in hypertension or pregnancy.",
		interactions: "May interact with blood pressure medicines and diuretics with prolonged use.",
	},
	{
		category: "siddha",
		nameEn:   "Neem / Vembu",
		nameTa:   "வேப்பிலை / வேம்பு",
		nameMl:   "ആര്യവേപ്പ് / നീം",
		nameHi:   "नीम",
		keywords: []string{
			"neem", "vembu", "veepam", "veppilai", "arya veppu",
			"வேம்பு", "வேப்பிலை", "ആര്യവേപ്പ്", "नीम",
			"azadirachta indica", "blood purifier", "skin medicine siddha",
		},
		uses:         "Skin diseases, acne, blood purification, diabetes management, wound healing, anti-fungal, immunity",
		sideEffects:  "High doses of neem oil/extract can cause liver toxicity; safe in food/leaf quantities",
		dosage:       "5–10 fresh leaves daily (chewed on empty stomach); neem capsules 250–500 mg twice daily",
		notes:        "One of the most versatile Siddha herbs. Neem leaves, bark, oil, and flowers all have medicinal uses. Do not give to infants or young children.",
		interactions: "May enhance effect of diabetes medications — monitor blood sugar.",
	},
	{
		category: "siddha",
		nameEn:   "Kandankathiri",
		nameTa:   "கண்டங்கத்திரி",
		nameMl:   "കണ്ടകാരി",
		nameHi:   "कंटकारी / छोटी कटेरी",
		keywords: []string{
			"kandankathiri", "kandakathari", "kantakari", "katakathari",
			"கண்டங்கத்திரி", "കണ്ടകാരി", "कंटकारी",
			"solanum xanthocarpum", "asthma herb", "respiratory siddha",
		},
		uses:         "Asthma, bronchitis, chronic cough, respiratory congestion, fever, urinary problems",
		sideEffects:  "Bitter taste; avoid in large doses during pregnancy",
		dosage:       "Decoction 30–60 ml twice daily; or as directed by Siddha practitioner",
		notes:        "A primary herb for respiratory diseases in Siddha medicine. Often used in Kabasura Kudineer.",
		interactions: "No significant interactions reported.",
	},

	// ── AYURVEDIC MEDICINES ───────────────────────────────────────────────────

	{
		category: "ayurveda",
		nameEn:   "Ashwagandha",
		nameTa:   "அஸ்வகந்தா / அமுக்கிரா",
		nameMl:   "അശ്വഗന്ധ",
		nameHi:   "अश्वगंधा",
		keywords: []string{
			"ashwagandha", "aswagandha", "withania somnifera",
			"amukkiraa", "amukura", "indian ginseng",
			"அஸ்வகந்தா", "அமுக்கிரா", "അശ്വഗന്ധ", "अश्वगंधा",
			"stress relief herb", "energy ayurveda",
		},
		uses:         "Stress, anxiety, fatigue, low energy, immunity, testosterone support, thyroid health, insomnia",
		sideEffects:  "Mild drowsiness in high doses; occasional stomach upset",
		dosage:       "300–600 mg extract (KSM-66 or Sensoril) once or twice daily; or 1–2 g root powder with warm milk at night",
		notes:        "Best taken with warm milk at bedtime. One of Ayurveda's most researched adaptogens. Avoid in pregnancy and autoimmune conditions.",
		interactions: "May enhance sedative and thyroid medications — consult doctor if on such medicines.",
	},
	{
		category: "ayurveda",
		nameEn:   "Triphala",
		nameTa:   "திரிபலா / த்ரிபலா சூரணம்",
		nameMl:   "ത്രിഫല",
		nameHi:   "त्रिफला",
		keywords: []string{
			"triphala", "tripala", "triphala churna", "triphala choornam",
			"திரிபலா", "ത്രിഫല", "त्रिफला",
			"haritaki bibhitaki amalaki", "three fruits ayurveda", "bowel ayurveda",
		},
		uses:         "Constipation, digestion, gut health, detox, eye health, immunity, weight management, diabetes support",
		sideEffects:  "May cause loose stools initially; mild stomach cramping when starting",
		dosage:       "1–3 g powder with warm water at bedtime; or as directed by Ayurveda practitioner",
		notes:        "One of Ayurveda's most important formulas. Contains equal parts Haritaki, Bibhitaki, and Amalaki. Start with a small dose and increase gradually.",
		interactions: "May increase effect of blood thinners. Space 2 hours from other medications.",
	},
	{
		category: "ayurveda",
		nameEn:   "Brahmi",
		nameTa:   "பிரம்மி",
		nameMl:   "ബ്രഹ്മി",
		nameHi:   "ब्राह्मी",
		keywords: []string{
			"brahmi", "bacopa monnieri", "brahmi capsule",
			"பிரம்மி", "ബ്രഹ്മി", "ब्राह्मी",
			"memory herb", "brain ayurveda", "cognitive ayurveda", "study herb",
		},
		uses:         "Memory, concentration, anxiety, ADHD support, brain health, learning, stress, epilepsy support",
		sideEffects:  "Mild nausea or stomach upset (take with food); drowsiness at high doses",
		dosage:       "300–600 mg standardised extract twice daily; or 1–2 g powder with milk",
		notes:        "Best taken with food to reduce nausea. Results build over 6–12 weeks. Popular for students, elderly, and anyone wanting better focus.",
		interactions: "May enhance sedatives. Avoid combining with thyroid medications without guidance.",
	},
	{
		category: "ayurveda",
		nameEn:   "Shatavari",
		nameTa:   "சதாவரி",
		nameMl:   "ശതാവരി",
		nameHi:   "शतावरी",
		keywords: []string{
			"shatavari", "satavari", "asparagus racemosus",
			"சதாவரி", "ശതാവരி", "शतावरी",
			"women health ayurveda", "hormonal balance", "lactation herb",
		},
		uses:         "Women's hormonal health, PCOS, menstrual disorders, fertility, breast milk production, menopause, immunity",
		sideEffects:  "Rare: allergic reaction in those sensitive to asparagus; mild bloating",
		dosage:       "500 mg – 1 g powder twice daily with warm milk; or as directed",
		notes:        "The premier women's tonic in Ayurveda. Safe during breastfeeding. Also beneficial for men's immunity and digestion.",
		interactions: "May interact with diuretics — use caution.",
	},
	{
		category: "ayurveda",
		nameEn:   "Arjuna",
		nameTa:   "அர்ஜுன பட்டை",
		nameMl:   "അർജ്ജുന",
		nameHi:   "अर्जुन",
		keywords: []string{
			"arjuna", "arjuna bark", "terminalia arjuna",
			"அர்ஜுனா", "அர்ஜுன பட்டை", "അർജ്ജുന", "अर्जुन",
			"heart ayurveda", "cardiac ayurveda", "blood pressure ayurveda",
		},
		uses:         "Heart health, high blood pressure, high cholesterol, heart failure support, chest pain, blood thinning",
		sideEffects:  "Generally safe; may cause mild stomach upset",
		dosage:       "500 mg – 1 g bark powder twice daily with water or milk; or 60 ml decoction",
		notes:        "One of Ayurveda's primary cardiac herbs. Works best as a long-term tonic. Should complement, not replace, prescribed heart medicines.",
		interactions: "May interact with blood pressure and heart medications — inform your doctor.",
	},
	{
		category: "ayurveda",
		nameEn:   "Tulsi / Holy Basil",
		nameTa:   "துளசி",
		nameMl:   "തുളസി",
		nameHi:   "तुलसी",
		keywords: []string{
			"tulsi", "thulasi", "holy basil", "ocimum tenuiflorum",
			"துளசி", "തുളസി", "तुलसी",
			"immunity herb", "respiratory herb", "cold cough tulsi",
		},
		uses:         "Cough, cold, fever, respiratory infections, immunity, stress, skin health, ear infections",
		sideEffects:  "Very safe in food quantities; high medicinal doses may thin blood slightly",
		dosage:       "5–10 fresh leaves daily; or 300–600 mg extract; or tulsi tea 2–3 cups daily",
		notes:        "Sacred and versatile herb. Best taken fresh. Tulsi tea with ginger and honey is a classic immunity drink. Avoid very high doses in pregnancy.",
		interactions: "May mildly thin blood — use caution with blood-thinning medicines.",
	},
	{
		category: "ayurveda",
		nameEn:   "Chyawanprash",
		nameTa:   "சவ்யனப்ராஷ் / சவானப்ராஸ்",
		nameMl:   "ച്യവനപ്രാശം",
		nameHi:   "च्यवनप्राश",
		keywords: []string{
			"chyawanprash", "chyavanprash", "chawanprash",
			"சவ்யனப்ராஷ்", "ച്യവനപ്രാശം", "च्यवनप्राश",
			"immunity ayurveda", "rasayana", "amla jam",
		},
		uses:         "Overall immunity, respiratory health, energy, anti-aging, digestion, mental clarity",
		sideEffects:  "High sugar content — diabetics should choose sugar-free variants",
		dosage:       "1–2 teaspoons daily with warm milk, best in winter mornings",
		notes:        "Ancient Ayurvedic rasayana (rejuvenator). Contains Amla and 40+ herbs. Safe for all ages. Look for sugar-free options if diabetic.",
		interactions: "No significant interactions. Sugar-free versions recommended for diabetics.",
	},
	{
		category: "ayurveda",
		nameEn:   "Amla / Nelli / Indian Gooseberry",
		nameTa:   "நெல்லிக்காய்",
		nameMl:   "നെല്ലിക്ക",
		nameHi:   "आंवला",
		keywords: []string{
			"amla", "nelli", "nellikkai", "amalaki", "indian gooseberry",
			"நெல்லிக்காய்", "நெல்லி", "നെല്ലിക്ക", "आंवला",
			"vitamin c ayurveda", "hair health ayurveda",
		},
		uses:         "Rich Vitamin C source, immunity, hair growth, skin health, digestion, liver detox, cholesterol",
		sideEffects:  "Very safe; high doses may cause loose stools",
		dosage:       "1–2 fresh fruits daily; or 1–3 g powder; or 20 ml fresh juice",
		notes:        "One of the most potent natural sources of Vitamin C. A cornerstone of Triphala. Excellent for hair and skin when taken regularly.",
		interactions: "May enhance blood-thinning effects — monitor if on warfarin.",
	},
	{
		category: "ayurveda",
		nameEn:   "Shankhpushpi",
		nameTa:   "சங்குப்புஷ்பி",
		nameMl:   "ശംഖുപുഷ്പം",
		nameHi:   "शंखपुष्पी",
		keywords: []string{
			"shankhpushpi", "shankupushpi", "convolvulus pluricaulis",
			"சங்குப்புஷ்பி", "ശംഖുപുഷ്പം", "शंखपुष्पी",
			"memory ayurveda", "anxiety ayurveda", "sleep ayurveda",
		},
		uses:         "Memory enhancement, anxiety, stress, insomnia, ADHD support, epilepsy support, mental clarity",
		sideEffects:  "Rare; mild drowsiness at high doses",
		dosage:       "5–10 ml syrup or 250–500 mg powder twice daily with milk",
		notes:        "Medhya Rasayana (brain tonic) in Ayurveda. Works best with consistent use over 4–8 weeks.",
		interactions: "May enhance phenytoin (seizure medicine) levels — inform your doctor.",
	},
	{
		category: "ayurveda",
		nameEn:   "Punarnava",
		nameTa:   "முக்கரட்டை / பூனர்னவா",
		nameMl:   "താഴ്ചക്കൂർ / പൂനർനവ",
		nameHi:   "पुनर्नवा",
		keywords: []string{
			"punarnava", "mukarattai", "boerhavia diffusa",
			"முக்கரட்டை", "പൂനർനവ", "पुनर्नवा",
			"kidney ayurveda", "joint pain ayurveda", "water retention herb",
		},
		uses:         "Kidney health, water retention (edema), joint pain, liver support, urinary tract infections, anemia",
		sideEffects:  "Rare; very safe in recommended doses",
		dosage:       "3–6 g root powder daily with water; or 30 ml decoction twice daily",
		notes:        "Excellent kidney and liver tonic. A natural diuretic. Helps reduce swelling in legs and ankles.",
		interactions: "May enhance diuretic medications — monitor fluid levels.",
	},
	{
		category: "ayurveda",
		nameEn:   "Bhringraj",
		nameTa:   "கரிசலாங்கண்ணி",
		nameMl:   "കയ്യോന്നി / ഭൃംഗരാജ",
		nameHi:   "भृंगराज",
		keywords: []string{
			"bhringraj", "bhringaraj", "karisalanganni", "eclipta alba",
			"கரிசலாங்கண்ணி", "കയ്യോന്നി", "भृंगराज",
			"hair oil ayurveda", "hair fall herb",
		},
		uses:         "Hair fall, premature greying, scalp health, liver detox, skin health, memory",
		sideEffects:  "Very safe; topical use occasionally causes scalp irritation in sensitive persons",
		dosage:       "500 mg – 1 g powder twice daily; or Bhringraj oil applied to scalp 2–3 times weekly",
		notes:        "The king of hair herbs in Ayurveda. Best used both internally (capsules/powder) and externally (oil). Consistent use for 3+ months gives best results.",
		interactions: "No significant interactions known.",
	},

	// ── ORGANIC / NATURAL ────────────────────────────────────────────────────

	{
		category: "organic",
		nameEn:   "Turmeric / Manjal",
		nameTa:   "மஞ்சள்",
		nameMl:   "മഞ്ഞൾ",
		nameHi:   "हल्दी",
		keywords: []string{
			"turmeric", "manjal", "haldi", "curcuma longa",
			"மஞ்சள்", "மஞ்சள் பால்", "മഞ്ഞൾ", "हल्दी",
			"curcumin", "golden milk", "anti-inflammatory", "paal manjal",
		},
		uses:         "Anti-inflammatory, joint pain, arthritis, immunity, wound healing, skin health, digestion, liver support",
		sideEffects:  "High doses may cause stomach upset; may thin blood slightly",
		dosage:       "1/2 – 1 teaspoon powder daily in milk or food; curcumin supplements: 500–1000 mg daily",
		notes:        "Curcumin absorption increases with black pepper (piperine) and fat. Golden milk (turmeric + milk + honey) at night is an excellent daily ritual.",
		interactions: "May increase blood-thinning effects of medications. Avoid high doses before surgery.",
	},
	{
		category: "organic",
		nameEn:   "Ginger / Inji",
		nameTa:   "இஞ்சி",
		nameMl:   "ഇഞ്ചി",
		nameHi:   "अदरक",
		keywords: []string{
			"ginger", "inji", "inchi", "adrak", "zingiber officinale",
			"இஞ்சி", "ഇഞ്ചി", "अदरक",
			"nausea ginger", "digestion ginger", "anti-nausea",
		},
		uses:         "Nausea, vomiting (including morning sickness), indigestion, bloating, cold, cough, joint pain, menstrual pain",
		sideEffects:  "Heartburn or stomach irritation in high doses; avoid very high doses in pregnancy",
		dosage:       "1–2 g fresh ginger daily; or ginger tea (boil 1 cm piece in water); or 250–500 mg extract",
		notes:        "One of the safest and most versatile herbs. Excellent for nausea and motion sickness. Ginger + lemon + honey tea is a powerful cold remedy.",
		interactions: "High doses may mildly thin blood — caution with blood-thinning medicines.",
	},
	{
		category: "organic",
		nameEn:   "Honey / Then / Madhu",
		nameTa:   "தேன்",
		nameMl:   "തേൻ",
		nameHi:   "शहद",
		keywords: []string{
			"honey", "then", "madhu", "shahed",
			"தேன்", "தேன் மருந்து", "തേൻ", "शहद",
			"organic honey", "raw honey", "cough honey",
		},
		uses:         "Cough and sore throat soothing, wound healing, energy, immunity, skin healing, natural sweetener",
		sideEffects:  "Safe; never give to infants under 1 year (risk of botulism)",
		dosage:       "1–2 tablespoons daily; mixed with ginger, turmeric, or lemon for best effect",
		notes:        "Raw, unprocessed honey is best. An excellent carrier for Siddha/Ayurveda medicines. Never heat honey (destroys medicinal properties).",
		interactions: "No significant interactions. Avoid giving to children under 12 months.",
	},
	{
		category: "organic",
		nameEn:   "Moringa / Murungai",
		nameTa:   "முருங்கை",
		nameMl:   "മുരിങ്ങ",
		nameHi:   "सहजन / मोरिंगा",
		keywords: []string{
			"moringa", "murungai", "murunga", "drumstick tree", "sahajan",
			"முருங்கை", "முருங்கை இலை", "மருங்கை", "മുരിങ്ങ", "सहजन",
			"moringa leaf", "moringa powder", "superfood",
		},
		uses:         "Nutritional supplement, energy, iron deficiency, blood sugar, inflammation, lactation support, bone health",
		sideEffects:  "Root and bark can cause uterine contractions — pregnant women should avoid",
		dosage:       "1–2 teaspoons moringa leaf powder daily; or fresh leaves in food; or 500 mg capsules",
		notes:        "Called 'miracle tree' — moringa leaves have more Vitamin C than oranges, more iron than spinach. Safe for most people when used as food.",
		interactions: "May lower blood sugar and blood pressure — monitor if on related medications.",
	},
	{
		category: "organic",
		nameEn:   "Aloe Vera / Kathalai",
		nameTa:   "கற்றாழை",
		nameMl:   "കറ്റാർ‌വാഴ",
		nameHi:   "एलोवेरा / घृतकुमारी",
		keywords: []string{
			"aloe vera", "kathalai", "kattralai", "ghrit kumari",
			"கற்றாழை", "கற்றாழை ஜெல்", "കറ്റാർ‌വാഴ", "एलोवेरा",
			"aloe gel", "aloe juice",
		},
		uses:         "Skin burns, wounds, skin moisturising, constipation (juice), digestion, acid reflux, immunity",
		sideEffects:  "Latex (yellow layer) is a strong laxative — avoid internal use of latex. Gel is safe.",
		dosage:       "Topical gel: apply 2–3 times daily. Juice: 20–30 ml twice daily (gel only, not latex)",
		notes:        "Use only the clear inner gel — remove the bitter yellow latex layer before use. Aloe juice is a mild digestive tonic.",
		interactions: "Aloe latex may interact with digoxin and diuretics.",
	},
	{
		category: "organic",
		nameEn:   "Fenugreek / Vendhayam",
		nameTa:   "வெந்தயம்",
		nameMl:   "ഉലുവ",
		nameHi:   "मेथी",
		keywords: []string{
			"fenugreek", "vendhayam", "uluva", "methi",
			"வெந்தயம்", "ഉലുവ", "मेथी",
			"diabetes herb", "blood sugar organic", "hair fenugreek",
		},
		uses:         "Blood sugar control (diabetes), cholesterol, digestion, breast milk production, hair growth, joint pain",
		sideEffects:  "Strong smell in sweat/urine; stomach upset in large amounts; low blood sugar if overused",
		dosage:       "1 teaspoon soaked seeds daily (overnight-soaked water in the morning); or 500–1000 mg capsules",
		notes:        "Excellent daily habit for diabetics and new mothers. Soaked overnight in water and drinking in the morning is a classic Tamil remedy.",
		interactions: "May lower blood sugar — caution if on diabetes medication. Monitor levels.",
	},
	{
		category: "organic",
		nameEn:   "Black Seed / Karunseeragam / Kalonji",
		nameTa:   "கருஞ்சீரகம்",
		nameMl:   "കരിഞ്ജീരകം",
		nameHi:   "कलौंजी / काला जीरा",
		keywords: []string{
			"black seed", "kalonji", "karunseeragam", "nigella sativa",
			"கருஞ்சீரகம்", "കരിഞ്ജീരകം", "कलौंजी",
			"black cumin", "habbatus sauda", "immunity seed",
		},
		uses:         "Immunity booster, respiratory health, blood sugar, allergies, blood pressure, anti-fungal, anti-bacterial",
		sideEffects:  "Large amounts may cause low blood sugar or blood pressure; mild stomach upset",
		dosage:       "1/2 – 1 teaspoon seeds or oil daily; mix with honey for palatability",
		notes:        "Prophetic medicine — often called 'the cure for everything except death'. Cold-pressed black seed oil is most potent.",
		interactions: "May enhance blood-thinning and blood-pressure-lowering medicines.",
	},
	{
		category: "organic",
		nameEn:   "Cinnamon / Pattai",
		nameTa:   "பட்டை",
		nameMl:   "ഇലവംഗപ്പട്ട",
		nameHi:   "दालचीनी",
		keywords: []string{
			"cinnamon", "pattai", "dalchini", "cinnamomum",
			"பட்டை", "ഇലവംഗപ്പട്ട", "दालचीनी",
			"blood sugar cinnamon", "diabetes cinnamon",
		},
		uses:         "Blood sugar control, cholesterol, anti-inflammatory, digestion, cold, warming tonic",
		sideEffects:  "High doses of Cassia cinnamon may harm liver — prefer Ceylon cinnamon",
		dosage:       "1/2 – 1 teaspoon powder daily in food, water, or tea; or 500 mg capsules",
		notes:        "Choose Ceylon cinnamon (true cinnamon) over Cassia for daily use. Adding to morning tea or oatmeal is an easy daily habit.",
		interactions: "May enhance the effect of diabetes medications — monitor blood sugar.",
	},
	{
		category: "organic",
		nameEn:   "Coconut Oil / Thengai Ennai",
		nameTa:   "தேங்காய் எண்ணெய்",
		nameMl:   "തേങ്ങ എണ്ണ",
		nameHi:   "नारियल तेल",
		keywords: []string{
			"coconut oil", "thengai ennai", "thenga enna", "naariyal tel",
			"தேங்காய் எண்ணெய்", "தேங்காய் எண்ணெய்", "തേങ്ങ എണ്ണ", "नारियल तेल",
			"virgin coconut oil", "vco", "oil pulling",
		},
		uses:         "Skin moisturiser, hair health, oil pulling (oral hygiene), cooking, wound healing, anti-fungal (topical)",
		sideEffects:  "High in saturated fat — excess consumption may raise LDL cholesterol in some people",
		dosage:       "Cooking: 1–2 tablespoons daily. Oil pulling: 1 tablespoon swished for 10–15 minutes before brushing.",
		notes:        "Use virgin (cold-pressed) coconut oil for best medicinal benefits. A staple in Tamil and Kerala traditional medicine.",
		interactions: "No significant medicinal interactions.",
	},
	{
		category: "organic",
		nameEn:   "Sesame Oil / Nallennai",
		nameTa:   "நல்லெண்ணெய்",
		nameMl:   "എള്ളെണ്ണ",
		nameHi:   "तिल का तेल",
		keywords: []string{
			"sesame oil", "nallennai", "gingelly oil", "yellenne",
			"நல்லெண்ணெய்", "எള്ളെണ്ണ", "तिल का तेल",
			"joint massage oil", "sesame massage",
		},
		uses:         "Joint pain massage, skin nourishment, hair health, oil pulling, traditional cooking, stress relief (head massage)",
		sideEffects:  "Sesame allergy possible (rare) — do a patch test before full use",
		dosage:       "Massage: warm oil applied to joints/body, leave 30 min before bath. Head massage: 2–3 times weekly.",
		notes:        "Traditional Siddha Abhyanga (oil massage) base oil. Warm sesame oil massage is recommended before bathing in Siddha medicine.",
		interactions: "No significant interactions.",
	},
}

// ── Entry points ──────────────────────────────────────────────────────────────

func (e *Engine) medicineInfoStart(ctx context.Context, uc *UserContext, msg string, state *sessionState) FlowResult {
	st := MedicineInfoState{Step: 0, Question: msg}
	state.Intent = string(intentMedicineInfo)
	state.Payload = marshalPayload(st)
	return e.processMedicineQuery(ctx, uc, state, &st, msg)
}

func (e *Engine) medicineInfoContinue(ctx context.Context, uc *UserContext, state *sessionState, msg string) FlowResult {
	var st MedicineInfoState
	_ = unmarshalGeneric(state.Payload, &st)
	return e.processMedicineQuery(ctx, uc, state, &st, msg)
}

// ── Core handler ──────────────────────────────────────────────────────────────

func (e *Engine) processMedicineQuery(ctx context.Context, uc *UserContext, state *sessionState, st *MedicineInfoState, msg string) FlowResult {
	lang := detectMessageLanguage(msg)
	lower := strings.ToLower(strings.TrimSpace(msg))

	// Deterministic lookup
	response, medName := lookupMedicineKnowledge(lower, lang)
	if response != "" {
		lbl := uiLabels[lang]
		actions := []ChatAction{btn(lbl.BuyPrompt+" "+medName, "pharmacy")}
		if uc.CanBook() {
			actions = append(actions, btn(lbl.ConsultPrompt, "book"))
		}
		return FlowResult{
			Message: response + "\n\n" + lbl.Disclaimer,
			Actions: actions,
			Done:    true,
		}
	}

	// LLM fallback (handles unlisted medicines in any language)
	if llmAnswer := e.askLLMForMedicineInfo(ctx, uc, msg); llmAnswer != "" {
		lbl := uiLabels[lang]
		return FlowResult{
			Message: llmAnswer,
			Actions: []ChatAction{btn(lbl.BuyPrompt, "pharmacy"), btn(lbl.ConsultPrompt, "book")},
			Done:    true,
		}
	}

	// Generic fallback
	lbl := uiLabels[lang]
	_ = lbl
	return done(fmt.Sprintf(
		"💊 *%s*\n\n"+
			"I don't have detailed information on this medicine in my knowledge base yet.\n\n"+
			"For accurate information:\n"+
			"1️⃣  Ask your Dr.Ayya Siddha / Ayurveda practitioner\n"+
			"2️⃣  Check the product label\n"+
			"3️⃣  Visit *Dr.Ayya.in/medicines* for our full catalogue\n\n"+
			"Type *pharmacy* to browse our Siddha & Ayurveda range | *book* to consult | *menu* to go back.", msg,
	))
}

// ── Knowledge base lookup ─────────────────────────────────────────────────────

// lookupMedicineKnowledge searches medicineDB for the user's query (any language).
// Returns (formatted response, display name) or ("", "") if no match.
func lookupMedicineKnowledge(lower, lang string) (string, string) {
	lbl, ok := uiLabels[lang]
	if !ok {
		lbl = uiLabels["en"]
	}

	for _, m := range medicineDB {
		// Match against keywords
		matched := false
		for _, kw := range m.keywords {
			if strings.Contains(lower, strings.ToLower(kw)) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}

		// Category label
		var catLabel string
		switch m.category {
		case "siddha":
			catLabel = lbl.CategorySiddha
		case "ayurveda":
			catLabel = lbl.CategoryAyurveda
		default:
			catLabel = lbl.CategoryOrganic
		}

		// Local name in the detected language
		localName := localMedName(m, lang)

		// Format response
		response := fmt.Sprintf("💊 *%s*\n", m.nameEn)
		if localName != m.nameEn {
			response += fmt.Sprintf("   (%s)\n", localName)
		}
		response += fmt.Sprintf("%s\n\n", catLabel)
		response += fmt.Sprintf("✅ *%s:* %s\n\n", lbl.Uses, m.uses)
		response += fmt.Sprintf("⚠️ *%s:* %s\n\n", lbl.SideEffects, m.sideEffects)
		response += fmt.Sprintf("💉 *%s:* %s\n\n", lbl.Dosage, m.dosage)
		response += fmt.Sprintf("📋 *%s:* %s", lbl.Notes, m.notes)
		if m.interactions != "" {
			response += fmt.Sprintf("\n\n🔗 *%s:* %s", lbl.Interactions, m.interactions)
		}

		return response, m.nameEn
	}

	return "", ""
}

// localMedName returns the medicine name in the detected language.
func localMedName(m medEntry, lang string) string {
	switch lang {
	case "ta":
		if m.nameTa != "" {
			return m.nameTa
		}
	case "ml":
		if m.nameMl != "" {
			return m.nameMl
		}
	case "hi":
		if m.nameHi != "" {
			return m.nameHi
		}
	}
	return m.nameEn
}

// askLLMForMedicineInfo routes medicine questions through the LLM if configured.
func (e *Engine) askLLMForMedicineInfo(ctx context.Context, uc *UserContext, query string) string {
	return e.processWithLLM(ctx, uc, query)
}

// unmarshalGeneric is a generic JSON unmarshaler helper.
func unmarshalGeneric(raw []byte, v interface{}) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, v)
}

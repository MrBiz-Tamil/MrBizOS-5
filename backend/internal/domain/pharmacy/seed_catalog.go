package pharmacy

import (
	"drayya-core/internal/money"

	"gorm.io/gorm"
)

// SeedAYUSHCatalog populates a starter Siddha + Ayurveda + Wellness catalog so
// the doctor's autocomplete and the pharmacy auto-cart have something real to
// work with on a fresh deploy. Idempotent — uses FirstOrCreate keyed on (name,
// brand) so re-running it doesn't duplicate.
//
// Tamil names are common Siddha preparations; Hindi names are common Ayurvedic
// ones. Prices in paise (₹). GST defaults to 5% (the AYUSH slab).
func SeedAYUSHCatalog(db *gorm.DB) {
	if db == nil {
		return
	}
	catalog := []Medicine{
		// ── Siddha (Tamil) ────────────────────────────────────────────────────
		{
			Name: "Nilavembu Kashayam", NameTa: "நிலவேம்பு கசாயம்",
			Brand: "Dr.Ayya", IsInhouseBrand: true,
			Category: "siddha", Form: "kashayam", PackSize: "100 ml",
			Description:        "Bitter decoction traditionally used for fever and immune support.",
			Indications:        "Viral fever, dengue prophylaxis, body ache",
			DosageInstructions: "15 ml twice daily after meals with warm water.",
			StorageInstructions: "Store below 25°C, away from direct sunlight.",
			Prices:             money.RegionalPricing{"INR": 12000},
			GSTPercent:         5, HSNCode: "30049011",
			IsActive: true,
		},
		{
			Name: "Adathodai Manappagu", NameTa: "ஆடாதோடை மணப்பாகு",
			Brand: "Dr.Ayya", IsInhouseBrand: true,
			Category: "siddha", Form: "manappagu", PackSize: "200 ml",
			Description: "Sweet syrup for productive cough and respiratory comfort.",
			Indications: "Cough with phlegm, sore throat, mild bronchitis",
			DosageInstructions: "5–10 ml twice daily after meals.",
			Prices:     money.RegionalPricing{"INR": 18000},
			GSTPercent: 5, HSNCode: "30049011",
			IsActive: true,
		},
		{
			Name: "Thripala Chooranam", NameTa: "திரிபலா சூரணம்",
			Brand: "IMPCOPS", Category: "siddha", Form: "chooranam", PackSize: "100 g",
			Description: "Classical three-fruit churna for gentle bowel regulation.",
			Indications: "Constipation, indigestion, mild detox",
			DosageInstructions: "3–5 g at bedtime with warm water.",
			Prices:     money.RegionalPricing{"INR": 9500},
			GSTPercent: 5, HSNCode: "30049011",
			IsActive: true,
		},
		{
			Name: "Amukkura Chooranam", NameTa: "அமுக்கரா சூரணம்",
			Brand: "Dr.Ayya", IsInhouseBrand: true,
			Category: "siddha", Form: "chooranam", PackSize: "100 g",
			Description: "Ashwagandha-equivalent rejuvenative powder.",
			Indications: "Stress, fatigue, sleep support, general weakness",
			DosageInstructions: "3 g twice daily with milk after meals.",
			Prices:     money.RegionalPricing{"INR": 14000},
			GSTPercent: 5, HSNCode: "30049011",
			IsActive: true,
		},
		{
			Name: "Karpooradi Thailam", NameTa: "கற்பூராதி தைலம்",
			Brand: "SKM Siddha", Category: "siddha", Form: "thailam", PackSize: "100 ml",
			Description: "Camphor-based oil for muscular pain and stiffness.",
			Indications: "Joint pain, muscle stiffness, headache (external use)",
			DosageInstructions: "Apply locally 2–3 times daily; for external use only.",
			Prices:     money.RegionalPricing{"INR": 11000},
			GSTPercent: 5, HSNCode: "30049011",
			IsActive: true,
		},
		{
			Name: "Pavala Parpam", NameTa: "பவள பற்பம்",
			Brand: "IMPCOPS", Category: "siddha", Form: "parpam", PackSize: "10 g",
			Description: "Calcined coral preparation; supplemental calcium.",
			Indications: "Hyperacidity, calcium deficiency, brittle nails",
			DosageInstructions: "65 mg twice daily with honey under physician guidance.",
			Prices:     money.RegionalPricing{"INR": 32000},
			GSTPercent: 5, HSNCode: "30049011",
			IsActive: true,
		},

		// ── Ayurveda (Hindi) ──────────────────────────────────────────────────
		{
			Name: "Triphala Churna", NameHi: "त्रिफला चूर्ण",
			Brand: "Patanjali", Category: "ayurveda", Form: "churna", PackSize: "100 g",
			Description: "Equal-part Amla + Haritaki + Bibhitaki powder.",
			Indications: "Constipation, eye health, mild detox",
			DosageInstructions: "3–6 g at bedtime with warm water.",
			Prices:     money.RegionalPricing{"INR": 8000},
			GSTPercent: 5, HSNCode: "30049011",
			IsActive: true,
		},
		{
			Name: "Triphala Churna", NameHi: "त्रिफला चूर्ण",
			Brand: "Dr.Ayya", IsInhouseBrand: true,
			Category: "ayurveda", Form: "churna", PackSize: "100 g",
			Description: "Stone-ground Triphala — Dr.Ayya house blend.",
			Indications: "Constipation, mild detox, hair and eye support",
			DosageInstructions: "3–6 g at bedtime with warm water.",
			Prices:     money.RegionalPricing{"INR": 9500},
			GSTPercent: 5, HSNCode: "30049011",
			IsActive: true,
		},
		{
			Name: "Ashwagandha Capsules", NameHi: "अश्वगंधा कैप्सूल",
			Brand: "Himalaya", Category: "ayurveda", Form: "capsule", PackSize: "60 capsules",
			Description: "Standardised Withania somnifera root extract, 500 mg.",
			StrengthValue: "500", StrengthUnit: "mg",
			Indications: "Stress, sleep support, muscle recovery",
			DosageInstructions: "1 capsule twice daily after meals.",
			Prices:     money.RegionalPricing{"INR": 28000},
			GSTPercent: 5, HSNCode: "30049011",
			IsActive: true,
		},
		{
			Name: "Brahmi Ghrita", NameHi: "ब्राह्मी घृत",
			Brand: "Kottakkal", Category: "ayurveda", Form: "lehyam", PackSize: "150 g",
			Description: "Medicated ghee with Bacopa monnieri for cognitive support.",
			Indications: "Memory support, focus, mild anxiety",
			DosageInstructions: "1 tsp morning and evening on empty stomach.",
			Prices:     money.RegionalPricing{"INR": 42000},
			GSTPercent: 5, HSNCode: "30049011",
			IsActive: true,
		},
		{
			Name: "Mahanarayan Tailam", NameHi: "महानारायण तैलम",
			Brand: "Kottakkal", Category: "ayurveda", Form: "tailam", PackSize: "200 ml",
			Description: "Classical 56-herb oil for joint and muscle therapy.",
			Indications: "Joint pain, sciatica, muscle stiffness (external use)",
			DosageInstructions: "Warm gently; apply locally with mild massage.",
			Prices:     money.RegionalPricing{"INR": 38000},
			GSTPercent: 5, HSNCode: "30049011",
			IsActive: true,
		},
		{
			Name: "Chyawanprash", NameHi: "च्यवनप्राश",
			Brand: "Dr.Ayya", IsInhouseBrand: true,
			Category: "ayurveda", Form: "lehyam", PackSize: "500 g",
			Description: "Amla-based rasayana for daily immunity.",
			Indications: "Immunity, daily nutrition, post-illness recovery",
			DosageInstructions: "1 tsp morning with warm milk.",
			Prices:     money.RegionalPricing{"INR": 35000},
			GSTPercent: 5, HSNCode: "30049011",
			IsActive: true,
		},

		// ── Wellness (English-first; OTC-grade) ───────────────────────────────
		{
			Name: "Tulsi Drops", Brand: "Dr.Ayya", IsInhouseBrand: true,
			Category: "wellness", Form: "drops", PackSize: "20 ml",
			Description: "Five-Tulsi concentrate; immunity support.",
			Indications: "Daily immunity, cough, throat irritation",
			DosageInstructions: "5 drops in a glass of water twice daily.",
			Prices:     money.RegionalPricing{"INR": 9900},
			GSTPercent: 18, HSNCode: "33049990",
			IsActive: true,
		},
		{
			Name: "Cold-Pressed Coconut Oil", Brand: "Dr.Ayya", IsInhouseBrand: true,
			Category: "wellness", Form: "oil", PackSize: "500 ml",
			Description: "Wood-pressed virgin coconut oil — cooking + body use.",
			Prices:     money.RegionalPricing{"INR": 45000},
			GSTPercent: 5, HSNCode: "15131100",
			IsActive: true,
		},
		{
			Name: "Moringa Capsules", Brand: "Organic India",
			Category: "wellness", Form: "capsule", PackSize: "60 capsules",
			Description: "Moringa oleifera leaf, 500 mg per capsule.",
			StrengthValue: "500", StrengthUnit: "mg",
			Indications: "Daily multivitamin, iron support, post-natal nutrition",
			DosageInstructions: "1–2 capsules daily with meals.",
			Prices:     money.RegionalPricing{"INR": 32000},
			GSTPercent: 18, HSNCode: "21069099",
			IsActive: true,
		},
		{
			Name: "Herbal Immunity Tea", Brand: "Dr.Ayya", IsInhouseBrand: true,
			Category: "wellness", Form: "tea", PackSize: "100 g",
			Description: "Tulsi + Ashwagandha + Ginger blend; caffeine-free.",
			DosageInstructions: "Brew 1 tsp in hot water for 5 minutes; sip twice daily.",
			Prices:     money.RegionalPricing{"INR": 24900},
			GSTPercent: 18, HSNCode: "21069099",
			IsActive: true,
		},
		{
			Name: "Spirulina Tablets", Brand: "Patanjali",
			Category: "wellness", Form: "tablet", PackSize: "60 tablets",
			Description: "Blue-green algae nutritional supplement.",
			StrengthValue: "500", StrengthUnit: "mg",
			DosageInstructions: "1 tablet twice daily after meals.",
			Prices:     money.RegionalPricing{"INR": 22000},
			GSTPercent: 18, HSNCode: "21069099",
			IsActive: true,
		},
		{
			Name: "Aloe Vera Gel", Brand: "Forest Essentials",
			Category: "wellness", Form: "gel", PackSize: "100 ml",
			Description: "99.5% pure aloe gel for skin and hair.",
			Indications: "Skin hydration, minor burns, scalp care",
			DosageInstructions: "Apply a thin layer to affected area; for external use only.",
			Prices:     money.RegionalPricing{"INR": 29500},
			GSTPercent: 18, HSNCode: "33049990",
			IsActive: true,
		},
	}

	for _, m := range catalog {
		// FirstOrCreate keyed on (name, brand) so re-running is a no-op for rows
		// already seeded — and so an admin who has *edited* a seeded SKU (price,
		// description, etc.) won't have their changes overwritten on restart.
		key := Medicine{Name: m.Name, Brand: m.Brand}
		var existing Medicine
		if err := db.Where(key).First(&existing).Error; err == nil {
			continue
		}
		// New row — seed it.
		_ = db.Create(&m).Error
	}
}

package identity

import "gorm.io/gorm"

// SeedDoctorTypes seeds the 4 AYUSH doctor types and 39 sub-specialties.
// Safe to call multiple times — uses FirstOrCreate.
func SeedDoctorTypes(db *gorm.DB) {
	types := []DoctorType{
		{Name: "Siddha", Slug: "siddha", Icon: "🔥", Color: "amber", Description: "Ancient Tamil healing science using herbs, minerals and pulse diagnosis. Originated in Tamil Nadu over 5000 years ago."},
		{Name: "Ayurveda", Slug: "ayurveda", Icon: "🌿", Color: "emerald", Description: "5000-year holistic Indian medicine system balancing body, mind and spirit through diet, herbs and lifestyle."},
		{Name: "Yoga & Naturopathy", Slug: "yoga_naturopathy", Icon: "🧘", Color: "sky", Description: "Natural healing through yoga therapy, hydrotherapy, diet counseling and movement for preventive wellness."},
		{Name: "Psychology", Slug: "psychology", Icon: "🧠", Color: "violet", Description: "Mind and emotional wellness through evidence-based talk therapy, CBT, stress counseling and mindfulness."},
	}
	for _, t := range types {
		db.Where(DoctorType{Slug: t.Slug}).FirstOrCreate(&t)
	}

	// Fetch IDs
	var siddha, ayurveda, yoga, psych DoctorType
	db.Where("slug = ?", "siddha").First(&siddha)
	db.Where("slug = ?", "ayurveda").First(&ayurveda)
	db.Where("slug = ?", "yoga_naturopathy").First(&yoga)
	db.Where("slug = ?", "psychology").First(&psych)

	subs := []DoctorSubSpecialty{
		// Siddha
		{DoctorTypeID: siddha.ID, Name: "Kuzhandhai Maruthuvam (Siddha Pediatrics)", Slug: "siddha_pediatrics", RequiresCertificate: true},
		{DoctorTypeID: siddha.ID, Name: "Varmam Therapy", Slug: "varmam", RequiresCertificate: true},
		{DoctorTypeID: siddha.ID, Name: "Skin & Cosmetology", Slug: "siddha_skin", RequiresCertificate: false},
		{DoctorTypeID: siddha.ID, Name: "Diabetes Management", Slug: "siddha_diabetes", RequiresCertificate: false},
		{DoctorTypeID: siddha.ID, Name: "Auto-immune Disorders", Slug: "siddha_autoimmune", RequiresCertificate: false},
		{DoctorTypeID: siddha.ID, Name: "Arthritis & Joint Care", Slug: "siddha_arthritis", RequiresCertificate: false},
		{DoctorTypeID: siddha.ID, Name: "Liver & Digestive Health", Slug: "siddha_digestive", RequiresCertificate: false},
		{DoctorTypeID: siddha.ID, Name: "Women's Health (Siddha Gynecology)", Slug: "siddha_gynecology", RequiresCertificate: true},
		{DoctorTypeID: siddha.ID, Name: "Neurological Disorders", Slug: "siddha_neuro", RequiresCertificate: true},
		{DoctorTypeID: siddha.ID, Name: "Cancer Supportive Care", Slug: "siddha_cancer", RequiresCertificate: true},
		// Ayurveda
		{DoctorTypeID: ayurveda.ID, Name: "Panchakarma Specialist", Slug: "panchakarma", RequiresCertificate: true},
		{DoctorTypeID: ayurveda.ID, Name: "Kaya Chikitsa (General Medicine)", Slug: "kaya_chikitsa", RequiresCertificate: false},
		{DoctorTypeID: ayurveda.ID, Name: "Heart & Cardiovascular", Slug: "ayurveda_cardio", RequiresCertificate: true},
		{DoctorTypeID: ayurveda.ID, Name: "Diabetes & Endocrine", Slug: "ayurveda_diabetes", RequiresCertificate: false},
		{DoctorTypeID: ayurveda.ID, Name: "Stri Roga (Gynecology)", Slug: "ayurveda_gynecology", RequiresCertificate: true},
		{DoctorTypeID: ayurveda.ID, Name: "Skin & Dermatology", Slug: "ayurveda_skin", RequiresCertificate: false},
		{DoctorTypeID: ayurveda.ID, Name: "Rasayana (Anti-aging & Rejuvenation)", Slug: "rasayana", RequiresCertificate: false},
		{DoctorTypeID: ayurveda.ID, Name: "Neuropathy", Slug: "ayurveda_neuro", RequiresCertificate: true},
		{DoctorTypeID: ayurveda.ID, Name: "Kidney & Urological Health", Slug: "ayurveda_kidney", RequiresCertificate: true},
		{DoctorTypeID: ayurveda.ID, Name: "Kaumarabhritya (Pediatrics)", Slug: "ayurveda_pediatrics", RequiresCertificate: true},
		// Yoga & Naturopathy
		{DoctorTypeID: yoga.ID, Name: "Therapeutic Yoga", Slug: "therapeutic_yoga", RequiresCertificate: true},
		{DoctorTypeID: yoga.ID, Name: "Pregnancy & Prenatal Yoga", Slug: "prenatal_yoga", RequiresCertificate: true},
		{DoctorTypeID: yoga.ID, Name: "Hydrotherapy", Slug: "hydrotherapy", RequiresCertificate: true},
		{DoctorTypeID: yoga.ID, Name: "Diet & Nutrition Counseling", Slug: "nutrition", RequiresCertificate: false},
		{DoctorTypeID: yoga.ID, Name: "Chronic Pain Management", Slug: "chronic_pain", RequiresCertificate: false},
		{DoctorTypeID: yoga.ID, Name: "Weight Management", Slug: "weight_management", RequiresCertificate: false},
		{DoctorTypeID: yoga.ID, Name: "Respiratory Health", Slug: "respiratory", RequiresCertificate: false},
		{DoctorTypeID: yoga.ID, Name: "Cardiovascular Wellness", Slug: "yoga_cardio", RequiresCertificate: false},
		{DoctorTypeID: yoga.ID, Name: "Detox & Cleansing Programs", Slug: "detox", RequiresCertificate: false},
		// Psychology
		{DoctorTypeID: psych.ID, Name: "Clinical Psychology", Slug: "clinical_psychology", RequiresCertificate: true},
		{DoctorTypeID: psych.ID, Name: "Child & Adolescent Therapy", Slug: "child_therapy", RequiresCertificate: true},
		{DoctorTypeID: psych.ID, Name: "Relationship & Couples Counseling", Slug: "relationship_counseling", RequiresCertificate: false},
		{DoctorTypeID: psych.ID, Name: "Trauma & PTSD", Slug: "trauma_ptsd", RequiresCertificate: true},
		{DoctorTypeID: psych.ID, Name: "OCD & Anxiety Disorders", Slug: "anxiety_ocd", RequiresCertificate: true},
		{DoctorTypeID: psych.ID, Name: "Sleep Disorders", Slug: "sleep_disorders", RequiresCertificate: false},
		{DoctorTypeID: psych.ID, Name: "Addiction Counseling", Slug: "addiction", RequiresCertificate: true},
		{DoctorTypeID: psych.ID, Name: "Grief Counseling", Slug: "grief", RequiresCertificate: false},
		{DoctorTypeID: psych.ID, Name: "Cognitive Behavioral Therapy (CBT)", Slug: "cbt", RequiresCertificate: true},
		{DoctorTypeID: psych.ID, Name: "Mindfulness-Based Therapy", Slug: "mindfulness", RequiresCertificate: false},
	}
	for _, s := range subs {
		db.Where(DoctorSubSpecialty{Slug: s.Slug}).FirstOrCreate(&s)
	}
}

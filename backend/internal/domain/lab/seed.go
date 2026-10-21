package lab

import (
	"context"
	"log"

	"gorm.io/gorm"
)

// SeedIfEmpty seeds the lab catalog, partners and centers only when the
// lab_tests table is empty. Safe to call on every startup.
func SeedIfEmpty(db *gorm.DB) {
	repo := NewRepository(db)
	ctx := context.Background()

	count, err := repo.CountTests(ctx)
	if err != nil || count > 0 {
		return
	}

	log.Println("[Lab] Seeding catalog, partners and centers...")

	// ── Partners ──────────────────────────────────────────────────────────────
	partners := []LabPartner{
		{Name: "Thyrocare Technologies", Slug: "thyrocare", IsNABL: true, Rating: 4.5,
			LogoURL:     "https://upload.wikimedia.org/wikipedia/commons/thumb/3/3b/Thyrocare_Logo.svg/320px-Thyrocare_Logo.svg.png",
			Description: "India's largest diagnostic chain with NABL & ISO accreditation"},
		{Name: "DRTC Diagnostics", Slug: "drtc", IsNABL: true, Rating: 4.3,
			LogoURL:     "",
			Description: "Tamil Nadu's trusted diagnostic partner with 100+ collection centers"},
		{Name: "SRL Diagnostics", Slug: "srl", IsNABL: true, Rating: 4.4,
			LogoURL:     "",
			Description: "One of India's largest diagnostic companies with CAP accreditation"},
		{Name: "Metropolis Healthcare", Slug: "metropolis", IsNABL: true, IsCAP: true, Rating: 4.6,
			LogoURL:     "",
			Description: "Premium diagnostics with ISO 15189 accreditation and 4000+ tests"},
		{Name: "Vijaya Diagnostics", Slug: "vijaya", IsNABL: true, Rating: 4.4,
			LogoURL:     "",
			Description: "South India's leading diagnostic chain with 24-hour reporting"},
	}
	if err := repo.BulkCreatePartners(ctx, partners); err != nil {
		log.Printf("[Lab] Partner seed error: %v", err)
		return
	}

	// Re-fetch to get IDs
	var dbPartners []LabPartner
	db.Where("slug IN ?", []string{"thyrocare", "drtc", "srl", "metropolis", "vijaya"}).Find(&dbPartners)
	pID := map[string]uint{}
	for _, p := range dbPartners {
		pID[p.Slug] = p.ID
	}

	// ── Tests (individual) ────────────────────────────────────────────────────
	tests := []LabTest{
		// Hematology
		{Name: "Complete Blood Count (CBC)", Category: "Hematology", HealthConcern: "General", SampleType: "blood", FastingRequired: false, ReportHours: 12, Unit: "cells/μL", BasePrice: 24900, OriginalPricePaise: 39900, DiscountPct: 38, IsPopular: true, Icon: "🩸"},
		{Name: "ESR (Erythrocyte Sedimentation Rate)", Category: "Hematology", HealthConcern: "Infections", SampleType: "blood", FastingRequired: false, ReportHours: 6, Unit: "mm/hr", BasePrice: 9900, OriginalPricePaise: 15900, DiscountPct: 38, Icon: "🩸"},
		// Biochemistry
		{Name: "Lipid Profile", Category: "Biochemistry", HealthConcern: "Heart", SampleType: "blood", FastingRequired: true, ReportHours: 12, Unit: "mg/dL", BasePrice: 34900, OriginalPricePaise: 59900, DiscountPct: 42, IsPopular: true, Icon: "❤️"},
		{Name: "Liver Function Test (LFT)", Category: "Biochemistry", HealthConcern: "Liver", SampleType: "blood", FastingRequired: true, ReportHours: 24, Unit: "U/L", BasePrice: 44900, OriginalPricePaise: 79900, DiscountPct: 44, IsPopular: true, Icon: "🫀"},
		{Name: "Kidney Function Test (KFT)", Category: "Biochemistry", HealthConcern: "Kidney", SampleType: "blood", FastingRequired: false, ReportHours: 24, Unit: "mg/dL", BasePrice: 44900, OriginalPricePaise: 79900, DiscountPct: 44, IsPopular: true, Icon: "🫘"},
		{Name: "Blood Sugar Fasting (FBS)", Category: "Biochemistry", HealthConcern: "Diabetes", SampleType: "blood", FastingRequired: true, ReportHours: 6, Unit: "mg/dL", BasePrice: 9900, OriginalPricePaise: 14900, DiscountPct: 34, IsPopular: true, Icon: "🩸"},
		{Name: "Blood Sugar Post Prandial (PPBS)", Category: "Biochemistry", HealthConcern: "Diabetes", SampleType: "blood", FastingRequired: false, ReportHours: 6, Unit: "mg/dL", BasePrice: 9900, OriginalPricePaise: 14900, DiscountPct: 34, Icon: "🩸"},
		{Name: "HbA1c (Glycated Hemoglobin)", Category: "Biochemistry", HealthConcern: "Diabetes", SampleType: "blood", FastingRequired: false, ReportHours: 12, Unit: "%", BasePrice: 34900, OriginalPricePaise: 59900, DiscountPct: 42, IsPopular: true, Icon: "🩸"},
		{Name: "Uric Acid", Category: "Biochemistry", HealthConcern: "Kidney", SampleType: "blood", FastingRequired: false, ReportHours: 12, Unit: "mg/dL", BasePrice: 14900, OriginalPricePaise: 24900, DiscountPct: 40, Icon: "🫘"},
		{Name: "C-Reactive Protein (CRP)", Category: "Biochemistry", HealthConcern: "Infections", SampleType: "blood", FastingRequired: false, ReportHours: 12, Unit: "mg/L", BasePrice: 29900, OriginalPricePaise: 49900, DiscountPct: 40, Icon: "🦠"},
		// Endocrinology
		{Name: "TSH (Thyroid Stimulating Hormone)", Category: "Endocrinology", HealthConcern: "Thyroid", SampleType: "blood", FastingRequired: false, ReportHours: 24, Unit: "mIU/L", BasePrice: 29900, OriginalPricePaise: 49900, DiscountPct: 40, IsPopular: true, Icon: "🦋"},
		{Name: "T3 (Triiodothyronine)", Category: "Endocrinology", HealthConcern: "Thyroid", SampleType: "blood", FastingRequired: false, ReportHours: 24, Unit: "ng/dL", BasePrice: 24900, OriginalPricePaise: 39900, DiscountPct: 38, Icon: "🦋"},
		{Name: "T4 (Thyroxine)", Category: "Endocrinology", HealthConcern: "Thyroid", SampleType: "blood", FastingRequired: false, ReportHours: 24, Unit: "μg/dL", BasePrice: 24900, OriginalPricePaise: 39900, DiscountPct: 38, Icon: "🦋"},
		// Vitamins & Minerals
		{Name: "Vitamin D (25-OH)", Category: "Vitamins", HealthConcern: "Vitamins", SampleType: "blood", FastingRequired: false, ReportHours: 24, Unit: "ng/mL", BasePrice: 59900, OriginalPricePaise: 99900, DiscountPct: 40, IsPopular: true, Icon: "☀️"},
		{Name: "Vitamin B12 (Cobalamin)", Category: "Vitamins", HealthConcern: "Vitamins", SampleType: "blood", FastingRequired: false, ReportHours: 24, Unit: "pg/mL", BasePrice: 49900, OriginalPricePaise: 79900, DiscountPct: 38, IsPopular: true, Icon: "☀️"},
		{Name: "Iron Studies (Serum Iron + TIBC)", Category: "Vitamins", HealthConcern: "Vitamins", SampleType: "blood", FastingRequired: true, ReportHours: 24, Unit: "μg/dL", BasePrice: 49900, OriginalPricePaise: 84900, DiscountPct: 41, Icon: "☀️"},
		{Name: "Ferritin", Category: "Vitamins", HealthConcern: "Vitamins", SampleType: "blood", FastingRequired: false, ReportHours: 24, Unit: "ng/mL", BasePrice: 44900, OriginalPricePaise: 74900, DiscountPct: 40, Icon: "☀️"},
		// Hormones
		{Name: "Testosterone (Total)", Category: "Hormones", HealthConcern: "Hormones", SampleType: "blood", FastingRequired: false, ReportHours: 24, Unit: "ng/dL", BasePrice: 54900, OriginalPricePaise: 89900, DiscountPct: 39, Icon: "🔬"},
		{Name: "FSH (Follicle Stimulating Hormone)", Category: "Hormones", HealthConcern: "Hormones", SampleType: "blood", FastingRequired: false, ReportHours: 24, Unit: "mIU/mL", BasePrice: 49900, OriginalPricePaise: 84900, DiscountPct: 41, Icon: "🔬"},
		{Name: "AMH (Anti-Müllerian Hormone)", Category: "Hormones", HealthConcern: "Hormones", SampleType: "blood", FastingRequired: false, ReportHours: 48, Unit: "ng/mL", BasePrice: 149900, OriginalPricePaise: 249900, DiscountPct: 40, Icon: "🔬"},
		// Urinalysis
		{Name: "Urine Routine Examination", Category: "Urinalysis", HealthConcern: "Kidney", SampleType: "urine", FastingRequired: false, ReportHours: 6, Unit: "", BasePrice: 9900, OriginalPricePaise: 14900, DiscountPct: 34, IsPopular: true, Icon: "🔬"},
		{Name: "Stool Routine Examination", Category: "Microbiology", HealthConcern: "Infections", SampleType: "stool", FastingRequired: false, ReportHours: 12, Unit: "", BasePrice: 9900, OriginalPricePaise: 14900, DiscountPct: 34, Icon: "🔬"},
		// Cancer markers
		{Name: "PSA (Prostate Specific Antigen)", Category: "Oncology", HealthConcern: "Cancer Markers", SampleType: "blood", FastingRequired: false, ReportHours: 24, Unit: "ng/mL", BasePrice: 59900, OriginalPricePaise: 99900, DiscountPct: 40, Icon: "🔬"},
		{Name: "CEA (Carcinoembryonic Antigen)", Category: "Oncology", HealthConcern: "Cancer Markers", SampleType: "blood", FastingRequired: false, ReportHours: 24, Unit: "ng/mL", BasePrice: 79900, OriginalPricePaise: 129900, DiscountPct: 38, Icon: "🔬"},
		// Infections
		{Name: "HIV 1 & 2 Screening", Category: "Serology", HealthConcern: "Infections", SampleType: "blood", FastingRequired: false, ReportHours: 24, Unit: "", BasePrice: 29900, OriginalPricePaise: 49900, DiscountPct: 40, Icon: "🦠"},
		{Name: "HbsAg (Hepatitis B Surface Antigen)", Category: "Serology", HealthConcern: "Infections", SampleType: "blood", FastingRequired: false, ReportHours: 12, Unit: "", BasePrice: 24900, OriginalPricePaise: 39900, DiscountPct: 38, Icon: "🦠"},
		{Name: "Dengue NS1 Antigen", Category: "Serology", HealthConcern: "Infections", SampleType: "blood", FastingRequired: false, ReportHours: 12, Unit: "", BasePrice: 54900, OriginalPricePaise: 89900, DiscountPct: 39, Icon: "🦠"},
		// Cardiac
		{Name: "hsCRP (High Sensitivity CRP)", Category: "Cardiology", HealthConcern: "Heart", SampleType: "blood", FastingRequired: false, ReportHours: 24, Unit: "mg/L", BasePrice: 54900, OriginalPricePaise: 89900, DiscountPct: 39, Icon: "❤️"},
		{Name: "Troponin I (High Sensitivity)", Category: "Cardiology", HealthConcern: "Heart", SampleType: "blood", FastingRequired: false, ReportHours: 12, Unit: "ng/mL", BasePrice: 79900, OriginalPricePaise: 129900, DiscountPct: 38, Icon: "❤️"},
		// Allergy
		{Name: "Total IgE (Allergy Screen)", Category: "Allergy", HealthConcern: "Allergy", SampleType: "blood", FastingRequired: false, ReportHours: 24, Unit: "IU/mL", BasePrice: 69900, OriginalPricePaise: 119900, DiscountPct: 42, Icon: "🤧"},
	}
	if err := repo.BulkCreateTests(ctx, tests); err != nil {
		log.Printf("[Lab] Test seed error: %v", err)
		return
	}

	// Fetch inserted test IDs by name
	var dbTests []LabTest
	db.Where("is_package = ?", false).Find(&dbTests)
	tID := map[string]uint{}
	for _, t := range dbTests {
		tID[t.Name] = t.ID
	}

	// ── Health packages ───────────────────────────────────────────────────────
	packages := []LabTest{
		{Name: "Full Body Checkup", Category: "Packages", HealthConcern: "General", SampleType: "blood", FastingRequired: true, ReportHours: 24, BasePrice: 199900, OriginalPricePaise: 399900, DiscountPct: 50, IsPackage: true, IsPopular: true, Icon: "🔬", Description: "80 tests covering all major organ systems"},
		{Name: "Diabetes Panel", Category: "Packages", HealthConcern: "Diabetes", SampleType: "blood", FastingRequired: true, ReportHours: 12, BasePrice: 59900, OriginalPricePaise: 99900, DiscountPct: 40, IsPackage: true, IsPopular: true, Icon: "🩸", Description: "15 tests for comprehensive diabetes assessment"},
		{Name: "Thyroid Profile (T3/T4/TSH)", Category: "Packages", HealthConcern: "Thyroid", SampleType: "blood", FastingRequired: false, ReportHours: 24, BasePrice: 39900, OriginalPricePaise: 59900, DiscountPct: 33, IsPackage: true, IsPopular: true, Icon: "🦋", Description: "Complete thyroid function assessment"},
		{Name: "Heart Health Package", Category: "Packages", HealthConcern: "Heart", SampleType: "blood", FastingRequired: true, ReportHours: 24, BasePrice: 149900, OriginalPricePaise: 249900, DiscountPct: 40, IsPackage: true, IsPopular: true, Icon: "❤️", Description: "25 tests for cardiac risk assessment"},
		{Name: "Women's Wellness Panel", Category: "Packages", HealthConcern: "Hormones", SampleType: "blood", FastingRequired: true, ReportHours: 48, BasePrice: 129900, OriginalPricePaise: 219900, DiscountPct: 41, IsPackage: true, IsPopular: false, Icon: "👩", Description: "40 tests for women's comprehensive health"},
		{Name: "Men's Health Panel", Category: "Packages", HealthConcern: "Hormones", SampleType: "blood", FastingRequired: true, ReportHours: 24, BasePrice: 149900, OriginalPricePaise: 249900, DiscountPct: 40, IsPackage: true, IsPopular: false, Icon: "👨", Description: "30 tests for men's comprehensive health"},
		{Name: "Senior Citizen Health Package", Category: "Packages", HealthConcern: "General", SampleType: "blood", FastingRequired: true, ReportHours: 48, BasePrice: 249900, OriginalPricePaise: 499900, DiscountPct: 50, IsPackage: true, IsPopular: false, Icon: "👴", Description: "90 tests for comprehensive senior health monitoring"},
		{Name: "Vitamin Deficiency Panel", Category: "Packages", HealthConcern: "Vitamins", SampleType: "blood", FastingRequired: false, ReportHours: 24, BasePrice: 99900, OriginalPricePaise: 199900, DiscountPct: 50, IsPackage: true, IsPopular: false, Icon: "☀️", Description: "6 tests for key vitamin and mineral levels"},
		{Name: "Kidney Function Panel", Category: "Packages", HealthConcern: "Kidney", SampleType: "blood", FastingRequired: false, ReportHours: 24, BasePrice: 44900, OriginalPricePaise: 79900, DiscountPct: 44, IsPackage: true, IsPopular: false, Icon: "🫘", Description: "10 tests for comprehensive kidney health"},
		{Name: "Pre-Employment Health Screen", Category: "Packages", HealthConcern: "General", SampleType: "multiple", FastingRequired: true, ReportHours: 24, BasePrice: 79900, OriginalPricePaise: 149900, DiscountPct: 47, IsPackage: true, IsPopular: false, Icon: "💼", Description: "20 tests for pre-employment medical screening"},
	}
	if err := repo.BulkCreateTests(ctx, packages); err != nil {
		log.Printf("[Lab] Package seed error: %v", err)
		return
	}

	// Fetch package IDs
	var dbPkgs []LabTest
	db.Where("is_package = ?", true).Find(&dbPkgs)
	pkgID := map[string]uint{}
	for _, p := range dbPkgs {
		pkgID[p.Name] = p.ID
	}

	// ── Package → component mappings ─────────────────────────────────────────
	var pkgItems []LabPackageItem
	addItems := func(pkgName string, testNames []string) {
		pid, ok := pkgID[pkgName]
		if !ok {
			return
		}
		for _, tname := range testNames {
			if cid, ok := tID[tname]; ok {
				pkgItems = append(pkgItems, LabPackageItem{PackageTestID: pid, ComponentTestID: cid})
			}
		}
	}
	addItems("Full Body Checkup", []string{
		"Complete Blood Count (CBC)", "Lipid Profile", "Liver Function Test (LFT)",
		"Kidney Function Test (KFT)", "TSH (Thyroid Stimulating Hormone)", "T3 (Triiodothyronine)",
		"T4 (Thyroxine)", "HbA1c (Glycated Hemoglobin)", "Blood Sugar Fasting (FBS)",
		"Vitamin D (25-OH)", "Vitamin B12 (Cobalamin)", "Iron Studies (Serum Iron + TIBC)",
		"Uric Acid", "C-Reactive Protein (CRP)", "Urine Routine Examination",
	})
	addItems("Diabetes Panel", []string{
		"Blood Sugar Fasting (FBS)", "Blood Sugar Post Prandial (PPBS)",
		"HbA1c (Glycated Hemoglobin)", "Kidney Function Test (KFT)",
		"Lipid Profile", "Urine Routine Examination",
	})
	addItems("Thyroid Profile (T3/T4/TSH)", []string{
		"TSH (Thyroid Stimulating Hormone)", "T3 (Triiodothyronine)", "T4 (Thyroxine)",
	})
	addItems("Heart Health Package", []string{
		"Lipid Profile", "hsCRP (High Sensitivity CRP)", "Complete Blood Count (CBC)",
		"Blood Sugar Fasting (FBS)", "Troponin I (High Sensitivity)",
	})
	addItems("Women's Wellness Panel", []string{
		"Complete Blood Count (CBC)", "TSH (Thyroid Stimulating Hormone)",
		"FSH (Follicle Stimulating Hormone)", "AMH (Anti-Müllerian Hormone)",
		"Vitamin D (25-OH)", "Iron Studies (Serum Iron + TIBC)", "Ferritin",
	})
	addItems("Men's Health Panel", []string{
		"Complete Blood Count (CBC)", "Testosterone (Total)", "PSA (Prostate Specific Antigen)",
		"Lipid Profile", "Liver Function Test (LFT)", "Kidney Function Test (KFT)",
	})
	addItems("Vitamin Deficiency Panel", []string{
		"Vitamin D (25-OH)", "Vitamin B12 (Cobalamin)", "Iron Studies (Serum Iron + TIBC)",
		"Ferritin",
	})
	addItems("Kidney Function Panel", []string{
		"Kidney Function Test (KFT)", "Uric Acid", "Urine Routine Examination",
	})
	addItems("Pre-Employment Health Screen", []string{
		"Complete Blood Count (CBC)", "Blood Sugar Fasting (FBS)", "Liver Function Test (LFT)",
		"HIV 1 & 2 Screening", "HbsAg (Hepatitis B Surface Antigen)", "Urine Routine Examination",
	})

	if len(pkgItems) > 0 {
		if err := repo.BulkCreatePackageItems(ctx, pkgItems); err != nil {
			log.Printf("[Lab] Package items seed error: %v", err)
		}
	}

	// ── Lab Centers ───────────────────────────────────────────────────────────
	centers := []LabCenter{
		// Chennai
		{PartnerID: pID["thyrocare"], Name: "Thyrocare Anna Nagar", Address: "47 7th Ave, Anna Nagar, Chennai", Pincode: "600040", District: "Chennai", Taluk: "Anna Nagar", Phone: "044-48612345", OffersHomePickup: true, OffersLabVisit: true},
		{PartnerID: pID["drtc"], Name: "DRTC Tambaram", Address: "120 GST Road, Tambaram, Chennai", Pincode: "600045", District: "Chennai", Taluk: "Tambaram", Phone: "044-22238910", OffersHomePickup: true, OffersLabVisit: true},
		{PartnerID: pID["srl"], Name: "SRL Adyar", Address: "82 LB Road, Adyar, Chennai", Pincode: "600020", District: "Chennai", Taluk: "Adyar", Phone: "044-24412345", OffersHomePickup: true, OffersLabVisit: true},
		{PartnerID: pID["metropolis"], Name: "Metropolis T.Nagar", Address: "14 Venkatanarayana Rd, T.Nagar, Chennai", Pincode: "600017", District: "Chennai", Taluk: "T.Nagar", Phone: "044-43008900", OffersHomePickup: true, OffersLabVisit: true},
		{PartnerID: pID["vijaya"], Name: "Vijaya Diagnostics Velachery", Address: "200 Velachery Main Rd, Velachery, Chennai", Pincode: "600042", District: "Chennai", Taluk: "Velachery", Phone: "044-22432345", OffersHomePickup: true, OffersLabVisit: true},
		{PartnerID: pID["thyrocare"], Name: "Thyrocare Chromepet", Address: "5 GST Road, Chromepet, Chennai", Pincode: "600044", District: "Chennai", Taluk: "Chromepet", Phone: "044-22381234", OffersHomePickup: true, OffersLabVisit: true},
		// Coimbatore
		{PartnerID: pID["thyrocare"], Name: "Thyrocare RS Puram", Address: "21 Bharathi Park Rd, RS Puram, Coimbatore", Pincode: "641002", District: "Coimbatore", Taluk: "Coimbatore South", Phone: "0422-2543210", OffersHomePickup: true, OffersLabVisit: true},
		{PartnerID: pID["drtc"], Name: "DRTC Gandhipuram", Address: "45 Cross Cut Rd, Gandhipuram, Coimbatore", Pincode: "641012", District: "Coimbatore", Taluk: "Coimbatore North", Phone: "0422-2238910", OffersHomePickup: true, OffersLabVisit: true},
		{PartnerID: pID["srl"], Name: "SRL Peelamedu", Address: "100 Avinashi Rd, Peelamedu, Coimbatore", Pincode: "641004", District: "Coimbatore", Taluk: "Coimbatore North", Phone: "0422-2654321", OffersHomePickup: false, OffersLabVisit: true},
		// Madurai
		{PartnerID: pID["thyrocare"], Name: "Thyrocare KK Nagar Madurai", Address: "35 Koodal Nagar, KK Nagar, Madurai", Pincode: "625020", District: "Madurai", Taluk: "Madurai North", Phone: "0452-2381234", OffersHomePickup: true, OffersLabVisit: true},
		{PartnerID: pID["vijaya"], Name: "Vijaya Diagnostics Madurai", Address: "12 Anna Nagar 2nd St, Madurai", Pincode: "625020", District: "Madurai", Taluk: "Madurai North", Phone: "0452-4321098", OffersHomePickup: true, OffersLabVisit: true},
		// Trichy
		{PartnerID: pID["thyrocare"], Name: "Thyrocare Srirangam", Address: "8 Thillai Nagar Main Rd, Trichy", Pincode: "620006", District: "Tiruchirappalli", Taluk: "Trichy", Phone: "0431-2381234", OffersHomePickup: true, OffersLabVisit: true},
		{PartnerID: pID["drtc"], Name: "DRTC Thillainagar", Address: "22 Thillainagar, Trichy", Pincode: "620018", District: "Tiruchirappalli", Taluk: "Trichy", Phone: "0431-2234567", OffersHomePickup: false, OffersLabVisit: true},
		// Salem
		{PartnerID: pID["metropolis"], Name: "Metropolis Salem", Address: "14 Omalur Main Rd, Salem", Pincode: "636004", District: "Salem", Taluk: "Salem", Phone: "0427-2381234", OffersHomePickup: true, OffersLabVisit: true},
		// Erode
		{PartnerID: pID["thyrocare"], Name: "Thyrocare Erode", Address: "7 Brough Rd, Erode", Pincode: "638001", District: "Erode", Taluk: "Erode", Phone: "0424-2381234", OffersHomePickup: true, OffersLabVisit: true},
	}

	if err := repo.BulkCreateCenters(ctx, centers); err != nil {
		log.Printf("[Lab] Center seed error: %v", err)
		return
	}

	// Re-fetch centers with IDs
	var dbCenters []LabCenter
	db.Find(&dbCenters)

	// Re-fetch all tests (including packages) for pricing
	var allTests []LabTest
	db.Find(&allTests)

	// ── Pricing: every center × every test ───────────────────────────────────
	// Thyrocare: cheapest; Metropolis: premium; others: mid-range
	priceMult := map[string]float64{
		"thyrocare":  1.0,
		"drtc":       1.15,
		"srl":        1.2,
		"metropolis": 1.35,
		"vijaya":     1.2,
	}
	homePickupBySlug := map[string]int64{
		"thyrocare":  10000, // ₹100
		"drtc":       15000, // ₹150
		"srl":        10000,
		"metropolis": 0, // free home pickup
		"vijaya":     12000,
	}

	// Build slug lookup for centers
	centerSlug := map[uint]string{}
	for _, c := range dbCenters {
		for _, p := range dbPartners {
			if p.ID == c.PartnerID {
				centerSlug[c.ID] = p.Slug
				break
			}
		}
	}

	var pricing []LabCenterPricing
	for _, c := range dbCenters {
		slug := centerSlug[c.ID]
		mult := priceMult[slug]
		if mult == 0 {
			mult = 1.0
		}
		pickup := homePickupBySlug[slug]
		for _, t := range allTests {
			price := int64(float64(t.BasePrice) * mult)
			hrs := t.ReportHours
			if !c.OffersHomePickup {
				pickup = 0
			}
			pricing = append(pricing, LabCenterPricing{
				LabCenterID:           c.ID,
				LabTestID:             t.ID,
				PricePaise:            price,
				HomePickupChargePaise: pickup,
				TurnaroundHours:       hrs,
				IsAvailable:           true,
			})
		}
	}

	if err := repo.BulkCreatePricing(ctx, pricing); err != nil {
		log.Printf("[Lab] Pricing seed error: %v", err)
		return
	}

	log.Printf("[Lab] Seed complete — %d tests, %d packages, %d centers, %d pricing rows",
		len(tests), len(packages), len(dbCenters), len(pricing))
}

package main

import (
	"log"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/money"

	"gorm.io/gorm"
)

func seedPharmacy(db *gorm.DB) {
	// Pharmacies
	pharmacies := []pharmacy.Pharmacy{
		{Name: "Dr.Ayya Central Warehouse", Address: "Industrial Area, Mumbai", City: "Mumbai", Latitude: 19.0760, Longitude: 72.8777, IsCentralWarehouse: true, HubTier: "global", ServiceRadius: 500, IsActive: true},
		{Name: "Dr.Ayya Mumbai Pharmacy", Address: "Andheri West, Mumbai", City: "Mumbai", Latitude: 19.1364, Longitude: 72.8296, IsCentralWarehouse: false, HubTier: "local", ServiceRadius: 15, IsActive: true},
		{Name: "Dr.Ayya Chennai Pharmacy", Address: "T Nagar, Chennai", City: "Chennai", Latitude: 13.0418, Longitude: 80.2341, IsCentralWarehouse: false, HubTier: "local", ServiceRadius: 15, IsActive: true},
	}
	for i := range pharmacies {
		db.FirstOrCreate(&pharmacies[i], "name = ?", pharmacies[i].Name)
	}
	// Link local pharmacies to central warehouse
	db.Model(&pharmacies[1]).Update("parent_hub_id", pharmacies[0].ID)
	db.Model(&pharmacies[2]).Update("parent_hub_id", pharmacies[0].ID)

	// Medicines
	meds := []pharmacy.Medicine{
		{Name: "Dolo 650mg", Brand: "Micro Labs", Category: "Antipyretic", Description: "Paracetamol for fever and pain", Manufacturer: "Micro Labs", Prices: money.RegionalPricing{"INR": 3000}, IsPrescriptionRequired: false},
		{Name: "Azithromycin 500mg", Brand: "Cipla", Category: "Antibiotic", Description: "Broad-spectrum antibiotic", Manufacturer: "Cipla", Prices: money.RegionalPricing{"INR": 15000}, IsPrescriptionRequired: true},
		{Name: "Metformin 500mg", Brand: "USV", Category: "Antidiabetic", Description: "For type 2 diabetes", Manufacturer: "USV Ltd", Prices: money.RegionalPricing{"INR": 5000}, IsPrescriptionRequired: true},
		{Name: "Amlodipine 5mg", Brand: "Pfizer", Category: "Antihypertensive", Description: "Blood pressure medication", Manufacturer: "Pfizer", Prices: money.RegionalPricing{"INR": 8000}, IsPrescriptionRequired: true},
		{Name: "Cetirizine 10mg", Brand: "Dr Reddy's", Category: "Antihistamine", Description: "For allergies", Manufacturer: "Dr Reddy's", Prices: money.RegionalPricing{"INR": 2500}, IsPrescriptionRequired: false},
		{Name: "Omeprazole 20mg", Brand: "AstraZeneca", Category: "Antacid", Description: "For acid reflux and GERD", Manufacturer: "AstraZeneca", Prices: money.RegionalPricing{"INR": 6000}, IsPrescriptionRequired: true},
		{Name: "Vitamin D3 60K", Brand: "Cholecalciferol", Category: "Supplement", Description: "Weekly vitamin D supplement", Manufacturer: "Cadila", Prices: money.RegionalPricing{"INR": 12000}, IsPrescriptionRequired: false},
		{Name: "Cough Syrup Benadryl", Brand: "J&J", Category: "Cough", Description: "For dry and wet cough", Manufacturer: "J&J", Prices: money.RegionalPricing{"INR": 15000}, IsPrescriptionRequired: false},
		{Name: "Insulin Glargine", Brand: "Sanofi", Category: "Insulin", Description: "Long-acting insulin", Manufacturer: "Sanofi", Prices: money.RegionalPricing{"INR": 45000}, IsPrescriptionRequired: true, RequiresColdChain: true},
		{Name: "Amoxicillin 500mg", Brand: "GSK", Category: "Antibiotic", Description: "Penicillin-type antibiotic", Manufacturer: "GSK", Prices: money.RegionalPricing{"INR": 7000}, IsPrescriptionRequired: true},
	}
	for i := range meds {
		db.FirstOrCreate(&meds[i], "name = ?", meds[i].Name)
	}

	// Inventory for each pharmacy
	for _, ph := range pharmacies {
		for _, med := range meds {
			var existing pharmacy.PharmacyInventory
			if db.Where("pharmacy_id = ? AND medicine_id = ?", ph.ID, med.ID).First(&existing).Error == nil {
				continue
			}
			stock := 500
			if !ph.IsCentralWarehouse {
				stock = 100
			}
			db.Create(&pharmacy.PharmacyInventory{
				PharmacyID:    ph.ID,
				MedicineID:    med.ID,
				StockQuantity: stock,
				ReorderLevel:  20,
				IsActive:      true,
			})
		}
	}

	// Supplier
	supplier := pharmacy.Supplier{
		Name:         "PharmaCorp India Ltd",
		ContactEmail: "orders@pharmacorp.in",
		Phone:        "022-55555555",
		Address:      "MIDC Industrial Area, Pune 411018",
		IsActive:     true,
	}
	db.FirstOrCreate(&supplier, "name = ?", supplier.Name)

	// Delivery zones
	zones := []pharmacy.DeliveryZone{
		{City: "Mumbai", IsQuickEnabled: true, MaxRadiusKm: 15},
		{City: "Chennai", IsQuickEnabled: true, MaxRadiusKm: 12},
		{City: "Delhi", IsQuickEnabled: false, MaxRadiusKm: 20},
	}
	for i := range zones {
		db.FirstOrCreate(&zones[i], "city = ?", zones[i].City)
	}

	log.Printf("       Created %d pharmacies, %d medicines, %d zones", len(pharmacies), len(meds), len(zones))
}


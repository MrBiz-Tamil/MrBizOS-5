package main

import (
	"log"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/money"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")
	database.Connect()
	db := database.DB

	// Seed Medicines if they don't exist
	meds := []pharmacy.Medicine{
		{ID: 1, Name: "Dolo 650mg", Brand: "Micro Labs", Category: "Fever", Prices: money.RegionalPricing{"INR": 30}, Manufacturer: "Micro Labs"},
		{ID: 2, Name: "Benadryl", Brand: "J&J", Category: "Cough", Prices: money.RegionalPricing{"INR": 150}, Manufacturer: "J&J"},
	}

	for _, m := range meds {
		db.FirstOrCreate(&m, m.ID)
	}

	// Seed Inventory with Specific IDs from Frontend Mock
	invs := []pharmacy.PharmacyInventory{
		{ID: 101, MedicineID: 1, StockQuantity: 100, IsActive: true, PharmacyID: 1},
		{ID: 102, MedicineID: 2, StockQuantity: 50, IsActive: true, PharmacyID: 1},
	}

	for _, inv := range invs {
		db.FirstOrCreate(&inv, inv.ID)
	}

	log.Println("    Database seeded with Frontend-compatible IDs")
}


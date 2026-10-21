package main

import (
	"log"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/store"

	"gorm.io/gorm"
)

func seedStore(db *gorm.DB) {
	// Cities and Serviceable Areas
	cities := []store.City{
		{Name: "Mumbai", State: "Maharashtra", IsActive: true},
		{Name: "Chennai", State: "Tamil Nadu", IsActive: true},
	}
	for i := range cities {
		db.FirstOrCreate(&cities[i], "name = ?", cities[i].Name)
	}

	areas := []store.ServiceableArea{
		{Pincode: "400001", CityID: cities[0].ID, PharmacyID: 2},
		{Pincode: "600017", CityID: cities[1].ID, PharmacyID: 3},
	}
	for i := range areas {
		db.FirstOrCreate(&areas[i], "pincode = ?", areas[i].Pincode)
	}

	products := []store.WellnessProduct{
		{Name: "Ashwagandha Capsules", Brand: "Himalaya", Category: "Supplements", Description: "Stress relief and immunity booster", SellingPrice: 29900, IsActive: true},
		{Name: "Organic Honey", Brand: "Dabur", Category: "Natural Foods", Description: "Pure organic honey", SellingPrice: 39900, IsActive: true},
		{Name: "Triphala Tablets", Brand: "Patanjali", Category: "Digestive", Description: "Herbal digestive aid", SellingPrice: 17500, IsActive: true},
		{Name: "Protein Powder (Whey)", Brand: "MuscleBlaze", Category: "Fitness", Description: "Whey protein for muscle recovery", SellingPrice: 199900, IsActive: true},
		{Name: "Aloe Vera Juice", Brand: "Kapiva", Category: "Detox", Description: "Cold-pressed aloe vera juice", SellingPrice: 24900, IsActive: true},
		{Name: "Yoga Mat Premium", Brand: "Boldfit", Category: "Fitness", Description: "Anti-skid yoga mat 6mm", SellingPrice: 149900, IsActive: true},
		{Name: "Multivitamin Daily", Brand: "HealthKart", Category: "Supplements", Description: "Complete daily multivitamin", SellingPrice: 59900, IsActive: true},
		{Name: "Tulsi Green Tea", Brand: "Organic India", Category: "Beverages", Description: "Organic tulsi green tea bags", SellingPrice: 18900, IsActive: true},
		{Name: "Flaxseed Oil Capsules", Brand: "WOW", Category: "Supplements", Description: "Omega-3 rich flaxseed oil", SellingPrice: 44900, IsActive: true},
		{Name: "Chyawanprash", Brand: "Dabur", Category: "Immunity", Description: "Traditional immunity booster", SellingPrice: 36900, IsActive: true},
	}

	stocks := []int{200, 150, 300, 80, 120, 50, 180, 250, 100, 175}

	for i := range products {
		db.FirstOrCreate(&products[i], "name = ?", products[i].Name)

		// Create Inventory at Hub 1
		inv := store.WellnessInventory{
			ProductID:     products[i].ID,
			PharmacyID:    1,
			StockQuantity: stocks[i],
			ReorderLevel:  10,
		}
		db.FirstOrCreate(&inv, "product_id = ? AND pharmacy_id = ?", inv.ProductID, inv.PharmacyID)
	}

	// Orders
	var patients []identity.User
	db.Where("role = ?", "patient").Limit(3).Find(&patients)
	if len(patients) < 3 {
		log.Println("          Not enough patients for store seeding, skipping orders")
		return
	}

	orders := []store.WellnessOrder{
		{OrderNumber: "WO-SEED-001", UserID: patients[0].ID, TotalAmount: 49800, Status: "delivered", DeliveryAddress: "123 MG Road, Mumbai"},
		{OrderNumber: "WO-SEED-002", UserID: patients[1].ID, TotalAmount: 199900, Status: "shipped", DeliveryAddress: "456 Park Street, Chennai"},
		{OrderNumber: "WO-SEED-003", UserID: patients[2].ID, TotalAmount: 77800, Status: "pending", DeliveryAddress: "789 Gandhi Nagar, Delhi"},
	}
	for i := range orders {
		db.FirstOrCreate(&orders[i], "order_number = ?", orders[i].OrderNumber)
	}

	// Order items
	items := []store.WellnessOrderItem{
		{OrderID: orders[0].ID, ProductID: products[0].ID, Quantity: 1, UnitPrice: products[0].SellingPrice},
		{OrderID: orders[0].ID, ProductID: products[2].ID, Quantity: 1, UnitPrice: products[2].SellingPrice},
		{OrderID: orders[1].ID, ProductID: products[3].ID, Quantity: 1, UnitPrice: products[3].SellingPrice},
		{OrderID: orders[2].ID, ProductID: products[1].ID, Quantity: 1, UnitPrice: products[1].SellingPrice},
		{OrderID: orders[2].ID, ProductID: products[7].ID, Quantity: 2, UnitPrice: products[7].SellingPrice},
	}
	for i := range items {
		db.FirstOrCreate(&items[i], "order_id = ? AND product_id = ?", items[i].OrderID, items[i].ProductID)
	}

	log.Printf("           Created %d products, %d orders", len(products), len(orders))
}


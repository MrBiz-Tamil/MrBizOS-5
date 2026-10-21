package hospital

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// SeedIfEmpty seeds 1 default hospital with wards and beds.
// Idempotent — only runs when no hospitals exist.
func SeedIfEmpty(db *gorm.DB) {
	var count int64
	db.Model(&Hospital{}).Count(&count)
	if count > 0 {
		return
	}

	log.Println("[Hospital] Seeding default hospital, wards and beds...")

	h := Hospital{
		Name:         "Dr.Ayya General Hospital",
		Address:      "Anna Nagar, Chennai 600040",
		ContactInfo:  "044-12345678",
		Type:         "General",
		HospitalType: "owned",
		IsActive:     true,
	}
	if err := db.Create(&h).Error; err != nil {
		log.Printf("[Hospital] Hospital seed error: %v", err)
		return
	}

	wards := []Ward{
		{HospitalID: h.ID, Name: "General Ward A", WardType: "general", Floor: 1, Capacity: 20, IsActive: true, Description: "General admissions"},
		{HospitalID: h.ID, Name: "ICU", WardType: "icu", Floor: 2, Capacity: 8, IsActive: true, Description: "Intensive care"},
		{HospitalID: h.ID, Name: "Maternity Ward", WardType: "maternity", Floor: 3, Capacity: 15, IsActive: true, Description: "Maternity and newborn care"},
		{HospitalID: h.ID, Name: "Private Rooms", WardType: "private", Floor: 4, Capacity: 10, IsActive: true, Description: "Private patient rooms"},
	}
	for i := range wards {
		if err := db.Create(&wards[i]).Error; err != nil {
			log.Printf("[Hospital] Ward seed error: %v", err)
			return
		}
	}

	// Per-bed pricing by ward type
	rateByType := map[string]float64{
		"general":   1500.0,
		"icu":       8000.0,
		"maternity": 3000.0,
		"private":   5000.0,
	}

	totalBeds := 0
	for _, w := range wards {
		rate := rateByType[w.WardType]
		if rate == 0 {
			rate = 2000.0
		}
		for i := 1; i <= w.Capacity; i++ {
			wardID := w.ID
			bed := Bed{
				HospitalID:    h.ID,
				WardID:        &wardID,
				RoomNumber:    fmt.Sprintf("R%d-%02d", w.Floor, i),
				BedNumber:     fmt.Sprintf("B%02d", i),
				BedType:       w.WardType,
				Status:        "available",
				PricePerNight: rate,
			}
			if err := db.Create(&bed).Error; err != nil {
				log.Printf("[Hospital] Bed seed error: %v", err)
				return
			}
			totalBeds++
		}
	}

	log.Printf("[Hospital] Seed complete — 1 hospital, %d wards, %d beds",
		len(wards), totalBeds)
}

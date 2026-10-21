package main

import (
	"log"
	"drayya-core/internal/domain/franchise"
	"drayya-core/internal/domain/hospital"
	"drayya-core/internal/domain/identity"

	"gorm.io/gorm"
)

func seedHospital(db *gorm.DB) {
	// Create franchise owner user first
	var franchiseOwner identity.User
	db.Where("email = ?", "admin@Dr.Ayya.com").First(&franchiseOwner)

	// Create franchise
	f := franchise.Franchise{
		OwnerID:     franchiseOwner.ID,
		Name:        "Dr.Ayya South Franchise",
		RoyaltyRate: 10.0,
		Region:      "South India",
		IsActive:    true,
	}
	db.FirstOrCreate(&f, "name = ?", f.Name)

	// Hospitals
	hospitals := []hospital.Hospital{
		{
			Name:         "Dr.Ayya Central Hospital",
			Address:      "100 Healthcare Avenue, Mumbai, MH 400001",
			ContactInfo:  "022-12345678",
			Type:         "General",
			HospitalType: "owned",
			IsActive:     true,
		},
		{
			Name:         "Dr.Ayya South Hospital",
			Address:      "200 Medical Street, Chennai, TN 600001",
			ContactInfo:  "044-87654321",
			Type:         "Specialized",
			HospitalType: "franchise",
			FranchiseID:  &f.ID,
			IsActive:     true,
		},
	}
	for i := range hospitals {
		db.FirstOrCreate(&hospitals[i], "name = ?", hospitals[i].Name)
	}

	// Departments for Hospital 1
	depts := []hospital.HospitalDepartment{
		{HospitalID: hospitals[0].ID, Name: "General Medicine", Description: "Primary care and internal medicine"},
		{HospitalID: hospitals[0].ID, Name: "Cardiology", Description: "Heart and cardiovascular care"},
		{HospitalID: hospitals[0].ID, Name: "Pediatrics", Description: "Children's healthcare"},
		{HospitalID: hospitals[0].ID, Name: "Emergency", Description: "24/7 emergency services"},
		{HospitalID: hospitals[0].ID, Name: "Laboratory", Description: "Diagnostic lab services"},
		{HospitalID: hospitals[1].ID, Name: "Dermatology", Description: "Skin and hair care"},
		{HospitalID: hospitals[1].ID, Name: "General Medicine", Description: "Primary care"},
	}
	for i := range depts {
		db.FirstOrCreate(&depts[i], "hospital_id = ? AND name = ?", depts[i].HospitalID, depts[i].Name)
	}

	// Beds
	beds := []hospital.Bed{
		{HospitalID: hospitals[0].ID, DepartmentID: depts[0].ID, RoomNumber: "101", BedNumber: "A", Status: "available", PricePerNight: 250000},
		{HospitalID: hospitals[0].ID, DepartmentID: depts[0].ID, RoomNumber: "101", BedNumber: "B", Status: "occupied", PricePerNight: 250000},
		{HospitalID: hospitals[0].ID, DepartmentID: depts[1].ID, RoomNumber: "201", BedNumber: "A", Status: "available", PricePerNight: 500000},
		{HospitalID: hospitals[0].ID, DepartmentID: depts[3].ID, RoomNumber: "ICU-1", BedNumber: "A", Status: "available", PricePerNight: 1000000},
		{HospitalID: hospitals[0].ID, DepartmentID: depts[3].ID, RoomNumber: "ICU-1", BedNumber: "B", Status: "available", PricePerNight: 1000000},
		{HospitalID: hospitals[1].ID, DepartmentID: depts[5].ID, RoomNumber: "301", BedNumber: "A", Status: "available", PricePerNight: 300000},
	}
	for i := range beds {
		db.FirstOrCreate(&beds[i], "hospital_id = ? AND room_number = ? AND bed_number = ?",
			beds[i].HospitalID, beds[i].RoomNumber, beds[i].BedNumber)
	}

	log.Printf("       Created %d hospitals, %d departments, %d beds", len(hospitals), len(depts), len(beds))
}


package main

import (
	"log"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/money"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env", "../../.env"); err != nil {
		log.Println("Warning: .env not found")
	}
	database.Connect()

	// 1. Check for existing doctor user or create one
	var user identity.User
	email := "doctor@example.com"
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		log.Printf("Doctor user not found, creating new one...")
		hashedPassword := "$2a$10$YourHashedPasswordHere" // details omitted for brevity
		user = identity.User{
			Email:         email,
			PasswordHash:  hashedPassword,
			Role:          "doctor",
			FullName:      "Dr. Test One",
			IsActive:      true,
			IsVerified:    true,
			WalletBalance: 100000, // Cents
		}
		if err := database.DB.Create(&user).Error; err != nil {
			log.Fatalf("Failed to create user: %v", err)
		}
	} else {
		log.Printf("Found existing doctor user: %d", user.ID)
	}

	// 2. Check/Create Doctor Profile
	var doctor identity.DoctorProfile
	if err := database.DB.Where("user_id = ?", user.ID).First(&doctor).Error; err != nil {
		log.Printf("Doctor profile not found, creating...")
		doctor = identity.DoctorProfile{
			UserID:            user.ID,
			Specialization:    "General Medicine",
			ConsultationFees:  money.RegionalPricing{"INR": 50000}, // Cents
			YearsOfExperience: 5,
			Qualification:     "MBBS",
			Bio:               "Test Doctor",
			IsVerified:        true,
			IsOnline:          true,
			AverageRating:     5.0,
		}
		if err := database.DB.Create(&doctor).Error; err != nil {
			log.Fatalf("Failed to create profile: %v", err)
		}
	} else {
		// Update existing
		doctor.IsOnline = true
		doctor.IsVerified = true
		doctor.AverageRating = 5.0
		if err := database.DB.Save(&doctor).Error; err != nil {
			log.Fatalf("Failed to save doctor: %v", err)
		}
	}

	log.Printf("Successfully set Doctor %d (User %d) to Online & Verified", doctor.ID, doctor.UserID)
}

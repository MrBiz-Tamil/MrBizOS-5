package main

import (
	"log"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	log.Println("[Production Seed] Starting...")
	
	// Load production environment variables
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("No .env file found, relying on environment variables.")
	}

	database.Connect()
	db := database.DB

	log.Println("[Production Seed] Running strict auto-migrations...")
	identity.AutoMigrate(db)
	
	log.Println("[Production Seed] Preparing Super Admin Account...")
	
	adminEmail := "dr.ayya@mrbiz.in"
	adminPassword := "AyyaSecure@2026"
	
	hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash admin password: %v", err)
	}

	superAdmin := identity.User{
		Email:        adminEmail,
		PasswordHash: string(hash),
		Role:         "admin",
		FullName:     "Dr. Ayya Admin",
		IsActive:     true,
		IsVerified:   true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// FirstOrCreate so it doesn't duplicate if script is run twice
	res := db.Where("email = ?", adminEmail).FirstOrCreate(&superAdmin)
	if res.Error != nil {
		log.Fatalf("Failed to create super admin: %v", res.Error)
	}

	log.Println("[Production Seed] Admin account provisioned successfully.")
	log.Println("[Production Seed] DONE. Database is ready for production.")
}

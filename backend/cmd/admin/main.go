package main

import (
	"flag"
	"fmt"
	"log"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	emailFlag := flag.String("email", "sithcare@gmail.com", "Admin email")
	nameFlag := flag.String("name", "Ashok Pauliah", "Admin full name")
	passFlag := flag.String("password", "Test@123", "Admin password")
	roleFlag := flag.String("role", "super_admin", "User role")
	flag.Parse()

	if err := godotenv.Load(".env", "../../.env"); err != nil {
		log.Println("Note: .env not found, using system env vars")
	}

	database.Connect()
	db := database.DB

	var user identity.User
	hash, err := bcrypt.GenerateFromPassword([]byte(*passFlag), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	if db.Where("email = ?", *emailFlag).First(&user).Error == nil {
		// Update existing user
		updates := map[string]interface{}{
			"password_hash": string(hash),
			"role":          *roleFlag,
			"full_name":     *nameFlag,
			"is_active":     true,
			"is_verified":   true,
		}
		
		if user.UHEID == nil || *user.UHEID == "" {
			date := time.Now().Format("20060102")
			random := uuid.New().String()[:8]
			uheid := fmt.Sprintf("MH-ADMN-%s-%s", date, strings.ToUpper(random))
			updates["uheid"] = &uheid
		}
		
		if err := db.Model(&user).Updates(updates).Error; err != nil {
			log.Fatalf("Failed to update user: %v", err)
		}
		log.Printf("Successfully updated user: %s", *emailFlag)
	} else {
		// Create new user
		date := time.Now().Format("20060102")
		random := uuid.New().String()[:8]
		uheid := fmt.Sprintf("MH-ADMN-%s-%s", date, strings.ToUpper(random))
		
		user = identity.User{
			Email:        *emailFlag,
			PasswordHash: string(hash),
			FullName:     *nameFlag,
			Role:         *roleFlag,
			IsActive:     true,
			IsVerified:   true,
			UHEID:        &uheid,
		}
		
		if err := db.Create(&user).Error; err != nil {
			log.Fatalf("Failed to create user: %v", err)
		}
		log.Printf("Successfully created user: %s (UHEID: %s)", *emailFlag, uheid)
	}
}

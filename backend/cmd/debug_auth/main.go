package main

import (
	"fmt"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	godotenv.Load(".env")
	database.Connect()
	db := database.DB

	emails := []string{
		"doctor@Dr.Ayya.com",
		"admin@Dr.Ayya.com",
		"hr@Dr.Ayya.com",
		"finance@Dr.Ayya.com",
	}

	fmt.Println("--- User Verification ---")
	for _, email := range emails {
		var user identity.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			fmt.Printf("    %s: Not found\n", email)
			continue
		}

		err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("test123"))
		passMatch := "MATCH"
		if err != nil {
			passMatch = "FAIL"
		}

		fmt.Printf("    %s | Role: %s | Active: %v | Pass: %s\n", email, user.Role, user.IsActive, passMatch)
	}
}


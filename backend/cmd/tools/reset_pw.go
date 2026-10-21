package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run reset_pw.go <email> <new_password>")
		os.Exit(1)
	}

	email := strings.TrimSpace(os.Args[1])
	newPassword := os.Args[2]

	// Load .env from project root
	godotenv.Load("../../.env")

	// Init DB
	database.Connect()
	if database.DB == nil {
		log.Fatal("Failed to connect to database")
	}

	var user identity.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		log.Fatalf("User with email %s not found: %v", email, err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	if err := database.DB.Model(&user).Update("password_hash", string(hash)).Error; err != nil {
		log.Fatalf("Failed to update password: %v", err)
	}

	fmt.Printf("Successfully updated password for %s\n", email)
}

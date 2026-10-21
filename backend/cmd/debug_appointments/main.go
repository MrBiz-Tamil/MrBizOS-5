package main

import (
	"fmt"
	"log"
	"drayya-core/internal/domain/appointment"
	"os"

	"github.com/glebarez/sqlite"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "Dr.Ayya.db"
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	var appt appointment.Appointment
	// Just like the handler does
	result := db.Preload("Patient").Preload("Doctor").First(&appt, 3)

	if result.Error != nil {
		fmt.Printf("Error finding appointment 3: %v\n", result.Error)
	} else {
		fmt.Printf("Found appointment 3: PatientID=%d, Status=%s\n", appt.PatientID, appt.Status)
		if appt.Patient.ID != 0 {
			fmt.Printf("  Patient found: ID=%d\n", appt.Patient.ID)
		} else {
			fmt.Printf("  Patient NOT preloaded!\n")
		}
	}
}

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"drayya-core/internal/database"

	"github.com/joho/godotenv"
)

func main() {
	reset := flag.Bool("reset", false, "Drop all tables before seeding")
	domain := flag.String("domain", "", "Seed only a specific domain")
	flag.Parse()

	if err := godotenv.Load(".env", "../../.env"); err != nil {
		log.Println("Note: .env not found, using system env vars")
	}

	if *reset {
		appEnv := os.Getenv("APP_ENV")
		if appEnv == "production" || appEnv == "prod" {
			fmt.Fprintln(os.Stderr, "ERROR: -reset is blocked in production (APP_ENV=production). This would destroy all data.")
			os.Exit(1)
		}
		// Require explicit confirmation so accidental runs are caught
		fmt.Print("WARNING: -reset will DROP ALL TABLES and DELETE ALL DATA. Type 'yes-delete-everything' to confirm: ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes-delete-everything" {
			fmt.Fprintln(os.Stderr, "Aborted.")
			os.Exit(1)
		}
	}

	database.Connect()
	db := database.DB

	if *reset {
		log.Println("        Reset flag set     dropping and recreating all tables...")
		dropAllTables(db)
	}

	log.Println("     Running AutoMigrate...")
	runMigrations(db)

	seeders := []struct {
		name string
		fn   func()
	}{
		{"identity", func() { seedIdentity(db) }},
		{"hospital", func() { seedHospital(db) }},
		{"pharmacy", func() { seedPharmacy(db) }},
		{"appointment", func() { seedAppointment(db) }},
		{"emr", func() { seedEMR(db) }},
		{"billing", func() { seedBilling(db) }},
		{"lab", func() { seedLab(db) }},
		{"communication", func() { seedCommunication(db) }},
		{"hr", func() { seedHR(db) }},
		{"store", func() { seedStore(db) }},
		{"dispatch", func() { seedDispatch(db) }},
		{"finance", func() { seedFinance(db) }},
	}

	for _, s := range seeders {
		if *domain != "" && *domain != s.name {
			continue
		}
		log.Printf("     Seeding %s...", s.name)
		s.fn()
		log.Printf("    %s seeded", s.name)
	}

	log.Println("\n                                                                                                                                                ")
	log.Println("     Database seeding complete!")
	printCredentials()
}


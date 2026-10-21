//go:build ignore

package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// User model (simplified for seeding)
type User struct {
	ID           uint   `gorm:"primaryKey"`
	Email        string `gorm:"uniqueIndex;not null"`
	PasswordHash string `gorm:"not null"`
	FullName     string
	Role         string `gorm:"default:'patient'"`
	IsActive     bool   `gorm:"default:true"`
}

func (User) TableName() string {
	return "user"
}

// DoctorProfile model
type DoctorProfile struct {
	ID             uint `gorm:"primaryKey"`
	UserID         uint `gorm:"uniqueIndex;not null"`
	Specialization string
	IsVerified     bool `gorm:"default:false"`
	IsOnline       bool `gorm:"default:false"`
}

func (DoctorProfile) TableName() string {
	return "doctorprofile"
}

func main() {
	// Load environment variables
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Connect to database
	db, err := connectDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("    Connected to database")

	// Auto-migrate tables
	db.AutoMigrate(&User{}, &DoctorProfile{})

	// Create test users
	testUsers := []struct {
		Email    string
		Password string
		FullName string
		Role     string
		Doctor   *DoctorProfile
	}{
		{
			Email:    "admin@Dr.Ayya.com",
			Password: "admin123",
			FullName: "Admin User",
			Role:     "admin",
		},
		{
			Email:    "doctor@Dr.Ayya.com",
			Password: "doctor123",
			FullName: "Dr. John Smith",
			Role:     "doctor",
			Doctor: &DoctorProfile{
				Specialization: "General Medicine",
				IsVerified:     true,
				IsOnline:       true,
			},
		},
		{
			Email:    "doctor2@Dr.Ayya.com",
			Password: "doctor123",
			FullName: "Dr. Sarah Johnson",
			Role:     "doctor",
			Doctor: &DoctorProfile{
				Specialization: "Cardiology",
				IsVerified:     true,
				IsOnline:       false,
			},
		},
		{
			Email:    "patient@Dr.Ayya.com",
			Password: "patient123",
			FullName: "Jane Doe",
			Role:     "patient",
		},
		{
			Email:    "patient2@Dr.Ayya.com",
			Password: "patient123",
			FullName: "Robert Brown",
			Role:     "patient",
		},
		{
			Email:    "pharmacist@Dr.Ayya.com",
			Password: "pharmacy123",
			FullName: "Pharmacy Manager",
			Role:     "pharmacist",
		},
	}

	log.Println("\n     Seeding test users...")

	for _, testUser := range testUsers {
		// Check if user already exists
		var existingUser User
		result := db.Where("email = ?", testUser.Email).First(&existingUser)

		if result.Error == nil {
			log.Printf("        User %s already exists, skipping...", testUser.Email)
			continue
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(testUser.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("    Failed to hash password for %s: %v", testUser.Email, err)
			continue
		}

		// Create user
		user := User{
			Email:        testUser.Email,
			PasswordHash: string(hashedPassword),
			FullName:     testUser.FullName,
			Role:         testUser.Role,
			IsActive:     true,
		}

		if err := db.Create(&user).Error; err != nil {
			log.Printf("    Failed to create user %s: %v", testUser.Email, err)
			continue
		}

		log.Printf("    Created user: %s (%s)", testUser.Email, testUser.Role)

		// Create doctor profile if needed
		if testUser.Doctor != nil {
			testUser.Doctor.UserID = user.ID
			if err := db.Create(testUser.Doctor).Error; err != nil {
				log.Printf("        Failed to create doctor profile for %s: %v", testUser.Email, err)
			} else {
				log.Printf("                  Created doctor profile: %s", testUser.Doctor.Specialization)
			}
		}
	}

	log.Println("\n    Seeding completed!")
	log.Println("\n     Test Credentials:")
	log.Println("                                                                                                                                          ")
	log.Println("Admin:")
	log.Println("  Email: admin@Dr.Ayya.com")
	log.Println("  Password: admin123")
	log.Println("")
	log.Println("Doctors:")
	log.Println("  Email: doctor@Dr.Ayya.com")
	log.Println("  Password: doctor123")
	log.Println("  (Dr. John Smith - General Medicine)")
	log.Println("")
	log.Println("  Email: doctor2@Dr.Ayya.com")
	log.Println("  Password: doctor123")
	log.Println("  (Dr. Sarah Johnson - Cardiology)")
	log.Println("")
	log.Println("Patients:")
	log.Println("  Email: patient@Dr.Ayya.com")
	log.Println("  Password: patient123")
	log.Println("  (Jane Doe)")
	log.Println("")
	log.Println("  Email: patient2@Dr.Ayya.com")
	log.Println("  Password: patient123")
	log.Println("  (Robert Brown)")
	log.Println("")
	log.Println("Pharmacist:")
	log.Println("  Email: pharmacist@Dr.Ayya.com")
	log.Println("  Password: pharmacy123")
	log.Println("                                                                                                                                          ")
}

func connectDB() (*gorm.DB, error) {
	useSQLite := os.Getenv("USE_SQLITE") == "true"
	dbURL := os.Getenv("DB_URL")

	if useSQLite || (dbURL != "" && len(dbURL) > 7 && dbURL[:7] == "sqlite:") {
		// SQLite
		sqliteDB := "./Dr.Ayya.db"
		if dbURL != "" && len(dbURL) > 9 {
			sqliteDB = dbURL[9:] // Remove "sqlite://"
		}
		log.Printf("Using SQLite database: %s", sqliteDB)
		return gorm.Open(sqlite.Open(sqliteDB), &gorm.Config{})
	}

	// PostgreSQL
	if dbURL == "" {
		// Build from individual variables
		host := os.Getenv("POSTGRES_SERVER")
		user := os.Getenv("POSTGRES_USER")
		password := os.Getenv("POSTGRES_PASSWORD")
		dbname := os.Getenv("POSTGRES_DB")
		port := os.Getenv("POSTGRES_PORT")

		dbURL = "host=" + host + " user=" + user + " password=" + password +
			" dbname=" + dbname + " port=" + port + " sslmode=disable"
	}

	log.Println("Using PostgreSQL database")
	return gorm.Open(postgres.Open(dbURL), &gorm.Config{})
}


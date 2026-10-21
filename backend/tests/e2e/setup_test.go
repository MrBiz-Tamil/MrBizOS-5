package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/billing"
	"drayya-core/internal/domain/communication"
	"drayya-core/internal/domain/emr"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/hospital"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/domain/rating"
	"drayya-core/internal/domain/loyalty"
	"drayya-core/internal/domain/hr"
	"drayya-core/internal/domain/subscription"
	"drayya-core/internal/server"
	"drayya-core/internal/tests/common"
	"drayya-core/internal/money"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var app *fiber.App

func TestMain(m *testing.M) {
	// Setup
	app = setupLocalApp()

	// Additional seed data if needed for E2E
	seedData()

	// Run Tests
	code := m.Run()

	// Teardown (if any)
	os.Exit(code)
}

// setupLocalApp initializes the app without logger middleware to keep test output clean
func setupLocalApp() *fiber.App {
	// 0. Setup Test Secret
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("SKIP_LIMITER", "true")
	os.Setenv("ALLOW_DEBUG_WALLET_TOPUP", "true")

	// 1. Setup In-Memory Database
	var err error
	// Use shared cache to ensure all connections see the same DB
	database.DB, err = gorm.Open(sqlite.Open("file:medi_test_db?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // Keep info for debug
	})
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}

	// 2. Run Migrations
	identity.AutoMigrate(database.DB)
	subscription.AutoMigrate(database.DB)
	emr.AutoMigrate(database.DB)
	pharmacy.AutoMigrate(database.DB)
	appointment.AutoMigrate(database.DB)
	billing.AutoMigrate(database.DB)
	finance.AutoMigrate(database.DB)
	hospital.AutoMigrate(database.DB)
	communication.AutoMigrate(database.DB)
	rating.AutoMigrate(database.DB)
	loyalty.AutoMigrate(database.DB)
	hr.AutoMigrate(database.DB)

	// Debug: Check if LedgerAccount table exists
	if !database.DB.Migrator().HasTable(&finance.LedgerAccount{}) {
		log.Println("CRITIAL ERROR: LedgerAccount table missing after AutoMigrate!")
		// Ensure we log to file too if possible or just panic
		os.WriteFile("migration_fail.txt", []byte("LedgerAccount table missing"), 0644)
	} else {
		os.WriteFile("migration_success.txt", []byte("LedgerAccount table exists"), 0644)
	}

	// 3. Initialize App (Routes)
	// NOTE: We are using server.New() which adds logger middleware.
	// Since we cannot easily modify server.New() without affecting prod,
	// we accept it but maybe we can silence stdout?
	// For now, let's just accept it. The previous failure might be due to panic, not just noise.
	app := server.New()

	return app
}

func seedData() {
	// Ensure we have at least one doctor
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(TestPassword), bcrypt.DefaultCost)

	// Create specific E2E users
	patient := identity.User{
		Email:         TestPatientEmail,
		PasswordHash:  string(passwordHash),
		Role:          "patient",
		FullName:      "E2E Patient",
		IsActive:      true,
		IsVerified:    true,
		WalletBalance: 5000.0, // Pre-topup
	}
	database.DB.Create(&patient)
	PatientID = patient.ID

	doctor := identity.User{
		Email:         TestDoctorEmail,
		PasswordHash:  string(passwordHash),
		Role:          "admin",
		FullName:      "E2E Doctor",
		IsActive:      true,
		IsVerified:    true,
		WalletBalance: 0,
	}
	database.DB.Create(&doctor)
	DoctorID = doctor.ID

	// Create Profile
	docProfile := identity.DoctorProfile{
		UserID:          doctor.ID,
		Specialization:  "General",
		ConsultationFees: money.RegionalPricing{"INR": 50000},
		IsOnline:        true,
		IsVerified:      true,
		AverageRating:   5.0,
		UpdatedAt:       time.Now(),
	}
	database.DB.Create(&docProfile)

	// Generate Tokens
	PatientToken = common.GenerateTestToken(patient)
	DoctorToken = common.GenerateTestToken(doctor)
}

// Helper: Make Authenticated Request
func makeRequest(method, url, token string, body interface{}) (*http.Request, *http.Response, error) {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}

	req, _ := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	fmt.Printf("DEBUG: Request %s %s with Auth Header: %s\n", method, url, req.Header.Get("Authorization"))

	resp, err := app.Test(req, -1)
	return req, resp, err
}

// Helper: Parse Response - Automatically handles the standardized API envelope
func parseResponse(resp *http.Response, target interface{}) error {
	var tr TestResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return err
	}
	if !tr.Success {
		return fmt.Errorf("API error: %s", tr.Message)
	}
	if target != nil && len(tr.Data) > 0 {
		return json.Unmarshal(tr.Data, target)
	}
	return nil
}

func GetDB() *gorm.DB {
	return database.DB
}

package common

import (
	"log"
	"os"
	"strconv"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/billing"
	"drayya-core/internal/domain/communication"
	devdomain "drayya-core/internal/domain/devi"
	"drayya-core/internal/domain/emr"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/hospital"
	"drayya-core/internal/domain/hr"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/subscription"
	"drayya-core/internal/domain/insurance"
	"drayya-core/internal/domain/lab"
	"drayya-core/internal/handlers"
	payoutdomain "drayya-core/internal/domain/payout"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/domain/rating"
	"drayya-core/internal/services/notification"
	"drayya-core/internal/domain/store"
	"drayya-core/internal/server"
	"drayya-core/internal/tokens"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupApp initializes the app and an in-memory test database
func SetupApp() *fiber.App {
	// 0. Setup Test Secret
	os.Setenv("JWT_SECRET", "test-secret")

	// 1. Setup In-Memory Database
	// Pin to 1 connection — SQLite :memory: is per-connection, so multiple
	// connections would each see a separate empty database.
	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}
	if sqlDB, err := database.DB.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
	}

	// 2. Run Migrations — delegate to each domain's AutoMigrate so tables stay
	// in sync with the models without needing to manually track every struct.
	for _, migrateFn := range []func(*gorm.DB) error{
		identity.AutoMigrate,
		subscription.AutoMigrate,
		appointment.AutoMigrate,
		billing.AutoMigrate,
		finance.AutoMigrate,
		insurance.AutoMigrate,
		rating.AutoMigrate,
		hr.AutoMigrate,
		emr.AutoMigrate,
		communication.AutoMigrate,
		notification.AutoMigrateOutbox,
	} {
		if err := migrateFn(database.DB); err != nil {
			log.Fatalf("Failed to migrate test database: %v", err)
		}
	}
	err = database.DB.AutoMigrate(
		&emr.Prescription{},
		&emr.MedicalRecord{},
		&pharmacy.Pharmacy{},
		&pharmacy.Medicine{},
		&pharmacy.PharmacyOrder{},
		&pharmacy.PharmacyOrderItem{},
		&pharmacy.PharmacyInventory{},
		&billing.Payment{},
		&hospital.Hospital{},
		&hospital.HospitalDepartment{},
		&hospital.Bed{},
		&hospital.Ward{},
		&hospital.NursingTask{},
		&hospital.ShiftSchedule{},
		&lab.LabTest{},
		&lab.LabPackageItem{},
		&lab.LabPartner{},
		&lab.LabCenter{},
		&lab.LabCenterPricing{},
		&lab.LabBooking{},
		&lab.LabRequest{},
		&lab.LabResult{},
		&lab.Specimen{},
		&handlers.SOP{},
		&communication.NotificationLog{},
		&store.WellnessProduct{},
		&store.WellnessOrder{},
		&store.WellnessOrderItem{},
		&pharmacy.CourierProfile{},
		&pharmacy.CourierDelivery{},
		&pharmacy.Shipment{},
		&pharmacy.ShipmentUpdate{},
		&devdomain.SymptomLog{},
		&payoutdomain.Record{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate test database: %v", err)
	}

	// 3. Initialize App (Routes)
	// We use the same server.New() to get all routes
	app := server.New()

	return app
}

// CreateTestUser creates a user for testing
func CreateTestUser(db *gorm.DB, role string) identity.User {
	user := identity.User{
		Email:        "test_" + role + "@example.com",
		PasswordHash: "hashed_secret", // In real tests, might need valid hash if login tested
		Role:         role,
		FullName:     "Test " + role,
		IsActive:     true,
	}
	db.Create(&user)
	return user
}

// GenerateTestToken creates a valid token for testing
func GenerateTestToken(user identity.User) string {
	token, _ := tokens.CreateAccessToken(strconv.FormatUint(uint64(user.ID), 10), user.Role, user.HospitalID, nil)
	return token
}

// CreateTestAppointment creates a dummy appointment
func CreateTestAppointment(db *gorm.DB, status string) appointment.Appointment {
	appt := appointment.Appointment{
		Status:    appointment.AppointmentStatus(status),
		StartTime: time.Now(),
		EndTime:   time.Now().Add(1 * time.Hour),
	}
	db.Create(&appt)
	return appt
}

// CreateTestPayment creates a dummy payment
func CreateTestPayment(db *gorm.DB, amount int64, status string) billing.Payment {
	payment := billing.Payment{
		Amount:    amount,
		Status:    status,
		UpdatedAt: time.Now(),
	}
	db.Create(&payment)
	return payment
}

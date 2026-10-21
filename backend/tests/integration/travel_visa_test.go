package integration

import (
	"testing"

	"drayya-core/internal/domain/journey"
	"drayya-core/internal/domain/task"
	"drayya-core/internal/domain/travel"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTravelTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to memory db: %v", err)
	}

	if err := task.AutoMigrate(db); err != nil {
		t.Fatalf("task automigrate failed: %v", err)
	}
	if err := journey.AutoMigrate(db); err != nil {
		t.Fatalf("journey automigrate failed: %v", err)
	}
	if err := travel.AutoMigrate(db); err != nil {
		t.Fatalf("travel automigrate failed: %v", err)
	}

	return db
}

func TestTravelBookingAndVisaWorkflow(t *testing.T) {
	db := setupTravelTestDB(t)

	taskSvc := task.NewService(task.NewRepository(db))
	journeySvc := journey.NewService(db)
	travelSvc := travel.NewService(travel.NewRepository(db), journeySvc, taskSvc)

	// 1. Create Destination
	dest, err := travelSvc.CreateDestination(&travel.Destination{
		Name:        "Bali Island Paradise",
		Country:     "Indonesia",
		CountryCode: "ID",
	})
	if err != nil || dest.ID == 0 {
		t.Fatalf("CreateDestination failed: %v", err)
	}

	// 2. Create Tour Package
	pkg, err := travelSvc.CreateTourPackage(&travel.TourPackage{
		DestinationID: dest.ID,
		Title:         "7 Days Luxury Villa Resort & Tour",
		Days:          7,
		Nights:        6,
		Price:         1250.00,
	})
	if err != nil || pkg.ID == 0 {
		t.Fatalf("CreateTourPackage failed: %v", err)
	}

	// 3. Create Travel Booking (Triggers auto-task creation)
	booking, err := travelSvc.CreateBooking(101, pkg.ID, 2, "2026-10-15")
	if err != nil || booking.ID == 0 {
		t.Fatalf("CreateBooking failed: %v", err)
	}
	if booking.TotalPrice != 2500.00 {
		t.Fatalf("expected total price 2500.00, got %.2f", booking.TotalPrice)
	}

	// Verify task creation
	tasks, _, err := taskSvc.ListTasks(task.Filter{BusinessUnitID: 1})
	if err != nil || len(tasks) == 0 {
		t.Fatalf("expected travel task created, got err=%v tasksCount=%d", err, len(tasks))
	}

	// 4. Submit Visa Application
	visaApp, err := travelSvc.SubmitVisaApplication(101, "ID", "TOURIST", "B987654321")
	if err != nil || visaApp.ID == 0 {
		t.Fatalf("SubmitVisaApplication failed: %v", err)
	}
	if visaApp.Status != "SUBMITTED" {
		t.Fatalf("expected SUBMITTED, got %s", visaApp.Status)
	}

	// Update Visa status
	updatedVisa, err := travelSvc.UpdateVisaStatus(visaApp.ID, "APPROVED", "Visa granted electronically")
	if err != nil || updatedVisa.Status != "APPROVED" {
		t.Fatalf("UpdateVisaStatus failed: %v", err)
	}
}

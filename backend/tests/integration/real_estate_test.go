package integration

import (
	"testing"
	"time"

	"drayya-core/internal/domain/journey"
	"drayya-core/internal/domain/realestate"
	"drayya-core/internal/domain/task"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupRealEstateTestDB(t *testing.T) *gorm.DB {
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
	if err := realestate.AutoMigrate(db); err != nil {
		t.Fatalf("realestate automigrate failed: %v", err)
	}

	return db
}

func TestRealEstateOperationsWorkflow(t *testing.T) {
	db := setupRealEstateTestDB(t)

	taskSvc := task.NewService(task.NewRepository(db))
	journeySvc := journey.NewService(db)
	reSvc := realestate.NewService(realestate.NewRepository(db), journeySvc, taskSvc)

	// 1. Create Property Listing
	prop, err := reSvc.CreateProperty(&realestate.Property{
		Title:           "Grand Horizon 3BHK Luxury Penthouse",
		PropertyType:    realestate.TypeApartment,
		TransactionType: realestate.TxSale,
		Price:           850000.00,
		City:            "Dubai",
		Location:        "Downtown Marina",
		Bedrooms:        3,
		Bathrooms:       3,
		AreaSqFt:        2400.0,
	})
	if err != nil || prop.ID == 0 {
		t.Fatalf("CreateProperty failed: %v", err)
	}

	// 2. Schedule Site Visit (Auto-dispatches agent task)
	visitTime := time.Now().Add(48 * time.Hour)
	visit, err := reSvc.ScheduleSiteVisit(prop.ID, 201, 10, visitTime, "Client interested in high floor view")
	if err != nil || visit.ID == 0 {
		t.Fatalf("ScheduleSiteVisit failed: %v", err)
	}
	if visit.TaskID == nil {
		t.Fatalf("expected auto-created task ID for real estate agent")
	}

	// Verify agent task
	tasks, _, err := taskSvc.ListTasks(task.Filter{BusinessUnitID: 3})
	if err != nil || len(tasks) == 0 {
		t.Fatalf("expected agent task, got count=%d", len(tasks))
	}

	// 3. Create Deal
	deal, err := reSvc.CreateDeal(prop.ID, 201, 301, 840000.00)
	if err != nil || deal.ID == 0 {
		t.Fatalf("CreateDeal failed: %v", err)
	}
	if deal.CommissionAmount != 16800.00 {
		t.Fatalf("expected 2%% commission 16800.00, got %.2f", deal.CommissionAmount)
	}

	// Verify property reservation status
	updatedProp, _ := reSvc.GetProperty(prop.ID)
	if updatedProp.Status != "RESERVED" {
		t.Fatalf("expected RESERVED status, got %s", updatedProp.Status)
	}

	// 4. Close Deal
	closedDeal, err := reSvc.UpdateDealStatus(deal.ID, "CLOSED")
	if err != nil || closedDeal.Status != "CLOSED" {
		t.Fatalf("UpdateDealStatus failed: %v", err)
	}

	soldProp, _ := reSvc.GetProperty(prop.ID)
	if soldProp.Status != "SOLD" {
		t.Fatalf("expected SOLD status, got %s", soldProp.Status)
	}
}

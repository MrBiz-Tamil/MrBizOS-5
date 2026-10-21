package integration

import (
	"testing"

	"drayya-core/internal/domain/ecommerce"
	"drayya-core/internal/domain/task"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupEcommerceTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to memory db: %v", err)
	}

	if err := task.AutoMigrate(db); err != nil {
		t.Fatalf("task automigrate failed: %v", err)
	}
	if err := ecommerce.AutoMigrate(db); err != nil {
		t.Fatalf("ecommerce automigrate failed: %v", err)
	}

	return db
}

func TestEcommercePODContributionMarginWorkflow(t *testing.T) {
	db := setupEcommerceTestDB(t)

	taskSvc := task.NewService(task.NewRepository(db))
	ecSvc := ecommerce.NewService(ecommerce.NewRepository(db), taskSvc)

	// 1. Create Marketplace Listing
	listing, err := ecSvc.CreateListing("ETSY", "TSHIRT-AYUSH-L", "Custom AYUSH Botanical Heavyweight Tee", 34.99)
	if err != nil || listing.ID == 0 {
		t.Fatalf("CreateListing failed: %v", err)
	}

	// 2. Create POD Order (Calculates unit contribution profit margin)
	// Revenue: $34.99
	// Costs: Product=$11.50, Shipping=$4.50, EtsyFee=$2.20, PaymentFee=$1.10, AdSpend=$3.50 = Total Cost $22.80
	// Expected Contribution Margin: $34.99 - $22.80 = $12.19 (34.84%)
	order, err := ecSvc.CreatePODOrder(601, "TSHIRT-AYUSH-L", 1, 34.99, 11.50, 4.50, 2.20, 1.10, 3.50)
	if err != nil || order.ID == 0 {
		t.Fatalf("CreatePODOrder failed: %v", err)
	}

	if order.ContributionMargin < 12.18 || order.ContributionMargin > 12.20 {
		t.Fatalf("expected contribution margin ~12.19, got %.2f", order.ContributionMargin)
	}
	if order.ContributionMarginPct < 34.8 || order.ContributionMarginPct > 35.0 {
		t.Fatalf("expected margin approx 34.84%%, got %.2f%%", order.ContributionMarginPct)
	}

	// Verify auto-created fulfillment task for POD team
	tasks, _, err := taskSvc.ListTasks(task.Filter{BusinessUnitID: 4})
	if err != nil || len(tasks) == 0 {
		t.Fatalf("expected POD fulfillment task, got count=%d", len(tasks))
	}
}

package pharmacy

import (
	"context"
	"fmt"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/money"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupIntegrationDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=30000", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1) // Single connection for SQLite stability during cross-domain operations in tests
	db.Exec("PRAGMA journal_mode=WAL;")

	// Migrate both domains
	err = finance.AutoMigrate(db)
	if err != nil {
		t.Fatalf("finance migration failed: %v", err)
	}
	err = AutoMigrate(db)
	if err != nil {
		t.Fatalf("pharmacy migration failed: %v", err)
	}

	return db
}

func TestPurchaseOrderIntegration(t *testing.T) {
	db := setupIntegrationDB(t)
	ctx := context.Background()

	financeRepo := finance.NewRepository(db)
	financeSvc := finance.NewService(financeRepo)

	pharmacyRepo := NewDefaultRepository(db)
	pharmacySvc := NewService(pharmacyRepo, financeSvc)

	fmt.Println("--- Setup Data Start ---")

	// 1. Setup Data
	// Supplier
	supplier := Supplier{Name: "Global Biotech", ContactEmail: "sales@global.com"}
	db.Create(&supplier)

	// Pharmacy
	pharma := Pharmacy{Name: "Central Hub", City: "Bangalore", Region: "Bangalore", IsCentralWarehouse: true}
	db.Create(&pharma)

	// Medicine — UnitPrice was replaced with Prices (RegionalPricing) in the
	// money model migration; INR price stored in paise.
	med := Medicine{Name: "Paracetamol 500mg", Brand: "Dolo", Category: "tablet",
		Prices: money.RegionalPricing{"INR": 100}} // 1 INR
	db.Create(&med)

	// Tax Rule (12% GST for MEDICINE in Bangalore region)
	err := financeSvc.CreateTaxRule(ctx, finance.TaxRule{
		Region:   "Bangalore",
		Category: "MEDICINE",
		TaxType:  "GST",
		Rate:     12.0,
		IsActive: true,
	})
	assert.NoError(t, err)

	// 2. Create Purchase Order
	items := []map[string]interface{}{
		{"medicine_id": float64(med.ID), "qty": 1000.0, "unit_cost": 80.0}, // 800 INR total cost
	}
	fmt.Println("--- Creating PO ---")
	po, err := pharmacySvc.CreatePurchaseOrder(ctx, supplier.ID, pharma.ID, items)
	assert.NoError(t, err)
	assert.Equal(t, int64(80000), po.Amount) // TotalCost was renamed to Amount
	assert.Equal(t, int64(9600), po.TaxAmount) // 12% of 80000
	assert.Equal(t, int64(89600), po.TotalWithTax)
	assert.Equal(t, "draft", po.Status)

	// Approve PO (manually update status for test as we didn't implement ApprovePO logic in svc yet, just in handler/draft)
	fmt.Println("--- Updating PO status to ordered (manual) ---")
	po.Status = "ordered"
	db.Save(po)
	fmt.Println("--- PO status updated ---")

	// 3. Receive Purchase Order
	// This should trigger ledger entries
	fmt.Println("--- Receiving PO (calling svc) ---")
	err = pharmacySvc.ReceivePurchaseOrder(ctx, po.ID)
	fmt.Println("--- ReceivePurchaseOrder returned ---")
	if err != nil {
		t.Fatalf("ReceivePurchaseOrder failed: %v", err)
	}

	// 4. Verify PO State
	db.First(po, po.ID)
	assert.Equal(t, "received", po.Status)

	// 5. Verify Ledger Entries in Finance
	// Inventory Account: Debit 80000
	var invAcc finance.LedgerAccount
	err = db.WithContext(ctx).Where("name = ?", fmt.Sprintf("Asset:Inventory:Pharmacy:%d", pharma.ID)).Take(&invAcc).Error
	require.NoError(t, err)
	assert.Equal(t, int64(80000), invAcc.Balance)

	// Tax Account: Debit 9600
	var taxAcc finance.LedgerAccount
	err = db.WithContext(ctx).Where("name = ?", "Asset:Tax:InputGST").Take(&taxAcc).Error
	require.NoError(t, err)
	assert.Equal(t, int64(9600), taxAcc.Balance)

	// Payable Account: Credit 89600 (Increases for Liability)
	var payAcc finance.LedgerAccount
	err = db.WithContext(ctx).Where("name = ?", fmt.Sprintf("Liability:AccountsPayable:Supplier:%d", supplier.ID)).Take(&payAcc).Error
	require.NoError(t, err)
	assert.Equal(t, int64(89600), payAcc.Balance)

	// 6. Settle Supplier Payment
	// Setup Cash account for settlement and seed with balance
	cashAcct, err := financeSvc.GetOrCreateAccount(ctx, "Asset:Cash", finance.TypeAsset)
	require.NoError(t, err)
	cashAcct.Balance = 1000000 // Seed 10,000 INR
	cashAcct.LastChecksum = cashAcct.CalculateChecksum("SEED")
	db.Save(cashAcct)
	
	fmt.Println("--- Settling Payment ---")
	err = pharmacySvc.SettleSupplierPayment(ctx, po.ID)
	require.NoError(t, err)

	// Verify Payable is now 0
	err = db.WithContext(ctx).Where("name = ?", fmt.Sprintf("Liability:AccountsPayable:Supplier:%d", supplier.ID)).Take(&payAcc).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), payAcc.Balance)

	// Verify Cash is credited
	var cashAcc finance.LedgerAccount
	err = db.WithContext(ctx).Where("name = ?", "Asset:Cash").Take(&cashAcc).Error
	require.NoError(t, err)
	assert.Equal(t, int64(910400), cashAcc.Balance)
}

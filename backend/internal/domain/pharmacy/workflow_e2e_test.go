package pharmacy

import (
	"context"
	"encoding/json"
	"fmt"
	"drayya-core/internal/crypto"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/money"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ═══════════════════════════════════════════════════════════════════════════
// FULL END-TO-END WORKFLOW TEST
//
// Exercises the complete pharmacy cycle across multiple cities:
//
//   Raw Material Purchase  → QA Approve → BOM → Production Order →
//   Material Issuance → Manufacturing → QA → Release Finished Goods →
//   Finished Goods appear in PharmacyInventory → Customer Order →
//   Payment → Pack → Dispense → Ship → Deliver
//
// Also tests:
//   - Multi-city inventory transfer
//   - GRN (Goods Receipt Note) workflow
//   - RFQ (Request for Quotation) workflow
//   - Raw Material indent → approval
//   - Production order state machine enforcement
// ═══════════════════════════════════════════════════════════════════════════

func setupFullWorkflowDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=30000", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "failed to open test database")

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(2) // 2 conns: allows loyalty/global-DB reads while a transaction is active
	db.Exec("PRAGMA journal_mode=WAL;")
	db.Exec("PRAGMA busy_timeout=10000;")

	// Migrate all domains
	require.NoError(t, finance.AutoMigrate(db), "finance migration")
	require.NoError(t, AutoMigrate(db), "pharmacy migration")
	require.NoError(t, AutoMigrateManufacturing(db), "manufacturing migration")
	require.NoError(t, AutoMigrateRawMaterial(db), "raw material migration")

	return db
}

func TestFullPharmacyWorkflow_OrderToDispatch(t *testing.T) {
	db := setupFullWorkflowDB(t)
	ctx := context.Background()

	// Set global DB for loyalty and other packages that depend on it
	database.DB = db

	// ── Service Layer Setup ──
	financeRepo := finance.NewRepository(db)
	financeSvc := finance.NewService(financeRepo)
	pharmacyRepo := NewDefaultRepository(db)
	pharmacySvc := NewService(pharmacyRepo, financeSvc)
	mfgRepo := NewManufacturingRepository(db)
	rmRepo := NewRawMaterialRepository(db)
	rmSvc := NewRawMaterialService(rmRepo)
	mfgSvc := NewManufacturingService(mfgRepo, rmRepo, pharmacyRepo)

	// ── Tax Rule Setup ──
	require.NoError(t, financeSvc.CreateTaxRule(ctx, finance.TaxRule{
		Region: "Karnataka", Category: "MEDICINE", TaxType: "GST", Rate: 12.0, IsActive: true,
	}))

	t.Log("╔══════════════════════════════════════════════════════════╗")
	t.Log("║  STEP 1: SETUP — Cities, Warehouses, Suppliers         ║")
	t.Log("╚══════════════════════════════════════════════════════════╝")

	// Central manufacturing hub (Bangalore)
	centralHub := Pharmacy{
		Name: "Central Manufacturing Hub", City: "Bangalore", Region: "Karnataka",
		IsCentralWarehouse: true, HubTier: "global", IsActive: true,
		ServiceRadius: 50, Latitude: 12.9716, Longitude: 77.5946,
	}
	require.NoError(t, db.Create(&centralHub).Error)
	t.Logf("  ✓ Central Hub created: ID=%d, City=%s", centralHub.ID, centralHub.City)

	// Regional hub (Mumbai)
	mumbaiHub := Pharmacy{
		Name: "Mumbai Regional Hub", City: "Mumbai", Region: "Maharashtra",
		HubTier: "regional", ParentHubID: &centralHub.ID, IsActive: true,
		ServiceRadius: 30, Latitude: 19.076, Longitude: 72.8777,
	}
	require.NoError(t, db.Create(&mumbaiHub).Error)
	t.Logf("  ✓ Mumbai Hub created: ID=%d", mumbaiHub.ID)

	// Local pharmacy (Mumbai)
	mumbaiLocal := Pharmacy{
		Name: "Mumbai MG Road Pharmacy", City: "Mumbai", Region: "Maharashtra",
		HubTier: "local", ParentHubID: &mumbaiHub.ID, IsActive: true,
		ServiceRadius: 5,
	}
	require.NoError(t, db.Create(&mumbaiLocal).Error)
	t.Logf("  ✓ Mumbai Local Pharmacy created: ID=%d", mumbaiLocal.ID)

	// Supplier
	supplier := Supplier{Name: "PharmaChem India Pvt Ltd", ContactEmail: "sales@pharmachem.in", Phone: "+91-9876543210", IsActive: true}
	require.NoError(t, db.Create(&supplier).Error)
	t.Logf("  ✓ Supplier created: %s (ID=%d)", supplier.Name, supplier.ID)

	t.Log("")
	t.Log("╔══════════════════════════════════════════════════════════╗")
	t.Log("║  STEP 2: RAW MATERIAL — Register + Receive + QA        ║")
	t.Log("╚══════════════════════════════════════════════════════════╝")

	// Register raw materials
	paracetamolAPI := RawMaterial{
		Name: "Paracetamol API", CASNumber: "103-90-2", GenericName: "Acetaminophen",
		Category: "api", GradeType: "IP", PhysicalForm: "powder",
		StorageCondition: "ambient", DefaultUOM: "grams", ShelfLifeMonths: 36, IsActive: true,
	}
	require.NoError(t, rmSvc.CreateRawMaterial(ctx, &paracetamolAPI))
	t.Logf("  ✓ Raw material registered: %s (ID=%d, CAS=%s)", paracetamolAPI.Name, paracetamolAPI.ID, paracetamolAPI.CASNumber)

	excipientMCC := RawMaterial{
		Name: "Microcrystalline Cellulose (MCC)", Category: "excipient",
		GradeType: "IP", PhysicalForm: "powder", StorageCondition: "ambient",
		DefaultUOM: "grams", ShelfLifeMonths: 60, IsActive: true,
	}
	require.NoError(t, rmSvc.CreateRawMaterial(ctx, &excipientMCC))
	t.Logf("  ✓ Raw material registered: %s (ID=%d)", excipientMCC.Name, excipientMCC.ID)

	// Create approved vendor
	av := ApprovedVendor{
		SupplierID: supplier.ID, RawMaterialID: paracetamolAPI.ID,
		CountryOfOrigin: "India", LeadTimeDays: 7, PricePerUnit: 40000, // ₹400/kg in paise
		CoAOnFile: true, IsPreferred: true, IsActive: true,
	}
	require.NoError(t, rmSvc.CreateApprovedVendor(ctx, &av))
	t.Logf("  ✓ Approved vendor: Supplier #%d for %s", supplier.ID, paracetamolAPI.Name)

	// Receive raw materials (goes to quarantine automatically)
	paracetamolInv := RawMaterialInventory{
		RawMaterialID: paracetamolAPI.ID, WarehouseID: centralHub.ID,
		Quantity: 5000, // 5kg in grams
		UnitOfMeasure: "grams", SupplierID: supplier.ID,
		BatchNumber: "BATCH-PC-2026-001",
		ExpiryDate:  time.Now().AddDate(2, 0, 0), // 2 years
		CostPerUnit: 40, // 0.40 INR/gram
		ReorderLevel: 1000,
	}
	require.NoError(t, rmSvc.ReceiveRawMaterial(ctx, &paracetamolInv, 1))
	assert.Equal(t, "quarantine", paracetamolInv.QAStatus)
	assert.NotEmpty(t, paracetamolInv.LotNumber)
	t.Logf("  ✓ Paracetamol API received: %d grams → Quarantine (Lot: %s)", int(paracetamolInv.Quantity), paracetamolInv.LotNumber)

	mccInv := RawMaterialInventory{
		RawMaterialID: excipientMCC.ID, WarehouseID: centralHub.ID,
		Quantity: 3000, UnitOfMeasure: "grams", SupplierID: supplier.ID,
		BatchNumber: "BATCH-MCC-2026-001",
		ExpiryDate: time.Now().AddDate(4, 0, 0),
		CostPerUnit: 10, ReorderLevel: 500,
	}
	require.NoError(t, rmSvc.ReceiveRawMaterial(ctx, &mccInv, 1))
	t.Logf("  ✓ MCC excipient received: %d grams → Quarantine", int(mccInv.Quantity))

	// Verify quarantine — balance should be 0 (only approved counts)
	balance, err := rmSvc.GetRMBalance(ctx, paracetamolAPI.ID, centralHub.ID)
	require.NoError(t, err)
	assert.Equal(t, 0.0, balance, "Quarantined material should have 0 approved balance")
	t.Log("  ✓ Verified: quarantined stock has 0 approved balance")

	// QA Approve both
	require.NoError(t, rmSvc.ApproveQA(ctx, paracetamolInv.ID, 1, "Purity 99.7% — PASS per IP standards"))
	require.NoError(t, rmSvc.ApproveQA(ctx, mccInv.ID, 1, "Identity + moisture within limits"))
	t.Log("  ✓ QA Approved: Paracetamol API")
	t.Log("  ✓ QA Approved: MCC Excipient")

	// Verify balance after approval
	balance, err = rmSvc.GetRMBalance(ctx, paracetamolAPI.ID, centralHub.ID)
	require.NoError(t, err)
	assert.Equal(t, 5000.0, balance, "After QA approval, 5000g should be available")
	t.Logf("  ✓ Verified: approved balance = %.0f grams", balance)

	// Verify transaction trail
	txns, err := rmSvc.GetRMTransactions(ctx, paracetamolInv.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(txns), 2, "Should have receipt + qa_release transactions")
	t.Logf("  ✓ Transaction audit trail: %d entries", len(txns))

	t.Log("")
	t.Log("╔══════════════════════════════════════════════════════════╗")
	t.Log("║  STEP 3: BOM + PRODUCTION ORDER — Manufacturing        ║")
	t.Log("╚══════════════════════════════════════════════════════════╝")

	// Create finished medicine
	paracetamol := Medicine{
		Name: "Paracetamol 500mg", Brand: "Dolo", Category: "tablet",
		GenericName: "Paracetamol", Manufacturer: "Drayya Pharma",
		Prices: money.RegionalPricing{"INR": 200}, // ₹2.00 per strip
		IsActive: true,
	}
	require.NoError(t, db.Create(&paracetamol).Error)
	t.Logf("  ✓ Medicine created: %s (ID=%d)", paracetamol.Name, paracetamol.ID)

	// Create Bill of Materials
	bom := BillOfMaterials{
		MedicineID: paracetamol.ID,
		Version:    1,
		IsActive:   true,
		Notes:      "Standard formulation for Paracetamol 500mg tablets",
		Items: []BOMItem{
			{
				RawMaterialID: paracetamolAPI.ID, QtyPerUnit: 0.5, // 0.5g API per tablet
				UnitOfMeasure: "grams", WastagePercent: 2.0,
			},
			{
				RawMaterialID: excipientMCC.ID, QtyPerUnit: 0.15, // 0.15g excipient per tablet
				UnitOfMeasure: "grams", WastagePercent: 1.5,
			},
		},
	}
	require.NoError(t, mfgSvc.CreateBOM(ctx, &bom))
	assert.Greater(t, bom.ID, uint(0))
	t.Logf("  ✓ BOM created: ID=%d, Version=%d, Items=%d", bom.ID, bom.Version, len(bom.Items))
	t.Logf("    - %s: %.2fg/unit + %.1f%% wastage", paracetamolAPI.Name, 0.5, 2.0)
	t.Logf("    - %s: %.2fg/unit + %.1f%% wastage", excipientMCC.Name, 0.15, 1.5)

	// Validate BOM availability for 1000 tablets
	shortages, err := mfgSvc.ValidateBOMAvailability(ctx, bom.ID, 1000, centralHub.ID)
	require.NoError(t, err)
	assert.Empty(t, shortages, "Should have enough raw material for 1000 tablets")
	t.Log("  ✓ BOM availability check: PASS — enough material for 1000 tablets")

	// Validate BOM for 20000 tablets (should have shortage)
	shortages, err = mfgSvc.ValidateBOMAvailability(ctx, bom.ID, 20000, centralHub.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, shortages, "Should be short for 20000 tablets")
	for _, s := range shortages {
		t.Logf("  ⚠ Shortage detected: %s — need %.0f %s, have %.0f, short %.0f",
			s.RawMaterialName, s.RequiredQty, s.UnitOfMeasure, s.AvailableQty, s.ShortageQty)
	}

	// Create Production Order
	scheduledDate := time.Now().Add(24 * time.Hour)
	prodOrder := ProductionOrder{
		MedicineID:    paracetamol.ID,
		BOMID:         bom.ID,
		TargetQty:     1000, // 1000 tablets
		WarehouseID:   centralHub.ID,
		DestCityName:  "Mumbai",
		Priority:      3,
		ScheduledDate: &scheduledDate,
		CreatedByID:   1,
		Notes:         "Production batch for Mumbai distribution",
	}
	require.NoError(t, mfgSvc.CreateProductionOrder(ctx, &prodOrder))
	assert.NotEmpty(t, prodOrder.PONumber)
	assert.Equal(t, ProdStatusDraft, prodOrder.Status)
	t.Logf("  ✓ Production Order created: %s, Target=%d units, Status=%s",
		prodOrder.PONumber, prodOrder.TargetQty, prodOrder.Status)

	// Test invalid state transition
	err = mfgSvc.UpdateProductionStatus(ctx, prodOrder.ID, ProdStatusInProduction, "skip", 1)
	assert.Error(t, err, "Should not allow draft → in_production directly")
	t.Log("  ✓ State machine: draft → in_production blocked (correct)")

	// Schedule
	require.NoError(t, mfgSvc.UpdateProductionStatus(ctx, prodOrder.ID, ProdStatusScheduled, "Scheduled for tomorrow", 1))
	t.Log("  ✓ Production order: draft → scheduled")

	// Issue materials (FEFO from raw material inventory)
	issuances, err := mfgSvc.IssueMaterialsForProduction(ctx, prodOrder.ID, 1)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(issuances), 2, "Should issue at least 2 raw materials (API + excipient)")
	t.Logf("  ✓ Materials issued: %d lots from inventory", len(issuances))
	for _, iss := range issuances {
		t.Logf("    - RM #%d: %.2f %s issued from Inventory #%d",
			iss.RawMaterialID, iss.IssuedQty, iss.UnitOfMeasure, iss.InventoryID)
	}

	// Verify raw material deducted
	balanceAfter, err := rmSvc.GetRMBalance(ctx, paracetamolAPI.ID, centralHub.ID)
	require.NoError(t, err)
	assert.Less(t, balanceAfter, 5000.0, "Paracetamol stock should be reduced after issuance")
	t.Logf("  ✓ Paracetamol API balance: %.0f grams (was 5000)", balanceAfter)

	// Verify production order advanced to material_reserved
	updatedPO, err := mfgSvc.GetProductionOrder(ctx, prodOrder.ID)
	require.NoError(t, err)
	assert.Equal(t, ProdStatusMaterialReserved, updatedPO.Status)
	t.Log("  ✓ Production order: scheduled → material_reserved")

	// Start production
	require.NoError(t, mfgSvc.UpdateProductionStatus(ctx, prodOrder.ID, ProdStatusInProduction, "Production line started", 1))
	t.Log("  ✓ Production order: material_reserved → in_production")

	// Add manufacturing logs
	require.NoError(t, mfgSvc.AddManufacturingLog(ctx, &ManufacturingLog{
		ProductionOrderID: prodOrder.ID, Stage: "mixing",
		OperatorID: 1, Notes: "Dry mixing completed", StartedAt: time.Now(),
	}))
	require.NoError(t, mfgSvc.AddManufacturingLog(ctx, &ManufacturingLog{
		ProductionOrderID: prodOrder.ID, Stage: "compression",
		OperatorID: 1, Notes: "Tablet compression at 5kN force",
		StartedAt: time.Now(),
	}))
	logs, err := mfgSvc.GetManufacturingLogs(ctx, prodOrder.ID)
	require.NoError(t, err)
	t.Logf("  ✓ Manufacturing logs: %d entries (mixing, compression, ...)", len(logs))

	// Complete production → send to QA
	require.NoError(t, mfgSvc.UpdateProductionStatus(ctx, prodOrder.ID, ProdStatusQAPending, "Batch complete, sending to QA", 1))
	t.Log("  ✓ Production order: in_production → qa_pending")

	// QA Pass
	require.NoError(t, mfgSvc.UpdateProductionStatus(ctx, prodOrder.ID, ProdStatusQAPassed, "All tests passed: dissolution, hardness, friability", 2))
	t.Log("  ✓ Production order: qa_pending → qa_passed")

	// Release finished goods (980 tablets produced — 2% yield loss)
	actualYield := 980
	expiryDate := time.Now().AddDate(2, 0, 0) // 2-year shelf life
	require.NoError(t, mfgSvc.ReleaseFinishedGoods(ctx, prodOrder.ID, actualYield, expiryDate, 1))
	t.Logf("  ✓ Finished goods released: %d tablets (target: %d, yield: %.1f%%)",
		actualYield, prodOrder.TargetQty, float64(actualYield)/float64(prodOrder.TargetQty)*100)

	// Verify finished goods appear in PharmacyInventory
	var invCount int64
	db.Model(&PharmacyInventory{}).
		Where("pharmacy_id = ? AND medicine_id = ? AND stock_quantity > 0", centralHub.ID, paracetamol.ID).
		Count(&invCount)
	assert.Greater(t, invCount, int64(0), "Finished goods should be in PharmacyInventory")
	t.Log("  ✓ Verified: Finished goods present in PharmacyInventory")

	// Verify production order is now released
	releasedPO, err := mfgSvc.GetProductionOrder(ctx, prodOrder.ID)
	require.NoError(t, err)
	assert.Equal(t, ProdStatusReleased, releasedPO.Status)
	assert.NotNil(t, releasedPO.ActualYieldQty)
	assert.Equal(t, actualYield, *releasedPO.ActualYieldQty)
	assert.NotEmpty(t, releasedPO.BatchNumber)
	t.Logf("  ✓ Production order final: Status=%s, BatchNumber=%s", releasedPO.Status, releasedPO.BatchNumber)

	t.Log("")
	t.Log("╔══════════════════════════════════════════════════════════╗")
	t.Log("║  STEP 4: TRANSFER — Central Hub → Mumbai Hub           ║")
	t.Log("╚══════════════════════════════════════════════════════════╝")

	transfer := InventoryTransfer{
		SourceID:      centralHub.ID,
		DestinationID: mumbaiHub.ID,
		MedicineID:    paracetamol.ID,
		Quantity:      500, // Send 500 tablets to Mumbai
		Status:        "requested",
		RequestedByID: 1,
		CreatedAt:     time.Now(),
	}
	require.NoError(t, db.Create(&transfer).Error)
	t.Logf("  ✓ Transfer requested: %d units %s → %s", transfer.Quantity, centralHub.Name, mumbaiHub.Name)

	// Complete the transfer (source → destination)
	err = db.Transaction(func(tx *gorm.DB) error {
		// Deduct from source
		var sourceInv PharmacyInventory
		if err := tx.Where("pharmacy_id = ? AND medicine_id = ? AND stock_quantity >= ?",
			centralHub.ID, paracetamol.ID, transfer.Quantity).First(&sourceInv).Error; err != nil {
			return fmt.Errorf("source inventory not found: %w", err)
		}
		sourceInv.StockQuantity -= transfer.Quantity
		if err := tx.Save(&sourceInv).Error; err != nil {
			return err
		}

		// Add to destination
		destInv := PharmacyInventory{
			PharmacyID:    mumbaiHub.ID,
			MedicineID:    paracetamol.ID,
			BatchNumber:   sourceInv.BatchNumber,
			StockQuantity: transfer.Quantity,
			ExpiryDate:    sourceInv.ExpiryDate,
			IsActive:      true,
		}
		if err := tx.Create(&destInv).Error; err != nil {
			return err
		}

		transfer.Status = "completed"
		transfer.UpdatedAt = time.Now()
		return tx.Save(&transfer).Error
	})
	require.NoError(t, err)
	t.Log("  ✓ Transfer completed: 500 tablets now at Mumbai Hub")

	// Verify Mumbai inventory
	var mumbaiStock int
	db.Model(&PharmacyInventory{}).
		Select("COALESCE(SUM(stock_quantity), 0)").
		Where("pharmacy_id = ? AND medicine_id = ?", mumbaiHub.ID, paracetamol.ID).
		Scan(&mumbaiStock)
	assert.Equal(t, 500, mumbaiStock)
	t.Logf("  ✓ Mumbai Hub stock: %d tablets", mumbaiStock)

	t.Log("")
	t.Log("╔══════════════════════════════════════════════════════════╗")
	t.Log("║  STEP 5: CUSTOMER ORDER — Place + Pay + Fulfill        ║")
	t.Log("╚══════════════════════════════════════════════════════════╝")

	// Customer places order from Mumbai pharmacy
	orderItems := []map[string]interface{}{
		{"medicine_id": float64(paracetamol.ID), "quantity": 10},
	}
	orderItemsJSON, _ := json.Marshal(orderItems)

	customerOrder := PharmacyOrder{
		OrderNumber:     fmt.Sprintf("ORD-%s", strings.ToUpper(crypto.SecureToken(4))),
		UserID:          100, // Customer
		PharmacyID:      mumbaiHub.ID,
		Items:           orderItemsJSON,
		Amount:          2000, // 10 strips × ₹2.00 = ₹20.00 = 2000 paise
		Currency:        "INR",
		Status:          "pending",
		PaymentStatus:   "pending",
		PaymentMethod:   "wallet",
		DeliveryMode:    ModeStandard,
		ShippingAddress: "Plot 42, Andheri West, Mumbai 400053",
		DeliveryPhone:   "+91-9876500100",
		DestPincode:     "400053",
	}
	require.NoError(t, pharmacySvc.CreateOrder(ctx, &customerOrder))
	assert.Greater(t, customerOrder.ID, uint(0))
	t.Logf("  ✓ Customer order created: %s (Amount: ₹%.2f, Items: 10 strips)",
		customerOrder.OrderNumber, float64(customerOrder.Amount)/100)

	// Setup patient wallet with sufficient balance
	patientAcct, err := financeSvc.GetOrCreateAccount(ctx, fmt.Sprintf("PatientWallet:%d", customerOrder.UserID), finance.TypeLiability)
	require.NoError(t, err)
	// Credit the wallet manually for test
	db.Model(&finance.LedgerAccount{}).Where("id = ?", patientAcct.ID).Update("balance", int64(50000)) // ₹500
	t.Log("  ✓ Patient wallet funded: ₹500.00")

	// Pay the order
	require.NoError(t, pharmacySvc.PayOrder(ctx, customerOrder.ID))

	var paidOrder PharmacyOrder
	require.NoError(t, db.First(&paidOrder, customerOrder.ID).Error)
	assert.Equal(t, "paid", paidOrder.PaymentStatus)
	assert.Equal(t, "confirmed", paidOrder.Status)
	t.Logf("  ✓ Order paid: PaymentStatus=%s, Status=%s", paidOrder.PaymentStatus, paidOrder.Status)

	// Verify ledger entries created
	var txnCount int64
	db.Model(&finance.LedgerTransaction{}).Count(&txnCount)
	assert.Greater(t, txnCount, int64(0), "Ledger transactions should exist")
	t.Logf("  ✓ Ledger transactions recorded: %d", txnCount)

	t.Log("")
	t.Log("╔══════════════════════════════════════════════════════════╗")
	t.Log("║  STEP 6: PACK — Reserve Inventory (FEFO)               ║")
	t.Log("╚══════════════════════════════════════════════════════════╝")

	// Reserve inventory (FEFO) for the order
	require.NoError(t, pharmacySvc.ReserveInventory(ctx, customerOrder.ID))
	t.Log("  ✓ Inventory reserved (FEFO) for order")

	// Update status to packed
	require.NoError(t, pharmacySvc.UpdateOrderStatus(ctx, customerOrder.ID, "packed"))
	var packedOrder PharmacyOrder
	db.First(&packedOrder, customerOrder.ID)
	assert.Equal(t, "packed", packedOrder.Status)
	t.Logf("  ✓ Order packed: Status=%s", packedOrder.Status)

	// Verify stock decremented
	var stockAfterPack int
	db.Model(&PharmacyInventory{}).
		Select("COALESCE(SUM(stock_quantity), 0)").
		Where("pharmacy_id = ? AND medicine_id = ?", mumbaiHub.ID, paracetamol.ID).
		Scan(&stockAfterPack)
	assert.Equal(t, 490, stockAfterPack, "Stock should be 500-10=490 after reservation")
	t.Logf("  ✓ Mumbai stock after pack: %d (was 500, reserved 10)", stockAfterPack)

	t.Log("")
	t.Log("╔══════════════════════════════════════════════════════════╗")
	t.Log("║  STEP 7: DISPENSE — Batch Traceability                 ║")
	t.Log("╚══════════════════════════════════════════════════════════╝")

	// Find the inventory batch to dispense from
	var dispBatch PharmacyInventory
	require.NoError(t, db.Where("pharmacy_id = ? AND medicine_id = ?", mumbaiHub.ID, paracetamol.ID).
		First(&dispBatch).Error)

	// Create dispense record
	dispenseRec := DispenseRecord{
		OrderID:          customerOrder.ID,
		OrderItemID:      1, // First item
		InventoryBatchID: dispBatch.ID,
		MedicineID:       paracetamol.ID,
		Quantity:         10,
		DispensedByID:    2, // Pharmacist
		DispensedAt:      time.Now(),
		LabelPrinted:     true,
	}
	require.NoError(t, db.Create(&dispenseRec).Error)
	t.Logf("  ✓ Dispense record created: Batch #%s, Qty=%d, Pharmacist=#%d",
		dispBatch.BatchNumber, dispenseRec.Quantity, dispenseRec.DispensedByID)

	// Advance to dispensed
	require.NoError(t, pharmacySvc.UpdateOrderStatus(ctx, customerOrder.ID, "dispensed"))
	var dispensedOrder PharmacyOrder
	db.First(&dispensedOrder, customerOrder.ID)
	assert.Equal(t, "dispensed", dispensedOrder.Status)
	t.Log("  ✓ Order dispensed: Status=dispensed")

	t.Log("")
	t.Log("╔══════════════════════════════════════════════════════════╗")
	t.Log("║  STEP 8: SHIP + DELIVER — Final Mile                   ║")
	t.Log("╚══════════════════════════════════════════════════════════╝")

	// Ship
	require.NoError(t, pharmacySvc.UpdateOrderStatus(ctx, customerOrder.ID, "shipped"))
	var shippedOrder PharmacyOrder
	db.First(&shippedOrder, customerOrder.ID)
	assert.Equal(t, "shipped", shippedOrder.Status)
	t.Log("  ✓ Order shipped: Status=shipped")

	// Create shipment tracking
	shipment := Shipment{
		TrackingNumber: fmt.Sprintf("SHIP-%s", strings.ToUpper(crypto.SecureToken(4))),
		OrderID:        customerOrder.ID,
		OrderType:      "pharmacy",
		ProviderName:   "Delhivery",
		Status:         "in_transit",
		Origin:         mumbaiHub.Name,
		Destination:    string(customerOrder.ShippingAddress),
	}
	require.NoError(t, db.Create(&shipment).Error)
	t.Logf("  ✓ Shipment created: %s via %s", shipment.TrackingNumber, shipment.ProviderName)

	// Deliver
	require.NoError(t, pharmacySvc.UpdateOrderStatus(ctx, customerOrder.ID, "delivered"))
	var deliveredOrder PharmacyOrder
	db.First(&deliveredOrder, customerOrder.ID)
	assert.Equal(t, "delivered", deliveredOrder.Status)
	t.Log("  ✓ Order delivered: Status=delivered")

	t.Log("")
	t.Log("╔══════════════════════════════════════════════════════════╗")
	t.Log("║  STEP 9: GRN (Goods Receipt Note) Workflow             ║")
	t.Log("╚══════════════════════════════════════════════════════════╝")

	grn := GoodsReceiptNote{
		WarehouseID:  centralHub.ID,
		SourceType:   "supplier",
		SupplierID:   &supplier.ID,
		ReceivedByID: 1,
		VehicleNumber: "KA-01-AB-1234",
		InvoiceNumber: "INV-2026-00567",
		InvoiceAmount: 50000,
		Items: []GRNItem{
			{
				ItemType: "raw_material", ItemID: paracetamolAPI.ID, ItemName: "Paracetamol API",
				OrderedQty: 10000, ReceivedQty: 9800, UnitOfMeasure: "grams",
				BatchNumber: "BATCH-PC-2026-002",
			},
		},
	}
	require.NoError(t, mfgSvc.CreateGRN(ctx, &grn))
	assert.NotEmpty(t, grn.GRNNumber)
	assert.Equal(t, "pending_inspection", grn.Status)
	t.Logf("  ✓ GRN created: %s (Vehicle: %s, Invoice: %s)", grn.GRNNumber, grn.VehicleNumber, grn.InvoiceNumber)
	t.Logf("    - Paracetamol API: ordered=%d, received=%d grams", 10000, 9800)

	// QA the GRN items
	require.NoError(t, mfgSvc.ProcessGRNItemQA(ctx, grn.Items[0].ID, "passed", "Purity 99.6% — within spec"))
	t.Log("  ✓ GRN Item QA: PASSED")

	// Verify GRN item updated
	var grnItem GRNItem
	db.First(&grnItem, grn.Items[0].ID)
	assert.Equal(t, "passed", grnItem.QAStatus)
	assert.Equal(t, 9800.0, grnItem.AcceptedQty)
	t.Logf("  ✓ GRN Item: AcceptedQty=%.0f, QAStatus=%s", grnItem.AcceptedQty, grnItem.QAStatus)

	t.Log("")
	t.Log("╔══════════════════════════════════════════════════════════╗")
	t.Log("║  STEP 10: RAW MATERIAL INDENT → RFQ WORKFLOW           ║")
	t.Log("╚══════════════════════════════════════════════════════════╝")

	// Create indent (demand signal)
	indent := RawMaterialIndent{
		RawMaterialID:  paracetamolAPI.ID,
		RequiredQty:    50000, // 50kg
		UnitOfMeasure:  "grams",
		RequiredByDate: time.Now().AddDate(0, 0, 14), // 2 weeks
		WarehouseID:    centralHub.ID,
		Priority:       3,
		RequestedByID:  1,
		Notes:          "Needed for next month's production plan",
	}
	require.NoError(t, rmSvc.CreateIndent(ctx, &indent))
	assert.NotEmpty(t, indent.IndentNumber)
	assert.Equal(t, "pending", indent.Status)
	t.Logf("  ✓ Indent created: %s — %.0fg %s, Priority=%d",
		indent.IndentNumber, indent.RequiredQty, paracetamolAPI.Name, indent.Priority)

	// Approve indent
	require.NoError(t, rmSvc.ApproveIndent(ctx, indent.ID, 1))
	approvedIndent, _ := rmSvc.GetIndent(ctx, indent.ID)
	assert.Equal(t, "approved", approvedIndent.Status)
	t.Log("  ✓ Indent approved by procurement manager")

	// Create RFQ
	rfq := RFQ{
		RawMaterialID: paracetamolAPI.ID,
		IndentID:      &indent.ID,
		RequiredQty:   50000,
		UnitOfMeasure: "grams",
		Deadline:      time.Now().AddDate(0, 0, 3), // 3-day response window
		CreatedByID:   1,
		Notes:         "Request for 50kg Paracetamol API — IP grade",
	}
	require.NoError(t, rmSvc.CreateRFQ(ctx, &rfq))
	assert.NotEmpty(t, rfq.RFQNumber)
	assert.Equal(t, "open", rfq.Status)
	t.Logf("  ✓ RFQ created: %s (Deadline: %s)", rfq.RFQNumber, rfq.Deadline.Format("2006-01-02"))

	// Submit vendor quotations
	q1 := VendorQuotation{
		RFQID: rfq.ID, SupplierID: supplier.ID,
		UnitPrice: 38000, Currency: "INR", // ₹380/kg
		LeadTimeDays: 7, MinOrderQty: 10000,
		ValidUntil: time.Now().AddDate(0, 0, 30),
	}
	require.NoError(t, rmSvc.SubmitQuotation(ctx, &q1))
	t.Logf("  ✓ Quotation from Supplier #%d: ₹%.2f/kg, Lead=%dd", supplier.ID, float64(q1.UnitPrice)/100, q1.LeadTimeDays)

	// Award RFQ
	require.NoError(t, rmSvc.AwardRFQ(ctx, rfq.ID, q1.ID))
	awardedRFQ, _ := rmSvc.GetRFQ(ctx, rfq.ID)
	assert.Equal(t, "awarded", awardedRFQ.Status)
	t.Log("  ✓ RFQ awarded to selected vendor")

	// Verify quotation marked as selected
	var selectedQ VendorQuotation
	db.First(&selectedQ, q1.ID)
	assert.True(t, selectedQ.IsSelected)
	t.Log("  ✓ Winning quotation marked as selected")

	t.Log("")
	t.Log("╔══════════════════════════════════════════════════════════╗")
	t.Log("║  STEP 11: MULTI-CITY DASHBOARD                         ║")
	t.Log("╚══════════════════════════════════════════════════════════╝")

	// Get Bangalore dashboard
	dashboard, err := mfgSvc.GetCityWorkflowDashboard(ctx, "Bangalore")
	require.NoError(t, err)
	assert.Equal(t, "Bangalore", dashboard.CityName)
	assert.Equal(t, centralHub.ID, dashboard.WarehouseID)
	t.Logf("  ✓ Bangalore Dashboard:")
	t.Logf("    Manufacturing: active=%d, qa_review=%d, planned_week=%d, released_today=%d",
		dashboard.Manufacturing.ActiveOrders, dashboard.Manufacturing.InQAReview,
		dashboard.Manufacturing.PlannedThisWeek, dashboard.Manufacturing.ReleasedToday)
	t.Logf("    Procurement: pending_pos=%d, awaiting_grn=%d, reorder_alerts=%d",
		dashboard.Procurement.PendingPOs, dashboard.Procurement.AwaitingGRN, dashboard.Procurement.ReorderAlerts)
	t.Logf("    Raw Materials: open_indents=%d, rfqs_pending=%d, in_qa=%d",
		dashboard.RawMaterials.OpenIndents, dashboard.RawMaterials.RFQsPending, dashboard.RawMaterials.InQALab)
	t.Logf("    Inventory: total_skus=%d, low_stock=%d, cold_chain_ok=%v",
		dashboard.InventoryStats.TotalSKUs, dashboard.InventoryStats.LowStockItems, dashboard.InventoryStats.ColdChainOK)

	t.Log("")
	t.Log("╔══════════════════════════════════════════════════════════╗")
	t.Log("║  STEP 12: CANCEL / RESTORE FLOW                        ║")
	t.Log("╚══════════════════════════════════════════════════════════╝")

	// Create a second order and cancel it to test restore
	orderItems2 := []map[string]interface{}{
		{"medicine_id": float64(paracetamol.ID), "quantity": 5},
	}
	orderItemsJSON2, _ := json.Marshal(orderItems2)
	cancelOrder := PharmacyOrder{
		OrderNumber:   fmt.Sprintf("ORD-CANCEL-%s", strings.ToUpper(crypto.SecureToken(4))),
		UserID:        200,
		PharmacyID:    mumbaiHub.ID,
		Items:         orderItemsJSON2,
		Amount:        1000,
		Currency:      "INR",
		Status:        "confirmed",
		PaymentStatus: "paid",
		DeliveryMode:  ModeStandard,
	}
	require.NoError(t, db.Create(&cancelOrder).Error)

	// Reserve inventory
	require.NoError(t, pharmacySvc.ReserveInventory(ctx, cancelOrder.ID))
	var stockBeforeCancel int
	db.Model(&PharmacyInventory{}).
		Select("COALESCE(SUM(stock_quantity), 0)").
		Where("pharmacy_id = ? AND medicine_id = ?", mumbaiHub.ID, paracetamol.ID).
		Scan(&stockBeforeCancel)
	t.Logf("  Mumbai stock before cancel: %d", stockBeforeCancel)

	// Pack the order
	require.NoError(t, pharmacySvc.UpdateOrderStatus(ctx, cancelOrder.ID, "packed"))

	// Cancel it — should restore inventory
	require.NoError(t, pharmacySvc.RestoreInventory(ctx, cancelOrder.ID))
	require.NoError(t, pharmacySvc.UpdateOrderStatus(ctx, cancelOrder.ID, "cancelled"))

	var stockAfterCancel int
	db.Model(&PharmacyInventory{}).
		Select("COALESCE(SUM(stock_quantity), 0)").
		Where("pharmacy_id = ? AND medicine_id = ?", mumbaiHub.ID, paracetamol.ID).
		Scan(&stockAfterCancel)
	assert.Equal(t, stockBeforeCancel+5, stockAfterCancel, "Stock should be restored after cancel")
	t.Logf("  ✓ Cancel + Restore: stock %d → %d (+5 restored)", stockBeforeCancel, stockAfterCancel)

	t.Log("")
	t.Log("╔══════════════════════════════════════════════════════════╗")
	t.Log("║  STEP 13: RAW MATERIAL DESTRUCTION                     ║")
	t.Log("╚══════════════════════════════════════════════════════════╝")

	// Test QA rejection + destruction flow
	badInv := RawMaterialInventory{
		RawMaterialID: paracetamolAPI.ID, WarehouseID: centralHub.ID,
		Quantity: 500, UnitOfMeasure: "grams", SupplierID: supplier.ID,
		BatchNumber: "BATCH-BAD-001", ExpiryDate: time.Now().AddDate(1, 0, 0),
		CostPerUnit: 40, ReorderLevel: 100,
	}
	require.NoError(t, rmSvc.ReceiveRawMaterial(ctx, &badInv, 1))
	require.NoError(t, rmSvc.RejectQA(ctx, badInv.ID, 2, "Purity 92.1% — BELOW SPEC (min 99.0%)"))
	t.Log("  ✓ Batch received and QA REJECTED")

	// Destroy the rejected material
	require.NoError(t, rmSvc.DestroyRawMaterial(ctx, badInv.ID, 500, "Environmental disposal per SOP-D-001", 2))
	t.Log("  ✓ Material destroyed: 500g disposed")

	// Verify destruction audit trail
	destroyTxns, err := rmSvc.GetRMTransactions(ctx, badInv.ID)
	require.NoError(t, err)
	hasDestruction := false
	for _, tx := range destroyTxns {
		if tx.TransactionType == "destruction" {
			hasDestruction = true
			assert.Equal(t, -500.0, tx.Quantity)
			assert.Equal(t, 0.0, tx.BalanceAfter)
		}
	}
	assert.True(t, hasDestruction, "Should have destruction transaction")
	t.Logf("  ✓ Destruction audit trail: %d transactions, final balance=0", len(destroyTxns))

	t.Log("")
	t.Log("╔══════════════════════════════════════════════════════════╗")
	t.Log("║  WORKFLOW COMPLETE — ALL STEPS PASSED                   ║")
	t.Log("╚══════════════════════════════════════════════════════════╝")
	t.Log("")
	t.Log("  Full cycle validated:")
	t.Log("  1. ✓ Setup: Cities, warehouses, suppliers")
	t.Log("  2. ✓ Raw Material: Receive → Quarantine → QA Approve")
	t.Log("  3. ✓ Manufacturing: BOM → Production Order → Issue Materials → QA → Release")
	t.Log("  4. ✓ Transfer: Central Hub → Mumbai Hub (inter-city)")
	t.Log("  5. ✓ Customer Order: Place → Pay (wallet + ledger)")
	t.Log("  6. ✓ Pack: FEFO inventory reservation")
	t.Log("  7. ✓ Dispense: Batch traceability records")
	t.Log("  8. ✓ Ship + Deliver: Shipment tracking → delivered")
	t.Log("  9. ✓ GRN: Goods receipt + QA inspection")
	t.Log("  10. ✓ Indent + RFQ: Demand signal → quotation → award")
	t.Log("  11. ✓ Multi-City Dashboard: Aggregated stats per city")
	t.Log("  12. ✓ Cancel + Restore: Inventory recovery on cancellation")
	t.Log("  13. ✓ Destruction: QA reject → disposal with audit trail")
}

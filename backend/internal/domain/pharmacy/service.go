package pharmacy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"drayya-core/internal/crypto"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/loyalty"
	"drayya-core/internal/utils/privacy"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Service interface {
	GetMedicines(ctx context.Context) ([]Medicine, error)
	GetMedicine(ctx context.Context, id uint) (*Medicine, error)
	CreateMedicine(ctx context.Context, med *Medicine) error
	GetInventory(ctx context.Context, pharmacyID uint) ([]PharmacyInventory, error)
	GetLowStockInventory(ctx context.Context) ([]PharmacyInventory, error)
	GetExpiringInventory(ctx context.Context) ([]PharmacyInventory, error)
	GetStockAlerts(ctx context.Context) (interface{}, error)
	GetMultiCityInventory(ctx context.Context) ([]CityStock, error)

	// Pharmacy/Branch Management
	CreatePharmacy(ctx context.Context, p *Pharmacy) error
	UpdatePharmacy(ctx context.Context, p *Pharmacy) error
	TogglePharmacy(ctx context.Context, id uint) (*Pharmacy, error)
	GetPharmacies(ctx context.Context) ([]Pharmacy, error)
	GetAllPharmacies(ctx context.Context, onlyActive bool) ([]Pharmacy, error)
	GetPharmacyByID(ctx context.Context, id uint) (*Pharmacy, error)
	ListAllPharmaciesAdmin(ctx context.Context) ([]Pharmacy, error)

	// Order Management
	CreateOrder(ctx context.Context, order *PharmacyOrder) error
	GetOrder(ctx context.Context, id uint) (*PharmacyOrder, error)
	GetPharmacyOrders(ctx context.Context, pharmacyID uint) ([]PharmacyOrder, error)
	UpdateOrderStatus(ctx context.Context, id uint, status string) error
	PayOrder(ctx context.Context, orderID uint) error
	ReserveInventory(ctx context.Context, orderID uint) error
	RestoreInventory(ctx context.Context, orderID uint) error
	ConvertPrescriptionToOrder(ctx context.Context, prescriptionID uint, pharmacyID uint, address string, phone string, deliveryMode DeliveryMode) (*PharmacyOrder, error)
	GenerateCartFromPrescription(ctx context.Context, prescriptionID uint) ([]interface{}, error)

	// Procurement & Supply Chain
	CreatePurchaseOrder(ctx context.Context, supplierID uint, pharmacyID uint, items []map[string]interface{}) (*PurchaseOrder, error)
	GetPurchaseOrder(ctx context.Context, poID uint) (*PurchaseOrder, error)
	ReceivePurchaseOrder(ctx context.Context, poID uint) error
	SettleSupplierPayment(ctx context.Context, poID uint) error
	CheckAndTriggerRestock(ctx context.Context) (int, error)

	// Analytics & Metrics
	GetPharmacyFinancials(ctx context.Context, pharmacyID uint) (*PharmacyFinancials, error)
	RecordWarehouseMetric(ctx context.Context, warehouseID uint, temp float64, humidity float64) error
	GetWarehouseMetrics(ctx context.Context, warehouseID uint) ([]ColdChainMetric, error)
	GetDashboardStats(ctx context.Context) (map[string]interface{}, error)
}

type service struct {
	repo       Repository
	financeSvc finance.Service
}

func NewService(repo Repository, financeSvc finance.Service) Service {
	return &service{
		repo:       repo,
		financeSvc: financeSvc,
	}
}

func (s *service) GetMedicines(ctx context.Context) ([]Medicine, error) {
	return s.repo.GetMedicines(ctx)
}

func (s *service) GetMedicine(ctx context.Context, id uint) (*Medicine, error) {
	return s.repo.GetMedicine(ctx, id)
}

func (s *service) CreateMedicine(ctx context.Context, med *Medicine) error {
	return s.repo.CreateMedicine(ctx, med)
}

func (s *service) GetInventory(ctx context.Context, pharmacyID uint) ([]PharmacyInventory, error) {
	return s.repo.GetInventory(ctx, pharmacyID)
}

func (s *service) GetLowStockInventory(ctx context.Context) ([]PharmacyInventory, error) {
	return s.repo.GetLowStockInventory(ctx)
}

func (s *service) GetExpiringInventory(ctx context.Context) ([]PharmacyInventory, error) {
	threshold := time.Now().AddDate(0, 3, 0) // 3 months
	return s.repo.GetExpiringInventory(ctx, threshold)
}

func (s *service) GetStockAlerts(ctx context.Context) (interface{}, error) {
	lowStock, err := s.repo.GetLowStockInventory(ctx)
	if err != nil {
		return nil, err
	}
	expiringSoon, err := s.repo.GetExpiringInventory(ctx, time.Now().AddDate(0, 1, 0)) // within 1 month
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"low_stock":     lowStock,
		"expiring_soon": expiringSoon,
	}, nil
}

func (s *service) GetMultiCityInventory(ctx context.Context) ([]CityStock, error) {
	return s.repo.GetMultiCityInventory(ctx)
}

func (s *service) CreatePharmacy(ctx context.Context, p *Pharmacy) error {
	return s.repo.CreatePharmacy(ctx, p)
}

func (s *service) UpdatePharmacy(ctx context.Context, p *Pharmacy) error {
	return s.repo.UpdatePharmacy(ctx, p)
}

func (s *service) TogglePharmacy(ctx context.Context, id uint) (*Pharmacy, error) {
	p, err := s.repo.GetPharmacyByID(ctx, id)
	if err != nil {
		return nil, err
	}
	p.IsActive = !p.IsActive
	err = s.repo.UpdatePharmacy(ctx, p)
	return p, err
}

func (s *service) GetPharmacies(ctx context.Context) ([]Pharmacy, error) {
	return s.repo.GetAllPharmacies(ctx, true)
}

func (s *service) GetAllPharmacies(ctx context.Context, onlyActive bool) ([]Pharmacy, error) {
	return s.repo.GetAllPharmacies(ctx, onlyActive)
}

func (s *service) GetPharmacyByID(ctx context.Context, id uint) (*Pharmacy, error) {
	return s.repo.GetPharmacyByID(ctx, id)
}

func (s *service) ListAllPharmaciesAdmin(ctx context.Context) ([]Pharmacy, error) {
	return s.repo.GetAllPharmacies(ctx, false)
}

func (s *service) CreateOrder(ctx context.Context, order *PharmacyOrder) error {
	return s.repo.CreateOrder(ctx, order)
}

func (s *service) GetOrder(ctx context.Context, id uint) (*PharmacyOrder, error) {
	return s.repo.GetOrder(ctx, id)
}

func (s *service) GetPharmacyOrders(ctx context.Context, pharmacyID uint) ([]PharmacyOrder, error) {
	var orders []PharmacyOrder
	err := s.repo.GetDB(ctx).
		Where("pharmacy_id = ?", pharmacyID).
		Order("created_at desc").
		Find(&orders).Error
	return orders, err
}

func (s *service) UpdateOrderStatus(ctx context.Context, id uint, status string) error {
	order, err := s.repo.GetOrder(ctx, id)
	if err != nil {
		return err
	}
	order.Status = status
	return s.repo.UpdateOrder(ctx, order)
}

func (s *service) ReserveInventory(ctx context.Context, orderID uint) error {
	order, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}

	var items []struct {
		MedicineID uint `json:"medicine_id"`
		Quantity   int  `json:"quantity"`
	}
	if err := json.Unmarshal(order.Items, &items); err != nil {
		return fmt.Errorf("reserve: cannot parse order items: %w", err)
	}

	db := s.repo.GetDB(ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			remaining := item.Quantity
			// Fetch FEFO-ordered batches (earliest expiry first)
			var batches []PharmacyInventory
			if err := tx.
				Where("pharmacy_id = ? AND medicine_id = ? AND stock_quantity > 0 AND is_active = ?",
					order.PharmacyID, item.MedicineID, true).
				Order("expiry_date ASC").
				Find(&batches).Error; err != nil || len(batches) == 0 {
				return fmt.Errorf("no stock available for medicine ID %d", item.MedicineID)
			}
			for _, batch := range batches {
				if remaining <= 0 {
					break
				}
				deduct := remaining
				if batch.StockQuantity < deduct {
					deduct = batch.StockQuantity
				}
				// Atomic: UPDATE WHERE stock_quantity >= deduct
				result := tx.Exec(
					"UPDATE pharmacyinventory SET stock_quantity = stock_quantity - ? WHERE id = ? AND stock_quantity >= ?",
					deduct, batch.ID, deduct,
				)
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected == 0 {
					return fmt.Errorf("concurrent stock change on batch %d — please retry", batch.ID)
				}
				remaining -= deduct
			}
			if remaining > 0 {
				return fmt.Errorf("insufficient stock for medicine ID %d (short by %d units)", item.MedicineID, remaining)
			}
		}
		return nil
	})
}

// RestoreInventory returns reserved stock to inventory when an order is cancelled.
// It does a best-effort restore — adds back to the first active batch per medicine.
func (s *service) RestoreInventory(ctx context.Context, orderID uint) error {
	order, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}

	var items []struct {
		MedicineID uint `json:"medicine_id"`
		Quantity   int  `json:"quantity"`
	}
	if err := json.Unmarshal(order.Items, &items); err != nil {
		return nil // cannot parse — nothing to restore
	}

	db := s.repo.GetDB(ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			// Add back to any active batch for this pharmacy + medicine
			var batch PharmacyInventory
			if err := tx.
				Where("pharmacy_id = ? AND medicine_id = ? AND is_active = ?",
					order.PharmacyID, item.MedicineID, true).
				First(&batch).Error; err != nil {
				continue // no batch found — skip silently (may have been disposed)
			}
			tx.Exec(
				"UPDATE pharmacyinventory SET stock_quantity = stock_quantity + ? WHERE id = ?",
				item.Quantity, batch.ID,
			)
		}
		return nil
	})
}

// ConvertPrescriptionToOrder converts a prescription to a pharmacy order.
// H-8: Rx status must be "verified" before conversion.
func (s *service) ConvertPrescriptionToOrder(ctx context.Context, prescriptionID uint, pharmacyID uint, address string, phone string, deliveryMode DeliveryMode) (*PharmacyOrder, error) {
	rx, err := s.repo.GetRxRecord(ctx, prescriptionID)
	if err != nil {
		return nil, errors.New("prescription not found")
	}
	// H-8: require verified status
	if rx.Status != "verified" {
		return nil, fmt.Errorf("prescription must be verified before it can be used")
	}

	// M-25: use crypto.SecureToken to avoid millisecond collisions
	orderNum := fmt.Sprintf("RX-%s", strings.ToUpper(crypto.SecureToken(4)))

	order := &PharmacyOrder{
		OrderNumber:     orderNum,
		UserID:          rx.UserID,
		PharmacyID:      pharmacyID,
		Items:           datatypes.JSON("[]"), // pharmacist will add items during verification
		Amount:     0,
		Status:          "pending_rx_verification",
		PaymentStatus:   "pending",
		DeliveryMode:    deliveryMode,
		ShippingAddress: privacy.EncryptedString(address),
		DeliveryPhone:   privacy.EncryptedString(phone),
	}

	if err := s.repo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

// CartLine is one auto-generated cart item, priced and stock-checked, ready for
// the patient to review or the pharmacist to confirm.
type CartLine struct {
	PrescriptionItemID uint    `json:"prescription_item_id"`
	MedicineID         uint    `json:"medicine_id"`
	Name               string  `json:"name"`
	NameLocal          string  `json:"name_local,omitempty"` // Tamil/Hindi
	Form               string  `json:"form,omitempty"`
	PackSize           string  `json:"pack_size,omitempty"`
	Quantity           float64 `json:"quantity"`
	UnitPricePaise     int64   `json:"unit_price_paise"`
	LineTotalPaise     int64   `json:"line_total_paise"`
	GSTPercent         float64 `json:"gst_percent"`
	InStock            bool    `json:"in_stock"`
	StockAvailable     int64   `json:"stock_available"`
	SubstitutionNote   string  `json:"substitution_note,omitempty"`
}

func (s *service) GenerateCartFromPrescription(ctx context.Context, prescriptionID uint) ([]interface{}, error) {
	// Read structured PrescriptionItem rows via raw query — the pharmacy domain
	// mustn't import emr (cycle). Old free-form-JSON prescriptions have no rows
	// here, so we fall back to the legacy "pharmacist enters manually" path.
	type row struct {
		ID                    uint
		MedicineID            uint
		TotalQuantity         float64
		SubstitutedMedicineID *uint
	}
	var items []row
	if err := database.DB.Table("prescription_items").
		Select("id, medicine_id, total_quantity, substituted_medicine_id").
		Where("prescription_id = ? AND deleted_at IS NULL", prescriptionID).
		Scan(&items).Error; err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return []interface{}{}, nil
	}

	out := make([]interface{}, 0, len(items))
	for _, it := range items {
		medID := it.MedicineID
		if it.SubstitutedMedicineID != nil && *it.SubstitutedMedicineID != 0 {
			medID = *it.SubstitutedMedicineID
		}
		var med Medicine
		if err := database.DB.First(&med, medID).Error; err != nil {
			continue // skip rather than fail the whole cart on a missing SKU
		}

		qty := it.TotalQuantity
		if qty <= 0 {
			qty = 1
		}
		// RegionalPricing is a map[currency]paise.
		unitPrice := med.Prices["INR"]

		// Stock = sum across inventory rows (multiple batches possible per SKU,
		// only count active ones; expired batches stay in the table for audit).
		var available int64
		database.DB.Table("pharmacyinventory").
			Where("medicine_id = ? AND is_active = ?", medID, true).
			Select("COALESCE(SUM(stock_quantity), 0)").Scan(&available)

		// Pick whichever localised name is set; fall back to English.
		nameLocal := med.NameTa
		if nameLocal == "" {
			nameLocal = med.NameHi
		}

		note := ""
		if it.SubstitutedMedicineID != nil && *it.SubstitutedMedicineID != it.MedicineID {
			note = "Substituted by pharmacist"
		}

		out = append(out, CartLine{
			PrescriptionItemID: it.ID,
			MedicineID:         medID,
			Name:               med.Name,
			NameLocal:          nameLocal,
			Form:               med.Form,
			PackSize:           med.PackSize,
			Quantity:           qty,
			UnitPricePaise:     unitPrice,
			LineTotalPaise:     int64(qty * float64(unitPrice)),
			GSTPercent:         med.GSTPercent,
			InStock:            available >= int64(qty),
			StockAvailable:     available,
			SubstitutionNote:   note,
		})
	}
	return out, nil
}

func (s *service) PayOrder(ctx context.Context, orderID uint) error {
	order, err := s.repo.GetOrderWithPharmacy(ctx, orderID)
	if err != nil {
		return err
	}

	if order.PaymentStatus == "paid" {
		return errors.New("order already paid")
	}

	// Avoid recording empty transactions (fails DB constraint amount > 0)
	if order.Amount <= 0 {
		order.PaymentStatus = "paid"
		order.Status = "confirmed"
		return s.repo.UpdateOrder(ctx, order)
	}

	// Fetch accounts
	patientAcct, err := s.financeSvc.GetOrCreateAccount(ctx, fmt.Sprintf("PatientWallet:%d", order.UserID), finance.TypeLiability)
	if err != nil {
		return fmt.Errorf("failed to fetch patient wallet: %w", err)
	}

	// Wallet balance check — reject if insufficient funds
	if patientAcct.Balance < order.Amount {
		return fmt.Errorf("insufficient wallet balance: have %d paise, need %d paise", patientAcct.Balance, order.Amount)
	}

	platformAcct, err := s.financeSvc.GetOrCreateAccount(ctx, finance.RevPharmacy, finance.TypeRevenue)
	if err != nil {
		return fmt.Errorf("failed to fetch platform account: %w", err)
	}

	gstAcct, err := s.financeSvc.GetOrCreateAccount(ctx, finance.LiabGST, finance.TypeLiability)
	if err != nil {
		return fmt.Errorf("failed to fetch GST account: %w", err)
	}

	// Pharmacy's share. For a platform-owned central warehouse it IS platform
	// revenue; for an independent pharmacy it's a payable (liability) we owe the
	// merchant — NOT revenue. This mirrors the Razorpay path exactly.
	merchantAcctName := finance.MerchantWallet(order.PharmacyID)
	merchantAcctType := finance.TypeLiability
	if order.Pharmacy.IsCentralWarehouse {
		merchantAcctName = finance.RevCentralWarehouse
		merchantAcctType = finance.TypeRevenue
	}
	merchantAcct, err := s.financeSvc.GetOrCreateAccount(ctx, merchantAcctName, merchantAcctType)
	if err != nil {
		return fmt.Errorf("failed to fetch merchant account: %w", err)
	}

	// Unified split: 10% platform + 5% GST + 85% pharmacy (same as Razorpay path)
	totalPaise := order.Amount
	platformCommission := (totalPaise * 10) / 100
	gst := (totalPaise * 5) / 100
	merchantShare := totalPaise - platformCommission - gst

	entries := []finance.EntryData{
		{AccountID: patientAcct.ID, Direction: finance.DirectionDebit, Amount: totalPaise},
		{AccountID: platformAcct.ID, Direction: finance.DirectionCredit, Amount: platformCommission},
		{AccountID: gstAcct.ID, Direction: finance.DirectionCredit, Amount: gst},
		{AccountID: merchantAcct.ID, Direction: finance.DirectionCredit, Amount: merchantShare},
	}

	// Record ledger + update order atomically inside a single transaction
	err = s.repo.ExecuteTransaction(ctx, func(txCtx context.Context, txRepo Repository) error {
		txn, txErr := s.financeSvc.RecordTransaction(txCtx,
			fmt.Sprintf("PAY-ORD-%d", order.ID), "PAYMENT",
			fmt.Sprintf("Order payment for #%s", order.OrderNumber), entries)
		if txErr != nil {
			return fmt.Errorf("ledger recording failed: %w", txErr)
		}

		order.PaymentStatus = "paid"
		order.Status = "confirmed"
		order.LedgerTransactionID = &txn.ID
		if updateErr := txRepo.UpdateOrder(txCtx, order); updateErr != nil {
			return fmt.Errorf("order status update failed: %w", updateErr)
		}

		// Broadcast new order event (non-fatal if Redis is down)
		if database.Rdb != nil {
			event := map[string]interface{}{
				"type":    "new_order_received",
				"room_id": fmt.Sprintf("room:pharmacy:%d", order.PharmacyID),
				"metadata": map[string]interface{}{
					"order_id":     order.ID,
					"order_number": order.OrderNumber,
					"total_amount": order.Amount,
				},
			}
			if data, marshalErr := json.Marshal(event); marshalErr == nil {
				database.Rdb.Publish(txCtx, "chat:global", data)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Award loyalty points (best-effort; never blocks payment).
	_, _ = loyalty.EarnPoints(order.UserID, order.Amount, "medicine", order.ID)

	return nil
}

type PharmacyFinancials struct {
	TotalSales       int64 `json:"total_sales"`
	PlatformRoyalty  int64 `json:"platform_royalty"`
	MerchantEarnings int64 `json:"merchant_earnings"`
	OrderCount       int64 `json:"order_count"`
	PendingPayments  int64 `json:"pending_payments"`
}

func (s *service) GetPharmacyFinancials(ctx context.Context, pharmacyID uint) (*PharmacyFinancials, error) {
	var stats PharmacyFinancials

	paidOrders, err := s.repo.GetPaidOrders(ctx, pharmacyID)
	if err != nil {
		return nil, err
	}

	for _, o := range paidOrders {
		stats.TotalSales += o.Amount
		stats.OrderCount++
	}

	stats.PlatformRoyalty = (stats.TotalSales * 20) / 100
	stats.MerchantEarnings = stats.TotalSales - stats.PlatformRoyalty

	pendingAmount, err := s.repo.GetPendingPaymentAmount(ctx, pharmacyID)
	if err == nil {
		stats.PendingPayments = pendingAmount
	}

	return &stats, nil
}

func (s *service) RecordWarehouseMetric(ctx context.Context, warehouseID uint, temp float64, humidity float64) error {
	metric := ColdChainMetric{
		WarehouseID: warehouseID,
		Temperature: temp,
		Humidity:    humidity,
		RecordedAt:  time.Now(),
	}
	return s.repo.CreateColdChainMetric(ctx, &metric)
}

func (s *service) GetWarehouseMetrics(ctx context.Context, warehouseID uint) ([]ColdChainMetric, error) {
	return s.repo.GetWarehouseMetrics(ctx, warehouseID)
}

func (s *service) GetPurchaseOrder(ctx context.Context, poID uint) (*PurchaseOrder, error) {
	return s.repo.GetPurchaseOrder(ctx, poID)
}

// ReceivePurchaseOrder processes goods receipt for a purchase order.
// C-2: uses Create (not Save/UpdateInventoryBatch) for new inventory batch records.
// C-3: removes hardcoded ExpiryDate and ReorderLevel; uses zero-values.
// H-6: safe type assertions for item fields.
func (s *service) ReceivePurchaseOrder(ctx context.Context, poID uint) error {
	return s.repo.ExecuteTransaction(ctx, func(txCtx context.Context, txRepo Repository) error {
		po, err := txRepo.GetPurchaseOrder(txCtx, poID)
		if err != nil {
			return err
		}

		if po.Status == "received" || po.Status == "cancelled" {
			return errors.New("purchase order already processed or cancelled")
		}

		var items []map[string]interface{}
		if err := json.Unmarshal(po.Items, &items); err != nil {
			return err
		}

		for _, item := range items {
			// H-6: safe type assertion for medicine_id
			medIDFloat, ok := item["medicine_id"].(float64)
			if !ok {
				return fmt.Errorf("invalid medicine_id for item")
			}
			medID := uint(medIDFloat)

			// H-6: safe type assertion for qty
			qty, ok := item["qty"].(float64)
			if !ok {
				return fmt.Errorf("invalid qty for item")
			}

			// C-3: use zero-values for ExpiryDate and ReorderLevel unless provided in item
			var expiryDate time.Time
			if expStr, ok := item["expiry_date"].(string); ok && expStr != "" {
				if t, err := time.Parse("2006-01-02", expStr); err == nil {
					expiryDate = t
				}
			}
			reorderLevel := 0
			if rl, ok := item["reorder_level"].(float64); ok {
				reorderLevel = int(rl)
			}

			// C-2: use Create (not Save/UpdateInventoryBatch) for new batch records
			batch := PharmacyInventory{
				PharmacyID:    po.PharmacyID,
				MedicineID:    medID,
				BatchNumber:   fmt.Sprintf("RECV-%d-%d", po.ID, time.Now().Unix()),
				StockQuantity: int(qty),
				ReorderLevel:  reorderLevel,
				ExpiryDate:    expiryDate,
				IsActive:      true,
			}

			if err := txRepo.GetDB(txCtx).Create(&batch).Error; err != nil {
				return err
			}
		}

		now := time.Now()
		po.Status = "received"
		po.ReceivedDate = &now
		if err := txRepo.UpdatePurchaseOrder(txCtx, po); err != nil {
			return err
		}

		// Use txCtx for cross-domain finance calls to maintain same transaction
		inventoryAcct, err := s.financeSvc.GetOrCreateAccount(txCtx, fmt.Sprintf("Asset:Inventory:Pharmacy:%d", po.PharmacyID), finance.TypeAsset)
		if err != nil {
			return err
		}
		taxAcct, err := s.financeSvc.GetOrCreateAccount(txCtx, "Asset:Tax:InputGST", finance.TypeAsset)
		if err != nil {
			return err
		}
		payableAcct, err := s.financeSvc.GetOrCreateAccount(txCtx, fmt.Sprintf("Liability:AccountsPayable:Supplier:%d", po.SupplierID), finance.TypeLiability)
		if err != nil {
			return err
		}

		entries := []finance.EntryData{
			{AccountID: inventoryAcct.ID, Direction: finance.DirectionDebit, Amount: po.Amount},
			{AccountID: taxAcct.ID, Direction: finance.DirectionDebit, Amount: po.TaxAmount},
			{AccountID: payableAcct.ID, Direction: finance.DirectionCredit, Amount: po.TotalWithTax},
		}

		_, err = s.financeSvc.RecordTransaction(txCtx, fmt.Sprintf("GRN-%d", po.ID), "PROCUREMENT", fmt.Sprintf("Goods Receipt for PO #%s", po.PONumber), entries)
		return err
	})
}

// CreatePurchaseOrder creates a new purchase order with tax calculation.
// H-6: safe type assertions for qty and unit_cost.
func (s *service) CreatePurchaseOrder(ctx context.Context, supplierID uint, pharmacyID uint, items []map[string]interface{}) (*PurchaseOrder, error) {
	pharmacy, err := s.repo.GetPharmacyByID(ctx, pharmacyID)
	if err != nil {
		return nil, err
	}

	var Amount int64
	for _, item := range items {
		// H-6: safe assertions
		qty, ok := item["qty"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid qty for item")
		}
		unitCost, ok := item["unit_cost"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid unit_cost for item")
		}
		Amount += int64(qty) * int64(unitCost)
	}

	taxAmount, _ := s.financeSvc.CalculateTax(ctx, Amount, pharmacy.Region, "MEDICINE")
	itemsJSON, _ := json.Marshal(items)

	po := &PurchaseOrder{
		PONumber:     fmt.Sprintf("PO-%d-%d", time.Now().Unix(), pharmacyID),
		SupplierID:   supplierID,
		PharmacyID:   pharmacyID,
		Items:        itemsJSON,
		Amount:    Amount,
		TaxAmount:    taxAmount,
		TotalWithTax: Amount + taxAmount,
		Status:       "draft",
		ExpectedDate: time.Now().AddDate(0, 0, 7),
	}

	if err := s.repo.CreatePurchaseOrder(ctx, po); err != nil {
		return nil, err
	}

	return po, nil
}

// SettleSupplierPayment settles a supplier payment for a received PO.
// H-27: proper error handling for GetOrCreateAccount calls.
func (s *service) SettleSupplierPayment(ctx context.Context, poID uint) error {
	po, err := s.repo.GetPurchaseOrder(ctx, poID)
	if err != nil {
		return err
	}

	if po.Status != "received" {
		return errors.New("can only settle payments for received purchase orders")
	}

	payableAcct, err := s.financeSvc.GetOrCreateAccount(ctx, fmt.Sprintf("Liability:AccountsPayable:Supplier:%d", po.SupplierID), finance.TypeLiability)
	if err != nil {
		return fmt.Errorf("failed to get accounts payable account: %w", err)
	}
	bankAcct, err := s.financeSvc.GetOrCreateAccount(ctx, "Asset:Cash", finance.TypeAsset)
	if err != nil {
		return fmt.Errorf("failed to get bank account: %w", err)
	}

	entries := []finance.EntryData{
		{AccountID: payableAcct.ID, Direction: finance.DirectionDebit, Amount: po.TotalWithTax},
		{AccountID: bankAcct.ID, Direction: finance.DirectionCredit, Amount: po.TotalWithTax},
	}

	_, err = s.financeSvc.RecordTransaction(ctx, fmt.Sprintf("PAY-SUP-%d", po.ID), "PAYMENT", fmt.Sprintf("Supplier payment for PO #%s", po.PONumber), entries)
	return err
}

// CheckAndTriggerRestock auto-creates POs for low stock medicines.
// H-25: uses medicine's Price field instead of unit_cost: 0; skips if no price available.
func (s *service) CheckAndTriggerRestock(ctx context.Context) (int, error) {
	lowStock, err := s.repo.GetLowStockInventory(ctx)
	if err != nil {
		return 0, err
	}
	if len(lowStock) == 0 {
		return 0, nil
	}

	centralWarehouse, err := s.repo.GetCentralWarehouse(ctx)
	if err != nil {
		return 0, errors.New("central warehouse not configured — cannot auto-restock")
	}

	supplier, err := s.repo.GetSupplier(ctx)
	if err != nil {
		return 0, errors.New("no active supplier found — cannot auto-restock")
	}

	count := 0
	medsSeen := map[uint]bool{}
	for _, inv := range lowStock {
		if medsSeen[inv.MedicineID] {
			continue
		}
		medsSeen[inv.MedicineID] = true

		// H-25: fetch medicine's price instead of hardcoding 0
		med, medErr := s.repo.GetMedicine(ctx, inv.MedicineID)
		if medErr != nil {
			log.Printf("[restock] warning: could not fetch medicine %d: %v — skipping", inv.MedicineID, medErr)
			continue
		}
		if med.Prices["INR"] <= 0 {
			log.Printf("[restock] warning: medicine %d (%s) has no price — skipping zero-cost PO", med.ID, med.Name)
			continue
		}

		restockQty := float64(inv.ReorderLevel * 3) // order 3x reorder level
		items := []map[string]interface{}{
			{"medicine_id": float64(inv.MedicineID), "qty": restockQty, "unit_cost": float64(med.Prices["INR"])},
		}

		_, poErr := s.CreatePurchaseOrder(ctx, supplier.ID, centralWarehouse.ID, items)
		if poErr != nil {
			continue
		}
		count++
	}
	return count, nil
}

func (s *service) GetDashboardStats(ctx context.Context) (map[string]interface{}, error) {
	var totalOrders, ordersToday, pendingOrders, totalMedicines, totalInventoryItems, lowStockItems int64
	var totalSales, todaySales, inventoryValue float64

	today := time.Now().Truncate(24 * time.Hour)

	// Orders Stats
	s.repo.GetDB(ctx).Model(&PharmacyOrder{}).Count(&totalOrders)
	s.repo.GetDB(ctx).Model(&PharmacyOrder{}).Where("created_at >= ?", today).Count(&ordersToday)
	s.repo.GetDB(ctx).Model(&PharmacyOrder{}).Where("status = ?", "pending").Count(&pendingOrders)

	// Total Sales (Paid Orders) — lifetime
	s.repo.GetDB(ctx).Model(&PharmacyOrder{}).
		Where("payment_status = ?", "paid").
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&totalSales)

	// Today's revenue — paid orders created today
	s.repo.GetDB(ctx).Model(&PharmacyOrder{}).
		Where("payment_status = ? AND created_at >= ?", "paid", today).
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&todaySales)

	// Inventory Stats
	s.repo.GetDB(ctx).Model(&Medicine{}).Count(&totalMedicines)
	s.repo.GetDB(ctx).Model(&PharmacyInventory{}).Count(&totalInventoryItems)
	s.repo.GetDB(ctx).Model(&PharmacyInventory{}).Where("is_active = ? AND stock_quantity <= reorder_level", true).Count(&lowStockItems)

	// Inventory Value
	s.repo.GetDB(ctx).Table("pharmacyinventory").
		Joins("JOIN medicine ON pharmacyinventory.medicine_id = medicine.id").
		Select("COALESCE(SUM(pharmacyinventory.stock_quantity * medicine.unit_price), 0)").
		Scan(&inventoryValue)

	return map[string]interface{}{
		"total_sales":           totalSales,
		"today_sales":           todaySales,
		"orders_today":          ordersToday,
		"pending_orders":        pendingOrders,
		"low_stock_items":       lowStockItems,
		"total_medicines":       totalMedicines,
		"total_inventory_items": totalInventoryItems,
		"inventory_value":       inventoryValue,
	}, nil
}

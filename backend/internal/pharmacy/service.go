//go:build ignore
// +build ignore

package pharmacy

import (
	"encoding/json"
	"errors"
	"fmt"
	"drayya-core/internal/database"
	"drayya-core/internal/ledger"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/models"
	"time"
)

func GetInventory(pharmacyID uint) ([]pharmacy.PharmacyInventory, error) {
	var inventory []pharmacy.PharmacyInventory
	db := database.DB.Preload("Medicine")

	if pharmacyID != 0 {
		db = db.Where("pharmacy_id = ?", pharmacyID)
	}

	if err := db.Where("is_active = ?", true).Find(&inventory).Error; err != nil {
		return nil, err
	}
	return inventory, nil
}

type StockAlerts struct {
	LowStock     []pharmacy.PharmacyInventory `json:"low_stock"`
	OutOfStock   []pharmacy.PharmacyInventory `json:"out_of_stock"`
	ExpiringSoon []pharmacy.PharmacyInventory `json:"expiring_soon"`
	TotalAlerts  int                        `json:"total_alerts"`
}

func GetStockAlerts() (StockAlerts, error) {
	var allInventory []pharmacy.PharmacyInventory
	if err := database.DB.Preload("Medicine").Where("is_active = ?", true).Find(&allInventory).Error; err != nil {
		return StockAlerts{}, err
	}

	alerts := StockAlerts{}
	expiryThreshold := time.Now().AddDate(0, 0, 90) // 90 days

	for _, item := range allInventory {
		if item.StockQuantity == 0 {
			alerts.OutOfStock = append(alerts.OutOfStock, item)
		} else if item.StockQuantity <= item.ReorderLevel {
			alerts.LowStock = append(alerts.LowStock, item)
		}

		if item.StockQuantity > 0 && item.ExpiryDate.Before(expiryThreshold) {
			alerts.ExpiringSoon = append(alerts.ExpiringSoon, item)
		}
	}

	alerts.TotalAlerts = len(alerts.LowStock) + len(alerts.OutOfStock) + len(alerts.ExpiringSoon)
	return alerts, nil
}

func GetLowStockInventory() ([]pharmacy.PharmacyInventory, error) {
	var inventory []pharmacy.PharmacyInventory
	err := database.DB.Preload("Medicine").
		Where("is_active = ? AND stock_quantity <= reorder_level", true).
		Find(&inventory).Error
	return inventory, err
}

func CheckAndTriggerRestock() (int, error) {
	lowStockItems, err := GetLowStockInventory()
	if err != nil {
		return 0, err
	}

	triggeredCount := 0

	for _, item := range lowStockItems {
		var pharmacy pharmacy.Pharmacy
		if err := database.DB.First(&pharmacy, item.PharmacyID).Error; err != nil {
			continue
		}

		// Calculate order amount (e.g., to reach 3x reorder level)
		targetStock := item.ReorderLevel * 3
		orderQty := targetStock - item.StockQuantity
		if orderQty <= 0 {
			orderQty = item.ReorderLevel
		}

		if pharmacy.IsCentralWarehouse {
			// Central Warehouse restocks from external Suppliers via Purchase Order
			var supplier pharmacy.Supplier
			if err := database.DB.Where("is_active = ?", true).First(&supplier).Error; err != nil {
				continue
			}

			itemsJSON, _ := json.Marshal([]map[string]interface{}{
				{
					"medicine_id": item.MedicineID,
					"qty":         orderQty,
					"unit_cost":   (item.Medicine.UnitPrice * 60) / 100,
				},
			})

			po := pharmacy.PurchaseOrder{
				PONumber:      fmt.Sprintf("PO-%d-%d", time.Now().Unix(), item.ID),
				SupplierID:    supplier.ID,
				PharmacyID:    pharmacy.ID,
				Items:         itemsJSON,
				TotalCost:     int64(orderQty) * ((item.Medicine.UnitPrice * 60) / 100),
				Status:        "ordered",
				ExpectedDate:  time.Now().AddDate(0, 0, 3),
				IsAutoTrigger: true,
			}
			database.DB.Create(&po)
			triggeredCount++

		} else {
			// Franchise/Regional Hub restocks from Parent Hub via Inventory Transfer
			parentID := pharmacy.ParentHubID
			if parentID == nil {
				// Fallback to any Central Warehouse if no parent defined
				var centralWarehouse pharmacy.Pharmacy
				if err := database.DB.Where("is_central_warehouse = ? AND is_active = ?", true, true).First(&centralWarehouse).Error; err != nil {
					continue
				}
				pID := centralWarehouse.ID
				parentID = &pID
			}

			// Check if pending transfer already exists
			var existingTransfer pharmacy.InventoryTransfer
			err := database.DB.Where("destination_id = ? AND medicine_id = ? AND status IN ?",
				pharmacy.ID, item.MedicineID, []string{"requested", "in_transit"}).First(&existingTransfer).Error

			if err != nil {
				transfer := pharmacy.InventoryTransfer{
					SourceID:      *parentID,
					DestinationID: pharmacy.ID,
					MedicineID:    item.MedicineID,
					Quantity:      orderQty,
					Status:        "requested",
					RequestedByID: pharmacy.ID,
				}
				database.DB.Create(&transfer)
				triggeredCount++
			}
		}
	}

	return triggeredCount, nil
}

func GetExpiringInventory() ([]pharmacy.PharmacyInventory, error) {
	var inventory []pharmacy.PharmacyInventory
	expiryThreshold := time.Now().AddDate(0, 0, 90)
	err := database.DB.Preload("Medicine").
		Where("is_active = ? AND expiry_date < ?", true, expiryThreshold).
		Find(&inventory).Error
	return inventory, err
}

func GetAllPharmacies() ([]pharmacy.Pharmacy, error) {
	var pharmacies []pharmacy.Pharmacy
	err := database.DB.Where("is_active = ?", true).Find(&pharmacies).Error
	return pharmacies, err
}

func ConvertPrescriptionToOrder(prescriptionID uint, pharmacyID uint, shippingAddress string, phone string) (*pharmacy.PharmacyOrder, error) {
	var prescription models.Prescription
	if err := database.DB.First(&prescription, prescriptionID).Error; err != nil {
		return nil, err
	}

	// In a real app, you would parse prescription.Medicines JSON array, map names to pharmacy.Medicine IDs,
	// calculate unit prices, build items, and get total amounts.
	// We'll mock the JSON array and TotalAmount setup here for brevity.

	order := pharmacy.PharmacyOrder{
		OrderNumber:     "ORD-" + time.Now().Format("20060102150405"),
		UserID:          prescription.PatientID,
		PharmacyID:      pharmacyID,
		Items:           prescription.Medicines, // Directly transferring prescribed JSON for now
		TotalAmount:     15000,                  // Mocked total (150.00 in cents)
		Status:          "pending",
		PaymentStatus:   "unpaid",
		ShippingAddress: shippingAddress,
		DeliveryPhone:   phone,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := database.DB.Create(&order).Error; err != nil {
		return nil, err
	}

	return &order, nil
}

type CartItem struct {
	Product  pharmacy.PharmacyInventory `json:"product"`
	Quantity int                      `json:"quantity"`
	Type     string                   `json:"type"`
}

func GenerateCartFromPrescription(prescriptionID uint) ([]CartItem, error) {
	var prescription models.Prescription
	if err := database.DB.First(&prescription, prescriptionID).Error; err != nil {
		return nil, err
	}

	var cart []CartItem

	// Parse the prescription.Medicines JSON array
	var meds []map[string]interface{}
	if err := json.Unmarshal(prescription.Medicines, &meds); err != nil {
		// Return empty cart if parsing fails
		return cart, nil
	}

	// This is a simplified matching logic. We look for any active inventory that matches the name.
	for _, medMap := range meds {
		name, _ := medMap["name"].(string)
		if name == "" {
			continue
		}

		// Find a matching product in inventory (Central Warehouse or a generic one)
		var inventory pharmacy.PharmacyInventory
		// We join with Medicine to search by name
		err := database.DB.Preload("Medicine").
			Joins("JOIN medicine ON medicine.id = pharmacyinventory.medicine_id").
			Where("medicine.name ILIKE ?", "%"+name+"%").
			First(&inventory).Error

		if err == nil {
			cart = append(cart, CartItem{
				Product:  inventory,
				Quantity: 1, // Default to 1 unit, could be parsed from duration
				Type:     "pharmacy",
			})
		}
	}

	return cart, nil
}

func ReserveInventory(orderID uint) error {
	var order pharmacy.PharmacyOrder
	if err := database.DB.First(&order, orderID).Error; err != nil {
		return err
	}

	var items []map[string]interface{}
	if err := json.Unmarshal(order.Items, &items); err != nil {
		return fmt.Errorf("failed to parse order items: %w", err)
	}

	// Begin Transaction
	tx := database.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	for _, item := range items {
		medIDFloat, ok := item["medicine_id"].(float64)
		if !ok {
			// Try as string if it wasn't a float
			continue
		}
		medID := uint(medIDFloat)

		qtyFloat, ok := item["quantity"].(float64)
		if !ok {
			continue
		}
		requiredQty := int(qtyFloat)

		// Get all inventory batches for this medicine at this pharmacy, ordered by expiry date ascending (FEFO logic)
		var batches []pharmacy.PharmacyInventory
		if err := tx.Where("pharmacy_id = ? AND medicine_id = ? AND stock_quantity > 0 AND is_active = ?", order.PharmacyID, medID, true).
			Order("expiry_date ASC").Find(&batches).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("error finding inventory for medicine ID %d: %w", medID, err)
		}

		remainingToFulfill := requiredQty
		for _, batch := range batches {
			if remainingToFulfill <= 0 {
				break
			}

			deduct := remainingToFulfill
			if batch.StockQuantity < deduct {
				deduct = batch.StockQuantity
			}

			batch.StockQuantity -= deduct
			remainingToFulfill -= deduct

			if err := tx.Save(&batch).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("error updating batch %s: %w", batch.BatchNumber, err)
			}
		}

		if remainingToFulfill > 0 {
			tx.Rollback()
			return fmt.Errorf("insufficient stock for medicine ID %d (short by %d)", medID, remainingToFulfill)
		}
	}

	return tx.Commit().Error
}

type CityStock struct {
	City       string `json:"city"`
	MedID      uint   `json:"med_id"`
	MedName    string `json:"med_name"`
	TotalStock int    `json:"total_stock"`
}

func GetMultiCityInventory() ([]CityStock, error) {
	var results []CityStock
	err := database.DB.Table("pharmacyinventory").
		Select("pharmacy.city, medicine.id as med_id, medicine.name as med_name, SUM(pharmacyinventory.stock_quantity) as total_stock").
		Joins("JOIN pharmacy ON pharmacy.id = pharmacyinventory.pharmacy_id").
		Joins("JOIN medicine ON medicine.id = pharmacyinventory.medicine_id").
		Group("pharmacy.city, medicine.id, medicine.name").
		Scan(&results).Error
	return results, err
}

// Admin CRUD functions

func ListAllPharmaciesAdmin() ([]pharmacy.Pharmacy, error) {
	var pharmacies []pharmacy.Pharmacy
	err := database.DB.Find(&pharmacies).Error
	return pharmacies, err
}

func CreatePharmacy(name, address, city string, lat, lng, radius float64, isHub bool) (*pharmacy.Pharmacy, error) {
	p := pharmacy.Pharmacy{
		Name:               name,
		Address:            address,
		City:               city,
		Latitude:           lat,
		Longitude:          lng,
		ServiceRadius:      radius,
		IsCentralWarehouse: isHub,
		IsActive:           true,
	}
	if err := database.DB.Create(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func UpdatePharmacy(id uint, name, address, city string, lat, lng, radius float64, isHub, isActive bool) (*pharmacy.Pharmacy, error) {
	var p pharmacy.Pharmacy
	if err := database.DB.First(&p, id).Error; err != nil {
		return nil, err
	}
	p.Name = name
	p.Address = address
	p.City = city
	p.Latitude = lat
	p.Longitude = lng
	p.ServiceRadius = radius
	p.IsCentralWarehouse = isHub
	p.IsActive = isActive
	if err := database.DB.Save(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func TogglePharmacy(id uint) (*pharmacy.Pharmacy, error) {
	var p pharmacy.Pharmacy
	if err := database.DB.First(&p, id).Error; err != nil {
		return nil, err
	}
	p.IsActive = !p.IsActive
	if err := database.DB.Save(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func PayOrder(orderID uint) error {
	var order pharmacy.PharmacyOrder
	if err := database.DB.Preload("Pharmacy").First(&order, orderID).Error; err != nil {
		return err
	}

	if order.PaymentStatus == "paid" {
		return errors.New("order already paid")
	}

	// 1. Initialize Ledger Service
	ledgerSvc := ledger.NewService()

	// 2. Fetch Accounts
	// Patient Account (Liability to system/Asset to patient)
	patientAcct, err := ledgerSvc.GetOrCreateAccount(fmt.Sprintf("PatientWallet:%d", order.UserID), models.TypeLiability)
	if err != nil {
		return err
	}

	// System/Commission Account (Revenue)
	platformAcct, err := ledgerSvc.GetOrCreateAccount("Revenue:PlatformRoyalty", models.TypeRevenue)
	if err != nil {
		return err
	}

	// Merchant/Franchise Account (Liability to them)
	merchantAcctName := fmt.Sprintf("MerchantWallet:Pharmacy:%d", order.PharmacyID)
	if order.Pharmacy.IsCentralWarehouse {
		merchantAcctName = "Revenue:CentralWarehouseSales"
	}
	merchantAcct, err := ledgerSvc.GetOrCreateAccount(merchantAcctName, models.TypeRevenue)
	if err != nil {
		return err
	}

	// 3. Calculate Splits (e.g. 20% platform commission)
	totalPaise := order.TotalAmount
	platformCommission := (totalPaise * 20) / 100
	merchantShare := totalPaise - platformCommission

	// 4. Record Ledger Transaction
	entries := []ledger.EntryData{
		{
			AccountID: patientAcct.ID,
			Direction: models.DirectionDebit, // Liability decreases (we owe them less because they spent it)
			Amount:    totalPaise,
		},
		{
			AccountID: platformAcct.ID,
			Direction: models.DirectionCredit, // Revenue increases
			Amount:    platformCommission,
		},
		{
			AccountID: merchantAcct.ID,
			Direction: models.DirectionCredit, // Revenue/Liability increases
			Amount:    merchantShare,
		},
	}

	txn, err := ledgerSvc.RecordTransaction(fmt.Sprintf("PAY-ORD-%d", order.ID), "PAYMENT", fmt.Sprintf("Order payment for #%s", order.OrderNumber), entries)
	if err != nil {
		return err
	}

	// 5. Update Order Status
	order.PaymentStatus = "paid"
	order.Status = "confirmed"
	order.LedgerTransactionID = &txn.ID

	return database.DB.Save(&order).Error
}

type PharmacyFinancials struct {
	TotalSales       int64 `json:"total_sales"`
	PlatformRoyalty  int64 `json:"platform_royalty"`
	MerchantEarnings int64 `json:"merchant_earnings"`
	OrderCount       int64 `json:"order_count"`
	PendingPayments  int64 `json:"pending_payments"`
}

func GetPharmacyFinancials(pharmacyID uint) (*PharmacyFinancials, error) {
	var stats PharmacyFinancials

	// Total Paid Orders
	var paidOrders []pharmacy.PharmacyOrder
	query := database.DB.Where("payment_status = ?", "paid")
	if pharmacyID != 0 {
		query = query.Where("pharmacy_id = ?", pharmacyID)
	}

	if err := query.Find(&paidOrders).Error; err != nil {
		return nil, err
	}

	for _, o := range paidOrders {
		stats.TotalSales += o.TotalAmount
		stats.OrderCount++
	}

	// Calculate splits (matching PayOrder logic)
	stats.PlatformRoyalty = (stats.TotalSales * 20) / 100
	stats.MerchantEarnings = stats.TotalSales - stats.PlatformRoyalty

	// Pending Payments
	var pendingAmount int64
	queryPending := database.DB.Model(&pharmacy.PharmacyOrder{}).Where("payment_status = ?", "pending")
	if pharmacyID != 0 {
		queryPending = queryPending.Where("pharmacy_id = ?", pharmacyID)
	}
	queryPending.Select("SUM(total_amount)").Scan(&pendingAmount)
	stats.PendingPayments = pendingAmount

	return &stats, nil
}

func RecordWarehouseMetric(warehouseID uint, temp float64, humidity float64) error {
	metric := pharmacy.ColdChainMetric{
		WarehouseID: warehouseID,
		Temperature: temp,
		Humidity:    humidity,
		RecordedAt:  time.Now(),
	}
	return database.DB.Create(&metric).Error
}

func GetWarehouseMetrics(warehouseID uint) ([]pharmacy.ColdChainMetric, error) {
	var metrics []pharmacy.ColdChainMetric
	err := database.DB.Where("warehouse_id = ?", warehouseID).Order("recorded_at desc").Limit(100).Find(&metrics).Error
	return metrics, err
}


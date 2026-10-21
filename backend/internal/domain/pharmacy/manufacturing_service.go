package pharmacy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"drayya-core/internal/crypto"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// ManufacturingService provides business logic for BOM, production orders,
// material issuance, GRN, and the multi-city workflow dashboard.
type ManufacturingService interface {
	// BOM
	CreateBOM(ctx context.Context, bom *BillOfMaterials) error
	GetBOM(ctx context.Context, id uint) (*BillOfMaterials, error)
	GetBOMByMedicine(ctx context.Context, medicineID uint) (*BillOfMaterials, error)
	ListBOMs(ctx context.Context) ([]BillOfMaterials, error)

	// Production Orders
	CreateProductionOrder(ctx context.Context, po *ProductionOrder) error
	GetProductionOrder(ctx context.Context, id uint) (*ProductionOrder, error)
	ListProductionOrders(ctx context.Context, filters ProductionOrderFilters) ([]ProductionOrder, int64, error)
	UpdateProductionStatus(ctx context.Context, id uint, newStatus ProductionStatus, notes string, userID uint) error
	GetProductionDashboard(ctx context.Context, warehouseID uint) (*ProductionDashboard, error)

	// Manufacturing Logs
	AddManufacturingLog(ctx context.Context, log *ManufacturingLog) error
	GetManufacturingLogs(ctx context.Context, productionOrderID uint) ([]ManufacturingLog, error)

	// Material Issuance
	IssueMaterialsForProduction(ctx context.Context, productionOrderID uint, userID uint) ([]MaterialIssuance, error)
	ValidateBOMAvailability(ctx context.Context, bomID uint, targetQty int, warehouseID uint) ([]MaterialShortage, error)

	// Finished Goods Release
	ReleaseFinishedGoods(ctx context.Context, productionOrderID uint, yieldQty int, expiryDate time.Time, userID uint) error

	// GRN
	CreateGRN(ctx context.Context, grn *GoodsReceiptNote) error
	GetGRN(ctx context.Context, id uint) (*GoodsReceiptNote, error)
	ListGRNs(ctx context.Context, warehouseID uint, status string) ([]GoodsReceiptNote, error)
	ProcessGRNItemQA(ctx context.Context, grnItemID uint, qaStatus string, notes string) error

	// Multi-City Dashboard
	GetCityWorkflowDashboard(ctx context.Context, cityName string) (*CityWorkflowDashboard, error)
}

type manufacturingService struct {
	mfgRepo ManufacturingRepository
	rmRepo  RawMaterialRepository
	repo    Repository // existing pharmacy repository for inventory
}

func NewManufacturingService(mfgRepo ManufacturingRepository, rmRepo RawMaterialRepository, repo Repository) ManufacturingService {
	return &manufacturingService{
		mfgRepo: mfgRepo,
		rmRepo:  rmRepo,
		repo:    repo,
	}
}

// ── BOM ──

func (s *manufacturingService) CreateBOM(ctx context.Context, bom *BillOfMaterials) error {
	// Validate medicine exists
	_, err := s.repo.GetMedicine(ctx, bom.MedicineID)
	if err != nil {
		return fmt.Errorf("medicine not found: %w", err)
	}

	// Validate all raw materials exist
	for _, item := range bom.Items {
		_, err := s.rmRepo.GetRawMaterial(ctx, item.RawMaterialID)
		if err != nil {
			return fmt.Errorf("raw material ID %d not found: %w", item.RawMaterialID, err)
		}
	}

	return s.mfgRepo.CreateBOM(ctx, bom)
}

func (s *manufacturingService) GetBOM(ctx context.Context, id uint) (*BillOfMaterials, error) {
	return s.mfgRepo.GetBOM(ctx, id)
}

func (s *manufacturingService) GetBOMByMedicine(ctx context.Context, medicineID uint) (*BillOfMaterials, error) {
	return s.mfgRepo.GetBOMByMedicine(ctx, medicineID)
}

func (s *manufacturingService) ListBOMs(ctx context.Context) ([]BillOfMaterials, error) {
	return s.mfgRepo.ListBOMs(ctx)
}

// ── Production Orders ──

func (s *manufacturingService) CreateProductionOrder(ctx context.Context, po *ProductionOrder) error {
	// Validate BOM exists and is active
	bom, err := s.mfgRepo.GetBOM(ctx, po.BOMID)
	if err != nil {
		return fmt.Errorf("BOM not found: %w", err)
	}
	if !bom.IsActive {
		return fmt.Errorf("BOM version %d is inactive", bom.Version)
	}

	// Validate medicine matches BOM
	if bom.MedicineID != po.MedicineID {
		return fmt.Errorf("BOM medicine_id %d does not match production order medicine_id %d", bom.MedicineID, po.MedicineID)
	}

	// Validate warehouse exists
	_, err = s.repo.GetPharmacyByID(ctx, po.WarehouseID)
	if err != nil {
		return fmt.Errorf("warehouse not found: %w", err)
	}

	// Generate production order number
	po.PONumber = fmt.Sprintf("MFG-%s", strings.ToUpper(crypto.SecureToken(6)))
	po.Status = ProdStatusDraft

	return s.mfgRepo.CreateProductionOrder(ctx, po)
}

func (s *manufacturingService) GetProductionOrder(ctx context.Context, id uint) (*ProductionOrder, error) {
	return s.mfgRepo.GetProductionOrder(ctx, id)
}

func (s *manufacturingService) ListProductionOrders(ctx context.Context, filters ProductionOrderFilters) ([]ProductionOrder, int64, error) {
	return s.mfgRepo.ListProductionOrders(ctx, filters)
}

func (s *manufacturingService) UpdateProductionStatus(ctx context.Context, id uint, newStatus ProductionStatus, notes string, userID uint) error {
	po, err := s.mfgRepo.GetProductionOrder(ctx, id)
	if err != nil {
		return fmt.Errorf("production order not found: %w", err)
	}

	if !po.IsTransitionAllowed(newStatus) {
		return fmt.Errorf("invalid transition from %s to %s", po.Status, newStatus)
	}

	now := time.Now()

	switch newStatus {
	case ProdStatusScheduled:
		if po.ScheduledDate == nil {
			return fmt.Errorf("scheduled_date is required to schedule production")
		}
	case ProdStatusInProduction:
		po.StartedAt = &now
	case ProdStatusQAPending:
		po.CompletedAt = &now
	case ProdStatusQAPassed:
		po.QAResult = "pass"
		po.QAInspectorID = &userID
		po.QACompletedAt = &now
	case ProdStatusQAFailed:
		po.QAResult = "fail"
		po.QAInspectorID = &userID
		po.QACompletedAt = &now
	}

	po.Status = newStatus
	if notes != "" {
		po.Notes = notes
	}

	// Log the status change
	logEntry := ManufacturingLog{
		ProductionOrderID: po.ID,
		Stage:             fmt.Sprintf("status_change:%s", newStatus),
		OperatorID:        userID,
		Notes:             notes,
		StartedAt:         now,
	}
	if err := s.mfgRepo.CreateManufacturingLog(ctx, &logEntry); err != nil {
		return fmt.Errorf("failed to create log: %w", err)
	}

	return s.mfgRepo.UpdateProductionOrder(ctx, po)
}

func (s *manufacturingService) GetProductionDashboard(ctx context.Context, warehouseID uint) (*ProductionDashboard, error) {
	return s.mfgRepo.GetProductionDashboard(ctx, warehouseID)
}

// ── Manufacturing Logs ──

func (s *manufacturingService) AddManufacturingLog(ctx context.Context, log *ManufacturingLog) error {
	// Validate production order exists
	_, err := s.mfgRepo.GetProductionOrder(ctx, log.ProductionOrderID)
	if err != nil {
		return fmt.Errorf("production order not found: %w", err)
	}
	return s.mfgRepo.CreateManufacturingLog(ctx, log)
}

func (s *manufacturingService) GetManufacturingLogs(ctx context.Context, productionOrderID uint) ([]ManufacturingLog, error) {
	return s.mfgRepo.GetManufacturingLogs(ctx, productionOrderID)
}

// ── Material Issuance ──

func (s *manufacturingService) ValidateBOMAvailability(ctx context.Context, bomID uint, targetQty int, warehouseID uint) ([]MaterialShortage, error) {
	bom, err := s.mfgRepo.GetBOM(ctx, bomID)
	if err != nil {
		return nil, fmt.Errorf("BOM not found: %w", err)
	}

	var shortages []MaterialShortage

	for _, item := range bom.Items {
		if item.IsOptional {
			continue
		}

		// Calculate required quantity including wastage
		requiredQty := item.QtyPerUnit * float64(targetQty)
		requiredQty *= (1 + item.WastagePercent/100) // Add wastage allowance

		// Check available stock (only QA-approved inventory)
		available, err := s.rmRepo.GetRMBalance(ctx, item.RawMaterialID, warehouseID)
		if err != nil {
			return nil, fmt.Errorf("failed to check balance for raw material %d: %w", item.RawMaterialID, err)
		}

		if available < requiredQty {
			shortages = append(shortages, MaterialShortage{
				RawMaterialID:   item.RawMaterialID,
				RawMaterialName: item.RawMaterial.Name,
				RequiredQty:     requiredQty,
				AvailableQty:    available,
				ShortageQty:     requiredQty - available,
				UnitOfMeasure:   item.UnitOfMeasure,
			})
		}
	}

	return shortages, nil
}

func (s *manufacturingService) IssueMaterialsForProduction(ctx context.Context, productionOrderID uint, userID uint) ([]MaterialIssuance, error) {
	po, err := s.mfgRepo.GetProductionOrder(ctx, productionOrderID)
	if err != nil {
		return nil, fmt.Errorf("production order not found: %w", err)
	}

	if po.Status != ProdStatusScheduled && po.Status != ProdStatusMaterialReserved {
		return nil, fmt.Errorf("production order must be in 'scheduled' or 'material_reserved' status to issue materials")
	}

	bom, err := s.mfgRepo.GetBOM(ctx, po.BOMID)
	if err != nil {
		return nil, fmt.Errorf("BOM not found: %w", err)
	}

	var issuances []MaterialIssuance

	err = s.mfgRepo.ExecuteTransaction(ctx, func(txCtx context.Context) error {
		for _, item := range bom.Items {
			if item.IsOptional {
				continue
			}

			requiredQty := item.QtyPerUnit * float64(po.TargetQty)
			requiredQty *= (1 + item.WastagePercent/100)

			// Find approved inventory sorted by expiry (FEFO)
			inventoryList, err := s.rmRepo.ListRMInventory(txCtx, po.WarehouseID, item.RawMaterialID, "approved")
			if err != nil {
				return fmt.Errorf("failed to fetch inventory for raw material %d: %w", item.RawMaterialID, err)
			}

			remaining := requiredQty
			for _, inv := range inventoryList {
				if remaining <= 0 {
					break
				}

				issueQty := remaining
				if inv.Quantity < issueQty {
					issueQty = inv.Quantity
				}

				// Deduct from inventory
				inv.Quantity -= issueQty
				if err := s.rmRepo.UpdateRMInventory(txCtx, &inv); err != nil {
					return fmt.Errorf("failed to update inventory: %w", err)
				}

				// Create transaction log
				txn := RawMaterialTransaction{
					InventoryID:     inv.ID,
					TransactionType: "manufacturing_issue",
					Quantity:        -issueQty,
					BalanceAfter:    inv.Quantity,
					UnitOfMeasure:   item.UnitOfMeasure,
					ReferenceType:   "production_order",
					ReferenceID:     &productionOrderID,
					PerformedByID:   userID,
					Notes:           fmt.Sprintf("Issued for production order %s", po.PONumber),
					CreatedAt:       time.Now(),
				}
				if err := s.rmRepo.CreateRMTransaction(txCtx, &txn); err != nil {
					return fmt.Errorf("failed to create transaction log: %w", err)
				}

				// Create issuance record
				issuance := MaterialIssuance{
					ProductionOrderID: productionOrderID,
					RawMaterialID:     item.RawMaterialID,
					InventoryID:       inv.ID,
					IssuedQty:         issueQty,
					UnitOfMeasure:     item.UnitOfMeasure,
					IssuedByID:        userID,
					IssuedAt:          time.Now(),
					CreatedAt:         time.Now(),
				}
				if err := s.mfgRepo.CreateMaterialIssuance(txCtx, &issuance); err != nil {
					return fmt.Errorf("failed to create issuance: %w", err)
				}
				issuances = append(issuances, issuance)

				remaining -= issueQty
			}

			if remaining > 0 {
				return fmt.Errorf("insufficient stock for raw material %s (short by %.2f %s)",
					item.RawMaterial.Name, remaining, item.UnitOfMeasure)
			}
		}

		// Update production order status
		po.Status = ProdStatusMaterialReserved
		return s.mfgRepo.UpdateProductionOrder(txCtx, po)
	})

	if err != nil {
		return nil, err
	}

	return issuances, nil
}

// ── Finished Goods Release ──

func (s *manufacturingService) ReleaseFinishedGoods(ctx context.Context, productionOrderID uint, yieldQty int, expiryDate time.Time, userID uint) error {
	po, err := s.mfgRepo.GetProductionOrder(ctx, productionOrderID)
	if err != nil {
		return fmt.Errorf("production order not found: %w", err)
	}

	if po.Status != ProdStatusQAPassed {
		return fmt.Errorf("production order must pass QA before release (current status: %s)", po.Status)
	}

	return s.mfgRepo.ExecuteTransaction(ctx, func(txCtx context.Context) error {
		// Generate batch number
		batchNumber := fmt.Sprintf("FG-%s-%s", po.PONumber, strings.ToUpper(crypto.SecureToken(4)))

		// Create finished goods batch record
		fgBatch := FinishedGoodsBatch{
			ProductionOrderID: productionOrderID,
			MedicineID:        po.MedicineID,
			BatchNumber:       batchNumber,
			QtyProduced:       yieldQty,
			ExpiryDate:        expiryDate,
			ManufactureDate:   time.Now(),
			HubID:             po.WarehouseID,
			Status:            "available",
		}
		if err := s.mfgRepo.CreateFinishedGoodsBatch(txCtx, &fgBatch); err != nil {
			return fmt.Errorf("failed to create finished goods batch: %w", err)
		}

		// Add to pharmacy inventory (existing PharmacyInventory table)
		invBatch := PharmacyInventory{
			PharmacyID:    po.WarehouseID,
			MedicineID:    po.MedicineID,
			BatchNumber:   batchNumber,
			StockQuantity: yieldQty,
			ReorderLevel:  0, // Manufacturing batches don't have reorder level
			ExpiryDate:    expiryDate,
			IsActive:      true,
		}
		if err := s.mfgRepo.GetDB(txCtx).Create(&invBatch).Error; err != nil {
			return fmt.Errorf("failed to add to inventory: %w", err)
		}

		// Update production order
		po.ActualYieldQty = &yieldQty
		po.BatchNumber = batchNumber
		po.Status = ProdStatusReleased
		now := time.Now()
		po.CompletedAt = &now

		// Log the release
		logEntry := ManufacturingLog{
			ProductionOrderID: po.ID,
			Stage:             "released",
			OperatorID:        userID,
			Notes:             fmt.Sprintf("Released %d units. Batch: %s. Yield: %.1f%%", yieldQty, batchNumber, float64(yieldQty)/float64(po.TargetQty)*100),
			StartedAt:         now,
		}
		if err := s.mfgRepo.CreateManufacturingLog(txCtx, &logEntry); err != nil {
			return fmt.Errorf("failed to create release log: %w", err)
		}

		return s.mfgRepo.UpdateProductionOrder(txCtx, po)
	})
}

// ── GRN ──

func (s *manufacturingService) CreateGRN(ctx context.Context, grn *GoodsReceiptNote) error {
	grn.GRNNumber = fmt.Sprintf("GRN-%s", strings.ToUpper(crypto.SecureToken(6)))
	grn.ReceivedAt = time.Now()
	if grn.Status == "" {
		grn.Status = "pending_inspection"
	}
	return s.mfgRepo.CreateGRN(ctx, grn)
}

func (s *manufacturingService) GetGRN(ctx context.Context, id uint) (*GoodsReceiptNote, error) {
	return s.mfgRepo.GetGRN(ctx, id)
}

func (s *manufacturingService) ListGRNs(ctx context.Context, warehouseID uint, status string) ([]GoodsReceiptNote, error) {
	return s.mfgRepo.ListGRNs(ctx, warehouseID, status)
}

func (s *manufacturingService) ProcessGRNItemQA(ctx context.Context, grnItemID uint, qaStatus string, notes string) error {
	db := s.mfgRepo.GetDB(ctx)

	var item GRNItem
	if err := db.First(&item, grnItemID).Error; err != nil {
		return fmt.Errorf("GRN item not found: %w", err)
	}

	item.QAStatus = qaStatus
	item.QANotes = notes

	if qaStatus == "passed" {
		item.AcceptedQty = item.ReceivedQty
		item.RejectedQty = 0
	} else if qaStatus == "failed" {
		item.AcceptedQty = 0
		item.RejectedQty = item.ReceivedQty
	}

	return s.mfgRepo.UpdateGRNItem(ctx, &item)
}

// ── Multi-City Dashboard ──

func (s *manufacturingService) GetCityWorkflowDashboard(ctx context.Context, cityName string) (*CityWorkflowDashboard, error) {
	db := s.mfgRepo.GetDB(ctx)

	// Use a silent-logger session for the lookups below — they have legitimate
	// "no rows" outcomes that we handle gracefully (fallback to any pharmacy in
	// the city, then return a friendly error). Without this, GORM logs every
	// empty result at WARN level which floods production logs.
	silent := db.Session(&gorm.Session{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})

	// Find the regional/global hub for this city
	var warehouse Pharmacy
	err := silent.Where("city = ? AND (hub_tier = 'regional' OR hub_tier = 'global') AND is_active = ?", cityName, true).
		First(&warehouse).Error
	if err != nil {
		// Fallback: try any active pharmacy in that city
		err = silent.Where("city = ? AND is_active = ?", cityName, true).First(&warehouse).Error
		if err != nil {
			return nil, fmt.Errorf("no warehouse found for city %s", cityName)
		}
	}

	// Manufacturing stats
	mfgDashboard, _ := s.mfgRepo.GetProductionDashboard(ctx, warehouse.ID)
	if mfgDashboard == nil {
		mfgDashboard = &ProductionDashboard{}
	}

	// Procurement stats
	var procDashboard ProcurementDashboard
	db.Model(&PurchaseOrder{}).Where("pharmacy_id = ? AND status = ?", warehouse.ID, "draft").Count(&procDashboard.PendingPOs)
	db.Model(&PurchaseOrder{}).Where("pharmacy_id = ? AND status = ?", warehouse.ID, "ordered").Count(&procDashboard.AwaitingGRN)
	// Reorder alerts: count low stock items for this pharmacy
	db.Model(&PharmacyInventory{}).
		Where("pharmacy_id = ? AND is_active = ? AND stock_quantity <= reorder_level", warehouse.ID, true).
		Count(&procDashboard.ReorderAlerts)

	// Raw material stats
	rmDashboard, _ := s.rmRepo.GetRawMaterialDashboard(ctx, warehouse.ID)
	if rmDashboard == nil {
		rmDashboard = &RawMaterialDashboard{}
	}

	// Inventory stats
	var invStats InventoryDashboardStats
	db.Model(&PharmacyInventory{}).Where("pharmacy_id = ? AND is_active = ?", warehouse.ID, true).
		Select("COUNT(DISTINCT medicine_id)").Scan(&invStats.TotalSKUs)
	db.Model(&PharmacyInventory{}).
		Where("pharmacy_id = ? AND is_active = ? AND stock_quantity <= reorder_level", warehouse.ID, true).
		Count(&invStats.LowStockItems)
	threshold := time.Now().AddDate(0, 1, 0) // 30 days
	db.Model(&PharmacyInventory{}).
		Where("pharmacy_id = ? AND is_active = ? AND expiry_date < ?", warehouse.ID, true, threshold).
		Count(&invStats.ExpiringItems)

	// Cold chain check — "no rows" is normal (warehouse may have no sensor
	// data yet). Use silent session to avoid log spam.
	var latestMetric ColdChainMetric
	coldChainErr := silent.Where("warehouse_id = ?", warehouse.ID).Order("recorded_at DESC").First(&latestMetric).Error
	invStats.ColdChainOK = coldChainErr != nil || (latestMetric.Temperature >= 2 && latestMetric.Temperature <= 8)

	return &CityWorkflowDashboard{
		CityName:       cityName,
		WarehouseID:    warehouse.ID,
		Manufacturing:  *mfgDashboard,
		Procurement:    procDashboard,
		RawMaterials:   *rmDashboard,
		InventoryStats: invStats,
	}, nil
}

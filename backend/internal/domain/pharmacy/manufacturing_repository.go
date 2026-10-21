package pharmacy

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ManufacturingRepository handles data access for manufacturing, BOM, and GRN operations.
type ManufacturingRepository interface {
	// BOM
	CreateBOM(ctx context.Context, bom *BillOfMaterials) error
	GetBOM(ctx context.Context, id uint) (*BillOfMaterials, error)
	GetBOMByMedicine(ctx context.Context, medicineID uint) (*BillOfMaterials, error)
	ListBOMs(ctx context.Context) ([]BillOfMaterials, error)
	UpdateBOM(ctx context.Context, bom *BillOfMaterials) error

	// Production Orders
	CreateProductionOrder(ctx context.Context, po *ProductionOrder) error
	GetProductionOrder(ctx context.Context, id uint) (*ProductionOrder, error)
	ListProductionOrders(ctx context.Context, filters ProductionOrderFilters) ([]ProductionOrder, int64, error)
	UpdateProductionOrder(ctx context.Context, po *ProductionOrder) error
	GetProductionDashboard(ctx context.Context, warehouseID uint) (*ProductionDashboard, error)

	// Manufacturing Logs
	CreateManufacturingLog(ctx context.Context, log *ManufacturingLog) error
	GetManufacturingLogs(ctx context.Context, productionOrderID uint) ([]ManufacturingLog, error)

	// Material Issuance
	CreateMaterialIssuance(ctx context.Context, issuance *MaterialIssuance) error
	GetIssuancesForOrder(ctx context.Context, productionOrderID uint) ([]MaterialIssuance, error)
	ReturnMaterial(ctx context.Context, issuanceID uint, returnedQty float64) error

	// Finished Goods
	CreateFinishedGoodsBatch(ctx context.Context, batch *FinishedGoodsBatch) error
	GetFinishedGoodsBatches(ctx context.Context, medicineID uint) ([]FinishedGoodsBatch, error)

	// GRN
	CreateGRN(ctx context.Context, grn *GoodsReceiptNote) error
	GetGRN(ctx context.Context, id uint) (*GoodsReceiptNote, error)
	ListGRNs(ctx context.Context, warehouseID uint, status string) ([]GoodsReceiptNote, error)
	UpdateGRN(ctx context.Context, grn *GoodsReceiptNote) error
	UpdateGRNItem(ctx context.Context, item *GRNItem) error

	// PO Extension
	CreatePOExtension(ctx context.Context, ext *PurchaseOrderExtension) error
	GetPOExtension(ctx context.Context, purchaseOrderID uint) (*PurchaseOrderExtension, error)
	UpdatePOExtension(ctx context.Context, ext *PurchaseOrderExtension) error

	// Transaction support
	GetDB(ctx context.Context) *gorm.DB
	ExecuteTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type ProductionOrderFilters struct {
	WarehouseID uint
	CityName    string
	Status      string
	Priority    int
	Page        int
	Limit       int
}

// manufacturingRepository implements ManufacturingRepository.
type manufacturingRepository struct {
	db *gorm.DB
}

func NewManufacturingRepository(db *gorm.DB) ManufacturingRepository {
	return &manufacturingRepository{db: db}
}

// AutoMigrateManufacturing runs migrations for all manufacturing-related tables.
func AutoMigrateManufacturing(db *gorm.DB) error {
	return db.AutoMigrate(
		&BillOfMaterials{},
		&BOMItem{},
		&ProductionOrder{},
		&ManufacturingLog{},
		&MaterialIssuance{},
		&FinishedGoodsBatch{},
		&GoodsReceiptNote{},
		&GRNItem{},
		&PurchaseOrderExtension{},
	)
}

func (r *manufacturingRepository) GetDB(ctx context.Context) *gorm.DB {
	return r.withTx(ctx)
}

func (r *manufacturingRepository) ExecuteTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		return fn(txCtx)
	})
}

type txKey struct{}

func (r *manufacturingRepository) withTx(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return r.db.WithContext(ctx)
}

// ── BOM ──

func (r *manufacturingRepository) CreateBOM(ctx context.Context, bom *BillOfMaterials) error {
	return r.withTx(ctx).Create(bom).Error
}

func (r *manufacturingRepository) GetBOM(ctx context.Context, id uint) (*BillOfMaterials, error) {
	var bom BillOfMaterials
	err := r.withTx(ctx).Preload("Items").Preload("Items.RawMaterial").Preload("Medicine").First(&bom, id).Error
	return &bom, err
}

func (r *manufacturingRepository) GetBOMByMedicine(ctx context.Context, medicineID uint) (*BillOfMaterials, error) {
	var bom BillOfMaterials
	err := r.withTx(ctx).Preload("Items").Preload("Items.RawMaterial").
		Where("medicine_id = ? AND is_active = ?", medicineID, true).
		Order("version DESC").First(&bom).Error
	return &bom, err
}

func (r *manufacturingRepository) ListBOMs(ctx context.Context) ([]BillOfMaterials, error) {
	var boms []BillOfMaterials
	err := r.withTx(ctx).Preload("Medicine").Preload("Items").
		Where("is_active = ?", true).
		Order("created_at DESC").Find(&boms).Error
	return boms, err
}

func (r *manufacturingRepository) UpdateBOM(ctx context.Context, bom *BillOfMaterials) error {
	return r.withTx(ctx).Save(bom).Error
}

// ── Production Orders ──

func (r *manufacturingRepository) CreateProductionOrder(ctx context.Context, po *ProductionOrder) error {
	return r.withTx(ctx).Create(po).Error
}

func (r *manufacturingRepository) GetProductionOrder(ctx context.Context, id uint) (*ProductionOrder, error) {
	var po ProductionOrder
	err := r.withTx(ctx).
		Preload("Medicine").Preload("BOM").Preload("BOM.Items").Preload("BOM.Items.RawMaterial").
		Preload("Warehouse").Preload("Issuances").Preload("Issuances.RawMaterial").
		Preload("Logs").
		First(&po, id).Error
	return &po, err
}

func (r *manufacturingRepository) ListProductionOrders(ctx context.Context, filters ProductionOrderFilters) ([]ProductionOrder, int64, error) {
	q := r.withTx(ctx).Preload("Medicine").Preload("Warehouse")

	if filters.WarehouseID > 0 {
		q = q.Where("warehouse_id = ?", filters.WarehouseID)
	}
	if filters.CityName != "" {
		q = q.Joins("JOIN pharmacy ON pharmacy.id = production_orders.warehouse_id").
			Where("pharmacy.city = ?", filters.CityName)
	}
	if filters.Status != "" {
		q = q.Where("status = ?", filters.Status)
	}
	if filters.Priority > 0 {
		q = q.Where("priority <= ?", filters.Priority)
	}

	var total int64
	q.Model(&ProductionOrder{}).Count(&total)

	page := filters.Page
	if page < 1 {
		page = 1
	}
	limit := filters.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var orders []ProductionOrder
	err := q.Order("priority ASC, created_at DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&orders).Error

	return orders, total, err
}

func (r *manufacturingRepository) UpdateProductionOrder(ctx context.Context, po *ProductionOrder) error {
	return r.withTx(ctx).Save(po).Error
}

func (r *manufacturingRepository) GetProductionDashboard(ctx context.Context, warehouseID uint) (*ProductionDashboard, error) {
	db := r.withTx(ctx)
	var dashboard ProductionDashboard

	q := db.Model(&ProductionOrder{})
	if warehouseID > 0 {
		q = q.Where("warehouse_id = ?", warehouseID)
	}

	// Active orders (not draft, not released, not cancelled)
	q.Where("status NOT IN ?", []string{"draft", "released", "cancelled"}).Count(&dashboard.ActiveOrders)

	// In QA review
	q2 := db.Model(&ProductionOrder{})
	if warehouseID > 0 {
		q2 = q2.Where("warehouse_id = ?", warehouseID)
	}
	q2.Where("status = ?", "qa_pending").Count(&dashboard.InQAReview)

	// Planned this week
	weekStart := time.Now().Truncate(24 * time.Hour)
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	weekEnd := weekStart.AddDate(0, 0, 7)
	q3 := db.Model(&ProductionOrder{})
	if warehouseID > 0 {
		q3 = q3.Where("warehouse_id = ?", warehouseID)
	}
	q3.Where("scheduled_date >= ? AND scheduled_date < ?", weekStart, weekEnd).Count(&dashboard.PlannedThisWeek)

	// Released today
	today := time.Now().Truncate(24 * time.Hour)
	q4 := db.Model(&ProductionOrder{})
	if warehouseID > 0 {
		q4 = q4.Where("warehouse_id = ?", warehouseID)
	}
	q4.Where("status = ? AND updated_at >= ?", "released", today).Count(&dashboard.ReleasedToday)

	return &dashboard, nil
}

// ── Manufacturing Logs ──

func (r *manufacturingRepository) CreateManufacturingLog(ctx context.Context, log *ManufacturingLog) error {
	return r.withTx(ctx).Create(log).Error
}

func (r *manufacturingRepository) GetManufacturingLogs(ctx context.Context, productionOrderID uint) ([]ManufacturingLog, error) {
	var logs []ManufacturingLog
	err := r.withTx(ctx).Where("production_order_id = ?", productionOrderID).
		Order("created_at ASC").Find(&logs).Error
	return logs, err
}

// ── Material Issuance ──

func (r *manufacturingRepository) CreateMaterialIssuance(ctx context.Context, issuance *MaterialIssuance) error {
	return r.withTx(ctx).Create(issuance).Error
}

func (r *manufacturingRepository) GetIssuancesForOrder(ctx context.Context, productionOrderID uint) ([]MaterialIssuance, error) {
	var issuances []MaterialIssuance
	err := r.withTx(ctx).Preload("RawMaterial").
		Where("production_order_id = ?", productionOrderID).Find(&issuances).Error
	return issuances, err
}

func (r *manufacturingRepository) ReturnMaterial(ctx context.Context, issuanceID uint, returnedQty float64) error {
	now := time.Now()
	return r.withTx(ctx).Model(&MaterialIssuance{}).Where("id = ?", issuanceID).
		Updates(map[string]interface{}{
			"returned_qty": gorm.Expr("returned_qty + ?", returnedQty),
			"returned_at":  &now,
		}).Error
}

// ── Finished Goods ──

func (r *manufacturingRepository) CreateFinishedGoodsBatch(ctx context.Context, batch *FinishedGoodsBatch) error {
	return r.withTx(ctx).Create(batch).Error
}

func (r *manufacturingRepository) GetFinishedGoodsBatches(ctx context.Context, medicineID uint) ([]FinishedGoodsBatch, error) {
	var batches []FinishedGoodsBatch
	q := r.withTx(ctx)
	if medicineID > 0 {
		q = q.Where("medicine_id = ?", medicineID)
	}
	err := q.Order("created_at DESC").Find(&batches).Error
	return batches, err
}

// ── GRN ──

func (r *manufacturingRepository) CreateGRN(ctx context.Context, grn *GoodsReceiptNote) error {
	return r.withTx(ctx).Create(grn).Error
}

func (r *manufacturingRepository) GetGRN(ctx context.Context, id uint) (*GoodsReceiptNote, error) {
	var grn GoodsReceiptNote
	err := r.withTx(ctx).Preload("Items").First(&grn, id).Error
	return &grn, err
}

func (r *manufacturingRepository) ListGRNs(ctx context.Context, warehouseID uint, status string) ([]GoodsReceiptNote, error) {
	q := r.withTx(ctx).Preload("Items")
	if warehouseID > 0 {
		q = q.Where("warehouse_id = ?", warehouseID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var grns []GoodsReceiptNote
	err := q.Order("created_at DESC").Limit(50).Find(&grns).Error
	return grns, err
}

func (r *manufacturingRepository) UpdateGRN(ctx context.Context, grn *GoodsReceiptNote) error {
	return r.withTx(ctx).Save(grn).Error
}

func (r *manufacturingRepository) UpdateGRNItem(ctx context.Context, item *GRNItem) error {
	return r.withTx(ctx).Save(item).Error
}

// ── PO Extension ──

func (r *manufacturingRepository) CreatePOExtension(ctx context.Context, ext *PurchaseOrderExtension) error {
	return r.withTx(ctx).Create(ext).Error
}

func (r *manufacturingRepository) GetPOExtension(ctx context.Context, purchaseOrderID uint) (*PurchaseOrderExtension, error) {
	var ext PurchaseOrderExtension
	err := r.withTx(ctx).Where("purchase_order_id = ?", purchaseOrderID).First(&ext).Error
	if err != nil {
		return nil, fmt.Errorf("PO extension not found for PO %d: %w", purchaseOrderID, err)
	}
	return &ext, nil
}

func (r *manufacturingRepository) UpdatePOExtension(ctx context.Context, ext *PurchaseOrderExtension) error {
	return r.withTx(ctx).Save(ext).Error
}

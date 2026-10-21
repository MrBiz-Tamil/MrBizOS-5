package pharmacy

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// RawMaterialRepository handles data access for raw materials, vendors, indents, and RFQs.
type RawMaterialRepository interface {
	// Raw Materials CRUD
	CreateRawMaterial(ctx context.Context, rm *RawMaterial) error
	GetRawMaterial(ctx context.Context, id uint) (*RawMaterial, error)
	ListRawMaterials(ctx context.Context, category string, search string) ([]RawMaterial, error)
	UpdateRawMaterial(ctx context.Context, rm *RawMaterial) error

	// Approved Vendors
	CreateApprovedVendor(ctx context.Context, av *ApprovedVendor) error
	GetApprovedVendors(ctx context.Context, rawMaterialID uint) ([]ApprovedVendor, error)
	GetApprovedVendor(ctx context.Context, id uint) (*ApprovedVendor, error)
	UpdateApprovedVendor(ctx context.Context, av *ApprovedVendor) error

	// Raw Material Inventory
	CreateRMInventory(ctx context.Context, inv *RawMaterialInventory) error
	GetRMInventory(ctx context.Context, id uint) (*RawMaterialInventory, error)
	ListRMInventory(ctx context.Context, warehouseID uint, rawMaterialID uint, qaStatus string) ([]RawMaterialInventory, error)
	UpdateRMInventory(ctx context.Context, inv *RawMaterialInventory) error
	GetRMBalance(ctx context.Context, rawMaterialID uint, warehouseID uint) (float64, error)
	GetLowStockRawMaterials(ctx context.Context, warehouseID uint) ([]RawMaterialInventory, error)
	GetExpiringRawMaterials(ctx context.Context, warehouseID uint, threshold time.Time) ([]RawMaterialInventory, error)
	GetRMSummary(ctx context.Context, warehouseID uint) ([]RawMaterialSummary, error)

	// Transactions (audit log)
	CreateRMTransaction(ctx context.Context, txn *RawMaterialTransaction) error
	GetRMTransactions(ctx context.Context, inventoryID uint) ([]RawMaterialTransaction, error)

	// Indents
	CreateIndent(ctx context.Context, indent *RawMaterialIndent) error
	GetIndent(ctx context.Context, id uint) (*RawMaterialIndent, error)
	ListIndents(ctx context.Context, warehouseID uint, status string) ([]RawMaterialIndent, error)
	UpdateIndent(ctx context.Context, indent *RawMaterialIndent) error

	// RFQ
	CreateRFQ(ctx context.Context, rfq *RFQ) error
	GetRFQ(ctx context.Context, id uint) (*RFQ, error)
	ListRFQs(ctx context.Context, status string) ([]RFQ, error)
	UpdateRFQ(ctx context.Context, rfq *RFQ) error

	// Vendor Quotations
	CreateQuotation(ctx context.Context, q *VendorQuotation) error
	GetQuotation(ctx context.Context, id uint) (*VendorQuotation, error)
	GetQuotationsForRFQ(ctx context.Context, rfqID uint) ([]VendorQuotation, error)
	UpdateQuotation(ctx context.Context, q *VendorQuotation) error

	// Dashboard
	GetRawMaterialDashboard(ctx context.Context, warehouseID uint) (*RawMaterialDashboard, error)

	// Transaction support
	GetDB(ctx context.Context) *gorm.DB
	ExecuteTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type rawMaterialRepository struct {
	db *gorm.DB
}

func NewRawMaterialRepository(db *gorm.DB) RawMaterialRepository {
	return &rawMaterialRepository{db: db}
}

// AutoMigrateRawMaterial runs migrations for all raw material tables.
func AutoMigrateRawMaterial(db *gorm.DB) error {
	return db.AutoMigrate(
		&RawMaterial{},
		&ApprovedVendor{},
		&RawMaterialInventory{},
		&RawMaterialTransaction{},
		&RawMaterialIndent{},
		&RFQ{},
		&VendorQuotation{},
	)
}

func (r *rawMaterialRepository) GetDB(ctx context.Context) *gorm.DB {
	return r.withTx(ctx)
}

func (r *rawMaterialRepository) ExecuteTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		return fn(txCtx)
	})
}

func (r *rawMaterialRepository) withTx(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return r.db.WithContext(ctx)
}

// ── Raw Materials ──

func (r *rawMaterialRepository) CreateRawMaterial(ctx context.Context, rm *RawMaterial) error {
	return r.withTx(ctx).Create(rm).Error
}

func (r *rawMaterialRepository) GetRawMaterial(ctx context.Context, id uint) (*RawMaterial, error) {
	var rm RawMaterial
	err := r.withTx(ctx).First(&rm, id).Error
	return &rm, err
}

func (r *rawMaterialRepository) ListRawMaterials(ctx context.Context, category string, search string) ([]RawMaterial, error) {
	q := r.withTx(ctx).Where("is_active = ?", true)
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("name LIKE ? OR generic_name LIKE ? OR cas_number LIKE ?", like, like, like)
	}
	var materials []RawMaterial
	err := q.Order("name ASC").Find(&materials).Error
	return materials, err
}

func (r *rawMaterialRepository) UpdateRawMaterial(ctx context.Context, rm *RawMaterial) error {
	return r.withTx(ctx).Save(rm).Error
}

// ── Approved Vendors ──

func (r *rawMaterialRepository) CreateApprovedVendor(ctx context.Context, av *ApprovedVendor) error {
	return r.withTx(ctx).Create(av).Error
}

func (r *rawMaterialRepository) GetApprovedVendors(ctx context.Context, rawMaterialID uint) ([]ApprovedVendor, error) {
	q := r.withTx(ctx).Preload("Supplier").Where("is_active = ?", true)
	if rawMaterialID > 0 {
		q = q.Where("raw_material_id = ?", rawMaterialID)
	}
	var vendors []ApprovedVendor
	err := q.Order("is_preferred DESC, rating DESC").Find(&vendors).Error
	return vendors, err
}

func (r *rawMaterialRepository) GetApprovedVendor(ctx context.Context, id uint) (*ApprovedVendor, error) {
	var av ApprovedVendor
	err := r.withTx(ctx).Preload("Supplier").Preload("RawMaterial").First(&av, id).Error
	return &av, err
}

func (r *rawMaterialRepository) UpdateApprovedVendor(ctx context.Context, av *ApprovedVendor) error {
	return r.withTx(ctx).Save(av).Error
}

// ── Raw Material Inventory ──

func (r *rawMaterialRepository) CreateRMInventory(ctx context.Context, inv *RawMaterialInventory) error {
	return r.withTx(ctx).Create(inv).Error
}

func (r *rawMaterialRepository) GetRMInventory(ctx context.Context, id uint) (*RawMaterialInventory, error) {
	var inv RawMaterialInventory
	err := r.withTx(ctx).Preload("RawMaterial").First(&inv, id).Error
	return &inv, err
}

func (r *rawMaterialRepository) ListRMInventory(ctx context.Context, warehouseID uint, rawMaterialID uint, qaStatus string) ([]RawMaterialInventory, error) {
	q := r.withTx(ctx).Preload("RawMaterial").Where("is_active = ?", true)
	if warehouseID > 0 {
		q = q.Where("warehouse_id = ?", warehouseID)
	}
	if rawMaterialID > 0 {
		q = q.Where("raw_material_id = ?", rawMaterialID)
	}
	if qaStatus != "" {
		q = q.Where("qa_status = ?", qaStatus)
	}
	var inventory []RawMaterialInventory
	err := q.Order("expiry_date ASC").Find(&inventory).Error
	return inventory, err
}

func (r *rawMaterialRepository) UpdateRMInventory(ctx context.Context, inv *RawMaterialInventory) error {
	return r.withTx(ctx).Save(inv).Error
}

func (r *rawMaterialRepository) GetRMBalance(ctx context.Context, rawMaterialID uint, warehouseID uint) (float64, error) {
	var total float64
	q := r.withTx(ctx).Model(&RawMaterialInventory{}).
		Where("raw_material_id = ? AND is_active = ? AND qa_status = ?", rawMaterialID, true, "approved")
	if warehouseID > 0 {
		q = q.Where("warehouse_id = ?", warehouseID)
	}
	err := q.Select("COALESCE(SUM(quantity), 0)").Scan(&total).Error
	return total, err
}

func (r *rawMaterialRepository) GetLowStockRawMaterials(ctx context.Context, warehouseID uint) ([]RawMaterialInventory, error) {
	q := r.withTx(ctx).Preload("RawMaterial").
		Where("is_active = ? AND qa_status = ? AND quantity <= reorder_level", true, "approved")
	if warehouseID > 0 {
		q = q.Where("warehouse_id = ?", warehouseID)
	}
	var inventory []RawMaterialInventory
	err := q.Find(&inventory).Error
	return inventory, err
}

func (r *rawMaterialRepository) GetExpiringRawMaterials(ctx context.Context, warehouseID uint, threshold time.Time) ([]RawMaterialInventory, error) {
	q := r.withTx(ctx).Preload("RawMaterial").
		Where("is_active = ? AND expiry_date < ?", true, threshold)
	if warehouseID > 0 {
		q = q.Where("warehouse_id = ?", warehouseID)
	}
	var inventory []RawMaterialInventory
	err := q.Order("expiry_date ASC").Find(&inventory).Error
	return inventory, err
}

func (r *rawMaterialRepository) GetRMSummary(ctx context.Context, warehouseID uint) ([]RawMaterialSummary, error) {
	q := r.withTx(ctx).Table("raw_material_inventory").
		Select(`
			raw_material_inventory.raw_material_id,
			raw_materials.name,
			raw_materials.category,
			SUM(raw_material_inventory.quantity) as total_qty,
			raw_materials.default_uom as unit_of_measure,
			SUM(CASE WHEN raw_material_inventory.qa_status = 'approved' THEN raw_material_inventory.quantity ELSE 0 END) as qa_approved_qty,
			SUM(CASE WHEN raw_material_inventory.qa_status = 'quarantine' THEN raw_material_inventory.quantity ELSE 0 END) as quarantine_qty,
			COUNT(DISTINCT raw_material_inventory.warehouse_id) as warehouse_count,
			MIN(raw_material_inventory.expiry_date) as nearest_expiry
		`).
		Joins("JOIN raw_materials ON raw_materials.id = raw_material_inventory.raw_material_id").
		Where("raw_material_inventory.is_active = ?", true)

	if warehouseID > 0 {
		q = q.Where("raw_material_inventory.warehouse_id = ?", warehouseID)
	}

	q = q.Group("raw_material_inventory.raw_material_id, raw_materials.name, raw_materials.category, raw_materials.default_uom")

	var summaries []RawMaterialSummary
	err := q.Scan(&summaries).Error
	return summaries, err
}

// ── Transactions ──

func (r *rawMaterialRepository) CreateRMTransaction(ctx context.Context, txn *RawMaterialTransaction) error {
	return r.withTx(ctx).Create(txn).Error
}

func (r *rawMaterialRepository) GetRMTransactions(ctx context.Context, inventoryID uint) ([]RawMaterialTransaction, error) {
	var txns []RawMaterialTransaction
	err := r.withTx(ctx).Where("inventory_id = ?", inventoryID).
		Order("created_at DESC").Find(&txns).Error
	return txns, err
}

// ── Indents ──

func (r *rawMaterialRepository) CreateIndent(ctx context.Context, indent *RawMaterialIndent) error {
	return r.withTx(ctx).Create(indent).Error
}

func (r *rawMaterialRepository) GetIndent(ctx context.Context, id uint) (*RawMaterialIndent, error) {
	var indent RawMaterialIndent
	err := r.withTx(ctx).Preload("RawMaterial").First(&indent, id).Error
	return &indent, err
}

func (r *rawMaterialRepository) ListIndents(ctx context.Context, warehouseID uint, status string) ([]RawMaterialIndent, error) {
	q := r.withTx(ctx).Preload("RawMaterial")
	if warehouseID > 0 {
		q = q.Where("warehouse_id = ?", warehouseID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var indents []RawMaterialIndent
	err := q.Order("priority ASC, created_at DESC").Find(&indents).Error
	return indents, err
}

func (r *rawMaterialRepository) UpdateIndent(ctx context.Context, indent *RawMaterialIndent) error {
	return r.withTx(ctx).Save(indent).Error
}

// ── RFQ ──

func (r *rawMaterialRepository) CreateRFQ(ctx context.Context, rfq *RFQ) error {
	return r.withTx(ctx).Create(rfq).Error
}

func (r *rawMaterialRepository) GetRFQ(ctx context.Context, id uint) (*RFQ, error) {
	var rfq RFQ
	err := r.withTx(ctx).Preload("RawMaterial").Preload("Quotations").Preload("Quotations.Supplier").
		First(&rfq, id).Error
	return &rfq, err
}

func (r *rawMaterialRepository) ListRFQs(ctx context.Context, status string) ([]RFQ, error) {
	q := r.withTx(ctx).Preload("RawMaterial").Preload("Quotations")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var rfqs []RFQ
	err := q.Order("created_at DESC").Limit(50).Find(&rfqs).Error
	return rfqs, err
}

func (r *rawMaterialRepository) UpdateRFQ(ctx context.Context, rfq *RFQ) error {
	return r.withTx(ctx).Save(rfq).Error
}

// ── Quotations ──

func (r *rawMaterialRepository) CreateQuotation(ctx context.Context, q *VendorQuotation) error {
	return r.withTx(ctx).Create(q).Error
}

func (r *rawMaterialRepository) GetQuotation(ctx context.Context, id uint) (*VendorQuotation, error) {
	var q VendorQuotation
	err := r.withTx(ctx).Preload("Supplier").First(&q, id).Error
	return &q, err
}

func (r *rawMaterialRepository) GetQuotationsForRFQ(ctx context.Context, rfqID uint) ([]VendorQuotation, error) {
	var quotations []VendorQuotation
	err := r.withTx(ctx).Preload("Supplier").
		Where("rfq_id = ?", rfqID).
		Order("unit_price ASC").Find(&quotations).Error
	return quotations, err
}

func (r *rawMaterialRepository) UpdateQuotation(ctx context.Context, q *VendorQuotation) error {
	return r.withTx(ctx).Save(q).Error
}

// ── Dashboard ──

func (r *rawMaterialRepository) GetRawMaterialDashboard(ctx context.Context, warehouseID uint) (*RawMaterialDashboard, error) {
	db := r.withTx(ctx)
	var dashboard RawMaterialDashboard

	// Open indents
	iq := db.Model(&RawMaterialIndent{}).Where("status IN ?", []string{"pending", "approved"})
	if warehouseID > 0 {
		iq = iq.Where("warehouse_id = ?", warehouseID)
	}
	iq.Count(&dashboard.OpenIndents)

	// Pending RFQs
	db.Model(&RFQ{}).Where("status = ?", "open").Count(&dashboard.RFQsPending)

	// In QA lab (quarantine inventory)
	qq := db.Model(&RawMaterialInventory{}).Where("qa_status = ? AND is_active = ?", "quarantine", true)
	if warehouseID > 0 {
		qq = qq.Where("warehouse_id = ?", warehouseID)
	}
	qq.Count(&dashboard.InQALab)

	// Total active materials
	db.Model(&RawMaterial{}).Where("is_active = ?", true).Count(&dashboard.TotalMaterials)

	// Low stock
	lq := db.Model(&RawMaterialInventory{}).Where("is_active = ? AND qa_status = ? AND quantity <= reorder_level", true, "approved")
	if warehouseID > 0 {
		lq = lq.Where("warehouse_id = ?", warehouseID)
	}
	lq.Count(&dashboard.LowStockMaterials)

	return &dashboard, nil
}

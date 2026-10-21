package pharmacy

import (
	"context"
	"time"

	"gorm.io/gorm"
	"drayya-core/internal/database"
)

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&Pharmacy{},
		&Medicine{},
		&ProductVariant{},
		&Prescription{},
		&PharmacyInventory{},
		&PharmacyOrder{},
		&PharmacyOrderItem{},
		&InventoryTransfer{},
		&Supplier{},
		&PurchaseOrder{},
		&CourierDelivery{},
		&ColdChainMetric{},
		&CourierProfile{},
		&DeliveryZone{},
		&Shipment{},
		&ShipmentUpdate{},
		&ProductInteraction{},
		&ProductSalesSummary{},
		&CartSession{},
		&DrugInteraction{},
		&DispenseRecord{},
		&ProductReview{},
		&Coupon{},
		&ProductBundle{},
		&StockReservation{},
	); err != nil {
		return err
	}

	// One-time backfill for the catalog_status / sales_channel columns. Rows that
	// existed before these columns were added land with an empty value; publish
	// them on the public channel so the storefront keeps showing the catalog that
	// was already live. New medicines get "draft"/"public" via BeforeCreate, so
	// this WHERE clause never touches genuine drafts and is safe to re-run.
	if err := db.Model(&Medicine{}).
		Where("catalog_status IS NULL OR catalog_status = ''").
		Update("catalog_status", CatalogPublished).Error; err != nil {
		return err
	}
	if err := db.Model(&Medicine{}).
		Where("sales_channel IS NULL OR sales_channel = ''").
		Update("sales_channel", ChannelPublic).Error; err != nil {
		return err
	}
	// Existing published rows should have a published_at so the "first publish"
	// stock gate doesn't re-trigger for already-live medicines.
	if err := db.Model(&Medicine{}).
		Where("catalog_status = ? AND published_at IS NULL", CatalogPublished).
		Update("published_at", time.Now()).Error; err != nil {
		return err
	}
	return nil
}

type Repository interface {
	GetMedicines(ctx context.Context) ([]Medicine, error)
	GetMedicine(ctx context.Context, id uint) (*Medicine, error)
	CreateMedicine(ctx context.Context, med *Medicine) error
	GetInventory(ctx context.Context, pharmacyID uint) ([]PharmacyInventory, error)
	GetAllInventory(ctx context.Context) ([]PharmacyInventory, error)
	GetLowStockInventory(ctx context.Context) ([]PharmacyInventory, error)
	GetPharmacyByID(ctx context.Context, id uint) (*Pharmacy, error)
	GetAllPharmacies(ctx context.Context, onlyActive bool) ([]Pharmacy, error)
	GetSupplier(ctx context.Context) (*Supplier, error)
	GetCentralWarehouse(ctx context.Context) (*Pharmacy, error)
	CreatePurchaseOrder(ctx context.Context, po *PurchaseOrder) error
	GetPendingTransfer(ctx context.Context, destinationID, medicineID uint) (*InventoryTransfer, error)
	CreateTransfer(ctx context.Context, transfer *InventoryTransfer) error
	GetExpiringInventory(ctx context.Context, threshold time.Time) ([]PharmacyInventory, error)
	CreateOrder(ctx context.Context, order *PharmacyOrder) error
	GetOrderWithPharmacy(ctx context.Context, id uint) (*PharmacyOrder, error)
	UpdateOrder(ctx context.Context, order *PharmacyOrder) error
	GetOrder(ctx context.Context, id uint) (*PharmacyOrder, error)
	FindInventoryByMedicineName(ctx context.Context, name string) (*PharmacyInventory, error)
	GetPurchaseOrder(ctx context.Context, id uint) (*PurchaseOrder, error)
	UpdatePurchaseOrder(ctx context.Context, po *PurchaseOrder) error
	// Prescription logic would ideally move to EMR, keeping loosely coupled for now
	GetPrescription(ctx context.Context, id uint) (interface{}, error)
	GetRxRecord(ctx context.Context, id uint) (*Prescription, error)
	ExecuteTransaction(ctx context.Context, fn func(ctx context.Context, txRepo Repository) error) error
	FindBatchesForMed(ctx context.Context, pharmacyID, medicineID uint) ([]PharmacyInventory, error)
	UpdateInventoryBatch(ctx context.Context, batch *PharmacyInventory) error
	GetMultiCityInventory(ctx context.Context) ([]CityStock, error)
	GetPaidOrders(ctx context.Context, pharmacyID uint) ([]PharmacyOrder, error)
	GetPendingPaymentAmount(ctx context.Context, pharmacyID uint) (int64, error)
	CreatePharmacy(ctx context.Context, p *Pharmacy) error
	UpdatePharmacy(ctx context.Context, p *Pharmacy) error
	CreateColdChainMetric(ctx context.Context, metric *ColdChainMetric) error
	GetWarehouseMetrics(ctx context.Context, warehouseID uint) ([]ColdChainMetric, error)
	GetDB(ctx context.Context) *gorm.DB
}

type defaultRepository struct {
	db *gorm.DB
}

func NewDefaultRepository(db *gorm.DB) Repository {
	return &defaultRepository{db: db}
}

func (r *defaultRepository) withTx(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(database.TxKey{}).(*gorm.DB); ok {
		return tx
	}
	return r.db.WithContext(ctx)
}

// Using shared database.TxKey now

func (r *defaultRepository) GetMedicines(ctx context.Context) ([]Medicine, error) {
	var medicines []Medicine
	err := r.withTx(ctx).Find(&medicines).Error
	return medicines, err
}

func (r *defaultRepository) GetMedicine(ctx context.Context, id uint) (*Medicine, error) {
	var m Medicine
	err := r.withTx(ctx).First(&m, id).Error
	return &m, err
}

func (r *defaultRepository) CreateMedicine(ctx context.Context, med *Medicine) error {
	return r.withTx(ctx).Create(med).Error
}

func (r *defaultRepository) ExecuteTransaction(ctx context.Context, fn func(ctx context.Context, txRepo Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, database.TxKey{}, tx)
		txRepo := &defaultRepository{db: tx.WithContext(txCtx)}
		return fn(txCtx, txRepo)
	})
}

func (r *defaultRepository) GetInventory(ctx context.Context, pharmacyID uint) ([]PharmacyInventory, error) {
	var inventory []PharmacyInventory
	db := r.withTx(ctx).Preload("Medicine")
	if pharmacyID != 0 {
		db = db.Where("pharmacy_id = ?", pharmacyID)
	}
	err := db.Where("is_active = ?", true).Find(&inventory).Error
	return inventory, err
}

func (r *defaultRepository) GetAllInventory(ctx context.Context) ([]PharmacyInventory, error) {
	var inventory []PharmacyInventory
	err := r.withTx(ctx).Preload("Medicine").Where("is_active = ?", true).Find(&inventory).Error
	return inventory, err
}

func (r *defaultRepository) GetLowStockInventory(ctx context.Context) ([]PharmacyInventory, error) {
	var inventory []PharmacyInventory
	err := r.withTx(ctx).Preload("Medicine").
		Where("is_active = ? AND stock_quantity <= reorder_level", true).
		Find(&inventory).Error
	return inventory, err
}

func (r *defaultRepository) GetPharmacyByID(ctx context.Context, id uint) (*Pharmacy, error) {
	var p Pharmacy
	err := r.withTx(ctx).First(&p, id).Error
	return &p, err
}

func (r *defaultRepository) GetAllPharmacies(ctx context.Context, onlyActive bool) ([]Pharmacy, error) {
	var p []Pharmacy
	db := r.withTx(ctx)
	if onlyActive {
		db = db.Where("is_active = ?", true)
	}
	err := db.Find(&p).Error
	return p, err
}

func (r *defaultRepository) GetSupplier(ctx context.Context) (*Supplier, error) {
	var s Supplier
	err := r.withTx(ctx).Where("is_active = ?", true).First(&s).Error
	return &s, err
}

func (r *defaultRepository) GetCentralWarehouse(ctx context.Context) (*Pharmacy, error) {
	var p Pharmacy
	err := r.withTx(ctx).Where("is_central_warehouse = ? AND is_active = ?", true, true).First(&p).Error
	return &p, err
}

func (r *defaultRepository) CreatePurchaseOrder(ctx context.Context, po *PurchaseOrder) error {
	return r.withTx(ctx).Create(po).Error
}

func (r *defaultRepository) GetPendingTransfer(ctx context.Context, destinationID, medicineID uint) (*InventoryTransfer, error) {
	var t InventoryTransfer
	err := r.withTx(ctx).Where("destination_id = ? AND medicine_id = ? AND status IN ?",
		destinationID, medicineID, []string{"requested", "in_transit"}).First(&t).Error
	return &t, err
}

func (r *defaultRepository) CreateTransfer(ctx context.Context, transfer *InventoryTransfer) error {
	return r.withTx(ctx).Create(transfer).Error
}

func (r *defaultRepository) GetExpiringInventory(ctx context.Context, threshold time.Time) ([]PharmacyInventory, error) {
	var inv []PharmacyInventory
	err := r.withTx(ctx).Preload("Medicine").
		Where("is_active = ? AND expiry_date < ?", true, threshold).
		Find(&inv).Error
	return inv, err
}

func (r *defaultRepository) CreateOrder(ctx context.Context, order *PharmacyOrder) error {
	return r.withTx(ctx).Create(order).Error
}

func (r *defaultRepository) GetOrderWithPharmacy(ctx context.Context, id uint) (*PharmacyOrder, error) {
	var o PharmacyOrder
	err := r.withTx(ctx).Preload("Pharmacy").First(&o, id).Error
	return &o, err
}

func (r *defaultRepository) GetOrder(ctx context.Context, id uint) (*PharmacyOrder, error) {
	var o PharmacyOrder
	err := r.withTx(ctx).First(&o, id).Error
	return &o, err
}

func (r *defaultRepository) UpdateOrder(ctx context.Context, order *PharmacyOrder) error {
	return r.withTx(ctx).Save(order).Error
}

func (r *defaultRepository) FindInventoryByMedicineName(ctx context.Context, name string) (*PharmacyInventory, error) {
	var inv PharmacyInventory
	err := r.withTx(ctx).Preload("Medicine").
		Joins("JOIN medicine ON medicine.id = pharmacyinventory.medicine_id").
		Where("medicine.name ILIKE ?", "%"+name+"%").
		First(&inv).Error
	return &inv, err
}

func (r *defaultRepository) GetPurchaseOrder(ctx context.Context, id uint) (*PurchaseOrder, error) {
	var po PurchaseOrder
	err := r.withTx(ctx).First(&po, id).Error
	return &po, err
}

func (r *defaultRepository) UpdatePurchaseOrder(ctx context.Context, po *PurchaseOrder) error {
	return r.withTx(ctx).Save(po).Error
}


func (r *defaultRepository) GetRxRecord(ctx context.Context, id uint) (*Prescription, error) {
	var rx Prescription
	err := r.withTx(ctx).First(&rx, id).Error
	return &rx, err
}

func (r *defaultRepository) GetPrescription(ctx context.Context, id uint) (interface{}, error) {
	// Dynamically handle prescription to avoid hard dependency on monolithic models
	var p struct {
		ID        uint
		UserID    uint `gorm:"column:user_id"`
		Medicines []byte
	}
	err := r.withTx(ctx).Table("prescriptions").First(&p, id).Error
	return &p, err
}

func (r *defaultRepository) FindBatchesForMed(ctx context.Context, pharmacyID, medicineID uint) ([]PharmacyInventory, error) {
	var batches []PharmacyInventory
	err := r.withTx(ctx).Where("pharmacy_id = ? AND medicine_id = ? AND stock_quantity > 0 AND is_active = ?",
		pharmacyID, medicineID, true).Order("expiry_date ASC").Find(&batches).Error
	return batches, err
}

func (r *defaultRepository) UpdateInventoryBatch(ctx context.Context, batch *PharmacyInventory) error {
	return r.withTx(ctx).Save(batch).Error
}

type CityStock struct {
	City       string `json:"city"`
	MedID      uint   `json:"med_id"`
	MedName    string `json:"med_name"`
	TotalStock int    `json:"total_stock"`
}

func (r *defaultRepository) GetMultiCityInventory(ctx context.Context) ([]CityStock, error) {
	var results []CityStock
	err := r.withTx(ctx).Table("pharmacyinventory").
		Select("pharmacy.city, medicine.id as med_id, medicine.name as med_name, SUM(pharmacyinventory.stock_quantity) as total_stock").
		Joins("JOIN pharmacy ON pharmacy.id = pharmacyinventory.pharmacy_id").
		Joins("JOIN medicine ON medicine.id = pharmacyinventory.medicine_id").
		Group("pharmacy.city, medicine.id, medicine.name").
		Scan(&results).Error
	return results, err
}

func (r *defaultRepository) GetPaidOrders(ctx context.Context, pharmacyID uint) ([]PharmacyOrder, error) {
	var orders []PharmacyOrder
	query := r.withTx(ctx).Where("payment_status = ?", "paid")
	if pharmacyID != 0 {
		query = query.Where("pharmacy_id = ?", pharmacyID)
	}
	err := query.Find(&orders).Error
	return orders, err
}

// GetPendingPaymentAmount returns the total pending payment amount.
// M-13/15: checks the result of Scan and returns any error.
func (r *defaultRepository) GetPendingPaymentAmount(ctx context.Context, pharmacyID uint) (int64, error) {
	var pendingAmount int64
	query := r.withTx(ctx).Model(&PharmacyOrder{}).Where("payment_status = ?", "pending")
	if pharmacyID != 0 {
		query = query.Where("pharmacy_id = ?", pharmacyID)
	}
	if err := query.Select("COALESCE(SUM(total_amount), 0)").Scan(&pendingAmount).Error; err != nil {
		return 0, err
	}
	return pendingAmount, nil
}

func (r *defaultRepository) CreatePharmacy(ctx context.Context, p *Pharmacy) error {
	return r.withTx(ctx).Create(p).Error
}

func (r *defaultRepository) UpdatePharmacy(ctx context.Context, p *Pharmacy) error {
	return r.withTx(ctx).Save(p).Error
}

func (r *defaultRepository) CreateColdChainMetric(ctx context.Context, metric *ColdChainMetric) error {
	return r.withTx(ctx).Create(metric).Error
}

func (r *defaultRepository) GetWarehouseMetrics(ctx context.Context, warehouseID uint) ([]ColdChainMetric, error) {
	var metrics []ColdChainMetric
	err := r.withTx(ctx).Where("warehouse_id = ?", warehouseID).Order("recorded_at desc").Limit(100).Find(&metrics).Error
	return metrics, err
}
func (r *defaultRepository) GetDB(ctx context.Context) *gorm.DB {
	return r.withTx(ctx)
}

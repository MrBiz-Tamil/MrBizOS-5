package finance

import (
	"context"
	"gorm.io/gorm"
)

type Repository interface {
	CreateCostCenter(cc *CostCenter) error
	ListCostCenters() ([]CostCenter, error)
	CreateEntry(entry *FinancialLedgerEntry) error
	ListEntries(buID uint) ([]FinancialLedgerEntry, error)
	GetBUSummary(buID uint) (*BusinessUnitFinancialSummary, error)
	GetAccountByName(ctx context.Context, name string) (*LedgerAccount, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateCostCenter(cc *CostCenter) error {
	return r.db.Create(cc).Error
}

func (r *repository) GetAccountByName(ctx context.Context, name string) (*LedgerAccount, error) {
	// Dummy implementation for now
	return &LedgerAccount{Name: name}, nil
}

func (r *repository) ListCostCenters() ([]CostCenter, error) {
	var ccs []CostCenter
	err := r.db.Where("status = ?", "ACTIVE").Order("code ASC").Find(&ccs).Error
	return ccs, err
}

func (r *repository) CreateEntry(entry *FinancialLedgerEntry) error {
	return r.db.Create(entry).Error
}

func (r *repository) ListEntries(buID uint) ([]FinancialLedgerEntry, error) {
	var entries []FinancialLedgerEntry
	q := r.db.Model(&FinancialLedgerEntry{})
	if buID > 0 {
		q = q.Where("business_unit_id = ?", buID)
	}
	err := q.Order("entry_date DESC").Find(&entries).Error
	return entries, err
}

func (r *repository) GetBUSummary(buID uint) (*BusinessUnitFinancialSummary, error) {
	summary := &BusinessUnitFinancialSummary{
		BusinessUnitID: buID,
	}

	var rev float64
	r.db.Model(&FinancialLedgerEntry{}).
		Where("business_unit_id = ? AND entry_type = ?", buID, EntryRevenue).
		Select("COALESCE(SUM(amount), 0)").Scan(&rev)
	summary.GrossRevenue = rev

	var cogs float64
	r.db.Model(&FinancialLedgerEntry{}).
		Where("business_unit_id = ? AND entry_type = ?", buID, EntryCOGS).
		Select("COALESCE(SUM(amount), 0)").Scan(&cogs)
	summary.COGS = cogs

	var ad float64
	r.db.Model(&FinancialLedgerEntry{}).
		Where("business_unit_id = ? AND entry_type = ?", buID, EntryAdvertising).
		Select("COALESCE(SUM(amount), 0)").Scan(&ad)
	summary.AdSpend = ad

	var exp float64
	r.db.Model(&FinancialLedgerEntry{}).
		Where("business_unit_id = ? AND entry_type = ?", buID, EntryExpense).
		Select("COALESCE(SUM(amount), 0)").Scan(&exp)
	summary.OperatingExpenses = exp

	summary.ContributionMargin = summary.GrossRevenue - (summary.COGS + summary.AdSpend + summary.OperatingExpenses)
	if summary.GrossRevenue > 0 {
		summary.MarginPct = (summary.ContributionMargin / summary.GrossRevenue) * 100.0
	}

	return summary, nil
}

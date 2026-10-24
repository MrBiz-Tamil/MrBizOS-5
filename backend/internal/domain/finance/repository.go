package finance

import (
	"context"
	"drayya-core/internal/database"
	"gorm.io/gorm"
)

type Repository interface {
	DB(ctx ...context.Context) *gorm.DB
	CreateCostCenter(cc *CostCenter) error
	ListCostCenters() ([]CostCenter, error)
	CreateEntry(entry *FinancialLedgerEntry) error
	ListEntries(buID uint) ([]FinancialLedgerEntry, error)
	GetBUSummary(buID uint) (*BusinessUnitFinancialSummary, error)
	GetAccountByName(ctx context.Context, name string) (*LedgerAccount, error)
	CreateAccount(ctx context.Context, acct *LedgerAccount) error
	SaveAccount(ctx context.Context, acct *LedgerAccount) error
	CreateTaxRule(ctx context.Context, rule *TaxRule) error
	GetTaxRule(ctx context.Context, region, category string) (*TaxRule, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) withTx(ctx context.Context) *gorm.DB {
	if ctx != nil {
		if tx, ok := ctx.Value(database.TxKey{}).(*gorm.DB); ok {
			return tx
		}
		return r.db.WithContext(ctx)
	}
	return r.db
}

func (r *repository) DB(ctx ...context.Context) *gorm.DB {
	if len(ctx) > 0 && ctx[0] != nil {
		return r.withTx(ctx[0])
	}
	return r.db
}

func (r *repository) CreateCostCenter(cc *CostCenter) error {
	return r.db.Create(cc).Error
}

func (r *repository) GetAccountByName(ctx context.Context, name string) (*LedgerAccount, error) {
	var acct LedgerAccount
	err := r.withTx(ctx).Where("name = ?", name).First(&acct).Error
	if err != nil {
		return nil, err
	}
	return &acct, nil
}

func (r *repository) CreateAccount(ctx context.Context, acct *LedgerAccount) error {
	return r.withTx(ctx).Create(acct).Error
}

func (r *repository) SaveAccount(ctx context.Context, acct *LedgerAccount) error {
	return r.withTx(ctx).Save(acct).Error
}

func (r *repository) CreateTaxRule(ctx context.Context, rule *TaxRule) error {
	return r.withTx(ctx).Create(rule).Error
}

func (r *repository) GetTaxRule(ctx context.Context, region, category string) (*TaxRule, error) {
	var rule TaxRule
	err := r.withTx(ctx).Where("region = ? AND category = ?", region, category).First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
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

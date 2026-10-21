package budget

import "gorm.io/gorm"

// Repository defines persistence operations for the budget domain.
type Repository interface {
	// BudgetPeriod
	CreatePeriod(p *BudgetPeriod) error
	ListPeriods() ([]BudgetPeriod, error)
	GetPeriod(id uint) (*BudgetPeriod, error)

	// BudgetLine
	CreateLine(line *BudgetLine) error
	ListLines(periodID uint) ([]BudgetLine, error)
	GetLine(id uint) (*BudgetLine, error)
	SaveLine(line *BudgetLine) error
	IncrementCommitted(lineID uint, amountPaise int64) error
	MoveCommittedToSpent(lineID uint, amountPaise int64) error
	DecrementCommitted(lineID uint, amountPaise int64) error

	// BudgetExpense
	CreateExpense(exp *BudgetExpense) error
	ListExpenses(lineID uint) ([]BudgetExpense, error)
	GetExpense(id uint) (*BudgetExpense, error)
	SaveExpense(exp *BudgetExpense) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository returns a GORM-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// ── BudgetPeriod ──────────────────────────────────────────────────────────────

func (r *repository) CreatePeriod(p *BudgetPeriod) error {
	return r.db.Create(p).Error
}

func (r *repository) ListPeriods() ([]BudgetPeriod, error) {
	var periods []BudgetPeriod
	err := r.db.Order("start_date DESC").Find(&periods).Error
	return periods, err
}

func (r *repository) GetPeriod(id uint) (*BudgetPeriod, error) {
	var p BudgetPeriod
	err := r.db.First(&p, id).Error
	return &p, err
}

// ── BudgetLine ────────────────────────────────────────────────────────────────

func (r *repository) CreateLine(line *BudgetLine) error {
	return r.db.Create(line).Error
}

func (r *repository) ListLines(periodID uint) ([]BudgetLine, error) {
	var lines []BudgetLine
	err := r.db.Where("period_id = ?", periodID).Find(&lines).Error
	return lines, err
}

func (r *repository) GetLine(id uint) (*BudgetLine, error) {
	var line BudgetLine
	err := r.db.First(&line, id).Error
	return &line, err
}

func (r *repository) SaveLine(line *BudgetLine) error {
	return r.db.Save(line).Error
}

func (r *repository) IncrementCommitted(lineID uint, amountPaise int64) error {
	return r.db.Model(&BudgetLine{}).Where("id = ?", lineID).
		UpdateColumn("committed_paise", gorm.Expr("committed_paise + ?", amountPaise)).Error
}

func (r *repository) MoveCommittedToSpent(lineID uint, amountPaise int64) error {
	return r.db.Model(&BudgetLine{}).Where("id = ?", lineID).
		Updates(map[string]interface{}{
			"spent_paise":     gorm.Expr("spent_paise + ?", amountPaise),
			"committed_paise": gorm.Expr("committed_paise - ?", amountPaise),
		}).Error
}

func (r *repository) DecrementCommitted(lineID uint, amountPaise int64) error {
	return r.db.Model(&BudgetLine{}).Where("id = ?", lineID).
		UpdateColumn("committed_paise", gorm.Expr("committed_paise - ?", amountPaise)).Error
}

// ── BudgetExpense ─────────────────────────────────────────────────────────────

func (r *repository) CreateExpense(exp *BudgetExpense) error {
	return r.db.Create(exp).Error
}

func (r *repository) ListExpenses(lineID uint) ([]BudgetExpense, error) {
	var expenses []BudgetExpense
	err := r.db.Where("line_id = ?", lineID).Order("expense_date DESC").Find(&expenses).Error
	return expenses, err
}

func (r *repository) GetExpense(id uint) (*BudgetExpense, error) {
	var exp BudgetExpense
	err := r.db.First(&exp, id).Error
	return &exp, err
}

func (r *repository) SaveExpense(exp *BudgetExpense) error {
	return r.db.Save(exp).Error
}

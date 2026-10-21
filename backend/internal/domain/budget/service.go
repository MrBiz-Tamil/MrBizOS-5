package budget

import (
	"errors"
	"time"
)

// Service defines business-logic operations for the budget domain.
type Service interface {
	// Periods
	CreatePeriod(p *BudgetPeriod) error
	ListPeriods() ([]BudgetPeriod, error)
	GetPeriod(id uint) (*BudgetPeriod, error)

	// Lines
	CreateLine(periodID uint, line *BudgetLine) error
	ListLines(periodID uint) ([]BudgetLine, error)
	UpdateLine(id uint, update *BudgetLine) (*BudgetLine, error)
	PeriodSummary(periodID uint) (*PeriodSummaryResult, error)

	// Expenses
	SubmitExpense(lineID uint, submitterID uint, exp *BudgetExpense) error
	ListExpenses(lineID uint) ([]BudgetExpense, error)
	ApproveExpense(id uint, approverID uint) (*BudgetExpense, error)
	RejectExpense(id uint) (*BudgetExpense, error)
}

// PeriodSummaryResult holds aggregated budget totals for a period.
type PeriodSummaryResult struct {
	PeriodID        int     `json:"period_id"`
	TotalAllocated  int64   `json:"total_allocated"`
	TotalSpent      int64   `json:"total_spent"`
	TotalCommitted  int64   `json:"total_committed"`
	TotalRemaining  int64   `json:"total_remaining"`
	UtilizationPct  float64 `json:"utilization_pct"`
	LineCount       int     `json:"line_count"`
}

type service struct {
	repo Repository
}

// NewService returns a Service backed by the given Repository.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// ── BudgetPeriod ──────────────────────────────────────────────────────────────

func (s *service) CreatePeriod(p *BudgetPeriod) error {
	if p.Name == "" {
		return errors.New("name is required")
	}
	return s.repo.CreatePeriod(p)
}

func (s *service) ListPeriods() ([]BudgetPeriod, error) {
	return s.repo.ListPeriods()
}

func (s *service) GetPeriod(id uint) (*BudgetPeriod, error) {
	return s.repo.GetPeriod(id)
}

// ── BudgetLine ────────────────────────────────────────────────────────────────

func (s *service) CreateLine(periodID uint, line *BudgetLine) error {
	if line.Category == "" {
		return errors.New("category is required")
	}
	line.PeriodID = periodID
	return s.repo.CreateLine(line)
}

func (s *service) ListLines(periodID uint) ([]BudgetLine, error) {
	return s.repo.ListLines(periodID)
}

func (s *service) UpdateLine(id uint, update *BudgetLine) (*BudgetLine, error) {
	line, err := s.repo.GetLine(id)
	if err != nil {
		return nil, err
	}
	// Preserve identity fields; overwrite mutable fields from update.
	update.ID = line.ID
	update.PeriodID = line.PeriodID
	update.CreatedAt = line.CreatedAt
	if err := s.repo.SaveLine(update); err != nil {
		return nil, err
	}
	return update, nil
}

func (s *service) PeriodSummary(periodID uint) (*PeriodSummaryResult, error) {
	lines, err := s.repo.ListLines(periodID)
	if err != nil {
		return nil, err
	}
	var totalAllocated, totalSpent, totalCommitted int64
	for _, l := range lines {
		totalAllocated += l.AllocatedPaise
		totalSpent += l.SpentPaise
		totalCommitted += l.CommittedPaise
	}
	utilPct := float64(0)
	if totalAllocated > 0 {
		utilPct = float64(totalSpent) / float64(totalAllocated) * 100
	}
	return &PeriodSummaryResult{
		PeriodID:       int(periodID),
		TotalAllocated: totalAllocated,
		TotalSpent:     totalSpent,
		TotalCommitted: totalCommitted,
		TotalRemaining: totalAllocated - totalSpent - totalCommitted,
		UtilizationPct: utilPct,
		LineCount:      len(lines),
	}, nil
}

// ── BudgetExpense ─────────────────────────────────────────────────────────────

func (s *service) SubmitExpense(lineID uint, submitterID uint, exp *BudgetExpense) error {
	if exp.AmountPaise <= 0 {
		return errors.New("amount_paise must be positive")
	}
	exp.LineID = lineID
	exp.SubmittedBy = submitterID
	if exp.ExpenseDate.IsZero() {
		exp.ExpenseDate = time.Now()
	}
	// Commit amount against the budget line.
	if err := s.repo.IncrementCommitted(lineID, exp.AmountPaise); err != nil {
		return err
	}
	return s.repo.CreateExpense(exp)
}

func (s *service) ListExpenses(lineID uint) ([]BudgetExpense, error) {
	return s.repo.ListExpenses(lineID)
}

func (s *service) ApproveExpense(id uint, approverID uint) (*BudgetExpense, error) {
	exp, err := s.repo.GetExpense(id)
	if err != nil {
		return nil, err
	}
	exp.Status = "approved"
	exp.ApprovedBy = &approverID
	// Move from committed to spent.
	if err := s.repo.MoveCommittedToSpent(exp.LineID, exp.AmountPaise); err != nil {
		return nil, err
	}
	if err := s.repo.SaveExpense(exp); err != nil {
		return nil, err
	}
	return exp, nil
}

func (s *service) RejectExpense(id uint) (*BudgetExpense, error) {
	exp, err := s.repo.GetExpense(id)
	if err != nil {
		return nil, err
	}
	exp.Status = "rejected"
	// Release the committed amount.
	if err := s.repo.DecrementCommitted(exp.LineID, exp.AmountPaise); err != nil {
		return nil, err
	}
	if err := s.repo.SaveExpense(exp); err != nil {
		return nil, err
	}
	return exp, nil
}

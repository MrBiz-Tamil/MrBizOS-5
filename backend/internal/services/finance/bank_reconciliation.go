package bankreconcile

import (
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
)

// BankStatement represents a line from a bank statement import
type BankStatement struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	AccountID   string    `json:"account_id"`
	Date        time.Time `json:"date"`
	Description string    `json:"description"`
	Debit       int64     `json:"debit"`  // paise
	Credit      int64     `json:"credit"` // paise
	Balance     int64     `json:"balance"` // paise
	Reference   string    `json:"reference"`
	Status      string    `gorm:"default:'unmatched'" json:"status"` // matched, unmatched, ignored
	LedgerID    *uint     `json:"ledger_id,omitempty"` // matched ledger entry
	ImportedAt  time.Time `json:"imported_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func (BankStatement) TableName() string {
	return "bank_statements"
}

// ReconciliationResult summarizes a reconciliation run
type ReconciliationResult struct {
	Period          string    `json:"period"`
	TotalStatement  int       `json:"total_statement_entries"`
	Matched         int       `json:"matched"`
	Unmatched       int       `json:"unmatched"`
	Ignored         int       `json:"ignored"`
	OpeningBalance  int64     `json:"opening_balance"` // paise
	ClosingBalance  int64     `json:"closing_balance"` // paise
	LedgerBalance   int64     `json:"ledger_balance"`  // paise
	Discrepancy     int64     `json:"discrepancy"`     // paise (abs)
	IsReconciled    bool      `json:"is_reconciled"`
	RunAt           time.Time `json:"run_at"`
}

// ReconciliationService handles bank statement reconciliation
type ReconciliationService struct {
	db *gorm.DB
}

// NewReconciliationService creates a new reconciliation service
func NewReconciliationService(db *gorm.DB) *ReconciliationService {
	return &ReconciliationService{db: db}
}

// ImportCSV parses and imports bank statement lines
// Supports standard bank CSV format: Date,Description,Debit,Credit,Balance,Reference
func (s *ReconciliationService) ImportCSV(accountID string, rows [][]string) (int, error) {
	imported := 0
	for i, row := range rows {
		if i == 0 {
			continue // skip header
		}
		if len(row) < 5 {
			continue
		}

		date, err := parseFlexDate(row[0])
		if err != nil {
			continue
		}

		stmt := &BankStatement{
			AccountID:   accountID,
			Date:        date,
			Description: row[1],
			Debit:       parsePaise(row[2]),
			Credit:      parsePaise(row[3]),
			Balance:     parsePaise(row[4]),
			ImportedAt:  time.Now(),
		}
		if len(row) >= 6 {
			stmt.Reference = row[5]
		}

		// Upsert by (account_id + date + reference)
		result := s.db.Where(BankStatement{
			AccountID: accountID,
			Date:      date,
			Reference: stmt.Reference,
		}).FirstOrCreate(stmt)
		if result.Error != nil {
			continue
		}
		if result.RowsAffected > 0 {
			imported++
		}
	}
	return imported, nil
}

// Reconcile runs auto-matching between bank statement entries and ledger records
func (s *ReconciliationService) Reconcile(accountID string, startDate, endDate time.Time) (*ReconciliationResult, error) {
	// Fetch unmatched bank statement entries
	var statements []BankStatement
	if err := s.db.Where(
		"account_id = ? AND date BETWEEN ? AND ? AND status = 'unmatched'",
		accountID, startDate, endDate,
	).Find(&statements).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch statements: %w", err)
	}

	matched := 0
	for _, stmt := range statements {
		// Try to find a matching ledger entry by amount and date proximity (±1 day)
		var ledgerMatch struct {
			ID uint
		}
		amount := stmt.Credit - stmt.Debit // positive = inflow, negative = outflow
		s.db.Raw(`
			SELECT id FROM ledger_entries
			WHERE ABS(amount - ?) < 100
			AND date BETWEEN ? AND ?
			AND reconciled = false
			LIMIT 1
		`, amount, stmt.Date.AddDate(0, 0, -1), stmt.Date.AddDate(0, 0, 1)).Scan(&ledgerMatch)

		if ledgerMatch.ID > 0 {
			// Mark as matched
			s.db.Model(&stmt).Updates(map[string]interface{}{
				"status":    "matched",
				"ledger_id": ledgerMatch.ID,
			})
			s.db.Exec("UPDATE ledger_entries SET reconciled = true WHERE id = ?", ledgerMatch.ID)
			matched++
		}
	}

	// Compute summary
	var total int64
	s.db.Model(&BankStatement{}).
		Where("account_id = ? AND date BETWEEN ? AND ?", accountID, startDate, endDate).
		Count(&total)

	var matchedCount, unmatchedCount, ignoredCount int64
	s.db.Model(&BankStatement{}).Where("account_id = ? AND date BETWEEN ? AND ? AND status = 'matched'", accountID, startDate, endDate).Count(&matchedCount)
	s.db.Model(&BankStatement{}).Where("account_id = ? AND date BETWEEN ? AND ? AND status = 'unmatched'", accountID, startDate, endDate).Count(&unmatchedCount)
	s.db.Model(&BankStatement{}).Where("account_id = ? AND date BETWEEN ? AND ? AND status = 'ignored'", accountID, startDate, endDate).Count(&ignoredCount)

	// Get opening and closing balances from statement
	var openRow, closeRow BankStatement
	s.db.Where("account_id = ? AND date >= ?", accountID, startDate).Order("date ASC, id ASC").First(&openRow)
	s.db.Where("account_id = ? AND date <= ?", accountID, endDate).Order("date DESC, id DESC").First(&closeRow)

	// Get ledger balance for period
	var ledgerBalance struct{ Total int64 }
	s.db.Raw("SELECT COALESCE(SUM(amount),0) AS total FROM ledger_entries WHERE account_id = ? AND date BETWEEN ? AND ?", accountID, startDate, endDate).Scan(&ledgerBalance)

	discrepancy := int64(math.Abs(float64(closeRow.Balance - openRow.Balance - ledgerBalance.Total)))

	result := &ReconciliationResult{
		Period:         fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
		TotalStatement: int(total),
		Matched:        int(matchedCount),
		Unmatched:      int(unmatchedCount),
		Ignored:        int(ignoredCount),
		OpeningBalance: openRow.Balance,
		ClosingBalance: closeRow.Balance,
		LedgerBalance:  ledgerBalance.Total,
		Discrepancy:    discrepancy,
		IsReconciled:   discrepancy < 100 && unmatchedCount == 0, // within ₹1 tolerance
		RunAt:          time.Now(),
	}
	return result, nil
}

// IgnoreEntry marks a bank statement entry as intentionally ignored
func (s *ReconciliationService) IgnoreEntry(entryID uint) error {
	return s.db.Model(&BankStatement{}).Where("id = ?", entryID).Update("status", "ignored").Error
}

// ManualMatch manually links a bank entry to a ledger entry
func (s *ReconciliationService) ManualMatch(statementID, ledgerID uint) error {
	if err := s.db.Model(&BankStatement{}).Where("id = ?", statementID).Updates(map[string]interface{}{
		"status": "matched", "ledger_id": ledgerID,
	}).Error; err != nil {
		return err
	}
	return s.db.Exec("UPDATE ledger_entries SET reconciled = true WHERE id = ?", ledgerID).Error
}

// AutoMigrate creates the bank_statements table
func (s *ReconciliationService) AutoMigrate() error {
	return s.db.AutoMigrate(&BankStatement{})
}

// --- helpers ---

func parsePaise(s string) int64 {
	if s == "" || s == "-" {
		return 0
	}
	// Remove commas and ₹ sign
	clean := ""
	for _, r := range s {
		if r >= '0' && r <= '9' || r == '.' {
			clean += string(r)
		}
	}
	if clean == "" {
		return 0
	}
	var f float64
	fmt.Sscanf(clean, "%f", &f)
	return int64(f * 100)
}

func parseFlexDate(s string) (time.Time, error) {
	formats := []string{
		"02/01/2006", "2006-01-02", "01/02/2006",
		"02-01-2006", "2-1-2006", "Jan 02, 2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %s", s)
}

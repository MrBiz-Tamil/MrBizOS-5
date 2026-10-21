package anomaly

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Financial-integrity rules — the "find immoral activities" layer.
//
// These extend the generic anomaly engine with money-specific fraud signals.
// Like all rules they are PURE (read-only) and run every scan tick over the
// recent window. Some read audit_logs (mutation diffs); others read the finance
// tables directly for point-in-time conditions.

// Money thresholds, in paise.
const (
	highValueExpensePaise = 5_000_000  // ₹50,000
	highValuePayoutPaise  = 10_000_000 // ₹1,00,000
)

// RegisterFinanceRules appends the financial-integrity rules to the Registry.
// Call once at startup, before StartScanner.
func RegisterFinanceRules() {
	Registry = append(Registry,
		duplicatePayoutRule(),
		manualBalanceEditRule(),
		oversizedExpenseRule(),
		expenseFourEyesRule(),
		taxRateLoweredRule(),
		dividendTamperRule(),
		stuckHighValuePayoutRule(),
		negativeAssetBalanceRule(),
	)
}

// 1. Duplicate payout — same doctor + same amount more than once in the window.
func duplicatePayoutRule() Rule {
	return Rule{
		Name:        "duplicate_payout",
		Severity:    SeverityCritical,
		Description: "The same doctor + amount payout request appeared more than once in the scan window",
		Detect: func(db *gorm.DB, since time.Time) ([]Anomaly, error) {
			type group struct {
				DoctorID uint
				Amount   int64
				N        int
				FirstID  uint
				LastID   uint
			}
			var groups []group
			if err := db.Table("payout_requests").
				Select("doctor_id, amount, COUNT(*) AS n, MIN(id) AS first_id, MAX(id) AS last_id").
				Where("created_at >= ?", since).
				Group("doctor_id, amount").
				Having("COUNT(*) > 1").
				Scan(&groups).Error; err != nil {
				return nil, err
			}
			out := make([]Anomaly, 0, len(groups))
			for _, g := range groups {
				out = append(out, Anomaly{
					RuleName:    "duplicate_payout",
					Severity:    SeverityCritical,
					Title:       fmt.Sprintf("Duplicate payout: doctor #%d × ₹%.2f appeared %d times", g.DoctorID, float64(g.Amount)/100, g.N),
					Description: fmt.Sprintf("%d payout requests for doctor #%d at the same amount within the window — possible double-pay", g.N, g.DoctorID),
					Table:       "payout_requests",
					Hits:        g.N,
					Evidence:    mustJSON(map[string]interface{}{"doctor_id": g.DoctorID, "amount": g.Amount, "first_id": g.FirstID, "last_id": g.LastID}),
					DetectedAt:  time.Now(),
				})
			}
			return out, nil
		},
	}
}

// 2. Manual balance edit — an admin directly UPDATEd a ledger account balance.
// Balances must only move through double-entry transactions, never by hand.
func manualBalanceEditRule() Rule {
	return Rule{
		Name:        "manual_ledger_balance_edit",
		Severity:    SeverityCritical,
		Description: "A ledger account balance was edited directly (not via a posted transaction)",
		Detect: func(db *gorm.DB, since time.Time) ([]Anomaly, error) {
			rows, err := scanLogs(db, since,
				`action = ? AND table_name = ? AND actor_role IN ?`,
				"UPDATE", "ledger_account_v2", []string{"admin", "super_admin"})
			if err != nil {
				return nil, err
			}
			var out []Anomaly
			for _, r := range rows {
				oldV, newV, ok := readDiffPair(r.Diff, "balance")
				if !ok {
					continue
				}
				out = append(out, Anomaly{
					RuleName:    "manual_ledger_balance_edit",
					Severity:    SeverityCritical,
					Title:       fmt.Sprintf("Ledger balance hand-edited on %s — ₹%.2f → ₹%.2f", r.RowID, oldV/100, newV/100),
					Description: fmt.Sprintf("Account row %s balance changed directly by actor #%d (%s) — bypasses double-entry", r.RowID, r.ActorID, r.ActorRole),
					ActorID:     r.ActorID,
					ActorRole:   r.ActorRole,
					Table:       "ledger_account_v2",
					Hits:        1,
					Evidence:    mustJSON(map[string]interface{}{"audit_log_id": r.ID, "row_id": r.RowID, "old": oldV, "new": newV}),
					DetectedAt:  time.Now(),
				})
			}
			return out, nil
		},
	}
}

// 3. Oversized expense — a single expense booked at or above the high-value
// threshold. Not necessarily fraud, but always worth a human glance.
func oversizedExpenseRule() Rule {
	return Rule{
		Name:        "oversized_expense",
		Severity:    SeverityWarning,
		Description: fmt.Sprintf("An expense ≥ ₹%.0f was created", float64(highValueExpensePaise)/100),
		Detect: func(db *gorm.DB, since time.Time) ([]Anomaly, error) {
			type row struct {
				ID          uint
				Amount      int64
				Description string
				CreatedBy   uint
			}
			var rows []row
			if err := db.Table("expenses").
				Where("created_at >= ? AND amount >= ?", since, int64(highValueExpensePaise)).
				Scan(&rows).Error; err != nil {
				return nil, err
			}
			out := make([]Anomaly, 0, len(rows))
			for _, e := range rows {
				out = append(out, Anomaly{
					RuleName:    "oversized_expense",
					Severity:    SeverityWarning,
					Title:       fmt.Sprintf("High-value expense ₹%.2f — %s", float64(e.Amount)/100, e.Description),
					Description: fmt.Sprintf("Expense #%d for ₹%.2f created by user #%d", e.ID, float64(e.Amount)/100, e.CreatedBy),
					ActorID:     e.CreatedBy,
					Table:       "expenses",
					Hits:        1,
					Evidence:    mustJSON(map[string]interface{}{"expense_id": e.ID, "amount": e.Amount, "created_by": e.CreatedBy}),
					DetectedAt:  time.Now(),
				})
			}
			return out, nil
		},
	}
}

// 4. 4-eyes bypass — a high-value expense approved by the same person who
// created it. Segregation of duties requires creator ≠ approver.
func expenseFourEyesRule() Rule {
	return Rule{
		Name:        "expense_four_eyes_bypass",
		Severity:    SeverityCritical,
		Description: "A high-value expense was approved by the same user who created it",
		Detect: func(db *gorm.DB, since time.Time) ([]Anomaly, error) {
			type row struct {
				ID         uint
				Amount     int64
				CreatedBy  uint
				ApprovedBy uint
			}
			var rows []row
			if err := db.Table("expenses").
				Where("updated_at >= ? AND status = ? AND approved_by IS NOT NULL AND approved_by = created_by AND amount >= ?",
					since, "approved", int64(highValueExpensePaise)).
				Scan(&rows).Error; err != nil {
				return nil, err
			}
			out := make([]Anomaly, 0, len(rows))
			for _, e := range rows {
				out = append(out, Anomaly{
					RuleName:    "expense_four_eyes_bypass",
					Severity:    SeverityCritical,
					Title:       fmt.Sprintf("Self-approved expense ₹%.2f by user #%d", float64(e.Amount)/100, e.CreatedBy),
					Description: fmt.Sprintf("Expense #%d (₹%.2f) created AND approved by user #%d — segregation-of-duties violation", e.ID, float64(e.Amount)/100, e.CreatedBy),
					ActorID:     e.ApprovedBy,
					Table:       "expenses",
					Hits:        1,
					Evidence:    mustJSON(map[string]interface{}{"expense_id": e.ID, "amount": e.Amount, "created_by": e.CreatedBy, "approved_by": e.ApprovedBy}),
					DetectedAt:  time.Now(),
				})
			}
			return out, nil
		},
	}
}

// 5. Tax rate lowered — a TaxRule's rate was decreased (tax-evasion signal).
func taxRateLoweredRule() Rule {
	return Rule{
		Name:        "tax_rate_lowered",
		Severity:    SeverityCritical,
		Description: "A tax rule's rate was decreased",
		Detect: func(db *gorm.DB, since time.Time) ([]Anomaly, error) {
			rows, err := scanLogs(db, since, `action = ? AND table_name = ?`, "UPDATE", "tax_rules")
			if err != nil {
				return nil, err
			}
			var out []Anomaly
			for _, r := range rows {
				oldV, newV, ok := readDiffPair(r.Diff, "rate")
				if !ok || newV >= oldV {
					continue
				}
				out = append(out, Anomaly{
					RuleName:    "tax_rate_lowered",
					Severity:    SeverityCritical,
					Title:       fmt.Sprintf("Tax rate lowered %.2f%% → %.2f%% on rule %s", oldV, newV, r.RowID),
					Description: fmt.Sprintf("tax_rules #%s rate decreased by actor #%d (%s)", r.RowID, r.ActorID, r.ActorRole),
					ActorID:     r.ActorID,
					ActorRole:   r.ActorRole,
					Table:       "tax_rules",
					Hits:        1,
					Evidence:    mustJSON(map[string]interface{}{"audit_log_id": r.ID, "row_id": r.RowID, "old": oldV, "new": newV}),
					DetectedAt:  time.Now(),
				})
			}
			return out, nil
		},
	}
}

// 6. Dividend tampering — an investor's dividends_paid was decreased. It is a
// cumulative figure that should only ever rise.
func dividendTamperRule() Rule {
	return Rule{
		Name:        "investor_dividend_tamper",
		Severity:    SeverityCritical,
		Description: "An investor's cumulative dividends_paid was decreased",
		Detect: func(db *gorm.DB, since time.Time) ([]Anomaly, error) {
			rows, err := scanLogs(db, since, `action = ? AND table_name = ?`, "UPDATE", "investors")
			if err != nil {
				return nil, err
			}
			var out []Anomaly
			for _, r := range rows {
				oldV, newV, ok := readDiffPair(r.Diff, "dividends_paid")
				if !ok || newV >= oldV {
					continue
				}
				out = append(out, Anomaly{
					RuleName:    "investor_dividend_tamper",
					Severity:    SeverityCritical,
					Title:       fmt.Sprintf("Investor dividends_paid decreased ₹%.2f → ₹%.2f", oldV/100, newV/100),
					Description: fmt.Sprintf("investors #%s dividends_paid lowered by actor #%d (%s)", r.RowID, r.ActorID, r.ActorRole),
					ActorID:     r.ActorID,
					ActorRole:   r.ActorRole,
					Table:       "investors",
					Hits:        1,
					Evidence:    mustJSON(map[string]interface{}{"audit_log_id": r.ID, "row_id": r.RowID, "old": oldV, "new": newV}),
					DetectedAt:  time.Now(),
				})
			}
			return out, nil
		},
	}
}

// 7. Stuck high-value payout — a large payout left pending for over 30 days.
// Point-in-time; ignores the scan window.
func stuckHighValuePayoutRule() Rule {
	return Rule{
		Name:        "stuck_high_value_payout",
		Severity:    SeverityWarning,
		Description: "A high-value payout has been pending for more than 30 days",
		Detect: func(db *gorm.DB, _ time.Time) ([]Anomaly, error) {
			cutoff := time.Now().AddDate(0, 0, -30)
			type row struct {
				ID       uint
				DoctorID uint
				Amount   int64
			}
			var rows []row
			if err := db.Table("payout_requests").
				Where("status = ? AND created_at < ? AND amount >= ?", "pending", cutoff, int64(highValuePayoutPaise)).
				Scan(&rows).Error; err != nil {
				return nil, err
			}
			out := make([]Anomaly, 0, len(rows))
			for _, p := range rows {
				out = append(out, Anomaly{
					RuleName:    "stuck_high_value_payout",
					Severity:    SeverityWarning,
					Title:       fmt.Sprintf("Payout #%d (₹%.2f) pending >30 days", p.ID, float64(p.Amount)/100),
					Description: fmt.Sprintf("Payout request #%d for doctor #%d has been pending over 30 days", p.ID, p.DoctorID),
					Table:       "payout_requests",
					Hits:        1,
					Evidence:    mustJSON(map[string]interface{}{"payout_id": p.ID, "doctor_id": p.DoctorID, "amount": p.Amount}),
					DetectedAt:  time.Now(),
				})
			}
			return out, nil
		},
	}
}

// 8. Negative asset balance — an asset account went negative (e.g. cash/bank
// paid out more than it held). Point-in-time.
func negativeAssetBalanceRule() Rule {
	return Rule{
		Name:        "negative_asset_balance",
		Severity:    SeverityWarning,
		Description: "An asset-type ledger account has a negative balance",
		Detect: func(db *gorm.DB, _ time.Time) ([]Anomaly, error) {
			type row struct {
				Name    string
				Balance int64
			}
			var rows []row
			if err := db.Table("ledger_account_v2").
				Where("type = ? AND balance < 0", "asset").
				Scan(&rows).Error; err != nil {
				return nil, err
			}
			out := make([]Anomaly, 0, len(rows))
			for _, a := range rows {
				out = append(out, Anomaly{
					RuleName:    "negative_asset_balance",
					Severity:    SeverityWarning,
					Title:       fmt.Sprintf("Asset account %q is negative: ₹%.2f", a.Name, float64(a.Balance)/100),
					Description: fmt.Sprintf("Asset account %q holds a negative balance — investigate over-disbursement", a.Name),
					Table:       "ledger_account_v2",
					Hits:        1,
					Evidence:    mustJSON(map[string]interface{}{"account": a.Name, "balance": a.Balance}),
					DetectedAt:  time.Now(),
				})
			}
			return out, nil
		},
	}
}

package finance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Continuous reconciliation — the ledger proving its own correctness.
//
// At 1M users with minimal staff nobody reconciles by hand. These checks run
// nightly (and on demand) and any failure becomes a CRITICAL anomaly that
// surfaces in the existing Anomalies dashboard + alert pipeline. A clean run is
// silent. Each check is read-only.

// CheckResult is the outcome of one integrity check.
type CheckResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

// ReconciliationReport aggregates all checks for one run.
type ReconciliationReport struct {
	RanAt     time.Time     `json:"ran_at"`
	AllPassed bool          `json:"all_passed"`
	Results   []CheckResult `json:"results"`
}

// Reconcile runs every ledger-integrity check and returns a report. It never
// mutates data. Callers (cron, admin endpoint) decide what to do with failures.
func (s *service) Reconcile(db *gorm.DB) ReconciliationReport {
	report := ReconciliationReport{RanAt: time.Now(), AllPassed: true}
	add := func(r CheckResult) {
		report.Results = append(report.Results, r)
		if !r.Passed {
			report.AllPassed = false
		}
	}

	add(checkTrialBalance(db))
	add(checkAccountBalances(db))
	add(checkTransactionChain(db))
	add(checkEntryHashes(db))
	add(checkPharmacyOrphans(db))

	return report
}

// 1. Trial balance: across the whole ledger, debits must equal credits.
func checkTrialBalance(db *gorm.DB) CheckResult {
	var totalDebit, totalCredit int64
	db.Model(&LedgerEntry{}).Where("direction = ?", "debit").
		Select("COALESCE(SUM(amount),0)").Scan(&totalDebit)
	db.Model(&LedgerEntry{}).Where("direction = ?", "credit").
		Select("COALESCE(SUM(amount),0)").Scan(&totalCredit)

	if totalDebit != totalCredit {
		return CheckResult{
			Name:   "trial_balance",
			Passed: false,
			Detail: fmt.Sprintf("ledger out of balance: debits=%d paise, credits=%d paise (diff %d)",
				totalDebit, totalCredit, totalDebit-totalCredit),
		}
	}
	return CheckResult{Name: "trial_balance", Passed: true,
		Detail: fmt.Sprintf("balanced at %d paise", totalDebit)}
}

// 2. Balance proof: each account's stored balance must equal the balance
// recomputed from its entries (catches out-of-band UPDATEs to a balance).
func checkAccountBalances(db *gorm.DB) CheckResult {
	type agg struct {
		AccountID uuid.UUID
		Dr        int64
		Cr        int64
	}
	var rows []agg
	db.Model(&LedgerEntry{}).
		Select("account_id, " +
			"COALESCE(SUM(CASE WHEN direction='debit' THEN amount ELSE 0 END),0) AS dr, " +
			"COALESCE(SUM(CASE WHEN direction='credit' THEN amount ELSE 0 END),0) AS cr").
		Group("account_id").Scan(&rows)

	computed := make(map[uuid.UUID]agg, len(rows))
	for _, r := range rows {
		computed[r.AccountID] = r
	}

	var accounts []LedgerAccount
	db.Find(&accounts)

	var mismatches []string
	for _, acc := range accounts {
		a := computed[acc.ID]
		var want int64
		if acc.Type == TypeAsset || acc.Type == TypeExpense {
			want = a.Dr - a.Cr
		} else {
			want = a.Cr - a.Dr
		}
		if want != acc.Balance {
			mismatches = append(mismatches,
				fmt.Sprintf("%s: stored=%d computed=%d", acc.Name, acc.Balance, want))
		}
	}

	if len(mismatches) > 0 {
		detail := fmt.Sprintf("%d account(s) drifted", len(mismatches))
		if len(mismatches) <= 5 {
			detail += ": "
			for i, m := range mismatches {
				if i > 0 {
					detail += "; "
				}
				detail += m
			}
		}
		return CheckResult{Name: "account_balance_proof", Passed: false, Detail: detail}
	}
	return CheckResult{Name: "account_balance_proof", Passed: true,
		Detail: fmt.Sprintf("%d accounts verified", len(accounts))}
}

// 3. Transaction chain linkage: each transaction's PreviousHash must equal the
// prior transaction's Hash (tamper-evidence against insert/delete/reorder).
func checkTransactionChain(db *gorm.DB) CheckResult {
	var txns []LedgerTransaction
	db.Find(&txns)

	if len(txns) == 0 {
		return CheckResult{Name: "transaction_chain", Passed: true, Detail: "0 transactions chained"}
	}

	byPrevHash := make(map[string]LedgerTransaction, len(txns))
	var genesisCount int
	for _, txn := range txns {
		if _, exists := byPrevHash[txn.PreviousHash]; exists {
			return CheckResult{
				Name:   "transaction_chain",
				Passed: false,
				Detail: fmt.Sprintf("chain broken: duplicate previous_hash %q detected", txn.PreviousHash),
			}
		}
		byPrevHash[txn.PreviousHash] = txn
		if txn.PreviousHash == "GENESIS" {
			genesisCount++
		}
	}

	if genesisCount == 0 {
		return CheckResult{
			Name:   "transaction_chain",
			Passed: false,
			Detail: "chain broken: no genesis transaction found",
		}
	}

	visited := make(map[string]bool, len(txns))
	prevHash := "GENESIS"
	count := 0
	for count < len(txns) {
		txn, ok := byPrevHash[prevHash]
		if !ok {
			return CheckResult{
				Name:   "transaction_chain",
				Passed: false,
				Detail: fmt.Sprintf("chain broken: transaction following hash %q is missing", prevHash),
			}
		}
		if visited[txn.Hash] {
			return CheckResult{
				Name:   "transaction_chain",
				Passed: false,
				Detail: fmt.Sprintf("chain broken: loop detected at transaction %s", txn.ReferenceID),
			}
		}
		visited[txn.Hash] = true
		prevHash = txn.Hash
		count++
	}

	return CheckResult{Name: "transaction_chain", Passed: true,
		Detail: fmt.Sprintf("%d transactions chained", len(txns))}
}

// 4. Per-entry hash: each entry's stored Hash must match a recompute over its
// immutable fields (id:direction:amount:balance_after), matching the formula in
// repository.RecordTransaction. Detects in-place edits to an entry's amount.
func checkEntryHashes(db *gorm.DB) CheckResult {
	const batch = 1000
	var offset int
	var checked, tampered int
	var firstBad string

	for {
		var entries []LedgerEntry
		db.Order("created_at asc").Limit(batch).Offset(offset).Find(&entries)
		if len(entries) == 0 {
			break
		}
		for _, e := range entries {
			// Entries written before per-entry hashing was introduced have an
			// empty Hash — skip rather than false-flag legacy rows.
			if e.Hash == "" {
				continue
			}
			data := fmt.Sprintf("%s:%s:%d:%d", e.ID.String(), e.Direction, e.Amount, e.BalanceAfter)
			sum := sha256.Sum256([]byte(data))
			if hex.EncodeToString(sum[:]) != e.Hash {
				tampered++
				if firstBad == "" {
					firstBad = e.ID.String()
				}
			}
			checked++
		}
		offset += batch
	}

	if tampered > 0 {
		return CheckResult{Name: "entry_hash_integrity", Passed: false,
			Detail: fmt.Sprintf("%d tampered entr(ies); first=%s", tampered, firstBad)}
	}
	return CheckResult{Name: "entry_hash_integrity", Passed: true,
		Detail: fmt.Sprintf("%d entries verified", checked)}
}

// 5. Orphan detection: every paid pharmacy order must carry a ledger transaction
// id. A paid order with no ledger link means money was collected but never
// booked. (Pharmacy orders carry the link column; other streams are added as
// they gain one.)
func checkPharmacyOrphans(db *gorm.DB) CheckResult {
	if !db.Migrator().HasTable("pharmacy_orders") {
		return CheckResult{Name: "pharmacy_orphans", Passed: true, Detail: "no pharmacy_orders table"}
	}
	var orphans int64
	db.Table("pharmacy_orders").
		Where("payment_status = ? AND (ledger_transaction_id IS NULL OR ledger_transaction_id = ?)", "paid", "").
		Count(&orphans)

	if orphans > 0 {
		return CheckResult{Name: "pharmacy_orphans", Passed: false,
			Detail: fmt.Sprintf("%d paid pharmacy order(s) with no ledger transaction", orphans)}
	}
	return CheckResult{Name: "pharmacy_orphans", Passed: true, Detail: "all paid orders booked"}
}

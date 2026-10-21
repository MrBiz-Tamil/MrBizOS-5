package financials

import (
	"fmt"
	"time"

	"drayya-core/internal/database"
)

// ── Common ────────────────────────────────────────────────────────────────────

// PeriodFilter is used by all report generators.
type PeriodFilter struct {
	From  time.Time
	To    time.Time
	Label string // e.g. "FY 2025-26", "Q1 2026"
}

func ThisMonth() PeriodFilter {
	now := time.Now()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	to := from.AddDate(0, 1, -1).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	return PeriodFilter{From: from, To: to, Label: from.Format("January 2006")}
}

func ThisFY() PeriodFilter {
	now := time.Now()
	year := now.Year()
	if now.Month() < time.April {
		year--
	}
	from := time.Date(year, time.April, 1, 0, 0, 0, 0, now.Location())
	to := time.Date(year+1, time.March, 31, 23, 59, 59, 0, now.Location())
	return PeriodFilter{
		From:  from,
		To:    to,
		Label: fmt.Sprintf("FY %d-%02d", year, (year+1)%100),
	}
}

type ledgerRow struct {
	AccType  string
	Name     string
	NetPaise int64
}

// ledgerNetByType aggregates net amounts (credit − debit) per account for a period,
// grouped by account type and name. Uses the real v2 schema:
//   - ledger_account_v2.type  (not account_type)
//   - ledger_entry_v2.direction + amount  (not separate credit/debit columns)
func ledgerNetByType(from, to time.Time) ([]ledgerRow, error) {
	var rows []ledgerRow
	err := database.DB.Table("ledger_entry_v2 le").
		Joins("JOIN ledger_account_v2 la ON la.id = le.account_id").
		Select(`la.type as acc_type, la.name,
			SUM(CASE WHEN le.direction = 'credit' THEN le.amount ELSE 0 END) -
			SUM(CASE WHEN le.direction = 'debit'  THEN le.amount ELSE 0 END) AS net_paise`).
		Where("le.created_at BETWEEN ? AND ?", from, to).
		Group("la.type, la.name").
		Scan(&rows).Error
	return rows, err
}

// ── Profit & Loss ─────────────────────────────────────────────────────────────

// PLReport is the income statement for a period.
type PLReport struct {
	Period       PeriodFilter     `json:"period"`
	Revenue      map[string]int64 `json:"revenue"`    // by account name (paise)
	TotalRevenue int64            `json:"total_revenue"`
	COGS         map[string]int64 `json:"cogs"`
	TotalCOGS    int64            `json:"total_cogs"`
	GrossProfit  int64            `json:"gross_profit"`
	GrossMargin  float64          `json:"gross_margin_pct"`
	OpEx         map[string]int64 `json:"opex"`
	TotalOpEx    int64            `json:"total_opex"`
	EBITDA       int64            `json:"ebitda"`
	Depreciation int64            `json:"depreciation"`
	EBIT         int64            `json:"ebit"`
	Interest     int64            `json:"interest"`
	EBT          int64            `json:"ebt"`
	TaxProvision int64            `json:"tax_provision"`
	NetProfit    int64            `json:"net_profit"`
	NetMargin    float64          `json:"net_margin_pct"`
	GeneratedAt  time.Time        `json:"generated_at"`
}

// GeneratePLReport builds the P&L from the double-entry ledger.
func GeneratePLReport(period PeriodFilter) (*PLReport, error) {
	rows, err := ledgerNetByType(period.From, period.To)
	if err != nil {
		return nil, err
	}

	report := &PLReport{
		Period:      period,
		Revenue:     map[string]int64{},
		COGS:        map[string]int64{},
		OpEx:        map[string]int64{},
		GeneratedAt: time.Now(),
	}

	for _, r := range rows {
		switch r.AccType {
		case "revenue":
			report.Revenue[r.Name] = r.NetPaise
			report.TotalRevenue += r.NetPaise
		case "expense":
			// Split COGS from OpEx by naming convention: accounts named "cogs:*" go to COGS
			if len(r.Name) >= 5 && r.Name[:5] == "cogs:" {
				report.COGS[r.Name] = -r.NetPaise // expenses are debits → positive cost
				report.TotalCOGS += -r.NetPaise
			} else {
				report.OpEx[r.Name] = -r.NetPaise
				report.TotalOpEx += -r.NetPaise
			}
		}
	}

	report.GrossProfit = report.TotalRevenue - report.TotalCOGS
	if report.TotalRevenue > 0 {
		report.GrossMargin = float64(report.GrossProfit) / float64(report.TotalRevenue) * 100
	}
	report.EBITDA = report.GrossProfit - report.TotalOpEx
	report.EBIT = report.EBITDA - report.Depreciation
	report.EBT = report.EBIT - report.Interest
	report.TaxProvision = int64(float64(report.EBT) * 0.25) // 25 % effective rate
	if report.TaxProvision < 0 {
		report.TaxProvision = 0
	}
	report.NetProfit = report.EBT - report.TaxProvision
	if report.TotalRevenue > 0 {
		report.NetMargin = float64(report.NetProfit) / float64(report.TotalRevenue) * 100
	}

	return report, nil
}

// ── Balance Sheet ─────────────────────────────────────────────────────────────

// BalanceSheet is the statement of financial position as at a date.
type BalanceSheet struct {
	AsAt             time.Time        `json:"as_at"`
	Assets           map[string]int64 `json:"assets"`
	TotalAssets      int64            `json:"total_assets"`
	Liabilities      map[string]int64 `json:"liabilities"`
	TotalLiabilities int64            `json:"total_liabilities"`
	Equity           map[string]int64 `json:"equity"`
	TotalEquity      int64            `json:"total_equity"`
	Balanced         bool             `json:"balanced"` // Assets = Liabilities + Equity
	GeneratedAt      time.Time        `json:"generated_at"`
}

// GenerateBalanceSheet builds the balance sheet as at the given date.
// Assets use the debit-normal convention (debit increases asset balance).
func GenerateBalanceSheet(asAt time.Time) (*BalanceSheet, error) {
	type row struct {
		AccType string
		Name    string
		Balance int64
	}
	var rows []row
	err := database.DB.Table("ledger_entry_v2 le").
		Joins("JOIN ledger_account_v2 la ON la.id = le.account_id").
		Select(`la.type as acc_type, la.name,
			SUM(CASE WHEN le.direction = 'debit'  THEN le.amount ELSE 0 END) -
			SUM(CASE WHEN le.direction = 'credit' THEN le.amount ELSE 0 END) AS balance`).
		Where("le.created_at <= ?", asAt).
		Group("la.type, la.name").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	bs := &BalanceSheet{
		AsAt:        asAt,
		Assets:      map[string]int64{},
		Liabilities: map[string]int64{},
		Equity:      map[string]int64{},
		GeneratedAt: time.Now(),
	}

	for _, r := range rows {
		switch r.AccType {
		case "asset":
			bs.Assets[r.Name] = r.Balance
			bs.TotalAssets += r.Balance
		case "liability":
			// Liabilities are credit-normal; negate so balance reads as positive
			bs.Liabilities[r.Name] = -r.Balance
			bs.TotalLiabilities += -r.Balance
		case "equity":
			bs.Equity[r.Name] = -r.Balance
			bs.TotalEquity += -r.Balance
		}
	}

	bs.Balanced = bs.TotalAssets == (bs.TotalLiabilities + bs.TotalEquity)
	return bs, nil
}

// ── Cash Flow ─────────────────────────────────────────────────────────────────

// CashFlowReport is the statement of cash flows for a period (indirect method).
// Since the schema has no cash_flow_type column, we classify by account type:
//   operating  → revenue + expense accounts
//   investing  → asset accounts (non-cash)
//   financing  → liability + equity accounts
type CashFlowReport struct {
	Period         PeriodFilter     `json:"period"`
	Operating      map[string]int64 `json:"operating"`
	TotalOperating int64            `json:"total_operating"`
	Investing      map[string]int64 `json:"investing"`
	TotalInvesting int64            `json:"total_investing"`
	Financing      map[string]int64 `json:"financing"`
	TotalFinancing int64            `json:"total_financing"`
	NetCashFlow    int64            `json:"net_cash_flow"`
	OpeningBalance int64            `json:"opening_balance"`
	ClosingBalance int64            `json:"closing_balance"`
	GeneratedAt    time.Time        `json:"generated_at"`
}

// GenerateCashFlowReport derives cash movements from ledger entries.
func GenerateCashFlowReport(period PeriodFilter) (*CashFlowReport, error) {
	rows, err := ledgerNetByType(period.From, period.To)
	if err != nil {
		return nil, err
	}

	cf := &CashFlowReport{
		Period:      period,
		Operating:   map[string]int64{},
		Investing:   map[string]int64{},
		Financing:   map[string]int64{},
		GeneratedAt: time.Now(),
	}

	for _, r := range rows {
		switch r.AccType {
		case "revenue", "expense":
			cf.Operating[r.Name] = r.NetPaise
			cf.TotalOperating += r.NetPaise
		case "asset":
			cf.Investing[r.Name] = r.NetPaise
			cf.TotalInvesting += r.NetPaise
		case "liability", "equity":
			cf.Financing[r.Name] = r.NetPaise
			cf.TotalFinancing += r.NetPaise
		}
	}

	cf.NetCashFlow = cf.TotalOperating + cf.TotalInvesting + cf.TotalFinancing
	cf.ClosingBalance = cf.OpeningBalance + cf.NetCashFlow
	return cf, nil
}

// ── Revenue KPIs ──────────────────────────────────────────────────────────────

// GetRevenueKPIs returns top-line metrics for the admin dashboard.
func GetRevenueKPIs(period PeriodFilter) (map[string]interface{}, error) {
	type aggregate struct{ Total int64 }
	var revenue, refunds, payouts aggregate

	// Gross revenue = credits to revenue accounts
	database.DB.Table("ledger_entry_v2 le").
		Joins("JOIN ledger_account_v2 la ON la.id = le.account_id").
		Select("COALESCE(SUM(CASE WHEN le.direction = 'credit' THEN le.amount ELSE 0 END), 0) as total").
		Where("la.type = 'revenue' AND le.created_at BETWEEN ? AND ?", period.From, period.To).
		Scan(&revenue)

	// Refunds = debits on accounts with 'refund' in the name
	database.DB.Table("ledger_entry_v2 le").
		Joins("JOIN ledger_account_v2 la ON la.id = le.account_id").
		Select("COALESCE(SUM(CASE WHEN le.direction = 'debit' THEN le.amount ELSE 0 END), 0) as total").
		Where("LOWER(la.name) LIKE '%refund%' AND le.created_at BETWEEN ? AND ?", period.From, period.To).
		Scan(&refunds)

	// Payouts = debits on accounts with 'payout' or 'doctor' in the name
	database.DB.Table("ledger_entry_v2 le").
		Joins("JOIN ledger_account_v2 la ON la.id = le.account_id").
		Select("COALESCE(SUM(CASE WHEN le.direction = 'debit' THEN le.amount ELSE 0 END), 0) as total").
		Where("(LOWER(la.name) LIKE '%payout%' OR LOWER(la.name) LIKE '%doctor%') AND le.created_at BETWEEN ? AND ?", period.From, period.To).
		Scan(&payouts)

	net := revenue.Total - refunds.Total
	return map[string]interface{}{
		"period":        period.Label,
		"gross_revenue": revenue.Total,
		"refunds":       refunds.Total,
		"net_revenue":   net,
		"payouts":       payouts.Total,
		"platform_take": net - payouts.Total,
		"generated_at":  time.Now(),
	}, nil
}

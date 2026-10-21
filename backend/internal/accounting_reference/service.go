package accounting

import (
	"context"
	"fmt"
	"time"

	"saas-billing/internal/audit"
	"saas-billing/internal/model"
	"saas-billing/internal/sequence"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type Service struct {
	db     *pgxpool.Pool
	logger *audit.Logger
}

func NewService(db *pgxpool.Pool, logger *audit.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// Standard Chart of Accounts definitions
var standardAccounts = []struct {
	Code        string
	Name        string
	Type        string
	SubCategory string
	Description string
}{
	{"1100", "Cash in Hand", model.AccountTypeAsset, "Current Assets", "Physical cash and register balance"},
	{"1200", "Bank Operating Account", model.AccountTypeAsset, "Bank Accounts", "Primary business operating bank account"},
	{"1300", "Accounts Receivable", model.AccountTypeAsset, "Current Assets", "Amounts owed by trade debtors (Customers)"},
	{"1350", "GST Refund Receivable", model.AccountTypeAsset, "Current Assets", "IGST paid on exports and claimed back as refund"},
	{"1400", "Inventory Asset", model.AccountTypeAsset, "Current Assets", "Value of stock goods on hand"},
	{"2100", "Accounts Payable", model.AccountTypeLiability, "Current Liabilities", "Amounts owed to trade suppliers (Creditors)"},
	{"2210", "Output CGST Payable", model.AccountTypeLiability, "Duties & Taxes", "Central GST collected on sales"},
	{"2220", "Output SGST Payable", model.AccountTypeLiability, "Duties & Taxes", "State GST collected on sales"},
	{"2230", "Output IGST Payable", model.AccountTypeLiability, "Duties & Taxes", "Integrated GST collected on interstate sales"},
	{"2240", "Output UTGST Payable", model.AccountTypeLiability, "Duties & Taxes", "Union Territory GST collected on sales"},
	{"3100", "Owner Capital", model.AccountTypeEquity, "Capital Account", "Owner equity and invested capital"},
	{"3200", "Retained Earnings", model.AccountTypeEquity, "Reserves & Surplus", "Cumulative net operational earnings"},
	{"4100", "Domestic Sales Revenue", model.AccountTypeRevenue, "Sales Accounts", "Revenue generated from taxable domestic goods & services"},
	{"4200", "Interstate Sales Revenue", model.AccountTypeRevenue, "Sales Accounts", "Revenue generated from interstate transactions"},
	{"4400", "Export Sales Revenue", model.AccountTypeRevenue, "Sales Accounts", "Revenue from exports (zero-rated supplies)"},
	{"4500", "Foreign Exchange Gain", model.AccountTypeRevenue, "Direct Incomes", "Gain on realisation of foreign-currency receipts"},
	{"4300", "Other Operating Income", model.AccountTypeRevenue, "Direct Incomes", "Miscellaneous revenue from core operations"},
	{"5500", "Foreign Exchange Loss", model.AccountTypeExpense, "Indirect Expenses", "Loss on realisation of foreign-currency receipts"},
	{"5100", "Round-off Account", model.AccountTypeExpense, "Direct Expenses", "Fractions gained or lost during currency round-off"},
	{"5200", "Discounts Allowed", model.AccountTypeExpense, "Indirect Expenses", "Commercial discounts granted to customers"},
	{"5300", "Cost of Goods Sold", model.AccountTypeExpense, "Direct Expenses", "Direct cost of products sold"},
	{"5400", "Operating Expenses", model.AccountTypeExpense, "Indirect Expenses", "General operating and administrative business expenses"},
}

// EnsureStandardAccounts initializes the default Indian Chart of Accounts for a tenant if not present
func (s *Service) EnsureStandardAccounts(ctx context.Context, tenantID uuid.UUID) error {
	for _, acc := range standardAccounts {
		query := `
			INSERT INTO accounts (tenant_id, account_code, account_name, account_type, sub_category, description, is_system, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, TRUE, TRUE)
			ON CONFLICT (tenant_id, account_code) DO NOTHING
		`
		_, err := s.db.Exec(ctx, query, tenantID, acc.Code, acc.Name, acc.Type, acc.SubCategory, acc.Description)
		if err != nil {
			return fmt.Errorf("failed to seed account %s: %w", acc.Code, err)
		}
	}
	return nil
}

// GetAccountByCode fetches an account by tenant and code
func (s *Service) GetAccountByCode(ctx context.Context, tenantID uuid.UUID, code string) (*model.Account, error) {
	acc := &model.Account{}
	query := `
		SELECT id, tenant_id, account_code, account_name, account_type, sub_category, description, is_system, is_active, current_balance, created_at, updated_at
		FROM accounts
		WHERE tenant_id = $1 AND account_code = $2
	`
	err := s.db.QueryRow(ctx, query, tenantID, code).Scan(
		&acc.ID, &acc.TenantID, &acc.AccountCode, &acc.AccountName, &acc.AccountType,
		&acc.SubCategory, &acc.Description, &acc.IsSystem, &acc.IsActive,
		&acc.CurrentBalance, &acc.CreatedAt, &acc.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("account code %s not found: %w", code, err)
	}
	return acc, nil
}

// NextJournalNumber generates sequential voucher number like JV-2026-00001
func (s *Service) NextJournalNumber(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) (string, error) {
	year := fmt.Sprintf("%d", time.Now().Year())
	return sequence.NextNumber(ctx, tx, tenantID, sequence.TypeJournal, year, "JV", 5)
}

type JournalLineDraft struct {
	AccountID     uuid.UUID
	CustomerID    *uuid.UUID
	DebitAmount   decimal.Decimal
	CreditAmount  decimal.Decimal
	LineNarration string
}

// PostInvoiceJournal generates a balanced double-entry voucher when an invoice is issued
func (s *Service) PostInvoiceJournal(ctx context.Context, tx pgx.Tx, inv *model.Invoice, userID *uuid.UUID) (*model.JournalEntry, error) {
	// Ensure standard accounts exist
	if err := s.EnsureStandardAccounts(ctx, inv.TenantID); err != nil {
		return nil, err
	}

	arAcc, err := s.GetAccountByCode(ctx, inv.TenantID, "1300") // Accounts Receivable
	if err != nil {
		return nil, err
	}

	isExport, igstRefund := tradeFlags(ctx, tx, inv.ID)
	salesCode := "4100"
	if inv.IsInterState {
		salesCode = "4200"
	}
	if isExport {
		salesCode = "4400"
	}
	salesAcc, err := s.GetAccountByCode(ctx, inv.TenantID, salesCode)
	if err != nil {
		return nil, err
	}

	var lines []JournalLineDraft

	// 1. Debit Accounts Receivable (Customer Sub-ledger) with Grand Total. For an export with IGST paid, the buyer owes
	// only the goods value; the IGST is a refund receivable from the government, not from the customer.
	arAmount := inv.GrandTotal
	if igstRefund && inv.IGSTTotal.GreaterThan(decimal.Zero) {
		refundAcc, rerr := s.GetAccountByCode(ctx, inv.TenantID, "1350")
		if rerr != nil {
			return nil, rerr
		}
		arAmount = inv.GrandTotal.Sub(inv.IGSTTotal)
		lines = append(lines, JournalLineDraft{
			AccountID:     refundAcc.ID,
			DebitAmount:   inv.IGSTTotal,
			CreditAmount:  decimal.Zero,
			LineNarration: fmt.Sprintf("IGST refund receivable on export #%s", inv.InvoiceNumber),
		})
	}
	lines = append(lines, JournalLineDraft{
		AccountID:     arAcc.ID,
		CustomerID:    &inv.CustomerID,
		DebitAmount:   arAmount,
		CreditAmount:  decimal.Zero,
		LineNarration: fmt.Sprintf("Invoice #%s issued to %s", inv.InvoiceNumber, inv.CustomerName),
	})

	// 2. Credit Sales Revenue with Taxable Subtotal
	lines = append(lines, JournalLineDraft{
		AccountID:     salesAcc.ID,
		DebitAmount:   decimal.Zero,
		CreditAmount:  inv.TaxableAmount,
		LineNarration: fmt.Sprintf("Sales revenue for #%s", inv.InvoiceNumber),
	})

	// 3. Credit GST Output Accounts
	if inv.CGSTTotal.GreaterThan(decimal.Zero) {
		cgstAcc, err := s.GetAccountByCode(ctx, inv.TenantID, "2210")
		if err == nil {
			lines = append(lines, JournalLineDraft{
				AccountID:     cgstAcc.ID,
				DebitAmount:   decimal.Zero,
				CreditAmount:  inv.CGSTTotal,
				LineNarration: fmt.Sprintf("CGST Output on #%s", inv.InvoiceNumber),
			})
		}
	}
	sgstAmount := inv.SGSTTotal
	if inv.UTGSTTotal.GreaterThan(decimal.Zero) {
		utgstAcc, err := s.GetAccountByCode(ctx, inv.TenantID, "2240")
		if err == nil {
			lines = append(lines, JournalLineDraft{
				AccountID:     utgstAcc.ID,
				DebitAmount:   decimal.Zero,
				CreditAmount:  inv.UTGSTTotal,
				LineNarration: fmt.Sprintf("UTGST Output on #%s", inv.InvoiceNumber),
			})
		}
		sgstAmount = sgstAmount.Sub(inv.UTGSTTotal)
	}
	if sgstAmount.GreaterThan(decimal.Zero) {
		sgstAcc, err := s.GetAccountByCode(ctx, inv.TenantID, "2220")
		if err == nil {
			lines = append(lines, JournalLineDraft{
				AccountID:     sgstAcc.ID,
				DebitAmount:   decimal.Zero,
				CreditAmount:  sgstAmount,
				LineNarration: fmt.Sprintf("SGST Output on #%s", inv.InvoiceNumber),
			})
		}
	}
	if inv.IGSTTotal.GreaterThan(decimal.Zero) {
		igstAcc, err := s.GetAccountByCode(ctx, inv.TenantID, "2230")
		if err == nil {
			lines = append(lines, JournalLineDraft{
				AccountID:     igstAcc.ID,
				DebitAmount:   decimal.Zero,
				CreditAmount:  inv.IGSTTotal,
				LineNarration: fmt.Sprintf("IGST Output on #%s", inv.InvoiceNumber),
			})
		}
	}

	// 4. Handle Round-off
	if !inv.RoundOff.IsZero() {
		roundAcc, err := s.GetAccountByCode(ctx, inv.TenantID, "5100")
		if err == nil {
			if inv.RoundOff.GreaterThan(decimal.Zero) {
				// Positive round-off added to total: Credit round-off income
				lines = append(lines, JournalLineDraft{
					AccountID:     roundAcc.ID,
					DebitAmount:   decimal.Zero,
					CreditAmount:  inv.RoundOff,
					LineNarration: fmt.Sprintf("Round-off on #%s", inv.InvoiceNumber),
				})
			} else {
				// Negative round-off deducted from total: Debit round-off expense
				lines = append(lines, JournalLineDraft{
					AccountID:     roundAcc.ID,
					DebitAmount:   inv.RoundOff.Abs(),
					CreditAmount:  decimal.Zero,
					LineNarration: fmt.Sprintf("Round-off deduction on #%s", inv.InvoiceNumber),
				})
			}
		}
	}

	return s.insertJournalEntry(ctx, tx, inv.TenantID, model.RefInvoice, &inv.ID, fmt.Sprintf("Invoice #%s to %s", inv.InvoiceNumber, inv.CustomerName), lines, userID)
}

// PostPaymentJournal generates a balanced double-entry voucher when a customer payment is received
func (s *Service) PostPaymentJournal(ctx context.Context, tx pgx.Tx, payment *model.Payment, userID *uuid.UUID) (*model.JournalEntry, error) {
	if err := s.EnsureStandardAccounts(ctx, payment.TenantID); err != nil {
		return nil, err
	}

	// Determine receiving account: Cash or Bank
	accCode := "1200" // Default Bank
	if payment.PaymentMethod == model.PaymentMethodCash {
		accCode = "1100" // Cash in Hand
	}
	recAcc, err := s.GetAccountByCode(ctx, payment.TenantID, accCode)
	if err != nil {
		return nil, err
	}

	arAcc, err := s.GetAccountByCode(ctx, payment.TenantID, "1300")
	if err != nil {
		return nil, err
	}

	// Foreign-currency receipt: the bank is debited with what it actually credited; the difference to the settlement
	// value (at the invoice rate) is a forex gain or loss.
	bankAmount := payment.Amount
	if payment.INRReceived != nil && payment.INRReceived.GreaterThan(decimal.Zero) {
		bankAmount = *payment.INRReceived
	}
	lines := []JournalLineDraft{
		// 1. Debit Bank or Cash account
		{
			AccountID:     recAcc.ID,
			DebitAmount:   bankAmount,
			CreditAmount:  decimal.Zero,
			LineNarration: fmt.Sprintf("Payment #%s via %s (Ref: %s)", payment.PaymentNumber, payment.PaymentMethod, payment.ReferenceNumber),
		},
		// 2. Credit Accounts Receivable (Customer Sub-ledger)
		{
			AccountID:     arAcc.ID,
			CustomerID:    &payment.CustomerID,
			DebitAmount:   decimal.Zero,
			CreditAmount:  payment.Amount,
			LineNarration: fmt.Sprintf("Settlement on invoice via payment #%s", payment.PaymentNumber),
		},
	}

	if diff := bankAmount.Sub(payment.Amount); !diff.IsZero() {
		code, debit, credit := "4500", decimal.Zero, diff
		if diff.IsNegative() {
			code, debit, credit = "5500", diff.Abs(), decimal.Zero
		}
		fxAcc, ferr := s.GetAccountByCode(ctx, payment.TenantID, code)
		if ferr != nil {
			return nil, ferr
		}
		lines = append(lines, JournalLineDraft{AccountID: fxAcc.ID, DebitAmount: debit, CreditAmount: credit,
			LineNarration: fmt.Sprintf("Forex difference on payment #%s", payment.PaymentNumber)})
	}

	return s.insertJournalEntry(ctx, tx, payment.TenantID, model.RefPayment, &payment.ID, fmt.Sprintf("Payment #%s received via %s", payment.PaymentNumber, payment.PaymentMethod), lines, userID)
}

// PostCancellationJournal generates a balanced reversal voucher when an invoice is cancelled
func (s *Service) PostCancellationJournal(ctx context.Context, tx pgx.Tx, inv *model.Invoice, reason string, userID *uuid.UUID) (*model.JournalEntry, error) {
	if err := s.EnsureStandardAccounts(ctx, inv.TenantID); err != nil {
		return nil, err
	}

	arAcc, err := s.GetAccountByCode(ctx, inv.TenantID, "1300")
	if err != nil {
		return nil, err
	}

	isExport, igstRefund := tradeFlags(ctx, tx, inv.ID)
	salesCode := "4100"
	if inv.IsInterState {
		salesCode = "4200"
	}
	if isExport {
		salesCode = "4400"
	}
	salesAcc, err := s.GetAccountByCode(ctx, inv.TenantID, salesCode)
	if err != nil {
		return nil, err
	}

	var lines []JournalLineDraft

	// Reversal: Credit Accounts Receivable (Customer); for IGST-refund exports the refund receivable is reversed too
	arAmount := inv.GrandTotal
	if igstRefund && inv.IGSTTotal.GreaterThan(decimal.Zero) {
		refundAcc, rerr := s.GetAccountByCode(ctx, inv.TenantID, "1350")
		if rerr != nil {
			return nil, rerr
		}
		arAmount = inv.GrandTotal.Sub(inv.IGSTTotal)
		lines = append(lines, JournalLineDraft{
			AccountID:     refundAcc.ID,
			DebitAmount:   decimal.Zero,
			CreditAmount:  inv.IGSTTotal,
			LineNarration: fmt.Sprintf("IGST refund receivable reversal on #%s", inv.InvoiceNumber),
		})
	}
	lines = append(lines, JournalLineDraft{
		AccountID:     arAcc.ID,
		CustomerID:    &inv.CustomerID,
		DebitAmount:   decimal.Zero,
		CreditAmount:  arAmount,
		LineNarration: fmt.Sprintf("Reversal of #%s (Cancelled: %s)", inv.InvoiceNumber, reason),
	})

	// Reversal: Debit Sales Revenue
	lines = append(lines, JournalLineDraft{
		AccountID:     salesAcc.ID,
		DebitAmount:   inv.TaxableAmount,
		CreditAmount:  decimal.Zero,
		LineNarration: fmt.Sprintf("Revenue reversal on cancelled #%s", inv.InvoiceNumber),
	})

	// Reversal: Debit GST Output Accounts
	if inv.CGSTTotal.GreaterThan(decimal.Zero) {
		cgstAcc, err := s.GetAccountByCode(ctx, inv.TenantID, "2210")
		if err == nil {
			lines = append(lines, JournalLineDraft{
				AccountID:     cgstAcc.ID,
				DebitAmount:   inv.CGSTTotal,
				CreditAmount:  decimal.Zero,
				LineNarration: fmt.Sprintf("CGST Output reversal on #%s", inv.InvoiceNumber),
			})
		}
	}
	revSGST := inv.SGSTTotal
	if inv.UTGSTTotal.GreaterThan(decimal.Zero) {
		utgstAcc, err := s.GetAccountByCode(ctx, inv.TenantID, "2240")
		if err == nil {
			lines = append(lines, JournalLineDraft{
				AccountID:     utgstAcc.ID,
				DebitAmount:   inv.UTGSTTotal,
				CreditAmount:  decimal.Zero,
				LineNarration: fmt.Sprintf("UTGST Output reversal on #%s", inv.InvoiceNumber),
			})
		}
		revSGST = revSGST.Sub(inv.UTGSTTotal)
	}
	if revSGST.GreaterThan(decimal.Zero) {
		sgstAcc, err := s.GetAccountByCode(ctx, inv.TenantID, "2220")
		if err == nil {
			lines = append(lines, JournalLineDraft{
				AccountID:     sgstAcc.ID,
				DebitAmount:   revSGST,
				CreditAmount:  decimal.Zero,
				LineNarration: fmt.Sprintf("SGST Output reversal on #%s", inv.InvoiceNumber),
			})
		}
	}
	if inv.IGSTTotal.GreaterThan(decimal.Zero) {
		igstAcc, err := s.GetAccountByCode(ctx, inv.TenantID, "2230")
		if err == nil {
			lines = append(lines, JournalLineDraft{
				AccountID:     igstAcc.ID,
				DebitAmount:   inv.IGSTTotal,
				CreditAmount:  decimal.Zero,
				LineNarration: fmt.Sprintf("IGST Output reversal on #%s", inv.InvoiceNumber),
			})
		}
	}

	// Reversal: Round-off
	if !inv.RoundOff.IsZero() {
		roundAcc, err := s.GetAccountByCode(ctx, inv.TenantID, "5100")
		if err == nil {
			if inv.RoundOff.GreaterThan(decimal.Zero) {
				lines = append(lines, JournalLineDraft{
					AccountID:     roundAcc.ID,
					DebitAmount:   inv.RoundOff,
					CreditAmount:  decimal.Zero,
					LineNarration: fmt.Sprintf("Round-off reversal on #%s", inv.InvoiceNumber),
				})
			} else {
				lines = append(lines, JournalLineDraft{
					AccountID:     roundAcc.ID,
					DebitAmount:   decimal.Zero,
					CreditAmount:  inv.RoundOff.Abs(),
					LineNarration: fmt.Sprintf("Round-off reversal on #%s", inv.InvoiceNumber),
				})
			}
		}
	}

	return s.insertJournalEntry(ctx, tx, inv.TenantID, model.RefCancellation, &inv.ID, fmt.Sprintf("Reversal of Invoice #%s: %s", inv.InvoiceNumber, reason), lines, userID)
}

// PostExpenseJournal generates a balanced double-entry voucher when a company expense is paid
func (s *Service) PostExpenseJournal(ctx context.Context, tx pgx.Tx, exp *model.Expense, userID *uuid.UUID) (*model.JournalEntry, error) {
	if err := s.EnsureStandardAccounts(ctx, exp.TenantID); err != nil {
		return nil, err
	}

	// Expense account: default to 5400 Operating Expenses, fallback to 5300 COGS
	expAcc, err := s.GetAccountByCode(ctx, exp.TenantID, "5400")
	if err != nil {
		expAcc, err = s.GetAccountByCode(ctx, exp.TenantID, "5300")
		if err != nil {
			return nil, err
		}
	}

	// Paying account: 1100 Cash or 1200 Bank
	payAccCode := "1100"
	if exp.PaymentMode == "BANK_TRANSFER" || exp.PaymentMode == "CARD" || exp.PaymentMode == "UPI" || exp.PaymentMode == "CHEQUE" {
		payAccCode = "1200"
	}
	payAcc, err := s.GetAccountByCode(ctx, exp.TenantID, payAccCode)
	if err != nil {
		return nil, err
	}

	lines := []JournalLineDraft{
		{
			AccountID:     expAcc.ID,
			DebitAmount:   exp.Amount,
			CreditAmount:  decimal.Zero,
			LineNarration: fmt.Sprintf("Expense [%s]: %s (Payee: %s)", exp.Category, exp.Description, exp.PayeeName),
		},
		{
			AccountID:     payAcc.ID,
			DebitAmount:   decimal.Zero,
			CreditAmount:  exp.Amount,
			LineNarration: fmt.Sprintf("Disbursement for expense #%s via %s", exp.ExpenseNumber, exp.PaymentMode),
		},
	}

	return s.insertJournalEntry(ctx, tx, exp.TenantID, "EXPENSE", &exp.ID, fmt.Sprintf("Expense #%s: %s", exp.ExpenseNumber, exp.Category), lines, userID)
}

// PostCreditNoteJournal generates a balanced double-entry reversal voucher when a credit note is issued
func (s *Service) PostCreditNoteJournal(ctx context.Context, tx pgx.Tx, cn *model.CreditNote, userID *uuid.UUID) (*model.JournalEntry, error) {
	if err := s.EnsureStandardAccounts(ctx, cn.TenantID); err != nil {
		return nil, err
	}

	salesAcc, err := s.GetAccountByCode(ctx, cn.TenantID, "4100")
	if err != nil {
		return nil, err
	}

	arAcc, err := s.GetAccountByCode(ctx, cn.TenantID, "1300")
	if err != nil {
		return nil, err
	}

	var lines []JournalLineDraft

	if cn.TaxableAmount.GreaterThan(decimal.Zero) {
		lines = append(lines, JournalLineDraft{
			AccountID:     salesAcc.ID,
			DebitAmount:   cn.TaxableAmount,
			CreditAmount:  decimal.Zero,
			LineNarration: fmt.Sprintf("Sales return / credit reduction for CN #%s (Inv: %s)", cn.CreditNoteNumber, cn.InvoiceNumber),
		})
	}

	if cn.CGSTTotal.GreaterThan(decimal.Zero) {
		cgstAcc, err := s.GetAccountByCode(ctx, cn.TenantID, "2210")
		if err == nil {
			lines = append(lines, JournalLineDraft{
				AccountID:     cgstAcc.ID,
				DebitAmount:   cn.CGSTTotal,
				CreditAmount:  decimal.Zero,
				LineNarration: fmt.Sprintf("Output CGST reversal on CN #%s", cn.CreditNoteNumber),
			})
		}
	}
	if cn.SGSTTotal.GreaterThan(decimal.Zero) {
		sgstAcc, err := s.GetAccountByCode(ctx, cn.TenantID, "2220")
		if err == nil {
			lines = append(lines, JournalLineDraft{
				AccountID:     sgstAcc.ID,
				DebitAmount:   cn.SGSTTotal,
				CreditAmount:  decimal.Zero,
				LineNarration: fmt.Sprintf("Output SGST reversal on CN #%s", cn.CreditNoteNumber),
			})
		}
	}
	if cn.IGSTTotal.GreaterThan(decimal.Zero) {
		igstAcc, err := s.GetAccountByCode(ctx, cn.TenantID, "2230")
		if err == nil {
			lines = append(lines, JournalLineDraft{
				AccountID:     igstAcc.ID,
				DebitAmount:   cn.IGSTTotal,
				CreditAmount:  decimal.Zero,
				LineNarration: fmt.Sprintf("Output IGST reversal on CN #%s", cn.CreditNoteNumber),
			})
		}
	}

	if !cn.RoundOff.IsZero() {
		roundAcc, err := s.GetAccountByCode(ctx, cn.TenantID, "5100")
		if err == nil {
			if cn.RoundOff.IsPositive() {
				lines = append(lines, JournalLineDraft{
					AccountID:     roundAcc.ID,
					DebitAmount:   cn.RoundOff,
					CreditAmount:  decimal.Zero,
					LineNarration: fmt.Sprintf("Round-off difference on CN #%s", cn.CreditNoteNumber),
				})
			} else {
				lines = append(lines, JournalLineDraft{
					AccountID:     roundAcc.ID,
					DebitAmount:   decimal.Zero,
					CreditAmount:  cn.RoundOff.Abs(),
					LineNarration: fmt.Sprintf("Round-off difference on CN #%s", cn.CreditNoteNumber),
				})
			}
		}
	}

	lines = append(lines, JournalLineDraft{
		AccountID:     arAcc.ID,
		DebitAmount:   decimal.Zero,
		CreditAmount:  cn.GrandTotal,
		LineNarration: fmt.Sprintf("AR adjustment for customer %s on CN #%s", cn.CustomerName, cn.CreditNoteNumber),
	})

	return s.insertJournalEntry(ctx, tx, cn.TenantID, "CREDIT_NOTE", &cn.ID, fmt.Sprintf("Credit Note #%s for %s (%s)", cn.CreditNoteNumber, cn.CustomerName, cn.ReasonCode), lines, userID)
}

// PostDebitNoteJournal generates a balanced double-entry voucher when a debit note is issued to vendor
func (s *Service) PostDebitNoteJournal(ctx context.Context, tx pgx.Tx, dn *model.DebitNote, userID *uuid.UUID) (*model.JournalEntry, error) {
	if err := s.EnsureStandardAccounts(ctx, dn.TenantID); err != nil {
		return nil, err
	}

	apAcc, err := s.GetAccountByCode(ctx, dn.TenantID, "2100")
	if err != nil {
		return nil, err
	}

	cogsAcc, err := s.GetAccountByCode(ctx, dn.TenantID, "1400")
	if err != nil {
		cogsAcc, err = s.GetAccountByCode(ctx, dn.TenantID, "5300")
		if err != nil {
			return nil, err
		}
	}

	var lines []JournalLineDraft

	lines = append(lines, JournalLineDraft{
		AccountID:     apAcc.ID,
		DebitAmount:   dn.GrandTotal,
		CreditAmount:  decimal.Zero,
		LineNarration: fmt.Sprintf("Debit Note #%s reduction of AP for vendor %s", dn.DebitNoteNumber, dn.SupplierName),
	})

	if dn.TaxableAmount.GreaterThan(decimal.Zero) {
		lines = append(lines, JournalLineDraft{
			AccountID:     cogsAcc.ID,
			DebitAmount:   decimal.Zero,
			CreditAmount:  dn.TaxableAmount,
			LineNarration: fmt.Sprintf("Purchase return / vendor rebate on DN #%s", dn.DebitNoteNumber),
		})
	}

	if dn.CGSTTotal.GreaterThan(decimal.Zero) {
		cgstAcc, err := s.GetAccountByCode(ctx, dn.TenantID, "2210")
		if err == nil {
			lines = append(lines, JournalLineDraft{
				AccountID:     cgstAcc.ID,
				DebitAmount:   decimal.Zero,
				CreditAmount:  dn.CGSTTotal,
				LineNarration: fmt.Sprintf("CGST ITC reversal on DN #%s", dn.DebitNoteNumber),
			})
		}
	}
	if dn.SGSTTotal.GreaterThan(decimal.Zero) {
		sgstAcc, err := s.GetAccountByCode(ctx, dn.TenantID, "2220")
		if err == nil {
			lines = append(lines, JournalLineDraft{
				AccountID:     sgstAcc.ID,
				DebitAmount:   decimal.Zero,
				CreditAmount:  dn.SGSTTotal,
				LineNarration: fmt.Sprintf("SGST ITC reversal on DN #%s", dn.DebitNoteNumber),
			})
		}
	}
	if dn.IGSTTotal.GreaterThan(decimal.Zero) {
		igstAcc, err := s.GetAccountByCode(ctx, dn.TenantID, "2230")
		if err == nil {
			lines = append(lines, JournalLineDraft{
				AccountID:     igstAcc.ID,
				DebitAmount:   decimal.Zero,
				CreditAmount:  dn.IGSTTotal,
				LineNarration: fmt.Sprintf("IGST ITC reversal on DN #%s", dn.DebitNoteNumber),
			})
		}
	}

	if !dn.RoundOff.IsZero() {
		roundAcc, err := s.GetAccountByCode(ctx, dn.TenantID, "5100")
		if err == nil {
			if dn.RoundOff.IsPositive() {
				lines = append(lines, JournalLineDraft{
					AccountID:     roundAcc.ID,
					DebitAmount:   decimal.Zero,
					CreditAmount:  dn.RoundOff,
					LineNarration: fmt.Sprintf("Round-off difference on DN #%s", dn.DebitNoteNumber),
				})
			} else {
				lines = append(lines, JournalLineDraft{
					AccountID:     roundAcc.ID,
					DebitAmount:   dn.RoundOff.Abs(),
					CreditAmount:  decimal.Zero,
					LineNarration: fmt.Sprintf("Round-off difference on DN #%s", dn.DebitNoteNumber),
				})
			}
		}
	}

	return s.insertJournalEntry(ctx, tx, dn.TenantID, "DEBIT_NOTE", &dn.ID, fmt.Sprintf("Debit Note #%s for %s (%s)", dn.DebitNoteNumber, dn.SupplierName, dn.ReasonCode), lines, userID)
}

func (s *Service) insertJournalEntry(
	ctx context.Context,
	tx pgx.Tx,
	tenantID uuid.UUID,
	refType string,
	refID *uuid.UUID,
	narration string,
	lines []JournalLineDraft,
	userID *uuid.UUID,
) (*model.JournalEntry, error) {
	totalDebit := decimal.Zero
	totalCredit := decimal.Zero

	for _, l := range lines {
		totalDebit = totalDebit.Add(l.DebitAmount)
		totalCredit = totalCredit.Add(l.CreditAmount)
	}

	// Accounting Integrity Check: Sum of Debits MUST equal Sum of Credits
	if !totalDebit.Equal(totalCredit) {
		return nil, fmt.Errorf("unbalanced journal entry rejected: debits (₹%s) != credits (₹%s)", totalDebit.StringFixed(2), totalCredit.StringFixed(2))
	}

	entryNumber, err := s.NextJournalNumber(ctx, tx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate journal number: %w", err)
	}

	entryID := uuid.New()
	entry := &model.JournalEntry{
		ID:            entryID,
		TenantID:      tenantID,
		EntryNumber:   entryNumber,
		EntryDate:     time.Now(),
		ReferenceType: refType,
		ReferenceID:   refID,
		Narration:     narration,
		TotalDebit:    totalDebit,
		TotalCredit:   totalCredit,
		Status:        "POSTED",
		CreatedBy:     userID,
		CreatedAt:     time.Now(),
	}

	entryQuery := `
		INSERT INTO journal_entries (id, tenant_id, entry_number, entry_date, reference_type, reference_id, narration, total_debit, total_credit, status, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err = tx.Exec(ctx, entryQuery,
		entry.ID, entry.TenantID, entry.EntryNumber, entry.EntryDate, entry.ReferenceType,
		entry.ReferenceID, entry.Narration, entry.TotalDebit, entry.TotalCredit,
		entry.Status, entry.CreatedBy, entry.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert journal entry header: %w", err)
	}

	// Insert lines and update account balances
	for _, l := range lines {
		lineID := uuid.New()
		lineQuery := `
			INSERT INTO journal_lines (id, tenant_id, journal_entry_id, account_id, customer_id, debit_amount, credit_amount, line_narration)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		_, err = tx.Exec(ctx, lineQuery, lineID, tenantID, entryID, l.AccountID, l.CustomerID, l.DebitAmount, l.CreditAmount, l.LineNarration)
		if err != nil {
			return nil, fmt.Errorf("failed to insert journal line: %w", err)
		}

		// Update running balance of account
		// Assets & Expenses: Balance = Balance + (Debit - Credit)
		// Liabilities, Equity, Revenue: Balance = Balance + (Credit - Debit)
		updateBalQuery := `
			UPDATE accounts
			SET current_balance = current_balance + CASE 
				WHEN account_type IN ('ASSET', 'EXPENSE') THEN ($1::NUMERIC - $2::NUMERIC)
				ELSE ($2::NUMERIC - $1::NUMERIC)
			END,
			updated_at = NOW()
			WHERE id = $3 AND tenant_id = $4
		`
		_, err = tx.Exec(ctx, updateBalQuery, l.DebitAmount, l.CreditAmount, l.AccountID, tenantID)
		if err != nil {
			return nil, fmt.Errorf("failed to update account balance: %w", err)
		}
	}

	return entry, nil
}

// ListAccounts returns all Chart of Accounts items for a tenant
func (s *Service) ListAccounts(ctx context.Context, tenantID uuid.UUID) ([]model.Account, error) {
	_ = s.EnsureStandardAccounts(ctx, tenantID)

	query := `
		SELECT id, tenant_id, account_code, account_name, account_type, sub_category, description, is_system, is_active, current_balance, created_at, updated_at
		FROM accounts
		WHERE tenant_id = $1
		ORDER BY account_code ASC
	`
	rows, err := s.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.Account
	for rows.Next() {
		var a model.Account
		if err := rows.Scan(
			&a.ID, &a.TenantID, &a.AccountCode, &a.AccountName, &a.AccountType,
			&a.SubCategory, &a.Description, &a.IsSystem, &a.IsActive,
			&a.CurrentBalance, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}

type GeneralLedgerEntry struct {
	EntryID        uuid.UUID       `json:"entry_id"`
	EntryNumber    string          `json:"entry_number"`
	EntryDate      time.Time       `json:"entry_date"`
	ReferenceType  string          `json:"reference_type"`
	ReferenceID    *uuid.UUID      `json:"reference_id,omitempty"`
	AccountID      uuid.UUID       `json:"account_id"`
	AccountCode    string          `json:"account_code"`
	AccountName    string          `json:"account_name"`
	CustomerName   string          `json:"customer_name,omitempty"`
	DebitAmount    decimal.Decimal `json:"debit_amount"`
	CreditAmount   decimal.Decimal `json:"credit_amount"`
	LineNarration  string          `json:"line_narration"`
	RunningBalance decimal.Decimal `json:"running_balance"`
}

// GetGeneralLedger returns detailed general ledger entries for a tenant, optionally filtered by account
func (s *Service) GetGeneralLedger(ctx context.Context, tenantID uuid.UUID, accountID *uuid.UUID, fromDate, toDate *time.Time) ([]GeneralLedgerEntry, error) {
	query := `
		SELECT 
			je.id, je.entry_number, je.entry_date, je.reference_type, je.reference_id,
			a.id, a.account_code, a.account_name, COALESCE(c.name, ''),
			jl.debit_amount, jl.credit_amount, jl.line_narration
		FROM journal_lines jl
		JOIN journal_entries je ON jl.journal_entry_id = je.id
		JOIN accounts a ON jl.account_id = a.id
		LEFT JOIN customers c ON jl.customer_id = c.id
		WHERE jl.tenant_id = $1
		  AND ($2::UUID IS NULL OR jl.account_id = $2)
		  AND ($3::DATE IS NULL OR je.entry_date >= $3)
		  AND ($4::DATE IS NULL OR je.entry_date <= $4)
		ORDER BY je.entry_date ASC, je.created_at ASC, jl.created_at ASC
	`
	rows, err := s.db.Query(ctx, query, tenantID, accountID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []GeneralLedgerEntry
	runningBal := decimal.Zero
	if fromDate != nil {
		openQuery := `
			SELECT COALESCE(SUM(jl.debit_amount - jl.credit_amount), 0)
			FROM journal_lines jl
			JOIN journal_entries je ON jl.journal_entry_id = je.id
			WHERE jl.tenant_id = $1
			  AND ($2::UUID IS NULL OR jl.account_id = $2)
			  AND je.entry_date < $3
		`
		_ = s.db.QueryRow(ctx, openQuery, tenantID, accountID, fromDate).Scan(&runningBal)
	}

	for rows.Next() {
		var e GeneralLedgerEntry
		err := rows.Scan(
			&e.EntryID, &e.EntryNumber, &e.EntryDate, &e.ReferenceType, &e.ReferenceID,
			&e.AccountID, &e.AccountCode, &e.AccountName, &e.CustomerName,
			&e.DebitAmount, &e.CreditAmount, &e.LineNarration,
		)
		if err != nil {
			return nil, err
		}

		runningBal = runningBal.Add(e.DebitAmount).Sub(e.CreditAmount)
		e.RunningBalance = runningBal
		entries = append(entries, e)
	}

	return entries, nil
}

type CustomerStatementItem struct {
	Date           time.Time       `json:"date"`
	VoucherType    string          `json:"voucher_type"`
	VoucherNumber  string          `json:"voucher_number"`
	ReferenceNumber string         `json:"reference_number"`
	Narration      string          `json:"narration"`
	DebitAmount    decimal.Decimal `json:"debit_amount"`
	CreditAmount   decimal.Decimal `json:"credit_amount"`
	Balance        decimal.Decimal `json:"balance"`
}

type CustomerStatement struct {
	CustomerID      uuid.UUID               `json:"customer_id"`
	CustomerName    string                  `json:"customer_name"`
	CustomerGSTIN   string                  `json:"customer_gstin"`
	CustomerPhone   string                  `json:"customer_phone"`
	CustomerAddress string                  `json:"customer_address"`
	OpeningBalance  decimal.Decimal         `json:"opening_balance"`
	ClosingBalance  decimal.Decimal         `json:"closing_balance"`
	TotalInvoiced   decimal.Decimal         `json:"total_invoiced"`
	TotalPaid       decimal.Decimal         `json:"total_paid"`
	Items           []CustomerStatementItem `json:"items"`
}

// GetCustomerStatement generates an audited customer statement of account
func (s *Service) GetCustomerStatement(ctx context.Context, tenantID, customerID uuid.UUID, fromDate, toDate *time.Time) (*CustomerStatement, error) {
	// Fetch customer details
	custQuery := `SELECT id, name, gstin, phone, billing_address FROM customers WHERE id = $1 AND tenant_id = $2`
	stmt := &CustomerStatement{
		CustomerID: customerID,
	}
	err := s.db.QueryRow(ctx, custQuery, customerID, tenantID).Scan(
		&stmt.CustomerID, &stmt.CustomerName, &stmt.CustomerGSTIN, &stmt.CustomerPhone, &stmt.CustomerAddress,
	)
	if err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}

	openingBal := decimal.Zero
	if fromDate != nil {
		openQuery := `
			SELECT COALESCE(SUM(jl.debit_amount - jl.credit_amount), 0)
			FROM journal_lines jl
			JOIN journal_entries je ON jl.journal_entry_id = je.id
			WHERE jl.tenant_id = $1 AND jl.customer_id = $2
			  AND je.entry_date < $3
		`
		_ = s.db.QueryRow(ctx, openQuery, tenantID, customerID, fromDate).Scan(&openingBal)
	}

	stmt.OpeningBalance = openingBal

	query := `
		SELECT 
			je.entry_date, je.reference_type, je.entry_number,
			COALESCE(inv.invoice_number, pay.payment_number, ''),
			jl.line_narration, jl.debit_amount, jl.credit_amount
		FROM journal_lines jl
		JOIN journal_entries je ON jl.journal_entry_id = je.id
		LEFT JOIN invoices inv ON je.reference_id = inv.id AND je.reference_type = 'INVOICE'
		LEFT JOIN payments pay ON je.reference_id = pay.id AND je.reference_type = 'PAYMENT'
		WHERE jl.tenant_id = $1 AND jl.customer_id = $2
		  AND ($3::DATE IS NULL OR je.entry_date >= $3)
		  AND ($4::DATE IS NULL OR je.entry_date <= $4)
		ORDER BY je.entry_date ASC, je.created_at ASC
	`
	rows, err := s.db.Query(ctx, query, tenantID, customerID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runningBal := openingBal
	totalInvoiced := decimal.Zero
	totalPaid := decimal.Zero

	for rows.Next() {
		var item CustomerStatementItem
		err := rows.Scan(
			&item.Date, &item.VoucherType, &item.VoucherNumber, &item.ReferenceNumber,
			&item.Narration, &item.DebitAmount, &item.CreditAmount,
		)
		if err != nil {
			return nil, err
		}

		runningBal = runningBal.Add(item.DebitAmount).Sub(item.CreditAmount)
		item.Balance = runningBal

		totalInvoiced = totalInvoiced.Add(item.DebitAmount)
		totalPaid = totalPaid.Add(item.CreditAmount)

		stmt.Items = append(stmt.Items, item)
	}

	stmt.TotalInvoiced = totalInvoiced
	stmt.TotalPaid = totalPaid
	stmt.ClosingBalance = runningBal

	return stmt, nil
}

type TrialBalanceRow struct {
	AccountCode  string          `json:"account_code"`
	AccountName  string          `json:"account_name"`
	AccountType  string          `json:"account_type"`
	DebitAmount  decimal.Decimal `json:"debit_amount"`
	CreditAmount decimal.Decimal `json:"credit_amount"`
}

type TrialBalanceReport struct {
	AsOfDate     time.Time         `json:"as_of_date"`
	TotalDebit   decimal.Decimal   `json:"total_debit"`
	TotalCredit  decimal.Decimal   `json:"total_credit"`
	IsBalanced   bool              `json:"is_balanced"`
	Difference   decimal.Decimal   `json:"difference"`
	Accounts     []TrialBalanceRow `json:"accounts"`
}

// GetTrialBalance computes an audited Trial Balance
func (s *Service) GetTrialBalance(ctx context.Context, tenantID uuid.UUID, asOfDate *time.Time) (*TrialBalanceReport, error) {
	_ = s.EnsureStandardAccounts(ctx, tenantID)

	query := `
		SELECT 
			a.account_code, a.account_name, a.account_type,
			COALESCE(SUM(jl.debit_amount), 0) AS total_debit,
			COALESCE(SUM(jl.credit_amount), 0) AS total_credit
		FROM accounts a
		LEFT JOIN journal_lines jl ON a.id = jl.account_id
		LEFT JOIN journal_entries je ON jl.journal_entry_id = je.id
		WHERE a.tenant_id = $1
		  AND ($2::DATE IS NULL OR je.entry_date <= $2)
		GROUP BY a.id, a.account_code, a.account_name, a.account_type
		HAVING COALESCE(SUM(jl.debit_amount), 0) > 0 OR COALESCE(SUM(jl.credit_amount), 0) > 0
		ORDER BY a.account_code ASC
	`
	rows, err := s.db.Query(ctx, query, tenantID, asOfDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	report := &TrialBalanceReport{
		AsOfDate: time.Now(),
	}
	if asOfDate != nil {
		report.AsOfDate = *asOfDate
	}

	sumDebit := decimal.Zero
	sumCredit := decimal.Zero

	for rows.Next() {
		var r TrialBalanceRow
		var d, c decimal.Decimal
		if err := rows.Scan(&r.AccountCode, &r.AccountName, &r.AccountType, &d, &c); err != nil {
			return nil, err
		}

		// Calculate net balance based on natural account sign
		if r.AccountType == model.AccountTypeAsset || r.AccountType == model.AccountTypeExpense {
			if d.GreaterThanOrEqual(c) {
				r.DebitAmount = d.Sub(c)
				r.CreditAmount = decimal.Zero
			} else {
				r.DebitAmount = decimal.Zero
				r.CreditAmount = c.Sub(d)
			}
		} else {
			if c.GreaterThanOrEqual(d) {
				r.CreditAmount = c.Sub(d)
				r.DebitAmount = decimal.Zero
			} else {
				r.CreditAmount = decimal.Zero
				r.DebitAmount = d.Sub(c)
			}
		}

		sumDebit = sumDebit.Add(r.DebitAmount)
		sumCredit = sumCredit.Add(r.CreditAmount)
		report.Accounts = append(report.Accounts, r)
	}

	report.TotalDebit = sumDebit
	report.TotalCredit = sumCredit
	report.Difference = sumDebit.Sub(sumCredit).Abs()
	report.IsBalanced = report.Difference.IsZero()

	return report, nil
}

type ProfitAndLossReport struct {
	TotalRevenue decimal.Decimal   `json:"total_revenue"`
	TotalExpense decimal.Decimal   `json:"total_expense"`
	NetProfit    decimal.Decimal   `json:"net_profit"`
	Revenues     []TrialBalanceRow `json:"revenues"`
	Expenses     []TrialBalanceRow `json:"expenses"`
}

// GetProfitAndLoss computes Profit and Loss
func (s *Service) GetProfitAndLoss(ctx context.Context, tenantID uuid.UUID, fromDate, toDate *time.Time) (*ProfitAndLossReport, error) {
	query := `
		SELECT 
			a.account_code, a.account_name, a.account_type,
			COALESCE(SUM(jl.debit_amount), 0), COALESCE(SUM(jl.credit_amount), 0)
		FROM accounts a
		JOIN journal_lines jl ON a.id = jl.account_id
		JOIN journal_entries je ON jl.journal_entry_id = je.id
		WHERE a.tenant_id = $1
		  AND a.account_type IN ('REVENUE', 'EXPENSE')
		  AND ($2::DATE IS NULL OR je.entry_date >= $2)
		  AND ($3::DATE IS NULL OR je.entry_date <= $3)
		GROUP BY a.id, a.account_code, a.account_name, a.account_type
		ORDER BY a.account_code ASC
	`
	rows, err := s.db.Query(ctx, query, tenantID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	report := &ProfitAndLossReport{}
	totRev := decimal.Zero
	totExp := decimal.Zero

	for rows.Next() {
		var r TrialBalanceRow
		var d, c decimal.Decimal
		if err := rows.Scan(&r.AccountCode, &r.AccountName, &r.AccountType, &d, &c); err != nil {
			return nil, err
		}

		if r.AccountType == model.AccountTypeRevenue {
			r.CreditAmount = c.Sub(d)
			totRev = totRev.Add(r.CreditAmount)
			report.Revenues = append(report.Revenues, r)
		} else {
			r.DebitAmount = d.Sub(c)
			totExp = totExp.Add(r.DebitAmount)
			report.Expenses = append(report.Expenses, r)
		}
	}

	report.TotalRevenue = totRev
	report.TotalExpense = totExp
	report.NetProfit = totRev.Sub(totExp)

	return report, nil
}

// GetJournalEntryByReference retrieves the journal entry tied to an invoice or payment
func (s *Service) GetJournalEntryByReference(ctx context.Context, tenantID uuid.UUID, refType string, refID uuid.UUID) (*model.JournalEntry, error) {
	entry := &model.JournalEntry{}
	query := `
		SELECT id, tenant_id, entry_number, entry_date, reference_type, reference_id, narration, total_debit, total_credit, status, created_by, created_at
		FROM journal_entries
		WHERE tenant_id = $1 AND reference_type = $2 AND reference_id = $3
		LIMIT 1
	`
	err := s.db.QueryRow(ctx, query, tenantID, refType, refID).Scan(
		&entry.ID, &entry.TenantID, &entry.EntryNumber, &entry.EntryDate,
		&entry.ReferenceType, &entry.ReferenceID, &entry.Narration,
		&entry.TotalDebit, &entry.TotalCredit, &entry.Status,
		&entry.CreatedBy, &entry.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Fetch lines
	linesQuery := `
		SELECT jl.id, jl.tenant_id, jl.journal_entry_id, jl.account_id, a.account_code, a.account_name,
		       jl.customer_id, COALESCE(c.name, ''), jl.debit_amount, jl.credit_amount, jl.line_narration, jl.created_at
		FROM journal_lines jl
		JOIN accounts a ON jl.account_id = a.id
		LEFT JOIN customers c ON jl.customer_id = c.id
		WHERE jl.journal_entry_id = $1 AND jl.tenant_id = $2
		ORDER BY jl.debit_amount DESC, jl.created_at ASC
	`
	rows, err := s.db.Query(ctx, linesQuery, entry.ID, tenantID)
	if err != nil {
		return entry, nil
	}
	defer rows.Close()

	for rows.Next() {
		var l model.JournalLine
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.JournalEntryID, &l.AccountID, &l.AccountCode, &l.AccountName,
			&l.CustomerID, &l.CustomerName, &l.DebitAmount, &l.CreditAmount, &l.LineNarration, &l.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan journal line: %w", err)
		}
		entry.Lines = append(entry.Lines, l)
	}

	return entry, nil
}

type BalanceSheetAccount struct {
	AccountCode string          `json:"account_code"`
	AccountName string          `json:"account_name"`
	SubCategory string          `json:"sub_category"`
	Balance     decimal.Decimal `json:"balance"`
}

type BalanceSheetReport struct {
	AsOfDate                  time.Time             `json:"as_of_date"`
	Assets                    []BalanceSheetAccount `json:"assets"`
	Liabilities               []BalanceSheetAccount `json:"liabilities"`
	Equity                    []BalanceSheetAccount `json:"equity"`
	TotalAssets               decimal.Decimal       `json:"total_assets"`
	TotalLiabilities          decimal.Decimal       `json:"total_liabilities"`
	TotalEquity               decimal.Decimal       `json:"total_equity"`
	NetProfitToDate           decimal.Decimal       `json:"net_profit_to_date"`
	TotalLiabilitiesAndEquity decimal.Decimal       `json:"total_liabilities_and_equity"`
	Difference                decimal.Decimal       `json:"difference"`
	IsBalanced                bool                  `json:"is_balanced"`
}

// GetBalanceSheet computes an audited Balance Sheet (Assets = Liabilities + Equity) as of a given date
func (s *Service) GetBalanceSheet(ctx context.Context, tenantID uuid.UUID, asOfDate *time.Time) (*BalanceSheetReport, error) {
	if asOfDate == nil {
		now := time.Now()
		asOfDate = &now
	}

	query := `
		SELECT 
			a.account_code, a.account_name, a.account_type, a.sub_category,
			COALESCE(SUM(jl.debit_amount), 0), COALESCE(SUM(jl.credit_amount), 0)
		FROM accounts a
		LEFT JOIN journal_lines jl ON a.id = jl.account_id
		LEFT JOIN journal_entries je ON jl.journal_entry_id = je.id AND je.entry_date <= $2
		WHERE a.tenant_id = $1
		GROUP BY a.id, a.account_code, a.account_name, a.account_type, a.sub_category
		ORDER BY a.account_code ASC
	`
	rows, err := s.db.Query(ctx, query, tenantID, asOfDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query balance sheet accounts: %w", err)
	}
	defer rows.Close()

	report := &BalanceSheetReport{
		AsOfDate:         *asOfDate,
		Assets:           []BalanceSheetAccount{},
		Liabilities:      []BalanceSheetAccount{},
		Equity:           []BalanceSheetAccount{},
		TotalAssets:      decimal.Zero,
		TotalLiabilities: decimal.Zero,
		TotalEquity:      decimal.Zero,
	}

	var totRev, totExp decimal.Decimal

	for rows.Next() {
		var code, name, accType, subCat string
		var debit, credit decimal.Decimal
		if err := rows.Scan(&code, &name, &accType, &subCat, &debit, &credit); err != nil {
			return nil, err
		}

		switch accType {
		case model.AccountTypeAsset:
			bal := debit.Sub(credit)
			if !bal.IsZero() {
				report.Assets = append(report.Assets, BalanceSheetAccount{
					AccountCode: code,
					AccountName: name,
					SubCategory: subCat,
					Balance:     bal,
				})
				report.TotalAssets = report.TotalAssets.Add(bal)
			}
		case model.AccountTypeLiability:
			bal := credit.Sub(debit)
			if !bal.IsZero() {
				report.Liabilities = append(report.Liabilities, BalanceSheetAccount{
					AccountCode: code,
					AccountName: name,
					SubCategory: subCat,
					Balance:     bal,
				})
				report.TotalLiabilities = report.TotalLiabilities.Add(bal)
			}
		case model.AccountTypeEquity:
			bal := credit.Sub(debit)
			if !bal.IsZero() {
				report.Equity = append(report.Equity, BalanceSheetAccount{
					AccountCode: code,
					AccountName: name,
					SubCategory: subCat,
					Balance:     bal,
				})
				report.TotalEquity = report.TotalEquity.Add(bal)
			}
		case model.AccountTypeRevenue:
			totRev = totRev.Add(credit.Sub(debit))
		case model.AccountTypeExpense:
			totExp = totExp.Add(debit.Sub(credit))
		}
	}

	// Retained earnings / net profit to date transfers to Equity
	report.NetProfitToDate = totRev.Sub(totExp)
	effectiveEquity := report.TotalEquity.Add(report.NetProfitToDate)
	report.TotalLiabilitiesAndEquity = report.TotalLiabilities.Add(effectiveEquity)
	report.Difference = report.TotalAssets.Sub(report.TotalLiabilitiesAndEquity).Abs()
	report.IsBalanced = report.Difference.IsZero()

	return report, nil
}


// PostManualJournal validates and posts a caller-supplied voucher inside tx. It rejects lines that
// reference accounts outside the tenant, lines that are neither a pure debit nor a pure credit,
// negative amounts and unbalanced vouchers, then delegates to the same posting path used by invoices
// (journal numbering, immutable ledger, account type aware balance updates).
func (s *Service) PostManualJournal(
	ctx context.Context,
	tx pgx.Tx,
	tenantID uuid.UUID,
	refType string,
	refID *uuid.UUID,
	narration string,
	lines []JournalLineDraft,
	userID *uuid.UUID,
) (*model.JournalEntry, error) {
	if len(lines) < 2 {
		return nil, fmt.Errorf("a journal entry needs at least two lines")
	}
	for i, l := range lines {
		if l.DebitAmount.IsNegative() || l.CreditAmount.IsNegative() {
			return nil, fmt.Errorf("line %d: negative amounts are not allowed", i+1)
		}
		hasDr, hasCr := l.DebitAmount.IsPositive(), l.CreditAmount.IsPositive()
		if hasDr == hasCr {
			return nil, fmt.Errorf("line %d: exactly one of debit or credit must be greater than zero", i+1)
		}
		var ok bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM accounts WHERE id = $1 AND tenant_id = $2 AND is_active = TRUE)", l.AccountID, tenantID).Scan(&ok); err != nil {
			return nil, fmt.Errorf("failed to verify account: %w", err)
		}
		if !ok {
			return nil, fmt.Errorf("line %d: account not found for this company", i+1)
		}
	}
	return s.insertJournalEntry(ctx, tx, tenantID, refType, refID, narration, lines, userID)
}

// tradeFlags reports whether the invoice is an export and whether IGST paid on it is claimed as a refund.
func tradeFlags(ctx context.Context, tx pgx.Tx, invoiceID uuid.UUID) (isExport, igstRefund bool) {
	err := tx.QueryRow(ctx, `SELECT TRUE, igst_refund_claim FROM invoice_trade_details WHERE invoice_id = $1`, invoiceID).Scan(&isExport, &igstRefund)
	if err != nil {
		return false, false
	}
	return isExport, igstRefund
}

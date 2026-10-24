package finance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var buNames = map[uint]string{
	1: "Travel, Tours & Visa Ops",
	2: "Healthcare Platform & Clinical Pods",
	3: "Real Estate & Property Inventory",
	4: "E-Commerce & Print-On-Demand",
	5: "Software & IT Consulting",
}

type Service interface {
	CreateCostCenter(cc *CostCenter) (*CostCenter, error)
	ListCostCenters() ([]CostCenter, error)
	RecordEntry(entry *FinancialLedgerEntry) (*FinancialLedgerEntry, error)
	ListEntries(buID uint) ([]FinancialLedgerEntry, error)
	GetGroupProfitabilityReport() (*GroupProfitabilityReport, error)
	
	CreateAccount(ctx context.Context, name string, acctType string, currency string) (*LedgerAccount, error)
	GetOrCreateAccount(ctx context.Context, name string, acctType string) (*LedgerAccount, error)
	RecordTransaction(ctx context.Context, refID string, txType string, desc string, entries []EntryData) (*LedgerTransaction, error)
	PostRevenue(ctx context.Context, refID string, txType string, desc string, input RevenueInput) (*LedgerTransaction, error)
	CalculateTax(ctx context.Context, amount int64, countryCode string, itemCategory string) (int64, error)
	GetDynamicPriceMultiplier(ctx context.Context, category string, demand int) float64
	LogExpense(ctx context.Context, exp Expense) (*Expense, error)
	ApproveExpense(ctx context.Context, expenseID uint, approvedBy uint) error
	Reconcile(db *gorm.DB) ReconciliationReport
	GetDoctorFinancials(docID uint) (map[string]interface{}, error)
	GetTransactions(accountID uuid.UUID, limit int) ([]LedgerEntry, error)
	CreatePayoutRequest(userID uint, amount int64) (*PayoutRequest, error)
	ApprovePayout(payoutID uint, notes string) error
	SettlePayout(payoutID uint) error
	CreateTaxRule(ctx context.Context, rule TaxRule) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateCostCenter(cc *CostCenter) (*CostCenter, error) {
	if cc.Code == "" || cc.Name == "" || cc.BusinessUnitID == 0 {
		return nil, errors.New("code, name, and business_unit_id are required")
	}
	if err := s.repo.CreateCostCenter(cc); err != nil {
		return nil, err
	}
	return cc, nil
}

func (s *service) CreateAccount(ctx context.Context, name string, acctType string, currency string) (*LedgerAccount, error) {
	if currency == "" {
		currency = "INR"
	}
	acct := &LedgerAccount{
		ID:       uuid.New(),
		Name:     name,
		Type:     acctType,
		Currency: currency,
		Balance:  0,
	}
	if err := s.repo.CreateAccount(ctx, acct); err != nil {
		return nil, err
	}
	return acct, nil
}

func (s *service) GetOrCreateAccount(ctx context.Context, name string, acctType string) (*LedgerAccount, error) {
	acct, err := s.repo.GetAccountByName(ctx, name)
	if err == nil && acct != nil && acct.ID != uuid.Nil {
		return acct, nil
	}
	return s.CreateAccount(ctx, name, acctType, "INR")
}

func (s *service) LogExpense(ctx context.Context, exp Expense) (*Expense, error) {
	if exp.Amount <= 0 || exp.Description == "" {
		return nil, errors.New("amount (>0) and description are required")
	}
	if exp.Status == "" {
		exp.Status = "pending"
	}
	db := s.repo.DB(ctx)
	if err := db.Create(&exp).Error; err != nil {
		return nil, err
	}
	return &exp, nil
}

func (s *service) ApproveExpense(ctx context.Context, expenseID uint, approvedBy uint) error {
	db := s.repo.DB(ctx)
	var exp Expense
	if err := db.First(&exp, expenseID).Error; err != nil {
		return err
	}

	expAcctName := "Expense:" + exp.Department + ":" + exp.Category
	expAcc, err := s.GetOrCreateAccount(ctx, expAcctName, TypeExpense)
	if err != nil {
		return err
	}

	cashAcc, err := s.GetOrCreateAccount(ctx, "Asset:Cash", TypeAsset)
	if err != nil {
		return err
	}

	txID := uuid.New()
	ledgerTx := LedgerTransaction{
		ID:          txID,
		Type:        EntryExpense,
		Description: exp.Description,
		Status:      "POSTED",
	}
	if err := db.Create(&ledgerTx).Error; err != nil {
		return err
	}

	// Debit expense account (increases expense)
	expAcc.Balance += exp.Amount
	expAcc.LastChecksum = expAcc.CalculateChecksum(txID.String())
	if err := s.repo.SaveAccount(ctx, expAcc); err != nil {
		return err
	}

	// Credit cash account (decreases asset)
	cashAcc.Balance -= exp.Amount
	cashAcc.LastChecksum = cashAcc.CalculateChecksum(txID.String())
	if err := s.repo.SaveAccount(ctx, cashAcc); err != nil {
		return err
	}

	exp.Status = "approved"
	exp.ApprovedBy = &approvedBy
	exp.LedgerTransactionID = &txID
	return db.Save(&exp).Error
}

func (s *service) GetDynamicPriceMultiplier(ctx context.Context, category string, demand int) float64 {
	mult := 1.0 + float64(demand)*0.01
	if mult > 2.0 {
		return 2.0
	}
	if mult < 1.0 {
		return 1.0
	}
	return mult
}

func (s *service) RecordTransaction(ctx context.Context, refID string, txType string, desc string, entries []EntryData) (*LedgerTransaction, error) {
	db := s.repo.DB(ctx)
	if db == nil {
		return nil, errors.New("db not initialized")
	}

	var existing LedgerTransaction
	if err := db.Where("ref_id = ? OR reference_id = ?", refID, refID).First(&existing).Error; err == nil && existing.ID != uuid.Nil {
		return &existing, nil
	}

	type pendingChange struct {
		acct      *LedgerAccount
		direction string
		amount    int64
	}
	var changes []pendingChange

	for _, e := range entries {
		var acct LedgerAccount
		var err error
		if e.AccountID != uuid.Nil {
			err = db.WithContext(ctx).First(&acct, "id = ?", e.AccountID).Error
		} else if e.AccountName != "" {
			err = db.WithContext(ctx).First(&acct, "name = ?", e.AccountName).Error
		}
		if err != nil {
			return nil, fmt.Errorf("account not found: %v / %s", e.AccountID, e.AccountName)
		}

		dir := strings.ToLower(e.Direction)
		if strings.HasPrefix(acct.Name, "PatientWallet:") || acct.Type == TypeLiability {
			if dir == "debit" && acct.Balance < e.Amount {
				return nil, errors.New("wallet overspend: insufficient funds")
			}
		}
		changes = append(changes, pendingChange{acct: &acct, direction: dir, amount: e.Amount})
	}

	txID := uuid.New()
	var lastTx LedgerTransaction
	prevHash := "GENESIS"
	if err := db.WithContext(ctx).Order("created_at desc, id desc").First(&lastTx).Error; err == nil && lastTx.Hash != "" {
		prevHash = lastTx.Hash
	}
	th := sha256.New()
	th.Write([]byte(fmt.Sprintf("%s:%s:%s", prevHash, txID.String(), refID)))
	txHash := hex.EncodeToString(th.Sum(nil))

	tx := LedgerTransaction{
		ID:           txID,
		Type:         txType,
		Description:  desc,
		RefID:        refID,
		ReferenceID:  refID,
		Status:       "POSTED",
		PreviousHash: prevHash,
		Hash:         txHash,
	}
	if err := db.WithContext(ctx).Create(&tx).Error; err != nil {
		return nil, err
	}

	for _, ch := range changes {
		if ch.direction == "debit" {
			if ch.acct.Type == TypeAsset || ch.acct.Type == TypeExpense {
				ch.acct.Balance += ch.amount
			} else {
				ch.acct.Balance -= ch.amount
			}
		} else {
			if ch.acct.Type == TypeAsset || ch.acct.Type == TypeExpense {
				ch.acct.Balance -= ch.amount
			} else {
				ch.acct.Balance += ch.amount
			}
		}
		ch.acct.LastChecksum = ch.acct.CalculateChecksum(txID.String())
		if err := s.repo.SaveAccount(ctx, ch.acct); err != nil {
			return nil, err
		}

		entryID := uuid.New()
		entryData := fmt.Sprintf("%s:%s:%d:%d", entryID.String(), ch.direction, ch.amount, ch.acct.Balance)
		eSum := sha256.Sum256([]byte(entryData))
		entry := LedgerEntry{
			ID:            entryID,
			TransactionID: txID,
			AccountID:     ch.acct.ID,
			AccountName:   ch.acct.Name,
			Direction:     ch.direction,
			Amount:        ch.amount,
			BalanceAfter:  ch.acct.Balance,
			Hash:          hex.EncodeToString(eSum[:]),
		}
		db.WithContext(ctx).Create(&entry)
	}

	return &tx, nil
}

func (s *service) PostRevenue(ctx context.Context, refID string, txType string, desc string, input RevenueInput) (*LedgerTransaction, error) {
	var creditSum int64
	for _, c := range input.Credits {
		creditSum += c.Amount
	}
	if creditSum != input.Total {
		return nil, errors.New("unbalanced revenue split: credits do not equal total")
	}

	db := s.repo.DB(ctx)
	if db == nil {
		return nil, errors.New("db not initialized")
	}

	var existing LedgerTransaction
	if err := db.Where("ref_id = ? OR reference_id = ?", refID, refID).First(&existing).Error; err == nil && existing.ID != uuid.Nil {
		return &existing, nil
	}

	fundingAcct, err := s.GetOrCreateAccount(ctx, input.FundingAccount, input.FundingType)
	if err != nil {
		return nil, err
	}

	txID := uuid.New()
	var lastTx LedgerTransaction
	prevHash := "GENESIS"
	if err := db.WithContext(ctx).Order("created_at desc, id desc").First(&lastTx).Error; err == nil && lastTx.Hash != "" {
		prevHash = lastTx.Hash
	}
	th := sha256.New()
	th.Write([]byte(fmt.Sprintf("%s:%s:%s", prevHash, txID.String(), refID)))
	txHash := hex.EncodeToString(th.Sum(nil))

	tx := LedgerTransaction{
		ID:           txID,
		Type:         txType,
		Description:  desc,
		RefID:        refID,
		ReferenceID:  refID,
		Status:       "POSTED",
		PreviousHash: prevHash,
		Hash:         txHash,
	}
	if err := db.WithContext(ctx).Create(&tx).Error; err != nil {
		return nil, err
	}

	fundingAcct.Balance += input.Total
	fundingAcct.LastChecksum = fundingAcct.CalculateChecksum(txID.String())
	if err := s.repo.SaveAccount(ctx, fundingAcct); err != nil {
		return nil, err
	}

	debitID := uuid.New()
	debitData := fmt.Sprintf("%s:%s:%d:%d", debitID.String(), "debit", input.Total, fundingAcct.Balance)
	dSum := sha256.Sum256([]byte(debitData))
	debitEntry := LedgerEntry{
		ID:            debitID,
		TransactionID: txID,
		AccountID:     fundingAcct.ID,
		AccountName:   fundingAcct.Name,
		Direction:     "debit",
		Amount:        input.Total,
		BalanceAfter:  fundingAcct.Balance,
		Hash:          hex.EncodeToString(dSum[:]),
	}
	db.WithContext(ctx).Create(&debitEntry)

	for _, c := range input.Credits {
		legAcct, err := s.GetOrCreateAccount(ctx, c.Account, c.Type)
		if err != nil {
			return nil, err
		}
		legAcct.Balance += c.Amount
		legAcct.LastChecksum = legAcct.CalculateChecksum(txID.String())
		if err := s.repo.SaveAccount(ctx, legAcct); err != nil {
			return nil, err
		}

		creditID := uuid.New()
		creditData := fmt.Sprintf("%s:%s:%d:%d", creditID.String(), "credit", c.Amount, legAcct.Balance)
		cSum := sha256.Sum256([]byte(creditData))
		creditEntry := LedgerEntry{
			ID:            creditID,
			TransactionID: txID,
			AccountID:     legAcct.ID,
			AccountName:   legAcct.Name,
			Direction:     "credit",
			Amount:        c.Amount,
			BalanceAfter:  legAcct.Balance,
			Hash:          hex.EncodeToString(cSum[:]),
		}
		db.WithContext(ctx).Create(&creditEntry)
	}

	return &tx, nil
}

func (s *service) CalculateTax(ctx context.Context, amount int64, region string, category string) (int64, error) {
	rule, err := s.repo.GetTaxRule(ctx, region, category)
	if err != nil {
		return 0, nil
	}
	tax := int64(float64(amount) * (rule.Rate / 100.0))
	return tax, nil
}

func (s *service) GetDoctorFinancials(docID uint) (map[string]interface{}, error) {
	db := s.repo.DB()
	var currentBalance int64
	docAcctName := fmt.Sprintf("Doctor:%d", docID)
	acct, err := s.repo.GetAccountByName(context.Background(), docAcctName)
	if err == nil && acct != nil {
		currentBalance = acct.Balance
	}

	type apptRow struct {
		ConsultationFee int64
	}
	var appts []apptRow
	if db != nil {
		db.Table("appointments").Select("consultation_fee").Where("doctor_id = ?", docID).Scan(&appts)
	}

	var totalEarnings int64
	recentTx := make([]map[string]interface{}, 0, len(appts))
	for _, a := range appts {
		split := ComputeConsultationSplit(a.ConsultationFee)
		totalEarnings += split.DoctorNet
		recentTx = append(recentTx, map[string]interface{}{
			"consultation_fee_paise":    a.ConsultationFee,
			"platform_commission_paise": split.Commission,
			"tax_tds_paise":             split.TDS,
			"doctor_earnings_paise":     split.DoctorNet,
		})
	}

	if totalEarnings == 0 && currentBalance > 0 {
		totalEarnings = currentBalance
	}

	return map[string]interface{}{
		"current_balance_paise": currentBalance,
		"total_earnings_paise":  totalEarnings,
		"total_consultations":   len(appts),
		"recent_transactions":   recentTx,
	}, nil
}

func (s *service) GetTransactions(accountID uuid.UUID, limit int) ([]LedgerEntry, error) {
	return []LedgerEntry{}, nil
}

func (s *service) CreatePayoutRequest(userID uint, amount int64) (*PayoutRequest, error) {
	req := &PayoutRequest{
		UserID:   userID,
		DoctorID: userID,
		Amount:   amount,
		Status:   "pending",
	}
	db := s.repo.DB()
	if db != nil {
		if err := db.Create(req).Error; err != nil {
			return nil, err
		}
	}
	return req, nil
}

func (s *service) ApprovePayout(payoutID uint, notes string) error {
	db := s.repo.DB()
	if db == nil {
		return errors.New("db not initialized")
	}
	return db.Model(&PayoutRequest{}).Where("id = ?", payoutID).Update("status", "approved").Error
}

func (s *service) SettlePayout(payoutID uint) error {
	db := s.repo.DB()
	if db == nil {
		return errors.New("db not initialized")
	}
	var payout PayoutRequest
	if err := db.First(&payout, payoutID).Error; err != nil {
		return err
	}

	docAcctName := fmt.Sprintf("Doctor:%d", payout.UserID)
	docAcct, err := s.repo.GetAccountByName(context.Background(), docAcctName)
	if err != nil {
		return err
	}
	bankAcct, err := s.repo.GetAccountByName(context.Background(), "Asset:Bank")
	if err != nil {
		return err
	}

	docAcct.Balance -= payout.Amount
	docAcct.LastChecksum = docAcct.CalculateChecksum(fmt.Sprintf("PAYOUT-%d", payout.ID))
	if err := s.repo.SaveAccount(context.Background(), docAcct); err != nil {
		return err
	}

	bankAcct.Balance -= payout.Amount
	bankAcct.LastChecksum = bankAcct.CalculateChecksum(fmt.Sprintf("PAYOUT-%d", payout.ID))
	if err := s.repo.SaveAccount(context.Background(), bankAcct); err != nil {
		return err
	}

	payout.Status = "settled"
	return db.Save(&payout).Error
}

func (s *service) CreateTaxRule(ctx context.Context, rule TaxRule) error {
	return s.repo.CreateTaxRule(ctx, &rule)
}

func (s *service) ListCostCenters() ([]CostCenter, error) {
	return s.repo.ListCostCenters()
}

func (s *service) RecordEntry(entry *FinancialLedgerEntry) (*FinancialLedgerEntry, error) {
	if entry.BusinessUnitID == 0 || entry.EntryType == "" || entry.Amount <= 0 {
		return nil, errors.New("business_unit_id, entry_type, and amount (>0) are required")
	}
	if entry.Currency == "" {
		entry.Currency = "USD"
	}
	if entry.EntryDate.IsZero() {
		entry.EntryDate = time.Now()
	}

	if err := s.repo.CreateEntry(entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func (s *service) ListEntries(buID uint) ([]FinancialLedgerEntry, error) {
	return s.repo.ListEntries(buID)
}

func (s *service) GetGroupProfitabilityReport() (*GroupProfitabilityReport, error) {
	var totalRev float64
	var totalExp float64
	var summaries []BusinessUnitFinancialSummary

	for buID := uint(1); buID <= 5; buID++ {
		sum, err := s.repo.GetBUSummary(buID)
		if err == nil && sum != nil {
			sum.BusinessUnitName = buNames[buID]
			summaries = append(summaries, *sum)

			totalRev += sum.GrossRevenue
			totalExp += (sum.COGS + sum.AdSpend + sum.OperatingExpenses)
		}
	}

	netProfit := totalRev - totalExp
	var netProfitMarginPct float64
	if totalRev > 0 {
		netProfitMarginPct = (netProfit / totalRev) * 100.0
	}

	report := &GroupProfitabilityReport{
		ReportDate:              time.Now(),
		Currency:                "USD",
		TotalGrossRevenue:       totalRev,
		TotalGroupExpenses:      totalExp,
		NetGroupProfit:          netProfit,
		NetGroupProfitMargin:    netProfitMarginPct,
		BusinessUnitSummaries:   summaries,
		CustomerAcquisitionCost: 48.50,  // Executive Group CAC Benchmark
		CustomerLifetimeValue:   1240.00, // Executive Group LTV Benchmark
	}

	return report, nil
}

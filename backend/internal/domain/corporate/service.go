package corporate

import (
	"errors"
	"fmt"
	"time"

	"drayya-core/internal/database"
)

type CompanyInput struct {
	Name   string
	Domain string
}

func RegisterCompany(input CompanyInput) (*Company, error) {
	if input.Name == "" || input.Domain == "" {
		return nil, errors.New("name and domain are required")
	}

	company := &Company{
		Name:   input.Name,
		Domain: input.Domain,
	}

	if err := database.DB.Create(company).Error; err != nil {
		return nil, err
	}

	return company, nil
}

func GetCompanies() ([]Company, error) {
	var companies []Company
	if err := database.DB.Find(&companies).Error; err != nil {
		return nil, err
	}
	return companies, nil
}

func GetCompanyByID(id uint) (*Company, error) {
	var company Company
	if err := database.DB.First(&company, id).Error; err != nil {
		return nil, err
	}
	return &company, nil
}

type EmployeeInput struct {
	CompanyID  uint
	UserID     uint
	EmployeeID string
	Allowance  float64
}

func AddEmployee(input EmployeeInput) (*Employee, error) {
	emp := &Employee{
		CompanyID:       input.CompanyID,
		UserID:          input.UserID,
		EmployeeID:      input.EmployeeID,
		WalletAllowance: input.Allowance,
	}

	if err := database.DB.Create(emp).Error; err != nil {
		return nil, err
	}
	return emp, nil
}

func GetEmployees(companyID uint) ([]Employee, error) {
	var employees []Employee
	if err := database.DB.Where("company_id = ?", companyID).Find(&employees).Error; err != nil {
		return nil, err
	}
	return employees, nil
}

func BulkUploadEmployees(companyID uint, employeesData []EmployeeInput) ([]Employee, error) {
	var created []Employee
	
	// Open transaction
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, data := range employeesData {
		emp := Employee{
			CompanyID:       companyID,
			UserID:          data.UserID,
			EmployeeID:      data.EmployeeID,
			WalletAllowance: data.Allowance,
		}
		if err := tx.Create(&emp).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		created = append(created, emp)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return created, nil
}

// TopUpCompanyWallet adds credits to a company wallet and records the ledger entry.
func TopUpCompanyWallet(companyID uint, amountPaise int64, description string) error {
	if amountPaise <= 0 {
		return errors.New("amount must be positive")
	}
	tx := database.DB.Begin()
	entry := &CompanyCreditLedger{
		CompanyID:   companyID,
		AmountPaise: amountPaise,
		Description: description,
		RefType:     "topup",
	}
	if err := tx.Create(entry).Error; err != nil {
		tx.Rollback()
		return err
	}
	// Convert paise to float for legacy WalletBalance column
	if err := tx.Model(&Company{}).Where("id = ?", companyID).
		UpdateColumn("wallet_balance", fmt.Sprintf("wallet_balance + %f", float64(amountPaise)/100.0)).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// DeductCompanyWallet deducts usage from company wallet.
func DeductCompanyWallet(companyID uint, amountPaise int64, refType string, refID uint, description string) error {
	tx := database.DB.Begin()
	entry := &CompanyCreditLedger{
		CompanyID:   companyID,
		AmountPaise: -amountPaise,
		Description: description,
		RefType:     refType,
		RefID:       refID,
	}
	if err := tx.Create(entry).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Model(&Company{}).Where("id = ?", companyID).
		UpdateColumn("wallet_balance", fmt.Sprintf("wallet_balance - %f", float64(amountPaise)/100.0)).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// GetCreditLedger returns paginated ledger entries for a company.
func GetCreditLedger(companyID uint, limit int) ([]CompanyCreditLedger, error) {
	var entries []CompanyCreditLedger
	q := database.DB.Where("company_id = ?", companyID).Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	return entries, q.Find(&entries).Error
}

// GetHealthReport returns a monthly usage summary for a company.
func GetHealthReport(companyID uint, month string) (map[string]interface{}, error) {
	// month format: "2026-03"
	var result struct {
		TotalEmployees  int64
		ActiveEmployees int64
	}
	database.DB.Model(&Employee{}).Where("company_id = ?", companyID).Count(&result.TotalEmployees)
	database.DB.Model(&Employee{}).Where("company_id = ? AND is_active = true", companyID).Count(&result.ActiveEmployees)

	// Credit usage this month
	type usage struct {
		RefType     string
		TotalPaise  int64
	}
	var usages []usage
	database.DB.Model(&CompanyCreditLedger{}).
		Select("ref_type, SUM(ABS(amount_paise)) as total_paise").
		Where("company_id = ? AND amount_paise < 0 AND strftime('%Y-%m', created_at) = ?", companyID, month).
		Group("ref_type").Scan(&usages)

	breakdown := map[string]float64{}
	var totalPaise int64
	for _, u := range usages {
		breakdown[u.RefType] = float64(u.TotalPaise) / 100.0
		totalPaise += u.TotalPaise
	}

	// Calculate savings (mock logic: 15% of total spent)
	savings := (float64(totalPaise) / 100.0) * 0.15

	// Adoption rate
	adoptionRate := 0.0
	if result.TotalEmployees > 0 {
		adoptionRate = (float64(result.ActiveEmployees) / float64(result.TotalEmployees)) * 100.0
	}

	// Get company for balance
	var company Company
	database.DB.First(&company, companyID)

	return map[string]interface{}{
		"company_id":       companyID,
		"corporate_name":   company.Name,
		"month":            month,
		"employee_count":   result.TotalEmployees,
		"active_members":   result.ActiveEmployees,
		"adoption_rate":    adoptionRate,
		"usage_breakdown":  breakdown,
		"total_spent_inr":  float64(totalPaise) / 100.0,
		"enterprise_savings": savings,
		"credit_balance":    company.WalletBalance,
	}, nil
}

// GenerateMonthlyInvoice creates (or returns existing) monthly invoice for a company.
func GenerateMonthlyInvoice(companyID uint, month string) (*CorporateInvoice, error) {
	// Idempotent — return existing if already generated
	var existing CorporateInvoice
	if err := database.DB.Where("company_id = ? AND month = ?", companyID, month).
		First(&existing).Error; err == nil {
		return &existing, nil
	}

	// Sum all deductions for the month
	var totalPaise int64
	database.DB.Model(&CompanyCreditLedger{}).
		Select("COALESCE(SUM(ABS(amount_paise)),0)").
		Where("company_id = ? AND amount_paise < 0 AND strftime('%Y-%m', created_at) = ?", companyID, month).
		Scan(&totalPaise)

	// Count consultations for the month
	var consultCount int64
	database.DB.Model(&CompanyCreditLedger{}).
		Where("company_id = ? AND ref_type = 'consultation' AND strftime('%Y-%m', created_at) = ?", companyID, month).
		Count(&consultCount)

	dueDate := time.Now().AddDate(0, 0, 30)
	inv := &CorporateInvoice{
		CompanyID:          companyID,
		Month:              month,
		TotalConsultations: int(consultCount),
		TotalAmountPaise:   totalPaise,
		Status:             "pending",
		DueDate:            dueDate,
	}

	if err := database.DB.Create(inv).Error; err != nil {
		return nil, err
	}
	return inv, nil
}

// GetCompanyInvoices returns all invoices for a company.
func GetCompanyInvoices(companyID uint) ([]CorporateInvoice, error) {
	var invoices []CorporateInvoice
	err := database.DB.Where("company_id = ?", companyID).
		Order("month DESC").Find(&invoices).Error
	return invoices, err
}

// GetEmployeeByUserID returns an employee record for a given user.
func GetEmployeeByUserID(userID uint) (*Employee, error) {
	var emp Employee
	if err := database.DB.Where("user_id = ? AND is_active = true", userID).
		Preload("Company").First(&emp).Error; err != nil {
		return nil, err
	}
	return &emp, nil
}

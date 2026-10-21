package finance_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/billing"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/money"
)

func setupIntegrationDB(t *testing.T) *gorm.DB {
	// 1. Initialize SQLite in-memory DB with shared cache so connection pool shares the same DB
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// Set the global database instance so billing and finance services can find it
	database.DB = db

	// 2. AutoMigrate schemas
	err = db.AutoMigrate(
		&identity.User{},
		&identity.PatientProfile{},
		&identity.DoctorProfile{},
		&appointment.Appointment{},
		&billing.Billing{},
		&billing.Payment{},
		&finance.LedgerAccount{},
		&finance.LedgerTransaction{},
		&finance.LedgerEntry{},
		&finance.PayoutRequest{},
		&finance.TaxRule{},
	)
	require.NoError(t, err)

	// 3. Seed initial tax rules
	finance.SeedTaxRules(db)

	return db
}

func TestFinanceWorkflowE2E(t *testing.T) {
	db := setupIntegrationDB(t)

	// --- 1. Setup Data ---
	// Create Doctor User
	doctorUser := identity.User{
		Email:    "doctor@test.com",
		FullName: "Dr. Audit Test",
		Role:     "doctor",
	}
	db.Create(&doctorUser)

	doctorProfile := identity.DoctorProfile{
		UserID:           doctorUser.ID,
		ConsultationFees: money.RegionalPricing{"INR": 100000},
	}
	db.Create(&doctorProfile)

	// Create Patient User
	patientUser := identity.User{
		Email:    "patient@test.com",
		FullName: "Patient Test",
		Role:     "patient",
	}
	db.Create(&patientUser)

	patientProfile := identity.PatientProfile{
		UserID: patientUser.ID,
	}
	db.Create(&patientProfile)

	// Create Asset:Bank with initial balance of 1,000,000 paise (10,000 INR)
	bankAcctInit := finance.LedgerAccount{
		ID:       uuid.New(),
		Name:     "Asset:Bank",
		Type:     finance.TypeAsset,
		Currency: "INR",
		Balance:  1000000,
	}
	bankAcctInit.LastChecksum = bankAcctInit.CalculateChecksum("INIT")
	db.Create(&bankAcctInit)

	// Create Appointment
	doctorIDUint := doctorUser.ID
	appt := appointment.Appointment{
		PatientID:        patientUser.ID,
		DoctorID:         &doctorIDUint,
		StartTime:        time.Now().Add(1 * time.Hour),
		Status:           appointment.StatusConfirmed,
		AppointmentType:  "consultation",
		ConsultationFee:  100000,
	}
	db.Create(&appt)

	// Create Invoice
	invoice := billing.Billing{
		PatientID:     patientUser.ID,
		AppointmentID: &appt.ID,
		Amount:        100000,
		PaymentStatus: "pending",
	}
	db.Create(&invoice)

	// --- 2. Initialize Services ---
	finRepo := finance.NewRepository(db)
	finSvc := finance.NewService(finRepo)
	
	billRepo := billing.NewRepository(db)
	billSvc := billing.NewService(billRepo, finSvc)

	// --- 3. Action: Process Payment ---
	// Patient pays for the appointment. This triggers the double-entry accounting.
	_, err := billSvc.ProcessPaymentMock(invoice.ID, "upi")
	require.NoError(t, err, "Payment processing should succeed")

	// Verify Invoice Status
	db.First(&invoice, invoice.ID)
	assert.Equal(t, "paid", invoice.PaymentStatus)

	// --- 4. Verify: Ledger Entries ---
	// Validate Platform Fee (30% of 1000 = 300)
	var revAcct finance.LedgerAccount
	db.Where("name = ?", "Revenue:Consultations").First(&revAcct)
	assert.Equal(t, int64(30000), revAcct.Balance, "Platform fee should be 300 INR (30000 paise)")

	// Validate GST (18% of 300 = 54)
	var gstAcct finance.LedgerAccount
	db.Where("name = ?", "Liability:Tax:GST").First(&gstAcct)
	assert.Equal(t, int64(5400), gstAcct.Balance, "GST should be 54 INR (5400 paise)")

	// Validate TDS (10% of Doctor Gross)
	// Doc Gross = 1000 - 300 (comm) - 54 (GST) = 646
	// TDS = 10% of 646 = 64.6 -> rounded to 6460 paise
	var tdsAcct finance.LedgerAccount
	db.Where("name = ?", "Liability:Tax:TDS").First(&tdsAcct)
	assert.Equal(t, int64(6460), tdsAcct.Balance, "TDS should be 64.6 INR (6460 paise)")

	// Validate Doctor Payout (Net = 1000 - 300 - 54 - 64.6 = 581.4)
	var docAcct finance.LedgerAccount
	db.Where("name = ?", fmt.Sprintf("Doctor:%d", doctorUser.ID)).First(&docAcct)
	assert.Equal(t, int64(58140), docAcct.Balance, "Doctor net earnings should be 581.4 INR (58140 paise)")

	// Validate Total Asset (1000)
	var assetAcct finance.LedgerAccount
	db.Where("name = ?", "Asset:Razorpay").First(&assetAcct)
	assert.Equal(t, int64(100000), assetAcct.Balance, "Asset Razorpay should be 1000 INR (100000 paise)")

	// --- 5. Verify: Doctor Dashboard ---
	// The dashboard should fetch real stats instead of estimating
	docStats, err := finSvc.GetDoctorFinancials(doctorUser.ID)
	require.NoError(t, err)
	
	// Assert on paise fields directly
	assert.Equal(t, int64(58140), docStats["current_balance_paise"])
	assert.Equal(t, int64(58140), docStats["total_earnings_paise"])
	assert.Equal(t, 1, docStats["total_consultations"])

	transactions := docStats["recent_transactions"].([]map[string]interface{})
	assert.Len(t, transactions, 1)
	assert.Equal(t, int64(100000), transactions[0]["consultation_fee_paise"])
	assert.Equal(t, int64(30000), transactions[0]["platform_commission_paise"])
	assert.Equal(t, int64(6460), transactions[0]["tax_tds_paise"])
	assert.Equal(t, int64(58140), transactions[0]["doctor_earnings_paise"])

	// --- 6. Verify: Admin Payouts ---
	// Admin decides to execute pending payouts
	payout, err := finSvc.CreatePayoutRequest(doctorUser.ID, docAcct.Balance)
	require.NoError(t, err)
	assert.Equal(t, "pending", payout.Status)
	assert.Equal(t, int64(58140), payout.Amount)

	// Admin approves the payout
	err = finSvc.ApprovePayout(payout.ID, "Approved by admin for test")
	require.NoError(t, err)

	// System settles the payout (Calls Bank API, then updates Ledger)
	err = finSvc.SettlePayout(payout.ID)
	require.NoError(t, err)

	// Assert doctor balance is now zero
	db.Where("name = ?", fmt.Sprintf("Doctor:%d", doctorUser.ID)).First(&docAcct)
	assert.Equal(t, int64(0), docAcct.Balance, "Doctor balance should be 0 after settlement")

	// Assert Bank asset is credited (liability to bank paid out)
	var bankAcct finance.LedgerAccount
	db.Where("name = ?", "Asset:Bank").First(&bankAcct)
	assert.Equal(t, int64(941860), bankAcct.Balance, "Bank asset should have 941860 paise representing cash balance after payout")
}

package finance

import "fmt"

// Canonical chart of accounts — the single naming authority for the ledger.
//
// Every subsystem MUST resolve accounts through these constants and builders
// instead of hand-writing strings. Historically the same revenue stream landed
// in different accounts depending on the payment path (e.g. pharmacy wallet vs
// Razorpay used "Revenue:PlatformRoyalty" vs "Revenue:Pharmacy" and even a
// wrong account TYPE), which made consolidated reporting and reconciliation
// impossible. Keep all account-name decisions here.

// Fixed (singleton) accounts.
const (
	// Assets — where real money sits / is owed to us.
	AcctRazorpay            = "Asset:Razorpay"            // funds received via Razorpay gateway (India)
	AcctStripe              = "Asset:Stripe"              // funds received via Stripe gateway (global)
	AcctBank                = "Asset:Bank"                // cash in the bank
	AcctCash                = "Asset:Cash"                // physical cash (COD collections)
	AcctCorporateReceivable = "Asset:CorporateReceivable" // sponsor-funded, not yet collected

	// Escrow — held against an in-flight obligation.
	AcctHeldFunds = "HeldFunds" // liability: money held until a service completes

	// Revenue — platform earnings (only the platform's own share).
	RevConsultations       = "Revenue:Consultations"
	RevPharmacy            = "Revenue:Pharmacy"
	RevLab                 = "Revenue:Lab"
	RevSubscriptions       = "Revenue:Subscriptions"
	RevStore               = "Revenue:Store"
	RevCentralWarehouse    = "Revenue:CentralWarehouseSales" // platform-owned pharmacy share
	RevCancellations       = "Revenue:Cancellations"

	// Tax liabilities — owed to the government.
	LiabGST = "Liability:Tax:GST"
	LiabTDS = "Liability:Tax:TDS"

	// Expense pools.
	ExpMarketingSubsidy = "Marketing:GrowthSubsidy"
	ExpMarketingRewards = "Marketing:Rewards" // referral / loyalty payouts
)

// Per-stream tax/commission rates (percent). Single place to tune; flagged
// configurable. GST is treated as inclusive of the patient-facing total
// (gst = total*rate/(100+rate)).
const (
	LabGSTPercent    = 5  // diagnostics GST
	LabCommissionPct = 20 // platform's marketplace cut of the net (after GST)
	StoreGSTPercent  = 18 // wellness retail GST
)

// Per-entity account-name builders. The numeric suffix is the entity's ID.

// PatientWallet is the liability the platform owes a patient (their wallet).
func PatientWallet(userID uint) string { return fmt.Sprintf("PatientWallet:%d", userID) }

// DoctorWallet is the liability the platform owes a doctor (accrued earnings).
func DoctorWallet(doctorID uint) string { return fmt.Sprintf("Doctor:%d", doctorID) }

// MerchantWallet is the liability the platform owes a (non-central) pharmacy for
// its share of sales — a payable the pharmacy later withdraws. NOT revenue.
func MerchantWallet(pharmacyID uint) string {
	return fmt.Sprintf("MerchantWallet:Pharmacy:%d", pharmacyID)
}

// ProviderWallet is the liability owed to an external service provider (e.g. a
// lab or wellness merchant) for their share of a sale.
func ProviderWallet(kind string, providerID uint) string {
	return fmt.Sprintf("ProviderWallet:%s:%d", kind, providerID)
}

// SupplierPayable is the liability owed to a goods supplier (procurement).
func SupplierPayable(supplierID uint) string {
	return fmt.Sprintf("Liability:AccountsPayable:Supplier:%d", supplierID)
}

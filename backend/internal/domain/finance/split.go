package finance

// Canonical consultation revenue split — the SINGLE source of truth.
//
// Every code path that divides a consultation fee (direct Razorpay invoices,
// wallet/escrow completion, doctor-earnings display) MUST go through
// ComputeConsultationSplit. Historically three different formulas coexisted
// (30%+GST+TDS, 60/30/10, and a flat 20% cut) and the books could not be
// reconciled. Keep all rate changes here.
const (
	// PlatformCommissionPercent is the platform's commission on the gross fee.
	PlatformCommissionPercent = 30
	// GSTOnCommissionPercent is the GST levied on the platform commission
	// (the platform's service is taxable; the consultation itself is exempt).
	GSTOnCommissionPercent = 18
	// TDSPercent is the tax withheld at source from the doctor's gross earnings.
	TDSPercent = 10
)

// ConsultationSplit is the fully-resolved breakdown of one consultation fee.
// All amounts are in paise. Invariant (exact, no rounding gap):
//
//	Commission + GST + TDS + DoctorNet == Total
type ConsultationSplit struct {
	Total       int64 // gross fee charged to the patient
	Commission  int64 // platform revenue (Revenue:Consultations)
	GST         int64 // GST liability on the commission (Liability:Tax:GST)
	DoctorGross int64 // Total - Commission - GST
	TDS         int64 // tax withheld from the doctor (Liability:Tax:TDS)
	DoctorNet   int64 // amount actually owed to the doctor (credited to Doctor:%d)
}

// ComputeConsultationSplit breaks a gross consultation fee into its parts using
// integer (paise) arithmetic, in the order the ledger expects: commission off
// the top, GST on that commission, then TDS on the remaining doctor gross.
//
// Because DoctorGross and DoctorNet are derived by subtraction (not an
// independent percentage), the invariant above always holds exactly.
func ComputeConsultationSplit(total int64) ConsultationSplit {
	commission := (total * PlatformCommissionPercent) / 100
	gst := (commission * GSTOnCommissionPercent) / 100
	docGross := total - commission - gst
	tds := (docGross * TDSPercent) / 100
	return ConsultationSplit{
		Total:       total,
		Commission:  commission,
		GST:         gst,
		DoctorGross: docGross,
		TDS:         tds,
		DoctorNet:   docGross - tds,
	}
}

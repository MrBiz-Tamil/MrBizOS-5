package hr

import (
	"testing"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"
	payoutdomain "drayya-core/internal/domain/payout"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupHRTestDB spins up an isolated in-memory SQLite DB with the tables the HR
// money paths touch, and points database.DB at it.
func setupHRTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
	}
	for name, fn := range map[string]func(*gorm.DB) error{
		"identity":    identity.AutoMigrate,
		"appointment": appointment.AutoMigrate,
		"hr":          AutoMigrate,
		"payout":      payoutdomain.AutoMigrate,
	} {
		if err := fn(db); err != nil {
			t.Fatalf("%s migrate: %v", name, err)
		}
	}
	database.DB = db
	return db
}

// seedCompletedAppt inserts a completed consultation with a fixed created_at so
// the date-range queries are deterministic.
func seedCompletedAppt(t *testing.T, db *gorm.DB, doctorID uint, feePaise int64, createdAt time.Time, status string) {
	t.Helper()
	d := doctorID
	appt := appointment.Appointment{
		DoctorID:        &d,
		ConsultationFee: feePaise,
		Status:          appointment.AppointmentStatus(status),
	}
	if err := db.Create(&appt).Error; err != nil {
		t.Fatalf("create appt: %v", err)
	}
	// autoCreateTime stamps now(); override to the fixed instant we want to test.
	if err := db.Model(&appointment.Appointment{}).Where("id = ?", appt.ID).
		Update("created_at", createdAt).Error; err != nil {
		t.Fatalf("set created_at: %v", err)
	}
}

func TestQuarterDateRange(t *testing.T) {
	cases := []struct {
		fy, q, from, to string
		wantErr         bool
	}{
		{"2026-27", "Q1", "2026-04-01", "2026-07-01", false},
		{"2026-27", "Q2", "2026-07-01", "2026-10-01", false},
		{"2026-27", "Q3", "2026-10-01", "2027-01-01", false},
		{"2026-27", "Q4", "2027-01-01", "2027-04-01", false},
		{"2026-27", "Q5", "", "", true},
		{"bad", "Q1", "", "", true},
	}
	for _, c := range cases {
		from, to, err := quarterDateRange(c.fy, c.q)
		if c.wantErr {
			if err == nil {
				t.Errorf("%s %s: expected error, got none", c.fy, c.q)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s %s: unexpected error %v", c.fy, c.q, err)
			continue
		}
		if from != c.from || to != c.to {
			t.Errorf("%s %s: got [%s,%s) want [%s,%s)", c.fy, c.q, from, to, c.from, c.to)
		}
	}
}

func TestIsFreelancer(t *testing.T) {
	if (StaffProfile{EmploymentType: EmploymentFreelancer}).IsFreelancer() != true {
		t.Error("freelancer profile should report IsFreelancer()=true")
	}
	if (StaffProfile{EmploymentType: EmploymentSalaried}).IsFreelancer() != false {
		t.Error("salaried profile should report IsFreelancer()=false")
	}
	if (StaffProfile{}).IsFreelancer() != false {
		t.Error("unset employment type should default to salaried (not freelancer)")
	}
}

func TestComputeTDSWithheld_MatchesSplitAndRespectsRange(t *testing.T) {
	db := setupHRTestDB(t)
	const fee int64 = 100000 // ₹1000
	inRange := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	outRange := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	seedCompletedAppt(t, db, 1, fee, inRange, "completed")
	seedCompletedAppt(t, db, 1, fee, inRange, "completed")
	seedCompletedAppt(t, db, 1, fee, outRange, "completed") // outside quarter — excluded
	seedCompletedAppt(t, db, 1, fee, inRange, "pending")    // not completed — excluded

	total, consults := computeTDSWithheld("2026-04-01", "2026-07-01")

	wantTDS := finance.ComputeConsultationSplit(fee).TDS * 2
	if total != wantTDS {
		t.Errorf("withheld TDS = %d, want %d (2 × split.TDS)", total, wantTDS)
	}
	if consults != 2 {
		t.Errorf("consult count = %d, want 2", consults)
	}
}

func TestBackfillEmploymentTypes_ClassifiesDoctorsAndDoesNotClobber(t *testing.T) {
	db := setupHRTestDB(t)

	// Two doctors + one nurse, all with salaried-default profiles.
	doc1 := identity.User{Email: "d1@x.com", Role: "doctor", FullName: "Doc One", PasswordHash: "x"}
	doc2 := identity.User{Email: "d2@x.com", Role: "doctor", FullName: "Doc Two", PasswordHash: "x"}
	nurse := identity.User{Email: "n@x.com", Role: "nurse", FullName: "Nurse", PasswordHash: "x"}
	db.Create(&doc1)
	db.Create(&doc2)
	db.Create(&nurse)
	for i, u := range []identity.User{doc1, doc2, nurse} {
		p := StaffProfile{UserID: u.ID, EmployeeID: "EMP-" + time.Now().Format("150405") + string(rune('A'+i)), EmploymentType: EmploymentSalaried}
		if err := db.Create(&p).Error; err != nil {
			t.Fatalf("create profile: %v", err)
		}
	}

	// First run: both doctors become freelancers, nurse stays salaried.
	BackfillEmploymentTypes(db)
	assertEmploymentType(t, db, doc1.ID, EmploymentFreelancer)
	assertEmploymentType(t, db, doc2.ID, EmploymentFreelancer)
	assertEmploymentType(t, db, nurse.ID, EmploymentSalaried)

	// An admin deliberately moves doc2 onto salary. doc1 remains freelancer so the
	// "already populated" gate trips and the second run must NOT re-flip doc2.
	db.Model(&StaffProfile{}).Where("user_id = ?", doc2.ID).Update("employment_type", EmploymentSalaried)
	BackfillEmploymentTypes(db)
	assertEmploymentType(t, db, doc2.ID, EmploymentSalaried) // not clobbered
	assertEmploymentType(t, db, doc1.ID, EmploymentFreelancer)
}

func assertEmploymentType(t *testing.T, db *gorm.DB, userID uint, want string) {
	t.Helper()
	var p StaffProfile
	if err := db.Where("user_id = ?", userID).First(&p).Error; err != nil {
		t.Fatalf("load profile for user %d: %v", userID, err)
	}
	if p.EmploymentType != want {
		t.Errorf("user %d employment_type = %q, want %q", userID, p.EmploymentType, want)
	}
}

func TestCalculateMonthlyPayroll_RejectsFreelancer(t *testing.T) {
	db := setupHRTestDB(t)
	user := identity.User{Email: "f@x.com", Role: "doctor", FullName: "Freelancer", PasswordHash: "x"}
	db.Create(&user)
	db.Create(&StaffProfile{UserID: user.ID, EmployeeID: "EMP-F1", EmploymentType: EmploymentFreelancer})

	svc := NewService(NewRepository(db))
	if _, err := svc.CalculateMonthlyPayroll(user.ID, 5, 2026); err == nil {
		t.Fatal("expected payroll to reject a freelancer, got nil error")
	}
}

func TestCalculateMonthlyPayroll_SalariedProducesStatutoryDeductions(t *testing.T) {
	db := setupHRTestDB(t)
	user := identity.User{Email: "s@x.com", Role: "nurse", FullName: "Salaried Staff", PasswordHash: "x"}
	db.Create(&user)
	// ₹20,000/month gross — below ESI threshold, so PF + ESI + PT all apply.
	db.Create(&StaffProfile{UserID: user.ID, EmployeeID: "EMP-S1", EmploymentType: EmploymentSalaried, BaseSalary: 2000000, Designation: "Nurse"})

	svc := NewService(NewRepository(db))
	rec, err := svc.CalculateMonthlyPayroll(user.ID, 5, 2026)
	if err != nil {
		t.Fatalf("payroll failed: %v", err)
	}
	if rec.Deductions <= 0 {
		t.Errorf("expected statutory deductions > 0, got %d", rec.Deductions)
	}
	if rec.NetAmount >= rec.BaseAmount {
		t.Errorf("net (%d) should be below gross (%d) after deductions", rec.NetAmount, rec.BaseAmount)
	}
	// A payslip and compliance record must exist for the month.
	if _, err := GetPayslip(user.ID, "2026-05"); err != nil {
		t.Errorf("expected a payslip for 2026-05: %v", err)
	}
	// Re-running must be idempotent (no duplicate PayrollRecord rows).
	if _, err := svc.CalculateMonthlyPayroll(user.ID, 5, 2026); err != nil {
		t.Fatalf("re-run failed: %v", err)
	}
	var count int64
	db.Model(&PayrollRecord{}).Where("user_id = ? AND year = ? AND month = ?", user.ID, 2026, 5).Count(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 payroll record after re-run, got %d", count)
	}
}

func TestCalculateMonthlyPayroll_SkipsTerminated(t *testing.T) {
	db := setupHRTestDB(t)
	user := identity.User{Email: "t@x.com", Role: "nurse", FullName: "Ex Staff", PasswordHash: "x"}
	db.Create(&user)
	lwd := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC) // left before the pay month
	db.Create(&StaffProfile{
		UserID: user.ID, EmployeeID: "EMP-T1", EmploymentType: EmploymentSalaried,
		BaseSalary: 2000000, EmploymentStatus: "terminated", LastWorkingDay: &lwd,
	})

	svc := NewService(NewRepository(db))
	if _, err := svc.CalculateMonthlyPayroll(user.ID, 5, 2026); err == nil {
		t.Fatal("expected payroll to skip a terminated employee for a later month")
	}
}

func TestComputeFinalSettlement_Salaried(t *testing.T) {
	db := setupHRTestDB(t)
	user := identity.User{Email: "o@x.com", Role: "nurse", FullName: "Outgoing", PasswordHash: "x"}
	db.Create(&user)
	db.Create(&StaffProfile{UserID: user.ID, EmployeeID: "EMP-O1", EmploymentType: EmploymentSalaried, BaseSalary: 3000000})

	lwd := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC) // 15 of 31 days worked
	s, err := ComputeFinalSettlement(user.ID, lwd)
	if err != nil {
		t.Fatalf("compute failed: %v", err)
	}
	want := int64(3000000) * 15 / 31
	if s.ProratedSalaryPaise != want {
		t.Errorf("prorated salary = %d, want %d", s.ProratedSalaryPaise, want)
	}
	// No joining date on this profile → no gratuity.
	if s.GratuityEligible || s.GratuityPaise != 0 {
		t.Errorf("expected no gratuity without a joining date, got %d (eligible=%v)", s.GratuityPaise, s.GratuityEligible)
	}
	if s.NetPayablePaise != s.ProratedSalaryPaise+s.LeaveEncashmentPaise+s.GratuityPaise {
		t.Errorf("net %d should equal prorated+encashment+gratuity", s.NetPayablePaise)
	}
}

func TestClearanceComplete(t *testing.T) {
	s := FinalSettlement{}
	if s.ClearanceComplete() {
		t.Error("empty clearance should not be complete")
	}
	s.ITCleared, s.AssetsReturned, s.FinanceCleared = true, true, true
	if s.ClearanceComplete() {
		t.Error("clearance with handover pending should not be complete")
	}
	s.HandoverDone = true
	if !s.ClearanceComplete() {
		t.Error("all items signed off should be complete")
	}
}

func TestRupees(t *testing.T) {
	cases := map[int64]string{0: "₹0.00", 100: "₹1.00", 123456: "₹1234.56", -5000: "-₹50.00"}
	for paise, want := range cases {
		if got := rupees(paise); got != want {
			t.Errorf("rupees(%d) = %q, want %q", paise, got, want)
		}
	}
}

func TestComputeForm26Q_AggregatesAndFlagsMissingPAN(t *testing.T) {
	db := setupHRTestDB(t)
	doc := identity.User{Email: "q@x.com", Role: "doctor", FullName: "Q Doc", PasswordHash: "x"}
	db.Create(&doc)
	const fee int64 = 100000
	in := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	seedCompletedAppt(t, db, doc.ID, fee, in, "completed")
	seedCompletedAppt(t, db, doc.ID, fee, in, "completed")

	lines, gross, tds, missingPAN, _, _, err := computeForm26Q("2026-27", "Q1")
	if err != nil {
		t.Fatalf("computeForm26Q: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 deductee line, got %d", len(lines))
	}
	split := finance.ComputeConsultationSplit(fee)
	if tds != split.TDS*2 || gross != fee*2 {
		t.Errorf("totals off: gross=%d tds=%d", gross, tds)
	}
	if missingPAN != 1 {
		t.Errorf("expected 1 missing-PAN deductee, got %d", missingPAN)
	}
}

func TestComputeGratuity(t *testing.T) {
	const wage int64 = 5200000 // ₹52,000/month
	// 8 years of service → eligible, 8 completed years.
	joined := time.Date(2018, 4, 1, 0, 0, 0, 0, time.UTC)
	lwd := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	g, eligible, years := computeGratuity(wage, joined, lwd)
	if !eligible {
		t.Fatalf("8 years should be gratuity-eligible (years=%.2f)", years)
	}
	want := wage * 15 / 26 * 8
	if g != want {
		t.Errorf("gratuity = %d, want %d", g, want)
	}

	// Under 5 years → not eligible.
	if g2, e2, _ := computeGratuity(wage, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), lwd); e2 || g2 != 0 {
		t.Errorf("under 5 years should be ineligible, got %d (eligible=%v)", g2, e2)
	}
	// Unknown joining date → zero.
	if g3, e3, _ := computeGratuity(wage, time.Time{}, lwd); e3 || g3 != 0 {
		t.Errorf("unknown joining date should yield zero gratuity, got %d", g3)
	}
	// Statutory cap.
	bigJoin := time.Date(1985, 1, 1, 0, 0, 0, 0, time.UTC)
	if gCap, _, _ := computeGratuity(50000000, bigJoin, lwd); gCap != gratuityCapPaise {
		t.Errorf("gratuity should cap at %d, got %d", gratuityCapPaise, gCap)
	}
}

func TestComputeFinalSettlement_Freelancer(t *testing.T) {
	db := setupHRTestDB(t)
	doc := identity.User{Email: "fd@x.com", Role: "doctor", FullName: "Panel Doc", PasswordHash: "x"}
	db.Create(&doc)
	db.Create(&StaffProfile{UserID: doc.ID, EmployeeID: "EMP-FD", EmploymentType: EmploymentFreelancer})

	const fee int64 = 100000
	seedCompletedAppt(t, db, doc.ID, fee, time.Now(), "completed")
	seedCompletedAppt(t, db, doc.ID, fee, time.Now(), "completed")

	s, err := ComputeFinalSettlement(doc.ID, time.Now())
	if err != nil {
		t.Fatalf("compute failed: %v", err)
	}
	split := finance.ComputeConsultationSplit(fee)
	if s.PendingPayoutPaise != split.DoctorNet*2 {
		t.Errorf("pending payout = %d, want %d", s.PendingPayoutPaise, split.DoctorNet*2)
	}
	if s.PendingTDSPaise != split.TDS*2 {
		t.Errorf("pending TDS = %d, want %d", s.PendingTDSPaise, split.TDS*2)
	}
	if s.NetPayablePaise != s.PendingPayoutPaise {
		t.Errorf("freelancer net payable should equal pending payout")
	}
}

func TestSuggestPricingTier(t *testing.T) {
	cases := []struct {
		name           string
		rating         float64
		completion     float64
		consults       int
		want           string
	}{
		{"too new", 5.0, 1.0, 5, TierNew},
		{"low rating", 3.0, 0.95, 200, TierNeedsReview},
		{"premium", 4.8, 0.95, 150, TierPremium},
		{"premium with pct completion", 4.8, 95, 150, TierPremium}, // 0–100 normalised
		{"solid but not premium", 4.0, 0.8, 80, TierStandard},
		{"high rating low volume", 4.9, 1.0, 50, TierStandard}, // <100 consults
	}
	for _, c := range cases {
		if got := SuggestPricingTier(c.rating, c.completion, c.consults); got != c.want {
			t.Errorf("%s: SuggestPricingTier(%.1f,%.2f,%d) = %q, want %q", c.name, c.rating, c.completion, c.consults, got, c.want)
		}
	}
}

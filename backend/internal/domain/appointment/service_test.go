package appointment

import (
	"context"
	"errors"
	"testing"
	"time"

	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// Compile-time check that mockRepo satisfies Repository.
var _ Repository = (*mockRepo)(nil)

// ─── Mock Repository ──────────────────────────────────────────────────────────

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) CreateAppointment(appt *Appointment) error {
	return m.Called(appt).Error(0)
}
func (m *mockRepo) GetAppointmentByID(id string) (*Appointment, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Appointment), args.Error(1)
}
func (m *mockRepo) GetAppointmentByIDUint(id uint) (*Appointment, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Appointment), args.Error(1)
}
func (m *mockRepo) GetAppointmentsByPatientOrDoctor(userID uint, includePast bool, limit int) ([]Appointment, error) {
	args := m.Called(userID, includePast, limit)
	return args.Get(0).([]Appointment), args.Error(1)
}
func (m *mockRepo) GetPendingInstantAppointments() ([]Appointment, error) {
	args := m.Called()
	return args.Get(0).([]Appointment), args.Error(1)
}
func (m *mockRepo) GetUpcomingAppointmentsForDoctor(doctorID uint) ([]Appointment, error) {
	args := m.Called(doctorID)
	return args.Get(0).([]Appointment), args.Error(1)
}
func (m *mockRepo) GetAppointmentHistoryCombo(patientID, doctorID *uint) ([]Appointment, error) {
	args := m.Called(patientID, doctorID)
	return args.Get(0).([]Appointment), args.Error(1)
}
func (m *mockRepo) UpdateAppointment(id string, updates map[string]interface{}) (int64, error) {
	args := m.Called(id, updates)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockRepo) UpdateAppointmentStatus(id string, status string) (int64, error) {
	args := m.Called(id, status)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockRepo) GetInstantQueueForDoctor(doctorID uint) ([]Appointment, error) {
	args := m.Called(doctorID)
	return args.Get(0).([]Appointment), args.Error(1)
}
func (m *mockRepo) UpdateAppointmentAtomic(id string, fromStatus string, updates map[string]interface{}) (int64, error) {
	args := m.Called(id, fromStatus, updates)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockRepo) ExecuteTransaction(ctx context.Context, fn func(txCtx context.Context, txRepo Repository, tx *gorm.DB) error) error {
	return fn(ctx, m, nil)
}
func (m *mockRepo) GetPatientInfo(patientID uint) (*identity.User, error) {
	args := m.Called(patientID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.User), args.Error(1)
}
func (m *mockRepo) GetDoctorProfile(doctorID uint) (*DoctorProfile, error) {
	args := m.Called(doctorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DoctorProfile), args.Error(1)
}
func (m *mockRepo) IncrementPatientFreeConsult(patientID uint) error {
	return m.Called(patientID).Error(0)
}
func (m *mockRepo) IncrementDoctorFreeConsultGiven(doctorID uint) error {
	return m.Called(doctorID).Error(0)
}
func (m *mockRepo) IncrementDoctorConsultations(doctorUserID uint) error {
	args := m.Called(doctorUserID)
	return args.Error(0)
}
func (m *mockRepo) CheckDoctorAvailability(doctorUserID uint, scheduledAt time.Time, excludeID uint) (bool, error) {
	args := m.Called(doctorUserID, scheduledAt, excludeID)
	return args.Bool(0), args.Error(1)
}

// ─── Mock Finance Service ─────────────────────────────────────────────────────

type mockFinance struct {
	mock.Mock
}

func (m *mockFinance) CalculateChecksum(accountID uuid.UUID, balance int64, lastEntryID string) string {
	return ""
}
func (m *mockFinance) CreateAccount(ctx context.Context, name string, accType finance.AccountType, currency string) (*finance.LedgerAccount, error) {
	args := m.Called(ctx, name, accType, currency)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.LedgerAccount), args.Error(1)
}
func (m *mockFinance) RecordTransaction(ctx context.Context, referenceID, txnType, description string, entries []finance.EntryData) (*finance.LedgerTransaction, error) {
	args := m.Called(ctx, referenceID, txnType, description, entries)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.LedgerTransaction), args.Error(1)
}
func (m *mockFinance) PostRevenue(ctx context.Context, ref, txnType, description string, in finance.RevenueInput) (*finance.LedgerTransaction, error) {
	args := m.Called(ctx, ref, txnType, description, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.LedgerTransaction), args.Error(1)
}
func (m *mockFinance) Reconcile(db *gorm.DB) finance.ReconciliationReport {
	return finance.ReconciliationReport{}
}
func (m *mockFinance) GetTransactions(accountID uuid.UUID, limit int) ([]finance.LedgerEntry, error) {
	return nil, nil
}
func (m *mockFinance) GetTransactionDetails(id uuid.UUID) (*finance.LedgerTransaction, error) {
	return nil, nil
}
func (m *mockFinance) GetDoctorFinancials(doctorID interface{}) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockFinance) GetOrCreateAccount(ctx context.Context, name string, accType finance.AccountType) (*finance.LedgerAccount, error) {
	args := m.Called(ctx, name, accType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*finance.LedgerAccount), args.Error(1)
}
func (m *mockFinance) CreatePayoutRequest(id uint, amount int64) (*finance.PayoutRequest, error) {
	return nil, nil
}
func (m *mockFinance) ApprovePayout(id uint, comment string) error                   { return nil }
func (m *mockFinance) SettlePayout(id uint) error                                    { return nil }
func (m *mockFinance) GetPendingPayouts() ([]finance.PayoutRequest, error)           { return nil, nil }
func (m *mockFinance) SettleTaxPmt(amount int64, reference string) error             { return nil }
func (m *mockFinance) GetFinancialMetrics() (map[string]interface{}, error)          { return nil, nil }
func (m *mockFinance) CreateTaxRule(ctx context.Context, rule finance.TaxRule) error { return nil }
func (m *mockFinance) CalculateTax(ctx context.Context, amount int64, region string, category string) (int64, error) {
	return 0, nil
}
func (m *mockFinance) GetDynamicPriceMultiplier(ctx context.Context, category string, demand int) float64 {
	return 1.0
}
func (m *mockFinance) LogExpense(ctx context.Context, e finance.Expense) (*finance.Expense, error) {
	return nil, nil
}
func (m *mockFinance) ApproveExpense(ctx context.Context, expenseID uint, approvedBy uint) error {
	return nil
}
func (m *mockFinance) RejectExpense(ctx context.Context, expenseID uint, reason string) error {
	return nil
}
func (m *mockFinance) GetExpenses(department string) ([]finance.Expense, error) { return nil, nil }
func (m *mockFinance) CalculateTransactionHash(txn *finance.LedgerTransaction, entries []finance.LedgerEntry) string {
	return ""
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newTestService(repo Repository, fin finance.Service) Service {
	return NewService(repo, fin)
}

func ptrUint(v uint) *uint { return &v }

// ─── StartAppointment Tests ───────────────────────────────────────────────────

func TestStartAppointment_ConfirmedSucceeds(t *testing.T) {
	repo := &mockRepo{}
	fin := &mockFinance{}
	svc := newTestService(repo, fin)

	// StartTime must be within the -10m / +30m window checked by the service.
	appt := &Appointment{ID: 1, PatientID: 10, Status: StatusConfirmed, Version: 1, StartTime: time.Now()}
	repo.On("GetAppointmentByID", "1").Return(appt, nil)
	repo.On("UpdateAppointmentAtomic", "1", string(StatusConfirmed), mock.Anything).Return(int64(1), nil)

	err := svc.StartAppointment("1")
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestStartAppointment_PendingBlocked(t *testing.T) {
	repo := &mockRepo{}
	fin := &mockFinance{}
	svc := newTestService(repo, fin)

	appt := &Appointment{ID: 1, PatientID: 10, Status: StatusPending}
	repo.On("GetAppointmentByID", "1").Return(appt, nil)

	err := svc.StartAppointment("1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot start appointment with status")
	repo.AssertNotCalled(t, "UpdateAppointmentStatus")
}

func TestStartAppointment_CompletedBlocked(t *testing.T) {
	repo := &mockRepo{}
	fin := &mockFinance{}
	svc := newTestService(repo, fin)

	appt := &Appointment{ID: 1, Status: StatusCompleted}
	repo.On("GetAppointmentByID", "1").Return(appt, nil)

	err := svc.StartAppointment("1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot start appointment with status")
}

func TestStartAppointment_NotFound(t *testing.T) {
	repo := &mockRepo{}
	fin := &mockFinance{}
	svc := newTestService(repo, fin)

	repo.On("GetAppointmentByID", "99").Return(nil, errors.New("not found"))

	err := svc.StartAppointment("99")
	assert.Error(t, err)
	assert.Equal(t, "appointment not found", err.Error())
}

// ─── CompleteAppointment Tests ────────────────────────────────────────────────

func TestCompleteAppointment_InConsultSucceeds(t *testing.T) {
	repo := &mockRepo{}
	fin := &mockFinance{}
	svc := newTestService(repo, fin)

	// nil DoctorID skips the global financials.ProcessConsultationPayout call
	appt := &Appointment{ID: 1, PatientID: 10, DoctorID: nil, Status: StatusInConsult, Version: 1}
	repo.On("GetAppointmentByID", "1").Return(appt, nil)
	repo.On("UpdateAppointmentAtomic", "1", string(StatusInConsult), mock.Anything).Return(int64(1), nil)

	err := svc.CompleteAppointment(context.Background(), "1")
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCompleteAppointment_InProgressSucceeds(t *testing.T) {
	repo := &mockRepo{}
	fin := &mockFinance{}
	svc := newTestService(repo, fin)

	// nil DoctorID skips the global financials.ProcessConsultationPayout call
	appt := &Appointment{ID: 1, PatientID: 10, DoctorID: nil, Status: StatusInConsult, Version: 1}
	repo.On("GetAppointmentByID", "1").Return(appt, nil)
	repo.On("UpdateAppointmentAtomic", "1", string(StatusInConsult), mock.Anything).Return(int64(1), nil)

	err := svc.CompleteAppointment(context.Background(), "1")
	assert.NoError(t, err)
}

func TestCompleteAppointment_ConfirmedBlocked(t *testing.T) {
	repo := &mockRepo{}
	fin := &mockFinance{}
	svc := newTestService(repo, fin)

	appt := &Appointment{ID: 1, Status: StatusConfirmed}
	repo.On("GetAppointmentByID", "1").Return(appt, nil)

	err := svc.CompleteAppointment(context.Background(), "1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot complete appointment with status")
	repo.AssertNotCalled(t, "UpdateAppointmentStatus")
}

func TestCompleteAppointment_PendingBlocked(t *testing.T) {
	repo := &mockRepo{}
	fin := &mockFinance{}
	svc := newTestService(repo, fin)

	appt := &Appointment{ID: 1, Status: StatusPending}
	repo.On("GetAppointmentByID", "1").Return(appt, nil)

	err := svc.CompleteAppointment(context.Background(), "1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot complete appointment with status")
}

// ─── MarkNoShow Tests ─────────────────────────────────────────────────────────

func TestMarkNoShow_PastConfirmedSucceeds(t *testing.T) {
	repo := &mockRepo{}
	fin := &mockFinance{}
	svc := newTestService(repo, fin)

	appt := &Appointment{ID: 1, Status: StatusConfirmed, StartTime: time.Now().Add(-1 * time.Hour), Version: 1}
	repo.On("GetAppointmentByID", "1").Return(appt, nil)
	repo.On("UpdateAppointmentAtomic", "1", string(StatusConfirmed), mock.Anything).Return(int64(1), nil)

	err := svc.MarkNoShow("1")
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestMarkNoShow_FutureAppointmentBlocked(t *testing.T) {
	repo := &mockRepo{}
	fin := &mockFinance{}
	svc := newTestService(repo, fin)

	appt := &Appointment{ID: 1, Status: StatusConfirmed, StartTime: time.Now().Add(2 * time.Hour)}
	repo.On("GetAppointmentByID", "1").Return(appt, nil)

	err := svc.MarkNoShow("1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has not started yet")
	repo.AssertNotCalled(t, "UpdateAppointmentStatus")
}

func TestMarkNoShow_CompletedBlocked(t *testing.T) {
	repo := &mockRepo{}
	fin := &mockFinance{}
	svc := newTestService(repo, fin)

	appt := &Appointment{ID: 1, Status: StatusCompleted, StartTime: time.Now().Add(-1 * time.Hour)}
	repo.On("GetAppointmentByID", "1").Return(appt, nil)

	err := svc.MarkNoShow("1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot mark no-show")
	repo.AssertNotCalled(t, "UpdateAppointmentStatus")
}

// ─── CancelAppointment Tests ──────────────────────────────────────────────────

func TestCancelAppointment_NoFeeNoRefund(t *testing.T) {
	repo := &mockRepo{}
	fin := &mockFinance{}
	svc := newTestService(repo, fin)

	appt := &Appointment{ID: 1, PatientID: 10, Status: StatusConfirmed, ConsultationFee: 0, Version: 1}
	repo.On("GetAppointmentByID", "1").Return(appt, nil)
	repo.On("UpdateAppointmentAtomic", "1", string(StatusConfirmed), mock.Anything).Return(int64(1), nil)

	err := svc.CancelAppointment("1", "test reason")
	assert.NoError(t, err)
	// Finance service should NOT be called when fee is zero
	fin.AssertNotCalled(t, "GetOrCreateAccount")
	repo.AssertExpectations(t)
}

func TestCancelAppointment_WithFeeTriggersRefund(t *testing.T) {
	repo := &mockRepo{}
	fin := &mockFinance{}
	svc := newTestService(repo, fin)

	patientID := uint(10)
	walletID := uuid.New()
	heldID := uuid.New()

	appt := &Appointment{ID: 1, PatientID: patientID, Status: StatusConfirmed, ConsultationFee: 50000, StartTime: time.Now().Add(48 * time.Hour), Version: 1}
	repo.On("GetAppointmentByID", "1").Return(appt, nil)
	repo.On("UpdateAppointmentAtomic", "1", string(StatusConfirmed), mock.Anything).Return(int64(1), nil)

	fin.On("GetOrCreateAccount", mock.Anything, "PatientWallet:10", finance.TypeLiability).
		Return(&finance.LedgerAccount{ID: walletID, Balance: 100000}, nil)
	fin.On("GetOrCreateAccount", mock.Anything, "HeldFunds", finance.TypeLiability).
		Return(&finance.LedgerAccount{ID: heldID}, nil)
	fin.On("RecordTransaction", mock.Anything, mock.Anything, "CONSULTATION_REFUND", mock.Anything, mock.Anything).
		Return(&finance.LedgerTransaction{}, nil)

	err := svc.CancelAppointment("1", "patient request")
	assert.NoError(t, err)
	fin.AssertCalled(t, "RecordTransaction", mock.Anything, mock.Anything, "CONSULTATION_REFUND", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestCancelAppointment_CompletedRejected(t *testing.T) {
	repo := &mockRepo{}
	fin := &mockFinance{}
	svc := newTestService(repo, fin)

	// The service guards against cancelling terminal states (completed/no_show/cancelled/declined).
	appt := &Appointment{ID: 1, PatientID: 10, Status: StatusCompleted, ConsultationFee: 50000, Version: 1}
	repo.On("GetAppointmentByID", "1").Return(appt, nil)

	err := svc.CancelAppointment("1", "after completion")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot cancel an appointment in status")
	fin.AssertNotCalled(t, "GetOrCreateAccount")
}

func TestCancelAppointment_NotFound(t *testing.T) {
	repo := &mockRepo{}
	fin := &mockFinance{}
	svc := newTestService(repo, fin)

	repo.On("GetAppointmentByID", "999").Return(nil, errors.New("not found"))

	err := svc.CancelAppointment("999", "reason")
	assert.Error(t, err)
	assert.Equal(t, "appointment not found", err.Error())
}

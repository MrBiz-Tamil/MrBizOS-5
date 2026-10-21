package appointment

// state_machine_test.go — table-driven integration tests for the appointment
// state machine and concurrent acceptance race condition.
//
// These tests use a real in-memory repository (tableRepo) rather than a mock
// stub, which means the atomic-update logic in the mock mirrors what a real DB
// optimistic-lock would do: return RowsAffected=0 when the status has changed.

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"drayya-core/internal/domain/identity"

	"gorm.io/gorm"
)

// ──────────────────────────────────────────────────────────────────────────────
// tableRepo — in-memory repository for table-driven and race tests
// ──────────────────────────────────────────────────────────────────────────────

type tableRepo struct {
	mu     sync.RWMutex
	appts  map[uint]*Appointment
	nextID uint
}

func newTableRepo() *tableRepo {
	return &tableRepo{
		appts:  make(map[uint]*Appointment),
		nextID: 1,
	}
}

func (r *tableRepo) CreateAppointment(appt *Appointment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	appt.ID = r.nextID
	r.nextID++
	if appt.Version == 0 {
		appt.Version = 1
	}
	cp := *appt
	r.appts[appt.ID] = &cp
	return nil
}

func (r *tableRepo) GetAppointmentByID(id string) (*Appointment, error) {
	uid, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %s", id)
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.appts[uint(uid)]
	if !ok {
		return nil, fmt.Errorf("appointment %s not found", id)
	}
	cp := *a
	return &cp, nil
}

func (r *tableRepo) GetAppointmentByIDUint(id uint) (*Appointment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.appts[id]
	if !ok {
		return nil, fmt.Errorf("appointment %d not found", id)
	}
	cp := *a
	return &cp, nil
}

// UpdateAppointmentAtomic simulates optimistic locking: only applies the update
// when the current status matches fromStatus (returns RowsAffected=0 otherwise).
func (r *tableRepo) UpdateAppointmentAtomic(id string, fromStatus string, updates map[string]interface{}) (int64, error) {
	uid, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid id: %s", id)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.appts[uint(uid)]
	if !ok {
		return 0, fmt.Errorf("appointment %s not found", id)
	}
	if string(a.Status) != fromStatus {
		return 0, nil // optimistic lock lost — caller gets RowsAffected=0
	}
	if s, ok2 := updates["status"].(string); ok2 {
		a.Status = AppointmentStatus(s)
	}
	if did, ok2 := updates["doctor_id"].(uint); ok2 {
		a.DoctorID = &did
	}
	a.Version++
	return 1, nil
}

func (r *tableRepo) UpdateAppointment(id string, updates map[string]interface{}) (int64, error) {
	uid, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid id: %s", id)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.appts[uint(uid)]
	if !ok {
		return 0, fmt.Errorf("appointment %s not found", id)
	}
	if s, ok2 := updates["status"].(string); ok2 {
		a.Status = AppointmentStatus(s)
	}
	return 1, nil
}

func (r *tableRepo) UpdateAppointmentStatus(id string, status string) (int64, error) {
	return r.UpdateAppointment(id, map[string]interface{}{"status": status})
}

// Stubs for Repository interface methods not under test in these scenarios.

func (r *tableRepo) GetAppointmentsByPatientOrDoctor(_ uint, _ bool, _ int) ([]Appointment, error) {
	return nil, nil
}
func (r *tableRepo) GetPendingInstantAppointments() ([]Appointment, error)    { return nil, nil }
func (r *tableRepo) GetInstantQueueForDoctor(_ uint) ([]Appointment, error)   { return nil, nil }
func (r *tableRepo) GetUpcomingAppointmentsForDoctor(_ uint) ([]Appointment, error) {
	return nil, nil
}
func (r *tableRepo) GetAppointmentHistoryCombo(_, _ *uint) ([]Appointment, error) {
	return nil, nil
}
func (r *tableRepo) ExecuteTransaction(ctx context.Context, fn func(context.Context, Repository, *gorm.DB) error) error {
	return fn(ctx, r, nil)
}
func (r *tableRepo) GetPatientInfo(_ uint) (*identity.User, error)         { return nil, nil }
func (r *tableRepo) GetDoctorProfile(_ uint) (*identity.DoctorProfile, error) { return nil, nil }
func (r *tableRepo) IncrementPatientFreeConsult(_ uint) error               { return nil }
func (r *tableRepo) IncrementDoctorFreeConsultGiven(_ uint) error           { return nil }
func (r *tableRepo) IncrementDoctorConsultations(_ uint) error              { return nil }
func (r *tableRepo) CheckDoctorAvailability(_ uint, _ time.Time, _ uint) (bool, error) {
	return true, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

func idStr(appt *Appointment) string {
	return strconv.FormatUint(uint64(appt.ID), 10)
}

func strContains(s, sub string) bool {
	return strings.Contains(s, sub)
}

// ──────────────────────────────────────────────────────────────────────────────
// TestAppointmentStatusTransitions — table-driven state machine tests
// ──────────────────────────────────────────────────────────────────────────────

func TestAppointmentStatusTransitions(t *testing.T) {
	type testCase struct {
		name            string
		initialStatus   AppointmentStatus
		action          string // "accept" | "start" | "complete" | "cancel"
		actorID         uint
		wantErr         bool
		wantStatus      AppointmentStatus
		wantErrContains string
	}

	tests := []testCase{
		// ── Happy path ────────────────────────────────────────────────────────
		{
			name:          "pending → accept → confirmed",
			initialStatus: StatusPending,
			action:        "accept",
			actorID:       10,
			wantErr:       false,
			wantStatus:    StatusConfirmed,
		},
		{
			name:          "confirmed → start → in_consultation",
			initialStatus: StatusConfirmed,
			action:        "start",
			actorID:       10,
			wantErr:       false,
			wantStatus:    StatusInConsult,
		},
		{
			name:          "in_consultation → complete",
			initialStatus: StatusInConsult,
			action:        "complete",
			actorID:       10,
			wantErr:       false,
			wantStatus:    StatusCompleted,
		},
		{
			name:          "pending → cancel",
			initialStatus: StatusPending,
			action:        "cancel",
			actorID:       10,
			wantErr:       false,
			wantStatus:    StatusCancelled,
		},
		{
			name:          "confirmed → cancel",
			initialStatus: StatusConfirmed,
			action:        "cancel",
			actorID:       10,
			wantErr:       false,
			wantStatus:    StatusCancelled,
		},
		// ── Invalid / blocked transitions ─────────────────────────────────────
		{
			name:            "completed → accept (invalid)",
			initialStatus:   StatusCompleted,
			action:          "accept",
			actorID:         10,
			wantErr:         true,
			wantErrContains: "cannot",
		},
		{
			name:          "cancelled → start (invalid)",
			initialStatus: StatusCancelled,
			action:        "start",
			actorID:       10,
			wantErr:       true,
		},
		{
			name:          "pending → complete (skip steps, invalid)",
			initialStatus: StatusPending,
			action:        "complete",
			actorID:       10,
			wantErr:       true,
		},
		{
			name:          "pending → start (must be confirmed first, invalid)",
			initialStatus: StatusPending,
			action:        "start",
			actorID:       10,
			wantErr:       true,
		},
		{
			name:          "completed → start (invalid)",
			initialStatus: StatusCompleted,
			action:        "start",
			actorID:       10,
			wantErr:       true,
		},
		{
			name:          "in_consultation → accept (invalid)",
			initialStatus: StatusInConsult,
			action:        "accept",
			actorID:       10,
			wantErr:       true,
		},
		{
			name:          "no_show → complete (invalid)",
			initialStatus: StatusNoShow,
			action:        "complete",
			actorID:       10,
			wantErr:       true,
		},
		{
			name:          "cancelled → accept (invalid)",
			initialStatus: StatusCancelled,
			action:        "accept",
			actorID:       10,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := newTableRepo()
			// Reuse mockFinance from service_test.go (same package).
			// No expectations needed — finance calls are no-ops for these paths.
			fin := &mockFinance{}
			svc := NewService(repo, fin)

			// Seed appointment in the requested initial status.
			appt := &Appointment{
				Status:    tt.initialStatus,
				PatientID: 1,
				Version:   1,
			}
			// Transitions from confirmed/in_consultation require a DoctorID to be set.
			if tt.initialStatus == StatusConfirmed || tt.initialStatus == StatusInConsult {
				dID := tt.actorID
				appt.DoctorID = &dID
			}
			// StartAppointment checks the -10m / +30m time window — set a valid StartTime.
			if tt.action == "start" {
				appt.StartTime = time.Now()
			}
			if err := repo.CreateAppointment(appt); err != nil {
				t.Fatalf("seed CreateAppointment: %v", err)
			}
			sid := idStr(appt)

			var err error
			switch tt.action {
			case "accept":
				err = svc.AcceptAppointment(sid, tt.actorID)
			case "start":
				err = svc.StartAppointment(sid)
			case "complete":
				err = svc.CompleteAppointment(context.Background(), sid)
			case "cancel":
				err = svc.CancelAppointment(sid, "test-reason")
			default:
				t.Fatalf("unknown action: %s", tt.action)
			}

			if tt.wantErr {
				if err == nil {
					final, _ := repo.GetAppointmentByID(sid)
					var actualStatus AppointmentStatus
					if final != nil {
						actualStatus = final.Status
					}
					t.Errorf("expected error but got nil (final status: %s)", actualStatus)
				}
				if tt.wantErrContains != "" && err != nil {
					if !strContains(err.Error(), tt.wantErrContains) {
						t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrContains)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify the status was persisted correctly.
			updated, fetchErr := repo.GetAppointmentByID(sid)
			if fetchErr != nil {
				t.Fatalf("could not fetch updated appointment: %v", fetchErr)
			}
			if updated.Status != tt.wantStatus {
				t.Errorf("expected status %q, got %q", tt.wantStatus, updated.Status)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// TestConcurrentAppointmentAcceptance — race condition test
// ──────────────────────────────────────────────────────────────────────────────

// TestConcurrentAppointmentAcceptance verifies that only one doctor can accept
// a pending appointment even when multiple goroutines race simultaneously.
// The tableRepo.UpdateAppointmentAtomic uses a mutex to simulate optimistic
// locking: the first caller wins; subsequent callers get RowsAffected=0.
func TestConcurrentAppointmentAcceptance(t *testing.T) {
	const numDoctors = 10

	repo := newTableRepo()
	fin := &mockFinance{}
	svc := NewService(repo, fin)

	// Create one pending appointment.
	appt := &Appointment{
		Status:    StatusPending,
		PatientID: 1,
		Version:   1,
	}
	if err := repo.CreateAppointment(appt); err != nil {
		t.Fatalf("seed CreateAppointment: %v", err)
	}
	sid := idStr(appt)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		winners []uint
		losers  []uint
	)

	for i := 0; i < numDoctors; i++ {
		wg.Add(1)
		doctorID := uint(100 + i)
		go func(dID uint) {
			defer wg.Done()
			err := svc.AcceptAppointment(sid, dID)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				winners = append(winners, dID)
			} else {
				losers = append(losers, dID)
			}
		}(doctorID)
	}

	wg.Wait()

	if len(winners) != 1 {
		t.Errorf("expected exactly 1 winner, got %d: winners=%v", len(winners), winners)
	}
	if len(losers) != numDoctors-1 {
		t.Errorf("expected %d losers, got %d", numDoctors-1, len(losers))
	}

	// Final status must be confirmed.
	final, err := repo.GetAppointmentByID(sid)
	if err != nil {
		t.Fatalf("could not fetch final appointment: %v", err)
	}
	if final.Status != StatusConfirmed {
		t.Errorf("expected status confirmed, got %s", final.Status)
	}
	if len(winners) == 1 {
		t.Logf("Winner: doctor %d out of %d concurrent attempts", winners[0], numDoctors)
	}
}

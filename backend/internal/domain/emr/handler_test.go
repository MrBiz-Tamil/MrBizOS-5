package emr

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ─── Mock Service ─────────────────────────────────────────────────────────────

type mockEMRService struct {
	mock.Mock
}

func (m *mockEMRService) CreatePrescription(ctx context.Context, req CreatePrescriptionRequest, doctorID uint) (*Prescription, error) {
	args := m.Called(ctx, req, doctorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Prescription), args.Error(1)
}
func (m *mockEMRService) GetDoctorPrescriptions(ctx context.Context, doctorID uint) ([]Prescription, error) {
	args := m.Called(ctx, doctorID)
	return args.Get(0).([]Prescription), args.Error(1)
}
func (m *mockEMRService) GetPatientPrescriptions(ctx context.Context, patientID uint) ([]Prescription, error) {
	args := m.Called(ctx, patientID)
	return args.Get(0).([]Prescription), args.Error(1)
}
func (m *mockEMRService) CreateMedicalRecord(ctx context.Context, rec MedicalRecord) (*MedicalRecord, error) {
	args := m.Called(ctx, rec)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MedicalRecord), args.Error(1)
}
func (m *mockEMRService) GetMedicalRecords(ctx context.Context, patientID, doctorID uint) ([]MedicalRecord, error) {
	args := m.Called(ctx, patientID, doctorID)
	return args.Get(0).([]MedicalRecord), args.Error(1)
}
func (m *mockEMRService) GetRecordDetails(ctx context.Context, id uint) (*MedicalRecord, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MedicalRecord), args.Error(1)
}
func (m *mockEMRService) UpdateMedicalRecord(ctx context.Context, id uint, updates map[string]interface{}) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}
func (m *mockEMRService) DeleteMedicalRecord(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockEMRService) CreateDiagnosis(ctx context.Context, d Diagnosis) (*Diagnosis, error) {
	args := m.Called(ctx, d)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Diagnosis), args.Error(1)
}
func (m *mockEMRService) GetDiagnosisHistory(ctx context.Context, patientID uint) ([]Diagnosis, error) {
	args := m.Called(ctx, patientID)
	return args.Get(0).([]Diagnosis), args.Error(1)
}
func (m *mockEMRService) GetPatientTimeline(ctx context.Context, patientID uint) ([]TimelineItem, error) {
	args := m.Called(ctx, patientID)
	return args.Get(0).([]TimelineItem), args.Error(1)
}
func (m *mockEMRService) GetPrescriptionByID(ctx context.Context, id uint, requesterUserID uint, requesterRole string) (*Prescription, error) {
	args := m.Called(ctx, id, requesterUserID, requesterRole)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Prescription), args.Error(1)
}
func (m *mockEMRService) GetPrescriptionByAppointment(ctx context.Context, appointmentID uint, requesterUserID uint, requesterRole string) (*Prescription, error) {
	args := m.Called(ctx, appointmentID, requesterUserID, requesterRole)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Prescription), args.Error(1)
}
func (m *mockEMRService) GetPatientPrescriptionsByID(ctx context.Context, patientUserID uint, requesterUserID uint, requesterRole string) ([]Prescription, error) {
	args := m.Called(ctx, patientUserID, requesterUserID, requesterRole)
	return args.Get(0).([]Prescription), args.Error(1)
}

// ─── Test setup helpers ───────────────────────────────────────────────────────

func newTestApp(h *Handler) *fiber.App {
	app := fiber.New()
	// Inject locals manually in test routes
	app.Get("/emr/medical-records/:id", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(testCallerID))
		c.Locals("role", testCallerRole)
		return h.GetRecordDetails(c)
	})
	app.Put("/emr/medical-records/:id", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(testCallerID))
		c.Locals("role", testCallerRole)
		return h.UpdateMedicalRecord(c)
	})
	app.Delete("/emr/medical-records/:id", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(testCallerID))
		c.Locals("role", testCallerRole)
		return h.DeleteMedicalRecord(c)
	})
	app.Get("/emr/timeline/:id", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(testCallerID))
		c.Locals("role", testCallerRole)
		return h.GetPatientTimeline(c)
	})
	app.Post("/emr/diagnosis", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(testCallerID))
		c.Locals("role", testCallerRole)
		return h.CreateDiagnosis(c)
	})
	return app
}

var (
	testCallerID   uint = 42
	testCallerRole      = "patient"
)

func doRequest(app *fiber.App, method, path string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, _ := app.Test(req, -1)
	rec := httptest.NewRecorder()
	rec.WriteHeader(resp.StatusCode)
	io.Copy(rec.Body, resp.Body)
	return rec
}

// ─── GetRecordDetails ownership tests ────────────────────────────────────────

func TestGetRecordDetails_OwnPatientCanAccess(t *testing.T) {
	svc := &mockEMRService{}
	h := &Handler{svc: svc}

	testCallerID = 10
	testCallerRole = "patient"

	record := &MedicalRecord{ID: 1, PatientID: 10, DoctorID: 5}
	svc.On("GetRecordDetails", mock.Anything, uint(1)).Return(record, nil)

	app := newTestApp(h)
	resp, _ := app.Test(httptest.NewRequest("GET", "/emr/medical-records/1", nil))
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetRecordDetails_DoctorCanAccess(t *testing.T) {
	svc := &mockEMRService{}
	h := &Handler{svc: svc}

	testCallerID = 5
	testCallerRole = "doctor"

	record := &MedicalRecord{ID: 1, PatientID: 10, DoctorID: 5}
	svc.On("GetRecordDetails", mock.Anything, uint(1)).Return(record, nil)

	app := newTestApp(h)
	resp, _ := app.Test(httptest.NewRequest("GET", "/emr/medical-records/1", nil))
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetRecordDetails_OtherPatientForbidden(t *testing.T) {
	svc := &mockEMRService{}
	h := &Handler{svc: svc}

	testCallerID = 99 // different patient
	testCallerRole = "patient"

	record := &MedicalRecord{ID: 1, PatientID: 10, DoctorID: 5}
	svc.On("GetRecordDetails", mock.Anything, uint(1)).Return(record, nil)

	app := newTestApp(h)
	resp, _ := app.Test(httptest.NewRequest("GET", "/emr/medical-records/1", nil))
	assert.Equal(t, 403, resp.StatusCode)
}

func TestGetRecordDetails_AdminCanAccess(t *testing.T) {
	svc := &mockEMRService{}
	h := &Handler{svc: svc}

	testCallerID = 1
	testCallerRole = "admin"

	record := &MedicalRecord{ID: 1, PatientID: 10, DoctorID: 5}
	svc.On("GetRecordDetails", mock.Anything, uint(1)).Return(record, nil)

	app := newTestApp(h)
	resp, _ := app.Test(httptest.NewRequest("GET", "/emr/medical-records/1", nil))
	assert.Equal(t, 200, resp.StatusCode)
}

// ─── UpdateMedicalRecord ownership tests ─────────────────────────────────────

func TestUpdateMedicalRecord_RecordingDoctorCanUpdate(t *testing.T) {
	svc := &mockEMRService{}
	h := &Handler{svc: svc}

	testCallerID = 5
	testCallerRole = "doctor"

	record := &MedicalRecord{ID: 1, PatientID: 10, DoctorID: 5}
	svc.On("GetRecordDetails", mock.Anything, uint(1)).Return(record, nil)
	svc.On("UpdateMedicalRecord", mock.Anything, uint(1), mock.Anything).Return(nil)

	body := strings.NewReader(`{"diagnosis":"updated"}`)
	app := newTestApp(h)
	req := httptest.NewRequest("PUT", "/emr/medical-records/1", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestUpdateMedicalRecord_DifferentDoctorForbidden(t *testing.T) {
	svc := &mockEMRService{}
	h := &Handler{svc: svc}

	testCallerID = 7 // different doctor
	testCallerRole = "doctor"

	record := &MedicalRecord{ID: 1, PatientID: 10, DoctorID: 5}
	svc.On("GetRecordDetails", mock.Anything, uint(1)).Return(record, nil)

	body := strings.NewReader(`{"diagnosis":"hacked"}`)
	app := newTestApp(h)
	req := httptest.NewRequest("PUT", "/emr/medical-records/1", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, 403, resp.StatusCode)
}

func TestUpdateMedicalRecord_PatientForbidden(t *testing.T) {
	svc := &mockEMRService{}
	h := &Handler{svc: svc}

	testCallerID = 10 // the patient
	testCallerRole = "patient"

	record := &MedicalRecord{ID: 1, PatientID: 10, DoctorID: 5}
	svc.On("GetRecordDetails", mock.Anything, uint(1)).Return(record, nil)

	body := strings.NewReader(`{"diagnosis":"patient tamper"}`)
	app := newTestApp(h)
	req := httptest.NewRequest("PUT", "/emr/medical-records/1", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, 403, resp.StatusCode)
}

// ─── DeleteMedicalRecord ownership tests ─────────────────────────────────────

func TestDeleteMedicalRecord_RecordingDoctorCanDelete(t *testing.T) {
	svc := &mockEMRService{}
	h := &Handler{svc: svc}

	testCallerID = 5
	testCallerRole = "doctor"

	record := &MedicalRecord{ID: 1, PatientID: 10, DoctorID: 5}
	svc.On("GetRecordDetails", mock.Anything, uint(1)).Return(record, nil)
	svc.On("DeleteMedicalRecord", mock.Anything, uint(1)).Return(nil)

	app := newTestApp(h)
	resp, _ := app.Test(httptest.NewRequest("DELETE", "/emr/medical-records/1", nil))
	assert.Equal(t, 200, resp.StatusCode)
}

func TestDeleteMedicalRecord_OtherDoctorForbidden(t *testing.T) {
	svc := &mockEMRService{}
	h := &Handler{svc: svc}

	testCallerID = 99
	testCallerRole = "doctor"

	record := &MedicalRecord{ID: 1, PatientID: 10, DoctorID: 5}
	svc.On("GetRecordDetails", mock.Anything, uint(1)).Return(record, nil)

	app := newTestApp(h)
	resp, _ := app.Test(httptest.NewRequest("DELETE", "/emr/medical-records/1", nil))
	assert.Equal(t, 403, resp.StatusCode)
}

// ─── GetPatientTimeline access tests ─────────────────────────────────────────

func TestGetPatientTimeline_OwnPatientCanAccess(t *testing.T) {
	svc := &mockEMRService{}
	h := &Handler{svc: svc}

	testCallerID = 10
	testCallerRole = "patient"
	svc.On("GetPatientTimeline", mock.Anything, uint(10)).Return([]TimelineItem{}, nil)

	app := newTestApp(h)
	resp, _ := app.Test(httptest.NewRequest("GET", "/emr/timeline/10", nil))
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetPatientTimeline_DoctorCanAccess(t *testing.T) {
	svc := &mockEMRService{}
	h := &Handler{svc: svc}

	testCallerID = 5
	testCallerRole = "doctor"
	svc.On("GetPatientTimeline", mock.Anything, uint(10)).Return([]TimelineItem{}, nil)

	app := newTestApp(h)
	resp, _ := app.Test(httptest.NewRequest("GET", "/emr/timeline/10", nil))
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetPatientTimeline_OtherPatientForbidden(t *testing.T) {
	svc := &mockEMRService{}
	h := &Handler{svc: svc}

	testCallerID = 99
	testCallerRole = "patient"

	app := newTestApp(h)
	resp, _ := app.Test(httptest.NewRequest("GET", "/emr/timeline/10", nil))
	assert.Equal(t, 403, resp.StatusCode)
}

// ─── CreateDiagnosis DoctorID enforcement ─────────────────────────────────────

func TestCreateDiagnosis_DoctorIDForcedFromToken(t *testing.T) {
	svc := &mockEMRService{}
	h := &Handler{svc: svc}

	testCallerID = 42
	testCallerRole = "doctor"

	// Expect DoctorID to be 42 (from token), even if request body has a different value
	svc.On("CreateDiagnosis", mock.Anything, mock.MatchedBy(func(d Diagnosis) bool {
		return d.DoctorID == 42
	})).Return(&Diagnosis{ID: 1, DoctorID: 42}, nil)

	// Body sends doctor_id=99 — should be overridden to 42
	body := strings.NewReader(`{"patient_id":10,"doctor_id":99,"diagnosis":"Flu"}`)
	app := newTestApp(h)
	req := httptest.NewRequest("POST", "/emr/diagnosis", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, 201, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	svc.AssertExpectations(t)
}

func TestDeleteMedicalRecord_RecordNotFound(t *testing.T) {
	svc := &mockEMRService{}
	h := &Handler{svc: svc}

	testCallerID = 5
	testCallerRole = "doctor"

	svc.On("GetRecordDetails", mock.Anything, uint(999)).Return(nil, errors.New("not found"))

	app := newTestApp(h)
	resp, _ := app.Test(httptest.NewRequest("DELETE", "/emr/medical-records/999", nil))
	assert.Equal(t, 404, resp.StatusCode)
}

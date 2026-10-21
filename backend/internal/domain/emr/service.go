package emr

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/document"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/pdf"
	"drayya-core/internal/utils/privacy"
	"time"

	"gorm.io/datatypes"
)

type Service interface {
	CreatePrescription(ctx context.Context, req CreatePrescriptionRequest, doctorID uint) (*Prescription, error)
	GetPrescriptionByID(ctx context.Context, id uint, requesterUserID uint, requesterRole string) (*Prescription, error)
	GetPrescriptionByAppointment(ctx context.Context, appointmentID uint, requesterUserID uint, requesterRole string) (*Prescription, error)
	GetDoctorPrescriptions(ctx context.Context, doctorID uint) ([]Prescription, error)
	GetPatientPrescriptions(ctx context.Context, patientID uint) ([]Prescription, error)
	GetPatientPrescriptionsByID(ctx context.Context, patientUserID uint, requesterUserID uint, requesterRole string) ([]Prescription, error)

	CreateMedicalRecord(ctx context.Context, rec MedicalRecord) (*MedicalRecord, error)
	GetMedicalRecords(ctx context.Context, patientID, doctorID uint) ([]MedicalRecord, error)
	GetRecordDetails(ctx context.Context, id uint) (*MedicalRecord, error)
	UpdateMedicalRecord(ctx context.Context, id uint, updates map[string]interface{}) error
	DeleteMedicalRecord(ctx context.Context, id uint) error

	CreateDiagnosis(ctx context.Context, d Diagnosis) (*Diagnosis, error)
	GetDiagnosisHistory(ctx context.Context, patientID uint) ([]Diagnosis, error)

	GetPatientTimeline(ctx context.Context, patientID uint) ([]TimelineItem, error)
}

type CreatePrescriptionRequest struct {
	AppointmentID uint            `json:"appointment_id"`
	Diagnosis     string          `json:"diagnosis"`
	Symptoms      string          `json:"symptoms"`
	// Legacy free-form medicines payload (kept for backward compatibility with
	// the older doctor UI). New clients should send the structured Items array.
	Medicines    json.RawMessage `json:"medicines"`
	Instructions string          `json:"instructions"`

	// Items is the structured prescription. Each entry references a Medicine in
	// the AYUSH product master and carries the dose plan the cart generator and
	// adherence reminder service need.
	Items []PrescriptionItemInput `json:"items,omitempty"`
}

// PrescriptionItemInput is the doctor's payload for one structured Rx line.
type PrescriptionItemInput struct {
	MedicineID      uint    `json:"medicine_id"`
	DosageText      string  `json:"dosage_text"`        // e.g. "2 tabs twice a day after meals"
	QuantityPerDose float64 `json:"quantity_per_dose"`  // e.g. 2
	FrequencyPerDay float64 `json:"frequency_per_day"`  // e.g. 2 (BID), 3 (TID)
	DurationDays    int     `json:"duration_days"`
	RefillsAllowed  int     `json:"refills_allowed"`
	Instructions    string  `json:"instructions"`
}

type TimelineItem struct {
	Type    string      `json:"type"`
	Date    time.Time   `json:"date"`
	Title   string      `json:"title"`
	Detail  string      `json:"detail"`
	RawData interface{} `json:"data"`
}

type service struct {
	repo         Repository
	identityRepo identity.Repository
	apptRepo     appointment.Repository
	pdfSvc       *pdf.Service
}

func NewService(repo Repository, iRepo identity.Repository, aRepo appointment.Repository) Service {
	return &service{
		repo:         repo,
		identityRepo: iRepo,
		apptRepo:     aRepo,
		pdfSvc:       pdf.NewService(),
	}
}

func (s *service) CreatePrescription(ctx context.Context, req CreatePrescriptionRequest, doctorID uint) (*Prescription, error) {
	// Validate Appointment
	appt, err := s.apptRepo.GetAppointmentByIDUint(req.AppointmentID)
	if err != nil {
		return nil, err
	}

	// Ensure the caller is the doctor assigned to this appointment
	if appt.DoctorID == nil || *appt.DoctorID != doctorID {
		return nil, fmt.Errorf("unauthorized: you are not the assigned doctor for appointment %d", req.AppointmentID)
	}

	prescription := &Prescription{
		AppointmentID: req.AppointmentID,
		DoctorID:      doctorID,
		PatientID:     appt.PatientID,
		Diagnosis:     privacy.EncryptedString(req.Diagnosis),
		Symptoms:      privacy.EncryptedString(req.Symptoms),
		Medicines:     datatypes.JSON(req.Medicines),
		Instructions:  privacy.EncryptedString(req.Instructions),
		Status:        "issued",
		CreatedAt:     time.Now(),
	}

	if err := s.repo.CreatePrescription(ctx, prescription); err != nil {
		return nil, err
	}

	// Structured items (P1): when the doctor sends `items`, validate each
	// medicine_id against the AYUSH catalog (raw SQL to avoid an import cycle
	// into the pharmacy domain) and write PrescriptionItem rows. This is what
	// the cart auto-generator reads — without it the cart falls back to
	// pharmacist-manual entry. Backward-compatible: if items is empty, we only
	// keep the legacy `medicines` JSON blob.
	if len(req.Items) > 0 {
		if err := s.writePrescriptionItems(ctx, prescription.ID, req.Items); err != nil {
			return nil, fmt.Errorf("prescription saved but items failed: %w", err)
		}
	}

	// Generate PDF
	doctor, _ := s.identityRepo.GetUserByID(doctorID)
	patient, _ := s.identityRepo.GetUserByID(appt.PatientID)

	if doctor != nil && patient != nil {
		pdfData := &pdf.PrescriptionData{
			ID:           prescription.ID,
			Diagnosis:    string(prescription.Diagnosis),
			DoctorName:   doctor.FullName,
			PatientName:  patient.FullName,
			Instructions: string(prescription.Instructions),
			Date:         prescription.CreatedAt,
		}

		var meds []map[string]interface{}
		if err := json.Unmarshal(prescription.Medicines, &meds); err == nil {
			for _, m := range meds {
				pdfData.Medicines = append(pdfData.Medicines, pdf.MedicineLine{
					Name:     fmt.Sprintf("%v", m["name"]),
					Dosage:   fmt.Sprintf("%v", m["dosage"]),
					Duration: fmt.Sprintf("%v", m["duration"]),
					Notes:    fmt.Sprintf("%v", m["instructions"]),
				})
			}
		}

		pdfPath, err := s.pdfSvc.GeneratePrescription(pdfData)
		if err == nil {
			prescription.PdfURL = pdfPath // default to local path
			// Move the generated PDF into the private vault and keep a vault:<id>
			// reference, so it survives redeploys and ×N replicas. Falls back to
			// the local path if the vault is unavailable.
			if f, oerr := os.Open(pdfPath); oerr == nil {
				if fi, serr := f.Stat(); serr == nil {
					if doc, derr := document.SavePrivateReader(ctx, f, fi.Size(), filepath.Base(pdfPath), "application/pdf", "prescription_pdf", document.OwnerPatient, prescription.PatientID); derr == nil {
						prescription.PdfURL = fmt.Sprintf("vault:%d", doc.ID)
						_ = os.Remove(pdfPath) // cleaned up; bytes now in the vault
					}
				}
				f.Close()
			}
			s.repo.UpdatePrescription(ctx, prescription)
		} else {
			fmt.Printf("       PDF Generation Error: %v\n", err)
		}
	}

	return prescription, nil
}

// writePrescriptionItems validates every medicine_id exists + is active and
// inserts one PrescriptionItem row per line. Validation is done with raw SQL
// against the medicine table because the emr package can't import pharmacy
// without creating a cycle.
func (s *service) writePrescriptionItems(ctx context.Context, prescriptionID uint, items []PrescriptionItemInput) error {
	db := s.repo.GetDB().WithContext(ctx)

	// Collect distinct medicine IDs and confirm they're all valid in one query.
	idSet := map[uint]struct{}{}
	ids := make([]uint, 0, len(items))
	for _, it := range items {
		if it.MedicineID == 0 {
			return fmt.Errorf("each prescription item must reference a medicine")
		}
		if _, ok := idSet[it.MedicineID]; !ok {
			idSet[it.MedicineID] = struct{}{}
			ids = append(ids, it.MedicineID)
		}
	}
	var validCount int64
	if err := db.Table("medicine").Where("id IN ? AND is_active = ?", ids, true).Count(&validCount).Error; err != nil {
		return err
	}
	if int(validCount) != len(ids) {
		return fmt.Errorf("one or more medicine IDs are invalid or inactive")
	}

	now := time.Now()
	rows := make([]PrescriptionItem, 0, len(items))
	for _, in := range items {
		dur := in.DurationDays
		if dur < 0 {
			dur = 0
		}
		qty := in.QuantityPerDose
		freq := in.FrequencyPerDay
		// total_quantity = qty/dose × doses/day × days; if any factor is missing
		// fall back to 1 so the cart doesn't compute a zero quantity. The doctor's
		// free-text dosage_text remains the authoritative human-readable plan.
		total := qty * freq * float64(dur)
		if total <= 0 {
			total = 1
		}
		rows = append(rows, PrescriptionItem{
			PrescriptionID:  prescriptionID,
			MedicineID:      in.MedicineID,
			DosageText:      in.DosageText,
			QuantityPerDose: qty,
			FrequencyPerDay: freq,
			DurationDays:    dur,
			TotalQuantity:   total,
			RefillsAllowed:  in.RefillsAllowed,
			Instructions:    in.Instructions,
			CreatedAt:       now,
			UpdatedAt:       now,
		})
	}
	return db.Create(&rows).Error
}

func (s *service) isEMRDoctor(role string) bool {
	return role == "doctor" || role == "chief_doctor" || role == "hospital_head_doctor"
}

func (s *service) doctorHasActiveAccessToPatient(doctorID, patientID uint) bool {
	var exists bool
	err := s.repo.GetDB().Raw(`
		SELECT EXISTS (
			SELECT 1 FROM emr_access_grants 
			WHERE doctor_id = ? AND patient_id = ? AND expires_at > ?
		)`, doctorID, patientID, time.Now()).Scan(&exists).Error
	return err == nil && exists
}

func (s *service) GetPrescriptionByID(ctx context.Context, id uint, requesterUserID uint, requesterRole string) (*Prescription, error) {
	p, err := s.repo.GetPrescriptionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if requesterRole == "admin" || requesterRole == "super_admin" {
		return p, nil
	}
	if p.DoctorID == requesterUserID || p.PatientID == requesterUserID {
		return p, nil
	}
	if s.isEMRDoctor(requesterRole) && s.doctorHasActiveAccessToPatient(requesterUserID, p.PatientID) {
		return p, nil
	}
	return nil, fmt.Errorf("forbidden: access denied")
}

func (s *service) GetPrescriptionByAppointment(ctx context.Context, appointmentID uint, requesterUserID uint, requesterRole string) (*Prescription, error) {
	p, err := s.repo.GetPrescriptionByAppointmentID(ctx, appointmentID)
	if err != nil {
		return nil, err
	}
	if requesterRole == "admin" || requesterRole == "super_admin" {
		return p, nil
	}
	if p.DoctorID == requesterUserID || p.PatientID == requesterUserID {
		return p, nil
	}
	if s.isEMRDoctor(requesterRole) && s.doctorHasActiveAccessToPatient(requesterUserID, p.PatientID) {
		return p, nil
	}
	return nil, fmt.Errorf("forbidden: access denied")
}

func (s *service) GetDoctorPrescriptions(ctx context.Context, doctorID uint) ([]Prescription, error) {
	return s.repo.GetPrescriptionsByDoctor(ctx, doctorID)
}

func (s *service) GetPatientPrescriptions(ctx context.Context, patientID uint) ([]Prescription, error) {
	return s.repo.GetPrescriptionsByPatient(ctx, patientID)
}

func (s *service) GetPatientPrescriptionsByID(ctx context.Context, patientUserID uint, requesterUserID uint, requesterRole string) ([]Prescription, error) {
	if requesterRole == "admin" || requesterRole == "super_admin" {
		return s.repo.GetPrescriptionsByPatient(ctx, patientUserID)
	}
	if patientUserID == requesterUserID {
		return s.repo.GetPrescriptionsByPatient(ctx, patientUserID)
	}
	if s.isEMRDoctor(requesterRole) {
		if s.doctorHasActiveAccessToPatient(requesterUserID, patientUserID) {
			return s.repo.GetPrescriptionsByPatient(ctx, patientUserID)
		}
		return nil, fmt.Errorf("forbidden: access expired or unauthorized")
	}
	return nil, fmt.Errorf("forbidden: access denied")
}

func (s *service) CreateMedicalRecord(ctx context.Context, rec MedicalRecord) (*MedicalRecord, error) {
	if err := s.repo.CreateMedicalRecord(ctx, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *service) GetMedicalRecords(ctx context.Context, patientID, doctorID uint) ([]MedicalRecord, error) {
	if patientID != 0 {
		return s.repo.GetMedicalRecordsByPatient(ctx, patientID)
	}
	if doctorID != 0 {
		return s.repo.GetMedicalRecordsByDoctor(ctx, doctorID)
	}
	// Fallback or potentially error
	return nil, nil
}

func (s *service) GetRecordDetails(ctx context.Context, id uint) (*MedicalRecord, error) {
	return s.repo.GetMedicalRecordByID(ctx, id)
}

func (s *service) UpdateMedicalRecord(ctx context.Context, id uint, updates map[string]interface{}) error {
	return s.repo.UpdateMedicalRecord(ctx, id, updates)
}

func (s *service) DeleteMedicalRecord(ctx context.Context, id uint) error {
	return s.repo.DeleteMedicalRecord(ctx, id)
}

func (s *service) CreateDiagnosis(ctx context.Context, d Diagnosis) (*Diagnosis, error) {
	if err := s.repo.CreateDiagnosis(ctx, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *service) GetDiagnosisHistory(ctx context.Context, patientID uint) ([]Diagnosis, error) {
	return s.repo.GetDiagnosisHistory(ctx, patientID)
}

func (s *service) GetPatientTimeline(ctx context.Context, patientID uint) ([]TimelineItem, error) {
	var timeline []TimelineItem

	// 1. Appointments
	appts, _ := s.apptRepo.GetAppointmentsByPatientOrDoctor(patientID, true, 50)
	for _, a := range appts {
		timeline = append(timeline, TimelineItem{
			Type:    "appointment",
			Date:    a.CreatedAt,
			Title:   "Consultation",
			Detail:  string(a.Status),
			RawData: a,
		})
	}

	// 2. Medical Records
	recs, _ := s.repo.GetMedicalRecordsByPatient(ctx, patientID)
	for _, r := range recs {
		timeline = append(timeline, TimelineItem{
			Type:    "medical_record",
			Date:    r.CreatedAt,
			Title:   "Medical Note",
			Detail:  string(r.Diagnosis),
			RawData: r,
		})
	}

	// 3. Prescriptions
	prescs, _ := s.repo.GetPrescriptionsByPatient(ctx, patientID)
	for _, p := range prescs {
		timeline = append(timeline, TimelineItem{
			Type:    "prescription",
			Date:    p.CreatedAt,
			Title:   "Prescription Issued",
			Detail:  string(p.Diagnosis),
			RawData: p,
		})
	}

	// Sort timeline by date desc
	// Omitted for brevity in this step, but in practice use sort.Slice

	return timeline, nil
}


package hospital

import (
	"context"
	"time"

	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/pharmacy"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Hospital{},
		&HospitalDepartment{},
		&Bed{},
		&WardServiceRequest{},
		&IPDAdmission{},
		&Ward{},
		&NursingTask{},
		&ShiftSchedule{},
	)
}

type Repository interface {
	CountBeds(ctx context.Context, hospitalID uint) (int64, error)
	CountOccupiedBeds(ctx context.Context, hospitalID uint) (int64, error)
	CountWards(ctx context.Context, hospitalID uint) (int64, error)
	CountOPDByDate(ctx context.Context, hospitalID uint, date time.Time) (int64, error)
	CountOPDByStatus(ctx context.Context, hospitalID uint, status string) (int64, error)
	CountIPDByStatus(ctx context.Context, hospitalID uint, status string) (int64, error)

	GetOPDQueue(ctx context.Context, hospitalID uint) ([]appointment.Appointment, error)
	GetOPDByID(ctx context.Context, opdID uint) (*appointment.Appointment, error)
	GetNextWaitingOPD(ctx context.Context, hospitalID uint) (*appointment.Appointment, error)
	UpdateOPDStatus(ctx context.Context, opdID uint, status string) error
	CreateAppointment(ctx context.Context, appt *appointment.Appointment) error

	GetBedByID(ctx context.Context, bedID uint) (*Bed, error)
	ListBedsByWard(ctx context.Context, wardID uint) ([]Bed, error)
	ListBedsByHospital(ctx context.Context, hospitalID uint) ([]Bed, error)
	UpdateBedStatus(ctx context.Context, bedID uint, status string) error

	CreateAdmission(ctx context.Context, admission *IPDAdmission) error
	GetAdmissionByID(ctx context.Context, admissionID uint) (*IPDAdmission, error)
	UpdateAdmission(ctx context.Context, admission *IPDAdmission) error
	GetIPDAdmissions(ctx context.Context, hospitalID uint) ([]IPDAdmission, error)

	GetAvailableBeds(ctx context.Context, hospitalID uint) ([]Bed, error)
	GetHospitalInventory(ctx context.Context, hospitalID uint) ([]pharmacy.PharmacyInventory, error)
	GetInventoryItem(ctx context.Context, itemID uint) (*pharmacy.PharmacyInventory, error)
	UpdateInventoryItem(ctx context.Context, item *pharmacy.PharmacyInventory) error
	GetInventoryItemByMedicine(ctx context.Context, medicineID uint) (*pharmacy.PharmacyInventory, error)

	CreateWardServiceRequest(ctx context.Context, req *WardServiceRequest) error
	GetWardServiceRequests(ctx context.Context, hospitalID uint) ([]WardServiceRequest, error)
	UpdateWardServiceRequestStatus(ctx context.Context, requestID uint, status string) error

	ListAllHospitals(ctx context.Context, hospitalType string) ([]Hospital, error)
	CreateHospital(ctx context.Context, hospital *Hospital) error
	UpdateHospital(ctx context.Context, hospital *Hospital) error
	GetHospitalByID(ctx context.Context, id uint) (*Hospital, error)

	// Wards
	ListWards(ctx context.Context, hospitalID uint) ([]Ward, error)
	GetWardByID(ctx context.Context, id uint) (*Ward, error)
	CreateWard(ctx context.Context, w *Ward) error
	UpdateWard(ctx context.Context, id uint, updates map[string]interface{}) error

	// Tasks
	ListTasks(ctx context.Context, hospitalID uint, status string) ([]NursingTask, error)
	GetTaskByID(ctx context.Context, id uint) (*NursingTask, error)
	CreateTask(ctx context.Context, t *NursingTask) error
	UpdateTask(ctx context.Context, id uint, updates map[string]interface{}) error

	// Shifts
	ListShifts(ctx context.Context, hospitalID uint, date *time.Time) ([]ShiftSchedule, error)
	CreateShift(ctx context.Context, s *ShiftSchedule) error

	ExecuteTransaction(ctx context.Context, fn func(txRepo Repository) error) error
}

type defaultRepository struct {
	db *gorm.DB
}

func NewDefaultRepository(db *gorm.DB) Repository {
	return &defaultRepository{db: db}
}

func (r *defaultRepository) withTx(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return r.db.WithContext(ctx)
}

type txKey struct{}

func (r *defaultRepository) ExecuteTransaction(ctx context.Context, fn func(txRepo Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		txRepo := &defaultRepository{db: tx.WithContext(txCtx)}
		return fn(txRepo)
	})
}

func (r *defaultRepository) CountBeds(ctx context.Context, hospitalID uint) (int64, error) {
	var count int64
	err := r.withTx(ctx).Model(&Bed{}).Count(&count).Error
	return count, err
}

func (r *defaultRepository) CountOccupiedBeds(ctx context.Context, hospitalID uint) (int64, error) {
	var count int64
	err := r.withTx(ctx).Model(&Bed{}).Where("status = ?", "occupied").Count(&count).Error
	return count, err
}

func (r *defaultRepository) GetOPDQueue(ctx context.Context, hospitalID uint) ([]appointment.Appointment, error) {
	var appts []appointment.Appointment
	err := r.withTx(ctx).Preload("Patient").Preload("Doctor").
		Where("status = ?", "confirmed").
		Limit(20).Find(&appts).Error
	return appts, err
}

func (r *defaultRepository) CreateAppointment(ctx context.Context, appt *appointment.Appointment) error {
	return r.withTx(ctx).Create(appt).Error
}

func (r *defaultRepository) GetBedByID(ctx context.Context, bedID uint) (*Bed, error) {
	var bed Bed
	err := r.withTx(ctx).First(&bed, bedID).Error
	return &bed, err
}

func (r *defaultRepository) CreateAdmission(ctx context.Context, admission *IPDAdmission) error {
	return r.withTx(ctx).Create(admission).Error
}

func (r *defaultRepository) UpdateBedStatus(ctx context.Context, bedID uint, status string) error {
	return r.withTx(ctx).Model(&Bed{}).Where("id = ?", bedID).Update("status", status).Error
}

func (r *defaultRepository) GetAdmissionByID(ctx context.Context, admissionID uint) (*IPDAdmission, error) {
	var admission IPDAdmission
	err := r.withTx(ctx).First(&admission, admissionID).Error
	return &admission, err
}

func (r *defaultRepository) UpdateAdmission(ctx context.Context, admission *IPDAdmission) error {
	return r.withTx(ctx).Save(admission).Error
}

func (r *defaultRepository) GetIPDAdmissions(ctx context.Context, hospitalID uint) ([]IPDAdmission, error) {
	var admissions []IPDAdmission
	err := r.withTx(ctx).Preload("Patient").Preload("Doctor").Preload("Bed").
		Where("hospital_id = ? AND status != ?", hospitalID, "discharged").
		Order("admission_date desc").Find(&admissions).Error
	return admissions, err
}

func (r *defaultRepository) GetAvailableBeds(ctx context.Context, hospitalID uint) ([]Bed, error) {
	var beds []Bed
	err := r.withTx(ctx).Where("hospital_id = ? AND status = ?", hospitalID, "available").Find(&beds).Error
	return beds, err
}

func (r *defaultRepository) GetHospitalInventory(ctx context.Context, hospitalID uint) ([]pharmacy.PharmacyInventory, error) {
	var inventory []pharmacy.PharmacyInventory
	err := r.withTx(ctx).Preload("Medicine").
		Limit(50).Find(&inventory).Error
	return inventory, err
}

func (r *defaultRepository) GetInventoryItem(ctx context.Context, itemID uint) (*pharmacy.PharmacyInventory, error) {
	var item pharmacy.PharmacyInventory
	err := r.withTx(ctx).First(&item, itemID).Error
	return &item, err
}

func (r *defaultRepository) UpdateInventoryItem(ctx context.Context, item *pharmacy.PharmacyInventory) error {
	return r.withTx(ctx).Save(item).Error
}

func (r *defaultRepository) GetInventoryItemByMedicine(ctx context.Context, medicineID uint) (*pharmacy.PharmacyInventory, error) {
	var inv pharmacy.PharmacyInventory
	err := r.withTx(ctx).Where("medicine_id = ? AND stock_quantity > 0", medicineID).First(&inv).Error // Removed quantity condition to allow query and then check
	return &inv, err
}

func (r *defaultRepository) CreateWardServiceRequest(ctx context.Context, req *WardServiceRequest) error {
	return r.withTx(ctx).Create(req).Error
}

func (r *defaultRepository) GetWardServiceRequests(ctx context.Context, hospitalID uint) ([]WardServiceRequest, error) {
	var requests []WardServiceRequest
	err := r.withTx(ctx).Where("hospital_id = ?", hospitalID).Order("created_at desc").Find(&requests).Error
	return requests, err
}

func (r *defaultRepository) UpdateWardServiceRequestStatus(ctx context.Context, requestID uint, status string) error {
	return r.withTx(ctx).Model(&WardServiceRequest{}).Where("id = ?", requestID).Update("status", status).Error
}

func (r *defaultRepository) ListAllHospitals(ctx context.Context, hospitalType string) ([]Hospital, error) {
	var hospitals []Hospital
	query := r.withTx(ctx).Preload("Franchise")
	if hospitalType != "" && hospitalType != "all" {
		query = query.Where("hospital_type = ?", hospitalType)
	}
	err := query.Find(&hospitals).Error
	return hospitals, err
}

func (r *defaultRepository) CreateHospital(ctx context.Context, hospital *Hospital) error {
	return r.withTx(ctx).Create(hospital).Error
}

func (r *defaultRepository) UpdateHospital(ctx context.Context, hospital *Hospital) error {
	return r.withTx(ctx).Save(hospital).Error
}

func (r *defaultRepository) GetHospitalByID(ctx context.Context, id uint) (*Hospital, error) {
	var h Hospital
	err := r.withTx(ctx).First(&h, id).Error
	return &h, err
}

// ── New repository methods (Wards, Tasks, Shifts, OPD actions) ────────────────

func (r *defaultRepository) CountWards(ctx context.Context, hospitalID uint) (int64, error) {
	var count int64
	err := r.withTx(ctx).Model(&Ward{}).Where("is_active = ?", true).Count(&count).Error
	return count, err
}

func (r *defaultRepository) CountOPDByDate(ctx context.Context, hospitalID uint, date time.Time) (int64, error) {
	var count int64
	end := date.Add(24 * time.Hour)
	err := r.withTx(ctx).Model(&appointment.Appointment{}).
		Where("start_time >= ? AND start_time < ?", date, end).
		Count(&count).Error
	return count, err
}

func (r *defaultRepository) CountOPDByStatus(ctx context.Context, hospitalID uint, status string) (int64, error) {
	var count int64
	today := time.Now().Truncate(24 * time.Hour)
	end := today.Add(24 * time.Hour)
	err := r.withTx(ctx).Model(&appointment.Appointment{}).
		Where("status = ? AND start_time >= ? AND start_time < ?", status, today, end).
		Count(&count).Error
	return count, err
}

func (r *defaultRepository) CountIPDByStatus(ctx context.Context, hospitalID uint, status string) (int64, error) {
	var count int64
	err := r.withTx(ctx).Model(&IPDAdmission{}).
		Where("hospital_id = ? AND status = ?", hospitalID, status).
		Count(&count).Error
	return count, err
}

func (r *defaultRepository) GetOPDByID(ctx context.Context, opdID uint) (*appointment.Appointment, error) {
	var appt appointment.Appointment
	err := r.withTx(ctx).First(&appt, opdID).Error
	return &appt, err
}

func (r *defaultRepository) GetNextWaitingOPD(ctx context.Context, hospitalID uint) (*appointment.Appointment, error) {
	var appt appointment.Appointment
	today := time.Now().Truncate(24 * time.Hour)
	end := today.Add(24 * time.Hour)
	err := r.withTx(ctx).
		Where("status IN ? AND start_time >= ? AND start_time < ?",
			[]string{"confirmed", "waiting"}, today, end).
		Order("start_time ASC").
		First(&appt).Error
	return &appt, err
}

func (r *defaultRepository) UpdateOPDStatus(ctx context.Context, opdID uint, status string) error {
	return r.withTx(ctx).Model(&appointment.Appointment{}).
		Where("id = ?", opdID).Update("status", status).Error
}

func (r *defaultRepository) ListBedsByWard(ctx context.Context, wardID uint) ([]Bed, error) {
	var beds []Bed
	err := r.withTx(ctx).Where("ward_id = ?", wardID).Order("bed_number ASC").Find(&beds).Error
	return beds, err
}

func (r *defaultRepository) ListBedsByHospital(ctx context.Context, hospitalID uint) ([]Bed, error) {
	var beds []Bed
	err := r.withTx(ctx).Where("hospital_id = ?", hospitalID).Order("ward_id, bed_number").Find(&beds).Error
	return beds, err
}

func (r *defaultRepository) ListWards(ctx context.Context, hospitalID uint) ([]Ward, error) {
	var wards []Ward
	err := r.withTx(ctx).Where("hospital_id = ? AND is_active = ?", hospitalID, true).
		Order("floor ASC, name ASC").Find(&wards).Error
	if err != nil {
		return nil, err
	}

	// Compute current_occupancy + available_beds per ward
	for i := range wards {
		var occ, avail int64
		r.withTx(ctx).Model(&Bed{}).Where("ward_id = ? AND status = ?", wards[i].ID, "occupied").Count(&occ)
		r.withTx(ctx).Model(&Bed{}).Where("ward_id = ? AND status = ?", wards[i].ID, "available").Count(&avail)
		wards[i].CurrentOccupancy = int(occ)
		wards[i].AvailableBeds = int(avail)
	}
	return wards, nil
}

func (r *defaultRepository) GetWardByID(ctx context.Context, id uint) (*Ward, error) {
	var w Ward
	err := r.withTx(ctx).First(&w, id).Error
	return &w, err
}

func (r *defaultRepository) CreateWard(ctx context.Context, w *Ward) error {
	return r.withTx(ctx).Create(w).Error
}

func (r *defaultRepository) UpdateWard(ctx context.Context, id uint, updates map[string]interface{}) error {
	return r.withTx(ctx).Model(&Ward{}).Where("id = ?", id).Updates(updates).Error
}

func (r *defaultRepository) ListTasks(ctx context.Context, hospitalID uint, status string) ([]NursingTask, error) {
	var tasks []NursingTask
	qry := r.withTx(ctx).Where("hospital_id = ?", hospitalID)
	if status != "" {
		qry = qry.Where("status = ?", status)
	} else {
		qry = qry.Where("status NOT IN ?", []string{"completed", "cancelled"})
	}
	err := qry.Order("priority DESC, due_at ASC NULLS LAST, created_at ASC").Find(&tasks).Error
	if err != nil {
		// SQLite doesn't support NULLS LAST — fall back without it
		err = r.withTx(ctx).Where("hospital_id = ?", hospitalID).
			Order("created_at DESC").Find(&tasks).Error
	}
	return tasks, err
}

func (r *defaultRepository) GetTaskByID(ctx context.Context, id uint) (*NursingTask, error) {
	var t NursingTask
	err := r.withTx(ctx).First(&t, id).Error
	return &t, err
}

func (r *defaultRepository) CreateTask(ctx context.Context, t *NursingTask) error {
	return r.withTx(ctx).Create(t).Error
}

func (r *defaultRepository) UpdateTask(ctx context.Context, id uint, updates map[string]interface{}) error {
	return r.withTx(ctx).Model(&NursingTask{}).Where("id = ?", id).Updates(updates).Error
}

func (r *defaultRepository) ListShifts(ctx context.Context, hospitalID uint, date *time.Time) ([]ShiftSchedule, error) {
	var shifts []ShiftSchedule
	qry := r.withTx(ctx).Where("hospital_id = ?", hospitalID)
	if date != nil {
		end := date.Add(24 * time.Hour)
		qry = qry.Where("shift_date >= ? AND shift_date < ?", *date, end)
	}
	err := qry.Order("shift_date ASC, start_time ASC").Find(&shifts).Error
	return shifts, err
}

func (r *defaultRepository) CreateShift(ctx context.Context, s *ShiftSchedule) error {
	return r.withTx(ctx).Create(s).Error
}

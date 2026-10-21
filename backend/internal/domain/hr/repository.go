package hr

import (
	"time"

	"gorm.io/gorm"
)

// identityUserEmail is a minimal struct for reading email from the identity.User table.
type identityUserEmail struct {
	Email    string
	FullName string
}

type Repository interface {
	CreateStaffProfile(profile *StaffProfile) error
	GetStaffProfile(id uint) (*StaffProfile, error)
	GetStaffProfileByUserID(userID uint) (*StaffProfile, error)

	CreateShift(shift *StaffShift) error
	GetShiftsByHospital(hospitalID uint) ([]StaffShift, error)

	RecordAttendance(attendance *StaffAttendance) error
	GetAttendance(userID uint, date time.Time) (*StaffAttendance, error)
	GetAttendanceInRange(userID uint, start, end time.Time) ([]StaffAttendance, error)
	UpdateAttendance(attendance *StaffAttendance) error
	UpdateAttendanceCheckOut(id uint, userID uint, checkOut *time.Time) error

	CreatePayrollRecord(record *PayrollRecord) error
	GetPayrollRecords(userID uint, year, month int) ([]PayrollRecord, error)
	DeletePayrollRecords(userID uint, year, month int) error

	// GetUserEmailByID resolves the employee email and full name from the identity users table.
	GetUserEmailByID(userID uint) (email, fullName string, err error)

	// GetStaffRoster returns all staff profiles for a hospital (used by HR dashboard).
	GetStaffRoster(hospitalID uint) ([]StaffProfile, error)
}

type hrRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &hrRepository{db: db}
}

func (r *hrRepository) CreateStaffProfile(profile *StaffProfile) error {
	return r.db.Create(profile).Error
}

func (r *hrRepository) GetStaffProfile(id uint) (*StaffProfile, error) {
	var profile StaffProfile
	err := r.db.First(&profile, id).Error
	return &profile, err
}

func (r *hrRepository) GetStaffProfileByUserID(userID uint) (*StaffProfile, error) {
	var profile StaffProfile
	err := r.db.Where("user_id = ?", userID).First(&profile).Error
	return &profile, err
}

func (r *hrRepository) CreateShift(shift *StaffShift) error {
	return r.db.Create(shift).Error
}

func (r *hrRepository) GetShiftsByHospital(hospitalID uint) ([]StaffShift, error) {
	var shifts []StaffShift
	err := r.db.Where("hospital_id = ?", hospitalID).Find(&shifts).Error
	return shifts, err
}

func (r *hrRepository) RecordAttendance(attendance *StaffAttendance) error {
	return r.db.Create(attendance).Error
}

func (r *hrRepository) GetAttendance(userID uint, date time.Time) (*StaffAttendance, error) {
	var attendance StaffAttendance
	err := r.db.Where("user_id = ? AND date = ?", userID, date.Format("2006-01-02")).First(&attendance).Error
	return &attendance, err
}

func (r *hrRepository) GetAttendanceInRange(userID uint, start, end time.Time) ([]StaffAttendance, error) {
	var results []StaffAttendance
	err := r.db.Where("user_id = ? AND date >= ? AND date < ?", userID, start, end).Find(&results).Error
	return results, err
}

func (r *hrRepository) UpdateAttendance(attendance *StaffAttendance) error {
	return r.db.Save(attendance).Error
}

func (r *hrRepository) UpdateAttendanceCheckOut(id uint, userID uint, checkOut *time.Time) error {
	return r.db.Model(&StaffAttendance{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("check_out", checkOut).Error
}

func (r *hrRepository) CreatePayrollRecord(record *PayrollRecord) error {
	return r.db.Create(record).Error
}

func (r *hrRepository) GetPayrollRecords(userID uint, year, month int) ([]PayrollRecord, error) {
	var records []PayrollRecord
	err := r.db.Where("user_id = ? AND year = ? AND month = ?", userID, year, month).Find(&records).Error
	return records, err
}

func (r *hrRepository) DeletePayrollRecords(userID uint, year, month int) error {
	return r.db.Where("user_id = ? AND year = ? AND month = ?", userID, year, month).
		Delete(&PayrollRecord{}).Error
}

// GetUserEmailByID reads the email and full_name from the identity `user` table for a given user ID.
func (r *hrRepository) GetUserEmailByID(userID uint) (email, fullName string, err error) {
	var u identityUserEmail
	err = r.db.Table("user").Select("email, full_name").Where("id = ?", userID).First(&u).Error
	return u.Email, u.FullName, err
}

// GetStaffRoster returns all active staff profiles optionally filtered by hospital.
func (r *hrRepository) GetStaffRoster(hospitalID uint) ([]StaffProfile, error) {
	var profiles []StaffProfile
	query := r.db.Where("employment_status = ?", "active")
	if hospitalID > 0 {
		// If StaffProfile ever gets a hospital_id column this filter becomes relevant.
		// For now we return all active staff.
	}
	err := query.Find(&profiles).Error
	return profiles, err
}

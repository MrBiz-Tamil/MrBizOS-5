package travel

import (
	"gorm.io/gorm"
)

type Repository interface {
	CreateDestination(dest *Destination) error
	ListDestinations() ([]Destination, error)
	CreateTourPackage(pkg *TourPackage) error
	ListTourPackages(destID uint) ([]TourPackage, error)
	GetTourPackage(id uint) (*TourPackage, error)
	CreateBooking(booking *TravelBooking) error
	ListBookings(customerID uint) ([]TravelBooking, error)
	CreateVisaChecklist(checklist *VisaChecklist) error
	GetVisaChecklist(countryCode string, visaType string) (*VisaChecklist, error)
	CreateVisaApplication(app *VisaApplication) error
	GetVisaApplication(id uint) (*VisaApplication, error)
	ListVisaApplications(applicantID uint) ([]VisaApplication, error)
	UpdateVisaApplication(app *VisaApplication) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateDestination(dest *Destination) error {
	return r.db.Create(dest).Error
}

func (r *repository) ListDestinations() ([]Destination, error) {
	var dests []Destination
	err := r.db.Where("is_active = ?", true).Order("name ASC").Find(&dests).Error
	return dests, err
}

func (r *repository) CreateTourPackage(pkg *TourPackage) error {
	return r.db.Create(pkg).Error
}

func (r *repository) ListTourPackages(destID uint) ([]TourPackage, error) {
	var pkgs []TourPackage
	q := r.db.Preload("Destination").Where("is_active = ?", true)
	if destID > 0 {
		q = q.Where("destination_id = ?", destID)
	}
	err := q.Order("price ASC").Find(&pkgs).Error
	return pkgs, err
}

func (r *repository) GetTourPackage(id uint) (*TourPackage, error) {
	var pkg TourPackage
	err := r.db.Preload("Destination").First(&pkg, id).Error
	if err != nil {
		return nil, err
	}
	return &pkg, nil
}

func (r *repository) CreateBooking(booking *TravelBooking) error {
	return r.db.Create(booking).Error
}

func (r *repository) ListBookings(customerID uint) ([]TravelBooking, error) {
	var bookings []TravelBooking
	q := r.db.Preload("TourPackage.Destination")
	if customerID > 0 {
		q = q.Where("customer_id = ?", customerID)
	}
	err := q.Order("created_at DESC").Find(&bookings).Error
	return bookings, err
}

func (r *repository) CreateVisaChecklist(checklist *VisaChecklist) error {
	return r.db.Create(checklist).Error
}

func (r *repository) GetVisaChecklist(countryCode string, visaType string) (*VisaChecklist, error) {
	var checklist VisaChecklist
	err := r.db.Where("country_code = ? AND visa_type = ?", countryCode, visaType).First(&checklist).Error
	if err != nil {
		return nil, err
	}
	return &checklist, nil
}

func (r *repository) CreateVisaApplication(app *VisaApplication) error {
	return r.db.Create(app).Error
}

func (r *repository) GetVisaApplication(id uint) (*VisaApplication, error) {
	var app VisaApplication
	err := r.db.First(&app, id).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *repository) ListVisaApplications(applicantID uint) ([]VisaApplication, error) {
	var apps []VisaApplication
	q := r.db.Model(&VisaApplication{})
	if applicantID > 0 {
		q = q.Where("applicant_id = ?", applicantID)
	}
	err := q.Order("created_at DESC").Find(&apps).Error
	return apps, err
}

func (r *repository) UpdateVisaApplication(app *VisaApplication) error {
	return r.db.Save(app).Error
}

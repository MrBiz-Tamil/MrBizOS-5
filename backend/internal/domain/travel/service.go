package travel

import (
	"errors"

	"drayya-core/internal/domain/journey"
	"drayya-core/internal/domain/task"
)

type Service interface {
	CreateDestination(dest *Destination) (*Destination, error)
	ListDestinations() ([]Destination, error)
	CreateTourPackage(pkg *TourPackage) (*TourPackage, error)
	ListTourPackages(destID uint) ([]TourPackage, error)
	GetTourPackage(id uint) (*TourPackage, error)
	CreateBooking(customerID uint, packageID uint, passengers int, travelDate string) (*TravelBooking, error)
	ListBookings(customerID uint) ([]TravelBooking, error)
	CreateVisaChecklist(checklist *VisaChecklist) (*VisaChecklist, error)
	GetVisaChecklist(countryCode string, visaType string) (*VisaChecklist, error)
	SubmitVisaApplication(applicantID uint, countryCode string, visaType string, passportNumber string) (*VisaApplication, error)
	UpdateVisaStatus(appID uint, status string, notes string) (*VisaApplication, error)
	ListVisaApplications(applicantID uint) ([]VisaApplication, error)
}

type service struct {
	repo        Repository
	journeySvc  journey.Service
	taskSvc     task.Service
}

func NewService(repo Repository, journeySvc journey.Service, taskSvc task.Service) Service {
	return &service{
		repo:       repo,
		journeySvc: journeySvc,
		taskSvc:    taskSvc,
	}
}

func (s *service) CreateDestination(dest *Destination) (*Destination, error) {
	if dest.Name == "" || dest.Country == "" {
		return nil, errors.New("destination name and country are required")
	}
	if err := s.repo.CreateDestination(dest); err != nil {
		return nil, err
	}
	return dest, nil
}

func (s *service) ListDestinations() ([]Destination, error) {
	return s.repo.ListDestinations()
}

func (s *service) CreateTourPackage(pkg *TourPackage) (*TourPackage, error) {
	if pkg.Title == "" || pkg.DestinationID == 0 {
		return nil, errors.New("tour package title and destination_id are required")
	}
	if err := s.repo.CreateTourPackage(pkg); err != nil {
		return nil, err
	}
	return pkg, nil
}

func (s *service) ListTourPackages(destID uint) ([]TourPackage, error) {
	return s.repo.ListTourPackages(destID)
}

func (s *service) GetTourPackage(id uint) (*TourPackage, error) {
	return s.repo.GetTourPackage(id)
}

func (s *service) CreateBooking(customerID uint, packageID uint, passengers int, travelDate string) (*TravelBooking, error) {
	pkg, err := s.repo.GetTourPackage(packageID)
	if err != nil {
		return nil, errors.New("tour package not found")
	}

	if passengers <= 0 {
		passengers = 1
	}

	totalPrice := pkg.Price * float64(passengers)

	booking := &TravelBooking{
		OrganizationID: 1,
		BusinessUnitID: 1, // Travel
		CustomerID:     customerID,
		TourPackageID:  packageID,
		Passengers:     passengers,
		TotalPrice:     totalPrice,
		Status:         "CONFIRMED",
	}

	if err := s.repo.CreateBooking(booking); err != nil {
		return nil, err
	}

	// Auto-create confirmation task for travel team
	if s.taskSvc != nil {
		_, _ = s.taskSvc.CreateTask(&task.Task{
			OrganizationID: 1,
			BusinessUnitID: 1,
			CustomerID:     &customerID,
			OwnerID:        1,
			Title:          "Confirm Flight & Hotel Vouchers for Tour #" + pkg.Title,
			Priority:       task.PriorityHigh,
		})
	}

	return booking, nil
}

func (s *service) ListBookings(customerID uint) ([]TravelBooking, error) {
	return s.repo.ListBookings(customerID)
}

func (s *service) CreateVisaChecklist(checklist *VisaChecklist) (*VisaChecklist, error) {
	if checklist.CountryCode == "" || checklist.VisaType == "" {
		return nil, errors.New("country_code and visa_type are required")
	}
	if err := s.repo.CreateVisaChecklist(checklist); err != nil {
		return nil, err
	}
	return checklist, nil
}

func (s *service) GetVisaChecklist(countryCode string, visaType string) (*VisaChecklist, error) {
	return s.repo.GetVisaChecklist(countryCode, visaType)
}

func (s *service) SubmitVisaApplication(applicantID uint, countryCode string, visaType string, passportNumber string) (*VisaApplication, error) {
	app := &VisaApplication{
		OrganizationID: 1,
		BusinessUnitID: 1,
		ApplicantID:    applicantID,
		CountryCode:    countryCode,
		VisaType:       visaType,
		PassportNumber: passportNumber,
		Status:         "SUBMITTED",
	}

	if err := s.repo.CreateVisaApplication(app); err != nil {
		return nil, err
	}

	// Auto-create document verification task
	if s.taskSvc != nil {
		_, _ = s.taskSvc.CreateTask(&task.Task{
			OrganizationID: 1,
			BusinessUnitID: 1,
			CustomerID:     &applicantID,
			OwnerID:        1,
			Title:          "Verify Visa Documents & Passport for " + countryCode + " (" + visaType + ")",
			Priority:       task.PriorityHigh,
		})
	}

	return app, nil
}

func (s *service) UpdateVisaStatus(appID uint, status string, notes string) (*VisaApplication, error) {
	app, err := s.repo.GetVisaApplication(appID)
	if err != nil {
		return nil, errors.New("visa application not found")
	}

	app.Status = status
	if notes != "" {
		app.Notes = notes
	}

	if err := s.repo.UpdateVisaApplication(app); err != nil {
		return nil, err
	}
	return app, nil
}

func (s *service) ListVisaApplications(applicantID uint) ([]VisaApplication, error) {
	return s.repo.ListVisaApplications(applicantID)
}

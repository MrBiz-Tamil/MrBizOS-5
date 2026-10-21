package realestate

import (
	"errors"
	"fmt"
	"time"

	"drayya-core/internal/domain/journey"
	"drayya-core/internal/domain/task"
)

type Service interface {
	CreateProperty(prop *Property) (*Property, error)
	GetProperty(id uint) (*Property, error)
	ListProperties(city string, txType string, maxPrice float64) ([]Property, error)
	ScheduleSiteVisit(propertyID uint, buyerID uint, agentID uint, scheduledAt time.Time, notes string) (*SiteVisit, error)
	ListSiteVisits(agentID uint, buyerID uint) ([]SiteVisit, error)
	CreateDeal(propertyID uint, buyerID uint, sellerID uint, agreedPrice float64) (*PropertyDeal, error)
	UpdateDealStatus(dealID uint, status string) (*PropertyDeal, error)
	ListDeals() ([]PropertyDeal, error)
}

type service struct {
	repo       Repository
	journeySvc journey.Service
	taskSvc    task.Service
}

func NewService(repo Repository, journeySvc journey.Service, taskSvc task.Service) Service {
	return &service{
		repo:       repo,
		journeySvc: journeySvc,
		taskSvc:    taskSvc,
	}
}

func (s *service) CreateProperty(prop *Property) (*Property, error) {
	if prop.Title == "" || prop.Price <= 0 || prop.City == "" {
		return nil, errors.New("property title, price, and city are required")
	}
	if prop.PropertyType == "" {
		prop.PropertyType = TypeApartment
	}
	if prop.TransactionType == "" {
		prop.TransactionType = TxSale
	}
	if err := s.repo.CreateProperty(prop); err != nil {
		return nil, err
	}
	return prop, nil
}

func (s *service) GetProperty(id uint) (*Property, error) {
	return s.repo.GetProperty(id)
}

func (s *service) ListProperties(city string, txType string, maxPrice float64) ([]Property, error) {
	return s.repo.ListProperties(city, txType, maxPrice)
}

func (s *service) ScheduleSiteVisit(propertyID uint, buyerID uint, agentID uint, scheduledAt time.Time, notes string) (*SiteVisit, error) {
	prop, err := s.repo.GetProperty(propertyID)
	if err != nil {
		return nil, errors.New("property not found")
	}

	visit := &SiteVisit{
		OrganizationID: 1,
		BusinessUnitID: 3, // Real Estate
		PropertyID:     propertyID,
		BuyerID:        buyerID,
		AgentID:        agentID,
		ScheduledAt:    scheduledAt,
		Status:         "SCHEDULED",
		Notes:          notes,
	}

	// Auto-create task for real estate agent
	if s.taskSvc != nil {
		t, err := s.taskSvc.CreateTask(&task.Task{
			OrganizationID: 1,
			BusinessUnitID: 3,
			CustomerID:     &buyerID,
			AssigneeID:     &agentID,
			OwnerID:        1,
			Title:          fmt.Sprintf("Conduct Site Visit for %s (Client #%d)", prop.Title, buyerID),
			DueDate:        &scheduledAt,
			Priority:       task.PriorityHigh,
		})
		if err == nil && t != nil {
			visit.TaskID = &t.ID
		}
	}

	if err := s.repo.CreateSiteVisit(visit); err != nil {
		return nil, err
	}
	return visit, nil
}

func (s *service) ListSiteVisits(agentID uint, buyerID uint) ([]SiteVisit, error) {
	return s.repo.ListSiteVisits(agentID, buyerID)
}

func (s *service) CreateDeal(propertyID uint, buyerID uint, sellerID uint, agreedPrice float64) (*PropertyDeal, error) {
	prop, err := s.repo.GetProperty(propertyID)
	if err != nil {
		return nil, errors.New("property not found")
	}

	commissionAmount := agreedPrice * 0.02 // Default 2% commission

	deal := &PropertyDeal{
		OrganizationID:   1,
		BusinessUnitID:   3, // Real Estate
		PropertyID:       propertyID,
		BuyerID:          buyerID,
		SellerID:         sellerID,
		AgreedPrice:      agreedPrice,
		CommissionRate:   2.0,
		CommissionAmount: commissionAmount,
		Status:           "OFFER",
	}

	if err := s.repo.CreateDeal(deal); err != nil {
		return nil, err
	}

	// Reserve property listing
	prop.Status = "RESERVED"
	_ = s.repo.UpdateProperty(prop)

	return deal, nil
}

func (s *service) UpdateDealStatus(dealID uint, status string) (*PropertyDeal, error) {
	deal, err := s.repo.GetDeal(dealID)
	if err != nil {
		return nil, errors.New("property deal not found")
	}

	deal.Status = status
	if status == "CLOSED" && deal.Property != nil {
		deal.Property.Status = "SOLD"
		_ = s.repo.UpdateProperty(deal.Property)
	}

	if err := s.repo.UpdateDeal(deal); err != nil {
		return nil, err
	}
	return deal, nil
}

func (s *service) ListDeals() ([]PropertyDeal, error) {
	return s.repo.ListDeals()
}

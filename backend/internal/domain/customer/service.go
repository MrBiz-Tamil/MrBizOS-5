package customer

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Service interface {
	CreateLead(lead *Lead) (*Lead, error)
	ListLeads(buID uint, status string) ([]Lead, error)
	GetCustomer360(id uint, hasClinicalAccess bool) (*Customer360Profile, error)
}

type service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

func (s *service) CreateLead(lead *Lead) (*Lead, error) {
	if lead.Name == "" {
		return nil, errors.New("lead name is required")
	}
	if lead.Status == "" {
		lead.Status = "NEW"
	}
	if err := s.db.Create(lead).Error; err != nil {
		return nil, err
	}
	return lead, nil
}

func (s *service) ListLeads(buID uint, status string) ([]Lead, error) {
	var leads []Lead
	q := s.db.Model(&Lead{})
	if buID > 0 {
		q = q.Where("business_unit_id = ?", buID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Order("created_at DESC").Find(&leads).Error
	return leads, err
}

func (s *service) GetCustomer360(id uint, hasClinicalAccess bool) (*Customer360Profile, error) {
	// Aggregate user data
	var lead Lead
	if err := s.db.First(&lead, id).Error; err != nil {
		return nil, errors.New("customer lead record not found")
	}

	profile := &Customer360Profile{
		ID:                    lead.ID,
		OrganizationID:        lead.OrganizationID,
		Name:                  lead.Name,
		Email:                 lead.Email,
		Phone:                 lead.Phone,
		PrimaryBusinessUnitID: lead.BusinessUnitID,
		ActiveRelationships: []BusinessRelationship{
			{
				BusinessUnitID:   lead.BusinessUnitID,
				BusinessUnitName: getBUName(lead.BusinessUnitID),
				Status:           lead.Status,
				RelationshipID:   fmt.Sprintf("CUST-%05d", lead.ID),
			},
		},
		Timeline: []TimelineEvent{
			{
				ID:             fmt.Sprintf("EVT-%d", time.Now().UnixNano()),
				BusinessUnitID: lead.BusinessUnitID,
				EventType:      "LEAD_INTAKE",
				Summary:        fmt.Sprintf("Customer inquiry received via %s", lead.Source),
				Timestamp:      lead.CreatedAt,
				IsClinical:     false,
			},
		},
		TotalInvoiced:      lead.Value,
		TotalPaid:          0,
		OutstandingBalance: lead.Value,
		CreatedAt:          lead.CreatedAt,
		UpdatedAt:          lead.UpdatedAt,
	}

	// Filter timeline events for clinical PHI security guard
	if !hasClinicalAccess {
		sanitizedTimeline := make([]TimelineEvent, 0)
		for _, evt := range profile.Timeline {
			if !evt.IsClinical {
				sanitizedTimeline = append(sanitizedTimeline, evt)
			}
		}
		profile.Timeline = sanitizedTimeline
	}

	return profile, nil
}

func getBUName(id uint) string {
	switch id {
	case 1:
		return "Travel, Tours & Visa"
	case 2:
		return "Healthcare"
	case 3:
		return "Real Estate"
	case 4:
		return "E-Commerce & POD"
	case 5:
		return "Software Consulting"
	default:
		return "Group Operations"
	}
}

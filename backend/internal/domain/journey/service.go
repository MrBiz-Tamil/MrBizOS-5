package journey

import (
	"errors"
	"gorm.io/gorm"
)

type Service interface {
	CreateTemplate(tmpl *JourneyTemplate) (*JourneyTemplate, error)
	GetTemplateByCode(code string) (*JourneyTemplate, error)
	ListTemplates(buID uint) ([]JourneyTemplate, error)
	StartJourney(tmplID uint, customerID uint, leadID *uint, buID uint, orgID uint) (*JourneyInstance, error)
	TransitionStage(instanceID uint, nextStageID uint, actorID uint, notes string) (*JourneyInstance, error)
	GetInstance(id uint) (*JourneyInstance, error)
	ListInstances(buID uint, customerID uint) ([]JourneyInstance, error)
}

type service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

func (s *service) CreateTemplate(tmpl *JourneyTemplate) (*JourneyTemplate, error) {
	if tmpl.Code == "" || tmpl.Name == "" {
		return nil, errors.New("template code and name are required")
	}
	if err := s.db.Create(tmpl).Error; err != nil {
		return nil, err
	}
	return tmpl, nil
}

func (s *service) GetTemplateByCode(code string) (*JourneyTemplate, error) {
	var tmpl JourneyTemplate
	err := s.db.Preload("Stages", func(db *gorm.DB) *gorm.DB {
		return db.Order("stage_order ASC")
	}).Where("code = ?", code).First(&tmpl).Error
	if err != nil {
		return nil, err
	}
	return &tmpl, nil
}

func (s *service) ListTemplates(buID uint) ([]JourneyTemplate, error) {
	var templates []JourneyTemplate
	q := s.db.Preload("Stages")
	if buID > 0 {
		q = q.Where("business_unit_id = ?", buID)
	}
	err := q.Find(&templates).Error
	return templates, err
}

func (s *service) StartJourney(tmplID uint, customerID uint, leadID *uint, buID uint, orgID uint) (*JourneyInstance, error) {
	var tmpl JourneyTemplate
	if err := s.db.Preload("Stages", func(db *gorm.DB) *gorm.DB {
		return db.Order("stage_order ASC")
	}).First(&tmpl, tmplID).Error; err != nil {
		return nil, errors.New("journey template not found")
	}

	if len(tmpl.Stages) == 0 {
		return nil, errors.New("journey template has no stages configured")
	}

	firstStage := tmpl.Stages[0]

	instance := &JourneyInstance{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		TemplateID:     tmplID,
		CustomerID:     customerID,
		LeadID:         leadID,
		CurrentStageID: firstStage.ID,
		Status:         "ACTIVE",
	}

	if err := s.db.Create(instance).Error; err != nil {
		return nil, err
	}

	log := &JourneyLog{
		InstanceID: instance.ID,
		FromStage:  "START",
		ToStage:    firstStage.StageName,
		ActorID:    customerID,
		Notes:      "Journey initiated",
	}
	_ = s.db.Create(log)

	return instance, nil
}

func (s *service) TransitionStage(instanceID uint, nextStageID uint, actorID uint, notes string) (*JourneyInstance, error) {
	var instance JourneyInstance
	if err := s.db.Preload("CurrentStage").First(&instance, instanceID).Error; err != nil {
		return nil, errors.New("journey instance not found")
	}

	var nextStage JourneyStage
	if err := s.db.First(&nextStage, nextStageID).Error; err != nil {
		return nil, errors.New("next stage not found")
	}

	fromName := "UNKNOWN"
	if instance.CurrentStage != nil {
		fromName = instance.CurrentStage.StageName
	}

	instance.CurrentStageID = nextStage.ID
	instance.CurrentStage = &nextStage
	if err := s.db.Model(&instance).Update("current_stage_id", nextStage.ID).Error; err != nil {
		return nil, err
	}

	log := &JourneyLog{
		InstanceID: instance.ID,
		FromStage:  fromName,
		ToStage:    nextStage.StageName,
		ActorID:    actorID,
		Notes:      notes,
	}
	_ = s.db.Create(log)

	return &instance, nil
}

func (s *service) GetInstance(id uint) (*JourneyInstance, error) {
	var instance JourneyInstance
	err := s.db.Preload("CurrentStage").Preload("Logs").First(&instance, id).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (s *service) ListInstances(buID uint, customerID uint) ([]JourneyInstance, error) {
	var instances []JourneyInstance
	q := s.db.Preload("CurrentStage")
	if buID > 0 {
		q = q.Where("business_unit_id = ?", buID)
	}
	if customerID > 0 {
		q = q.Where("customer_id = ?", customerID)
	}
	err := q.Order("created_at DESC").Find(&instances).Error
	return instances, err
}

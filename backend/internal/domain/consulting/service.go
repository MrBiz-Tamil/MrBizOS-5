package consulting

import (
	"errors"
	"fmt"
	"time"

	"drayya-core/internal/domain/journey"
	"drayya-core/internal/domain/task"
)

type Service interface {
	CreateClient(client *ClientAccount) (*ClientAccount, error)
	ListClients() ([]ClientAccount, error)
	CreateProject(clientID uint, title string, code string, budget float64) (*SoftwareProject, error)
	GetProject(id uint) (*SoftwareProject, error)
	ListProjects(clientID uint) ([]SoftwareProject, error)
	CreateMilestone(projectID uint, title string, amount float64, targetDate *time.Time) (*ProjectMilestone, error)
	ListMilestones(projectID uint) ([]ProjectMilestone, error)
	LogTimesheet(projectID uint, userID uint, hours float64, description string) (*Timesheet, error)
	ListTimesheets(projectID uint, userID uint) ([]Timesheet, error)
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

func (s *service) CreateClient(client *ClientAccount) (*ClientAccount, error) {
	if client.CompanyName == "" {
		return nil, errors.New("company_name is required")
	}
	if err := s.repo.CreateClient(client); err != nil {
		return nil, err
	}
	return client, nil
}

func (s *service) ListClients() ([]ClientAccount, error) {
	return s.repo.ListClients()
}

func (s *service) CreateProject(clientID uint, title string, code string, budget float64) (*SoftwareProject, error) {
	if title == "" || budget <= 0 {
		return nil, errors.New("project title and budget are required")
	}
	if code == "" {
		code = fmt.Sprintf("PRJ-%d", time.Now().UnixNano()%10000)
	}

	proj := &SoftwareProject{
		OrganizationID: 1,
		BusinessUnitID: 5, // Software Consulting
		ClientID:       clientID,
		Title:          title,
		Code:           code,
		Budget:         budget,
		Status:         "IN_PROGRESS",
	}

	if err := s.repo.CreateProject(proj); err != nil {
		return nil, err
	}

	// Auto-create Kickoff task for engineering team
	if s.taskSvc != nil {
		_, _ = s.taskSvc.CreateTask(&task.Task{
			OrganizationID: 1,
			BusinessUnitID: 5,
			OwnerID:        1,
			Title:          fmt.Sprintf("Architecture & Sprint Setup for %s (%s)", title, code),
			Priority:       task.PriorityHigh,
		})
	}

	return proj, nil
}

func (s *service) GetProject(id uint) (*SoftwareProject, error) {
	return s.repo.GetProject(id)
}

func (s *service) ListProjects(clientID uint) ([]SoftwareProject, error) {
	return s.repo.ListProjects(clientID)
}

func (s *service) CreateMilestone(projectID uint, title string, amount float64, targetDate *time.Time) (*ProjectMilestone, error) {
	proj, err := s.repo.GetProject(projectID)
	if err != nil {
		return nil, errors.New("software project not found")
	}

	ms := &ProjectMilestone{
		ProjectID:  projectID,
		Title:      title,
		Amount:     amount,
		TargetDate: targetDate,
		Status:     "PENDING",
	}

	if err := s.repo.CreateMilestone(ms); err != nil {
		return nil, err
	}

	// Auto-create milestone task
	if s.taskSvc != nil {
		_, _ = s.taskSvc.CreateTask(&task.Task{
			OrganizationID: 1,
			BusinessUnitID: 5,
			OwnerID:        1,
			Title:          fmt.Sprintf("Deliver Milestone '%s' for Project %s", title, proj.Title),
			DueDate:        targetDate,
			Priority:       task.PriorityMedium,
		})
	}

	return ms, nil
}

func (s *service) ListMilestones(projectID uint) ([]ProjectMilestone, error) {
	return s.repo.ListMilestones(projectID)
}

func (s *service) LogTimesheet(projectID uint, userID uint, hours float64, description string) (*Timesheet, error) {
	if hours <= 0 {
		return nil, errors.New("hours must be greater than 0")
	}

	ts := &Timesheet{
		ProjectID:   projectID,
		UserID:      userID,
		WorkDate:    time.Now(),
		Hours:       hours,
		Description: description,
		Billable:    true,
	}

	if err := s.repo.LogTimesheet(ts); err != nil {
		return nil, err
	}

	// Track spent budget on project
	proj, err := s.repo.GetProject(projectID)
	if err == nil && proj != nil {
		proj.Spent += (hours * 85.0) // Standard hourly rate $85
		_ = s.repo.UpdateProject(proj)
	}

	return ts, nil
}

func (s *service) ListTimesheets(projectID uint, userID uint) ([]Timesheet, error) {
	return s.repo.ListTimesheets(projectID, userID)
}

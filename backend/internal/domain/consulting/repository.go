package consulting

import (
	"gorm.io/gorm"
)

type Repository interface {
	CreateClient(client *ClientAccount) error
	ListClients() ([]ClientAccount, error)
	CreateProject(proj *SoftwareProject) error
	GetProject(id uint) (*SoftwareProject, error)
	ListProjects(clientID uint) ([]SoftwareProject, error)
	UpdateProject(proj *SoftwareProject) error
	CreateMilestone(ms *ProjectMilestone) error
	ListMilestones(projectID uint) ([]ProjectMilestone, error)
	LogTimesheet(ts *Timesheet) error
	ListTimesheets(projectID uint, userID uint) ([]Timesheet, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateClient(client *ClientAccount) error {
	return r.db.Create(client).Error
}

func (r *repository) ListClients() ([]ClientAccount, error) {
	var clients []ClientAccount
	err := r.db.Order("company_name ASC").Find(&clients).Error
	return clients, err
}

func (r *repository) CreateProject(proj *SoftwareProject) error {
	return r.db.Create(proj).Error
}

func (r *repository) GetProject(id uint) (*SoftwareProject, error) {
	var proj SoftwareProject
	err := r.db.Preload("Client").First(&proj, id).Error
	if err != nil {
		return nil, err
	}
	return &proj, nil
}

func (r *repository) ListProjects(clientID uint) ([]SoftwareProject, error) {
	var projs []SoftwareProject
	q := r.db.Preload("Client")
	if clientID > 0 {
		q = q.Where("client_id = ?", clientID)
	}
	err := q.Order("created_at DESC").Find(&projs).Error
	return projs, err
}

func (r *repository) UpdateProject(proj *SoftwareProject) error {
	return r.db.Save(proj).Error
}

func (r *repository) CreateMilestone(ms *ProjectMilestone) error {
	return r.db.Create(ms).Error
}

func (r *repository) ListMilestones(projectID uint) ([]ProjectMilestone, error) {
	var mss []ProjectMilestone
	err := r.db.Where("project_id = ?", projectID).Order("target_date ASC").Find(&mss).Error
	return mss, err
}

func (r *repository) LogTimesheet(ts *Timesheet) error {
	return r.db.Create(ts).Error
}

func (r *repository) ListTimesheets(projectID uint, userID uint) ([]Timesheet, error) {
	var tss []Timesheet
	q := r.db.Model(&Timesheet{})
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	err := q.Order("work_date DESC").Find(&tss).Error
	return tss, err
}

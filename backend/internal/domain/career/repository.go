package career

import (
	"gorm.io/gorm"
)

type Repository interface {
	GetJobs() ([]JobOpening, error)
	GetJobByID(id uint) (*JobOpening, error)
	CreateApplication(app *JobApplication) error
	GetApplicationsByUserID(userID uint) ([]JobApplication, error)
	GetApplicationByJobAndEmail(jobID uint, email string) (*JobApplication, error)
	GetApplicationByEmail(email string) (*JobApplication, error)
	GetAllApplications() ([]JobApplication, error)
	UpdateApplicationStatus(id uint, status string) error
	UpdateApplicationStatusWithNotes(id uint, status, interviewDate, notes string) error
	GetApplicationByID(id uint) (*JobApplication, error)
	CreateJob(job *JobOpening) error
	UpdateJob(job *JobOpening) error
	GetAllJobs() ([]JobOpening, error)

	GetTemplates() ([]RecruitmentTemplate, error)
	GetTemplateByID(id uint) (*RecruitmentTemplate, error)
	GetTemplateByTypeAndTitle(tType, title string) (*RecruitmentTemplate, error)
	CreateTemplate(t *RecruitmentTemplate) error
	UpdateTemplate(t *RecruitmentTemplate) error
	DeleteTemplate(id uint) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetJobs() ([]JobOpening, error) {
	var jobs []JobOpening
	err := r.db.Where("is_active = ?", true).Order("created_at DESC").Find(&jobs).Error
	return jobs, err
}

func (r *repository) GetJobByID(id uint) (*JobOpening, error) {
	var job JobOpening
	err := r.db.First(&job, id).Error
	return &job, err
}

func (r *repository) CreateApplication(app *JobApplication) error {
	return r.db.Create(app).Error
}

func (r *repository) GetApplicationsByUserID(userID uint) ([]JobApplication, error) {
	var apps []JobApplication
	err := r.db.Preload("Job").Where("user_id = ?", userID).Order("applied_at DESC").Find(&apps).Error
	return apps, err
}

func (r *repository) GetApplicationByJobAndEmail(jobID uint, email string) (*JobApplication, error) {
	var app JobApplication
	err := r.db.Where("job_id = ? AND email = ?", jobID, email).First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *repository) GetApplicationByEmail(email string) (*JobApplication, error) {
	var app JobApplication
	err := r.db.Where("email = ?", email).Order("applied_at DESC").First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *repository) UpdateApplicationStatusWithNotes(id uint, status, interviewDate, notes string) error {
	updates := map[string]interface{}{"status": status}
	if interviewDate != "" {
		updates["interview_date"] = interviewDate
	}
	if notes != "" {
		updates["interview_notes"] = notes
	}
	return r.db.Model(&JobApplication{}).Where("id = ?", id).Updates(updates).Error
}

func (r *repository) GetAllApplications() ([]JobApplication, error) {
	var apps []JobApplication
	err := r.db.Preload("Job").Order("applied_at DESC").Find(&apps).Error
	return apps, err
}

func (r *repository) UpdateApplicationStatus(id uint, status string) error {
	return r.db.Model(&JobApplication{}).Where("id = ?", id).Update("status", status).Error
}

func (r *repository) GetApplicationByID(id uint) (*JobApplication, error) {
	var app JobApplication
	err := r.db.First(&app, id).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *repository) CreateJob(job *JobOpening) error {
	return r.db.Create(job).Error
}

func (r *repository) UpdateJob(job *JobOpening) error {
	return r.db.Save(job).Error
}

func (r *repository) GetAllJobs() ([]JobOpening, error) {
	var jobs []JobOpening
	err := r.db.Order("created_at DESC").Find(&jobs).Error
	return jobs, err
}

func (r *repository) GetTemplates() ([]RecruitmentTemplate, error) {
	var templates []RecruitmentTemplate
	err := r.db.Order("id ASC").Find(&templates).Error
	return templates, err
}

func (r *repository) GetTemplateByID(id uint) (*RecruitmentTemplate, error) {
	var template RecruitmentTemplate
	err := r.db.First(&template, id).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *repository) CreateTemplate(t *RecruitmentTemplate) error {
	return r.db.Create(t).Error
}

func (r *repository) UpdateTemplate(t *RecruitmentTemplate) error {
	return r.db.Save(t).Error
}

func (r *repository) DeleteTemplate(id uint) error {
	return r.db.Delete(&RecruitmentTemplate{}, id).Error
}

func (r *repository) GetTemplateByTypeAndTitle(tType, title string) (*RecruitmentTemplate, error) {
	var template RecruitmentTemplate
	err := r.db.Where("type = ? AND title = ?", tType, title).First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

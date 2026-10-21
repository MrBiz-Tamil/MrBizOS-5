package models

import (
	"time"
)

type JobOpening struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Title        string    `gorm:"not null" json:"title"`
	Department   string    `json:"department"`
	Location     string    `json:"location"`
	JobType      string    `json:"type"` // full-time, part-time, contract
	Description  string    `gorm:"type:text" json:"description"`
	Requirements string    `gorm:"type:text" json:"requirements"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (JobOpening) TableName() string {
	return "job_openings"
}

type JobApplication struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	JobID        uint      `gorm:"index" json:"job_id"`
	Job          JobOpening `gorm:"foreignKey:JobID" json:"job,omitempty"`
	UserID       *uint     `gorm:"index" json:"user_id"`
	FullName     string    `json:"full_name"`
	Email        string    `json:"email"`
	ResumeURL    string    `json:"resume_url"`
	Status       string    `gorm:"default:'applied'" json:"status"` // applied, interviewing, offered, rejected
	CoverLetter  string    `gorm:"type:text" json:"cover_letter"`
	AppliedAt    time.Time `json:"applied_at"`
}

func (JobApplication) TableName() string {
	return "job_applications"
}

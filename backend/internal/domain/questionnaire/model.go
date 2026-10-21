package questionnaire

import (
	"time"

	"gorm.io/gorm"
)

// QuestionType is the kind of input expected.
type QuestionType string

const (
	TypeText       QuestionType = "text"
	TypeMultiChoice QuestionType = "multi_choice"
	TypeSingleChoice QuestionType = "single_choice"
	TypeYesNo      QuestionType = "yes_no"
	TypeScale      QuestionType = "scale" // 1–10
	TypeDate       QuestionType = "date"
)

// Template is a reusable questionnaire definition (e.g. "General", "Cardiology", "Paediatrics").
type Template struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:200;not null;uniqueIndex" json:"name"`
	Specialty   string         `gorm:"size:100" json:"specialty"` // empty = general
	Description string         `gorm:"type:text" json:"description"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	Questions   []Question     `gorm:"foreignKey:TemplateID" json:"questions,omitempty"`
}

// Question is one item in a template.
type Question struct {
	ID           uint         `gorm:"primaryKey" json:"id"`
	TemplateID   uint         `gorm:"index;not null" json:"template_id"`
	SortOrder    int          `json:"sort_order"`
	Text         string       `gorm:"type:text;not null" json:"text"`
	Type         QuestionType `gorm:"size:30;not null" json:"type"`
	Options      string       `gorm:"type:text" json:"options,omitempty"` // JSON array for choice types
	IsRequired   bool         `gorm:"default:false" json:"is_required"`
	IsConditional bool        `gorm:"default:false" json:"is_conditional"`
	ShowIfQID    *uint        `json:"show_if_q_id,omitempty"` // show this question only if that Q is answered with ShowIfValue
	ShowIfValue  string       `gorm:"size:200" json:"show_if_value,omitempty"`
}

// Response is a completed pre-consultation form submitted by a patient.
type Response struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	TemplateID     uint           `gorm:"index;not null" json:"template_id"`
	PatientUserID  uint           `gorm:"index;not null" json:"patient_user_id"`
	ConsultationID *uint          `gorm:"index" json:"consultation_id,omitempty"` // linked after booking
	Answers        []Answer       `gorm:"foreignKey:ResponseID" json:"answers,omitempty"`
	Status         string         `gorm:"size:20;default:'submitted'" json:"status"` // submitted, reviewed
	ReviewedBy     *uint          `json:"reviewed_by,omitempty"`
	ReviewedAt     *time.Time     `json:"reviewed_at,omitempty"`
	Summary        string         `gorm:"type:text" json:"summary,omitempty"` // AI-generated or doctor notes
}

// Answer holds one question's response within a Response.
type Answer struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	ResponseID uint   `gorm:"index;not null" json:"response_id"`
	QuestionID uint   `gorm:"index;not null" json:"question_id"`
	Value      string `gorm:"type:text" json:"value"` // always stored as string
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Template{}, &Question{}, &Response{}, &Answer{})
}

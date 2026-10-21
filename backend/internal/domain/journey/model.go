package journey

import (
	"time"

	"gorm.io/gorm"
)

// JourneyTemplate defines a multi-stage business process for a specific Business Unit.
type JourneyTemplate struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	OrganizationID uint           `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID uint           `gorm:"index;not null" json:"business_unit_id"`
	Code           string         `gorm:"size:64;uniqueIndex;not null" json:"code"` // e.g. "TRAVEL_VISA", "REAL_ESTATE_DEAL", "SOFTWARE_SOW"
	Name           string         `gorm:"size:255;not null" json:"name"`
	Description    string         `gorm:"type:text" json:"description,omitempty"`
	IsActive       bool           `gorm:"default:true" json:"is_active"`
	Stages         []JourneyStage `gorm:"foreignKey:TemplateID;constraint:OnDelete:CASCADE" json:"stages,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// JourneyStage is an individual milestone stage inside a JourneyTemplate.
type JourneyStage struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TemplateID  uint      `gorm:"index;not null" json:"template_id"`
	StageName   string    `gorm:"size:128;not null" json:"stage_name"`
	StageOrder  int       `gorm:"not null" json:"stage_order"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	AutoTask    bool      `gorm:"default:false" json:"auto_task"` // Automatically creates a task upon entering stage
	CreatedAt   time.Time `json:"created_at"`
}

// JourneyInstance represents an active tracked business journey for a customer.
type JourneyInstance struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	OrganizationID uint           `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID uint           `gorm:"index;not null" json:"business_unit_id"`
	TemplateID     uint           `gorm:"index;not null" json:"template_id"`
	CustomerID     uint           `gorm:"index;not null" json:"customer_id"`
	LeadID         *uint          `gorm:"index" json:"lead_id,omitempty"`
	CurrentStageID uint           `gorm:"index;not null" json:"current_stage_id"`
	CurrentStage   *JourneyStage  `gorm:"foreignKey:CurrentStageID" json:"current_stage,omitempty"`
	Status         string         `gorm:"size:32;default:'ACTIVE';index" json:"status"` // ACTIVE | COMPLETED | CANCELLED
	Logs           []JourneyLog   `gorm:"foreignKey:InstanceID;constraint:OnDelete:CASCADE" json:"logs,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// JourneyLog records stage transition history.
type JourneyLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	InstanceID uint      `gorm:"index;not null" json:"instance_id"`
	FromStage  string    `gorm:"size:128" json:"from_stage"`
	ToStage    string    `gorm:"size:128;not null" json:"to_stage"`
	ActorID    uint      `gorm:"index;not null" json:"actor_id"`
	Notes      string    `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// AutoMigrate migrates journey models.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&JourneyTemplate{}, &JourneyStage{}, &JourneyInstance{}, &JourneyLog{})
}

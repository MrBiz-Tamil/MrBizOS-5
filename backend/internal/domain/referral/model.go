package referral

import (
	"time"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&ReferralCode{},
		&ReferralRecord{},
	)
}

type ReferralCode struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Code       string    `gorm:"uniqueIndex;not null" json:"code"`
	UserID     uint      `gorm:"index;not null" json:"user_id"`
	UsageCount int       `gorm:"default:0" json:"usage_count"`
	CreatedAt  time.Time `json:"created_at"`
}

func (ReferralCode) TableName() string {
	return "referral_codes"
}

type ReferralRecord struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	CodeID        uint      `gorm:"index" json:"code_id"`
	ReferralCode  ReferralCode `gorm:"foreignKey:CodeID" json:"referral_code,omitempty"`
	RefereeUserID uint      `gorm:"index" json:"referee_user_id"`
	RewardAmount  int64     `json:"reward_amount"`
	Status        string    `gorm:"default:'pending'" json:"status"` // pending, paid
	CreatedAt     time.Time `json:"created_at"`
}

func (ReferralRecord) TableName() string {
	return "referral_records"
}

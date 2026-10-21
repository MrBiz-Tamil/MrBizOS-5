package loyalty

import (
	"time"

	"gorm.io/gorm"
)

// PointsLedger records every points earn/spend transaction.
type PointsLedger struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	UserID      uint           `gorm:"index;not null" json:"user_id"`
	Points      int64          `json:"points"`      // positive = earn, negative = spend
	Balance     int64          `json:"balance"`     // running balance after this txn
	RefType     string         `gorm:"size:50" json:"ref_type"` // consultation, medicine, referral, subscription, redemption
	RefID       uint           `json:"ref_id,omitempty"`
	Description string         `gorm:"size:300" json:"description"`
	ExpiresAt   *time.Time     `json:"expires_at,omitempty"`
}

func (PointsLedger) TableName() string { return "points_ledger" }

// RewardRule defines how many points are awarded per rupee spent in each category.
type RewardRule struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	RefType       string         `gorm:"size:50;uniqueIndex" json:"ref_type"` // consultation, medicine, etc.
	PointsPerRupee float64       `json:"points_per_rupee"`
	MaxPointsPerTxn int64        `json:"max_points_per_txn"` // 0 = unlimited
	IsActive      bool           `gorm:"default:true" json:"is_active"`
}

// ReferralCode stores a user's personal referral code and usage stats.
type ReferralCode struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UserID       uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	Code         string    `gorm:"size:20;uniqueIndex;not null" json:"code"`
	TotalUses    int       `json:"total_uses"`
	TotalEarned  int64     `json:"total_earned"` // lifetime points earned via referral
}

// ReferralUse records a successful referral activation.
type ReferralUse struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	ReferrerID   uint      `gorm:"index;not null" json:"referrer_id"`
	RefereeID    uint      `gorm:"uniqueIndex;not null" json:"referee_id"` // one-time only
	Code         string    `gorm:"size:20" json:"code"`
	PointsGiven  int64     `json:"points_given"`
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&PointsLedger{}, &RewardRule{}, &ReferralCode{}, &ReferralUse{})
}

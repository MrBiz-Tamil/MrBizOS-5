package subscription

import (
	"time"

	"gorm.io/gorm"
)

type SubscriptionPlan struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	Tier           string         `gorm:"uniqueIndex;size:50" json:"tier"`
	Name           string         `gorm:"size:100" json:"name"`
	PriceMonthly   float64        `json:"price_monthly"`
	Features       string         `gorm:"type:text" json:"features"` // JSON string
	RazorpayPlanID string         `gorm:"size:100" json:"razorpay_plan_id"`
}

type UserSubscription struct {
	ID                     uint           `gorm:"primarykey" json:"id"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
	DeletedAt              gorm.DeletedAt `gorm:"index" json:"-"`
	UserID                 uint           `gorm:"index" json:"user_id"`
	PlanID                 uint           `gorm:"index" json:"plan_id"`
	Plan                   SubscriptionPlan `gorm:"foreignKey:PlanID" json:"plan"`
	Status                 string         `gorm:"size:50" json:"status"` // active, cancelled, past_due
	RazorpaySubscriptionID string         `gorm:"size:100;uniqueIndex" json:"razorpay_subscription_id"`
	CurrentPeriodStart     time.Time      `json:"current_period_start"`
	CurrentPeriodEnd       time.Time      `json:"current_period_end"`
	CancelAtCycleEnd       bool           `json:"cancel_at_cycle_end"`
}

// SubscriptionInvoice records every billing cycle payment attempt.
type SubscriptionInvoice struct {
	ID                uint       `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	UserID            uint       `gorm:"index;not null" json:"user_id"`
	SubscriptionID    uint       `gorm:"index;not null" json:"subscription_id"`
	PlanID            uint       `json:"plan_id"`
	AmountPaise       int64      `json:"amount_paise"` // in paise
	Status            string     `gorm:"default:'pending'" json:"status"` // paid, failed, pending
	RazorpayPaymentID string     `json:"razorpay_payment_id,omitempty"`
	PeriodStart       time.Time  `json:"period_start"`
	PeriodEnd         time.Time  `json:"period_end"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
}

func (SubscriptionInvoice) TableName() string { return "subscription_invoices" }

// SubscriptionEvent is an immutable audit log of subscription lifecycle changes.
type SubscriptionEvent struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	SubscriptionID uint      `gorm:"index;not null" json:"subscription_id"`
	UserID         uint      `gorm:"index" json:"user_id"`
	EventType      string    `json:"event_type"` // activated, cancelled, payment_failed, payment_success, upgraded, downgraded, paused, resumed
	Metadata       string    `gorm:"type:text" json:"metadata,omitempty"` // JSON
	CreatedAt      time.Time `json:"created_at"`
}

func (SubscriptionEvent) TableName() string { return "subscription_events" }

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&SubscriptionPlan{}, &UserSubscription{}, &SubscriptionInvoice{}, &SubscriptionEvent{})
}

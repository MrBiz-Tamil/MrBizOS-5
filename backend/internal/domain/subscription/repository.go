package subscription

import (
	"time"

	"drayya-core/internal/database"
)

func CreatePlanRecord(plan *SubscriptionPlan) error {
	return database.DB.Create(plan).Error
}

func GetPlanByTier(tier string) (*SubscriptionPlan, error) {
	var plan SubscriptionPlan
	if err := database.DB.Where("tier = ?", tier).First(&plan).Error; err != nil {
		return nil, err
	}
	return &plan, nil
}

func GetAllPlans() ([]SubscriptionPlan, error) {
	var plans []SubscriptionPlan
	if err := database.DB.Find(&plans).Error; err != nil {
		return nil, err
	}
	return plans, nil
}

func CreateUserSubscription(sub *UserSubscription) error {
	return database.DB.Create(sub).Error
}

func GetUserSubscriptionByRazorpayID(rzpSubID string) (*UserSubscription, error) {
	var sub UserSubscription
	if err := database.DB.Where("razorpay_subscription_id = ?", rzpSubID).First(&sub).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

func UpdateUserSubscription(sub *UserSubscription) error {
	return database.DB.Save(sub).Error
}

func GetActiveSubscriptionForUser(userID uint) (*UserSubscription, error) {
	var sub UserSubscription
	if err := database.DB.Where("user_id = ? AND status IN ?", userID, []string{"active", "trialing"}).
		Preload("Plan").Order("created_at DESC").First(&sub).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

func GetSubscriptionByID(id uint) (*UserSubscription, error) {
	var sub UserSubscription
	if err := database.DB.Preload("Plan").First(&sub, id).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

func GetUserSubscriptionHistory(userID uint) ([]UserSubscription, error) {
	var subs []UserSubscription
	err := database.DB.Where("user_id = ?", userID).
		Preload("Plan").Order("created_at DESC").Find(&subs).Error
	return subs, err
}

// Invoice helpers
func CreateInvoice(inv *SubscriptionInvoice) error {
	return database.DB.Create(inv).Error
}

func GetInvoicesByUserID(userID uint, limit int) ([]SubscriptionInvoice, error) {
	var invoices []SubscriptionInvoice
	q := database.DB.Where("user_id = ?", userID).Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	return invoices, q.Find(&invoices).Error
}

func UpdateInvoice(inv *SubscriptionInvoice) error {
	return database.DB.Save(inv).Error
}

// Event helpers
func LogSubscriptionEvent(e *SubscriptionEvent) error {
	return database.DB.Create(e).Error
}

func CountPaymentFailureEvents(subscriptionID uint) (int64, error) {
	var count int64
	err := database.DB.Model(&SubscriptionEvent{}).
		Where("subscription_id = ? AND event_type = ?", subscriptionID, "payment_failed").
		Count(&count).Error
	return count, err
}

// Admin helpers
func GetAllActiveSubscriptions() ([]UserSubscription, error) {
	var subs []UserSubscription
	err := database.DB.Where("status IN ?", []string{"active", "trialing"}).
		Preload("Plan").Find(&subs).Error
	return subs, err
}

func GetExpiringSubscriptions(before time.Time) ([]UserSubscription, error) {
	var subs []UserSubscription
	err := database.DB.Where("status = 'active' AND current_period_end < ?", before).
		Preload("Plan").Find(&subs).Error
	return subs, err
}

func CountSubscriptionsByStatus() (map[string]int64, error) {
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	err := database.DB.Model(&UserSubscription{}).
		Select("status, count(*) as count").
		Group("status").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64)
	for _, r := range rows {
		result[r.Status] = r.Count
	}
	return result, nil
}

func GetMonthlyRevenuePaise() (int64, error) {
	var total int64
	now := time.Now()
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	err := database.DB.Model(&SubscriptionInvoice{}).
		Where("status = 'paid' AND paid_at >= ?", firstOfMonth).
		Select("COALESCE(SUM(amount_paise),0)").Scan(&total).Error
	return total, err
}

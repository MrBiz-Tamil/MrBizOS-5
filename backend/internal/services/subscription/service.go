package subscription

import (
	"errors"
	"fmt"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"
	"time"
)

type Plan struct {
	Tier         string   `json:"tier"`
	Name         string   `json:"name"`
	PriceMonthly float64  `json:"price_monthly"`
	Features     []string `json:"features"`
}

type PlanConfig struct {
	Price    float64
	Benefits map[string]interface{}
}

var TierConfig = map[string]PlanConfig{
	"free": {
		Price:    0,
		Benefits: map[string]interface{}{"priority_boost": 0, "free_consults": 0, "discount": 0},
	},
	"silver": {
		Price:    499,
		Benefits: map[string]interface{}{"priority_boost": 10, "free_consults": 1, "discount": 5},
	},
	"gold": {
		Price:    999,
		Benefits: map[string]interface{}{"priority_boost": 20, "free_consults": 3, "discount": 10},
	},
	"platinum": {
		Price:    1999,
		Benefits: map[string]interface{}{"priority_boost": 50, "free_consults": 5, "discount": 15},
	},
	"prime": {
		Price:    14999,
		Benefits: map[string]interface{}{"priority_boost": 100, "free_consults": -1, "discount": 20}, // -1 for unlimited
	},
	"enterprise": {
		Price:    10000,
		Benefits: map[string]interface{}{"priority_boost": 0, "free_consults": 0, "discount": 0, "is_saas": true},
	},
}

var AvailablePlans = []Plan{
	{
		Tier:         "free",
		Name:         "Basic",
		PriceMonthly: 0,
		Features: []string{
			"Access to top doctors",
			"Standard response time",
			"Digital prescriptions",
		},
	},
	{
		Tier:         "silver",
		Name:         "Silver Pro",
		PriceMonthly: 499,
		Features: []string{
			"Priority booking",
			"1 Free consultation/month",
			"5% Off on medicines",
			"Basic health tracking",
		},
	},
	{
		Tier:         "gold",
		Name:         "Gold Elite",
		PriceMonthly: 999,
		Features: []string{
			"24/7 Priority access",
			"3 Free consultations/month",
			"10% Off on medicines",
			"Advanced health tracking",
			"Personal health assistant",
		},
	},
	{
		Tier:         "platinum",
		Name:         "Platinum Omni",
		PriceMonthly: 1999,
		Features: []string{
			"Instant access to specialists",
			"5 Free consultations/month",
			"15% Off on medicines",
			"Premium health analytics",
			"Priority home sample collection",
			"Exclusive wellness webinars",
		},
	},
	{
		Tier:         "prime",
		Name:         "Dr.Ayya Prime",
		PriceMonthly: 1250, // Approx 14999/12
		Features: []string{
			"Unlimited consultations",
			"Lifetime EMR storage",
			"20% Off on medicines",
			"Annual health checkup",
		},
	},
	{
		Tier:         "enterprise",
		Name:         "Dr.Ayya Enterprise",
		PriceMonthly: 10000,
		Features: []string{
			"White-label EMR",
			"Multi-doctor scheduling",
			"Advanced ledger & analytics",
			"24/7 Enterprise support",
		},
	},
}

// UpgradeSubscriptionWithPayment deducts the tier price from the user's wallet
// then upgrades the tier. Returns the updated user or an error.
func UpgradeSubscriptionWithPayment(userID uint, targetTier string) (*identity.User, error) {
	cfg, valid := TierConfig[targetTier]
	if !valid {
		return nil, errors.New("invalid tier")
	}

	db := database.DB

	// Deduct from wallet when the tier has a price (free tier skips deduction)
	if cfg.Price > 0 {
		amountPaise := int64(cfg.Price * 100)
		accountName := fmt.Sprintf("PatientWallet:%d", userID)

		var account finance.LedgerAccount
		if err := db.Where("name = ?", accountName).First(&account).Error; err != nil {
			return nil, errors.New("wallet not found — please top up your wallet first")
		}
		if account.Balance < amountPaise {
			return nil, fmt.Errorf("insufficient wallet balance (need ₹%.0f, have ₹%.2f)",
				cfg.Price, float64(account.Balance)/100.0)
		}
		if err := db.Model(&account).Update("balance", account.Balance-amountPaise).Error; err != nil {
			return nil, err
		}
	}

	return UpgradeSubscription(userID, targetTier)
}

// UpgradeSubscription upgrades a user's tier
func UpgradeSubscription(userID uint, targetTier string) (*identity.User, error) {
	db := database.DB

	_, valid := TierConfig[targetTier]
	if !valid {
		return nil, errors.New("invalid tier")
	}

	var user identity.User
	if err := db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	if user.Tier == targetTier {
		return nil, errors.New("already on this tier")
	}

	now := time.Now()
	expiry := now.AddDate(0, 1, 0) // 1 month

	user.Tier = targetTier
	user.SubscriptionEndDate = &expiry
	user.SubscriptionAutoRenew = true

	if err := db.Save(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func GetPlans() []Plan {
	return AvailablePlans
}

// CancelSubscription disables auto-renewal so the user keeps benefits until expiry
// but will not be charged again.
func CancelSubscription(userID uint) error {
	var user identity.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return err
	}
	if user.Tier == "free" {
		return errors.New("no active subscription to cancel")
	}
	user.SubscriptionAutoRenew = false
	return database.DB.Save(&user).Error
}

// GetSubscriptionStatus returns the current subscription for a user
func GetSubscriptionStatus(userID uint) (map[string]interface{}, error) {
	var user identity.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"tier":       user.Tier,
		"status":     "active",
		"end_date":   user.SubscriptionEndDate,
		"auto_renew": user.SubscriptionAutoRenew,
		"plan_name":  user.Tier,
	}, nil
}

package subscription

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/services/razorpay"
)

// postSubscriptionRevenue records a subscription charge into the central ledger.
// Subscriptions are 100% platform (SaaS) revenue with GST extracted inclusive at
// 18%. Idempotent on the Razorpay payment id, so duplicate webhook deliveries
// post only once. Money received into Asset:Razorpay; nothing else in the system
// records subscription money, so this only adds visibility.
func postSubscriptionRevenue(rzpPaymentID string, rzpSubID string, amountPaise int64, userID uint) {
	if amountPaise <= 0 {
		return
	}
	ref := rzpPaymentID
	if ref == "" {
		ref = fmt.Sprintf("SUB-%s-%d", rzpSubID, time.Now().Unix())
	}

	gst := amountPaise * 18 / 118 // GST-inclusive extraction
	netRevenue := amountPaise - gst

	finSvc := finance.NewService(finance.NewRepository(database.DB))
	if _, err := finSvc.PostRevenue(context.Background(), ref, "SUBSCRIPTION",
		fmt.Sprintf("Subscription charge for user %d", userID),
		finance.RevenueInput{
			FundingAccount: finance.AcctRazorpay,
			FundingType:    finance.TypeAsset,
			Total:          amountPaise,
			Credits: []finance.RevenueLeg{
				{Account: finance.RevSubscriptions, Type: finance.TypeRevenue, Amount: netRevenue},
				{Account: finance.LiabGST, Type: finance.TypeLiability, Amount: gst},
			},
		}); err != nil {
		log.Printf("[Subscription] CRITICAL: charge %s for user %d not posted to ledger: %v", ref, userID, err)
	}
}

// InitializePlans can be called during app startup to ensure base plans exist
// and are synced with Razorpay
func InitializePlans() error {
	plans, _ := GetAllPlans()
	if len(plans) == 0 {
		// Create base prime plan
		rzpPlanID, err := razorpay.CreatePlan(razorpay.PlanInput{
			Period:   "monthly",
			Interval: 1,
			Item: struct {
				Name        string
				Amount      int64
				Currency    string
				Description string
			}{
				Name:        "Dr.Ayya Prime",
				Amount:      1250 * 100, // INR 1250 in paise
				Currency:    "INR",
				Description: "Monthly subscription for Dr.Ayya Prime",
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create Razorpay plan: %v", err)
		}

		primePlan := &SubscriptionPlan{
			Tier:           "prime",
			Name:           "Dr.Ayya Prime",
			PriceMonthly:   1250,
			Features:       `["Unlimited consultations", "Lifetime EMR storage", "20% Off on medicines", "Annual health checkup"]`,
			RazorpayPlanID: rzpPlanID,
		}
		return CreatePlanRecord(primePlan)
	}
	return nil
}

// CreateCheckout creates a subscription via Razorpay for a user
func CreateCheckout(userID uint, tier string) (string, error) {
	plan, err := GetPlanByTier(tier)
	if err != nil {
		return "", errors.New("plan not found")
	}

	rzpSub, err := razorpay.CreateSubscription(razorpay.SubscriptionInput{
		PlanID:         plan.RazorpayPlanID,
		TotalCount:     120, // 10 years
		CustomerNotify: 1,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create razorpay subscription: %v", err)
	}

	subID, ok := rzpSub["id"].(string)
	if !ok {
		return "", errors.New("invalid response from Razorpay")
	}

	// Create pending subscription record
	userSub := &UserSubscription{
		UserID:                 userID,
		PlanID:                 plan.ID,
		Status:                 "pending",
		RazorpaySubscriptionID: subID,
	}

	if err := CreateUserSubscription(userSub); err != nil {
		return "", err
	}

	return subID, nil
}

// HandleWebhook updates subscription status based on Razorpay events
func HandleWebhook(payload map[string]interface{}, event string) error {
	if event != "subscription.charged" && event != "subscription.activated" && event != "subscription.halted" {
		return nil // Ignore other events
	}

	payloadData, ok := payload["payload"].(map[string]interface{})
	if !ok {
		return errors.New("invalid payload structure")
	}

	subData, ok := payloadData["subscription"].(map[string]interface{})
	if !ok {
		return errors.New("missing subscription in payload")
	}

	entity, ok := subData["entity"].(map[string]interface{})
	if !ok {
		return errors.New("missing entity in subscription")
	}

	rzpSubID, _ := entity["id"].(string)
	status, _ := entity["status"].(string)

	userSub, err := GetUserSubscriptionByRazorpayID(rzpSubID)
	if err != nil {
		return err // Could not find matching sub
	}

	// Update dates if available
	if currentStart, ok := entity["current_start"].(float64); ok {
		userSub.CurrentPeriodStart = time.Unix(int64(currentStart), 0)
	}
	if currentEnd, ok := entity["current_end"].(float64); ok {
		userSub.CurrentPeriodEnd = time.Unix(int64(currentEnd), 0)
	}

	userSub.Status = status
	if err := UpdateUserSubscription(userSub); err != nil {
		return err
	}

	// Record the money into the central ledger on an actual charge. The
	// subscription.charged payload carries a payment entity with the authoritative
	// amount (paise) and a unique id we use as the idempotency key.
	if event == "subscription.charged" {
		var amountPaise int64
		var rzpPaymentID string
		if paymentData, ok := payloadData["payment"].(map[string]interface{}); ok {
			if entity, ok := paymentData["entity"].(map[string]interface{}); ok {
				if amt, ok := entity["amount"].(float64); ok {
					amountPaise = int64(amt)
				}
				rzpPaymentID, _ = entity["id"].(string)
			}
		}
		if amountPaise == 0 {
			// Fallback to the plan price if the payload lacked a payment entity.
			amountPaise = int64(userSub.Plan.PriceMonthly * 100)
		}
		postSubscriptionRevenue(rzpPaymentID, rzpSubID, amountPaise, userSub.UserID)
	}

	// Sync with Identity User Model
	var user identity.User
	if err := database.DB.First(&user, userSub.UserID).Error; err == nil {
		if status == "active" {
			user.Tier = userSub.Plan.Tier
			user.SubscriptionEndDate = &userSub.CurrentPeriodEnd
			user.SubscriptionAutoRenew = true
		} else if status == "halted" || status == "cancelled" {
			user.Tier = "free"
			user.SubscriptionEndDate = nil
			user.SubscriptionAutoRenew = false
		}
		database.DB.Save(&user)
	}

	return nil
}

// CancelSubscription marks subscription cancelled immediately.
func CancelSubscription(userID uint) error {
	sub, err := GetActiveSubscriptionForUser(userID)
	if err != nil {
		return errors.New("no active subscription found")
	}
	sub.CancelAtCycleEnd = true
	sub.Status = "cancelled"
	if err := UpdateUserSubscription(sub); err != nil {
		return err
	}
	_ = LogSubscriptionEvent(&SubscriptionEvent{
		SubscriptionID: sub.ID,
		UserID:         userID,
		EventType:      "cancelled",
	})
	database.DB.Exec("UPDATE users SET tier = 'free' WHERE id = ?", userID)
	return nil
}

// PauseSubscription puts a subscription on hold.
func PauseSubscription(userID uint) error {
	sub, err := GetActiveSubscriptionForUser(userID)
	if err != nil {
		return errors.New("no active subscription found")
	}
	sub.Status = "paused"
	if err := UpdateUserSubscription(sub); err != nil {
		return err
	}
	_ = LogSubscriptionEvent(&SubscriptionEvent{
		SubscriptionID: sub.ID,
		UserID:         userID,
		EventType:      "paused",
	})
	return nil
}

// ResumeSubscription reactivates a paused subscription.
func ResumeSubscription(userID uint) error {
	var sub UserSubscription
	if err := database.DB.Where("user_id = ? AND status = 'paused'", userID).
		Preload("Plan").First(&sub).Error; err != nil {
		return errors.New("no paused subscription found")
	}
	now := time.Now()
	sub.Status = "active"
	sub.CurrentPeriodStart = now
	sub.CurrentPeriodEnd = now.AddDate(0, 1, 0)
	if err := UpdateUserSubscription(&sub); err != nil {
		return err
	}
	_ = LogSubscriptionEvent(&SubscriptionEvent{
		SubscriptionID: sub.ID,
		UserID:         userID,
		EventType:      "resumed",
	})
	return nil
}

// UpgradePlan switches user to a new plan immediately.
func UpgradePlan(userID uint, newTier string) (*UserSubscription, error) {
	newPlan, err := GetPlanByTier(newTier)
	if err != nil {
		return nil, errors.New("plan not found: " + newTier)
	}
	if old, err := GetActiveSubscriptionForUser(userID); err == nil {
		old.Status = "expired"
		_ = UpdateUserSubscription(old)
		_ = LogSubscriptionEvent(&SubscriptionEvent{
			SubscriptionID: old.ID,
			UserID:         userID,
			EventType:      "upgraded",
			Metadata:       fmt.Sprintf(`{"from":"%s","to":"%s"}`, old.Plan.Tier, newTier),
		})
	}
	now := time.Now()
	newSub := &UserSubscription{
		UserID:             userID,
		PlanID:             newPlan.ID,
		Status:             "active",
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	if err := CreateUserSubscription(newSub); err != nil {
		return nil, err
	}
	database.DB.Exec("UPDATE users SET tier = ?, subscription_end_date = ? WHERE id = ?",
		newPlan.Tier, newSub.CurrentPeriodEnd, userID)
	return newSub, nil
}

// HasActiveSubscription returns true if user has an active or trialing subscription.
func HasActiveSubscription(userID uint) bool {
	_, err := GetActiveSubscriptionForUser(userID)
	return err == nil
}

// GetAdminSubscriptionOverview returns aggregate stats for the admin dashboard.
func GetAdminSubscriptionOverview() (map[string]interface{}, error) {
	byStatus, err := CountSubscriptionsByStatus()
	if err != nil {
		return nil, err
	}
	mrr, _ := GetMonthlyRevenuePaise()
	return map[string]interface{}{
		"by_status":  byStatus,
		"mrr_paise":  mrr,
		"mrr_rupees": float64(mrr) / 100.0,
	}, nil
}


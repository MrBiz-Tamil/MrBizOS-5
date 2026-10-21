package loyalty

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
)

const (
	referralBonusPoints   int64 = 200  // points to referrer
	refereeWelcomePoints  int64 = 100  // points to new user
	redemptionRateRupees  float64 = 0.25 // 1 point = ₹0.25
	pointsValidityMonths  int   = 12
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// SeedDefaultRules creates starter reward rules if none exist.
func SeedDefaultRules() error {
	existing, _ := GetAllRewardRules()
	if len(existing) > 0 {
		return nil
	}
	defaults := []RewardRule{
		{RefType: "consultation", PointsPerRupee: 2.0, IsActive: true},
		{RefType: "medicine", PointsPerRupee: 1.0, IsActive: true},
		{RefType: "lab", PointsPerRupee: 1.5, IsActive: true},
		{RefType: "subscription", PointsPerRupee: 3.0, IsActive: true},
		{RefType: "referral", PointsPerRupee: 0, IsActive: true}, // fixed bonus
	}
	for i := range defaults {
		if err := UpsertRewardRule(&defaults[i]); err != nil {
			return err
		}
	}
	return nil
}

// EarnPoints awards points to a user for a transaction.
func EarnPoints(userID uint, amountPaise int64, refType string, refID uint) (*PointsLedger, error) {
	rule, err := GetRuleByRefType(refType)
	if err != nil {
		return nil, nil // no rule = no points, not an error
	}

	amountRupees := float64(amountPaise) / 100.0
	pts := int64(amountRupees * rule.PointsPerRupee)
	if rule.MaxPointsPerTxn > 0 && pts > rule.MaxPointsPerTxn {
		pts = rule.MaxPointsPerTxn
	}
	if pts <= 0 {
		return nil, nil
	}

	balance, _ := GetBalance(userID)
	expiry := time.Now().AddDate(0, pointsValidityMonths, 0)
	entry := &PointsLedger{
		UserID:      userID,
		Points:      pts,
		Balance:     balance + pts,
		RefType:     refType,
		RefID:       refID,
		Description: fmt.Sprintf("Earned %d pts for %s", pts, refType),
		ExpiresAt:   &expiry,
	}
	if err := AddEntry(entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// RedeemPoints deducts points from a user's balance.
func RedeemPoints(userID uint, pointsToRedeem int64, description string) (*PointsLedger, error) {
	if pointsToRedeem <= 0 {
		return nil, errors.New("points must be positive")
	}
	balance, err := GetBalance(userID)
	if err != nil {
		return nil, err
	}
	if balance < pointsToRedeem {
		return nil, fmt.Errorf("insufficient points: have %d, need %d", balance, pointsToRedeem)
	}

	entry := &PointsLedger{
		UserID:      userID,
		Points:      -pointsToRedeem,
		Balance:     balance - pointsToRedeem,
		RefType:     "redemption",
		Description: description,
	}
	if err := AddEntry(entry); err != nil {
		return nil, err
	}

	// Convert the redeemed points to spendable wallet money in the central
	// ledger: the cost is a marketing expense, the credit is the user's wallet
	// (a liability we now owe them). Idempotent on the points-ledger entry id.
	valuePaise := int64(math.Round(PointsValueRupees(pointsToRedeem) * 100))
	if valuePaise > 0 {
		postRedemptionToWallet(userID, entry.ID, valuePaise)
	}

	return entry, nil
}

// postRedemptionToWallet records the money leg of a loyalty redemption:
// Debit Marketing:Rewards (expense), Credit PatientWallet:{userID} (liability).
// On failure it logs CRITICAL — the reconciliation/orphan checks will surface
// any points redeemed without a matching wallet credit.
func postRedemptionToWallet(userID uint, ledgerID uint, valuePaise int64) {
	finSvc := finance.NewService(finance.NewRepository(database.DB))
	walletAcct, err := finSvc.GetOrCreateAccount(context.Background(), finance.PatientWallet(userID), finance.TypeLiability)
	if err != nil {
		log.Printf("[Loyalty] CRITICAL: redemption %d wallet account error: %v", ledgerID, err)
		return
	}
	rewardsAcct, err := finSvc.GetOrCreateAccount(context.Background(), finance.ExpMarketingRewards, finance.TypeExpense)
	if err != nil {
		log.Printf("[Loyalty] CRITICAL: redemption %d rewards account error: %v", ledgerID, err)
		return
	}
	if _, err := finSvc.RecordTransaction(context.Background(),
		fmt.Sprintf("LOY-%d", ledgerID), "LOYALTY_REDEMPTION",
		fmt.Sprintf("Loyalty redemption credited to wallet for user %d", userID),
		[]finance.EntryData{
			{AccountID: rewardsAcct.ID, Direction: finance.DirectionDebit, Amount: valuePaise},
			{AccountID: walletAcct.ID, Direction: finance.DirectionCredit, Amount: valuePaise},
		}); err != nil {
		log.Printf("[Loyalty] CRITICAL: redemption %d ledger post failed: %v", ledgerID, err)
	}
}

// PointsValueRupees converts points to their INR equivalent.
func PointsValueRupees(points int64) float64 {
	return float64(points) * redemptionRateRupees
}

// GetOrCreateReferralCode returns the user's referral code, creating one if needed.
func GetOrCreateReferralCode(userID uint) (*ReferralCode, error) {
	rc, err := GetReferralCode(userID)
	if err == nil {
		return rc, nil
	}
	// Generate unique 8-char code
	code := generateCode(userID)
	rc = &ReferralCode{
		UserID: userID,
		Code:   code,
	}
	if err := CreateReferralCode(rc); err != nil {
		return nil, err
	}
	return rc, nil
}

// ApplyReferralCode credits both referrer and referee.
func ApplyReferralCode(refereeID uint, code string) error {
	if HasUsedReferral(refereeID) {
		return errors.New("referral code already used")
	}
	rc, err := GetReferralCodeByCode(code)
	if err != nil {
		return errors.New("invalid referral code")
	}
	if rc.UserID == refereeID {
		return errors.New("cannot use own referral code")
	}

	// Credit referrer
	refBalance, _ := GetBalance(rc.UserID)
	expiry := time.Now().AddDate(0, pointsValidityMonths, 0)
	_ = AddEntry(&PointsLedger{
		UserID:      rc.UserID,
		Points:      referralBonusPoints,
		Balance:     refBalance + referralBonusPoints,
		RefType:     "referral",
		RefID:       refereeID,
		Description: fmt.Sprintf("Referral bonus for inviting user %d", refereeID),
		ExpiresAt:   &expiry,
	})

	// Credit referee (welcome bonus)
	reeBalance, _ := GetBalance(refereeID)
	_ = AddEntry(&PointsLedger{
		UserID:      refereeID,
		Points:      refereeWelcomePoints,
		Balance:     reeBalance + refereeWelcomePoints,
		RefType:     "referral",
		RefID:       rc.UserID,
		Description: "Welcome bonus for joining via referral",
		ExpiresAt:   &expiry,
	})

	// Update referral code usage
	rc.TotalUses++
	rc.TotalEarned += referralBonusPoints
	_ = SaveReferralCode(rc)

	// Log the use
	_ = CreateReferralUse(&ReferralUse{
		ReferrerID:  rc.UserID,
		RefereeID:   refereeID,
		Code:        code,
		PointsGiven: referralBonusPoints,
	})

	return nil
}

// generateCode creates an 8-character alphanumeric referral code seeded with userID.
func generateCode(userID uint) string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	prefix := fmt.Sprintf("MH%d", userID%100)
	var b strings.Builder
	b.WriteString(prefix)
	for b.Len() < 8 {
		b.WriteByte(charset[rand.Intn(len(charset))])
	}
	return b.String()[:8]
}

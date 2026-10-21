package loyalty

import (
	"time"

	"drayya-core/internal/database"
)

// ── Points Ledger ─────────────────────────────────────────────────────────────

func GetBalance(userID uint) (int64, error) {
	var balance int64
	err := database.DB.Model(&PointsLedger{}).
		Select("COALESCE(SUM(points), 0)").
		Where("user_id = ? AND (expires_at IS NULL OR expires_at > ?)", userID, time.Now()).
		Scan(&balance).Error
	return balance, err
}

func AddEntry(entry *PointsLedger) error {
	return database.DB.Create(entry).Error
}

func GetLedger(userID uint, limit int) ([]PointsLedger, error) {
	var entries []PointsLedger
	q := database.DB.Where("user_id = ?", userID).Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	return entries, q.Find(&entries).Error
}

// ── Reward Rules ──────────────────────────────────────────────────────────────

func GetRuleByRefType(refType string) (*RewardRule, error) {
	var rule RewardRule
	err := database.DB.Where("ref_type = ? AND is_active = true", refType).First(&rule).Error
	return &rule, err
}

func UpsertRewardRule(rule *RewardRule) error {
	var existing RewardRule
	err := database.DB.Where("ref_type = ?", rule.RefType).First(&existing).Error
	if err != nil {
		return database.DB.Create(rule).Error
	}
	rule.ID = existing.ID
	return database.DB.Save(rule).Error
}

func GetAllRewardRules() ([]RewardRule, error) {
	var rules []RewardRule
	return rules, database.DB.Find(&rules).Error
}

// ── Referral ──────────────────────────────────────────────────────────────────

func GetReferralCode(userID uint) (*ReferralCode, error) {
	var rc ReferralCode
	err := database.DB.Where("user_id = ?", userID).First(&rc).Error
	return &rc, err
}

func CreateReferralCode(rc *ReferralCode) error {
	return database.DB.Create(rc).Error
}

func GetReferralCodeByCode(code string) (*ReferralCode, error) {
	var rc ReferralCode
	err := database.DB.Where("code = ?", code).First(&rc).Error
	return &rc, err
}

func SaveReferralCode(rc *ReferralCode) error {
	return database.DB.Save(rc).Error
}

func HasUsedReferral(refereeID uint) bool {
	var count int64
	database.DB.Model(&ReferralUse{}).Where("referee_id = ?", refereeID).Count(&count)
	return count > 0
}

func CreateReferralUse(use *ReferralUse) error {
	return database.DB.Create(use).Error
}

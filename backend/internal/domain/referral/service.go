package referral

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"gorm.io/gorm"
)

type Service interface {
	GetMyCode(userID uint) (string, error)
	GenerateNewCode(userID uint) (string, error)
	GetMyReferrals(userID uint) ([]ReferralRecord, error)
	GetStats(userID uint) (map[string]interface{}, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetMyCode(userID uint) (string, error) {
	code, err := s.repo.GetCodeByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.GenerateNewCode(userID)
		}
		return "", err
	}
	return code.Code, nil
}

func (s *service) GenerateNewCode(userID uint) (string, error) {
	// Simple random code
	rand.Seed(time.Now().UnixNano())
	codeStr := fmt.Sprintf("MEDI-%d-%X", userID, rand.Intn(1000))
	
	code := &ReferralCode{
		UserID: userID,
		Code:   codeStr,
	}
	err := s.repo.CreateCode(code)
	return codeStr, err
}

func (s *service) GetMyReferrals(userID uint) ([]ReferralRecord, error) {
	code, err := s.repo.GetCodeByUserID(userID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetRecordsByCodeID(code.ID)
}

func (s *service) GetStats(userID uint) (map[string]interface{}, error) {
	joined, pending, earned, err := s.repo.GetStats(userID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"total_referrals": joined + pending,
		"pending":         pending,
		"total_earned":    float64(earned) / 100,
	}, nil
}

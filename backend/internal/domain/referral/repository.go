package referral

import (
	"gorm.io/gorm"
)

type Repository interface {
	GetCodeByUserID(userID uint) (*ReferralCode, error)
	GetCodeByValue(code string) (*ReferralCode, error)
	CreateCode(code *ReferralCode) error
	CreateRecord(record *ReferralRecord) error
	GetRecordsByCodeID(codeID uint) ([]ReferralRecord, error)
	GetStats(userID uint) (int, int, int64, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetCodeByUserID(userID uint) (*ReferralCode, error) {
	var code ReferralCode
	err := r.db.Where("user_id = ?", userID).First(&code).Error
	return &code, err
}

func (r *repository) GetCodeByValue(codeStr string) (*ReferralCode, error) {
	var code ReferralCode
	err := r.db.Where("code = ?", codeStr).First(&code).Error
	return &code, err
}

func (r *repository) CreateCode(code *ReferralCode) error {
	return r.db.Create(code).Error
}

func (r *repository) CreateRecord(record *ReferralRecord) error {
	return r.db.Create(record).Error
}

func (r *repository) GetRecordsByCodeID(codeID uint) ([]ReferralRecord, error) {
	var records []ReferralRecord
	err := r.db.Where("code_id = ?", codeID).Find(&records).Error
	return records, err
}

func (r *repository) GetStats(userID uint) (int, int, int64, error) {
	var code ReferralCode
	if err := r.db.Where("user_id = ?", userID).First(&code).Error; err != nil {
		return 0, 0, 0, err
	}

	var totalJoined int64
	var pending int64
	var totalEarned int64

	r.db.Model(&ReferralRecord{}).Where("code_id = ? AND status = ?", code.ID, "joined").Count(&totalJoined)
	r.db.Model(&ReferralRecord{}).Where("code_id = ? AND status = ?", code.ID, "pending").Count(&pending)
	r.db.Model(&ReferralRecord{}).Where("code_id = ? AND status = ?", code.ID, "joined").Select("SUM(reward_amount)").Scan(&totalEarned)

	return int(totalJoined), int(pending), totalEarned, nil
}

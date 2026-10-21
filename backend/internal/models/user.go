//go:build ignore
// +build ignore

package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

type UserRole string

const (
	RolePatient            UserRole = "patient"
	RoleDoctor             UserRole = "doctor"
	RoleAdmin              UserRole = "admin"
	RoleNurse              UserRole = "nurse"
	RolePharmacist         UserRole = "pharmacist"
	RoleLabTechnician      UserRole = "lab_technician"
	RoleHospitalHeadDoctor UserRole = "hospital_head_doctor"
	RoleBPO                UserRole = "bpo"
	RoleFinanceManager     UserRole = "finance_manager"
	RoleHRManager          UserRole = "hr_manager"
	RoleQAManager          UserRole = "qa_manager"
	RoleManager            UserRole = "manager"
	RoleStoreManager       UserRole = "store_manager"
	RoleFranchise          UserRole = "franchise_owner"
	RolePharmacyAdmin      UserRole = "pharmacy_admin"
	RolePharmacyManager    UserRole = "pharmacy_manager"
	RoleHospitalAdmin      UserRole = "hospital_admin"
	RoleSuperAdmin         UserRole = "super_admin"
	RoleChiefDoctor        UserRole = "chief_doctor"
)

type User struct {
	ID                        uint           `gorm:"primaryKey" json:"id"`
	Email                     string         `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash              string         `gorm:"not null" json:"-"` // Hide from JSON
	Role                      string         `gorm:"default:'patient'" json:"role"`
	FullName                  string         `json:"full_name"`
	Gender                    string         `json:"gender"`
	PhoneNumber               string         `gorm:"index" json:"phone_number"`
	IsActive                  bool           `gorm:"default:true" json:"is_active"`
	IsVerified                bool           `gorm:"default:false" json:"is_verified"`
	ProfilePicture            string         `json:"profile_picture"`
	Timezone                  string         `gorm:"default:'UTC'" json:"timezone"`
	HospitalID                *uint          `json:"hospital_id"` // Organizational unit context
	Tier                      string         `gorm:"default:'free'" json:"tier"`
	IsVerifiedPremium         bool           `gorm:"default:false" json:"is_verified_premium"`
	PreferredLanguage         string         `gorm:"default:'english'" json:"preferred_language"`
	SecondaryLanguages        string         `json:"secondary_languages"`
	WalletBalance             int64          `gorm:"default:0" json:"wallet_balance"` // In cents (100 = 1.00 currency)
	PrakritiScore             string         `json:"prakriti_score"`
	HasUsedFreeConsult        bool           `gorm:"default:false" json:"has_used_free_consult"` // NEW: One-time use tracking
	MFACode                   string         `json:"-"`                                          // Hidden from JSON
	MFALastSent               *time.Time     `json:"-"`
	SubscriptionStartDate     *time.Time     `json:"subscription_start_date"`
	SubscriptionEndDate       *time.Time     `json:"subscription_end_date"`
	SubscriptionAutoRenew     bool           `gorm:"default:false" json:"subscription_auto_renew"`
	SubscriptionPaymentMethod string         `json:"subscription_payment_method"`
	LastLogin                 *time.Time     `json:"last_login"`
	CreatedAt                 time.Time      `json:"created_at"`
	UpdatedAt                 time.Time      `json:"updated_at"`
	DeletedAt                 gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName overrides the default pluralization to match Python's "user" table
func (User) TableName() string {
	return "user"
}

// ActivityLog Model
type ActivityLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `json:"user_id"`
	User      User      `json:"user" gorm:"foreignKey:UserID"`
	Action    string    `json:"action"`
	Entity    string    `json:"entity"`
	EntityID  string    `json:"entity_id"`
	Details   JSON      `json:"details" gorm:"type:text"` // Changed from jsonb to text for SQLite compatibility
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

// Helper for JSON fields in Gorm
type JSON map[string]interface{}

func (j JSON) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, j)
}

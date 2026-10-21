package devi

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// B2BLicense represents an API license for a B2B customer (hospital/clinic chain)
type B2BLicense struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	OrgName      string    `gorm:"not null" json:"org_name"`
	OrgType      string    `json:"org_type"` // hospital, clinic, lab, insurance, pharmacy_chain
	ContactName  string    `json:"contact_name"`
	ContactEmail string    `gorm:"uniqueIndex;not null" json:"contact_email"`
	APIKey       string    `gorm:"uniqueIndex;not null" json:"api_key"`
	APISecret    string    `gorm:"not null" json:"-"` // never returned in JSON
	Plan         string    `gorm:"default:'starter'" json:"plan"` // starter, growth, enterprise
	MonthlyLimit int       `gorm:"default:1000" json:"monthly_limit"` // API calls/month
	UsedThisMonth int      `gorm:"default:0" json:"used_this_month"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	WebhookURL   string    `json:"webhook_url,omitempty"`
	AllowedIPs   string    `json:"allowed_ips,omitempty"` // comma-separated CIDR blocks
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (B2BLicense) TableName() string {
	return "b2b_licenses"
}

// B2BUsageLog tracks API usage per license
type B2BUsageLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	LicenseID  uint      `gorm:"index;not null" json:"license_id"`
	Endpoint   string    `json:"endpoint"`
	Method     string    `json:"method"`
	StatusCode int       `json:"status_code"`
	LatencyMs  int       `json:"latency_ms"`
	IPAddress  string    `json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`
}

func (B2BUsageLog) TableName() string {
	return "b2b_usage_logs"
}

// PlanLimits defines API call limits per plan
var PlanLimits = map[string]int{
	"starter":    1_000,
	"growth":     50_000,
	"enterprise": 10_000_000, // effectively unlimited
}

// PlanPriceMonthly is in paise
var PlanPriceMonthly = map[string]int64{
	"starter":    99900,    // ₹999/month
	"growth":     499900,   // ₹4,999/month
	"enterprise": 1999900,  // ₹19,999/month
}

// LicensingService handles Devi AI B2B API licensing
type LicensingService struct {
	db *gorm.DB
}

// NewLicensingService creates a new licensing service
func NewLicensingService(db *gorm.DB) *LicensingService {
	return &LicensingService{db: db}
}

// CreateLicense provisions a new B2B API license
func (s *LicensingService) CreateLicense(orgName, orgType, contactName, contactEmail, plan string, webhookURL string) (*B2BLicense, error) {
	// Check plan validity
	limit, ok := PlanLimits[plan]
	if !ok {
		plan = "starter"
		limit = PlanLimits["starter"]
	}

	apiKey, err := generateKey(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	apiSecret, err := generateKey(48)
	if err != nil {
		return nil, fmt.Errorf("failed to generate API secret: %w", err)
	}

	// One year expiry for starter, perpetual for enterprise
	var expiry *time.Time
	if plan == "starter" {
		t := time.Now().AddDate(1, 0, 0)
		expiry = &t
	}

	license := &B2BLicense{
		OrgName:      orgName,
		OrgType:      orgType,
		ContactName:  contactName,
		ContactEmail: contactEmail,
		APIKey:       "devi_" + apiKey,
		APISecret:    apiSecret,
		Plan:         plan,
		MonthlyLimit: limit,
		WebhookURL:   webhookURL,
		ExpiresAt:    expiry,
		IsActive:     true,
	}

	if err := s.db.Create(license).Error; err != nil {
		return nil, fmt.Errorf("failed to create license: %w", err)
	}
	return license, nil
}

// ValidateLicense checks if an API key is valid and within quota
func (s *LicensingService) ValidateLicense(apiKey string) (*B2BLicense, error) {
	var license B2BLicense
	if err := s.db.Where("api_key = ? AND is_active = true", apiKey).First(&license).Error; err != nil {
		return nil, fmt.Errorf("invalid or inactive API key")
	}

	// Check expiry
	if license.ExpiresAt != nil && time.Now().After(*license.ExpiresAt) {
		return nil, fmt.Errorf("API key has expired")
	}

	// Check monthly limit
	if license.UsedThisMonth >= license.MonthlyLimit {
		return nil, fmt.Errorf("monthly API limit reached (%d/%d)", license.UsedThisMonth, license.MonthlyLimit)
	}

	return &license, nil
}

// RecordUsage increments usage counter and logs the API call
func (s *LicensingService) RecordUsage(licenseID uint, endpoint, method, ipAddress string, statusCode, latencyMs int) error {
	// Increment usage
	s.db.Model(&B2BLicense{}).Where("id = ?", licenseID).
		Update("used_this_month", gorm.Expr("used_this_month + 1"))

	// Log the usage
	log := &B2BUsageLog{
		LicenseID:  licenseID,
		Endpoint:   endpoint,
		Method:     method,
		StatusCode: statusCode,
		LatencyMs:  latencyMs,
		IPAddress:  ipAddress,
	}
	return s.db.Create(log).Error
}

// ResetMonthlyUsage resets all usage counters (to be called by monthly cron)
func (s *LicensingService) ResetMonthlyUsage() error {
	return s.db.Model(&B2BLicense{}).Where("1=1").Update("used_this_month", 0).Error
}

// GetUsageSummary returns usage analytics for a license
func (s *LicensingService) GetUsageSummary(licenseID uint) (map[string]interface{}, error) {
	var license B2BLicense
	if err := s.db.First(&license, licenseID).Error; err != nil {
		return nil, fmt.Errorf("license not found")
	}

	// Endpoint breakdown
	var endpointStats []struct {
		Endpoint string
		Calls    int
		AvgMs    float64
		Errors   int
	}
	s.db.Raw(`
		SELECT endpoint, COUNT(*) AS calls, AVG(latency_ms) AS avg_ms,
		SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END) AS errors
		FROM b2b_usage_logs
		WHERE license_id = ? AND created_at >= date_trunc('month', NOW())
		GROUP BY endpoint ORDER BY calls DESC LIMIT 20
	`, licenseID).Scan(&endpointStats)

	return map[string]interface{}{
		"license_id":      licenseID,
		"org_name":        license.OrgName,
		"plan":            license.Plan,
		"used_this_month": license.UsedThisMonth,
		"monthly_limit":   license.MonthlyLimit,
		"usage_pct":       float64(license.UsedThisMonth) / float64(license.MonthlyLimit) * 100,
		"endpoints":       endpointStats,
	}, nil
}

// UpgradePlan upgrades a license to a higher plan
func (s *LicensingService) UpgradePlan(licenseID uint, newPlan string) error {
	limit, ok := PlanLimits[newPlan]
	if !ok {
		return fmt.Errorf("invalid plan: %s", newPlan)
	}

	updates := map[string]interface{}{
		"plan":          newPlan,
		"monthly_limit": limit,
	}
	// Remove expiry for enterprise/growth
	if newPlan != "starter" {
		updates["expires_at"] = nil
	}

	return s.db.Model(&B2BLicense{}).Where("id = ?", licenseID).Updates(updates).Error
}

// ListLicenses returns all B2B licenses.
func (s *LicensingService) ListLicenses() ([]B2BLicense, error) {
	var licenses []B2BLicense
	err := s.db.Order("created_at DESC").Find(&licenses).Error
	return licenses, err
}

// RevokeLicense deactivates an API key
func (s *LicensingService) RevokeLicense(licenseID uint) error {
	return s.db.Model(&B2BLicense{}).Where("id = ?", licenseID).Update("is_active", false).Error
}

// AutoMigrate creates B2B licensing tables
func (s *LicensingService) AutoMigrate() error {
	return s.db.AutoMigrate(&B2BLicense{}, &B2BUsageLog{})
}

// generateKey creates a cryptographically secure random hex string
func generateKey(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

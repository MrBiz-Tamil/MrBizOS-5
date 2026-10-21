package cms

import (
	"time"

	"gorm.io/gorm"
)

// SiteConfig represents a brand website domain configuration.
type SiteConfig struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	OrganizationID  uint           `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID  uint           `gorm:"index;not null" json:"business_unit_id"`
	Domain          string         `gorm:"size:255;uniqueIndex;not null" json:"domain"` // e.g. "travel-brand.com", "health-brand.com"
	BrandName       string         `gorm:"size:128;not null" json:"brand_name"`
	LogoURL         string         `gorm:"size:512" json:"logo_url,omitempty"`
	FaviconURL      string         `gorm:"size:512" json:"favicon_url,omitempty"`
	PrimaryColor    string         `gorm:"size:32;default:'#10b981'" json:"primary_color"`
	SecondaryColor  string         `gorm:"size:32;default:'#0f172a'" json:"secondary_color"`
	FontFamily      string         `gorm:"size:64;default:'Inter'" json:"font_family"`
	MetaTitle       string         `gorm:"size:255" json:"meta_title,omitempty"`
	MetaDescription string         `gorm:"type:text" json:"meta_description,omitempty"`
	AnalyticsID     string         `gorm:"size:64" json:"analytics_id,omitempty"`
	IsActive        bool           `gorm:"default:true" json:"is_active"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// CMSPage represents a dynamic SEO-optimized web page scoped to a specific site.
type CMSPage struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	SiteID          uint           `gorm:"index;not null" json:"site_id"`
	Slug            string         `gorm:"size:255;index;not null" json:"slug"` // e.g. "/", "/tours", "/doctors", "/properties"
	Title           string         `gorm:"size:255;not null" json:"title"`
	ContentJSON     string         `gorm:"type:text" json:"content_json"` // Section layout payload
	SEOKeywords     string         `gorm:"size:255" json:"seo_keywords,omitempty"`
	CanonicalURL    string         `gorm:"size:512" json:"canonical_url,omitempty"`
	MetaDescription string         `gorm:"type:text" json:"meta_description,omitempty"`
	IsPublished     bool           `gorm:"default:true;index" json:"is_published"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// AutoMigrate migrates multi-site CMS domain models.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&SiteConfig{}, &CMSPage{})
}

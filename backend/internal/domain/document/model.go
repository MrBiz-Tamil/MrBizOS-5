// Package document is the doctor/staff record vault: a single table that owns all
// uploaded files (passport photo, CV, license certificate, degree, ID proof,
// agreement) with metadata, verification status, expiry, and an audit trail.
// Files live in object storage (see internal/services/storage); this table holds
// only metadata + the storage key, never the bytes.
package document

import (
	"time"

	"gorm.io/gorm"
)

// Owner types.
const (
	OwnerApplication = "application" // pre-hire, before a user account exists
	OwnerDoctor      = "doctor"
	OwnerPatient     = "patient"
)

// Status values.
const (
	StatusPending  = "pending"
	StatusVerified = "verified"
	StatusRejected = "rejected"
)

// Visibility controls how a document is served. Private docs go through signed
// URLs + an access check; public docs (blog covers, product images) have a
// stable CDN URL and are served to everyone.
const (
	VisibilityPrivate = "private"
	VisibilityPublic  = "public"
)

// Document is one stored file plus its metadata.
type Document struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	OwnerType string `gorm:"index;size:20" json:"owner_type"` // application | doctor | patient
	OwnerID   uint   `gorm:"index" json:"owner_id"`           // 0 until linked (applicant pre-hire)
	Category  string `gorm:"index;size:32" json:"category"`   // passport_photo | resume | license_cert | degree | id_proof | agreement

	StorageKey   string `gorm:"size:255" json:"-"` // object key in the bucket — never exposed
	OriginalName string `gorm:"size:255" json:"original_name"`
	ContentType  string `gorm:"size:128" json:"content_type"`
	SizeBytes    int64  `json:"size_bytes"`

	// Visibility is private (signed-URL, access-checked) or public (CDN-served).
	Visibility string `gorm:"default:'private';size:10;index" json:"visibility"`
	// PublicURL is the stable CDN URL for public assets; empty for private docs.
	PublicURL string `gorm:"size:512" json:"public_url,omitempty"`

	Status     string     `gorm:"default:'pending';size:16;index" json:"status"`
	UploadedBy *uint      `json:"uploaded_by,omitempty"` // null for anonymous applicant
	VerifiedBy *uint      `json:"verified_by,omitempty"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	// ExpiryDate is set for credentials that lapse (license_cert) and feeds the
	// compliance monitor once wired.
	ExpiryDate *time.Time `json:"expiry_date,omitempty"`

	// Versioning — a re-upload supersedes the previous row instead of overwriting.
	Version      int   `gorm:"default:1" json:"version"`
	SupersedesID *uint `json:"supersedes_id,omitempty"`

	// AccessToken is an unguessable handle the anonymous applicant flow uses to
	// reference a freshly-uploaded file before any account exists. Authorized
	// (post-hire) access goes through the authenticated, owner-checked route.
	AccessToken string `gorm:"uniqueIndex;size:48" json:"access_token"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Document) TableName() string { return "documents" }

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Document{})
}

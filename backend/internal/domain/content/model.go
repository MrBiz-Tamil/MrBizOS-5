package content

import (
	"encoding/json"
	"drayya-core/internal/domain/identity"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Blog struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	Title      string         `gorm:"not null" json:"title"`
	Slug       string         `gorm:"uniqueIndex" json:"slug"`
	Excerpt    string         `json:"excerpt"`
	Content    string         `gorm:"type:text;not null" json:"content"`
	Category   string         `gorm:"index;not null" json:"category"`
	AuthorID   uint           `gorm:"index;not null" json:"author_id"`
	Author     identity.User  `gorm:"foreignKey:AuthorID" json:"author"`
	Status     string         `gorm:"default:'pending';index" json:"status"`
	ApproverID *uint          `json:"approver_id"`
	ImageURL   string         `json:"image_url"`
	ReadTime   string         `json:"read_time"`
	IsFeatured bool           `gorm:"default:false" json:"is_featured"`

	// SEO
	MetaTitle       string         `json:"meta_title"`
	MetaDescription string         `json:"meta_description"`
	Tags            datatypes.JSON `gorm:"type:jsonb" json:"tags"`    // ["siddha","diabetes"]
	Sources         datatypes.JSON `gorm:"type:jsonb" json:"sources"` // [{title,url,publisher}] — required for publish

	// Medical-review credibility (E-E-A-T). Reviewer must differ from the author.
	ReviewerID *uint         `json:"reviewer_id"`
	Reviewer   *identity.User `gorm:"foreignKey:ReviewerID" json:"reviewer,omitempty"`
	ReviewedAt *time.Time    `json:"reviewed_at"`

	PublishedAt        *time.Time `json:"published_at"`
	ScheduledPublishAt *time.Time `gorm:"index" json:"scheduled_publish_at"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Blog) TableName() string {
	return "blogs"
}

// Source is one cited reference backing the article's health claims.
type Source struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	Publisher string `json:"publisher"`
}

// ParsedSources decodes the Sources JSON into typed citations.
func (b *Blog) ParsedSources() []Source {
	if len(b.Sources) == 0 {
		return nil
	}
	var out []Source
	_ = json.Unmarshal(b.Sources, &out)
	return out
}

// NewsletterSubscriber tracks user emails for news and updates
type NewsletterSubscriber struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (NewsletterSubscriber) TableName() string {
	return "newsletter_subscribers"
}

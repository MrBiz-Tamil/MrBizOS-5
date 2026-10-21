package content

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	AutoMigrate() error
	CreateBlog(ctx context.Context, blog *Blog) error
	GetBlogByID(ctx context.Context, id uint) (*Blog, error)
	GetBlogBySlug(ctx context.Context, slug string) (*Blog, error)
	SlugExists(ctx context.Context, slug string) (bool, error)
	HasDoctorCredentials(ctx context.Context, userID uint) (bool, error)
	ListPublicBlogs(ctx context.Context, category, search string) ([]Blog, error)
	ListPendingBlogs(ctx context.Context) ([]Blog, error)
	ListApprovedSlugs(ctx context.Context) ([]Blog, error)
	DueScheduledBlogs(ctx context.Context) ([]Blog, error)
	UpdateBlog(ctx context.Context, id uint, updates map[string]interface{}) error
	Subscribe(ctx context.Context, subscriber *NewsletterSubscriber) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) AutoMigrate() error {
	return r.db.AutoMigrate(&Blog{}, &NewsletterSubscriber{})
}

func (r *repository) CreateBlog(ctx context.Context, blog *Blog) error {
	return r.db.WithContext(ctx).Create(blog).Error
}

func (r *repository) GetBlogByID(ctx context.Context, id uint) (*Blog, error) {
	var blog Blog
	err := r.db.WithContext(ctx).Preload("Author").First(&blog, id).Error
	return &blog, err
}

func (r *repository) GetBlogBySlug(ctx context.Context, slug string) (*Blog, error) {
	var blog Blog
	err := r.db.WithContext(ctx).Preload("Author").Preload("Reviewer").
		Where("slug = ?", slug).First(&blog).Error
	return &blog, err
}

func (r *repository) SlugExists(ctx context.Context, slug string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Blog{}).Where("slug = ?", slug).Count(&n).Error
	return n > 0, err
}

// HasDoctorCredentials reports whether the user has a verified doctor profile
// (specialization + license) — the bar to author/review authoritative health content.
func (r *repository) HasDoctorCredentials(ctx context.Context, userID uint) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("doctorprofile").
		Where("user_id = ? AND specialization <> '' AND license_number <> ''", userID).
		Count(&n).Error
	return n > 0, err
}

// ListApprovedSlugs returns approved posts (slug + updated_at) for the sitemap.
func (r *repository) ListApprovedSlugs(ctx context.Context) ([]Blog, error) {
	var blogs []Blog
	err := r.db.WithContext(ctx).Select("slug", "updated_at").
		Where("status = ? AND slug <> ''", "approved").Order("updated_at desc").Find(&blogs).Error
	return blogs, err
}

// DueScheduledBlogs returns pending posts whose scheduled time has arrived.
func (r *repository) DueScheduledBlogs(ctx context.Context) ([]Blog, error) {
	var blogs []Blog
	err := r.db.WithContext(ctx).Preload("Author").
		Where("status = ? AND scheduled_publish_at IS NOT NULL AND scheduled_publish_at <= ?", "pending", time.Now()).
		Find(&blogs).Error
	return blogs, err
}

func (r *repository) ListPublicBlogs(ctx context.Context, category, search string) ([]Blog, error) {
	var blogs []Blog
	query := r.db.WithContext(ctx).Preload("Author").Where("status = ?", "approved")

	if category != "" && category != "All" {
		query = query.Where("category = ?", category)
	}

	if search != "" {
		query = query.Where("title ILIKE ? OR excerpt ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	err := query.Order("created_at desc").Find(&blogs).Error
	return blogs, err
}

func (r *repository) ListPendingBlogs(ctx context.Context) ([]Blog, error) {
	var blogs []Blog
	err := r.db.WithContext(ctx).Preload("Author").Where("status = ?", "pending").Order("created_at asc").Find(&blogs).Error
	return blogs, err
}

func (r *repository) UpdateBlog(ctx context.Context, id uint, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&Blog{}).Where("id = ?", id).Updates(updates).Error
}

func (r *repository) Subscribe(ctx context.Context, subscriber *NewsletterSubscriber) error {
	// Use clauses to ignore duplicate emails or update them
	return r.db.WithContext(ctx).Save(subscriber).Error
}

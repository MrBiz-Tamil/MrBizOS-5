package content

import (
	"context"
	"fmt"
	"time"

	"drayya-core/internal/domain/communication"
)

type Service interface {
	SubmitBlog(ctx context.Context, blog *Blog) error
	GetBlogByID(ctx context.Context, id uint) (*Blog, error)
	GetBlogBySlug(ctx context.Context, slug string) (*Blog, error)
	GetPublicBlogs(ctx context.Context, category, search string) ([]Blog, error)
	GetPendingBlogs(ctx context.Context) ([]Blog, error)
	ListApprovedSlugs(ctx context.Context) ([]Blog, error)
	UpdateBlogStatus(ctx context.Context, blogID uint, approverID uint, status string) error
	PublishDueScheduled(ctx context.Context) (int, error)
	Subscribe(ctx context.Context, email, source string) error
}

type service struct {
	repo    Repository
	commSvc communication.Service
}

func NewService(repo Repository, commSvc communication.Service) Service {
	return &service{repo: repo, commSvc: commSvc}
}

func (s *service) SubmitBlog(ctx context.Context, blog *Blog) error {
	blog.Status = "pending"
	blog.Content = SanitizeHTML(blog.Content) // strip XSS before storing
	blog.Slug = s.uniqueSlug(ctx, blog.Title)
	return s.repo.CreateBlog(ctx, blog)
}

// uniqueSlug returns a slug for the title, suffixing -2, -3… on collision.
func (s *service) uniqueSlug(ctx context.Context, title string) string {
	base := slugify(title)
	slug := base
	for i := 2; i < 100; i++ {
		exists, err := s.repo.SlugExists(ctx, slug)
		if err != nil || !exists {
			break
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
	return slug
}

func (s *service) GetBlogByID(ctx context.Context, id uint) (*Blog, error) {
	return s.repo.GetBlogByID(ctx, id)
}

func (s *service) GetBlogBySlug(ctx context.Context, slug string) (*Blog, error) {
	return s.repo.GetBlogBySlug(ctx, slug)
}

func (s *service) ListApprovedSlugs(ctx context.Context) ([]Blog, error) {
	return s.repo.ListApprovedSlugs(ctx)
}

func (s *service) GetPublicBlogs(ctx context.Context, category, search string) ([]Blog, error) {
	return s.repo.ListPublicBlogs(ctx, category, search)
}

func (s *service) GetPendingBlogs(ctx context.Context) ([]Blog, error) {
	return s.repo.ListPendingBlogs(ctx)
}

func (s *service) UpdateBlogStatus(ctx context.Context, blogID uint, approverID uint, status string) error {
	if status != "approved" && status != "rejected" {
		return fmt.Errorf("invalid status: %s", status)
	}

	if status == "rejected" {
		return s.repo.UpdateBlog(ctx, blogID, map[string]interface{}{
			"status": "rejected", "approver_id": approverID,
		})
	}

	// status == "approved": enforce the strict "real health data only" gate.
	blog, err := s.repo.GetBlogByID(ctx, blogID)
	if err != nil {
		return fmt.Errorf("blog not found")
	}
	if err := s.assertPublishable(ctx, blog, approverID); err != nil {
		return err
	}

	now := time.Now()
	return s.repo.UpdateBlog(ctx, blogID, map[string]interface{}{
		"status":       "approved",
		"approver_id":  approverID,
		"reviewer_id":  approverID,
		"reviewed_at":  now,
		"published_at": now,
	})
}

// assertPublishable enforces medical E-E-A-T before a post can go live:
//   1. the reviewer must be a different person from the author (4-eyes);
//   2. the reviewer must be a verified doctor (medically reviewed by a professional);
//   3. the author must be a verified doctor (specialization + license);
//   4. at least one cited source must back the health claims.
func (s *service) assertPublishable(ctx context.Context, blog *Blog, reviewerID uint) error {
	if blog.AuthorID == reviewerID {
		return fmt.Errorf("the medical reviewer must be a different person from the author (4-eyes review)")
	}
	if ok, _ := s.repo.HasDoctorCredentials(ctx, reviewerID); !ok {
		return fmt.Errorf("only a verified doctor can medically review and publish health content")
	}
	if ok, _ := s.repo.HasDoctorCredentials(ctx, blog.AuthorID); !ok {
		return fmt.Errorf("the author must be a verified doctor (specialization + license) to publish health content")
	}
	if len(blog.ParsedSources()) < 1 {
		return fmt.Errorf("at least one cited source is required to publish health content")
	}
	return nil
}

// PublishDueScheduled flips pending posts whose scheduled time has arrived to
// approved — but only those that still satisfy the medical gate. Driven by cron.
func (s *service) PublishDueScheduled(ctx context.Context) (int, error) {
	due, err := s.repo.DueScheduledBlogs(ctx)
	if err != nil {
		return 0, err
	}
	published := 0
	for i := range due {
		b := &due[i]
		// A scheduled post still needs a valid reviewer; if none was assigned
		// (ReviewerID), it stays pending for manual review rather than auto-publishing.
		if b.ReviewerID == nil {
			continue
		}
		if err := s.assertPublishable(ctx, b, *b.ReviewerID); err != nil {
			continue
		}
		now := time.Now()
		if err := s.repo.UpdateBlog(ctx, b.ID, map[string]interface{}{
			"status": "approved", "reviewed_at": now, "published_at": now,
		}); err == nil {
			published++
		}
	}
	return published, nil
}

func (s *service) Subscribe(ctx context.Context, email, source string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}

	subscriber := &NewsletterSubscriber{
		Email:  email,
		Source: source,
		IsActive: true,
	}

	if err := s.repo.Subscribe(ctx, subscriber); err != nil {
		return err
	}

	// Send Welcome Email
	subject := "Welcome to Dr.Ayya Wellness Tips! 🌿"
	body := "Hi there,\n\nThank you for subscribing to our blog. We're excited to have you on board! You'll now receive weekly health tips, Siddha remedies, and wellness advice directly in your inbox.\n\nBe Well,\nThe Dr.Ayya Team"
	
	// Ignore email sending errors for subscription success
	_ = s.commSvc.SendEmail(ctx, email, subject, body)

	return nil
}

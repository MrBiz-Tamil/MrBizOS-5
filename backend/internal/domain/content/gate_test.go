package content

import (
	"context"
	"strings"
	"testing"

	"gorm.io/datatypes"
)

// mockRepo lets us exercise the publish-gate logic without a database.
type mockRepo struct {
	blog        *Blog
	doctors     map[uint]bool // userID -> has verified credentials
	lastUpdates map[string]interface{}
	updated     bool
}

func (m *mockRepo) AutoMigrate() error                                  { return nil }
func (m *mockRepo) CreateBlog(ctx context.Context, b *Blog) error       { return nil }
func (m *mockRepo) GetBlogByID(ctx context.Context, id uint) (*Blog, error) { return m.blog, nil }
func (m *mockRepo) GetBlogBySlug(ctx context.Context, s string) (*Blog, error) { return m.blog, nil }
func (m *mockRepo) SlugExists(ctx context.Context, s string) (bool, error)     { return false, nil }
func (m *mockRepo) HasDoctorCredentials(ctx context.Context, userID uint) (bool, error) {
	return m.doctors[userID], nil
}
func (m *mockRepo) ListPublicBlogs(ctx context.Context, c, s string) ([]Blog, error) { return nil, nil }
func (m *mockRepo) ListPendingBlogs(ctx context.Context) ([]Blog, error)             { return nil, nil }
func (m *mockRepo) ListApprovedSlugs(ctx context.Context) ([]Blog, error)            { return nil, nil }
func (m *mockRepo) DueScheduledBlogs(ctx context.Context) ([]Blog, error)            { return nil, nil }
func (m *mockRepo) UpdateBlog(ctx context.Context, id uint, u map[string]interface{}) error {
	m.updated = true
	m.lastUpdates = u
	return nil
}
func (m *mockRepo) Subscribe(ctx context.Context, n *NewsletterSubscriber) error { return nil }

func newSvc(blog *Blog, doctors map[uint]bool) (Service, *mockRepo) {
	m := &mockRepo{blog: blog, doctors: doctors}
	return NewService(m, nil), m
}

const oneSource = `[{"title":"WHO Diabetes Fact Sheet","url":"https://who.int","publisher":"WHO"}]`

func TestGate_RejectsSelfReview(t *testing.T) {
	blog := &Blog{ID: 1, AuthorID: 5, Sources: datatypes.JSON(oneSource)}
	svc, m := newSvc(blog, map[uint]bool{5: true})
	err := svc.UpdateBlogStatus(context.Background(), 1, 5, "approved") // reviewer == author
	if err == nil || !strings.Contains(err.Error(), "different person") {
		t.Fatalf("expected 4-eyes rejection, got %v", err)
	}
	if m.updated {
		t.Fatal("must not publish on self-review")
	}
}

func TestGate_RejectsNonDoctorReviewer(t *testing.T) {
	blog := &Blog{ID: 1, AuthorID: 5, Sources: datatypes.JSON(oneSource)}
	svc, _ := newSvc(blog, map[uint]bool{5: true /* author */, 9: false /* reviewer not doctor */})
	err := svc.UpdateBlogStatus(context.Background(), 1, 9, "approved")
	if err == nil || !strings.Contains(err.Error(), "verified doctor") {
		t.Fatalf("expected non-doctor reviewer rejection, got %v", err)
	}
}

func TestGate_RejectsNonDoctorAuthor(t *testing.T) {
	blog := &Blog{ID: 1, AuthorID: 5, Sources: datatypes.JSON(oneSource)}
	svc, _ := newSvc(blog, map[uint]bool{9: true /* reviewer */ /* author 5 missing */})
	err := svc.UpdateBlogStatus(context.Background(), 1, 9, "approved")
	if err == nil || !strings.Contains(err.Error(), "author must be a verified doctor") {
		t.Fatalf("expected non-doctor author rejection, got %v", err)
	}
}

func TestGate_RejectsNoSources(t *testing.T) {
	blog := &Blog{ID: 1, AuthorID: 5} // no Sources
	svc, _ := newSvc(blog, map[uint]bool{5: true, 9: true})
	err := svc.UpdateBlogStatus(context.Background(), 1, 9, "approved")
	if err == nil || !strings.Contains(err.Error(), "cited source") {
		t.Fatalf("expected missing-source rejection, got %v", err)
	}
}

func TestGate_AllowsWhenSatisfied(t *testing.T) {
	blog := &Blog{ID: 1, AuthorID: 5, Sources: datatypes.JSON(oneSource)}
	svc, m := newSvc(blog, map[uint]bool{5: true, 9: true})
	if err := svc.UpdateBlogStatus(context.Background(), 1, 9, "approved"); err != nil {
		t.Fatalf("expected publish to succeed, got %v", err)
	}
	if !m.updated || m.lastUpdates["status"] != "approved" || m.lastUpdates["reviewer_id"] != uint(9) {
		t.Fatalf("expected approved with reviewer set, got %+v", m.lastUpdates)
	}
	if m.lastUpdates["reviewed_at"] == nil || m.lastUpdates["published_at"] == nil {
		t.Fatal("reviewed_at/published_at must be set on publish")
	}
}

func TestSanitizeHTML_StripsXSS(t *testing.T) {
	out := SanitizeHTML(`<p onclick="steal()">hi</p><script>evil()</script><a href="javascript:bad()">x</a>`)
	if strings.Contains(out, "<script") || strings.Contains(out, "onclick") || strings.Contains(strings.ToLower(out), "javascript:") {
		t.Fatalf("sanitizer left an XSS vector: %q", out)
	}
	if !strings.Contains(out, "hi") {
		t.Fatalf("sanitizer dropped legitimate content: %q", out)
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Siddha for Diabetes!": "siddha-for-diabetes",
		"  Hello   World  ":    "hello-world",
		"":                     "post",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

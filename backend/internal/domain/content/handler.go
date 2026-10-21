package content

import (
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	blogGroup := router.Group("/blogs")
	blogGroup.Get("/", h.GetPublicPosts)
	router.Get("/blog/posts", h.GetPublicPosts) // Alias for compatibility
	blogGroup.Get("/sitemap", h.GetSitemapEntries)
	blogGroup.Get("/slug/:slug", h.GetPostBySlug)
	blogGroup.Get("/:id", h.GetPost)
	blogGroup.Post("/subscribe", h.Subscribe)

	// Protected Doctor Blog Routes
	doctorBlogs := router.Group("/doctor/blogs", middleware.Protected())
	doctorBlogs.Post("/", h.SubmitPost)

	// Admin Blog Routes
	adminBlogs := router.Group("/admin/blogs", middleware.Protected())
	adminBlogs.Get("/pending", h.GetPendingPosts)
	adminBlogs.Post("/approve/:id", h.ApprovePost)
}

func (h *Handler) SubmitPost(c *fiber.Ctx) error {
	role, _ := c.Locals("role").(string)
	if role != "doctor" && role != "hospital_head_doctor" && role != "chief_doctor" {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Only doctors can submit journals")
	}

	var newBlog Blog
	if err := c.BodyParser(&newBlog); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload")
	}

	authorID := c.Locals("user_id").(uint)
	newBlog.AuthorID = authorID

	if err := h.svc.SubmitBlog(c.Context(), &newBlog); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SUBMIT_FAILED", "Failed to submit post")
	}

	return response.JSON(c, fiber.StatusCreated, newBlog, "Journal submitted successfully")
}

func (h *Handler) GetPublicPosts(c *fiber.Ctx) error {
	category := c.Query("category")
	search := c.Query("search")

	blogs, err := h.svc.GetPublicBlogs(c.Context(), category, search)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch blogs")
	}

	return response.JSON(c, fiber.StatusOK, blogs, "Public blogs fetched")
}

func (h *Handler) GetPendingPosts(c *fiber.Ctx) error {
	role, _ := c.Locals("role").(string)
	if role != "admin" && role != "super_admin" && role != "hospital_head_doctor" && role != "chief_doctor" {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Not authorized to view pending posts")
	}

	blogs, err := h.svc.GetPendingBlogs(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch pending blogs")
	}

	return response.JSON(c, fiber.StatusOK, blogs, "Pending blogs fetched")
}

func (h *Handler) GetPost(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	b, err := h.svc.GetBlogByID(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Blog not found")
	}

	userID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	if b.Status != "approved" {
		if role != "admin" && role != "super_admin" && role != "hospital_head_doctor" && role != "chief_doctor" && userID != b.AuthorID {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Cannot access draft post")
		}
	}

	return response.JSON(c, fiber.StatusOK, b, "Blog details fetched")
}

// GetPostBySlug returns an approved post by slug (for server-side rendering +
// SEO). Only approved posts are exposed here.
func (h *Handler) GetPostBySlug(c *fiber.Ctx) error {
	b, err := h.svc.GetBlogBySlug(c.Context(), c.Params("slug"))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Blog not found")
	}
	if b.Status != "approved" {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Blog not found")
	}
	return response.JSON(c, fiber.StatusOK, b, "Blog details fetched")
}

// GetSitemapEntries returns approved slugs + last-modified for the sitemap.
func (h *Handler) GetSitemapEntries(c *fiber.Ctx) error {
	blogs, err := h.svc.ListApprovedSlugs(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed")
	}
	out := make([]fiber.Map, 0, len(blogs))
	for _, b := range blogs {
		out = append(out, fiber.Map{"slug": b.Slug, "updated_at": b.UpdatedAt})
	}
	return response.JSON(c, fiber.StatusOK, out, "Sitemap entries")
}

func (h *Handler) ApprovePost(c *fiber.Ctx) error {
	role, _ := c.Locals("role").(string)
	if role != "admin" && role != "super_admin" && role != "hospital_head_doctor" && role != "chief_doctor" {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Not authorized to approve posts")
	}

	id, _ := strconv.Atoi(c.Params("id"))
	approverID := c.Locals("user_id").(uint)

	var req struct {
		Status string `json:"status"` // allow 'approved' or 'rejected'
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload")
	}

	if err := h.svc.UpdateBlogStatus(c.Context(), uint(id), uint(approverID), req.Status); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Status updated successfully")
}

func (h *Handler) Subscribe(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload")
	}

	if err := h.svc.Subscribe(c.Context(), req.Email, "blog_footer"); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SUBSCRIBE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Successfully subscribed to newsletter")
}

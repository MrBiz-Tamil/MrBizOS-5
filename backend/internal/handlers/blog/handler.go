package blog

import (
	"drayya-core/internal/domain/content"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/middleware"
	"drayya-core/internal/services/blog"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func SubmitPost(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || (role != string(identity.RoleDoctor) && role != string(identity.RoleHospitalHeadDoctor) && role != "chief_doctor") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only doctors can submit journals"})
	}

	var newBlog content.Blog
	if err := c.BodyParser(&newBlog); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	authorID, ok := c.Locals("user_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	newBlog.AuthorID = authorID
	newBlog.Status = "pending"

	if err := blog.CreateBlog(&newBlog); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to submit post"})
	}

	return c.Status(fiber.StatusCreated).JSON(newBlog)
}

// GetPublicPosts returns approved posts for the frontend blog page
func GetPublicPosts(c *fiber.Ctx) error {
	category := c.Query("category")
	search := c.Query("search")

	blogs, err := blog.GetPublicBlogs(category, search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch blogs"})
	}

	return c.JSON(blogs)
}

// GetPendingPosts returning posts awaiting approval for Admins/ChiefDoctors
func GetPendingPosts(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || (role != string(identity.RoleAdmin) && role != string(identity.RoleHospitalHeadDoctor) && role != "chief_doctor") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Not authorized to view pending posts"})
	}

	blogs, err := blog.GetPendingBlogs()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch pending blogs"})
	}

	return c.JSON(blogs)
}

// GetPost returns a single blog detail
func GetPost(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	b, err := blog.GetBlogByID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Blog not found"})
	}

	// Important check for public access
	var hasUser bool
	var role string
	var userID uint
	if c.Locals("user_id") != nil {
		hasUser = true
		role, _ = c.Locals("role").(string)
		userID, _ = c.Locals("user_id").(uint)
	}

	if b.Status != "approved" {
		if !hasUser {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Cannot access draft post"})
		}

		if role != string(identity.RoleAdmin) && role != string(identity.RoleHospitalHeadDoctor) && role != "chief_doctor" && userID != b.AuthorID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Cannot access draft post"})
		}
	}

	return c.JSON(b)
}

// ApprovePost updates the status of a pending post
func ApprovePost(c *fiber.Ctx) error {
	role, ok := c.Locals("role").(string)
	if !ok || (role != string(identity.RoleAdmin) && role != string(identity.RoleHospitalHeadDoctor) && role != "chief_doctor") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Not authorized to approve posts"})
	}

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	approverID, ok := c.Locals("user_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	var req struct {
		Status string `json:"status"` // allow 'approved' or 'rejected'
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	if req.Status != "approved" && req.Status != "rejected" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid status"})
	}

	if err := blog.UpdateBlogStatus(uint(id), uint(approverID), req.Status); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update blog status"})
	}

	return c.JSON(fiber.Map{"message": "Status updated successfully"})
}

func RegisterBlogRoutes(router fiber.Router) {
	blogGroup := router.Group("/blogs")
	blogGroup.Get("/", GetPublicPosts)
	blogGroup.Get("/:id", GetPost)

	// Protected Doctor Blog Routes
	doctorBlogs := router.Group("/doctor/blogs", middleware.Protected())
	doctorBlogs.Post("/", SubmitPost)
}

package questionnaire

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"drayya-core/internal/middleware"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) RegisterRoutes(router fiber.Router) {
	q := router.Group("/questionnaires")

	// Public: list templates
	q.Get("/templates", listTemplates)
	q.Get("/templates/:id", getTemplate)

	// Patient
	q.Post("/submit", middleware.Protected(), submitResponse)
	q.Get("/my", middleware.Protected(), getMyResponses)

	// Doctor: review
	doctor := q.Group("/review", middleware.Protected(), middleware.RequireRoles("doctor", "admin"))
	doctor.Get("/pending", getPendingReviews)
	doctor.Post("/:id/mark-reviewed", markReviewed)

	// Admin: manage templates and questions
	admin := q.Group("/admin", middleware.Protected(), middleware.RequireRoles("admin", "super_admin"))
	admin.Post("/templates", createTemplate)
	admin.Put("/templates/:id", updateTemplate)
	admin.Post("/templates/:id/questions", addQuestion)
	admin.Delete("/questions/:id", deleteQuestion)
}

// ── Template handlers ─────────────────────────────────────────────────────────

func listTemplates(c *fiber.Ctx) error {
	templates, err := GetAllTemplates(true)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(templates)
}

func getTemplate(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	t, err := GetTemplateByID(uint(id))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "template not found"})
	}
	return c.JSON(t)
}

func createTemplate(c *fiber.Ctx) error {
	var t Template
	if err := c.BodyParser(&t); err != nil || t.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name is required"})
	}
	if err := CreateTemplate(&t); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(t)
}

func updateTemplate(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	existing, err := GetTemplateByID(uint(id))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	if err := c.BodyParser(existing); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if err := UpdateTemplate(existing); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(existing)
}

func addQuestion(c *fiber.Ctx) error {
	templateID, _ := strconv.Atoi(c.Params("id"))
	var q Question
	if err := c.BodyParser(&q); err != nil || q.Text == "" {
		return c.Status(400).JSON(fiber.Map{"error": "question text is required"})
	}
	q.TemplateID = uint(templateID)
	if err := AddQuestion(&q); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(q)
}

func deleteQuestion(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	if err := DeleteQuestion(uint(id)); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "question deleted"})
}

// ── Patient handlers ──────────────────────────────────────────────────────────

type SubmitRequest struct {
	TemplateID     uint     `json:"template_id"`
	ConsultationID *uint    `json:"consultation_id,omitempty"`
	Answers        []Answer `json:"answers"`
}

func submitResponse(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var req SubmitRequest
	if err := c.BodyParser(&req); err != nil || req.TemplateID == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "template_id and answers are required"})
	}

	resp := &Response{
		TemplateID:     req.TemplateID,
		PatientUserID:  userID,
		ConsultationID: req.ConsultationID,
		Answers:        req.Answers,
		Status:         "submitted",
	}
	if err := CreateResponse(resp); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(resp)
}

func getMyResponses(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	responses, err := GetResponsesByPatient(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(responses)
}

// ── Doctor review handlers ────────────────────────────────────────────────────

func getPendingReviews(c *fiber.Ctx) error {
	reviews, err := GetPendingReviews()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(reviews)
}

func markReviewed(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	reviewerID := c.Locals("user_id").(uint)
	var body struct {
		Summary string `json:"summary"`
	}
	_ = c.BodyParser(&body)

	resp, err := GetResponseByID(uint(id))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "response not found"})
	}
	now := time.Now()
	resp.Status = "reviewed"
	resp.ReviewedBy = &reviewerID
	resp.ReviewedAt = &now
	resp.Summary = body.Summary
	if err := UpdateResponse(resp); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(resp)
}

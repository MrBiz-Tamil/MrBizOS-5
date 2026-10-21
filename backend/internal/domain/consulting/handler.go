package consulting

import (
	"strconv"
	"time"

	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r fiber.Router) {
	c := r.Group("/consulting")
	c.Get("/clients", h.ListClients)
	c.Post("/clients", h.CreateClient)
	c.Get("/projects", h.ListProjects)
	c.Post("/projects", h.CreateProject)
	c.Get("/projects/:id", h.GetProject)
	c.Post("/milestones", h.CreateMilestone)
	c.Get("/milestones", h.ListMilestones)
	c.Post("/timesheets", h.LogTimesheet)
	c.Get("/timesheets", h.ListTimesheets)
}

func (h *Handler) ListClients(c *fiber.Ctx) error {
	clients, err := h.svc.ListClients()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, clients, "Client accounts fetched successfully")
}

func (h *Handler) CreateClient(c *fiber.Ctx) error {
	var client ClientAccount
	if err := c.BodyParser(&client); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	created, err := h.svc.CreateClient(&client)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, created, "Client account created successfully")
}

func (h *Handler) ListProjects(c *fiber.Ctx) error {
	var clientID uint
	if cid := c.Query("client_id"); cid != "" {
		if id, err := strconv.ParseUint(cid, 10, 32); err == nil {
			clientID = uint(id)
		}
	}

	projs, err := h.svc.ListProjects(clientID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, projs, "Software projects fetched successfully")
}

func (h *Handler) CreateProject(c *fiber.Ctx) error {
	var req struct {
		ClientID uint    `json:"client_id"`
		Title    string  `json:"title"`
		Code     string  `json:"code"`
		Budget   float64 `json:"budget"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	proj, err := h.svc.CreateProject(req.ClientID, req.Title, req.Code, req.Budget)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, proj, "Software project created successfully")
}

func (h *Handler) GetProject(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Project ID must be an integer")
	}

	proj, err := h.svc.GetProject(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Project not found")
	}
	return response.JSON(c, fiber.StatusOK, proj, "Software project fetched successfully")
}

func (h *Handler) CreateMilestone(c *fiber.Ctx) error {
	var req struct {
		ProjectID  uint    `json:"project_id"`
		Title      string  `json:"title"`
		Amount     float64 `json:"amount"`
		TargetDate string  `json:"target_date"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	var target *time.Time
	if req.TargetDate != "" {
		if parsed, err := time.Parse(time.RFC3339, req.TargetDate); err == nil {
			target = &parsed
		}
	}

	ms, err := h.svc.CreateMilestone(req.ProjectID, req.Title, req.Amount, target)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, ms, "Milestone created successfully")
}

func (h *Handler) ListMilestones(c *fiber.Ctx) error {
	var projectID uint
	if pid := c.Query("project_id"); pid != "" {
		if id, err := strconv.ParseUint(pid, 10, 32); err == nil {
			projectID = uint(id)
		}
	}

	mss, err := h.svc.ListMilestones(projectID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, mss, "Milestones fetched successfully")
}

func (h *Handler) LogTimesheet(c *fiber.Ctx) error {
	var req struct {
		ProjectID   uint    `json:"project_id"`
		UserID      uint    `json:"user_id"`
		Hours       float64 `json:"hours"`
		Description string  `json:"description"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	if req.UserID == 0 {
		if uid, ok := c.Locals("user_id").(uint); ok {
			req.UserID = uid
		} else {
			req.UserID = 1
		}
	}

	ts, err := h.svc.LogTimesheet(req.ProjectID, req.UserID, req.Hours, req.Description)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "LOG_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, ts, "Timesheet logged successfully")
}

func (h *Handler) ListTimesheets(c *fiber.Ctx) error {
	var projectID, userID uint
	if pid := c.Query("project_id"); pid != "" {
		if id, err := strconv.ParseUint(pid, 10, 32); err == nil {
			projectID = uint(id)
		}
	}
	if uid := c.Query("user_id"); uid != "" {
		if id, err := strconv.ParseUint(uid, 10, 32); err == nil {
			userID = uint(id)
		}
	}

	tss, err := h.svc.ListTimesheets(projectID, userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, tss, "Timesheets fetched successfully")
}

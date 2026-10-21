package task

import (
	"strconv"

	"drayya-core/internal/middleware"
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
	tasks := r.Group("/tasks")
	tasks.Get("/", h.ListTasks)
	tasks.Post("/", h.CreateTask)
	tasks.Get("/:id", h.GetTask)
	tasks.Put("/:id", h.UpdateTask)
	tasks.Post("/:id/complete", h.CompleteTask)
	tasks.Delete("/:id", h.DeleteTask)
	tasks.Post("/:id/comments", h.AddComment)
	tasks.Get("/:id/comments", h.GetComments)
}

func (h *Handler) ListTasks(c *fiber.Ctx) error {
	buID := middleware.GetBusinessUnitID(c)
	orgID := middleware.GetOrganizationID(c)

	filter := Filter{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         c.Query("status"),
		Priority:       c.Query("priority"),
		Search:         c.Query("search"),
	}

	if assigneeStr := c.Query("assignee_id"); assigneeStr != "" {
		if id, err := strconv.ParseUint(assigneeStr, 10, 32); err == nil {
			uid := uint(id)
			filter.AssigneeID = &uid
		}
	}
	if ownerStr := c.Query("owner_id"); ownerStr != "" {
		if id, err := strconv.ParseUint(ownerStr, 10, 32); err == nil {
			uid := uint(id)
			filter.OwnerID = &uid
		}
	}
	if custStr := c.Query("customer_id"); custStr != "" {
		if id, err := strconv.ParseUint(custStr, 10, 32); err == nil {
			uid := uint(id)
			filter.CustomerID = &uid
		}
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	filter.Page = page
	filter.Limit = limit

	tasks, total, err := h.svc.ListTasks(filter)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "TASK_ERROR", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"tasks": tasks,
		"total": total,
		"page":  page,
		"limit": limit,
	}, "Tasks fetched successfully")
}

func (h *Handler) CreateTask(c *fiber.Ctx) error {
	var req Task
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	if userID, ok := c.Locals("user_id").(uint); ok {
		req.OwnerID = userID
	} else {
		req.OwnerID = 1 // Default fallback for dev/testing
	}

	req.BusinessUnitID = middleware.GetBusinessUnitID(c)
	req.OrganizationID = middleware.GetOrganizationID(c)

	created, err := h.svc.CreateTask(&req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CREATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, created, "Task created successfully")
}

func (h *Handler) GetTask(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Task ID must be an integer")
	}

	t, err := h.svc.GetTask(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Task not found")
	}

	return response.JSON(c, fiber.StatusOK, t, "Task fetched successfully")
}

func (h *Handler) UpdateTask(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Task ID must be an integer")
	}

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	updated, err := h.svc.UpdateTask(uint(id), updates)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "UPDATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, updated, "Task updated successfully")
}

func (h *Handler) CompleteTask(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Task ID must be an integer")
	}

	completed, err := h.svc.CompleteTask(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "COMPLETE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, completed, "Task completed successfully")
}

func (h *Handler) DeleteTask(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Task ID must be an integer")
	}

	if err := h.svc.DeleteTask(uint(id)); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{"message": "Task deleted successfully"}, "Task deleted successfully")
}

func (h *Handler) AddComment(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Task ID must be an integer")
	}

	var req struct {
		Comment string `json:"comment"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	userID := uint(1)
	if uid, ok := c.Locals("user_id").(uint); ok {
		userID = uid
	}

	comment, err := h.svc.AddComment(uint(id), userID, req.Comment)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "COMMENT_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, comment, "Comment added successfully")
}

func (h *Handler) GetComments(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Task ID must be an integer")
	}

	comments, err := h.svc.GetComments(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, comments, "Comments fetched successfully")
}

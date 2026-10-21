package support

import (
	"drayya-core/internal/events"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"
	"strconv"
	"context"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service  Service
	eventBus *events.Bus
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) SetEventBus(bus *events.Bus) {
	h.eventBus = bus
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func userIDFromCtx(c *fiber.Ctx) uint {
	if id, ok := c.Locals("user_id").(uint); ok {
		return id
	}
	if id, ok := c.Locals("userID").(uint); ok {
		return id
	}
	return 0
}

func roleFromCtx(c *fiber.Ctx) string {
	if role, ok := c.Locals("role").(string); ok {
		return role
	}
	return ""
}

// Map roles to their specific department silo
func getDepartmentForRole(role string) string {
	switch role {
	case "finance_manager":
		return "billing"
	case "chief_doctor", "hospital_head_doctor", "doctor":
		return "medical"
	case "pharmacy_manager", "pharmacist":
		return "pharmacy"
	case "hr_manager":
		return "hr"
	case "inventory_manager":
		return "logistics"
	case "admin", "super_admin":
		return "all" // Can see everything
	default:
		return "none" // Prevents arbitrary users from accessing admin routes
	}
}

// ── User routes ───────────────────────────────────────────────────────────────

// GET /support/tickets — list caller's tickets
func (h *Handler) GetTickets(c *fiber.Ctx) error {
	userID := userIDFromCtx(c)
	tickets, err := h.service.GetUserTickets(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"tickets": tickets,
		"count":   len(tickets),
	}, "Tickets fetched")
}

// POST /support/tickets — create ticket
func (h *Handler) CreateTicket(c *fiber.Ctx) error {
	userID := userIDFromCtx(c)
	var req struct {
		Subject     string `json:"subject"`
		Category    string `json:"category"`
		Priority    string `json:"priority"`
		Message     string `json:"message"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}
	// Accept either "message" or "description" field from frontend
	desc := req.Description
	if desc == "" {
		desc = req.Message
	}
	ticket, err := h.service.CreateTicket(userID, req.Subject, req.Category, req.Priority, desc)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CREATE_FAILED", err.Error())
	}
	if h.eventBus != nil {
		h.eventBus.Publish(context.Background(), events.EventTicketCreated, userID, map[string]interface{}{
			"ticket_id": ticket.ID, "user_id": userID, "department": ticket.Department,
		})
	}
	return response.JSON(c, fiber.StatusCreated, ticket, "Support ticket created")
}

// GET /support/tickets/:id — ticket detail with messages
func (h *Handler) GetTicketDetails(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", err.Error())
	}
	ticket, err := h.service.GetTicketDetails(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Ticket not found")
	}
	return response.JSON(c, fiber.StatusOK, ticket, "Ticket details fetched")
}

// POST /support/tickets/:id/messages — user adds a message
func (h *Handler) AddTicketMessage(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	userID := userIDFromCtx(c)
	var req struct {
		Message string `json:"message"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}
	msg, err := h.service.AddMessage(uint(id), userID, req.Message, false)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "MESSAGE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, msg, "Message added")
}

// PUT /support/tickets/:id/status — user closes their own ticket
func (h *Handler) UpdateTicketStatus(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var req struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}
	ticket, err := h.service.UpdateStatus(uint(id), req.Status)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	if h.eventBus != nil {
		h.eventBus.Publish(context.Background(), events.EventTicketStatusUpdated, userIDFromCtx(c), map[string]interface{}{
			"ticket_id": ticket.ID, "status": ticket.Status,
		})
	}
	return response.JSON(c, fiber.StatusOK, ticket, "Status updated")
}

// ── Admin routes ──────────────────────────────────────────────────────────────

// GET /support/admin/tickets?status=&department=
func (h *Handler) AdminListTickets(c *fiber.Ctx) error {
	role := roleFromCtx(c)
	allowedDept := getDepartmentForRole(role)
	
	if allowedDept == "none" {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You do not have permission to view support tickets")
	}

	status := c.Query("status")
	dept := c.Query("department")
	
	// Override requested department if user is restricted
	if allowedDept != "all" {
		dept = allowedDept
	}

	tickets, err := h.service.GetAllTickets(status, dept)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"tickets": tickets,
		"count":   len(tickets),
	}, "All tickets fetched")
}

// GET /support/admin/tickets/:id
func (h *Handler) AdminGetTicket(c *fiber.Ctx) error {
	role := roleFromCtx(c)
	allowedDept := getDepartmentForRole(role)
	
	if allowedDept == "none" {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Access denied")
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", err.Error())
	}
	ticket, err := h.service.GetTicketDetails(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Ticket not found")
	}
	
	// Ensure restricted roles only see their department's tickets
	if allowedDept != "all" && ticket.Department != allowedDept {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "This ticket belongs to another department")
	}
	
	return response.JSON(c, fiber.StatusOK, ticket, "Ticket fetched")
}

// POST /support/admin/tickets/:id/reply
func (h *Handler) AdminReply(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	agentID := userIDFromCtx(c)
	var req struct {
		Message    string `json:"message"`
		SenderName string `json:"sender_name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}
	msg, err := h.service.AdminReply(uint(id), agentID, req.SenderName, req.Message)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "REPLY_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, msg, "Reply sent")
}

// PUT /support/admin/tickets/:id/status
func (h *Handler) AdminUpdateStatus(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var req struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}
	ticket, err := h.service.UpdateStatus(uint(id), req.Status)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	if h.eventBus != nil {
		h.eventBus.Publish(context.Background(), events.EventTicketStatusUpdated, userIDFromCtx(c), map[string]interface{}{
			"ticket_id": ticket.ID, "status": ticket.Status,
		})
	}
	return response.JSON(c, fiber.StatusOK, ticket, "Status updated")
}

// PUT /support/admin/tickets/:id/assign
func (h *Handler) AdminAssign(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var req struct {
		AgentID uint `json:"agent_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}
	if err := h.service.AssignTicket(uint(id), req.AgentID); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ASSIGN_FAILED", err.Error())
	}
	if h.eventBus != nil {
		h.eventBus.Publish(context.Background(), events.EventTicketAssigned, userIDFromCtx(c), map[string]interface{}{
			"ticket_id": id, "agent_id": req.AgentID,
		})
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"ticket_id": id, "agent_id": req.AgentID}, "Assigned")
}

// PUT /support/admin/tickets/:id/department
func (h *Handler) AdminSetDepartment(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var req struct {
		Department string `json:"department"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}
	if err := h.service.SetDepartment(uint(id), req.Department); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "DEPT_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"ticket_id": id, "department": req.Department}, "Department updated")
}

// GET /support/admin/stats
func (h *Handler) AdminStats(c *fiber.Ctx) error {
	stats, err := h.service.GetStats()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "STATS_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, stats, "Stats fetched")
}

// ── Route registration ────────────────────────────────────────────────────────

func (h *Handler) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler) {
	g := router.Group("/support", authMiddleware)

	// User routes
	g.Get("/tickets", h.GetTickets)
	g.Post("/tickets", h.CreateTicket)
	g.Get("/tickets/:id", h.GetTicketDetails)
	g.Post("/tickets/:id/messages", h.AddTicketMessage)
	g.Put("/tickets/:id/status", h.UpdateTicketStatus)

	// Admin/Staff routes
	admin := g.Group("/admin", middleware.StaffOnly())
	admin.Get("/tickets", h.AdminListTickets)
	admin.Get("/tickets/:id", h.AdminGetTicket)
	admin.Post("/tickets/:id/reply", h.AdminReply)
	admin.Put("/tickets/:id/status", h.AdminUpdateStatus)
	admin.Put("/tickets/:id/assign", h.AdminAssign)
	admin.Put("/tickets/:id/department", h.AdminSetDepartment)
	admin.Get("/stats", h.AdminStats)
}

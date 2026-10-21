//go:build ignore

package support

import (
	"drayya-core/internal/database"
	"drayya-core/internal/models"
	"drayya-core/internal/utils/response"

	"drayya-core/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// GetTickets returns a list of support tickets for the current user
func GetTickets(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(uint)

	var tickets []models.Ticket
	if err := database.DB.Where("user_id = ?", userID).Preload("Messages").Find(&tickets).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "TICKETS_FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"tickets": tickets,
		"count":   len(tickets),
	}, "Fetched")
}

// CreateTicket creates a new support ticket
func CreateTicket(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(uint)

	var req struct {
		Subject  string `json:"subject"`
		Priority string `json:"priority"`
		Message  string `json:"message"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	ticket := models.Ticket{
		UserID:   userID,
		Subject:  req.Subject,
		Priority: req.Priority,
		Status:   "open",
	}

	if err := database.DB.Create(&ticket).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_TICKET_FAILED", err.Error())
	}

	// Add the first message
	msg := models.TicketMessage{
		TicketID: ticket.ID,
		SenderID: userID,
		Message:  req.Message,
		IsStaff:  false,
	}
	database.DB.Create(&msg)

	return response.JSON(c, fiber.StatusCreated, ticket, "Created")
}

// AddTicketMessage adds a response to an existing ticket
func AddTicketMessage(c *fiber.Ctx) error {
	ticketID := c.Params("id")
	userID, _ := c.Locals("user_id").(uint)

	var req struct {
		Message string `json:"message"`
		IsStaff bool   `json:"is_staff"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	// Find ticket
	var ticket models.Ticket
	if err := database.DB.First(&ticket, ticketID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "TICKET_NOT_FOUND", "Ticket not found")
	}

	msg := models.TicketMessage{
		TicketID: ticket.ID,
		SenderID: userID,
		Message:  req.Message,
		IsStaff:  req.IsStaff,
	}

	if err := database.DB.Create(&msg).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_MESSAGE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, msg, "Created")
}

// GetTicketDetails returns a single ticket and its messages
func GetTicketDetails(c *fiber.Ctx) error {
	ticketID := c.Params("id")
	var ticket models.Ticket
	if err := database.DB.Preload("Messages").First(&ticket, ticketID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "TICKET_NOT_FOUND", "Ticket not found")
	}
	return response.JSON(c, fiber.StatusOK, ticket, "Fetched")
}

// UpdateTicketStatus updates the status of a ticket
func UpdateTicketStatus(c *fiber.Ctx) error {
	ticketID := c.Params("id")
	var req struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	var ticket models.Ticket
	if err := database.DB.First(&ticket, ticketID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "TICKET_NOT_FOUND", "Ticket not found")
	}

	ticket.Status = req.Status
	if err := database.DB.Save(&ticket).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_TICKET_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, ticket, "Fetched")
}

func RegisterSupportRoutes(router fiber.Router) {
	supportGroup := router.Group("/support", middleware.Protected())
	supportGroup.Get("/tickets", GetTickets)
	supportGroup.Post("/tickets", CreateTicket)
	supportGroup.Get("/tickets/:id", GetTicketDetails)
	supportGroup.Post("/tickets/:id/messages", AddTicketMessage)
	supportGroup.Put("/tickets/:id/status", UpdateTicketStatus)
}

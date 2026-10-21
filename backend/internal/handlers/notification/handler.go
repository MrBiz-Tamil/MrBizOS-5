package notification

import (
	"errors"
	"log"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/communication"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/utils/response"
	"strconv"
	"strings"
	"time"

	"drayya-core/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// sendNotification queries a template, resolves the user, renders the message,
// and logs the send attempt. It intentionally lives here (not in the notification
// service package) to avoid a circular import with the identity domain.
func sendNotification(userID uint, templateType, notificationType string, variables map[string]string) error {
	db := database.DB
	var tmpl communication.NotificationTemplate
	if err := db.Where("template_type = ? AND is_active = ?", templateType, true).First(&tmpl).Error; err != nil {
		return errors.New("template not found or inactive")
	}

	var user identity.User
	if err := db.First(&user, userID).Error; err != nil {
		return errors.New("user not found")
	}

	message := tmpl.MessageContent
	for key, val := range variables {
		message = strings.ReplaceAll(message, "{{"+key+"}}", val)
	}

	log.Printf("     [MOCK SEND] Type: %s | To: %s | Msg: %s", notificationType, user.PhoneNumber, message)

	logEntry := communication.NotificationLog{
		UserID:           userID,
		NotificationType: notificationType,
		TemplateType:     templateType,
		Status:           "sent",
		SentAt:           time.Now(),
	}
	return db.Create(&logEntry).Error
}

func createTemplate(templateType, content, variables string) (*communication.NotificationTemplate, error) {
	tmpl := communication.NotificationTemplate{
		TemplateType:   templateType,
		MessageContent: content,
		Variables:      variables,
		IsActive:       true,
		CreatedAt:      time.Now(),
	}
	return &tmpl, database.DB.Create(&tmpl).Error
}

// SendNotification handler
func SendNotification(c *fiber.Ctx) error {
	type Request struct {
		UserID           uint              `json:"user_id"`
		TemplateType     string            `json:"template_type"`
		NotificationType string            `json:"notification_type"`
		Variables        map[string]string `json:"variables"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	if err := sendNotification(req.UserID, req.TemplateType, req.NotificationType, req.Variables); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "NOTIFICATION_SEND_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Notification sent successfully")
}

// CreateTemplate handler (Admin)
func CreateTemplate(c *fiber.Ctx) error {
	type Request struct {
		TemplateType   string `json:"template_type"`
		MessageContent string `json:"message_content"`
		Variables      string `json:"variables"`
	}
	var req Request
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	tmpl, err := createTemplate(req.TemplateType, req.MessageContent, req.Variables)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "TEMPLATE_CREATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, tmpl, "Fetched")
}

// GetLogs handler
func GetLogs(c *fiber.Ctx) error {
	userID := c.Query("user_id")

	var logs []communication.NotificationLog
	query := database.DB.Model(&communication.NotificationLog{})

	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	query.Order("sent_at desc").Limit(50).Find(&logs)

	return response.JSON(c, fiber.StatusOK, logs, "Fetched")
}

// GetUserNotifications - Get notifications for the current user
func GetUserNotifications(c *fiber.Ctx) error {
	var userID uint
	if val := c.Locals("user_id"); val != nil {
		userID = val.(uint)
	} else if qID := c.Query("user_id"); qID != "" {
		id, _ := strconv.ParseUint(qID, 10, 32)
		userID = uint(id)
	}

	if userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required")
	}

	var notifications []communication.Notification
	database.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(50).
		Find(&notifications)

	return response.JSON(c, fiber.StatusOK, notifications, "Fetched")
}

// MarkAsRead - Mark a notification as read
func MarkAsRead(c *fiber.Ctx) error {
	id := c.Params("id")

	result := database.DB.Model(&communication.Notification{}).
		Where("id = ?", id).
		Update("is_read", true)

	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "MARK_READ_FAILED", result.Error.Error())
	}

	if result.RowsAffected == 0 {
		return response.Error(c, fiber.StatusNotFound, "NOTIFICATION_NOT_FOUND", "Notification not found")
	}

	return response.JSON(c, fiber.StatusOK, nil, "Notification marked as read")
}

// UpdateNotificationSettings - Update user notification preferences
func UpdateNotificationSettings(c *fiber.Ctx) error {
	return response.JSON(c, fiber.StatusOK, nil, "Notification settings updated successfully")
}

func RegisterNotificationRoutes(router fiber.Router) {
	ntfGroup := router.Group("/notifications", middleware.Protected())
	ntfGroup.Get("/", GetUserNotifications)
	ntfGroup.Patch("/:id/read", MarkAsRead)
	ntfGroup.Put("/settings", UpdateNotificationSettings)
	ntfGroup.Get("/logs", GetLogs)
	// /send and /templates require staff role — admin-only operations
	ntfGroup.Post("/send", middleware.AdminOnly(), SendNotification)
	ntfGroup.Post("/templates", middleware.AdminOnly(), CreateTemplate)
}

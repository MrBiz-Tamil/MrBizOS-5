//go:build ignore
// +build ignore

package handlers

import (
	"drayya-core/internal/database"
	"drayya-core/internal/models"
	"drayya-core/internal/utils/response"
	"strconv"

	"drayya-core/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// GetFamilyMembers - List all family members for a user
func GetFamilyMembers(c *fiber.Ctx) error {
	var userID uint
	if val := c.Locals("user_id"); val != nil {
		userID = val.(uint)
	} else if qID := c.Query("user_id"); qID != "" {
		id, _ := strconv.ParseUint(qID, 10, 32)
		userID = uint(id)
	}

	if userID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "UNAUTHORIZED_OR_MISSING_USER_ID", "Unauthorized or user_id required")
	}

	var members []models.FamilyMember
	database.DB.Where("user_id = ?", userID).Find(&members)

	return response.JSON(c, fiber.StatusOK, members, "Fetched")
}

// AddFamilyMember - Add a new family member
func AddFamilyMember(c *fiber.Ctx) error {
	var userID uint
	if val := c.Locals("user_id"); val != nil {
		userID = val.(uint)
	}

	var member models.FamilyMember
	if err := c.BodyParser(&member); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	member.UserID = userID
	result := database.DB.Create(&member)
	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_MEMBER_FAILED", result.Error.Error())
	}

	return response.JSON(c, fiber.StatusCreated, member, "Created")
}

// UpdateFamilyMember - Update family member details
func UpdateFamilyMember(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	result := database.DB.Model(&models.FamilyMember{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(updates)

	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_MEMBER_FAILED", result.Error.Error())
	}

	if result.RowsAffected == 0 {
		return response.Error(c, fiber.StatusNotFound, "MEMBER_NOT_FOUND", "Member not found")
	}

	return response.JSON(c, fiber.StatusOK, nil, "Member updated successfully")
}

// DeleteFamilyMember - Delete family member
func DeleteFamilyMember(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)

	result := database.DB.Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.FamilyMember{})

	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DELETE_MEMBER_FAILED", result.Error.Error())
	}

	if result.RowsAffected == 0 {
		return response.Error(c, fiber.StatusNotFound, "MEMBER_NOT_FOUND", "Member not found")
	}

	return response.JSON(c, fiber.StatusOK, nil, "Member deleted successfully")
}

// GetFamilyMember - Get single family member
func GetFamilyMember(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)

	var member models.FamilyMember
	result := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&member)

	if result.Error != nil {
		return response.Error(c, fiber.StatusNotFound, "MEMBER_NOT_FOUND", "Member not found")
	}

	return response.JSON(c, fiber.StatusOK, member, "Fetched")
}

func RegisterFamilyRoutes(router fiber.Router) {
	familyGroup := router.Group("/family", middleware.Protected())
	familyGroup.Get("/members", GetFamilyMembers)
	familyGroup.Post("/members", AddFamilyMember)
	familyGroup.Get("/members/:id", GetFamilyMember)
	familyGroup.Put("/members/:id", UpdateFamilyMember)
	familyGroup.Delete("/members/:id", DeleteFamilyMember)
}

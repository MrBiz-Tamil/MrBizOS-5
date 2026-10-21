//go:build ignore
// +build ignore

package handlers

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"

	"drayya-core/internal/auth"
	"drayya-core/internal/database"
	"drayya-core/internal/handlers/notification"
	"drayya-core/internal/middleware"
	"drayya-core/internal/models"
	"drayya-core/internal/utils/response"
	"drayya-core/internal/ws"

	"github.com/gofiber/fiber/v2"
)

// Login godoc
// @Summary User Login
// @Description Authenticate a user and return JWT tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body auth.LoginRequest true "Login Request"
// @Success 200 {object} auth.TokenResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
func Login(c *fiber.Ctx) error {
	var req auth.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	resp, err := auth.LoginUser(req)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "LOGIN_FAILED", err.Error())
	}

	// ---------------------------------------------------------
	// Real-Time Doctor Status Update
	// ---------------------------------------------------------
	if resp.User.Role == "doctor" {
		go func(userID uint, name string) {
			var profile models.DoctorProfile
			if result := database.DB.Where("user_id = ?", userID).First(&profile); result.Error == nil {
				// 1. Update Online Status
				database.DB.Model(&profile).Update("is_online", true)

				// 2. Prepare Broadcast Message
				msg := ws.WebSocketMessage{
					Type:       "doctor_status_update",
					SenderID:   userID,
					SenderName: name,
					SenderRole: "doctor",
					Timestamp:  time.Now().Format(time.RFC3339),
					Metadata: map[string]interface{}{
						"status":              "online",
						"specialization":      profile.Specialization,
						"average_rating":      profile.AverageRating,
						"total_consultations": profile.TotalConsultations,
						"is_verified":         profile.IsVerified,
					},
				}

				// 3. Publish to Redis (Hub will pick it up) - only if Redis is available
				if ws.Rdb != nil {
					jsonMsg, _ := json.Marshal(msg)
					ws.Rdb.Publish(context.Background(), "chat:global", jsonMsg)
					log.Printf("     Broadcasted Online Status for Doctor: %s (Rating: %.1f)", name, profile.AverageRating)
				}
			}
		}(resp.User.ID, resp.User.FullName)
	}
	// ---------------------------------------------------------

	return c.JSON(resp)
}

// StaffLogin godoc
// @Summary Staff Login
// @Description Authenticate a staff member and return JWT tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body auth.LoginRequest true "Staff Login Request"
// @Success 200 {object} auth.MFAResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/staff/login [post]
func StaffLogin(c *fiber.Ctx) error {
	var req auth.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	resp, err := auth.LoginStaff(req)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "STAFF_LOGIN_FAILED", err.Error())
	}

	return c.JSON(resp)
}

func VerifyMFA(c *fiber.Ctx) error {
	var req struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required")
	}

	resp, err := auth.VerifyMFA(userID, req.Code)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "MFA_VERIFICATION_FAILED", err.Error())
	}

	return c.JSON(resp)
}

func Logout(c *fiber.Ctx) error {
	// For stateless JWT, we just return success. Client discards token.
	// Optionally we could blacklist the token in Redis here if we wanted strict logout.
	return c.JSON(fiber.Map{"message": "Logged out successfully"})
}

func Register(c *fiber.Ctx) error {
	var user models.User
	var payload struct {
		models.User
		Password string `json:"password"`
	}

	if err := c.BodyParser(&payload); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	user = payload.User
	newUser, err := auth.RegisterUser(c.UserContext(), user, payload.Password)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "duplicate key") {
			return response.Error(c, fiber.StatusConflict, "USER_ALREADY_EXISTS", "User already exists")
		}
		return response.Error(c, fiber.StatusInternalServerError, "REGISTRATION_FAILED", "Could not register user")
	}

	// Auto-login to generate tokens
	tokenResp, err := auth.LoginUser(auth.LoginRequest{
		Email:    newUser.Email,
		Password: payload.Password,
	})
	if err != nil {
		// Fallback: Return user without tokens if login fails (unlikely)
		return response.JSON(c, fiber.StatusCreated, newUser, "User registered")
	}

	return response.JSON(c, fiber.StatusCreated, tokenResp, "User registered")
}

func RefreshToken(c *fiber.Ctx) error {
	type RefreshRequest struct {
		RefreshToken string `json:"refresh_token"`
	}

	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	resp, err := auth.RefreshToken(req.RefreshToken)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "TOKEN_REFRESH_FAILED", err.Error())
	}

	return c.JSON(resp)
}

func ResetPassword(c *fiber.Ctx) error {
	type ResetRequest struct {
		Email string `json:"email"`
	}

	var req ResetRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	// Implementation for password reset (email etc) would go here
	// For now, we return a success message stub
	return c.JSON(fiber.Map{"message": "If this email is registered, you will receive reset instructions."})
}

// GetProfile - Get current user profile

func GetProfile(c *fiber.Ctx) error {
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

	var user models.User
	result := database.DB.First(&user, userID)

	if result.Error != nil {
		return response.Error(c, fiber.StatusNotFound, "USER_NOT_FOUND", "User not found")
	}

	// Remove sensitive data
	user.PasswordHash = ""

	return c.JSON(user)
}

// UpdateProfile - Update user profile
func UpdateProfile(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required")
	}

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	// Remove fields that shouldn't be updated via this endpoint
	delete(updates, "id")
	delete(updates, "password_hash")
	delete(updates, "role")

	result := database.DB.Model(&models.User{}).Where("id = ?", userID).Updates(updates)

	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", result.Error.Error())
	}

	if result.RowsAffected == 0 {
		return response.Error(c, fiber.StatusNotFound, "USER_NOT_FOUND", "User not found")
	}

	return c.JSON(fiber.Map{"message": "Profile updated successfully"})
}

func RegisterAuthRoutes(router fiber.Router) {
	authGroup := router.Group("/auth")
	authGroup.Post("/login", middleware.AuthLimiter(), Login)
	authGroup.Post("/staff/login", middleware.AuthLimiter(), StaffLogin)
	authGroup.Post("/mfa/verify", middleware.AuthLimiter(), middleware.Protected(), VerifyMFA)
	authGroup.Post("/logout", Logout)
	authGroup.Post("/register", Register)
	authGroup.Post("/refresh-token", RefreshToken)
	authGroup.Post("/reset-password", ResetPassword)

	// Protected Auth Routes
	authGroup.Get("/profile", middleware.Protected(), GetProfile)
	authGroup.Get("/me", middleware.Protected(), GetProfile)
	authGroup.Put("/profile", middleware.Protected(), UpdateProfile)
	authGroup.Put("/notification-settings", middleware.Protected(), notification.UpdateNotificationSettings)
}


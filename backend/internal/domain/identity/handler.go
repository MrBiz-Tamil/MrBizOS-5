package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"drayya-core/internal/database"
	"drayya-core/internal/events"
	"drayya-core/internal/middleware"
	"drayya-core/internal/presence"
	"drayya-core/internal/services/activity_log"
	"drayya-core/internal/services/notification"
	"drayya-core/internal/tokens"
	"drayya-core/internal/utils/response"
	"drayya-core/internal/utils/validator"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service  Service
	eventBus *events.Bus
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// SetEventBus injects the domain event bus after construction.
// Called from server/app.go after the bus is initialized.
func (h *Handler) SetEventBus(bus *events.Bus) {
	h.eventBus = bus
}

// Login handles user authentication
// @Summary User Login
// @Description Authenticate user and return tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login Credentials"
// @Success 200 {object} response.StandardResponse{data=TokenResponse} "Login successful"
// @Failure 401 {object} response.StandardError "Invalid credentials"
// @Router /api/v1/auth/login [post]
func (h *Handler) Login(c *fiber.Ctx) error {
	// Parse a union body that accepts both email+password and phone+OTP.
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Phone    string `json:"phone"`
		OTP      string `json:"otp"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	// Phone+OTP branch (Android)
	if body.Phone != "" {
		if body.OTP == "" {
			return response.Error(c, fiber.StatusBadRequest, "OTP_REQUIRED", "OTP is required for phone login")
		}
		resp, err := h.service.LoginWithOTP(body.Phone, body.OTP)
		if err != nil {
			activity_log.LogAuthFailure(c, activity_log.ActionLoginFailed, body.Phone, err.Error())
			return response.Error(c, fiber.StatusUnauthorized, "LOGIN_FAILED", err.Error())
		}
		if resp.User != nil && resp.RefreshToken != "" {
			if err := tokens.StoreRefreshToken(c.Context(), resp.User.ID, resp.RefreshToken); err != nil {
				log.Printf("[auth] failed to store refresh token for user %d: %v", resp.User.ID, err)
			}
			activity_log.LogAuthSuccess(c, resp.User.ID, activity_log.ActionLogin, map[string]interface{}{
				"method": "phone_otp",
				"role":   resp.User.Role,
			})
		}
		return response.JSON(c, fiber.StatusOK, resp, "Login successful")
	}

	// Email+password branch (default)
	req := LoginRequest{Email: body.Email, Password: body.Password}
	if err := validator.ValidateStruct(c, &req); err != nil {
		return err
	}

	resp, err := h.service.Login(req)
	if err != nil {
		activity_log.LogAuthFailure(c, activity_log.ActionLoginFailed, req.Email, err.Error())
		return response.Error(c, fiber.StatusUnauthorized, "LOGIN_FAILED", err.Error())
	}

	// Store refresh token in Redis for rotation on next refresh / invalidation on logout
	if resp.User != nil && resp.RefreshToken != "" {
		if err := tokens.StoreRefreshToken(c.Context(), resp.User.ID, resp.RefreshToken); err != nil {
			log.Printf("[auth] failed to store refresh token for user %d: %v", resp.User.ID, err)
		}
	}
	if resp.User != nil {
		activity_log.LogAuthSuccess(c, resp.User.ID, activity_log.ActionLogin, map[string]interface{}{
			"method": "password",
			"role":   resp.User.Role,
		})
	}

	// Real-Time Doctor Status Update
	if resp.User.Role == "doctor" {
		go func(user *User) {
			profile, err := h.service.GetDoctorProfile(user.ID)
			if err == nil {
				if database.Rdb != nil {
					jsonMsg, _ := json.Marshal(map[string]interface{}{
						"type":        "doctor_status_update",
						"sender_id":   user.ID,
						"sender_name": user.FullName,
						"sender_role": "doctor",
						"timestamp":   time.Now().Format(time.RFC3339),
						"metadata": map[string]interface{}{
							"status":              "online",
							"specialization":      profile.Specialization,
							"average_rating":      profile.AverageRating,
							"total_consultations": profile.TotalConsultations,
							"is_verified":         profile.IsVerified,
						},
					})
					database.Rdb.Publish(context.Background(), "chat:global", jsonMsg)
					log.Printf("     Broadcasted Online Status for Doctor: %s", user.FullName)
				}
			}
		}(resp.User)
	}

	return response.JSON(c, fiber.StatusOK, resp, "Login successful")
}

// StaffLogin handles staff-specific authentication with MFA setup
// @Summary Staff Login
// @Description Authenticate staff and initiate MFA
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Staff Credentials"
// @Success 200 {object} response.StandardResponse{data=MFAResponse} "MFA code sent"
// @Failure 401 {object} response.StandardError "Invalid staff credentials"
// @Router /api/v1/auth/staff/login [post]
func (h *Handler) StaffLogin(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	if err := validator.ValidateStruct(c, &req); err != nil {
		return err
	}

	resp, err := h.service.LoginStaff(req)
	if err != nil {
		activity_log.LogAuthFailure(c, activity_log.ActionLoginFailed, req.Email, "staff_login: "+err.Error())
		return response.Error(c, fiber.StatusUnauthorized, "LOGIN_FAILED", err.Error())
	}
	// Note: this is the password-validated step. Full login completes after MFA.
	activity_log.LogAsync(0, "staff_login_step1", "auth", req.Email,
		map[string]interface{}{"email": req.Email, "mfa": "pending"}, c)

	// Send OTP via WhatsApp in addition to email (already done in service).
	// Publish via event bus when available; fall back to direct goroutine.
	if h.eventBus != nil {
		h.eventBus.Publish(c.Context(), events.EventOTPSent, 0, map[string]interface{}{
			"email": req.Email,
		})
	} else {
		go func() {
			var user User
			if database.DB.Select("phone_number, full_name, mfa_code").
				Where("email = ?", req.Email).First(&user).Error == nil && user.PhoneNumber != "" {
				notification.WhatsAppOTP(user.PhoneNumber, user.FullName, user.MFACode)
			}
		}()
	}

	return response.JSON(c, fiber.StatusOK, resp, "Staff login successful")
}

// VerifyMFA verifies the MFA code for staff
func (h *Handler) VerifyMFA(c *fiber.Ctx) error {
	var req struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	// Ensure this endpoint is only callable with an mfa_pending token
	pendingRole, _ := c.Locals("role").(string)
	if pendingRole != "mfa_pending" {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "MFA step not initiated")
	}

	// Rate-limit MFA attempts: max 3 failures per 5 minutes via Redis counter
	if database.Rdb != nil {
		attemptsKey := fmt.Sprintf("mfa_attempts:%d", userID)
		attempts, _ := database.Rdb.Incr(c.Context(), attemptsKey).Result()
		if attempts == 1 {
			database.Rdb.Expire(c.Context(), attemptsKey, 5*time.Minute)
		}
		if attempts > 3 {
			return response.Error(c, fiber.StatusTooManyRequests, "MFA_LOCKED",
				"Too many failed attempts. MFA token locked. Please log in again.")
		}
	}

	resp, err := h.service.VerifyMFA(userID, req.Code)
	if err != nil {
		log.Printf("[Auth] MFA Verification Failed for user %d: %v (code entered: %s)", userID, err, req.Code)
		activity_log.LogAsync(userID, activity_log.ActionMFAFailed, "auth", "",
			map[string]interface{}{"reason": err.Error()}, c)
		return response.Error(c, fiber.StatusUnauthorized, "MFA_VERIFICATION_FAILED", err.Error())
	}
	activity_log.LogAsync(userID, activity_log.ActionMFAVerified, "auth", "", nil, c)

	// Clear the attempt counter on success
	if database.Rdb != nil {
		database.Rdb.Del(c.Context(), fmt.Sprintf("mfa_attempts:%d", userID))
	}

	// BUG FIX: Store the refresh token in Redis for rotation on next refresh /
	// invalidation on logout. Without this, every staff-login's refresh token is
	// absent from Redis, causing ValidateAndRotateRefreshToken to return 401
	// exactly 15 minutes later (access token expiry), forcing a full logout.
	if resp != nil && resp.RefreshToken != "" {
		if err := tokens.StoreRefreshToken(c.Context(), userID, resp.RefreshToken); err != nil {
			log.Printf("[Auth] WARNING: failed to store refresh token in Redis for user %d: %v — "+
				"user will be logged out when access token expires in 15 min", userID, err)
		}
	}

	return response.JSON(c, fiber.StatusOK, resp, "MFA verified")
}

// ResendMFA generates a fresh OTP and re-sends it to the user's email.
// POST /api/v1/auth/mfa/resend
// Requires a valid mfa_token in the Authorization header.
func (h *Handler) ResendMFA(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	if err := h.service.ResendMFA(userID); err != nil {
		// Return 429 for rate-limit errors, 500 for others
		if len(err.Error()) > 12 && err.Error()[:12] == "please wait " {
			return response.Error(c, fiber.StatusTooManyRequests, "RATE_LIMITED", err.Error())
		}
		return response.Error(c, fiber.StatusInternalServerError, "RESEND_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "MFA code resent to your registered email")
}

// Register creates a new user account
// @Summary Register User
// @Description Create a new patient or doctor account
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body identity.User true "User Data"
// @Success 201 {object} response.StandardResponse{data=TokenResponse} "Registration successful"
// @Failure 400 {object} response.StandardError "Invalid request body"
// @Failure 409 {object} response.StandardError "User already exists"
// @Router /api/v1/auth/register [post]
func (h *Handler) Register(c *fiber.Ctx) error {
	var payload struct {
		User
		Password string `json:"password"`
	}

	if err := c.BodyParser(&payload); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	newUser, err := h.service.Register(payload.User, payload.Password)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "duplicate") {
			// Auto-login fallback for already registered users
			tokenResp, loginErr := h.service.Login(LoginRequest{
				Email:    payload.User.Email,
				Password: payload.Password,
			})
			if loginErr != nil {
				return response.Error(c, fiber.StatusUnauthorized, "LOGIN_FAILED", "User already exists, but login failed: "+loginErr.Error())
			}
			return response.JSON(c, fiber.StatusOK, tokenResp, "User already registered. Logged in successfully.")
		}
		return response.Error(c, fiber.StatusInternalServerError, "REGISTRATION_FAILED", "Could not register user")
	}

	// Send WhatsApp welcome (phone may be set)
	if newUser.PhoneNumber != "" {
		notification.WhatsAppWelcome(newUser.PhoneNumber, newUser.FullName, newUser.Role)
	}

	// Auto-login
	tokenResp, err := h.service.Login(LoginRequest{
		Email:    newUser.Email,
		Password: payload.Password,
	})
	if err != nil {
		// Registration succeeded but auto-login failed — tell client to log in manually
		return response.JSON(c, fiber.StatusCreated, fiber.Map{
			"message": "Registration successful. Please log in.",
			"user_id": newUser.ID,
		}, "Registered")
	}

	// Store refresh token in Redis for rotation on next refresh / invalidation on logout
	if tokenResp.RefreshToken != "" {
		if err := tokens.StoreRefreshToken(c.Context(), newUser.ID, tokenResp.RefreshToken); err != nil {
			log.Printf("[auth] failed to store refresh token for user %d: %v", newUser.ID, err)
		}
	}

	activity_log.LogAuthSuccess(c, newUser.ID, activity_log.ActionRegister, map[string]interface{}{
		"role":  newUser.Role,
		"email": newUser.Email,
	})

	return response.JSON(c, fiber.StatusCreated, tokenResp, "Registered successfully")
}

// Logout invalidates the refresh token in Redis so it cannot be replayed.
func (h *Handler) Logout(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(uint)
	if userID != 0 {
		tokens.RevokeRefreshToken(c.Context(), userID)
		activity_log.LogAuthSuccess(c, userID, activity_log.ActionLogout, nil)
	}
	return response.JSON(c, fiber.StatusOK, nil, "Logged out successfully")
}

// RefreshToken validates a refresh token, rotates it (one-time use), and
// returns a new access token + new refresh token.
func (h *Handler) RefreshToken(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&req); err != nil || req.RefreshToken == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "refresh_token required")
	}

	// Validate the JWT signature and expiry.
	claims, err := tokens.ValidateToken(req.RefreshToken)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "INVALID_TOKEN", "Invalid or expired refresh token")
	}

	if claims.Type != "refresh" {
		return response.Error(c, fiber.StatusUnauthorized, "INVALID_TOKEN", "Token is not a refresh token")
	}

	// Parse userID from the string claim.
	var userID uint
	fmt.Sscanf(claims.UserID, "%d", &userID)
	if userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "INVALID_TOKEN", "Invalid token subject")
	}

	// Validate and rotate in Redis (one-time use).
	if err := tokens.ValidateAndRotateRefreshToken(c.Context(), userID, req.RefreshToken); err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "TOKEN_REVOKED", err.Error())
	}

	// Issue a new access token + a new refresh token.
	newAccess, err := tokens.CreateAccessToken(claims.UserID, claims.Role, claims.HospitalID, nil)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "TOKEN_ERROR", "Failed to create access token")
	}
	newRefresh, err := tokens.CreateRefreshToken(claims.UserID, claims.Role, claims.HospitalID, nil)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "TOKEN_ERROR", "Failed to create refresh token")
	}

	// Store the new refresh token in Redis.
	if err := tokens.StoreRefreshToken(c.Context(), userID, newRefresh); err != nil {
		log.Printf("[auth] failed to store rotated refresh token for user %d: %v", userID, err)
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"access_token":  newAccess,
		"refresh_token": newRefresh,
	}, "Token refreshed")
}

// GetProfile handles fetching user profile details
// @Summary Get User Profile
// @Description Fetch current authenticated user profile
// @Tags Profile
// @Accept json
// @Produce json
// @Success 200 {object} response.StandardResponse{data=identity.User} "Profile details"
// @Failure 401 {object} response.StandardError "Unauthorized"
// @Failure 404 {object} response.StandardError "User not found"
// @Router /api/v1/auth/profile [get]
func (h *Handler) GetProfile(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		// Fallback: user_id_str is set when user_id string couldn't be parsed as uint
		if str, _ := c.Locals("user_id_str").(string); str != "" {
			if id, err := strconv.ParseUint(str, 10, 32); err == nil {
				userID = uint(id)
			}
		}
		if userID == 0 {
			return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		}
	}

	user, err := h.service.GetProfile(userID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "USER_NOT_FOUND", "User not found")
	}

	return response.JSON(c, fiber.StatusOK, user, "Profile fetched successfully")
}

// UpdateProfile updates user profile details
// @Summary Update User Profile
// @Description Update profile details of authenticated user
// @Tags Profile
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "Profile Updates"
// @Success 200 {object} response.StandardResponse "Profile updated successfully"
// @Failure 401 {object} response.StandardError "Unauthorized"
// @Failure 500 {object} response.StandardError "Update failed"
// @Router /api/v1/auth/profile [put]
func (h *Handler) UpdateProfile(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	if err := h.service.UpdateProfile(userID, updates); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Profile updated successfully")
}

func (h *Handler) GetDoctors(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	spec := c.Query("specialization")
	verified := c.Query("verified") == "true"

	// Real COUNT query so pagination meta is accurate
	countQ := database.DB.Table(`"user"`).
		Joins(`JOIN doctorprofile ON "user".id = doctorprofile.user_id`).
		Where(`"user".role = ?`, "doctor").
		Where(`"user".is_active = ?`, true)
	if spec != "" {
		countQ = countQ.Where("LOWER(doctorprofile.specialization) = LOWER(?)", spec)
	}
	if verified {
		countQ = countQ.Where("doctorprofile.is_verified = ?", true)
	}
	var total int64
	countQ.Count(&total)

	// Fetch filtered list, then apply in-memory pagination slice
	doctors, err := h.service.ListDoctors(spec, verified)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", err.Error())
	}

	start := offset
	if start > len(doctors) {
		start = len(doctors)
	}
	end := start + limit
	if end > len(doctors) {
		end = len(doctors)
	}
	pageDoctors := doctors[start:end]

	// Synchronize with real-time Redis presence
	for i := range pageDoctors {
		pageDoctors[i].IsOnline = presence.IsDoctorOnline(context.Background(), fmt.Sprintf("%d", pageDoctors[i].ID))
	}

	meta := response.NewPaginationMeta(total, page, limit)

	return response.JSON(c, fiber.StatusOK, pageDoctors, "Doctors listed successfully", meta)
}

func (h *Handler) GetDoctorDetails(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid doctor ID")
	}

	user, err := h.service.GetProfile(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "DOCTOR_NOT_FOUND", "Doctor not found")
	}

	profile, _ := h.service.GetDoctorProfile(uint(id))
	if profile != nil {
		// Sync is_online with real-time Redis presence
		profile.IsOnline = presence.IsDoctorOnline(context.Background(), fmt.Sprintf("%d", profile.UserID))
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{"user": user, "profile": profile}, "Doctor details fetched")
}

func (h *Handler) GetDoctorAvailability(c *fiber.Ctx) error {
	idStr := c.Params("id")
	var id uint
	if idStr == "" || idStr == "me" {
		id, _ = c.Locals("user_id").(uint)
	} else {
		uID, _ := strconv.ParseUint(idStr, 10, 32)
		id = uint(uID)
	}

	if id == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Doctor ID required")
	}

	availability, err := h.service.GetAvailability(id)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, availability, "Availability fetched")
}

func (h *Handler) UpdateDoctorProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	// Admin path
	if role == "admin" || role == "super_admin" || role == "hospital_admin" {
		idStr := c.Params("id")
		targetID, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil || targetID == 0 {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid doctor ID")
		}
		var req DoctorAdminUpdateRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		}
		if err := h.service.AdminUpdateDoctorProfile(userID, uint(targetID), req); err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		}
		return response.JSON(c, fiber.StatusOK, nil, "Doctor profile updated by admin")
	}

	// Doctor self-update path
	var req DoctorSelfUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}
	if err := h.service.UpdateDoctorProfileTyped(userID, req); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Doctor profile updated")
}

func (h *Handler) AddDoctorAvailability(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(uint)
	if userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	var slot DoctorAvailability
	if err := c.BodyParser(&slot); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	slot.DoctorID = userID
	// Reuse UpdateAvailability with a single item for now, or implement Add in repo
	current, _ := h.service.GetAvailability(userID)
	current = append(current, slot)

	if err := h.service.UpdateAvailability(userID, current); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ADD_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, slot, "Slot added successfully")
}

func (h *Handler) DeleteDoctorAvailability(c *fiber.Ctx) error {
	idStr := c.Params("id")
	slotID, _ := strconv.ParseUint(idStr, 10, 32)
	userID, _ := c.Locals("user_id").(uint)

	if userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	// Fetch current to find and remove
	current, _ := h.service.GetAvailability(userID)
	var updated []DoctorAvailability
	found := false
	for _, s := range current {
		if s.ID == uint(slotID) {
			found = true
			continue
		}
		updated = append(updated, s)
	}

	if !found {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Availability slot not found")
	}

	if err := h.service.UpdateAvailability(userID, updated); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Slot deleted successfully")
}

func (h *Handler) UpdateDoctorAvailability(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid doctor ID")
	}

	// Ownership check
	callerID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	if callerID != uint(id) && role != "admin" && role != "super_admin" {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Cannot modify another doctor's availability")
	}

	// Parse into validated slot struct (not raw DoctorAvailability)
	var slots []DoctorAvailabilitySlot
	if err := c.BodyParser(&slots); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	// Validate each slot
	for i, s := range slots {
		if s.DayOfWeek < 0 || s.DayOfWeek > 6 {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_DAY",
				fmt.Sprintf("slot %d: day_of_week must be 0–6", i))
		}
		startT, err := time.Parse("15:04", s.StartTime)
		if err != nil {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_TIME",
				fmt.Sprintf("slot %d: invalid start_time format, expected HH:MM", i))
		}
		endT, err := time.Parse("15:04", s.EndTime)
		if err != nil {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_TIME",
				fmt.Sprintf("slot %d: invalid end_time format, expected HH:MM", i))
		}
		if !endT.After(startT) {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_TIME_RANGE",
				fmt.Sprintf("slot %d: end_time must be after start_time", i))
		}
	}

	// Map validated slots to DB models
	schedules := make([]DoctorAvailability, len(slots))
	for i, s := range slots {
		schedules[i] = DoctorAvailability{
			DoctorID:    uint(id),
			DayOfWeek:   s.DayOfWeek,
			StartTime:   s.StartTime,
			EndTime:     s.EndTime,
			IsAvailable: s.IsAvailable,
		}
	}

	if err := h.service.UpdateAvailability(uint(id), schedules); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Availability updated successfully")
}

// GetFamilyMembers lists family members
// @Summary List Family Members
// @Description Fetch list of family members for the authenticated user
// @Tags Family
// @Accept json
// @Produce json
// @Success 200 {object} response.StandardResponse{data=[]identity.FamilyMember} "List of family members"
// @Failure 401 {object} response.StandardError "Unauthorized"
// @Failure 500 {object} response.StandardError "Fetch failed"
// @Router /api/v1/family/members [get]
func (h *Handler) GetFamilyMembers(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	}

	members, err := h.service.GetFamilyMembers(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, members, "Family members fetched")
}

// AddFamilyMember adds a family member
// @Summary Add Family Member
// @Description Add a new family member to user account
// @Tags Family
// @Accept json
// @Produce json
// @Param request body identity.FamilyMember true "Family Member Details"
// @Success 201 {object} response.StandardResponse{data=identity.FamilyMember} "Family member added"
// @Failure 401 {object} response.StandardError "Unauthorized"
// @Failure 500 {object} response.StandardError "Add failed"
// @Router /api/v1/family/members [post]
func (h *Handler) AddFamilyMember(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	}

	var member FamilyMember
	if err := c.BodyParser(&member); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	member.UserID = userID
	if err := h.service.AddFamilyMember(member); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ADD_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, member, "Family member added")
}

// UpdateFamilyMember updates a family member
// @Summary Update Family Member
// @Description Update specific family member details
// @Tags Family
// @Accept json
// @Produce json
// @Param id path int true "Family Member ID"
// @Param request body map[string]interface{} true "Updates"
// @Success 200 {object} response.StandardResponse "Member updated successfully"
// @Failure 401 {object} response.StandardError "Unauthorized"
// @Failure 500 {object} response.StandardError "Update failed"
// @Router /api/v1/family/members/{id} [put]
func (h *Handler) UpdateFamilyMember(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid member ID")
	}
	userID := c.Locals("user_id").(uint)

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	if err := h.service.UpdateFamilyMember(uint(id), userID, updates); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Member updated successfully")
}

// DeleteFamilyMember deletes a family member
// @Summary Delete Family Member
// @Description Delete a family member from user account
// @Tags Family
// @Accept json
// @Produce json
// @Param id path int true "Family Member ID"
// @Success 200 {object} response.StandardResponse "Member deleted successfully"
// @Failure 401 {object} response.StandardError "Unauthorized"
// @Failure 500 {object} response.StandardError "Delete failed"
// @Router /api/v1/family/members/{id} [delete]
func (h *Handler) DeleteFamilyMember(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid member ID")
	}
	userID := c.Locals("user_id").(uint)

	if err := h.service.DeleteFamilyMember(uint(id), userID); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Member deleted successfully")
}

// SendOTP handles POST /auth/send-otp
// Generates a 6-digit OTP for the given phone number and returns it
// (development mode). In production this would trigger an SMS gateway.
func (h *Handler) SendOTP(c *fiber.Ctx) error {
	var req struct {
		Phone string `json:"phone"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if req.Phone == "" {
		return response.Error(c, fiber.StatusBadRequest, "PHONE_REQUIRED", "Phone number is required")
	}

	if _, err := h.service.SendOTP(req.Phone); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "OTP_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"message": "OTP sent successfully",
	}, "OTP sent")
}

// LoginWithOTP handles POST /auth/otp-login
// Validates phone+OTP and returns access+refresh tokens.
func (h *Handler) LoginWithOTP(c *fiber.Ctx) error {
	var req struct {
		Phone string `json:"phone"`
		OTP   string `json:"otp"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if req.Phone == "" || req.OTP == "" {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELDS", "Phone and OTP are required")
	}

	resp, err := h.service.LoginWithOTP(req.Phone, req.OTP)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, "LOGIN_FAILED", err.Error())
	}

	if resp.User != nil && resp.RefreshToken != "" {
		if err := tokens.StoreRefreshToken(c.Context(), resp.User.ID, resp.RefreshToken); err != nil {
			log.Printf("[auth] failed to store refresh token for user %d: %v", resp.User.ID, err)
		}
	}

	return response.JSON(c, fiber.StatusOK, resp, "Login successful")
}

func (h *Handler) ChangePassword(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.BodyParser(&req); err != nil || req.CurrentPassword == "" || req.NewPassword == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "current_password and new_password are required")
	}
	if len(req.NewPassword) < 8 {
		return response.Error(c, fiber.StatusBadRequest, "WEAK_PASSWORD", "Password must be at least 8 characters")
	}
	if err := h.service.ChangePassword(userID, req.CurrentPassword, req.NewPassword); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CHANGE_FAILED", err.Error())
	}
	// Clear the force-change flag so the user is not redirected again on next login.
	database.DB.WithContext(c.UserContext()).Model(&User{}).Where("id = ?", userID).Update("must_change_password", false)
	return response.JSON(c, fiber.StatusOK, nil, "Password changed successfully")
}

func (h *Handler) ForgotPassword(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&req); err != nil || req.Email == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Email is required")
	}
	// We only return an error if email sending actually failed.
	// If the user doesn't exist, service.ForgotPassword returns nil to prevent enumeration.
	if err := h.service.ForgotPassword(req.Email); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "EMAIL_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "If that email exists, a reset link has been sent")
}

func (h *Handler) ResetPassword(c *fiber.Ctx) error {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := c.BodyParser(&req); err != nil || req.Token == "" || req.NewPassword == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "token and new_password are required")
	}
	if len(req.NewPassword) < 8 {
		return response.Error(c, fiber.StatusBadRequest, "WEAK_PASSWORD", "Password must be at least 8 characters")
	}
	if err := h.service.ResetPassword(req.Token, req.NewPassword); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "RESET_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Password reset successfully")
}

func (h *Handler) VerifyEmail(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_TOKEN", "Verification token is required")
	}
	if err := h.service.VerifyEmail(token); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "VERIFY_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Email verified successfully")
}

func (h *Handler) ResendVerification(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	if err := h.service.SendVerificationEmail(userID); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SEND_FAILED", "Failed to send verification email")
	}
	return response.JSON(c, fiber.StatusOK, nil, "Verification email sent")
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	// Auth
	authGroup := router.Group("/auth")
	authGroup.Post("/login", middleware.AuthLimiter(), h.Login)
	authGroup.Post("/send-otp", middleware.AuthLimiter(), h.SendOTP)
	authGroup.Post("/otp-login", middleware.AuthLimiter(), h.LoginWithOTP)
	authGroup.Post("/logout", middleware.Protected(), h.Logout)
	authGroup.Post("/staff/login", middleware.AuthLimiter(), h.StaffLogin)
	authGroup.Post("/mfa/verify", middleware.AuthLimiter(), middleware.Protected(), h.VerifyMFA)
	authGroup.Post("/mfa/resend", middleware.AuthLimiter(), middleware.Protected(), h.ResendMFA)
	authGroup.Post("/register", h.Register)
	authGroup.Post("/refresh-token", h.RefreshToken)
	authGroup.Get("/profile", middleware.Protected(), h.GetProfile)
	authGroup.Get("/me", middleware.Protected(), h.GetProfile)
	authGroup.Put("/profile", middleware.Protected(), h.UpdateProfile)
	authGroup.Post("/change-password", middleware.Protected(), h.ChangePassword)
	authGroup.Post("/forgot-password", middleware.AuthLimiter(), h.ForgotPassword)
	authGroup.Post("/reset-password", middleware.AuthLimiter(), h.ResetPassword)
	authGroup.Get("/verify-email", h.VerifyEmail)
	authGroup.Post("/resend-verification", middleware.Protected(), h.ResendVerification)

	// Doctor routes are owned exclusively by handlers.RegisterDoctorRoutes (handlers/doctors.go).
	// Registering them here too caused Fiber to use the identity versions (registered first)
	// and the /:id wildcard to swallow all /doctors/views/* static routes → 404 on every
	// doctor list call. All doctor browsing, availability, and rating routes live in the
	// handlers package now. Do NOT add doctor routes back here.

	// Patient
	patientGroup := router.Group("/patient", middleware.Protected())
	patientGroup.Get("/profile", func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(uint)
		profile, err := h.service.GetPatientProfile(userID)
		if err != nil {
			return response.Error(c, fiber.StatusNotFound, "PROFILE_NOT_FOUND", "Profile not found")
		}
		return response.JSON(c, fiber.StatusOK, profile, "Patient profile fetched")
	})

	// Family
	familyGroup := router.Group("/family", middleware.Protected())
	familyGroup.Get("/members", h.GetFamilyMembers)
	familyGroup.Post("/members", h.AddFamilyMember)
	familyGroup.Get("/members/:id", func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil || id == 0 {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid member ID")
		}
		userID := c.Locals("user_id").(uint)
		member, err := h.service.GetFamilyMember(uint(id), userID)
		if err != nil {
			return response.Error(c, fiber.StatusNotFound, "MEMBER_NOT_FOUND", "Member not found")
		}
		return response.JSON(c, fiber.StatusOK, member, "Family member fetched")
	})
	familyGroup.Put("/members/:id", h.UpdateFamilyMember)
	familyGroup.Delete("/members/:id", h.DeleteFamilyMember)

	// UHEID Portability
	uheidGroup := router.Group("/uheid")
	uheidGroup.Get("/:uheid/profile", h.GetUHEIDProfile)
	authGroup.Post("/consent", middleware.Protected(), h.GrantConsent)
}

func (h *Handler) RateDoctor(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid doctor ID")
	}

	var req struct {
		Rating float64 `json:"rating"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	if err := h.service.RateDoctor(uint(id), req.Rating); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "RATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Doctor rated successfully")
}

func (h *Handler) GrantConsent(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var req struct {
		ProviderID   uint         `json:"provider_id"`
		Scope        ConsentScope `json:"scope"`
		DurationDays int          `json:"duration_days"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
	}

	if err := h.service.GrantConsent(userID, req.ProviderID, req.Scope, req.DurationDays); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CONSENT_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Consent granted successfully")
}

func (h *Handler) GetUHEIDProfile(c *fiber.Ctx) error {
	uheid := c.Params("uheid")
	apiKey := c.Get("X-Provider-API-Key")
	scope := ConsentScope(c.Query("scope", string(ScopeUHEIDOnly)))

	if apiKey == "" {
		return response.Error(c, fiber.StatusUnauthorized, "MISSING_API_KEY", "X-Provider-API-Key header required")
	}

	profile, err := h.service.GetUHEIDProfile(uheid, apiKey, scope)
	if err != nil {
		return response.Error(c, fiber.StatusForbidden, "ACCESS_DENIED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, profile, "UHEID profile fetched")
}

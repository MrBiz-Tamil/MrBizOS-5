package admin

import (
	"fmt"
	"log"
	"strings"
	"time"
	"drayya-core/internal/auth"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/services/activity_log"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

type UserReq struct {
	identity.User
	Password string `json:"password"`
}

// GetUsers lists users with pagination and filtering
// GetUsers lists users with pagination and filtering
// @Summary List Users
// @Description Fetch a list of all users with optional role filtering and name search
// @Tags Admin Users
// @Produce json
// @Security ApiKeyAuth
// @Param role query string false "Filter by role"
// @Param q query string false "Search by name or email"
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} []identity.User "Users fetched"
// @Router /api/v1/admin/users [get]
func GetUsers(c *fiber.Ctx) error {
	db := database.DB
	var users []identity.User
	var total int64

	// Filtering
	query := db.Model(&identity.User{})
	if role := c.Query("role"); role != "" {
		query = query.Where("role = ?", role)
	}
	if search := c.Query("q"); search != "" {
		s := "%" + search + "%"
		query = query.Where("LOWER(full_name) LIKE LOWER(?) OR LOWER(email) LIKE LOWER(?)", s, s)
	}

	if err := query.Count(&total).Error; err != nil {
		log.Printf("[ERROR] GetUsers count failed: %v", err)
		return response.Error(c, fiber.StatusInternalServerError, "USER_COUNT_FAILED", "Failed to count users: "+err.Error())
	}

	// Pagination
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 1000)
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	if err := query.Order("id desc").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		log.Printf("[ERROR] GetUsers find failed: %v", err)
		return response.Error(c, fiber.StatusInternalServerError, "USER_FETCH_FAILED", "Failed to fetch users: "+err.Error())
	}

	return c.JSON(users)
}

// GetActivityLogs lists logs
// GetActivityLogs lists logs
// @Summary Get Activity Logs
// @Description Fetch a paginated list of system activity and audit logs
// @Tags Admin Users
// @Produce json
// @Security ApiKeyAuth
// @Param user_id query string false "Filter by user ID"
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} map[string]interface{} "Logs fetched"
// @Router /api/v1/admin/activity-logs [get]
func GetActivityLogs(c *fiber.Ctx) error {
	db := database.DB
	var logs []identity.ActivityLog
	var total int64

	query := db.Model(&identity.ActivityLog{}).Preload("User")

	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	query.Count(&total)

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	offset := (page - 1) * limit

	query.Order("created_at desc").Offset(offset).Limit(limit).Find(&logs)

	if logs == nil {
		logs = []identity.ActivityLog{}
	}
	return c.JSON(fiber.Map{
		"logs":  logs,
		"total": total,
	})
}

// ToggleUserStatus activates or deactivates a user
// ToggleUserStatus activates or deactivates a user
// @Summary Toggle User Status
// @Description Activate or deactivate a user account by ID
// @Tags Admin Users
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "User ID"
// @Param action path string true "Action (activate/deactivate)"
// @Success 200 {object} map[string]interface{} "Status updated"
// @Router /api/v1/admin/users/{id}/{action} [post]
func ToggleUserStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	action := c.Params("action") // activate or deactivate

	db := database.DB
	var user identity.User
	if err := db.First(&user, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "USER_NOT_FOUND", "User not found")
	}

	isActive := action == "activate"
	user.IsActive = isActive
	db.Save(&user)

	// Log Activity
	adminID := c.Locals("user_id").(uint)
	activity_log.LogAsync(adminID, action+"_user", "user", id, nil, c)

	return c.JSON(fiber.Map{"message": "User status updated", "user": user})
}


// CreateUsersBulk handles bulk user creation
// CreateUsersBulk handles bulk user creation
// @Summary Bulk Create Users
// @Description Register multiple users at once (Batch process)
// @Tags Admin Users
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param users body []UserReq true "List of users to create"
// @Success 200 {object} map[string]interface{} "Result summary"
// @Router /api/v1/admin/users/bulk [post]
func CreateUsersBulk(c *fiber.Ctx) error {
	var reqs []UserReq
	if err := c.BodyParser(&reqs); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	log.Printf("     Bulk Create Users: Processing %d requests", len(reqs))
	adminID := c.Locals("user_id").(uint)
	createdCount := 0
	failedCount := 0
	type failedUser struct {
		Email  string `json:"email"`
		Reason string `json:"reason"`
	}
	failedUsers := []failedUser{}

	for _, req := range reqs {
		user := req.User
		password := req.Password

		newUser, err := auth.RegisterUser(c.UserContext(), user, password)
		if err != nil {
			log.Printf("    Failed to register user %s: %v", user.Email, err)
			failedCount++
			failedUsers = append(failedUsers, failedUser{
				Email:  user.Email,
				Reason: err.Error(),
			})
			continue
		}

		createdCount++
		// Log Activity for each user created
		activity_log.LogAsync(adminID, "bulk_create_user", "user", fmt.Sprintf("%d", newUser.ID), nil, c)
	}

	log.Printf("    Bulk Create Finished: Created %d, Failed %d", createdCount, failedCount)
	return c.JSON(fiber.Map{
		"created_count": createdCount,
		"failed_count":  failedCount,
		"failed_users":  failedUsers,
	})
}

// GetUserDetails returns a single user with their doctor profile and recent appointments
// @Summary Get User Details
// @Description Fetch full profile for a user including doctor credentials and appointment history
// @Tags Admin Users
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "User ID"
// @Success 200 {object} map[string]interface{} "User details"
// @Router /api/v1/admin/users/{id} [get]
func GetUserDetails(c *fiber.Ctx) error {
	db := database.DB
	id := c.Params("id")

	var user identity.User
	if err := db.First(&user, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "USER_NOT_FOUND", "User not found")
	}

	// Doctor profile — only populated for doctor role
	var profile *identity.DoctorProfile
	if user.Role == "doctor" || user.Role == "chief_doctor" || user.Role == "hospital_head_doctor" {
		var dp identity.DoctorProfile
		if err := db.Where("user_id = ?", user.ID).First(&dp).Error; err == nil {
			profile = &dp
		}
	}

	// Recent appointments — last 5, with the "other party" name resolved
	type RecentAppointment struct {
		ID             uint      `json:"id"`
		OtherPartyName string    `json:"other_party_name"`
		Date           time.Time `json:"date"`
		Status         string    `json:"status"`
	}
	var recentAppointments []RecentAppointment

	if user.Role == "doctor" || user.Role == "chief_doctor" || user.Role == "hospital_head_doctor" {
		db.Table("appointments").
			Select("appointments.id, COALESCE(u.full_name, 'Unknown Patient') AS other_party_name, appointments.start_time AS date, appointments.status").
			Joins("LEFT JOIN users u ON u.id = appointments.patient_id").
			Where("appointments.doctor_id = ?", user.ID).
			Order("appointments.start_time DESC").
			Limit(5).
			Scan(&recentAppointments)
	} else {
		db.Table("appointments").
			Select("appointments.id, COALESCE(u.full_name, 'Unknown Doctor') AS other_party_name, appointments.start_time AS date, appointments.status").
			Joins("LEFT JOIN users u ON u.id = appointments.doctor_id").
			Where("appointments.patient_id = ?", user.ID).
			Order("appointments.start_time DESC").
			Limit(5).
			Scan(&recentAppointments)
	}

	if recentAppointments == nil {
		recentAppointments = []RecentAppointment{}
	}

	return c.JSON(fiber.Map{
		"user":                user,
		"profile":             profile,
		"recent_appointments": recentAppointments,
	})
}

// CreateUser handles single user creation
// CreateUser handles single user creation
// @Summary Create Single User
// @Description Register a new user with administrative privileges
// @Tags Admin Users
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param user body UserReq true "User data"
// @Success 200 {object} map[string]interface{} "User created"
// @Router /api/v1/admin/users [post]
func CreateUser(c *fiber.Ctx) error {
	var req UserReq
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	newUser, err := auth.RegisterUser(c.UserContext(), req.User, req.Password)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "USER_CREATION_FAILED", err.Error())
	}

	adminID := c.Locals("user_id").(uint)
	activity_log.LogAsync(adminID, "create_user", "user", fmt.Sprintf("%d", newUser.ID), nil, c)

	return c.JSON(fiber.Map{
		"message": "User created successfully",
		"user":    newUser,
	})
}

// GetActivityLogsStats returns activity log stats
// GetActivityLogsStats returns activity log stats
// @Summary Get Activity Log Stats
// @Description Fetch metrics on total logs, active users, and system actions
// @Tags Admin Users
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "Stats fetched"
// @Router /api/v1/admin/activity-logs/stats [get]
func GetActivityLogsStats(c *fiber.Ctx) error {
	var total, todayCount int64
	database.DB.Model(&identity.ActivityLog{}).Count(&total)
	today := time.Now().Truncate(24 * time.Hour)
	database.DB.Model(&identity.ActivityLog{}).Where("created_at >= ?", today).Count(&todayCount)
	return c.JSON(fiber.Map{
		"total_logs":       total,
		"today":            todayCount,
		"active_users":     0,
		"failed_logins":    0,
		"critical_actions": 0,
	})
}

// UpdateUser edits an existing user's profile fields.
// @Summary Update User
// @Tags Admin Users
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "User ID"
// @Router /api/v1/admin/users/{id} [put]
func UpdateUser(c *fiber.Ctx) error {
	id := c.Params("id")
	db := database.DB

	var user identity.User
	if err := db.First(&user, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "USER_NOT_FOUND", "User not found")
	}

	type UpdateReq struct {
		FullName       string `json:"full_name"`
		Email          string `json:"email"`
		PhoneNumber    string `json:"phone_number"`
		WhatsAppNumber string `json:"whatsapp_number"`
		Address        string `json:"address"`
		Role           string `json:"role"`
		HospitalID     *uint  `json:"hospital_id"`
		PharmacyID     *uint  `json:"pharmacy_id"`
	}

	var req UpdateReq
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	// Validate email uniqueness if it changed
	if req.Email != "" && !strings.EqualFold(req.Email, user.Email) {
		var count int64
		db.Model(&identity.User{}).Where("email = ? AND id <> ?", req.Email, user.ID).Count(&count)
		if count > 0 {
			return response.Error(c, fiber.StatusConflict, "EMAIL_EXISTS", "Email is already in use by another account")
		}
		user.Email = req.Email
	}

	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.PhoneNumber != "" {
		user.PhoneNumber = req.PhoneNumber
	}
	user.WhatsAppNumber = req.WhatsAppNumber
	user.Address = req.Address

	if req.Role != "" {
		user.Role = req.Role
	}
	user.HospitalID = req.HospitalID

	if err := db.WithContext(c.UserContext()).Save(&user).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update user: "+err.Error())
	}

	adminID := c.Locals("user_id").(uint)
	activity_log.LogAsync(adminID, "update_user", "user", id, fiber.Map{
		"updated_fields": req,
	}, c)

	return response.JSON(c, fiber.StatusOK, user, "User updated successfully")
}

// DeleteUser soft-deletes a user account.
// @Summary Delete User
// @Tags Admin Users
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "User ID"
// @Router /api/v1/admin/users/{id} [delete]
func DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")
	adminID := c.Locals("user_id").(uint)
	db := database.DB

	var user identity.User
	if err := db.First(&user, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "USER_NOT_FOUND", "User not found")
	}

	// Prevent self-deletion
	if user.ID == adminID {
		return response.Error(c, fiber.StatusForbidden, "SELF_DELETE", "Cannot delete your own account")
	}
	// Prevent deleting super_admin
	if user.Role == "super_admin" {
		return response.Error(c, fiber.StatusForbidden, "PROTECTED_ROLE", "Cannot delete a super_admin account")
	}

	if err := db.WithContext(c.UserContext()).Delete(&user).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", "Failed to delete user: "+err.Error())
	}

	activity_log.LogAsync(adminID, "delete_user", "user", id, nil, c)
	return response.JSON(c, fiber.StatusOK, nil, "User deleted successfully")
}

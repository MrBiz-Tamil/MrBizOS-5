package admin

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"drayya-core/internal/auth"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/hr"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/services/activity_log"
	"drayya-core/internal/services/email"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// HireStaff directly hires a SALARIED employee (nurse, lab tech, accountant, ops,
// receptionist, …) — the front door the salaried track was missing. Freelance /
// online doctors are NOT hired here; they go through recruitment → clinical
// verification. Doctors get a StaffProfile auto-created by RegisterUser, but every
// other staff role does not — so a salaried hire would otherwise be invisible to
// payroll and attendance. This creates the user, a salaried StaffProfile, and an
// onboarding record, then emails login credentials.
//
// POST /api/v1/admin/hr/staff   (HRManagerOnly, applied at the group level)
func HireStaff(c *fiber.Ctx) error {
	var req struct {
		FullName         string  `json:"full_name"`
		Email            string  `json:"email"`
		Phone            string  `json:"phone"`
		Role             string  `json:"role"`
		Designation      string  `json:"designation"`
		Department       string  `json:"department"`
		BaseSalaryRupees float64 `json:"base_salary_rupees"`
		JoiningDate      string  `json:"joining_date"` // YYYY-MM-DD, optional
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Role = strings.TrimSpace(req.Role)
	if req.FullName == "" || req.Email == "" || req.Role == "" {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELDS", "full_name, email and role are required")
	}
	// Direct-hire is for salaried staff only. Doctors must go through recruitment
	// + clinical verification; patients are not staff.
	if req.Role == string(identity.RoleDoctor) || req.Role == string(identity.RolePatient) {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ROLE",
			"Doctors are hired via recruitment; patients are not staff. Pick a salaried staff role.")
	}

	var existing identity.User
	if err := database.DB.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		return response.Error(c, fiber.StatusConflict, "EMAIL_EXISTS",
			fmt.Sprintf("A user already exists with email %s", req.Email))
	}

	tempPassword := generateTempPassword()
	created, err := auth.RegisterUser(c.UserContext(), identity.User{
		FullName:           req.FullName,
		Email:              req.Email,
		PhoneNumber:        req.Phone,
		Role:               req.Role,
		IsActive:           true,
		IsVerified:         true, // salaried staff are trusted on creation — no clinical gate
		MustChangePassword: true,
	}, tempPassword)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "HIRE_FAILED", err.Error())
	}

	joining := time.Now()
	if req.JoiningDate != "" {
		if t, e := time.Parse("2006-01-02", req.JoiningDate); e == nil {
			joining = t
		}
	}
	designation := req.Designation
	if designation == "" {
		designation = req.Role
	}

	// RegisterUser only auto-creates a StaffProfile for doctors. Upsert one here so
	// the salaried hire shows up in payroll/attendance.
	var sp hr.StaffProfile
	if err := database.DB.Where("user_id = ?", created.ID).First(&sp).Error; err != nil {
		sp = hr.StaffProfile{
			UserID:           created.ID,
			EmployeeID:       fmt.Sprintf("EMP-%d", created.ID),
			EmploymentType:   hr.EmploymentSalaried,
			Department:       req.Department,
			Designation:      designation,
			BaseSalary:       int64(req.BaseSalaryRupees * 100),
			JoiningDate:      joining,
			EmploymentStatus: "active",
		}
		if cErr := database.DB.WithContext(c.UserContext()).Create(&sp).Error; cErr != nil {
			return response.Error(c, fiber.StatusInternalServerError, "STAFF_PROFILE_FAILED", cErr.Error())
		}
	} else {
		database.DB.WithContext(c.UserContext()).Model(&hr.StaffProfile{}).Where("id = ?", sp.ID).
			Updates(map[string]interface{}{
				"employment_type":   hr.EmploymentSalaried,
				"department":        req.Department,
				"designation":       designation,
				"base_salary":       int64(req.BaseSalaryRupees * 100),
				"joining_date":      joining,
				"employment_status": "active",
			})
	}

	// Open an onboarding record so the new hire appears in the onboarding kanban.
	onboard := hr.OnboardingRecord{
		StaffID:          sp.ID,
		Status:           "pending_docs",
		ProbationEndDate: joining.AddDate(0, 3, 0),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	if oErr := database.DB.WithContext(c.UserContext()).Create(&onboard).Error; oErr != nil {
		log.Printf("[hire] warn: onboarding record creation failed for staff %d: %v", sp.ID, oErr)
	}

	if adminID, ok := c.Locals("user_id").(uint); ok {
		activity_log.LogAsync(adminID, "hire_staff", "user", fmt.Sprintf("%d", created.ID), nil, c)
	}

	// Email login credentials (best-effort).
	go func(to, name, pw, role string) {
		html := fmt.Sprintf(`<p>Hi %s,</p>
<p>Your Dr.Ayya staff account has been created.</p>
<p><b>Email:</b> %s<br/><b>Temporary password:</b> %s<br/><b>Role:</b> %s</p>
<p>Please log in and change your password on first sign-in.</p>`, name, to, pw, role)
		if e := email.SendSimple(to, "Your Dr.Ayya staff account", html); e != nil {
			log.Printf("[hire] credential email failed for %s: %v", to, e)
		}
	}(req.Email, req.FullName, tempPassword, req.Role)

	return response.JSON(c, fiber.StatusCreated, fiber.Map{
		"user_id":     created.ID,
		"employee_id": sp.EmployeeID,
		"email":       req.Email,
		"role":        req.Role,
	}, "Salaried staff hired")
}

// generateTempPassword returns a random, reasonably strong temporary password the
// new hire is forced to change on first login (MustChangePassword).
func generateTempPassword() string {
	b := make([]byte, 9)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("Mh!%d", time.Now().UnixNano())
	}
	return "Mh!" + base64.RawURLEncoding.EncodeToString(b)
}

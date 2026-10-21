package middleware

import (
	"drayya-core/internal/tokens"
	"drayya-core/internal/utils/response"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// parseJWT extracts and validates the Bearer token from the request, setting
// Fiber locals (user_id, role, hospital_id, pharmacy_id). It is idempotent —
// if Protected() already ran, the role local is already set and this is a no-op.
// Returns (tokenFound, validToken).
func parseJWT(c *fiber.Ctx) (found bool, valid bool) {
	// Already parsed by a prior middleware in the chain — skip.
	if _, alreadySet := c.Locals("role").(string); alreadySet {
		return true, true
	}

	authHeader := c.Get("Authorization")
	tokenString := ""
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			tokenString = parts[1]
		}
	}
	// Fallback to query param (WebSocket connections)
	if tokenString == "" {
		tokenString = c.Query("token")
	}
	if tokenString == "" {
		return false, false
	}

	claims, err := tokens.ValidateToken(tokenString)
	if err != nil {
		return true, false
	}

	id, err := strconv.ParseUint(claims.UserID, 10, 32)
	if err != nil {
		c.Locals("user_id_str", claims.UserID)
	} else {
		c.Locals("user_id", uint(id))
	}
	if claims.HospitalID != nil {
		c.Locals("hospital_id", *claims.HospitalID)
	}
	if claims.PharmacyID != nil {
		c.Locals("pharmacy_id", *claims.PharmacyID)
	}
	c.Locals("role", claims.Role)
	return true, true
}

// Protected validates the JWT and sets user locals. All protected routes must
// include this middleware (or a role middleware that calls parseJWT internally).
func Protected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		found, valid := parseJWT(c)
		if !found {
			return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid authorization")
		}
		if !valid {
			return response.Error(c, fiber.StatusUnauthorized, "TOKEN_EXPIRED", "Invalid or expired token")
		}
		return c.Next()
	}
}

// RequireRoles restricts access to the given roles. It also parses the JWT if
// Protected() was not chained before it, so role-only middlewares work standalone.
func RequireRoles(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		found, valid := parseJWT(c)
		if !found {
			return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid authorization")
		}
		if !valid {
			return response.Error(c, fiber.StatusUnauthorized, "TOKEN_EXPIRED", "Invalid or expired token")
		}

		userRole, ok := c.Locals("role").(string)
		if !ok || userRole == "" {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Access denied: role not found")
		}

		for _, role := range roles {
			if userRole == role || userRole == "admin" || userRole == "super_admin" {
				return c.Next()
			}
		}

		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Access denied: insufficient role")
	}
}

// AdminOnly restricts access to system administrators
func AdminOnly() fiber.Handler {
	return RequireRoles("admin")
}

// ManagerOnly restricts access to general managers
func ManagerOnly() fiber.Handler {
	return RequireRoles("manager", "admin")
}

// StaffOnly restricts access to all hospital staff (excludes patients)
func StaffOnly() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := c.Locals("role")
		if role == "patient" {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Staff access required")
		}
		return c.Next()
	}
}

// DoctorOnly restricts access to doctors and clinic leads
func DoctorOnly() fiber.Handler {
	return RequireRoles("doctor", "hospital_head_doctor", "chief_doctor", "admin")
}

// PatientOnly restricts access to patients
func PatientOnly() fiber.Handler {
	return RequireRoles("patient", "admin")
}

// ChiefDoctorOnly restricts access to clinical heads — both hospital head doctors
// (physical facility) and chief doctors (online network head). Previously this
// only allowed hospital_head_doctor which wrongly 403'd chief_doctor users.
func ChiefDoctorOnly() fiber.Handler {
	return RequireRoles("hospital_head_doctor", "chief_doctor", "admin")
}

// ClinicalHeadOnly is an alias for ChiefDoctorOnly for clearer intent.
func ClinicalHeadOnly() fiber.Handler {
	return ChiefDoctorOnly()
}

// OnlineChiefDoctorOnly restricts access to online leads
func OnlineChiefDoctorOnly() fiber.Handler {
	return RequireRoles("chief_doctor", "admin")
}

// HRManagerOnly restricts access to HR
func HRManagerOnly() fiber.Handler {
	return RequireRoles("hr_manager", "hr", "admin")
}

// FinanceManagerOnly restricts access to finance leads
func FinanceManagerOnly() fiber.Handler {
	return RequireRoles("finance_manager", "admin")
}

// PharmacyManagerOnly restricts access to pharmacists and pharmacy admins
func PharmacyManagerOnly() fiber.Handler {
	return RequireRoles("pharmacist", "pharmacy_admin", "admin")
}

// PharmacyAdminOnly restricts access to pharmacy administrators
func PharmacyAdminOnly() fiber.Handler {
	return RequireRoles("pharmacy_admin", "admin")
}

// SupplierOnly restricts access to registered suppliers
func SupplierOnly() fiber.Handler {
	return RequireRoles("supplier_agent", "admin")
}

// LogisticsOnly restricts access to logistics coordinators
func LogisticsOnly() fiber.Handler {
	return RequireRoles("logistics_coordinator", "admin")
}

// BPOOnly restricts access to BPO agents and dispatchers
func BPOOnly() fiber.Handler {
	return RequireRoles("bpo", "bpo_manager", "admin", "dispatcher")
}

// HospitalAdminOnly restricts access to hospital administrators
func HospitalAdminOnly() fiber.Handler {
	return RequireRoles("hospital_admin", "admin")
}

// NurseOnly restricts access to nurses
func NurseOnly() fiber.Handler {
	return RequireRoles("nurse", "admin")
}

// LabTechOnly restricts access to lab technicians
func LabTechOnly() fiber.Handler {
	return RequireRoles("lab_technician", "admin")
}

// QAManagerOnly restricts access to quality assurance managers
func QAManagerOnly() fiber.Handler {
	return RequireRoles("qa_manager", "admin")
}

// AccountantOnly restricts access to accountants and finance managers (read-only finance)
func AccountantOnly() fiber.Handler {
	return RequireRoles("accountant", "finance_manager", "admin")
}

// EnforceTenancy isolates data queries and actions for SaaS and Franchise tenants
func EnforceTenancy() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Attempt to parse JWT to extract user context/role if token is present
		parseJWT(c)

		role, _ := c.Locals("role").(string)
		if role == "super_admin" || role == "admin" {
			return c.Next()
		}

		userHospID, ok := c.Locals("hospital_id").(uint)
		if ok && userHospID > 0 {
			// Enforce matching ID parameter if present in routes (e.g., /hospitals/:id)
			if idParam := c.Params("id"); idParam != "" {
				reqID, err := strconv.ParseUint(idParam, 10, 32)
				if err == nil && uint(reqID) != userHospID {
					return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Access denied: cross-tenant boundary breach")
				}
			}

			// Enforce query parameter matching
			if c.Query("hospital_id") != "" {
				qID, err := strconv.ParseUint(c.Query("hospital_id"), 10, 32)
				if err == nil && uint(qID) != userHospID {
					return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Access denied: cross-tenant query parameter mismatch")
				}
			}
		}

		return c.Next()
	}
}

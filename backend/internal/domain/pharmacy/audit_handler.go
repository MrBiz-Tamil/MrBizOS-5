package pharmacy

import (
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// ═══════════════════════════════════════════════════════════════════════════
// AUDIT HANDLER
// HTTP endpoints for the Pharmacy Admin Audit System.
// ═══════════════════════════════════════════════════════════════════════════

type AuditHandler struct {
	svc AuditService
}

func NewAuditHandler(svc AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

// RunAudit executes the full audit and returns the structured report.
// GET /api/v1/pharmacy/audit/run
func (h *AuditHandler) RunAudit(c *fiber.Ctx) error {
	report, err := h.svc.RunFullAudit(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "AUDIT_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, report, "Audit completed")
}

// RunAuditByCategory runs only checks in a specific category.
// GET /api/v1/pharmacy/audit/run?category=compliance
func (h *AuditHandler) RunAuditByCategory(c *fiber.Ctx) error {
	category := c.Query("category")
	if category == "" {
		return h.RunAudit(c)
	}
	report, err := h.svc.RunChecksByCategory(c.Context(), AuditCategory(category))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "AUDIT_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, report, "Audit completed (filtered by category)")
}

// RunAuditByOwner runs all checks but returns only issues assigned to a specific owner.
// GET /api/v1/pharmacy/audit/by-owner?owner=pharmacist
func (h *AuditHandler) RunAuditByOwner(c *fiber.Ctx) error {
	owner := c.Query("owner")
	if owner == "" {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_PARAM", "owner query param required")
	}
	report, err := h.svc.RunChecksByOwner(c.Context(), AuditOwner(owner))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "AUDIT_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, report, "Audit completed (filtered by owner)")
}

// ListOwners returns the directory of all owner roles with contact info.
// GET /api/v1/pharmacy/audit/owners
func (h *AuditHandler) ListOwners(c *fiber.Ctx) error {
	owners := h.svc.ListOwners()
	return response.JSON(c, fiber.StatusOK, owners, "Owner directory")
}

func (h *AuditHandler) RegisterRoutes(router fiber.Router) {
	// Bug #15 fix: directory listing is safe for pharmacists
	audit := router.Group("/pharmacy/audit", middleware.PharmacyManagerOnly())
	audit.Get("/owners", h.ListOwners)
	audit.Get("/by-owner", h.RunAuditByOwner)

	// Full audit execution requires pharmacy_admin or admin
	auditAdmin := router.Group("/pharmacy/audit", middleware.PharmacyAdminOnly())
	auditAdmin.Get("/run", h.RunAudit)
	auditAdmin.Get("/by-category", h.RunAuditByCategory)
}

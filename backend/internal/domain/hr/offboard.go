package hr

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/utils/response"
	"time"

	"github.com/gofiber/fiber/v2"
)

// OffboardStaff terminates a staff member: marks the profile terminated with a
// last working day + reason, and (optionally) deactivates the login. This closes
// the employee lifecycle — terminated staff are skipped by payroll.
//
// POST /admin/hr/staff/:id/offboard
// body: { last_working_day: "YYYY-MM-DD", reason: string, deactivate_user?: bool }
//
// @Summary Offboard Staff
// @Description Terminate a staff member and stop their payroll/payout eligibility.
// @Tags HR
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "Staff user ID"
// @Success 200 {object} response.StandardResponse "Staff offboarded"
// @Router /api/v1/admin/hr/staff/{id}/offboard [post]
func OffboardStaff(c *fiber.Ctx) error {
	userID, err := c.ParamsInt("id")
	if err != nil || userID <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid staff user_id")
	}
	var req struct {
		LastWorkingDay string `json:"last_working_day"`
		Reason         string `json:"reason"`
		DeactivateUser bool   `json:"deactivate_user"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	lwd := time.Now()
	if req.LastWorkingDay != "" {
		if t, perr := time.Parse("2006-01-02", req.LastWorkingDay); perr == nil {
			lwd = t
		}
	}

	updates := map[string]interface{}{
		"employment_status":  "terminated",
		"last_working_day":   lwd,
		"termination_reason": req.Reason,
	}
	res := database.DB.Model(&StaffProfile{}).Where("user_id = ?", userID).Updates(updates)
	if res.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return response.Error(c, fiber.StatusNotFound, "PROFILE_NOT_FOUND", "Staff profile not found")
	}

	if req.DeactivateUser {
		database.DB.Model(&identity.User{}).Where("id = ?", userID).Update("is_active", false)
	}

	// Compute and persist the full-and-final settlement packet.
	settlement, serr := ComputeFinalSettlement(uint(userID), lwd)
	if serr == nil {
		_ = SaveFinalSettlement(settlement)
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"user_id":          userID,
		"last_working_day": lwd.Format("2006-01-02"),
		"deactivated":      req.DeactivateUser,
		"settlement":       settlement,
	}, "Staff offboarded")
}

// GetFinalSettlement returns the stored settlement for an offboarded staff member,
// recomputing a fresh preview if none has been saved yet.
//
// GET /admin/hr/staff/:id/settlement
//
// @Summary Get Final Settlement
// @Description Full-and-final settlement packet for an offboarded staff member.
// @Tags HR
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "Staff user ID"
// @Success 200 {object} response.StandardResponse{data=FinalSettlement} "Settlement"
// @Router /api/v1/admin/hr/staff/{id}/settlement [get]
func GetFinalSettlement(c *fiber.Ctx) error {
	userID, err := c.ParamsInt("id")
	if err != nil || userID <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid staff user_id")
	}
	var existing FinalSettlement
	if err := database.DB.Where("user_id = ?", userID).First(&existing).Error; err == nil {
		return response.JSON(c, fiber.StatusOK, existing, "Final settlement fetched")
	}
	// No saved settlement — return a live preview as of today.
	preview, perr := ComputeFinalSettlement(uint(userID), time.Now())
	if perr != nil {
		return response.Error(c, fiber.StatusNotFound, "NO_PROFILE", perr.Error())
	}
	return response.JSON(c, fiber.StatusOK, preview, "Final settlement preview")
}

// UpdateExitClearance toggles the offboarding clearance checklist and, once all
// items are cleared, marks the settlement paid.
//
// PUT /admin/hr/staff/:id/clearance
// body: { it_cleared, assets_returned, finance_cleared, handover_done: bool, mark_paid?: bool }
//
// @Summary Update Exit Clearance
// @Tags HR
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "Staff user ID"
// @Success 200 {object} response.StandardResponse{data=FinalSettlement} "Clearance updated"
// @Router /api/v1/admin/hr/staff/{id}/clearance [put]
func UpdateExitClearance(c *fiber.Ctx) error {
	userID, err := c.ParamsInt("id")
	if err != nil || userID <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid staff user_id")
	}
	var req struct {
		ITCleared      bool `json:"it_cleared"`
		AssetsReturned bool `json:"assets_returned"`
		FinanceCleared bool `json:"finance_cleared"`
		HandoverDone   bool `json:"handover_done"`
		MarkPaid       bool `json:"mark_paid"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	var s FinalSettlement
	if err := database.DB.Where("user_id = ?", userID).First(&s).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NO_SETTLEMENT", "No settlement found; offboard the staff member first")
	}
	s.ITCleared = req.ITCleared
	s.AssetsReturned = req.AssetsReturned
	s.FinanceCleared = req.FinanceCleared
	s.HandoverDone = req.HandoverDone

	if req.MarkPaid {
		if !s.ClearanceComplete() {
			return response.Error(c, fiber.StatusBadRequest, "CLEARANCE_INCOMPLETE",
				"All clearance items must be signed off before the settlement can be marked paid")
		}
		s.Status = "paid"
	}
	if err := database.DB.Save(&s).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, s, "Exit clearance updated")
}

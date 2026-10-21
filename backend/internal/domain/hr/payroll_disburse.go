package hr

import (
	"drayya-core/internal/database"
	"drayya-core/internal/services/disbursement"
	"time"

	"github.com/gofiber/fiber/v2"
)

// disbursePayroll pays out a salaried PayrollRecord through the configured
// disburser (stub until bank credentials are provisioned) and marks it paid.
//
// POST /hr/payroll/:id/disburse
// body: { bank_account_number, bank_ifsc, beneficiary_name }
//
// @Summary Disburse Payroll
// @Description Pay a salaried payroll record to a bank account (NACH/payout).
// @Tags HR
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "Payroll record ID"
// @Success 200 {object} response.StandardResponse "Disbursed"
// @Router /api/v1/hr/payroll/{id}/disburse [post]
func disbursePayroll(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payroll id"})
	}
	var req struct {
		BankAccountNumber string `json:"bank_account_number"`
		BankIFSC          string `json:"bank_ifsc"`
		BeneficiaryName   string `json:"beneficiary_name"`
	}
	_ = c.BodyParser(&req)

	var rec PayrollRecord
	if err := database.DB.First(&rec, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "payroll record not found"})
	}
	if rec.Status == "paid" {
		return c.Status(400).JSON(fiber.Map{"error": "payroll already disbursed"})
	}

	res, derr := disbursement.Default().Disburse(disbursement.Request{
		UserID:            rec.UserID,
		AmountPaise:       rec.NetAmount,
		BeneficiaryName:   req.BeneficiaryName,
		BankAccountNumber: req.BankAccountNumber,
		BankIFSC:          req.BankIFSC,
		Narration:         "Salary",
	})
	if derr != nil {
		return c.Status(502).JSON(fiber.Map{"error": derr.Error(), "provider": res.Provider})
	}

	now := time.Now()
	rec.Status = "paid"
	rec.PaymentDate = &now
	rec.TransactionID = res.Reference
	if err := database.DB.Save(&rec).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"payroll_id": rec.ID, "reference": res.Reference, "provider": res.Provider, "status": rec.Status})
}

package budget

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"drayya-core/internal/database"
	"drayya-core/internal/middleware"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) RegisterRoutes(router fiber.Router) {
	b := router.Group("/budget", middleware.Protected(), middleware.RequireRoles("admin", "super_admin", "finance_manager"))

	// Periods
	b.Post("/periods", createPeriod)
	b.Get("/periods", listPeriods)
	b.Get("/periods/:id", getPeriod)

	// Budget lines
	b.Post("/periods/:id/lines", createLine)
	b.Get("/periods/:id/lines", listLines)
	b.Put("/lines/:id", updateLine)
	b.Get("/periods/:id/summary", periodSummary)

	// Expenses
	b.Post("/lines/:id/expenses", submitExpense)
	b.Get("/lines/:id/expenses", listExpenses)
	b.Post("/expenses/:id/approve", approveExpense)
	b.Post("/expenses/:id/reject", rejectExpense)
}

// ── Period ────────────────────────────────────────────────────────────────────

func createPeriod(c *fiber.Ctx) error {
	var p BudgetPeriod
	if err := c.BodyParser(&p); err != nil || p.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name is required"})
	}
	if err := database.DB.Create(&p).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(p)
}

func listPeriods(c *fiber.Ctx) error {
	var periods []BudgetPeriod
	if err := database.DB.Order("start_date DESC").Find(&periods).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(periods)
}

func getPeriod(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var p BudgetPeriod
	if err := database.DB.First(&p, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "period not found"})
	}
	return c.JSON(p)
}

// ── Budget Lines ──────────────────────────────────────────────────────────────

func createLine(c *fiber.Ctx) error {
	periodID, _ := strconv.Atoi(c.Params("id"))
	var line BudgetLine
	if err := c.BodyParser(&line); err != nil || line.Category == "" {
		return c.Status(400).JSON(fiber.Map{"error": "category is required"})
	}
	line.PeriodID = uint(periodID)
	if err := database.DB.Create(&line).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(line)
}

func listLines(c *fiber.Ctx) error {
	periodID, _ := strconv.Atoi(c.Params("id"))
	var lines []BudgetLine
	if err := database.DB.Where("period_id = ?", periodID).Find(&lines).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	// Attach computed fields
	type lineOut struct {
		BudgetLine
		RemainingPaise  int64   `json:"remaining_paise"`
		UtilizationPct  float64 `json:"utilization_pct"`
	}
	out := make([]lineOut, len(lines))
	for i, l := range lines {
		out[i] = lineOut{l, l.RemainingPaise(), l.UtilizationPct()}
	}
	return c.JSON(out)
}

func updateLine(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var line BudgetLine
	if err := database.DB.First(&line, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "line not found"})
	}
	if err := c.BodyParser(&line); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if err := database.DB.Save(&line).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(line)
}

func periodSummary(c *fiber.Ctx) error {
	periodID, _ := strconv.Atoi(c.Params("id"))
	var lines []BudgetLine
	database.DB.Where("period_id = ?", periodID).Find(&lines)

	var totalAllocated, totalSpent, totalCommitted int64
	for _, l := range lines {
		totalAllocated += l.AllocatedPaise
		totalSpent += l.SpentPaise
		totalCommitted += l.CommittedPaise
	}
	utilPct := float64(0)
	if totalAllocated > 0 {
		utilPct = float64(totalSpent) / float64(totalAllocated) * 100
	}
	return c.JSON(fiber.Map{
		"period_id":         periodID,
		"total_allocated":   totalAllocated,
		"total_spent":       totalSpent,
		"total_committed":   totalCommitted,
		"total_remaining":   totalAllocated - totalSpent - totalCommitted,
		"utilization_pct":   utilPct,
		"line_count":        len(lines),
	})
}

// ── Expenses ──────────────────────────────────────────────────────────────────

func submitExpense(c *fiber.Ctx) error {
	lineID, _ := strconv.Atoi(c.Params("id"))
	submitterID := c.Locals("user_id").(uint)
	var exp BudgetExpense
	if err := c.BodyParser(&exp); err != nil || exp.AmountPaise <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "amount_paise is required"})
	}
	exp.LineID = uint(lineID)
	exp.SubmittedBy = submitterID
	if exp.ExpenseDate.IsZero() {
		exp.ExpenseDate = time.Now()
	}
	// Commit the amount
	if err := database.DB.Model(&BudgetLine{}).Where("id = ?", lineID).
		UpdateColumn("committed_paise", database.DB.Raw("committed_paise + ?", exp.AmountPaise)).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to commit budget"})
	}
	if err := database.DB.Create(&exp).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(exp)
}

func listExpenses(c *fiber.Ctx) error {
	lineID, _ := strconv.Atoi(c.Params("id"))
	var expenses []BudgetExpense
	if err := database.DB.Where("line_id = ?", lineID).Order("expense_date DESC").Find(&expenses).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(expenses)
}

func approveExpense(c *fiber.Ctx) error {
	approverID := c.Locals("user_id").(uint)
	id, _ := strconv.Atoi(c.Params("id"))
	var exp BudgetExpense
	if err := database.DB.First(&exp, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "expense not found"})
	}
	exp.Status = "approved"
	exp.ApprovedBy = &approverID
	// Move from committed to spent
	database.DB.Model(&BudgetLine{}).Where("id = ?", exp.LineID).
		Updates(map[string]interface{}{
			"spent_paise":     database.DB.Raw("spent_paise + ?", exp.AmountPaise),
			"committed_paise": database.DB.Raw("committed_paise - ?", exp.AmountPaise),
		})
	if err := database.DB.Save(&exp).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(exp)
}

func rejectExpense(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var exp BudgetExpense
	if err := database.DB.First(&exp, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "expense not found"})
	}
	exp.Status = "rejected"
	// Release committed amount
	database.DB.Model(&BudgetLine{}).Where("id = ?", exp.LineID).
		UpdateColumn("committed_paise", database.DB.Raw("committed_paise - ?", exp.AmountPaise))
	if err := database.DB.Save(&exp).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(exp)
}

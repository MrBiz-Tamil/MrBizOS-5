package pharmacy

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"drayya-core/internal/database"
	"drayya-core/internal/middleware"
)

// RegisterRecallRoutes wires drug recall and compliance endpoints.
func RegisterRecallRoutes(router fiber.Router) {
	admin := router.Group("/pharmacy/compliance", middleware.Protected(), middleware.RequireRoles("admin", "super_admin", "pharmacist"))

	// Drug recall
	admin.Post("/recalls", createRecall)
	admin.Get("/recalls", listRecalls)
	admin.Get("/recalls/:id", getRecall)
	admin.Post("/recalls/:id/action", recordRecallAction)
	admin.Put("/recalls/:id/resolve", resolveRecall)

	// Schedule H drugs registry
	admin.Post("/schedule-h", addScheduleHDrug)
	admin.Get("/schedule-h", listScheduleHDrugs)
	admin.Get("/schedule-h/check", checkDrugSchedule)

	// Narcotics register
	admin.Post("/narcotics", addNarcoticsEntry)
	admin.Get("/narcotics", listNarcoticsEntries)
}

// ── Drug Recall Handlers ──────────────────────────────────────────────────────

func createRecall(c *fiber.Ctx) error {
	var recall DrugRecall
	if err := c.BodyParser(&recall); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	if recall.DrugName == "" || recall.BatchNumber == "" {
		return c.Status(400).JSON(fiber.Map{"error": "drug_name and batch_number are required"})
	}
	if recall.RecallDate.IsZero() {
		recall.RecallDate = time.Now()
	}
	if err := database.DB.Create(&recall).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(recall)
}

// listRecalls lists drug recalls with optional status filter and pagination.
// M-20: page (default 1) and limit (default 20, max 100) query params.
func listRecalls(c *fiber.Ctx) error {
	status := c.Query("status", "active")

	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.Query("limit", "20"))
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	var recalls []DrugRecall
	q := database.DB.Order("recall_date DESC")
	if status != "all" {
		q = q.Where("status = ?", status)
	}
	if err := q.Offset(offset).Limit(limit).Find(&recalls).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"page": page, "limit": limit, "data": recalls})
}

func getRecall(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid ID"})
	}
	var recall DrugRecall
	if err := database.DB.First(&recall, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "recall not found"})
	}
	var actions []DrugRecallAction
	if err := database.DB.Where("recall_id = ?", id).Find(&actions).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"recall": recall, "actions": actions})
}

func recordRecallAction(c *fiber.Ctx) error {
	recallID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid ID"})
	}

	// C-1: comma-ok pattern instead of bare type assertion
	staffID, ok := c.Locals("user_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authentication required"})
	}

	var action DrugRecallAction
	if err := c.BodyParser(&action); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	action.RecallID = uint(recallID)
	action.ActionBy = staffID
	if err := database.DB.Create(&action).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(action)
}

func resolveRecall(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid ID"})
	}
	if err := database.DB.Model(&DrugRecall{}).Where("id = ?", id).
		Update("status", "resolved").Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "recall resolved"})
}

// ── Schedule H Handlers ───────────────────────────────────────────────────────

// addScheduleHDrug adds a Schedule H drug. M-22: checks for duplicate drug_name + schedule.
func addScheduleHDrug(c *fiber.Ctx) error {
	var drug ScheduleHDrug
	if err := c.BodyParser(&drug); err != nil || drug.DrugName == "" || drug.Schedule == "" {
		return c.Status(400).JSON(fiber.Map{"error": "drug_name and schedule are required"})
	}

	// M-22: duplicate check
	var existing ScheduleHDrug
	if err := database.DB.Where("drug_name = ? AND schedule = ?", drug.DrugName, drug.Schedule).
		First(&existing).Error; err == nil {
		return c.Status(409).JSON(fiber.Map{"error": "a record with the same drug_name and schedule already exists"})
	}

	if err := database.DB.Create(&drug).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(drug)
}

func listScheduleHDrugs(c *fiber.Ctx) error {
	schedule := c.Query("schedule")
	var drugs []ScheduleHDrug
	q := database.DB.Order("drug_name ASC")
	if schedule != "" {
		q = q.Where("schedule = ?", schedule)
	}
	if err := q.Find(&drugs).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(drugs)
}

// checkDrugSchedule returns schedule classification for a drug name (used at dispensing).
func checkDrugSchedule(c *fiber.Ctx) error {
	name := c.Query("name")
	if name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name query param required"})
	}
	var drug ScheduleHDrug
	err := database.DB.Where("LOWER(drug_name) LIKE LOWER(?)", "%"+name+"%").First(&drug).Error
	if err != nil {
		return c.JSON(fiber.Map{"schedule": "OTC", "requires_prescription": false})
	}
	return c.JSON(fiber.Map{
		"schedule":              drug.Schedule,
		"requires_prescription": true,
		"is_narcotic":           drug.IsNarcotic,
		"drug_name":             drug.DrugName,
	})
}

// ── Narcotics Register Handlers ───────────────────────────────────────────────

func addNarcoticsEntry(c *fiber.Ctx) error {
	var entry NarcoticsRegisterEntry
	if err := c.BodyParser(&entry); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	// C-1: comma-ok pattern instead of bare type assertion
	dispenserID, ok := c.Locals("user_id").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authentication required"})
	}
	entry.DispenserID = dispenserID

	if entry.DispensedAt.IsZero() {
		entry.DispensedAt = time.Now()
	}
	if err := database.DB.Create(&entry).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(entry)
}

// listNarcoticsEntries lists narcotics register entries with caller-adjustable pagination.
// M-21: default 50, max 200.
func listNarcoticsEntries(c *fiber.Ctx) error {
	limit, err := strconv.Atoi(c.Query("limit", "50"))
	if err != nil || limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	var entries []NarcoticsRegisterEntry
	q := database.DB.Order("dispensed_at DESC").Offset(offset).Limit(limit)
	if start := c.Query("from"); start != "" {
		if t, err := time.Parse("2006-01-02", start); err == nil {
			q = q.Where("dispensed_at >= ?", t)
		}
	}
	if err := q.Find(&entries).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"page": page, "limit": limit, "data": entries})
}

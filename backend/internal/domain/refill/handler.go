package refill

import (
	"encoding/json"
	"fmt"
	"drayya-core/internal/crypto"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

// GetRefillSchedules returns all refill schedules for the current user
// GET /api/v1/pharmacy/refills
func GetRefillSchedules(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var schedules []RefillSchedule
	if err := database.DB.Where("user_id = ? AND status != 'cancelled'", userID).
		Order("next_refill_date ASC").Find(&schedules).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, schedules, "Refill schedules fetched")
}

// CreateRefillSchedule creates a new auto-refill schedule
// POST /api/v1/pharmacy/refills
func CreateRefillSchedule(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var req struct {
		MedicineName      string `json:"medicine_name"`
		Dosage            string `json:"dosage"`
		Frequency         string `json:"frequency"`
		QuantityPerRefill int    `json:"quantity_per_refill"`
		NextRefillDate    string `json:"next_refill_date"` // YYYY-MM-DD
		AutoOrder         bool   `json:"auto_order"`
		PrescriptionID    string `json:"prescription_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if req.MedicineName == "" || req.NextRefillDate == "" {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELDS", "medicine_name and next_refill_date required")
	}

	nextDate, err := time.Parse("2006-01-02", req.NextRefillDate)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_DATE", "next_refill_date must be YYYY-MM-DD")
	}

	qty := req.QuantityPerRefill
	if qty <= 0 {
		qty = 30
	}

	schedule := &RefillSchedule{
		UserID:            userID,
		MedicineName:      req.MedicineName,
		Dosage:            req.Dosage,
		Frequency:         req.Frequency,
		QuantityPerRefill: qty,
		NextRefillDate:    nextDate,
		AutoOrder:         req.AutoOrder,
		PrescriptionID:    req.PrescriptionID,
		Status:            "active",
	}

	if err := database.DB.Create(schedule).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, schedule, "Auto-refill scheduled")
}

// UpdateRefillSchedule updates an existing refill schedule.
// M-18: re-sets UserID from JWT after BodyParser; protects PharmacyID and PrescriptionID
//       by re-fetching original record and only copying allowed fields.
func UpdateRefillSchedule(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid schedule ID")
	}

	// Fetch the original record first to protect immutable fields
	var original RefillSchedule
	if err := database.DB.Where("id = ? AND user_id = ?", uint(id), userID).First(&original).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Schedule not found")
	}

	// Parse only the allowed update fields from the request body
	var req struct {
		MedicineName      string `json:"medicine_name"`
		Dosage            string `json:"dosage"`
		Frequency         string `json:"frequency"`
		QuantityPerRefill int    `json:"quantity_per_refill"`
		NextRefillDate    string `json:"next_refill_date"`
		AutoOrder         bool   `json:"auto_order"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	// Apply only the allowed fields; preserve immutable ones from original
	if req.MedicineName != "" {
		original.MedicineName = req.MedicineName
	}
	if req.Dosage != "" {
		original.Dosage = req.Dosage
	}
	if req.Frequency != "" {
		original.Frequency = req.Frequency
	}
	if req.QuantityPerRefill > 0 {
		original.QuantityPerRefill = req.QuantityPerRefill
	}
	if req.NextRefillDate != "" {
		if t, parseErr := time.Parse("2006-01-02", req.NextRefillDate); parseErr == nil {
			original.NextRefillDate = t
		}
	}
	original.AutoOrder = req.AutoOrder

	// M-18: explicitly re-set ownership fields to prevent override
	original.ID = uint(id)
	original.UserID = userID
	// PharmacyID and PrescriptionID are NOT updated — they come from original

	if err := database.DB.Save(&original).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, original, "Schedule updated")
}

// allowedRefillStatuses defines the valid status values for a refill schedule.
// M-19: status must be one of these values.
var allowedRefillStatuses = map[string]bool{
	"active":    true,
	"paused":    true,
	"cancelled": true,
	"completed": true,
}

// UpdateRefillStatus pauses or resumes a refill schedule.
// M-19: validates that req.Status is one of the allowed values.
func UpdateRefillStatus(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid schedule ID")
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil || req.Status == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Status required")
	}

	// M-19: validate status
	if !allowedRefillStatuses[req.Status] {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_STATUS",
			"status must be one of: active, paused, cancelled, completed")
	}

	result := database.DB.Model(&RefillSchedule{}).
		Where("id = ? AND user_id = ?", uint(id), userID).
		Update("status", req.Status)

	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Schedule not found")
	}
	return response.JSON(c, fiber.StatusOK, nil, "Status updated")
}

// DeleteRefillSchedule cancels a refill schedule
// DELETE /api/v1/pharmacy/refills/:id
func DeleteRefillSchedule(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid schedule ID")
	}

	result := database.DB.Model(&RefillSchedule{}).
		Where("id = ? AND user_id = ?", uint(id), userID).
		Update("status", "cancelled")

	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Schedule not found")
	}
	return response.JSON(c, fiber.StatusOK, nil, "Auto-refill cancelled")
}

// ProcessDueRefills processes all auto-refill schedules due today.
// Feature 4: converts RefillOrder stubs into real PharmacyOrders.
// POST /api/v1/internal/refills/process (internal, admin only)
func ProcessDueRefills(c *fiber.Ctx) error {
	ctx := c.Context()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())

	// Fetch active schedules due today (next_refill_date <= today).
	var schedules []RefillSchedule
	if err := database.DB.WithContext(ctx).
		Where("status = 'active' AND auto_order = true AND next_refill_date <= ?", today).
		Find(&schedules).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	type result struct {
		ScheduleID uint   `json:"schedule_id"`
		OrderID    *uint  `json:"order_id,omitempty"`
		Status     string `json:"status"`
		Error      string `json:"error,omitempty"`
	}
	var results []result

	for _, sched := range schedules {
		res := result{ScheduleID: sched.ID}

		// Determine pharmacy ID.
		pharmacyID := uint(1) // default
		if sched.PharmacyID != nil && *sched.PharmacyID != 0 {
			pharmacyID = *sched.PharmacyID
		}

		// Find medicine by name.
		var med pharmacy.Medicine
		if err := database.DB.WithContext(ctx).
			Where("LOWER(name) = LOWER(?)", sched.MedicineName).
			First(&med).Error; err != nil {
			res.Status = "failed"
			res.Error = fmt.Sprintf("medicine '%s' not found in catalog", sched.MedicineName)
			results = append(results, res)
			continue
		}

		// Build cart items JSON.
		items, _ := json.Marshal([]map[string]interface{}{
			{"medicine_id": med.ID, "medicine_name": med.Name, "quantity": sched.QuantityPerRefill, "unit_price": med.Prices["INR"]},
		})

		orderNum := fmt.Sprintf("REFILL-%s", crypto.SecureToken(3))
		order := pharmacy.PharmacyOrder{
			OrderNumber:   orderNum,
			UserID:        sched.UserID,
			PharmacyID:    pharmacyID,
			Items:         items,
			Amount:   med.Prices["INR"] * int64(sched.QuantityPerRefill),
			Status:        "pending",
			PaymentStatus: "pending",
			PaymentMethod: "auto_refill",
		}

		if err := database.DB.WithContext(ctx).Create(&order).Error; err != nil {
			res.Status = "failed"
			res.Error = err.Error()
			results = append(results, res)
			continue
		}

		// Create a RefillOrder linking the schedule to the new PharmacyOrder.
		refillOrder := RefillOrder{
			ScheduleID: sched.ID,
			UserID:     sched.UserID,
			OrderID:    &order.ID,
			Status:     "placed",
			Quantity:   sched.QuantityPerRefill,
		}
		database.DB.WithContext(ctx).Create(&refillOrder)

		// Advance the schedule's next refill date.
		database.DB.WithContext(ctx).Model(&sched).Updates(map[string]interface{}{
			"next_refill_date": advanceDate(sched.NextRefillDate, sched.Frequency),
			"last_filled_at":   now,
		})

		res.Status = "placed"
		res.OrderID = &order.ID
		results = append(results, res)
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"processed": len(results),
		"results":   results,
	}, "Due refills processed")
}

// ConvertRefillToOrder converts a single pending RefillOrder into a PharmacyOrder.
// Feature 4: POST /pharmacy/refills/:id/convert
func ConvertRefillToOrder(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid refill order ID")
	}

	var refillOrder RefillOrder
	if err := database.DB.WithContext(c.Context()).First(&refillOrder, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Refill order not found")
	}
	if refillOrder.Status != "pending" {
		return response.Error(c, fiber.StatusConflict, "ALREADY_PROCESSED",
			fmt.Sprintf("Refill order is already '%s'", refillOrder.Status))
	}
	if refillOrder.OrderID != nil {
		return response.Error(c, fiber.StatusConflict, "ALREADY_LINKED",
			"Refill order is already linked to a pharmacy order")
	}

	// Fetch schedule to get medicine name and quantity.
	var sched RefillSchedule
	if err := database.DB.WithContext(c.Context()).First(&sched, refillOrder.ScheduleID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "SCHEDULE_NOT_FOUND", "Refill schedule not found")
	}

	var med pharmacy.Medicine
	if err := database.DB.WithContext(c.Context()).
		Where("LOWER(name) = LOWER(?)", sched.MedicineName).
		First(&med).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "MEDICINE_NOT_FOUND",
			fmt.Sprintf("Medicine '%s' not found in catalog", sched.MedicineName))
	}

	pharmacyID := uint(1)
	if sched.PharmacyID != nil && *sched.PharmacyID != 0 {
		pharmacyID = *sched.PharmacyID
	}

	items, _ := json.Marshal([]map[string]interface{}{
		{"medicine_id": med.ID, "medicine_name": med.Name, "quantity": refillOrder.Quantity, "unit_price": med.Prices["INR"]},
	})
	orderNum := fmt.Sprintf("REFILL-%s", crypto.SecureToken(3))
	order := pharmacy.PharmacyOrder{
		OrderNumber:   orderNum,
		UserID:        refillOrder.UserID,
		PharmacyID:    pharmacyID,
		Items:         items,
		Amount:   med.Prices["INR"] * int64(refillOrder.Quantity),
		Status:        "pending",
		PaymentStatus: "pending",
		PaymentMethod: "auto_refill",
	}
	if err := database.DB.WithContext(c.Context()).Create(&order).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}

	// Link refill order → pharmacy order.
	database.DB.WithContext(c.Context()).Model(&refillOrder).
		Updates(map[string]interface{}{"order_id": order.ID, "status": "placed"})

	return response.JSON(c, fiber.StatusCreated, order, "Pharmacy order created from refill")
}

func advanceDate(from time.Time, frequency string) time.Time {
	switch frequency {
	case "Daily":
		return from.AddDate(0, 0, 1)
	case "Every 3 days":
		return from.AddDate(0, 0, 3)
	case "Weekly":
		return from.AddDate(0, 0, 7)
	case "Every 2 weeks":
		return from.AddDate(0, 0, 14)
	case "Monthly":
		return from.AddDate(0, 1, 0)
	case "Every 3 months":
		return from.AddDate(0, 3, 0)
	default:
		return from.AddDate(0, 1, 0)
	}
}

// RegisterRefillRoutes registers all auto-refill routes
func RegisterRefillRoutes(router fiber.Router) {
	r := router.Group("/pharmacy/refills", middleware.Protected())
	r.Get("/", GetRefillSchedules)
	r.Post("/", CreateRefillSchedule)
	r.Put("/:id", UpdateRefillSchedule)
	r.Patch("/:id/status", UpdateRefillStatus)
	r.Delete("/:id", DeleteRefillSchedule)

	// Feature 4: convert a pending refill order to a real pharmacy order
	r.Post("/:id/convert", middleware.PharmacyManagerOnly(), ConvertRefillToOrder)

	// Internal cron endpoint
	internal := router.Group("/internal", middleware.Protected(), middleware.AdminOnly())
	internal.Post("/refills/process", ProcessDueRefills)
}

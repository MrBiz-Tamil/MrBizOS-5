package shipping

// courier_handlers.go — HTTP endpoints used exclusively by the courier mobile app.
//
// Routes (all require JWT with role=courier):
//
//   PUT  /api/v1/courier/location           — GPS heartbeat (fixes Gap 1: scorer never activates)
//   PUT  /api/v1/courier/delivery/:id/status — status transitions (fixes Gap 2: active_deliveries never decrements)
//   POST /api/v1/courier/delivery/:id/complete — POD + final status (fixes Gap 6: fraud risk)
//   GET  /api/v1/courier/deliveries          — list active deliveries for the courier

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"
)

// validTransitions defines the allowed status state machine for a courier delivery.
// Only forward transitions are accepted; backwards moves are rejected.
var validTransitions = map[string]string{
	"assigned":  "picked_up",
	"picked_up": "en_route",
	"en_route":  "arrived",
	"arrived":   "completed",
}

// RegisterCourierRoutes wires all courier-app API endpoints.
func RegisterCourierRoutes(router fiber.Router) {
	g := router.Group("/courier", middleware.Protected(), middleware.RequireRoles("courier", "admin"))

	g.Put("/location", UpdateCourierLocationHandler)
	g.Get("/deliveries", ListActiveDeliveries)
	g.Put("/delivery/:id/status", UpdateDeliveryStatusHandler)
	g.Post("/delivery/:id/complete", CompleteDeliveryHandler)
}

// ─── PUT /courier/location ────────────────────────────────────────────────────

// UpdateCourierLocationHandler receives a GPS ping from the courier app every
// ~30 seconds and writes lat/lng to both PostgreSQL (durable) and Redis (fast
// cache used by TryAutoAssign scorer).
//
// Without this endpoint TryAutoAssign always sees (0,0) for every courier and
// skips them all — the entire smart scoring system is dead.
func UpdateCourierLocationHandler(c *fiber.Ctx) error {
	courierUserID, ok := c.Locals("user_id").(uint)
	if !ok || courierUserID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	var req struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Expected {latitude, longitude}")
	}
	if req.Latitude == 0 && req.Longitude == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_COORDS", "Latitude and longitude must be non-zero")
	}

	UpdateCourierPosition(c.Context(), courierUserID, req.Latitude, req.Longitude)

	return response.JSON(c, fiber.StatusOK, nil, "Location updated")
}

// ─── GET /courier/deliveries ─────────────────────────────────────────────────

// ListActiveDeliveries returns all deliveries currently assigned to the courier
// that are not yet completed or failed.
func ListActiveDeliveries(c *fiber.Ctx) error {
	courierUserID, ok := c.Locals("user_id").(uint)
	if !ok || courierUserID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	var deliveries []pharmacy.CourierDelivery
	if err := database.DB.WithContext(c.Context()).
		Where("courier_id = ? AND status NOT IN ?", courierUserID,
			[]string{"completed", "failed", "cancelled"}).
		Order("created_at ASC").
		Find(&deliveries).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Could not fetch deliveries")
	}

	return response.JSON(c, fiber.StatusOK, deliveries, "Active deliveries")
}

// ─── PUT /courier/delivery/:id/status ────────────────────────────────────────

// UpdateDeliveryStatusHandler advances a delivery through its state machine:
//   assigned → picked_up → en_route → arrived
//
// "completed" is handled separately by CompleteDeliveryHandler (requires POD).
// This is the critical missing endpoint — without it active_deliveries only
// ever increases and couriers permanently disappear from the scorer pool.
func UpdateDeliveryStatusHandler(c *fiber.Ctx) error {
	courierUserID, ok := c.Locals("user_id").(uint)
	if !ok || courierUserID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	deliveryID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid delivery ID")
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil || req.Status == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Expected {status}")
	}

	var delivery pharmacy.CourierDelivery
	if err := database.DB.WithContext(c.Context()).
		First(&delivery, deliveryID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Delivery not found")
	}

	// Ownership check — courier can only update their own deliveries
	if delivery.CourierID != courierUserID {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Not your delivery")
	}

	// Validate state machine transition
	expectedNext, ok := validTransitions[delivery.Status]
	if !ok || expectedNext != req.Status {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_TRANSITION",
			fmt.Sprintf("Cannot move from %q to %q (expected %q)", delivery.Status, req.Status, expectedNext))
	}

	// Reject "completed" here — must use the POD endpoint
	if req.Status == "completed" {
		return response.Error(c, fiber.StatusBadRequest, "USE_COMPLETE_ENDPOINT",
			"Use POST /courier/delivery/:id/complete to mark as delivered (requires proof photo)")
	}

	if err := database.DB.WithContext(c.Context()).
		Model(&delivery).
		Updates(map[string]interface{}{
			"status":     req.Status,
			"updated_at": time.Now(),
		}).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Could not update status")
	}

	// Push WS + FCM notification to patient
	var order pharmacy.PharmacyOrder
	if database.DB.WithContext(c.Context()).First(&order, delivery.OrderID).Error == nil {
		delivery.Status = req.Status
		SendDeliveryStatusNotification(c.Context(), &delivery, order.UserID)
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"delivery_id": delivery.ID,
		"new_status":  req.Status,
	}, "Delivery status updated")
}

// ─── POST /courier/delivery/:id/complete ────────────────────────────────────

// CompleteDeliveryHandler marks a delivery as completed, requires a proof-of-
// delivery photo URL, decrements the courier's active load, and triggers the
// finance payout.
//
// This fixes Gap 6 (no POD = fraud risk) and also ensures DecrementCourierLoad
// is called so the courier stays visible to the scorer.
func CompleteDeliveryHandler(c *fiber.Ctx) error {
	courierUserID, ok := c.Locals("user_id").(uint)
	if !ok || courierUserID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	deliveryID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid delivery ID")
	}

	var req struct {
		ProofPhotoURL string  `json:"proof_photo_url"` // required — S3/R2 URL
		Notes         string  `json:"notes"`
		FinalLatitude  float64 `json:"final_latitude"`
		FinalLongitude float64 `json:"final_longitude"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", "Invalid request body")
	}
	if req.ProofPhotoURL == "" {
		return response.Error(c, fiber.StatusBadRequest, "POD_REQUIRED",
			"proof_photo_url is required to complete a delivery")
	}

	var delivery pharmacy.CourierDelivery
	if err := database.DB.WithContext(c.Context()).
		First(&delivery, deliveryID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Delivery not found")
	}

	if delivery.CourierID != courierUserID {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Not your delivery")
	}

	if delivery.Status == "completed" {
		return response.Error(c, fiber.StatusConflict, "ALREADY_DONE", "Delivery already completed")
	}

	// Delivery must be in "arrived" state to complete
	if delivery.Status != "arrived" {
		return response.Error(c, fiber.StatusBadRequest, "WRONG_STATUS",
			fmt.Sprintf("Delivery must be in 'arrived' state before completing (current: %s)", delivery.Status))
	}

	now := time.Now()
	if err := database.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// 1. Mark delivery completed
		if err := tx.Model(&delivery).Updates(map[string]interface{}{
			"status":               "completed",
			"actual_delivery_time": now,
			"proof_photo_url":      req.ProofPhotoURL,
			"delivery_notes":       req.Notes,
			"updated_at":           now,
		}).Error; err != nil {
			return err
		}
		// 2. Mark parent order delivered
		if err := tx.Model(&pharmacy.PharmacyOrder{}).
			Where("id = ?", delivery.OrderID).
			Updates(map[string]interface{}{
				"status":     "delivered",
				"updated_at": now,
			}).Error; err != nil {
			return err
		}
		// 3. Increment total_trips + today_trips
		return tx.Model(&pharmacy.CourierProfile{}).
			Where("user_id = ?", courierUserID).
			Updates(map[string]interface{}{
				"total_trips": gorm.Expr("total_trips + 1"),
				"today_trips": gorm.Expr("today_trips + 1"),
			}).Error
	}); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "COMPLETE_FAILED", "Failed to complete delivery")
	}

	// Decrement active_deliveries (outside transaction — safe to retry)
	DecrementCourierLoad(c.Context(), courierUserID)

	// Notify patient
	var order pharmacy.PharmacyOrder
	if database.DB.WithContext(c.Context()).First(&order, delivery.OrderID).Error == nil {
		delivery.Status = "completed"
		SendDeliveryStatusNotification(c.Context(), &delivery, order.UserID)
	}

	// Update courier's last known position if provided
	if req.FinalLatitude != 0 || req.FinalLongitude != 0 {
		UpdateCourierPosition(c.Context(), courierUserID, req.FinalLatitude, req.FinalLongitude)
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"delivery_id": delivery.ID,
		"status":      "completed",
		"completed_at": now,
	}, "Delivery completed")
}

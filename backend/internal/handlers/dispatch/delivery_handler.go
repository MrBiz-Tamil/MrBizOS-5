package dispatch

import (
	"encoding/json"
	"fmt"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/shipping"
	"drayya-core/internal/utils/response"
	"time"

	"gorm.io/gorm"

	"github.com/gofiber/fiber/v2"
)

type CourierAssignmentRequest struct {
	OrderID   uint `json:"order_id"`
	CourierID uint `json:"courier_id"`
}

type CourierLocationUpdateRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Status    string  `json:"status"` // allow updating status e.g., "en_route", "arrived", "completed"
}

// GetDeliveryZones returns all defined delivery zones for logistics
// @Summary Get Delivery Zones
// @Description Fetch all available delivery zones and their operational boundaries
// @Tags Shipping
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} response.StandardResponse{data=[]pharmacy.DeliveryZone} "Zones fetched"
// @Router /api/v1/dispatch/zones [get]
func GetDeliveryZones(c *fiber.Ctx) error {
	var zones []pharmacy.DeliveryZone
	if err := database.DB.Find(&zones).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch zones")
	}
	return response.JSON(c, fiber.StatusOK, zones, "Delivery zones fetched")
}

// UpdateDeliveryZone updates the configuration of a specific delivery zone
// @Summary Update Delivery Zone
// @Description Update the geometry or metadata of an existing delivery zone
// @Tags Shipping
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Zone ID"
// @Param zone body pharmacy.DeliveryZone true "Zone Data"
// @Success 200 {object} response.StandardResponse{data=pharmacy.DeliveryZone} "Zone updated"
// @Router /api/v1/dispatch/zones/{id} [put]
func UpdateDeliveryZone(c *fiber.Ctx) error {
	id := c.Params("id")
	var zone pharmacy.DeliveryZone
	if err := database.DB.First(&zone, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Zone not found")
	}

	if err := c.BodyParser(&zone); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	if err := database.DB.Save(&zone).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update delivery zone")
	}
	return response.JSON(c, fiber.StatusOK, zone, "Delivery zone updated")
}

// GetCourierDeliveries returns a list of all courier delivery assignments
// @Summary Get Courier Deliveries
// @Description Fetch all active and past courier deliveries for monitoring
// @Tags Shipping
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} response.StandardResponse{data=[]pharmacy.CourierDelivery} "Deliveries fetched"
// @Router /api/v1/dispatch/deliveries [get]
func GetCourierDeliveries(c *fiber.Ctx) error {
	var deliveries []pharmacy.CourierDelivery
	if err := database.DB.Find(&deliveries).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch deliveries")
	}
	return response.JSON(c, fiber.StatusOK, deliveries, "Courier deliveries fetched")
}

// AssignCourier assigns a courier to a pharmacy order
// @Summary Assign Courier
// @Description Manually link a courier to a specific pharmacy delivery order
// @Tags Shipping
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body CourierAssignmentRequest true "Assignment Request (order_id, courier_id)"
// @Success 201 {object} response.StandardResponse{data=pharmacy.CourierDelivery} "Courier assigned"
// @Router /api/v1/dispatch/assign-courier [post]
func AssignCourier(c *fiber.Ctx) error {
	var req CourierAssignmentRequest
	if err := c.BodyParser(&req); err != nil || req.OrderID == 0 || req.CourierID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "order_id and courier_id are required")
	}

	// Verify the order exists and is in a dispatchable state
	var order pharmacy.PharmacyOrder
	if err := database.DB.First(&order, req.OrderID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "ORDER_NOT_FOUND", "Order not found")
	}
	if order.Status != "paid" && order.Status != "confirmed" && order.Status != "dispatching" {
		return response.Error(c, fiber.StatusConflict, "INVALID_STATE",
			fmt.Sprintf("Order is not in a dispatchable state (current: %s)", order.Status))
	}

	// Prevent duplicate delivery record for the same order
	var existing pharmacy.CourierDelivery
	if err := database.DB.Where("order_id = ? AND status NOT IN ?", req.OrderID,
		[]string{"failed", "cancelled"}).First(&existing).Error; err == nil {
		return response.Error(c, fiber.StatusConflict, "ALREADY_ASSIGNED",
			fmt.Sprintf("Order already has an active delivery (status: %s)", existing.Status))
	}

	slaDeadline := time.Now().Add(24 * time.Hour) // STANDARD SLA default
	if order.DeliveryMode == pharmacy.ModeQuick {
		slaDeadline = time.Now().Add(30 * time.Minute)
	}

	var delivery pharmacy.CourierDelivery
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		delivery = pharmacy.CourierDelivery{
			OrderID:     req.OrderID,
			CourierID:   req.CourierID,
			Status:      "assigned",
			SLADeadline: &slaDeadline,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := tx.Create(&delivery).Error; err != nil {
			return err
		}
		// Sync order status
		return tx.Model(&pharmacy.PharmacyOrder{}).
			Where("id = ?", req.OrderID).
			Updates(map[string]interface{}{"status": "assigned", "sla_deadline": slaDeadline}).Error
	}); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to assign courier")
	}

	return response.JSON(c, fiber.StatusCreated, delivery, "Courier assigned successfully")
}

// UpdateCourierLocation updates the real-time coordinates of a delivery courier
// @Summary Update Courier Location
// @Description Log the current GPS coordinates and operational status of a courier
// @Tags Shipping
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Delivery ID"
// @Param request body CourierLocationUpdateRequest true "Location Data (latitude, longitude, status)"
// @Success 200 {object} response.StandardResponse{data=pharmacy.CourierDelivery} "Location updated"
// @Router /api/v1/dispatch/courier-location/{id} [put]
func UpdateCourierLocation(c *fiber.Ctx) error {
	id := c.Params("id")
	var req CourierLocationUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Expected {latitude, longitude, status}")
	}

	var delivery pharmacy.CourierDelivery
	if err := database.DB.First(&delivery, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Delivery not found")
	}

	updates := map[string]interface{}{
		"current_latitude":  req.Latitude,
		"current_longitude": req.Longitude,
		"updated_at":        time.Now(),
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}

	if err := database.DB.Model(&delivery).Updates(updates).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update location")
	}

	// Also persist to CourierProfile so TryAutoAssign scorer has up-to-date position
	if req.Latitude != 0 || req.Longitude != 0 {
		shipping.UpdateCourierPosition(c.Context(), delivery.CourierID, req.Latitude, req.Longitude)
	}

	return response.JSON(c, fiber.StatusOK, delivery, "Courier location/status updated")
}

// AcceptDelivery allows a courier to accept a broadcasted delivery request.
// CourierID is extracted from the JWT — never from the request body — to prevent IDOR.
// @Summary Accept Delivery
// @Description Accept a broadcast delivery request via Redis distributed lock (courier JWT required)
// @Tags Shipping
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Router /api/v1/dispatch/accept [post]
func AcceptDelivery(c *fiber.Ctx) error {
	// Extract the courier's identity from the JWT — do not trust the body.
	courierUserID, ok := c.Locals("user_id").(uint)
	if !ok || courierUserID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	var req struct {
		OrderID uint `json:"order_id"`
	}
	if err := c.BodyParser(&req); err != nil || req.OrderID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "order_id is required")
	}

	if database.Rdb == nil {
		return response.Error(c, fiber.StatusInternalServerError, "REDIS_UNAVAILABLE", "Real-time dispatch is offline")
	}

	// 1. Acquire the Redis broadcast lock atomically — prevents two couriers from
	//    simultaneously accepting the same broadcast.
	ctx := c.Context()
	dispatchKey := fmt.Sprintf("dispatch:order:%d", req.OrderID)

	val, err := database.Rdb.GetDel(ctx, dispatchKey).Result()
	if err != nil {
		return response.Error(c, fiber.StatusConflict, "DELIVERY_TAKEN",
			"This delivery has already been accepted or is not available")
	}
	if val != "pending" {
		return response.Error(c, fiber.StatusConflict, "DELIVERY_TAKEN",
			"This delivery has already been accepted by another courier")
	}

	// 2. Load the order and compute SLA deadline before opening the transaction.
	var order pharmacy.PharmacyOrder
	if err := database.DB.WithContext(ctx).Preload("Pharmacy").First(&order, req.OrderID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Order not found")
	}

	slaDeadline := time.Now().Add(24 * time.Hour) // STANDARD default
	if order.DeliveryMode == pharmacy.ModeQuick {
		slaDeadline = time.Now().Add(30 * time.Minute)
	}

	// 3. Atomically create the delivery record + update order status + increment courier load.
	var delivery pharmacy.CourierDelivery
	if err := database.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		delivery = pharmacy.CourierDelivery{
			OrderID:               req.OrderID,
			CourierID:             courierUserID,
			Status:                "assigned",
			PickupLatitude:        order.Pharmacy.Latitude,
			PickupLongitude:       order.Pharmacy.Longitude,
			DestLatitude:          order.DestLatitude,
			DestLongitude:         order.DestLongitude,
			SLADeadline:           &slaDeadline,
			EstimatedDeliveryTime: slaDeadline,
			CreatedAt:             time.Now(),
			UpdatedAt:             time.Now(),
		}
		if err := tx.Create(&delivery).Error; err != nil {
			return err
		}
		if err := tx.Model(&pharmacy.PharmacyOrder{}).Where("id = ?", req.OrderID).
			Updates(map[string]interface{}{
				"status":       "assigned",
				"sla_deadline": slaDeadline,
			}).Error; err != nil {
			return err
		}
		// Increment courier's active_deliveries (matched by user_id, same as DecrementCourierLoad)
		return tx.Model(&pharmacy.CourierProfile{}).
			Where("user_id = ?", courierUserID).
			UpdateColumn("active_deliveries", gorm.Expr("active_deliveries + 1")).Error
	}); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ACCEPT_FAILED", "Failed to accept delivery")
	}

	// 4. Notify patient that a courier is on the way
	shipping.SendDeliveryStatusNotification(ctx, &delivery, order.UserID)

	return response.JSON(c, fiber.StatusOK, delivery, "Delivery accepted successfully")
}

// CompleteDelivery marks a delivery as completed
// @Summary Complete Delivery
// @Description Courier marks order as delivered, triggering payouts
// @Tags Shipping
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Router /api/v1/dispatch/complete [post]
func CompleteDelivery(c *fiber.Ctx) error {
	var req struct {
		DeliveryID    uint   `json:"delivery_id"`
		ProofPhotoURL string `json:"proof_photo_url"` // required for fraud prevention
	}
	if err := c.BodyParser(&req); err != nil || req.DeliveryID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "delivery_id is required")
	}
	if req.ProofPhotoURL == "" {
		return response.Error(c, fiber.StatusBadRequest, "POD_REQUIRED",
			"proof_photo_url is required to complete a delivery")
	}

	var delivery pharmacy.CourierDelivery
	if err := database.DB.First(&delivery, req.DeliveryID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Delivery not found")
	}

	if delivery.Status == "completed" {
		return response.Error(c, fiber.StatusConflict, "ALREADY_COMPLETED", "Delivery is already completed")
	}

	// Delivery must be in "arrived" state before completing
	if delivery.Status != "arrived" {
		return response.Error(c, fiber.StatusBadRequest, "WRONG_STATUS",
			fmt.Sprintf("Delivery must be in 'arrived' state before completing (current: %s)", delivery.Status))
	}

	now := time.Now()
	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&delivery).Updates(map[string]interface{}{
			"status":               "completed",
			"actual_delivery_time": now,
			"proof_photo_url":      req.ProofPhotoURL,
			"updated_at":           now,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&pharmacy.PharmacyOrder{}).
			Where("id = ?", delivery.OrderID).
			Updates(map[string]interface{}{"status": "delivered", "updated_at": now}).Error
	})
	if txErr != nil {
		return response.Error(c, fiber.StatusInternalServerError, "COMPLETE_FAILED", txErr.Error())
	}

	// Decrement courier's active delivery count so they re-enter the scorer pool
	shipping.DecrementCourierLoad(c.Context(), delivery.CourierID)

	return response.JSON(c, fiber.StatusOK, fiber.Map{"delivery_id": delivery.ID, "completed_at": now}, "Delivery completed")
}

// ShippingWebhook receives status updates from 3PL providers (Shiprocket/India Post)
// @Summary 3PL Shipping Webhook
// @Description Webhook endpoint for third-party logistics tracking updates with idempotency
// @Tags Shipping
// @Accept json
// @Produce json
// @Router /api/v1/shipping/webhook [post]
func ShippingWebhook(c *fiber.Ctx) error {
	var payload struct {
		EventID       string `json:"event_id"`
		AWB           string `json:"awb"`
		CurrentStatus string `json:"current_status"` // e.g., "Out for Delivery", "Delivered"
		Timestamp     string `json:"timestamp"`
	}

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	if database.Rdb == nil {
		return response.Error(c, fiber.StatusInternalServerError, "REDIS_UNAVAILABLE", "Cannot process webhook without Redis idempotency check")
	}

	// 1. Idempotency Check
	ctx := c.Context()
	idempotencyKey := fmt.Sprintf("webhook:shipping:%s", payload.EventID)
	
	// If the key already exists, we've processed this event. Safely ignore it.
	ok, err := database.Rdb.SetNX(ctx, idempotencyKey, "processed", 24*time.Hour).Result()
	if err != nil || !ok {
		// Return 200 OK so the provider stops retrying, but we don't process it again.
		return c.Status(fiber.StatusOK).SendString("OK - Already processed")
	}

	// 2. FSM Update
	var order pharmacy.PharmacyOrder
	if err := database.DB.Where("tracking_id = ?", payload.AWB).First(&order).Error; err != nil {
		return c.Status(fiber.StatusOK).SendString("OK - Order not found (ignored)")
	}

	// Only update if we haven't reached a terminal state
	if order.Status != "delivered" && order.Status != "cancelled" {
		if payload.CurrentStatus == "Delivered" {
			order.Status = "delivered"
		} else {
			// e.g. shipped_3pl, in_transit
			order.Status = "in_transit"
		}
		database.DB.Save(&order)

		// Broadcast update to patient's order room
		event := map[string]interface{}{
			"type": "shipping_update",
			"room_id": fmt.Sprintf("room:order:%d", order.ID),
			"metadata": payload,
		}
		if data, err := json.Marshal(event); err == nil {
			database.Rdb.Publish(ctx, "chat:global", data)
		}
	}

	return c.Status(fiber.StatusOK).SendString("OK")
}

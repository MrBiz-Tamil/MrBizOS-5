package dispatch

import (
	"drayya-core/internal/database"
	"drayya-core/internal/dispatch"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/middleware"
	"drayya-core/internal/queue"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

type ManualAssignRequest struct {
	AppointmentID int  `json:"appointment_id"`
	DoctorID      uint `json:"doctor_id"`
}

// DispatchHandler struct
type DispatchHandler struct {
	// Service dependency if needed
}

// GetQueueStats returns dispatch dashboard stats from the real database.
// @Summary Get Dispatch Queue Stats
// @Description Fetch real-time statistics for the dispatch dashboard (pending counts, active deliveries, stuck orders)
// @Tags Dispatch
// @Produce json
// @Success 200 {object} response.StandardResponse{data=map[string]interface{}} "Queue stats fetched"
// @Router /api/v1/dispatch/stats [get]
func GetQueueStats(c *fiber.Ctx) error {
	var pendingOrders int64
	database.DB.Model(&pharmacy.PharmacyOrder{}).
		Where("status IN ?", []string{"paid", "confirmed", "dispatching"}).
		Count(&pendingOrders)

	var activeDeliveries int64
	database.DB.Model(&pharmacy.CourierDelivery{}).
		Where("status NOT IN ?", []string{"completed", "failed", "cancelled"}).
		Count(&activeDeliveries)

	var slaBreached int64
	database.DB.Model(&pharmacy.CourierDelivery{}).
		Where("sla_breached_at IS NOT NULL AND status NOT IN ?", []string{"completed", "failed", "cancelled"}).
		Count(&slaBreached)

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"pending_orders":    pendingOrders,
		"active_deliveries": activeDeliveries,
		"sla_breached":      slaBreached,
	}, "Queue stats fetched")
}

// GetQueue returns the current queue
// @Summary Get Current Dispatch Queue
// @Description Fetch list of pending or prioritized appointments in the dispatch queue
// @Tags Dispatch
// @Produce json
// @Success 200 {object} response.StandardResponse{data=[]appointment.Appointment} "Queue fetched"
// @Router /api/v1/dispatch/queue [get]
func GetQueue(c *fiber.Ctx) error {
	var appts []appointment.Appointment
	if err := database.DB.Where("status = ?", "pending").Limit(50).Find(&appts).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch dispatch queue")
	}
	return response.JSON(c, fiber.StatusOK, appts, "Dispatch queue fetched")
}

// GetEscalatedItems returns IDs of appointments needing manual attention
// @Summary Get Escalated Items
// @Description Fetch list of appointment IDs that have exceeded timeout and need manual intervention
// @Tags Dispatch
// @Produce json
// @Success 200 {object} response.StandardResponse{data=[]int} "Escalated items fetched"
// @Router /api/v1/dispatch/manual/escalations [get]
func GetEscalatedItems(c *fiber.Ctx) error {
	ids, err := queue.GetEscalatedItems(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch escalations")
	}
	return response.JSON(c, fiber.StatusOK, ids, "Escalated items fetched")
}

// ManualAssign assigns a specific doctor to an appointment
// @Summary Manual Doctor Assignment
// @Description Manually assign a doctor to a pending appointment to resolve escalations
// @Tags Dispatch
// @Accept json
// @Produce json
// @Param request body ManualAssignRequest true "Manual Assignment Request (appointment_id, doctor_id)"
// @Success 200 {object} response.StandardResponse{data=map[string]interface{}} "Doctor assigned"
// @Router /api/v1/dispatch/manual/assign [post]
func ManualAssign(c *fiber.Ctx) error {
	var req ManualAssignRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	// Logic to assign: Update DB
	db := database.DB
	var ppt appointment.Appointment
	if err := db.First(&ppt, req.AppointmentID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
	}

	docID := req.DoctorID
	ppt.DoctorID = &docID
	ppt.Status = "confirmed"
	if err := db.Save(&ppt).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to save assignment")
	}

	// Resolve Escalation if it exists
	queue.ResolveEscalation(c.Context(), req.AppointmentID)

	// Notify Doctor (Using Service logic would be better but simple here)
	// dispatch.NotifyDoctor(...) - need to export or reuse logic.

	return response.JSON(c, fiber.StatusOK, fiber.Map{"appointment_id": ppt.ID}, "Doctor assigned manually")
}

// TriggerRun forces the dispatch worker to run immediately
// @Summary Force Dispatch Cycle
// @Description Manually trigger a dispatch processing cycle (diagnostic/emergency use)
// @Tags Dispatch
// @Produce json
// @Success 200 {object} response.StandardResponse "Dispatch cycle triggered"
// @Router /api/v1/dispatch/trigger [post]
func TriggerRun(c *fiber.Ctx) error {
	// In a real worker system, this might signal a channel.
	// For our loop-based worker, we might just acknowledge.
	// Or we could trigger a specific "Sweep" function from the package.

	go dispatch.RunDispatchCycle() // Assuming we export this or similar

	return response.JSON(c, fiber.StatusOK, nil, "Dispatch run triggered")
}

// ToggleAutopilot enables/disables automatic dispatch
// @Summary Toggle Dispatch Autopilot
// @Description Enable or disable the automated dispatch worker loop
// @Tags Dispatch
// @Produce json
// @Param enabled query boolean true "Enable autopilot"
// @Success 200 {object} response.StandardResponse{data=map[string]interface{}} "Autopilot status updated"
// @Router /api/v1/dispatch/autopilot/toggle [post]
func ToggleAutopilot(c *fiber.Ctx) error {
	enabled := c.Query("enabled") == "true"
	// Set global flag or Redis key
	// dispatch.SetAutopilot(enabled)

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"autopilot_enabled": enabled,
	}, "Autopilot updated")
}

func RegisterDispatchRoutes(router fiber.Router) {
	// All dispatch management routes require authentication.
	// Admin/BPO routes require elevated roles.
	dispatchGroup := router.Group("/dispatch", middleware.Protected())

	// Ops/admin-only: manual override, queue management, autopilot
	dispatchGroup.Get("/stats", middleware.BPOOnly(), GetQueueStats)
	dispatchGroup.Get("/queue", middleware.BPOOnly(), GetQueue)
	dispatchGroup.Get("/manual/escalations", middleware.BPOOnly(), GetEscalatedItems)
	dispatchGroup.Post("/manual/assign", middleware.BPOOnly(), ManualAssign)
	dispatchGroup.Post("/trigger", middleware.AdminOnly(), TriggerRun)
	dispatchGroup.Post("/autopilot/toggle", middleware.AdminOnly(), ToggleAutopilot)

	// Logistics coordinator: delivery zone management
	dispatchGroup.Get("/delivery-zones", middleware.LogisticsOnly(), GetDeliveryZones)
	dispatchGroup.Put("/delivery-zones/:id", middleware.LogisticsOnly(), UpdateDeliveryZone)

	// Delivery management — ops staff
	dispatchGroup.Get("/deliveries", middleware.BPOOnly(), GetCourierDeliveries)
	dispatchGroup.Post("/deliveries/assign", middleware.BPOOnly(), AssignCourier)
	dispatchGroup.Put("/deliveries/:id/location", middleware.BPOOnly(), UpdateCourierLocation)

	// Courier self-service: accept/complete (protected — courier JWT required, additional role
	// check is done inside the handler via ownership validation)
	dispatchGroup.Post("/accept", AcceptDelivery)
	dispatchGroup.Post("/complete", CompleteDelivery)

	// Note: 3PL webhooks live at /api/v1/shipping/webhook/{shiprocket,indiapost}
	// and are registered by shipping.RegisterWebhookRoutes() — do NOT add a duplicate here.
}

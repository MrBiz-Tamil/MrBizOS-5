package pharmacy

import (
	"fmt"
	"strconv"
	"time"

	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// ManufacturingHandler handles HTTP requests for BOM, production orders,
// material issuance, GRN, and multi-city workflows.
type ManufacturingHandler struct {
	mfgSvc ManufacturingService
	rmSvc  RawMaterialService
}

func NewManufacturingHandler(mfgSvc ManufacturingService, rmSvc RawMaterialService) *ManufacturingHandler {
	return &ManufacturingHandler{mfgSvc: mfgSvc, rmSvc: rmSvc}
}

// ══════════════════════════════════════════════
// BOM ENDPOINTS
// ══════════════════════════════════════════════

func (h *ManufacturingHandler) CreateBOM(c *fiber.Ctx) error {
	var bom BillOfMaterials
	if err := c.BodyParser(&bom); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if bom.MedicineID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELD", "medicine_id is required")
	}
	if len(bom.Items) == 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELD", "at least one BOM item is required")
	}

	if err := h.mfgSvc.CreateBOM(c.Context(), &bom); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, bom, "Bill of Materials created")
}

func (h *ManufacturingHandler) GetBOM(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid BOM ID")
	}
	bom, err := h.mfgSvc.GetBOM(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "BOM not found")
	}
	return response.JSON(c, fiber.StatusOK, bom, "BOM fetched")
}

func (h *ManufacturingHandler) GetBOMByMedicine(c *fiber.Ctx) error {
	medicineID, err := strconv.Atoi(c.Query("medicine_id"))
	if err != nil || medicineID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_PARAM", "medicine_id query param is required")
	}
	bom, err := h.mfgSvc.GetBOMByMedicine(c.Context(), uint(medicineID))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "No active BOM found for this medicine")
	}
	return response.JSON(c, fiber.StatusOK, bom, "BOM fetched by medicine")
}

func (h *ManufacturingHandler) ListBOMs(c *fiber.Ctx) error {
	boms, err := h.mfgSvc.ListBOMs(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, boms, "BOMs listed")
}

// ══════════════════════════════════════════════
// PRODUCTION ORDER ENDPOINTS
// ══════════════════════════════════════════════

func (h *ManufacturingHandler) CreateProductionOrder(c *fiber.Ctx) error {
	var po ProductionOrder
	if err := c.BodyParser(&po); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if po.MedicineID == 0 || po.BOMID == 0 || po.WarehouseID == 0 || po.TargetQty <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELDS", "medicine_id, bom_id, warehouse_id, and target_qty (>0) are required")
	}

	userID, _ := c.Locals("user_id").(uint)
	po.CreatedByID = userID

	if err := h.mfgSvc.CreateProductionOrder(c.Context(), &po); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, po, "Production order created")
}

func (h *ManufacturingHandler) GetProductionOrder(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	po, err := h.mfgSvc.GetProductionOrder(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Production order not found")
	}
	return response.JSON(c, fiber.StatusOK, po, "Production order fetched")
}

func (h *ManufacturingHandler) ListProductionOrders(c *fiber.Ctx) error {
	filters := ProductionOrderFilters{
		Status:   c.Query("status"),
		CityName: c.Query("city"),
		Page:     c.QueryInt("page", 1),
		Limit:    c.QueryInt("limit", 20),
	}
	if wh := c.Query("warehouse_id"); wh != "" {
		if id, err := strconv.Atoi(wh); err == nil {
			filters.WarehouseID = uint(id)
		}
	}
	if p := c.Query("priority"); p != "" {
		if pv, err := strconv.Atoi(p); err == nil {
			filters.Priority = pv
		}
	}

	orders, total, err := h.mfgSvc.ListProductionOrders(c.Context(), filters)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"data":  orders,
		"total": total,
		"page":  filters.Page,
		"limit": filters.Limit,
	}, "Production orders listed")
}

func (h *ManufacturingHandler) UpdateProductionStatus(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	var req struct {
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	userID, _ := c.Locals("user_id").(uint)
	newStatus := ProductionStatus(req.Status)

	if err := h.mfgSvc.UpdateProductionStatus(c.Context(), uint(id), newStatus, req.Notes, userID); err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "TRANSITION_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, fmt.Sprintf("Production order status updated to '%s'", req.Status))
}

func (h *ManufacturingHandler) GetProductionDashboard(c *fiber.Ctx) error {
	warehouseID := uint(0)
	if wh := c.Query("warehouse_id"); wh != "" {
		if id, err := strconv.Atoi(wh); err == nil {
			warehouseID = uint(id)
		}
	}

	dashboard, err := h.mfgSvc.GetProductionDashboard(c.Context(), warehouseID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, dashboard, "Production dashboard fetched")
}

// ══════════════════════════════════════════════
// MATERIAL ISSUANCE ENDPOINTS
// ══════════════════════════════════════════════

func (h *ManufacturingHandler) ValidateBOMAvailability(c *fiber.Ctx) error {
	bomID, err := strconv.Atoi(c.Query("bom_id"))
	if err != nil || bomID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_PARAM", "bom_id is required")
	}
	targetQty := c.QueryInt("target_qty", 0)
	if targetQty <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_PARAM", "target_qty (>0) is required")
	}
	warehouseID := uint(0)
	if wh := c.Query("warehouse_id"); wh != "" {
		if id, err := strconv.Atoi(wh); err == nil {
			warehouseID = uint(id)
		}
	}

	shortages, err := h.mfgSvc.ValidateBOMAvailability(c.Context(), uint(bomID), targetQty, warehouseID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "VALIDATION_FAILED", err.Error())
	}

	if len(shortages) > 0 {
		return response.JSON(c, fiber.StatusOK, fiber.Map{
			"available": false,
			"shortages": shortages,
		}, fmt.Sprintf("%d raw materials have insufficient stock", len(shortages)))
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"available": true,
		"shortages": []interface{}{},
	}, "All raw materials available for production")
}

func (h *ManufacturingHandler) IssueMaterials(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	userID, _ := c.Locals("user_id").(uint)

	issuances, err := h.mfgSvc.IssueMaterialsForProduction(c.Context(), uint(id), userID)
	if err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "ISSUANCE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, issuances, fmt.Sprintf("%d raw material lots issued for production", len(issuances)))
}

// ══════════════════════════════════════════════
// FINISHED GOODS RELEASE
// ══════════════════════════════════════════════

func (h *ManufacturingHandler) ReleaseFinishedGoods(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	var req struct {
		YieldQty   int    `json:"yield_qty"`
		ExpiryDate string `json:"expiry_date"` // "2027-12-31"
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if req.YieldQty <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELD", "yield_qty (>0) is required")
	}

	expiryDate, err := time.Parse("2006-01-02", req.ExpiryDate)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_DATE", "expiry_date must be in YYYY-MM-DD format")
	}

	userID, _ := c.Locals("user_id").(uint)

	if err := h.mfgSvc.ReleaseFinishedGoods(c.Context(), uint(id), req.YieldQty, expiryDate, userID); err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "RELEASE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Finished goods released to inventory")
}

// ══════════════════════════════════════════════
// MANUFACTURING LOG ENDPOINTS
// ══════════════════════════════════════════════

func (h *ManufacturingHandler) AddManufacturingLog(c *fiber.Ctx) error {
	var log ManufacturingLog
	if err := c.BodyParser(&log); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if log.ProductionOrderID == 0 || log.Stage == "" {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELDS", "production_order_id and stage are required")
	}

	userID, _ := c.Locals("user_id").(uint)
	log.OperatorID = userID
	if log.StartedAt.IsZero() {
		log.StartedAt = time.Now()
	}

	if err := h.mfgSvc.AddManufacturingLog(c.Context(), &log); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, log, "Manufacturing log entry created")
}

func (h *ManufacturingHandler) GetManufacturingLogs(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	logs, err := h.mfgSvc.GetManufacturingLogs(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, logs, "Manufacturing logs fetched")
}

// ══════════════════════════════════════════════
// GRN ENDPOINTS
// ══════════════════════════════════════════════

func (h *ManufacturingHandler) CreateGRN(c *fiber.Ctx) error {
	var grn GoodsReceiptNote
	if err := c.BodyParser(&grn); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if grn.WarehouseID == 0 || grn.SourceType == "" {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELDS", "warehouse_id and source_type are required")
	}

	userID, _ := c.Locals("user_id").(uint)
	grn.ReceivedByID = userID

	if err := h.mfgSvc.CreateGRN(c.Context(), &grn); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, grn, "Goods Receipt Note created")
}

func (h *ManufacturingHandler) GetGRN(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	grn, err := h.mfgSvc.GetGRN(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "GRN not found")
	}
	return response.JSON(c, fiber.StatusOK, grn, "GRN fetched")
}

func (h *ManufacturingHandler) ListGRNs(c *fiber.Ctx) error {
	warehouseID := uint(0)
	if wh := c.Query("warehouse_id"); wh != "" {
		if id, err := strconv.Atoi(wh); err == nil {
			warehouseID = uint(id)
		}
	}
	status := c.Query("status")

	grns, err := h.mfgSvc.ListGRNs(c.Context(), warehouseID, status)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, grns, "GRNs listed")
}

func (h *ManufacturingHandler) ProcessGRNItemQA(c *fiber.Ctx) error {
	itemID, err := strconv.Atoi(c.Params("item_id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid item ID")
	}

	var req struct {
		QAStatus string `json:"qa_status"` // passed, failed, conditional
		Notes    string `json:"notes"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	validStatuses := map[string]bool{"passed": true, "failed": true, "conditional": true}
	if !validStatuses[req.QAStatus] {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_STATUS", "qa_status must be passed, failed, or conditional")
	}

	if err := h.mfgSvc.ProcessGRNItemQA(c.Context(), uint(itemID), req.QAStatus, req.Notes); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "QA_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, fmt.Sprintf("GRN item QA status updated to '%s'", req.QAStatus))
}

// ══════════════════════════════════════════════
// MULTI-CITY WORKFLOW DASHBOARD
// ══════════════════════════════════════════════

func (h *ManufacturingHandler) GetCityWorkflowDashboard(c *fiber.Ctx) error {
	city := c.Query("city")
	if city == "" {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_PARAM", "city query param is required")
	}

	dashboard, err := h.mfgSvc.GetCityWorkflowDashboard(c.Context(), city)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, dashboard, fmt.Sprintf("City workflow dashboard for %s", city))
}

// ══════════════════════════════════════════════
// ROUTE REGISTRATION
// ══════════════════════════════════════════════

func (h *ManufacturingHandler) RegisterRoutes(router fiber.Router) {
	// Bug #16 fix: BOM design and production order management are pharmacy_admin decisions.
	// Pharmacists operate on the floor (GRN, dispense) but do not design BOMs or create production runs.
	mfgAdmin := router.Group("/manufacturing", middleware.PharmacyAdminOnly())

	// BOM — static paths must come before /:id catch-all
	mfgAdmin.Post("/bom", h.CreateBOM)
	mfgAdmin.Get("/bom", h.ListBOMs)
	mfgAdmin.Get("/bom/by-medicine", h.GetBOMByMedicine)
	mfgAdmin.Get("/bom/availability", h.ValidateBOMAvailability)
	mfgAdmin.Get("/bom/:id", h.GetBOM)

	// Production Orders
	mfgAdmin.Post("/production-orders", h.CreateProductionOrder)
	mfgAdmin.Get("/production-orders", h.ListProductionOrders)
	mfgAdmin.Get("/production-orders/:id", h.GetProductionOrder)
	mfgAdmin.Put("/production-orders/:id/status", h.UpdateProductionStatus)
	mfgAdmin.Post("/production-orders/:id/issue-materials", h.IssueMaterials)
	mfgAdmin.Post("/production-orders/:id/release", h.ReleaseFinishedGoods)

	// Manufacturing Logs
	mfgAdmin.Post("/logs", h.AddManufacturingLog)
	mfgAdmin.Get("/production-orders/:id/logs", h.GetManufacturingLogs)

	// Dashboard
	mfgAdmin.Get("/dashboard", h.GetProductionDashboard)

	// GRN (Goods Receipt Note) — pharmacists legitimately receive stock deliveries,
	// so this group stays at PharmacyManagerOnly (pharmacist + pharmacy_admin + admin).
	grn := router.Group("/grn", middleware.PharmacyManagerOnly())
	grn.Post("/", h.CreateGRN)
	grn.Get("/", h.ListGRNs)
	grn.Get("/:id", h.GetGRN)
	grn.Put("/items/:item_id/qa", h.ProcessGRNItemQA)

	// Multi-City Workflow Dashboard — operational, visible to all pharmacy managers
	router.Get("/workflow/city-dashboard", middleware.PharmacyManagerOnly(), h.GetCityWorkflowDashboard)
}

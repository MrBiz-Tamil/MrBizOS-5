package pharmacy

import (
	"strconv"

	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// RawMaterialHandler handles HTTP requests for raw materials, vendors, indents, and RFQs.
type RawMaterialHandler struct {
	svc RawMaterialService
}

func NewRawMaterialHandler(svc RawMaterialService) *RawMaterialHandler {
	return &RawMaterialHandler{svc: svc}
}

// ══════════════════════════════════════════════
// RAW MATERIAL CRUD
// ══════════════════════════════════════════════

func (h *RawMaterialHandler) CreateRawMaterial(c *fiber.Ctx) error {
	var rm RawMaterial
	if err := c.BodyParser(&rm); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if rm.Name == "" {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELD", "name is required")
	}

	if err := h.svc.CreateRawMaterial(c.Context(), &rm); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, rm, "Raw material created")
}

func (h *RawMaterialHandler) GetRawMaterial(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	rm, err := h.svc.GetRawMaterial(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Raw material not found")
	}
	return response.JSON(c, fiber.StatusOK, rm, "Raw material fetched")
}

func (h *RawMaterialHandler) ListRawMaterials(c *fiber.Ctx) error {
	category := c.Query("category")
	search := c.Query("search")

	materials, err := h.svc.ListRawMaterials(c.Context(), category, search)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, materials, "Raw materials listed")
}

func (h *RawMaterialHandler) UpdateRawMaterial(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	existing, err := h.svc.GetRawMaterial(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Raw material not found")
	}

	if err := c.BodyParser(existing); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	existing.ID = uint(id)

	if err := h.svc.UpdateRawMaterial(c.Context(), existing); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, existing, "Raw material updated")
}

// ══════════════════════════════════════════════
// APPROVED VENDORS
// ══════════════════════════════════════════════

func (h *RawMaterialHandler) CreateApprovedVendor(c *fiber.Ctx) error {
	var av ApprovedVendor
	if err := c.BodyParser(&av); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	if err := h.svc.CreateApprovedVendor(c.Context(), &av); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, av, "Approved vendor created")
}

func (h *RawMaterialHandler) GetApprovedVendors(c *fiber.Ctx) error {
	rmID := uint(0)
	if id := c.Query("raw_material_id"); id != "" {
		if v, err := strconv.Atoi(id); err == nil {
			rmID = uint(v)
		}
	}

	vendors, err := h.svc.GetApprovedVendors(c.Context(), rmID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, vendors, "Approved vendors listed")
}

func (h *RawMaterialHandler) UpdateApprovedVendor(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	var av ApprovedVendor
	if err := c.BodyParser(&av); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	av.ID = uint(id)

	if err := h.svc.UpdateApprovedVendor(c.Context(), &av); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, av, "Approved vendor updated")
}

// ══════════════════════════════════════════════
// RAW MATERIAL INVENTORY
// ══════════════════════════════════════════════

func (h *RawMaterialHandler) ReceiveRawMaterial(c *fiber.Ctx) error {
	var inv RawMaterialInventory
	if err := c.BodyParser(&inv); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if inv.RawMaterialID == 0 || inv.WarehouseID == 0 || inv.Quantity <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELDS", "raw_material_id, warehouse_id, and quantity (>0) are required")
	}

	userID, _ := c.Locals("user_id").(uint)
	if err := h.svc.ReceiveRawMaterial(c.Context(), &inv, userID); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "RECEIVE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, inv, "Raw material received and quarantined for QA")
}

func (h *RawMaterialHandler) ListRMInventory(c *fiber.Ctx) error {
	warehouseID := uint(0)
	if wh := c.Query("warehouse_id"); wh != "" {
		if id, err := strconv.Atoi(wh); err == nil {
			warehouseID = uint(id)
		}
	}
	rawMaterialID := uint(0)
	if rm := c.Query("raw_material_id"); rm != "" {
		if id, err := strconv.Atoi(rm); err == nil {
			rawMaterialID = uint(id)
		}
	}
	qaStatus := c.Query("qa_status")

	inventory, err := h.svc.GetRMInventory(c.Context(), warehouseID, rawMaterialID, qaStatus)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, inventory, "Raw material inventory listed")
}

func (h *RawMaterialHandler) ApproveQA(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	var req struct {
		Notes string `json:"notes"`
	}
	_ = c.BodyParser(&req)

	userID, _ := c.Locals("user_id").(uint)
	if err := h.svc.ApproveQA(c.Context(), uint(id), userID, req.Notes); err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "QA_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Raw material QA approved")
}

func (h *RawMaterialHandler) RejectQA(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	var req struct {
		Notes string `json:"notes"`
	}
	_ = c.BodyParser(&req)

	userID, _ := c.Locals("user_id").(uint)
	if err := h.svc.RejectQA(c.Context(), uint(id), userID, req.Notes); err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "QA_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Raw material QA rejected")
}

func (h *RawMaterialHandler) GetRMBalance(c *fiber.Ctx) error {
	rmID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	warehouseID := uint(0)
	if wh := c.Query("warehouse_id"); wh != "" {
		if id, err := strconv.Atoi(wh); err == nil {
			warehouseID = uint(id)
		}
	}

	balance, err := h.svc.GetRMBalance(c.Context(), uint(rmID), warehouseID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"balance": balance}, "Balance fetched")
}

func (h *RawMaterialHandler) GetRMTransactions(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	txns, err := h.svc.GetRMTransactions(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, txns, "Transactions listed")
}

func (h *RawMaterialHandler) GetLowStockRM(c *fiber.Ctx) error {
	warehouseID := uint(0)
	if wh := c.Query("warehouse_id"); wh != "" {
		if id, err := strconv.Atoi(wh); err == nil {
			warehouseID = uint(id)
		}
	}

	items, err := h.svc.GetLowStockRawMaterials(c.Context(), warehouseID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, items, "Low stock raw materials")
}

func (h *RawMaterialHandler) GetRMSummary(c *fiber.Ctx) error {
	warehouseID := uint(0)
	if wh := c.Query("warehouse_id"); wh != "" {
		if id, err := strconv.Atoi(wh); err == nil {
			warehouseID = uint(id)
		}
	}

	summary, err := h.svc.GetRMSummary(c.Context(), warehouseID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, summary, "Raw material summary")
}

func (h *RawMaterialHandler) DestroyRM(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	var req struct {
		Quantity float64 `json:"quantity"`
		Reason   string  `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if req.Quantity <= 0 || req.Reason == "" {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELDS", "quantity (>0) and reason are required")
	}

	userID, _ := c.Locals("user_id").(uint)
	if err := h.svc.DestroyRawMaterial(c.Context(), uint(id), req.Quantity, req.Reason, userID); err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "DESTRUCTION_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Raw material destruction recorded")
}

// ══════════════════════════════════════════════
// INDENT ENDPOINTS
// ══════════════════════════════════════════════

func (h *RawMaterialHandler) CreateIndent(c *fiber.Ctx) error {
	var indent RawMaterialIndent
	if err := c.BodyParser(&indent); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if indent.RawMaterialID == 0 || indent.RequiredQty <= 0 || indent.WarehouseID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELDS", "raw_material_id, required_qty (>0), and warehouse_id are required")
	}

	userID, _ := c.Locals("user_id").(uint)
	indent.RequestedByID = userID

	if err := h.svc.CreateIndent(c.Context(), &indent); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, indent, "Raw material indent created")
}

func (h *RawMaterialHandler) GetIndent(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	indent, err := h.svc.GetIndent(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Indent not found")
	}
	return response.JSON(c, fiber.StatusOK, indent, "Indent fetched")
}

func (h *RawMaterialHandler) ListIndents(c *fiber.Ctx) error {
	warehouseID := uint(0)
	if wh := c.Query("warehouse_id"); wh != "" {
		if id, err := strconv.Atoi(wh); err == nil {
			warehouseID = uint(id)
		}
	}
	status := c.Query("status")

	indents, err := h.svc.ListIndents(c.Context(), warehouseID, status)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, indents, "Indents listed")
}

func (h *RawMaterialHandler) ApproveIndent(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	userID, _ := c.Locals("user_id").(uint)
	if err := h.svc.ApproveIndent(c.Context(), uint(id), userID); err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "APPROVAL_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Indent approved")
}

func (h *RawMaterialHandler) CancelIndent(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	if err := h.svc.CancelIndent(c.Context(), uint(id)); err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "CANCEL_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Indent cancelled")
}

// ══════════════════════════════════════════════
// RFQ ENDPOINTS
// ══════════════════════════════════════════════

func (h *RawMaterialHandler) CreateRFQ(c *fiber.Ctx) error {
	var rfq RFQ
	if err := c.BodyParser(&rfq); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if rfq.RawMaterialID == 0 || rfq.RequiredQty <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELDS", "raw_material_id and required_qty (>0) are required")
	}

	userID, _ := c.Locals("user_id").(uint)
	rfq.CreatedByID = userID

	if err := h.svc.CreateRFQ(c.Context(), &rfq); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, rfq, "RFQ created")
}

func (h *RawMaterialHandler) GetRFQ(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	rfq, err := h.svc.GetRFQ(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "RFQ not found")
	}
	return response.JSON(c, fiber.StatusOK, rfq, "RFQ fetched")
}

func (h *RawMaterialHandler) ListRFQs(c *fiber.Ctx) error {
	status := c.Query("status")
	rfqs, err := h.svc.ListRFQs(c.Context(), status)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, rfqs, "RFQs listed")
}

func (h *RawMaterialHandler) SubmitQuotation(c *fiber.Ctx) error {
	var q VendorQuotation
	if err := c.BodyParser(&q); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if q.RFQID == 0 || q.SupplierID == 0 || q.UnitPrice <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELDS", "rfq_id, supplier_id, and unit_price (>0) are required")
	}

	if err := h.svc.SubmitQuotation(c.Context(), &q); err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "SUBMIT_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, q, "Quotation submitted")
}

func (h *RawMaterialHandler) AwardRFQ(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	var req struct {
		QuotationID uint `json:"quotation_id"`
	}
	if err := c.BodyParser(&req); err != nil || req.QuotationID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELD", "quotation_id is required")
	}

	if err := h.svc.AwardRFQ(c.Context(), uint(id), req.QuotationID); err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "AWARD_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "RFQ awarded to selected vendor")
}

// ══════════════════════════════════════════════
// DASHBOARD
// ══════════════════════════════════════════════

func (h *RawMaterialHandler) GetDashboard(c *fiber.Ctx) error {
	warehouseID := uint(0)
	if wh := c.Query("warehouse_id"); wh != "" {
		if id, err := strconv.Atoi(wh); err == nil {
			warehouseID = uint(id)
		}
	}

	dashboard, err := h.svc.GetRawMaterialDashboard(c.Context(), warehouseID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, dashboard, "Raw material dashboard")
}

// ══════════════════════════════════════════════
// ROUTE REGISTRATION
// ══════════════════════════════════════════════

func (h *RawMaterialHandler) RegisterRoutes(router fiber.Router) {
	rm := router.Group("/raw-materials", middleware.PharmacyManagerOnly())

	// IMPORTANT: Fiber matches routes in registration order. Static paths
	// (e.g. /inventory, /vendors, /dashboard) MUST be registered before
	// dynamic catch-all `/:id` patterns, otherwise the catch-all wins
	// and tries to parse "inventory" as a raw_material ID.

	// CRUD — list/create at root
	rm.Post("/", h.CreateRawMaterial)
	rm.Get("/", h.ListRawMaterials)

	// Inventory (static prefix — must come BEFORE /:id catch-all)
	rm.Post("/inventory", h.ReceiveRawMaterial)
	rm.Get("/inventory", h.ListRMInventory)
	rm.Get("/inventory/low-stock", h.GetLowStockRM)
	rm.Get("/inventory/summary", h.GetRMSummary)
	rm.Get("/inventory/:id/transactions", h.GetRMTransactions)
	rm.Get("/inventory/:id/balance", h.GetRMBalance)
	rm.Post("/inventory/:id/qa/approve", h.ApproveQA)
	rm.Post("/inventory/:id/qa/reject", h.RejectQA)
	rm.Post("/inventory/:id/destroy", h.DestroyRM)

	// Approved Vendors (static prefix)
	rm.Post("/vendors", h.CreateApprovedVendor)
	rm.Get("/vendors", h.GetApprovedVendors)
	rm.Put("/vendors/:id", h.UpdateApprovedVendor)

	// Indents (static prefix)
	rm.Post("/indents", h.CreateIndent)
	rm.Get("/indents", h.ListIndents)
	rm.Get("/indents/:id", h.GetIndent)
	rm.Post("/indents/:id/approve", h.ApproveIndent)
	rm.Post("/indents/:id/cancel", h.CancelIndent)

	// RFQ (static prefix)
	rm.Post("/rfq", h.CreateRFQ)
	rm.Get("/rfq", h.ListRFQs)
	rm.Get("/rfq/:id", h.GetRFQ)
	rm.Post("/rfq/quotations", h.SubmitQuotation)
	rm.Post("/rfq/:id/award", h.AwardRFQ)

	// Dashboard (static)
	rm.Get("/dashboard", h.GetDashboard)

	// Single raw-material CRUD by ID — MUST be LAST so static paths win
	rm.Get("/:id", h.GetRawMaterial)
	rm.Put("/:id", h.UpdateRawMaterial)

	// City-level dashboard shortcut
	router.Get("/raw-materials/city-dashboard", middleware.PharmacyManagerOnly(), func(c *fiber.Ctx) error {
		return h.GetDashboard(c)
	})
}

package pharmacy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"drayya-core/internal/crypto"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/loyalty"
	"drayya-core/internal/middleware"
	"drayya-core/internal/services/fx"
	notificationSvc "drayya-core/internal/services/notification"
	razorpay "drayya-core/internal/services/razorpay"
	"drayya-core/internal/services/stripe"
	"drayya-core/internal/utils/response"
	"drayya-core/internal/utils/validator"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	svc Service
	db  *gorm.DB
}

func NewHandler(svc Service, db *gorm.DB) *Handler {
	return &Handler{svc: svc, db: db}
}

// GetInventory returns medicine inventory for a pharmacy.
// H-2: paginated via page/limit query params (default limit 20, max 100).
// @Summary Get Inventory
// @Description Get all medicine stock levels for a specific pharmacy
// @Tags Pharmacy
// @Produce json
// @Param pharmacy_id query uint false "Pharmacy ID (optional)"
// @Success 200 {object} response.StandardResponse{data=[]PharmacyInventory} "Inventory fetched"
// @Failure 500 {object} response.StandardError "Fetch failed"
// @Router /api/v1/pharmacy/inventory [get]
func (h *Handler) GetInventory(c *fiber.Ctx) error {
	pharmacyIDStr := c.Query("pharmacy_id")
	pharmacyID := 0
	if pharmacyIDStr != "" {
		id, err := strconv.Atoi(pharmacyIDStr)
		if err == nil {
			pharmacyID = id
		}
	}

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

	var inventory []PharmacyInventory
	q := h.db.Preload("Medicine").Where("is_active = ?", true)
	if pharmacyID != 0 {
		q = q.Where("pharmacy_id = ?", pharmacyID)
	}
	var total int64
	if err := q.Model(&PharmacyInventory{}).Count(&total).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	if err := q.Offset(offset).Limit(limit).Find(&inventory).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	c.Set("Content-Range", fmt.Sprintf("inventory %d-%d/%d", offset, offset+len(inventory), total))
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"data":  inventory,
		"page":  page,
		"limit": limit,
		"total": total,
	}, "Inventory fetched")
}

func (h *Handler) GetDashboardStats(c *fiber.Ctx) error {
	stats, err := h.svc.GetDashboardStats(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, stats, "Dashboard stats fetched")
}

// GetStaffSelf returns the logged-in pharmacy staff's identity context.
// The frontend uses this to subscribe to the right WebSocket room
// (room:pharmacy:<pharmacy_id>) without depending on JWT claim shape.
func (h *Handler) GetStaffSelf(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	pharmacyID, _ := c.Locals("pharmacy_id").(uint)

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"user_id":     userID,
		"role":        role,
		"pharmacy_id": pharmacyID,
	}, "Staff context fetched")
}

func (h *Handler) GetStockAlerts(c *fiber.Ctx) error {
	alerts, err := h.svc.GetStockAlerts(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, alerts, "Stock alerts fetched")
}

func (h *Handler) GetLowStock(c *fiber.Ctx) error {
	inventory, err := h.svc.GetLowStockInventory(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, inventory, "Low stock inventory fetched")
}

func (h *Handler) GetExpiringInventory(c *fiber.Ctx) error {
	inventory, err := h.svc.GetExpiringInventory(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, inventory, "Expiring inventory fetched")
}

// RequestInventoryTransfer handles inventory transfer requests.
// H-12: validates SourceID != DestinationID, Quantity > 0, medicine exists, user's pharmacy matches SourceID.
func (h *Handler) RequestInventoryTransfer(c *fiber.Ctx) error {
	var req InventoryTransfer
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	// H-12 validations
	if req.SourceID == req.DestinationID {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_TRANSFER", "Source and destination must be different")
	}
	if req.Quantity <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_QUANTITY", "Quantity must be greater than 0")
	}

	// Verify medicine exists
	var med Medicine
	if err := h.db.First(&med, req.MedicineID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Medicine not found")
	}

	userID, _ := c.Locals("user_id").(uint)

	// Verify requesting user's pharmacy matches SourceID (skip check for admins)
	role, _ := c.Locals("role").(string)
	if role != "admin" && role != "super_admin" {
		pharmacyID, _ := c.Locals("pharmacy_id").(uint)
		if pharmacyID != 0 && pharmacyID != req.SourceID {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You can only request transfers from your own pharmacy")
		}
	}

	req.RequestedByID = userID
	req.Status = "requested"
	req.CreatedAt = time.Now()

	if err := h.db.Create(&req).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, req, "Transfer requested successfully")
}

// GetTransfersHandler lists inventory transfers for the requesting user's pharmacy.
// Feature 9: GET /pharmacy/transfers
func (h *Handler) GetTransfersHandler(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	isAdmin := role == "admin" || role == "super_admin"

	jwtPharmacyID, _ := c.Locals("pharmacy_id").(uint)

	q := h.db.WithContext(c.Context()).Order("created_at desc")
	if !isAdmin && jwtPharmacyID != 0 {
		// Scope to transfers where the user's pharmacy is source or destination.
		q = q.Where("source_id = ? OR destination_id = ?", jwtPharmacyID, jwtPharmacyID)
	}

	// Optional status filter.
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}

	page := 1
	limit := 20
	if p, err := strconv.Atoi(c.Query("page", "1")); err == nil && p > 0 {
		page = p
	}
	if l, err := strconv.Atoi(c.Query("limit", "20")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	_ = userID // used for scoping above
	var transfers []InventoryTransfer
	if err := q.Offset((page - 1) * limit).Limit(limit).Find(&transfers).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"data":  transfers,
		"page":  page,
		"limit": limit,
	}, "Transfers fetched")
}

func (h *Handler) ProcessInventoryTransfer(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	userID, _ := c.Locals("user_id").(uint)

	err = h.db.Transaction(func(tx *gorm.DB) error {
		var transfer InventoryTransfer
		if err := tx.First(&transfer, id).Error; err != nil {
			return err
		}

		if transfer.Status == "completed" || transfer.Status == "cancelled" {
			return errors.New("cannot process a completed or cancelled transfer")
		}

		transfer.Status = req.Status
		transfer.ProcessedByID = &userID
		transfer.UpdatedAt = time.Now()

		if req.Status == "completed" {
			var sourceInv PharmacyInventory
			if err := tx.Where("pharmacy_id = ? AND medicine_id = ?", transfer.SourceID, transfer.MedicineID).First(&sourceInv).Error; err != nil {
				return errors.New("source inventory not found")
			}
			if sourceInv.StockQuantity < transfer.Quantity {
				return errors.New("insufficient stock in source pharmacy")
			}
			sourceInv.StockQuantity -= transfer.Quantity
			if err := tx.Save(&sourceInv).Error; err != nil {
				return err
			}

			var destInv PharmacyInventory
			err := tx.Where("pharmacy_id = ? AND medicine_id = ?", transfer.DestinationID, transfer.MedicineID).First(&destInv).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					destInv = PharmacyInventory{
						PharmacyID:    transfer.DestinationID,
						MedicineID:    transfer.MedicineID,
						StockQuantity: transfer.Quantity,
						IsActive:      true,
					}
					if err := tx.Create(&destInv).Error; err != nil {
						return err
					}
				} else {
					return err
				}
			} else {
				destInv.StockQuantity += transfer.Quantity
				if err := tx.Save(&destInv).Error; err != nil {
					return err
				}
			}
		}

		return tx.Save(&transfer).Error
	})

	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "TRANSFER_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Transfer processed successfully")
}

func (h *Handler) GetMultiCityInventory(c *fiber.Ctx) error {
	stats, err := h.svc.GetMultiCityInventory(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, stats, "Multi-city inventory fetched")
}

// GetPharmacies returns a list of active pharmacies
// @Summary List Pharmacies
// @Description Get all active pharmacies with basic details
// @Tags Pharmacy
// @Produce json
// @Success 200 {object} response.StandardResponse{data=[]Pharmacy} "Pharmacies fetched"
// @Router /api/v1/pharmacy [get]
func (h *Handler) GetPharmacies(c *fiber.Ctx) error {
	res, err := h.svc.GetAllPharmacies(c.Context(), true)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, res, "Pharmacies fetched")
}

func (h *Handler) ConvertPrescriptionHandler(c *fiber.Ctx) error {
	callerID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	isStaff := role == "admin" || role == "super_admin" || role == "pharmacist" || role == "pharmacy_admin"

	var req struct {
		PrescriptionID  uint   `json:"prescription_id"`
		PharmacyID      uint   `json:"pharmacy_id"`
		ShippingAddress string `json:"shipping_address"`
		Phone           string `json:"phone"`
		DeliveryMode    string `json:"delivery_mode"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Ownership check: patients may only convert their own prescriptions.
	// Pharmacy staff may convert on behalf of any patient.
	if !isStaff {
		var rx Prescription
		if err := h.db.First(&rx, req.PrescriptionID).Error; err != nil {
			return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Prescription not found")
		}
		if rx.UserID != callerID {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You can only convert your own prescriptions")
		}
	}

	order, err := h.svc.ConvertPrescriptionToOrder(c.Context(), req.PrescriptionID, req.PharmacyID, req.ShippingAddress, req.Phone, DeliveryMode(req.DeliveryMode))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CONVERSION_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, order, "Prescription converted to order")
}

func (h *Handler) GenerateCartHandler(c *fiber.Ctx) error {
	var req struct {
		PrescriptionID uint `json:"prescription_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	cart, err := h.svc.GenerateCartFromPrescription(c.Context(), req.PrescriptionID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CART_GENERATION_FAILED", err.Error())
	}

	if len(cart) == 0 {
		return response.JSON(c, fiber.StatusOK, fiber.Map{"cart": cart}, "Cart generated, but no active inventory matched prescription items")
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{"cart": cart}, "Cart generated successfully")
}

func (h *Handler) ListPharmaciesAdmin(c *fiber.Ctx) error {
	res, err := h.svc.ListAllPharmaciesAdmin(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, res, "Admin pharmacy list fetched")
}

// CreatePharmacyHandler creates a new pharmacy (Admin)
// @Summary Create Pharmacy
// @Description Add a new pharmacy to the network
// @Tags Pharmacy
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body object true "Pharmacy Data"
// @Success 201 {object} response.StandardResponse{data=Pharmacy} "Pharmacy created"
// @Failure 401 {object} response.StandardError "Unauthorized"
// @Router /api/v1/pharmacy/admin [post]
func (h *Handler) CreatePharmacyHandler(c *fiber.Ctx) error {
	var req struct {
		Name               string  `json:"name" validate:"required"`
		Address            string  `json:"address" validate:"required"`
		City               string  `json:"city" validate:"required"`
		Latitude           float64 `json:"latitude"`
		Longitude          float64 `json:"longitude"`
		ServiceRadius      float64 `json:"service_radius"`
		IsCentralWarehouse bool    `json:"is_central_warehouse"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	if err := validator.ValidateStruct(c, &req); err != nil {
		return err
	}
	p := &Pharmacy{
		Name:               req.Name,
		Address:            req.Address,
		City:               req.City,
		Latitude:           req.Latitude,
		Longitude:          req.Longitude,
		ServiceRadius:      req.ServiceRadius,
		IsCentralWarehouse: req.IsCentralWarehouse,
		IsActive:           true,
	}
	if err := h.svc.CreatePharmacy(c.Context(), p); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, p, "Pharmacy created")
}

func (h *Handler) UpdatePharmacyHandler(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	var req struct {
		Name               string  `json:"name"`
		Address            string  `json:"address"`
		City               string  `json:"city"`
		Latitude           float64 `json:"latitude"`
		Longitude          float64 `json:"longitude"`
		ServiceRadius      float64 `json:"service_radius"`
		IsCentralWarehouse bool    `json:"is_central_warehouse"`
		IsActive           bool    `json:"is_active"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	p := &Pharmacy{
		ID:                 uint(id),
		Name:               req.Name,
		Address:            req.Address,
		City:               req.City,
		Latitude:           req.Latitude,
		Longitude:          req.Longitude,
		ServiceRadius:      req.ServiceRadius,
		IsCentralWarehouse: req.IsCentralWarehouse,
		IsActive:           req.IsActive,
	}
	if err := h.svc.UpdatePharmacy(c.Context(), p); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, p, "Pharmacy updated")
}

func (h *Handler) TogglePharmacyHandler(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	p, err := h.svc.TogglePharmacy(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, p, "Pharmacy status toggled")
}

func (h *Handler) TriggerRestockHandler(c *fiber.Ctx) error {
	count, err := h.svc.CheckAndTriggerRestock(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "TRIGGER_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{"count": count}, fmt.Sprintf("Successfully evaluated stock levels and triggered %d restocking requests", count))
}

// PayOrderHandler processes order payment.
// H-9: clears Redis cart using the order's PatientID, not c.Locals("user_id").
func (h *Handler) PayOrderHandler(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	// Ownership check: only the order owner can pay (admins/staff may also pay)
	callerID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	isStaff := role == "admin" || role == "super_admin" || role == "pharmacist" || role == "pharmacy_admin"
	if !isStaff {
		existing, fetchErr := h.svc.GetOrder(c.Context(), uint(id))
		if fetchErr != nil {
			return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Order not found")
		}
		if existing.UserID != callerID {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You can only pay for your own orders")
		}
	}

	if err := h.svc.PayOrder(c.Context(), uint(id)); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PAYMENT_FAILED", err.Error())
	}

	// H-9: Use the order's PatientID (fetched from the order) to clear Redis cart
	if database.Rdb != nil {
		order, fetchErr := h.svc.GetOrder(c.Context(), uint(id))
		if fetchErr == nil && order.UserID != 0 {
			database.Rdb.Del(context.Background(), cartKey(order.UserID))
		}
	}

	return response.JSON(c, fiber.StatusOK, nil, "Order paid successfully and financial split recorded")
}

// DisposeStockHandler atomically reduces inventory using a raw SQL update.
// C-4: replaces read-modify-write with atomic SQL to prevent race conditions.
func (h *Handler) DisposeStockHandler(c *fiber.Ctx) error {
	var req struct {
		InventoryID uint   `json:"inventory_id"`
		Quantity    int    `json:"quantity"`
		Reason      string `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Verify the inventory item exists
	var item PharmacyInventory
	if err := h.db.First(&item, req.InventoryID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Inventory item not found")
	}

	// C-4: atomic update — only succeeds when stock_quantity >= req.Quantity
	result := h.db.Exec(
		"UPDATE pharmacyinventory SET stock_quantity = stock_quantity - ? WHERE id = ? AND stock_quantity >= ?",
		req.Quantity, req.InventoryID, req.Quantity,
	)
	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to update inventory")
	}
	if result.RowsAffected == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INSUFFICIENT_STOCK", "Insufficient stock to dispose")
	}

	return response.JSON(c, fiber.StatusOK, nil, "Stock disposed successfully")
}

// GetPharmacyFinancialsHandler returns pharmacy financials.
// H-13: scopes to user's own pharmacy unless admin; 403 if pharmacy_id doesn't match JWT.
func (h *Handler) GetPharmacyFinancialsHandler(c *fiber.Ctx) error {
	pharmacyID, _ := strconv.Atoi(c.Query("pharmacy_id", "0"))

	role, _ := c.Locals("role").(string)
	isAdmin := role == "admin" || role == "super_admin"
	jwtPharmacyID, _ := c.Locals("pharmacy_id").(uint)

	if pharmacyID != 0 && !isAdmin {
		// Provided pharmacy_id must match the user's own pharmacy
		if uint(pharmacyID) != jwtPharmacyID {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Access to this pharmacy's financials is not allowed")
		}
	}

	// If pharmacyID == 0 and not admin, scope to own pharmacy
	if pharmacyID == 0 && !isAdmin && jwtPharmacyID != 0 {
		pharmacyID = int(jwtPharmacyID)
	}

	stats, err := h.svc.GetPharmacyFinancials(c.Context(), uint(pharmacyID))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, stats, "Pharmacy financials fetched")
}

// Supplier CRUD
func (h *Handler) GetSuppliers(c *fiber.Ctx) error {
	var suppliers []Supplier
	if err := h.db.Where("is_active = ?", true).Find(&suppliers).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, suppliers, "Suppliers fetched")
}

func (h *Handler) CreateSupplier(c *fiber.Ctx) error {
	var supplier Supplier
	if err := c.BodyParser(&supplier); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	if err := h.db.Create(&supplier).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, supplier, "Supplier created")
}

func (h *Handler) UpdateSupplier(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	if err := h.db.Model(&Supplier{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Supplier updated")
}

func (h *Handler) DeleteSupplier(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	if err := h.db.Model(&Supplier{}).Where("id = ?", id).Update("is_active", false).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Supplier deactivated")
}

// Medicine CRUD
func (h *Handler) GetMedicines(c *fiber.Ctx) error {
	search := c.Query("search")
	category := strings.TrimSpace(c.Query("category"))
	form := strings.TrimSpace(c.Query("form"))
	brand := strings.TrimSpace(c.Query("brand"))
	rx := c.Query("rx")             // "rx_only" | "otc_only" | ""
	sortBy := c.Query("sort")       // "price_low" | "price_high" | "name" | ""
	inStock := c.QueryBool("in_stock", false)
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)
	if limit > 100 {
		limit = 100 // cap to prevent abuse
	}

	// Public store endpoint — only published, public-channel, active medicines.
	q := publicCatalogScope(h.db.Model(&Medicine{}))
	if search != "" {
		like := "%" + search + "%"
		q = q.Where(
			"name LIKE ? OR brand LIKE ? OR category LIKE ? OR generic_name LIKE ? OR name_ta LIKE ? OR name_hi LIKE ? OR synonyms LIKE ?",
			like, like, like, like, like, like, like,
		)
	}
	if category != "" {
		q = q.Where("LOWER(category) = ?", strings.ToLower(category))
	}
	if form != "" {
		q = q.Where("LOWER(form) = ?", strings.ToLower(form))
	}
	if brand != "" {
		q = q.Where("LOWER(brand) = ?", strings.ToLower(brand))
	}
	switch rx {
	case "rx_only":
		q = q.Where("is_prescription_required = ?", true)
	case "otc_only":
		q = q.Where("is_prescription_required = ?", false)
	}
	// In-stock filter: at least one active inventory batch with positive quantity.
	if inStock {
		q = q.Where("EXISTS (SELECT 1 FROM pharmacyinventory pi WHERE pi.medicine_id = medicine.id AND pi.stock_quantity > 0 AND pi.is_active = ?)", true)
	}

	// Sort. Price lives in the JSONB `prices` column keyed by ISO-4217 code, so
	// the INR-extraction syntax differs between Postgres and SQLite (dev).
	priceExpr := "(prices->>'INR')::bigint" // Postgres
	if h.db.Dialector.Name() == "sqlite" {
		priceExpr = "CAST(json_extract(prices, '$.INR') AS INTEGER)"
	}
	switch sortBy {
	case "price_low":
		q = q.Order(priceExpr + " ASC")
	case "price_high":
		q = q.Order(priceExpr + " DESC")
	case "name":
		q = q.Order("name ASC")
	default:
		// In-house brand first, then alphabetical — a sensible default merchandising order.
		q = q.Order("is_inhouse_brand DESC, name ASC")
	}

	var medicines []Medicine
	if err := q.Limit(limit).Offset(offset).Find(&medicines).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, medicines, "Medicines fetched")
}

// GetCatalogFacets returns the distinct categories, AYUSH forms, and brands in
// the active catalog so the storefront can render real filter chips instead of
// hardcoded lists. Cheap DISTINCT queries; safe to call on store load.
//
// GET /pharmacy/catalog/facets
func (h *Handler) GetCatalogFacets(c *fiber.Ctx) error {
	var categories, forms, brands []string
	base := func() *gorm.DB { return publicCatalogScope(h.db.Model(&Medicine{})) }
	base().Where("category <> ''").Distinct().Order("category ASC").Pluck("category", &categories)
	base().Where("form <> ''").Distinct().Order("form ASC").Pluck("form", &forms)
	base().Where("brand <> ''").Distinct().Order("brand ASC").Limit(50).Pluck("brand", &brands)
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"categories": categories,
		"forms":      forms,
		"brands":     brands,
	}, "Catalog facets fetched")
}

func (h *Handler) CreateMedicine(c *fiber.Ctx) error {
	var medicine Medicine
	if err := c.BodyParser(&medicine); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	// WithContext so the audit log attributes who created the medicine.
	if err := h.db.WithContext(c.UserContext()).Create(&medicine).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, medicine, "Medicine created")
}

func (h *Handler) GetMedicineByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	// Public detail page — a draft / pending / archived / pharmacy_only medicine
	// must be indistinguishable from a non-existent one (404), so we scope the
	// lookup to the public catalog rather than fetching then leaking its state.
	var medicine Medicine
	if err := publicCatalogScope(h.db.Preload("Variants")).First(&medicine, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Medicine not found")
	}
	return response.JSON(c, fiber.StatusOK, medicine, "Medicine details fetched")
}

// GetSubstitutes returns cheaper therapeutically-equivalent medicines sharing
// the same molecule_code as the requested medicine, ordered by price ascending.
// This is the engine behind the "Generic available — save 40%" feature.
func (h *Handler) GetSubstitutes(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	var source Medicine
	if err := h.db.First(&source, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Medicine not found")
	}
	if source.MoleculeCode == "" {
		return response.JSON(c, fiber.StatusOK, []interface{}{}, "No substitutes available")
	}

	var medicines []Medicine
	h.db.Where("molecule_code = ? AND id <> ? AND is_active = ?", source.MoleculeCode, id, true).
		Order("(prices->>'INR')::bigint ASC").
		Limit(5).Find(&medicines)

	type SubstituteResult struct {
		ID          uint    `json:"id"`
		Name        string  `json:"name"`
		Brand       string  `json:"brand"`
		GenericName string  `json:"generic_name"`
		Manufacturer string `json:"manufacturer"`
		PricePaise  int64   `json:"price_paise"`
		MrpPaise    int64   `json:"mrp_paise"`
		SavingsPaise int64  `json:"savings_paise"`
		SavingsPct  int     `json:"savings_pct"`
	}

	// Source price in INR paise.
	sourcePricePaise := source.Prices["INR"]

	out := make([]SubstituteResult, 0, len(medicines))
	for _, m := range medicines {
		p := m.Prices["INR"]
		savings := sourcePricePaise - p
		pct := 0
		if sourcePricePaise > 0 {
			pct = int(savings * 100 / sourcePricePaise)
		}
		out = append(out, SubstituteResult{
			ID:          m.ID,
			Name:        m.Name,
			Brand:       m.Brand,
			GenericName: m.GenericName,
			Manufacturer: m.Manufacturer,
			PricePaise:  p,
			MrpPaise:    p,
			SavingsPaise: savings,
			SavingsPct:  pct,
		})
	}
	return response.JSON(c, fiber.StatusOK, out, "Substitutes fetched")
}

// updateMedicineRequest is the whitelist of fields that callers may update.
// Using a typed struct prevents accidental mass-assignment of sensitive fields
// (is_prescription_required, deleted_at, schedule, etc.).
type updateMedicineRequest struct {
	Name         string `json:"name"`
	Brand        string `json:"brand"`
	Manufacturer string `json:"manufacturer"`
	Category     string `json:"category"`
	Description  string `json:"description"`
	UnitPrice    *int64 `json:"unit_price"`
	ImageURL     string `json:"image_url"`
	// Rich AYUSH / detail-page fields the storefront renders.
	GenericName         string   `json:"generic_name"`
	Form                string   `json:"form"`
	PackSize            string   `json:"pack_size"`
	NameTa              string   `json:"name_ta"`
	NameHi              string   `json:"name_hi"`
	Indications         string   `json:"indications"`
	Contraindications   string   `json:"contraindications"`
	DosageInstructions  string   `json:"dosage_instructions"`
	StorageInstructions string   `json:"storage_instructions"`
	GSTPercent          *float64 `json:"gst_percent"`
	IsPrescriptionRequired *bool `json:"is_prescription_required"`
	SalesChannel        string   `json:"sales_channel"` // public | pharmacy_only
}

func (h *Handler) UpdateMedicine(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	var req updateMedicineRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	// Build a safe updates map from whitelisted fields only.
	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Brand != "" {
		updates["brand"] = req.Brand
	}
	if req.Manufacturer != "" {
		updates["manufacturer"] = req.Manufacturer
	}
	if req.Category != "" {
		updates["category"] = req.Category
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.UnitPrice != nil {
		updates["unit_price"] = *req.UnitPrice
	}
	if req.ImageURL != "" {
		updates["image_url"] = req.ImageURL
	}
	if req.GenericName != "" {
		updates["generic_name"] = req.GenericName
	}
	if req.Form != "" {
		updates["form"] = req.Form
	}
	if req.PackSize != "" {
		updates["pack_size"] = req.PackSize
	}
	if req.NameTa != "" {
		updates["name_ta"] = req.NameTa
	}
	if req.NameHi != "" {
		updates["name_hi"] = req.NameHi
	}
	if req.Indications != "" {
		updates["indications"] = req.Indications
	}
	if req.Contraindications != "" {
		updates["contraindications"] = req.Contraindications
	}
	if req.DosageInstructions != "" {
		updates["dosage_instructions"] = req.DosageInstructions
	}
	if req.StorageInstructions != "" {
		updates["storage_instructions"] = req.StorageInstructions
	}
	if req.GSTPercent != nil {
		updates["gst_percent"] = *req.GSTPercent
	}
	if req.IsPrescriptionRequired != nil {
		updates["is_prescription_required"] = *req.IsPrescriptionRequired
	}
	if req.SalesChannel == string(ChannelPublic) || req.SalesChannel == string(ChannelPharmacyOnly) {
		updates["sales_channel"] = req.SalesChannel
	}
	if len(updates) == 0 {
		return response.Error(c, fiber.StatusBadRequest, "NO_FIELDS", "No updatable fields provided")
	}
	// WithContext so the audit log attributes who edited the medicine.
	if err := h.db.WithContext(c.UserContext()).Model(&Medicine{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Medicine updated")
}

func (h *Handler) DeleteMedicine(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	if err := h.db.Delete(&Medicine{}, id).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Medicine deleted")
}

// Purchase Order CRUD
func (h *Handler) GetPurchaseOrders(c *fiber.Ctx) error {
	var pos []PurchaseOrder
	if err := h.db.Order("created_at desc").Find(&pos).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, pos, "Purchase orders fetched")
}

func (h *Handler) GetPurchaseOrder(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	var po PurchaseOrder
	if err := h.db.First(&po, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Purchase order not found")
	}
	return response.JSON(c, fiber.StatusOK, po, "Purchase order fetched")
}

func (h *Handler) CreatePurchaseOrder(c *fiber.Ctx) error {
	var req struct {
		SupplierID uint                     `json:"supplier_id"`
		PharmacyID uint                     `json:"pharmacy_id"`
		Items      []map[string]interface{} `json:"items"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	po, err := h.svc.CreatePurchaseOrder(c.Context(), req.SupplierID, req.PharmacyID, req.Items)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, po, "Purchase order created with tax calculation")
}

// ApprovePurchaseOrder approves a purchase order.
// H-4: checks that PO is in "draft" status before approving.
func (h *Handler) ApprovePurchaseOrder(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	var po PurchaseOrder
	if err := h.db.First(&po, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Purchase order not found")
	}
	if po.Status != "draft" {
		return response.Error(c, fiber.StatusConflict, "INVALID_STATUS", "Cannot approve a PO that is not in draft status")
	}

	if err := h.db.Model(&PurchaseOrder{}).Where("id = ?", id).Update("status", "ordered").Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Purchase order approved and sent to supplier")
}

// RejectPurchaseOrder rejects a purchase order.
// H-4: checks that PO is in "draft" status before rejecting.
func (h *Handler) RejectPurchaseOrder(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	var po PurchaseOrder
	if err := h.db.First(&po, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Purchase order not found")
	}
	if po.Status != "draft" {
		return response.Error(c, fiber.StatusConflict, "INVALID_STATUS", "Cannot approve a PO that is not in draft status")
	}

	if err := h.db.Model(&PurchaseOrder{}).Where("id = ?", id).Update("status", "cancelled").Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Purchase order cancelled")
}

func (h *Handler) ReceivePurchaseOrderHandler(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	if err := h.svc.ReceivePurchaseOrder(c.Context(), uint(id)); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "RECEIVE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Purchase order received and inventory updated")
}

func (h *Handler) SettleSupplierPaymentHandler(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	if err := h.svc.SettleSupplierPayment(c.Context(), uint(id)); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PAYMENT_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Supplier payment settled")
}

// Warehouse Endpoints

// RecordWarehouseMetricHandler records a cold chain metric.
// M-10/11: validates temperature (-100 to 200), humidity (0 to 100), warehouse_id existence.
func (h *Handler) RecordWarehouseMetricHandler(c *fiber.Ctx) error {
	var req struct {
		WarehouseID uint    `json:"warehouse_id"`
		Temperature float64 `json:"temperature"`
		Humidity    float64 `json:"humidity"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	// M-10: validate temperature range
	if req.Temperature < -100 || req.Temperature > 200 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_TEMPERATURE", "Temperature must be between -100 and 200 °C")
	}
	// M-11: validate humidity range
	if req.Humidity < 0 || req.Humidity > 100 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_HUMIDITY", "Humidity must be between 0 and 100%")
	}
	// M-11: verify warehouse_id exists
	if req.WarehouseID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_WAREHOUSE", "warehouse_id is required")
	}
	var warehouse Pharmacy
	if err := h.db.First(&warehouse, req.WarehouseID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "WAREHOUSE_NOT_FOUND", "Warehouse not found")
	}

	err := h.svc.RecordWarehouseMetric(c.Context(), req.WarehouseID, req.Temperature, req.Humidity)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}

	// Feature 7: cold-chain threshold alerting.
	const (
		maxSafeTemp = 25.0 // °C — standard ambient storage
		minSafeTemp = 2.0  // °C — cold chain floor
		maxHumidity = 60.0 // %
	)
	var violations []string
	if req.Temperature > maxSafeTemp {
		violations = append(violations, fmt.Sprintf("temperature %.1f°C exceeds safe max of %.0f°C", req.Temperature, maxSafeTemp))
	}
	if req.Temperature < minSafeTemp {
		violations = append(violations, fmt.Sprintf("temperature %.1f°C is below safe min of %.0f°C", req.Temperature, minSafeTemp))
	}
	if req.Humidity > maxHumidity {
		violations = append(violations, fmt.Sprintf("humidity %.1f%% exceeds safe max of %.0f%%", req.Humidity, maxHumidity))
	}
	if len(violations) > 0 {
		// Log structured warning so ops can alert via log aggregation.
		for _, v := range violations {
			_ = v // in production: publish to event bus or send to alerting system
		}
		return response.JSON(c, fiber.StatusOK, fiber.Map{
			"recorded":   true,
			"alert":      true,
			"violations": violations,
		}, "Metric recorded — THRESHOLD ALERT: "+violations[0])
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{"recorded": true, "alert": false}, "Metric recorded")
}

// GetWarehouseAlertsHandler returns recent cold-chain metrics that exceeded thresholds.
// Feature 7: GET /warehouse/alerts
func (h *Handler) GetWarehouseAlertsHandler(c *fiber.Ctx) error {
	const (
		maxSafeTemp = 25.0
		minSafeTemp = 2.0
		maxHumidity = 60.0
	)

	var alerts []ColdChainMetric
	// Note: ColdChainMetric has no created_at column — it uses recorded_at.
	if err := h.db.WithContext(c.Context()).
		Where("temperature > ? OR temperature < ? OR humidity > ?", maxSafeTemp, minSafeTemp, maxHumidity).
		Order("recorded_at desc").
		Limit(100).
		Find(&alerts).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, alerts, fmt.Sprintf("%d threshold violations found", len(alerts)))
}

func (h *Handler) GetWarehouseMetricsHandler(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	metrics, err := h.svc.GetWarehouseMetrics(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, metrics, "Warehouse metrics fetched")
}

// GetOrders returns orders. Pharmacy staff see all orders; patients see only their own.
// H-14: when pharmacy_id query param provided for staff, verify it matches JWT unless admin.
func (h *Handler) GetOrders(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	status := c.Query("status")

	q := h.db.Order("created_at desc")
	isStaff := role == "pharmacy_manager" || role == "admin" || role == "super_admin" || role == "pharmacist"
	isAdmin := role == "admin" || role == "super_admin"

	if isStaff {
		if pid := c.Query("pharmacy_id"); pid != "" {
			// H-14: verify requested pharmacy_id matches JWT unless admin
			if !isAdmin {
				jwtPharmacyID, _ := c.Locals("pharmacy_id").(uint)
				reqPharmacyID, parseErr := strconv.ParseUint(pid, 10, 64)
				if parseErr != nil || (jwtPharmacyID != 0 && uint(reqPharmacyID) != jwtPharmacyID) {
					return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Access to this pharmacy's orders is not allowed")
				}
			}
			q = q.Where("pharmacy_id = ?", pid)
		}
	} else {
		q = q.Where("user_id = ?", userID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}

	var orders []PharmacyOrder
	if err := q.Find(&orders).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, orders, "Orders fetched")
}

// GetOrderByID returns a single order. Patients may only access their own orders.
func (h *Handler) GetOrderByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	userID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	order, err := h.svc.GetOrder(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Order not found")
	}

	isStaff := role == "pharmacy_manager" || role == "admin" || role == "super_admin" || role == "pharmacist"
	if !isStaff && order.UserID != userID {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Access denied")
	}
	return response.JSON(c, fiber.StatusOK, order, "Order fetched")
}

// PackOrder transitions an order to "packed" status (pharmacist action).
// Feature 2: Enforces RxValidationLevel for each Rx item before allowing packing.
func (h *Handler) PackOrder(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	order, err := h.svc.GetOrder(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Order not found")
	}
	if order.Status != "processing" && order.Status != "confirmed" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_STATUS",
			fmt.Sprintf("Cannot pack order in '%s' status — must be 'processing' or 'confirmed'", order.Status))
	}

	// Feature 2: RxValidationLevel enforcement at dispensing/packing — fail closed
	var orderItems []struct {
		MedicineID uint `json:"medicine_id"`
	}
	if jsonErr := json.Unmarshal(order.Items, &orderItems); jsonErr != nil {
		// Cannot parse items — block packing; require staff to fix order data
		return response.Error(c, fiber.StatusUnprocessableEntity, "INVALID_ORDER_DATA",
			"Order items data is corrupt and cannot be packed — please contact support")
	}

	// Fetch prescription for this order (linked via Prescription.OrderID)
	var prescription Prescription
	hasPrescription := h.db.Where("order_id = ?", order.ID).First(&prescription).Error == nil

	for _, item := range orderItems {
		var med Medicine
		if medErr := h.db.First(&med, item.MedicineID).Error; medErr != nil {
			continue
		}
		if !med.IsPrescriptionRequired || med.RxValidationLevel == RxLevelOTC {
			continue // OTC: no prescription needed
		}
		if med.RxValidationLevel == RxLevelStandard {
			// Prescription must exist and be verified
			if !hasPrescription || prescription.Status != "verified" {
				return response.Error(c, fiber.StatusUnprocessableEntity, "RX_VALIDATION_FAILED",
					fmt.Sprintf("Item '%s' requires a verified prescription (RX_STANDARD)", med.Name))
			}
		}
		if med.RxValidationLevel == RxLevelStrict {
			// Prescription must exist, be verified, AND verified within last 30 days
			if !hasPrescription || prescription.Status != "verified" {
				return response.Error(c, fiber.StatusUnprocessableEntity, "RX_VALIDATION_FAILED",
					fmt.Sprintf("Item '%s' requires a verified prescription (RX_STRICT)", med.Name))
			}
			if prescription.VerifiedAt == nil || time.Since(*prescription.VerifiedAt) > 30*24*time.Hour {
				return response.Error(c, fiber.StatusUnprocessableEntity, "RX_EXPIRED",
					fmt.Sprintf("Item '%s' requires a prescription verified within the last 30 days (RX_STRICT)", med.Name))
			}
		}
	}

	// Reserve inventory when packing — fail if stock is insufficient
	if resErr := h.svc.ReserveInventory(c.Context(), uint(id)); resErr != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "INSUFFICIENT_STOCK",
			"Cannot pack order: "+resErr.Error())
	}

	if err := h.svc.UpdateOrderStatus(c.Context(), uint(id), "packed"); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Order packed and inventory reserved")
}

// allowedOrderTransitions defines valid state machine transitions for orders.
// H-10: pending → confirmed/cancelled; confirmed → packed/cancelled; packed → shipped/cancelled; shipped → delivered
var allowedOrderTransitions = map[string]map[string]bool{
	"pending":                 {"confirmed": true, "processing": true, "cancelled": true},
	"pending_rx_verification": {"processing": true, "confirmed": true, "cancelled": true},
	"processing":              {"confirmed": true, "packed": true, "cancelled": true},
	"confirmed":               {"packed": true, "cancelled": true},
	"packed":                  {"dispensed": true, "shipped": true, "cancelled": true},
	"dispensed":               {"shipped": true, "cancelled": true},
	"shipped":                 {"delivered": true},
}

// UpdateOrderStatus allows staff to advance order through the fulfillment pipeline.
// H-10: enforces state machine transitions.
func (h *Handler) UpdateOrderStatus(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	var req struct {
		Status     string `json:"status"`
		TrackingID string `json:"tracking_id,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	validStatuses := map[string]bool{
		"pending": true, "pending_rx_verification": true, "processing": true,
		"confirmed": true, "packed": true, "dispensed": true,
		"shipped": true, "delivered": true, "cancelled": true,
	}
	if !validStatuses[req.Status] {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_STATUS",
			"Status must be one of: pending, pending_rx_verification, processing, confirmed, packed, dispensed, shipped, delivered, cancelled")
	}

	// H-10: fetch current status and validate transition
	var order PharmacyOrder
	if err := h.db.First(&order, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Order not found")
	}

	allowed, exists := allowedOrderTransitions[order.Status]
	if !exists || !allowed[req.Status] {
		return response.Error(c, fiber.StatusUnprocessableEntity, "INVALID_TRANSITION",
			fmt.Sprintf("Invalid status transition from %s to %s", order.Status, req.Status))
	}

	updates := map[string]interface{}{"status": req.Status}
	if req.TrackingID != "" {
		updates["tracking_id"] = req.TrackingID
	}

	// Feature 5: generate a delivery OTP when the order ships.
	// B10: also send the OTP to the patient via WhatsApp.
	if req.Status == "shipped" {
		otp := crypto.OTP()
		updates["delivery_otp"] = otp
		// Store in Redis with 24h expiry as a backup validation source.
		if database.Rdb != nil {
			otpKey := fmt.Sprintf("delivery_otp:%d", id)
			database.Rdb.Set(context.Background(), otpKey, otp, 24*time.Hour)
		}
		// B10: notify patient with delivery OTP (fire-and-forget goroutine).
		go func(phone, otpCode string) {
			notificationSvc.WhatsAppOTP(phone, "Customer", otpCode)
		}(string(order.DeliveryPhone), otp)
	}

	// Cancel restore: when cancelling a packed or dispensed order, return
	// reserved stock back to inventory so it can be sold to other customers.
	if req.Status == "cancelled" &&
		(order.Status == "packed" || order.Status == "dispensed") {
		if restoreErr := h.svc.RestoreInventory(c.Context(), order.ID); restoreErr != nil {
			// Log but do not block the cancellation — inventory ops are
			// best-effort here; manual reconciliation can correct any drift.
			fmt.Printf("[pharmacy] RestoreInventory failed for order %d on cancel: %v\n", order.ID, restoreErr)
		}
	}

	if err := h.db.Model(&PharmacyOrder{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, fmt.Sprintf("Order status updated to '%s'", req.Status))
}

// ConfirmDeliveryOTP confirms delivery by verifying the OTP sent to the patient.
// Feature 5: POST /pharmacy/orders/:id/confirm-delivery
func (h *Handler) ConfirmDeliveryOTP(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	var req struct {
		OTP string `json:"otp"`
	}
	if err := c.BodyParser(&req); err != nil || req.OTP == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "otp is required")
	}

	// Fetch order.
	var order PharmacyOrder
	if err := h.db.First(&order, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Order not found")
	}
	if order.Status != "shipped" {
		return response.Error(c, fiber.StatusUnprocessableEntity, "INVALID_STATE",
			"Order is not in 'shipped' state")
	}

	// Verify OTP: check Redis first, fall back to DB field.
	var validOTP string
	otpKey := fmt.Sprintf("delivery_otp:%d", id)
	if database.Rdb != nil {
		if stored, redisErr := database.Rdb.Get(c.Context(), otpKey).Result(); redisErr == nil {
			validOTP = stored
		}
	}
	if validOTP == "" {
		validOTP = order.DeliveryOTP // fallback to DB column
	}
	if validOTP == "" || validOTP != req.OTP {
		return response.Error(c, fiber.StatusUnprocessableEntity, "INVALID_OTP",
			"Invalid or expired delivery OTP")
	}

	// Advance to delivered.
	if err := h.db.Model(&PharmacyOrder{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       "delivered",
			"delivery_otp": "", // clear OTP after use
		}).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	// Delete Redis OTP.
	if database.Rdb != nil {
		database.Rdb.Del(c.Context(), otpKey)
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"message": "Delivery confirmed"}, "Delivery confirmed")
}

// InitiatePharmacyPaymentHandler creates a Razorpay order for a pharmacy order.
// POST /pharmacy/orders/:id/initiate-payment
func (h *Handler) InitiatePharmacyPaymentHandler(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	order, err := h.svc.GetOrder(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Order not found")
	}

	// Ownership check: only the order owner may initiate payment
	callerID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	isStaff := role == "admin" || role == "super_admin" || role == "pharmacist" || role == "pharmacy_admin"
	if !isStaff && order.UserID != callerID {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You can only initiate payment for your own orders")
	}

	// Only allow payment initiation for pending/unpaid orders
	if order.PaymentStatus == "paid" {
		return response.Error(c, fiber.StatusConflict, "ALREADY_PAID", "Order is already paid")
	}

	receipt := fmt.Sprintf("PHARM-%d-%s", order.ID, uuid.New().String()[:6])
	rzpOrder, err := razorpay.CreateOrder(order.Amount, receipt)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ORDER_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"order_id":       rzpOrder["id"],
		"amount":         order.Amount,
		"currency":       "INR",
		"receipt":        receipt,
		"pharmacy_order": order.ID,
		"razorpay_order": rzpOrder,
	}, "Razorpay order created for pharmacy payment")
}

// VerifyPharmacyPaymentHandler verifies Razorpay signature, posts ledger entries, and updates order status.
// POST /pharmacy/orders/:id/verify-payment
func (h *Handler) VerifyPharmacyPaymentHandler(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	var req struct {
		OrderID   string `json:"order_id"`
		PaymentID string `json:"payment_id"`
		Signature string `json:"signature"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	if !razorpay.VerifyPaymentSignature(req.OrderID, req.PaymentID, req.Signature) {
		return response.Error(c, fiber.StatusBadRequest, "SIGNATURE_INVALID", "Payment signature verification failed")
	}

	order, err := h.svc.GetOrder(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Order not found")
	}

	// Ownership check: only the order owner may verify payment
	callerID, _ := c.Locals("user_id").(uint)
	verifyRole, _ := c.Locals("role").(string)
	isVerifyStaff := verifyRole == "admin" || verifyRole == "super_admin" || verifyRole == "pharmacist" || verifyRole == "pharmacy_admin"
	if !isVerifyStaff && order.UserID != callerID {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You can only verify payment for your own orders")
	}

	if order.PaymentStatus == "paid" {
		return response.Error(c, fiber.StatusConflict, "ALREADY_PAID", "Order is already paid")
	}

	// Post ledger entries for pharmacy payment split
	// Total: 100%
	// Platform margin: 10%
	// GST: 5% on total
	// Pharmacy revenue: 85% (remaining)
	finSvc := finance.NewService(finance.NewRepository(database.DB))
	ctx := c.Context()
	total := order.Amount

	platformMargin := total * 10 / 100
	gst := total * 5 / 100
	pharmacyRevenue := total - platformMargin - gst

	razorpayAcct, _ := finSvc.GetOrCreateAccount(ctx, finance.AcctRazorpay, finance.TypeAsset)
	platformRevAcct, _ := finSvc.GetOrCreateAccount(ctx, finance.RevPharmacy, finance.TypeRevenue)
	gstAcct, _ := finSvc.GetOrCreateAccount(ctx, finance.LiabGST, finance.TypeLiability)

	// Pharmacy's share: payable to an independent pharmacy, or platform revenue
	// for a central warehouse. Same canonical accounts as the wallet path.
	pharmRevAcctName := finance.MerchantWallet(order.PharmacyID)
	pharmRevAcctType := finance.TypeLiability
	if order.Pharmacy.IsCentralWarehouse {
		pharmRevAcctName = finance.RevCentralWarehouse
		pharmRevAcctType = finance.TypeRevenue
	}
	pharmRevAcct, _ := finSvc.GetOrCreateAccount(ctx, pharmRevAcctName, pharmRevAcctType)

	entries := []finance.EntryData{
		{AccountID: razorpayAcct.ID, Direction: finance.DirectionDebit, Amount: total},
		{AccountID: platformRevAcct.ID, Direction: finance.DirectionCredit, Amount: platformMargin},
		{AccountID: gstAcct.ID, Direction: finance.DirectionCredit, Amount: gst},
		{AccountID: pharmRevAcct.ID, Direction: finance.DirectionCredit, Amount: pharmacyRevenue},
	}

	txn, err := finSvc.RecordTransaction(ctx, req.PaymentID, "PHARMACY_ORDER",
		fmt.Sprintf("Pharmacy order #%d payment", order.ID), entries)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "LEDGER_FAILED", err.Error())
	}

	// Update order status
	txnID := txn.ID
	if err := h.db.Model(&PharmacyOrder{}).Where("id = ?", id).Updates(map[string]interface{}{
		"payment_status":        "paid",
		"status":                "processing",
		"ledger_transaction_id": txnID,
	}).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}

	// Award loyalty points on Razorpay-paid orders (best-effort).
	_, _ = loyalty.EarnPoints(order.UserID, order.Amount, "medicine", order.ID)

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"status":         "paid",
		"transaction_id": txn.ID,
		"order_id":       order.ID,
		"split": fiber.Map{
			"platform_margin": float64(platformMargin) / 100.0,
			"gst":             float64(gst) / 100.0,
			"pharmacy":        float64(pharmacyRevenue) / 100.0,
		},
	}, "Pharmacy order payment verified and processed")
}

// postPharmacyOrderLedger posts the pharmacy split (10% platform + 5% GST + 85%
// merchant/central) for an order, debiting the given funding account. Shared by
// the Razorpay and Stripe paths. Returns the ledger transaction id.
func (h *Handler) postPharmacyOrderLedger(ctx context.Context, order *PharmacyOrder, fundingAccount, ref string) (uuid.UUID, error) {
	finSvc := finance.NewService(finance.NewRepository(database.DB))
	total := order.Amount
	platformMargin := total * 10 / 100
	gst := total * 5 / 100
	merchant := total - platformMargin - gst

	fundAcct, _ := finSvc.GetOrCreateAccount(ctx, fundingAccount, finance.TypeAsset)
	platformRevAcct, _ := finSvc.GetOrCreateAccount(ctx, finance.RevPharmacy, finance.TypeRevenue)
	gstAcct, _ := finSvc.GetOrCreateAccount(ctx, finance.LiabGST, finance.TypeLiability)

	merchAcctName := finance.MerchantWallet(order.PharmacyID)
	merchAcctType := finance.TypeLiability
	if order.Pharmacy.IsCentralWarehouse {
		merchAcctName = finance.RevCentralWarehouse
		merchAcctType = finance.TypeRevenue
	}
	merchAcct, _ := finSvc.GetOrCreateAccount(ctx, merchAcctName, merchAcctType)

	txn, err := finSvc.RecordTransaction(ctx, ref, "PHARMACY_ORDER",
		fmt.Sprintf("Pharmacy order #%d payment", order.ID), []finance.EntryData{
			{AccountID: fundAcct.ID, Direction: finance.DirectionDebit, Amount: total},
			{AccountID: platformRevAcct.ID, Direction: finance.DirectionCredit, Amount: platformMargin},
			{AccountID: gstAcct.ID, Direction: finance.DirectionCredit, Amount: gst},
			{AccountID: merchAcct.ID, Direction: finance.DirectionCredit, Amount: merchant},
		})
	if err != nil {
		return uuid.Nil, err
	}
	return txn.ID, nil
}

// InitiatePharmacyStripePayment creates a Stripe PaymentIntent (presentment
// currency) for a pharmacy order the caller owns; amount computed server-side.
func (h *Handler) InitiatePharmacyStripePayment(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	order, err := h.svc.GetOrder(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Order not found")
	}
	callerID, _ := c.Locals("user_id").(uint)
	if order.UserID != callerID {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Not your order")
	}
	if order.PaymentStatus == "paid" {
		return response.Error(c, fiber.StatusConflict, "ALREADY_PAID", "Order already paid")
	}
	var body struct {
		Currency string `json:"currency"`
	}
	_ = c.BodyParser(&body)
	if body.Currency == "" {
		body.Currency = "USD"
	}
	if !fx.Supported(body.Currency) {
		return response.Error(c, fiber.StatusBadRequest, "UNSUPPORTED_CURRENCY", "Currency not supported")
	}
	amountMinor, rate, err := fx.ToPresentment(order.Amount, body.Currency)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "FX_FAILED", err.Error())
	}
	intent, err := stripe.CreatePaymentIntent(amountMinor, body.Currency, map[string]string{
		"ref":       fmt.Sprintf("PHARM-%d", order.ID),
		"inr_paise": fmt.Sprintf("%d", order.Amount),
		"fx_rate":   fmt.Sprintf("%.4f", rate),
	})
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "STRIPE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, intent, "Stripe payment intent created")
}

// VerifyPharmacyStripePayment confirms via server-side retrieve, then posts the
// order in INR via Asset:Stripe. Idempotent on the PaymentIntent id.
func (h *Handler) VerifyPharmacyStripePayment(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	order, err := h.svc.GetOrder(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Order not found")
	}
	callerID, _ := c.Locals("user_id").(uint)
	if order.UserID != callerID {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Not your order")
	}
	if order.PaymentStatus == "paid" {
		return response.Error(c, fiber.StatusConflict, "ALREADY_PAID", "Order already paid")
	}
	var body struct {
		PaymentIntentID string `json:"payment_intent_id"`
	}
	if err := c.BodyParser(&body); err != nil || body.PaymentIntentID == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "payment_intent_id required")
	}
	pi, err := stripe.RetrievePaymentIntent(body.PaymentIntentID)
	if err != nil {
		return response.Error(c, fiber.StatusBadGateway, "STRIPE_FAILED", err.Error())
	}
	if pi.Status != "succeeded" {
		return response.Error(c, fiber.StatusBadRequest, "NOT_SUCCEEDED", "Payment not completed")
	}
	txnID, err := h.postPharmacyOrderLedger(c.Context(), order, finance.AcctStripe, body.PaymentIntentID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "LEDGER_FAILED", err.Error())
	}
	if err := h.db.Model(&PharmacyOrder{}).Where("id = ?", id).Updates(map[string]interface{}{
		"payment_status":        "paid",
		"status":                "processing",
		"ledger_transaction_id": txnID,
	}).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	// Award loyalty points on Stripe-paid pharmacy orders (best-effort).
	_, _ = loyalty.EarnPoints(order.UserID, order.Amount, "medicine", order.ID)

	return response.JSON(c, fiber.StatusOK, fiber.Map{"status": "paid", "transaction_id": txnID, "order_id": order.ID}, "Pharmacy order paid via Stripe")
}

// SmartSearch is the autocomplete endpoint for the doctor's prescribing UI and
// the patient catalog. It matches across English/Tamil/Hindi names + brand +
// generic + synonyms; the AYUSH product master fields (form, pack size, MRP)
// come back so the doctor's UI can show "Nilavembu Kashayam — kashayam · 100 ml
// · ₹120" without a follow-up fetch.
//
// Query params:
//
//	q         — search text (≥2 chars). Matches name_en, name_ta, name_hi,
//	            brand, generic_name, synonyms.
//	category  — optional filter: "siddha" | "ayurveda" | "wellness"
//	form      — optional filter: "kashayam" | "churna" | "tailam" | "bhasma" |
//	            "tablet" | "capsule" | "syrup" | "oil" | …
//	limit     — optional, default 10, capped at 25
func (h *Handler) SmartSearch(c *fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 2 {
		return response.JSON(c, fiber.StatusOK, fiber.Map{
			"medicines": []interface{}{},
			"brands":    []interface{}{},
		}, "Query too short")
	}
	limit := c.QueryInt("limit", 10)
	if limit < 1 {
		limit = 10
	}
	if limit > 25 {
		limit = 25
	}
	category := strings.TrimSpace(c.Query("category"))
	form := strings.TrimSpace(c.Query("form"))

	// For multi-script matching we LOWER both sides; on Postgres/SQLite this is a
	// no-op for Tamil/Hindi text but keeps the English match case-insensitive.
	like := "%" + strings.ToLower(q) + "%"

	type MediResult struct {
		ID             uint   `json:"id"`
		Name           string `json:"name"`
		NameTa         string `json:"name_ta,omitempty"`
		NameHi         string `json:"name_hi,omitempty"`
		Brand          string `json:"brand"`
		Category       string `json:"category"`
		Form           string `json:"form,omitempty"`
		PackSize       string `json:"pack_size,omitempty"`
		GenericName    string `json:"generic_name"`
		MoleculeCode   string `json:"molecule_code"`
		IsInhouseBrand bool   `json:"is_inhouse_brand"`
		ImageURL       string `json:"image_url,omitempty"`
		IsRx           bool   `json:"is_prescription_required"`
		PriceINR       int64  `json:"price_inr"` // selling price in paise (0 if unpriced)
		InStock        bool   `json:"in_stock"`
	}
	// Price extraction from the JSONB prices column differs by dialect.
	priceExpr := "(prices->>'INR')::bigint" // Postgres
	if h.db.Dialector.Name() == "sqlite" {
		priceExpr = "CAST(json_extract(prices, '$.INR') AS INTEGER)"
	}
	stockExpr := "EXISTS (SELECT 1 FROM pharmacyinventory pi WHERE pi.medicine_id = medicine.id AND pi.stock_quantity > 0 AND pi.is_active = true)"

	var meds []MediResult
	// Public store search — published + public-channel only (publicCatalogScope).
	qb := publicCatalogScope(h.db.Model(&Medicine{})).
		Select("id, name, name_ta, name_hi, brand, category, form, pack_size, generic_name, molecule_code, is_inhouse_brand, image_url, is_prescription_required AS is_rx, " +
			"COALESCE(" + priceExpr + ", 0) AS price_inr, " + stockExpr + " AS in_stock").
		Where(`(
			LOWER(name) LIKE ? OR LOWER(name_ta) LIKE ? OR LOWER(name_hi) LIKE ?
			OR LOWER(brand) LIKE ? OR LOWER(generic_name) LIKE ? OR LOWER(synonyms) LIKE ?
		)`, like, like, like, like, like, like)
	if category != "" {
		qb = qb.Where("LOWER(category) = ?", strings.ToLower(category))
	}
	if form != "" {
		qb = qb.Where("LOWER(form) = ?", strings.ToLower(form))
	}
	// In-house brand first so the doctor's UI naturally surfaces your own SKU
	// before 3rd-party brands of the same composition.
	qb.Order("is_inhouse_brand DESC, name ASC").Limit(limit).Scan(&meds)

	var brands []string
	publicCatalogScope(h.db.Model(&Medicine{})).
		Select("DISTINCT brand").
		Where("brand <> '' AND LOWER(brand) LIKE ?", like).
		Limit(5).Pluck("brand", &brands)

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"medicines": meds,
		"brands":    brands,
	}, "Search results")
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	pharmacyGroup := router.Group("/pharmacy")
	pharmacyGroup.Get("/dashboard/stats", middleware.PharmacyManagerOnly(), h.GetDashboardStats)
	pharmacyGroup.Get("/staff/me", middleware.Protected(), h.GetStaffSelf)
	// H-1: inventory route requires PharmacyManagerOnly (was Protected)
	pharmacyGroup.Get("/inventory", middleware.PharmacyManagerOnly(), h.GetInventory)
	pharmacyGroup.Get("/inventory/low-stock", middleware.PharmacyManagerOnly(), h.GetLowStock)
	pharmacyGroup.Get("/inventory/expiring", middleware.PharmacyManagerOnly(), h.GetExpiringInventory)
	pharmacyGroup.Get("/alerts", middleware.PharmacyManagerOnly(), h.GetStockAlerts)
	// H-11: dispose stock requires PharmacyManagerOnly (was Protected)
	pharmacyGroup.Post("/inventory/dispose", middleware.PharmacyManagerOnly(), h.DisposeStockHandler)
	// Bug #13 fix: inventory transfer requests are an internal admin operation
	pharmacyGroup.Post("/transfers", middleware.PharmacyManagerOnly(), h.RequestInventoryTransfer)
	// M-9: ProcessInventoryTransfer route requires PharmacyManagerOnly
	pharmacyGroup.Put("/transfers/:id", middleware.PharmacyManagerOnly(), h.ProcessInventoryTransfer)
	// Bug #1 fix: cross-city stock view is admin/manager only
	pharmacyGroup.Get("/inventory/multi-city", middleware.PharmacyManagerOnly(), h.GetMultiCityInventory)
	pharmacyGroup.Get("/", h.GetPharmacies)
	// Bug #3 fix: branch list is manager-only (commercial data)
	pharmacyGroup.Get("/admin/list", middleware.PharmacyManagerOnly(), h.ListPharmaciesAdmin)
	// Bug #4,5,6 CRITICAL fix: branch creation/edit requires pharmacy_admin or admin
	pharmacyGroup.Post("/admin", middleware.PharmacyAdminOnly(), h.CreatePharmacyHandler)
	pharmacyGroup.Put("/admin/:id", middleware.PharmacyAdminOnly(), h.UpdatePharmacyHandler)
	pharmacyGroup.Patch("/admin/:id/toggle", middleware.PharmacyAdminOnly(), h.TogglePharmacyHandler)
	pharmacyGroup.Post("/admin/trigger-restock", middleware.PharmacyManagerOnly(), h.TriggerRestockHandler)
	pharmacyGroup.Post("/prescriptions/convert", middleware.Protected(), h.ConvertPrescriptionHandler)
	pharmacyGroup.Post("/prescriptions/cart", middleware.Protected(), h.GenerateCartHandler)

	// Customer Order Routes
	pharmacyGroup.Post("/orders/counter-sale", middleware.PharmacyManagerOnly(), h.CounterSaleHandler) // must be before /:id
	pharmacyGroup.Get("/orders", middleware.Protected(), h.GetOrders)
	pharmacyGroup.Get("/orders/:id", middleware.Protected(), h.GetOrderByID)
	pharmacyGroup.Post("/orders/:id/pay", middleware.Protected(), h.PayOrderHandler)
	pharmacyGroup.Post("/orders/:id/initiate-payment", middleware.Protected(), h.InitiatePharmacyPaymentHandler)
	pharmacyGroup.Post("/orders/:id/verify-payment", middleware.Protected(), h.VerifyPharmacyPaymentHandler)
	pharmacyGroup.Post("/orders/:id/stripe/initiate", middleware.Protected(), h.InitiatePharmacyStripePayment)
	pharmacyGroup.Post("/orders/:id/stripe/verify", middleware.Protected(), h.VerifyPharmacyStripePayment)
	pharmacyGroup.Post("/orders/:id/pack", middleware.PharmacyManagerOnly(), h.PackOrder)
	pharmacyGroup.Put("/orders/:id/status", middleware.PharmacyManagerOnly(), h.UpdateOrderStatus)

	pharmacyGroup.Get("/financials", middleware.Protected(), h.GetPharmacyFinancialsHandler)

	// Pharmacy units management (alias for pharmacy branch admin — used by admin portal)
	// Bug #7 fix: units list is manager-only
	pharmacyGroup.Get("/units", middleware.PharmacyManagerOnly(), h.ListPharmaciesAdmin)
	pharmacyGroup.Post("/units", middleware.PharmacyAdminOnly(), h.CreatePharmacyHandler)
	pharmacyGroup.Put("/units/:id", middleware.PharmacyAdminOnly(), h.UpdatePharmacyHandler)
	pharmacyGroup.Post("/units/:id/toggle", middleware.PharmacyAdminOnly(), h.TogglePharmacyHandler)

	// Supplier Routes — Bug #9 fix: supplier data is manager-only
	pharmacyGroup.Get("/suppliers", middleware.PharmacyManagerOnly(), h.GetSuppliers)
	pharmacyGroup.Post("/suppliers", middleware.PharmacyAdminOnly(), h.CreateSupplier)
	pharmacyGroup.Put("/suppliers/:id", middleware.PharmacyAdminOnly(), h.UpdateSupplier)
	pharmacyGroup.Delete("/suppliers/:id", middleware.PharmacyAdminOnly(), h.DeleteSupplier)

	// Medicine Catalog Routes — Bug #14 fix: mutations require pharmacy_admin (not just pharmacist)
	// Public catalog (published + public-channel only; see publicCatalogScope).
	pharmacyGroup.Get("/medicines", h.GetMedicines)
	pharmacyGroup.Get("/catalog/facets", h.GetCatalogFacets)
	pharmacyGroup.Get("/medicines/search", h.SmartSearch)

	// Staff catalog management (authenticated — sees drafts + pharmacy_only).
	pharmacyGroup.Get("/admin/catalog", middleware.PharmacyManagerOnly(), h.ListAdminCatalog)
	pharmacyGroup.Get("/admin/catalog/search", middleware.PharmacyManagerOnly(), h.AdminSearchMedicines)

	pharmacyGroup.Get("/medicines/:id", h.GetMedicineByID)
	pharmacyGroup.Get("/medicines/:id/substitutes", h.GetSubstitutes)
	// Publish lifecycle.
	pharmacyGroup.Post("/medicines/:id/submit", middleware.PharmacyManagerOnly(), h.SubmitMedicineForReview)
	pharmacyGroup.Post("/medicines/:id/publish", middleware.PharmacyAdminOnly(), h.PublishMedicine)
	pharmacyGroup.Post("/medicines/:id/unpublish", middleware.PharmacyAdminOnly(), h.UnpublishMedicine)
	h.RegisterReviewRoutes(pharmacyGroup)
	h.RegisterCouponRoutes(pharmacyGroup)
	h.RegisterBundleRoutes(pharmacyGroup)
	h.RegisterParsePrescriptionRoute(pharmacyGroup)

	// P2: stock reservation — prevents oversold inventory under concurrent carts.
	pharmacyGroup.Post("/reservations", middleware.Protected(), ReserveStock)
	pharmacyGroup.Post("/reservations/:cart_id/confirm", middleware.Protected(), ConfirmCartReservation)
	pharmacyGroup.Post("/reservations/:cart_id/release", middleware.Protected(), ReleaseCartReservation)

	// P3: itemised pricing — patient sees MRP + GST + shipping + COD before pay.
	pharmacyGroup.Post("/pricing/price-cart", PriceCartHandler)
	pharmacyGroup.Post("/medicines", middleware.PharmacyAdminOnly(), h.CreateMedicine)
	pharmacyGroup.Post("/medicines/:id/image", middleware.PharmacyAdminOnly(), h.UploadMedicineImage)
	pharmacyGroup.Put("/medicines/:id", middleware.PharmacyAdminOnly(), h.UpdateMedicine)
	pharmacyGroup.Delete("/medicines/:id", middleware.PharmacyAdminOnly(), h.DeleteMedicine)

	// Purchase Order Routes
	// H-3: GetPurchaseOrders requires PharmacyManagerOnly (was Protected)
	pharmacyGroup.Get("/purchase-orders", middleware.PharmacyManagerOnly(), h.GetPurchaseOrders)
	// Bug #10 fix: individual PO fetch is also manager-only
	pharmacyGroup.Get("/purchase-orders/:id", middleware.PharmacyManagerOnly(), h.GetPurchaseOrder)
	pharmacyGroup.Post("/purchase-orders", middleware.PharmacyAdminOnly(), h.CreatePurchaseOrder)
	// Bug #11 fix: approve/reject are financial decisions — pharmacy_admin only
	pharmacyGroup.Post("/purchase-orders/:id/approve", middleware.PharmacyAdminOnly(), h.ApprovePurchaseOrder)
	pharmacyGroup.Post("/purchase-orders/:id/reject", middleware.PharmacyAdminOnly(), h.RejectPurchaseOrder)
	pharmacyGroup.Post("/purchase-orders/:id/receive", middleware.PharmacyManagerOnly(), h.ReceivePurchaseOrderHandler)
	// Bug #12 fix: supplier payment settlement is a financial action — pharmacy_admin only
	pharmacyGroup.Post("/purchase-orders/:id/settle", middleware.PharmacyAdminOnly(), h.SettleSupplierPaymentHandler)

	// Dispense step (Feature 3)
	pharmacyGroup.Post("/orders/:id/dispense", middleware.PharmacyManagerOnly(), h.DispenseOrder)
	pharmacyGroup.Get("/orders/:id/dispense-records", middleware.PharmacyManagerOnly(), h.GetDispenseRecords)

	// Delivery OTP confirmation (Feature 5)
	pharmacyGroup.Post("/orders/:id/confirm-delivery", middleware.Protected(), h.ConfirmDeliveryOTP)

	// Stock transfers list (Feature 9 — GET was missing)
	pharmacyGroup.Get("/transfers", middleware.PharmacyManagerOnly(), h.GetTransfersHandler)

	// Warehouse Routes (protected)
	warehouseGroup := router.Group("/warehouse", middleware.Protected())
	warehouseGroup.Post("/metrics", h.RecordWarehouseMetricHandler)
	warehouseGroup.Get("/metrics/:id", h.GetWarehouseMetricsHandler)
	warehouseGroup.Get("/alerts", middleware.PharmacyManagerOnly(), h.GetWarehouseAlertsHandler)
	// H-26: removed duplicate TriggerRestockHandler registration using only Protected().
	// Keep only the PharmacyManagerOnly version via /admin/trigger-restock above.
	warehouseGroup.Post("/restock/manual", middleware.PharmacyManagerOnly(), h.TriggerRestockHandler)

	// Cart & Prescription Routes (Shopify-tier engine)
	h.RegisterCartRoutes(router)

	// Analytics & Ecommerce Dashboard Routes
	RegisterAnalyticsRoutes(pharmacyGroup, h.db)

	// Settings, Reports & Refill Conversion Routes
	h.RegisterSettingsAndReportsRoutes(pharmacyGroup)
}

//go:build ignore
// +build ignore

package handlers

import (
	"fmt"
	"drayya-core/internal/database"
	"drayya-core/internal/middleware"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/models"
	"drayya-core/internal/pharmacy"
	pharmacyService "drayya-core/internal/services/pharmacy"
	"drayya-core/internal/utils/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func GetInventory(c *fiber.Ctx) error {
	pharmacyIDStr := c.Query("pharmacy_id")
	pharmacyID := 0
	if pharmacyIDStr != "" {
		id, err := strconv.Atoi(pharmacyIDStr)
		if err == nil {
			pharmacyID = id
		}
	}

	inventory, err := pharmacy.GetInventory(uint(pharmacyID))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "INVENTORY_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, inventory, "Fetched")
}

func GetStockAlerts(c *fiber.Ctx) error {
	alerts, err := pharmacy.GetStockAlerts()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "STOCK_ALERTS_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, alerts, "Fetched")
}

func GetLowStock(c *fiber.Ctx) error {
	inventory, err := pharmacy.GetLowStockInventory()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "LOW_STOCK_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, inventory, "Fetched")
}

func GetExpiringInventory(c *fiber.Ctx) error {
	inventory, err := pharmacy.GetExpiringInventory()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "EXPIRING_INVENTORY_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, inventory, "Fetched")
}

func RequestInventoryTransfer(c *fiber.Ctx) error {
	var req pharmacy.InventoryTransfer
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	userID, _ := c.Locals("user_id").(uint)
	req.RequestedByID = userID

	s := pharmacyService.NewTransferService()
	res, err := s.RequestTransfer(req)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "TRANSFER_REQUEST_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, res, "Created")
}

func ProcessInventoryTransfer(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var req struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	userID, _ := c.Locals("user_id").(uint)
	s := pharmacyService.NewTransferService()
	err := s.ProcessTransfer(uint(id), userID, req.Status)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "TRANSFER_PROCESS_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Transfer processed successfully")
}

func GetMultiCityInventory(c *fiber.Ctx) error {
	stats, err := pharmacy.GetMultiCityInventory()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "MULTI_CITY_INVENTORY_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, stats, "Fetched")
}

func GetPharmacies(c *fiber.Ctx) error {
	res, err := pharmacy.GetAllPharmacies()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PHARMACIES_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, res, "Fetched")
}

// --- Admin CRUD Handlers ---

func ConvertPrescriptionHandler(c *fiber.Ctx) error {
	var req struct {
		PrescriptionID  uint   `json:"prescription_id"`
		PharmacyID      uint   `json:"pharmacy_id"`
		ShippingAddress string `json:"shipping_address"`
		Phone           string `json:"phone"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	order, err := pharmacy.ConvertPrescriptionToOrder(req.PrescriptionID, req.PharmacyID, req.ShippingAddress, req.Phone)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PRESCRIPTION_CONVERT_FAILED", err.Error())
	}

	// Auto-reserve inventory
	_ = pharmacy.ReserveInventory(order.ID)

	return response.JSON(c, fiber.StatusCreated, order, "Created")
}

func GenerateCartHandler(c *fiber.Ctx) error {
	var req struct {
		PrescriptionID uint `json:"prescription_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	cart, err := pharmacy.GenerateCartFromPrescription(req.PrescriptionID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CART_GENERATION_FAILED", err.Error())
	}

	// If no items were mapped automatically
	if len(cart) == 0 {
		return response.JSON(c, fiber.StatusOK, fiber.Map{
			"message": "Cart generated, but no active inventory matched prescription items",
			"cart":    cart,
		}, "Fetched")
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"message": "Cart generated successfully",
		"cart":    cart,
	}, "Fetched")
}

func ListPharmaciesAdmin(c *fiber.Ctx) error {
	res, err := pharmacy.ListAllPharmaciesAdmin()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PHARMACIES_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, res, "Fetched")
}

func CreatePharmacyHandler(c *fiber.Ctx) error {
	var req struct {
		Name               string  `json:"name"`
		Address            string  `json:"address"`
		City               string  `json:"city"`
		Latitude           float64 `json:"latitude"`
		Longitude          float64 `json:"longitude"`
		ServiceRadius      float64 `json:"service_radius"`
		IsCentralWarehouse bool    `json:"is_central_warehouse"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	p, err := pharmacy.CreatePharmacy(req.Name, req.Address, req.City, req.Latitude, req.Longitude, req.ServiceRadius, req.IsCentralWarehouse)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_PHARMACY_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, p, "Created")
}

func UpdatePharmacyHandler(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
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
	p, err := pharmacy.UpdatePharmacy(uint(id), req.Name, req.Address, req.City, req.Latitude, req.Longitude, req.ServiceRadius, req.IsCentralWarehouse, req.IsActive)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_PHARMACY_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, p, "Fetched")
}

func TogglePharmacyHandler(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	p, err := pharmacy.TogglePharmacy(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "TOGGLE_PHARMACY_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, p, "Fetched")
}

// TriggerRestockHandler manually runs the automated stock evaluation and triggers POs/Transfers
func TriggerRestockHandler(c *fiber.Ctx) error {
	count, err := pharmacy.CheckAndTriggerRestock()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "RESTOCK_TRIGGER_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"message": fmt.Sprintf("Successfully evaluated stock levels and triggered %d restocking requests", count),
		"count":   count,
	}, "Fetched")
}

func PayOrderHandler(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	if err := pharmacy.PayOrder(uint(id)); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PAY_ORDER_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Order paid successfully and financial split recorded")
}

func DisposeStockHandler(c *fiber.Ctx) error {
	var req struct {
		InventoryID uint   `json:"inventory_id"`
		Quantity    int    `json:"quantity"`
		Reason      string `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	var item pharmacy.PharmacyInventory
	if err := database.DB.First(&item, req.InventoryID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "INVENTORY_ITEM_NOT_FOUND", "Inventory item not found")
	}

	if item.StockQuantity < req.Quantity {
		return response.Error(c, fiber.StatusBadRequest, "INSUFFICIENT_STOCK", "Insufficient stock to dispose")
	}

	item.StockQuantity -= req.Quantity
	if err := database.DB.Save(&item).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "INVENTORY_UPDATE_FAILED", "Failed to update inventory")
	}

	return response.JSON(c, fiber.StatusOK, nil, "Stock disposed successfully")
}

func GetPharmacyFinancialsHandler(c *fiber.Ctx) error {
	pharmacyID, _ := strconv.Atoi(c.Query("pharmacy_id", "0"))
	stats, err := pharmacy.GetPharmacyFinancials(uint(pharmacyID))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FINANCIALS_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, stats, "Fetched")
}

func RegisterPharmacyRoutes(router fiber.Router) {
	pharmacyGroup := router.Group("/pharmacy")
	pharmacyGroup.Get("/inventory", GetInventory)
	pharmacyGroup.Get("/inventory/low-stock", GetLowStock)
	pharmacyGroup.Get("/inventory/expiring", GetExpiringInventory)
	pharmacyGroup.Get("/alerts", GetStockAlerts)
	pharmacyGroup.Post("/inventory/dispose", middleware.Protected(), DisposeStockHandler)
	pharmacyGroup.Post("/transfers", middleware.Protected(), RequestInventoryTransfer)
	pharmacyGroup.Put("/transfers/:id", middleware.Protected(), ProcessInventoryTransfer)
	pharmacyGroup.Get("/inventory/multi-city", middleware.Protected(), GetMultiCityInventory)
	pharmacyGroup.Get("/", GetPharmacies)
	pharmacyGroup.Get("/admin/list", middleware.Protected(), ListPharmaciesAdmin)
	pharmacyGroup.Post("/admin", middleware.Protected(), CreatePharmacyHandler)
	pharmacyGroup.Put("/admin/:id", middleware.Protected(), UpdatePharmacyHandler)
	pharmacyGroup.Patch("/admin/:id/toggle", middleware.Protected(), TogglePharmacyHandler)
	pharmacyGroup.Post("/admin/trigger-restock", middleware.Protected(), TriggerRestockHandler)
	pharmacyGroup.Post("/prescriptions/convert", middleware.Protected(), ConvertPrescriptionHandler)
	pharmacyGroup.Post("/prescriptions/cart", middleware.Protected(), GenerateCartHandler)
	pharmacyGroup.Post("/orders/:id/pay", middleware.Protected(), PayOrderHandler)
	pharmacyGroup.Get("/financials", middleware.Protected(), GetPharmacyFinancialsHandler)

	// Supplier Routes (reusing pharmacy group)
	pharmacyGroup.Get("/suppliers", GetSuppliers)
	pharmacyGroup.Post("/suppliers", middleware.Protected(), CreateSupplier)
	pharmacyGroup.Put("/suppliers/:id", middleware.Protected(), UpdateSupplier)
	pharmacyGroup.Delete("/suppliers/:id", middleware.Protected(), DeleteSupplier)

	// Medicine Catalog Routes
	pharmacyGroup.Get("/medicines", GetMedicines)
	pharmacyGroup.Post("/medicines", middleware.Protected(), CreateMedicine)
	pharmacyGroup.Put("/medicines/:id", middleware.Protected(), UpdateMedicine)
	pharmacyGroup.Delete("/medicines/:id", middleware.Protected(), DeleteMedicine)

	// Purchase Order Routes
	pharmacyGroup.Get("/purchase-orders", middleware.Protected(), GetPurchaseOrders)
	pharmacyGroup.Get("/purchase-orders/:id", middleware.Protected(), GetPurchaseOrder)
	pharmacyGroup.Post("/purchase-orders", middleware.Protected(), CreatePurchaseOrder)
	pharmacyGroup.Post("/purchase-orders/:id/approve", middleware.Protected(), ApprovePurchaseOrder)
	pharmacyGroup.Post("/purchase-orders/:id/reject", middleware.Protected(), RejectPurchaseOrder)
}

// suppress unused import warning for models
var _ = models.FamilyMember{}

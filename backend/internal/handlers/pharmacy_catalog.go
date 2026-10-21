//go:build ignore
// +build ignore

package handlers

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/utils/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// ==========================================
// Supplier CRUD
// ==========================================

func GetSuppliers(c *fiber.Ctx) error {
	var suppliers []pharmacy.Supplier
	if err := database.DB.Where("is_active = ?", true).Find(&suppliers).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SUPPLIERS_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, suppliers, "Fetched")
}

func CreateSupplier(c *fiber.Ctx) error {
	var supplier pharmacy.Supplier
	if err := c.BodyParser(&supplier); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	if err := database.DB.Create(&supplier).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_SUPPLIER_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, supplier, "Created")
}

func UpdateSupplier(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	if err := database.DB.Model(&pharmacy.Supplier{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_SUPPLIER_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Supplier updated")
}

func DeleteSupplier(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	if err := database.DB.Model(&pharmacy.Supplier{}).Where("id = ?", id).Update("is_active", false).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DELETE_SUPPLIER_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Supplier deactivated")
}

// ==========================================
// Medicine CRUD
// ==========================================

func GetMedicines(c *fiber.Ctx) error {
	search := c.Query("search")
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)
	if limit > 100 {
		limit = 100
	}
	var medicines []pharmacy.Medicine
	q := database.DB.Where("is_active = ?", true)
	if search != "" {
		q = q.Where("name LIKE ? OR brand LIKE ? OR category LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if err := q.Order("name ASC").Limit(limit).Offset(offset).Find(&medicines).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "MEDICINES_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, medicines, "Fetched")
}

func CreateMedicine(c *fiber.Ctx) error {
	var medicine pharmacy.Medicine
	if err := c.BodyParser(&medicine); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	if err := database.DB.Create(&medicine).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_MEDICINE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, medicine, "Created")
}

func UpdateMedicine(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	if err := database.DB.Model(&pharmacy.Medicine{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_MEDICINE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Medicine updated")
}

func DeleteMedicine(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	if err := database.DB.Delete(&pharmacy.Medicine{}, id).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DELETE_MEDICINE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Medicine deleted")
}

// ==========================================
// Purchase Order CRUD
// ==========================================

func GetPurchaseOrders(c *fiber.Ctx) error {
	var pos []pharmacy.PurchaseOrder
	if err := database.DB.Order("created_at desc").Find(&pos).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PURCHASE_ORDERS_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, pos, "Fetched")
}

func GetPurchaseOrder(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var po pharmacy.PurchaseOrder
	if err := database.DB.First(&po, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "PURCHASE_ORDER_NOT_FOUND", "Purchase order not found")
	}
	return response.JSON(c, fiber.StatusOK, po, "Fetched")
}

func CreatePurchaseOrder(c *fiber.Ctx) error {
	var po pharmacy.PurchaseOrder
	if err := c.BodyParser(&po); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	po.Status = "draft"
	if err := database.DB.Create(&po).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_PURCHASE_ORDER_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, po, "Created")
}

func ApprovePurchaseOrder(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	if err := database.DB.Model(&pharmacy.PurchaseOrder{}).Where("id = ?", id).Update("status", "ordered").Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "APPROVE_PURCHASE_ORDER_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Purchase order approved and sent to supplier")
}

func RejectPurchaseOrder(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	if err := database.DB.Model(&pharmacy.PurchaseOrder{}).Where("id = ?", id).Update("status", "cancelled").Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "REJECT_PURCHASE_ORDER_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Purchase order cancelled")
}

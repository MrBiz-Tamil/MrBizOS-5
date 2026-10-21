//go:build ignore
// +build ignore

package handlers

import (
	"drayya-core/internal/pharmacy"
	"drayya-core/internal/utils/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func RecordWarehouseMetric(c *fiber.Ctx) error {
	var req struct {
		WarehouseID uint    `json:"warehouse_id"`
		Temperature float64 `json:"temperature"`
		Humidity    float64 `json:"humidity"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	err := pharmacy.RecordWarehouseMetric(req.WarehouseID, req.Temperature, req.Humidity)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "WAREHOUSE_METRIC_FAILED", err.Error())
	}

	return c.SendStatus(200)
}

func GetWarehouseMetrics(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	metrics, err := pharmacy.GetWarehouseMetrics(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "WAREHOUSE_METRICS_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, metrics, "Fetched")
}

func TriggerManualRestock(c *fiber.Ctx) error {
	count, err := pharmacy.CheckAndTriggerRestock()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "RESTOCK_TRIGGER_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"triggered_orders": count}, "Fetched")
}

func RegisterWarehouseRoutes(router fiber.Router) {
	warehouseGroup := router.Group("/warehouse")

	warehouseGroup.Post("/metrics", RecordWarehouseMetric)
	warehouseGroup.Get("/metrics/:id", GetWarehouseMetrics)
	warehouseGroup.Post("/restock/manual", TriggerManualRestock)
}

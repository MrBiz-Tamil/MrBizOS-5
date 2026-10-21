package wellness

import (
	"drayya-core/internal/domain/store"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	storeSvc store.Service
}

func NewHandler(storeSvc store.Service) *Handler {
	return &Handler{storeSvc: storeSvc}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	wellGroup := router.Group("/wellness")
	wellGroup.Get("/products", h.GetProducts)
	wellGroup.Get("/products/:id", h.GetProduct)

	protected := wellGroup.Group("/", middleware.Protected())
	protected.Post("/orders", h.CreateOrder)
	protected.Get("/orders/my", h.GetMyOrders)
	protected.Get("/inventory", h.GetInventory)
	protected.Post("/inventory/add", h.AddToInventory)
	protected.Get("/goals", h.GetGoals)
	protected.Get("/programs/templates", h.GetTemplates)
	protected.Get("/stats", h.GetStats)
	protected.Get("/programs/mine", h.GetMyPrograms)
	protected.Get("/programs/:id", h.GetProgramDetails)
	protected.Post("/programs/:id/enroll", h.EnrollProgram)
}

func (h *Handler) GetProducts(c *fiber.Ctx) error {
	search := c.Query("search")
	category := c.Query("category")
	brand := c.Query("brand")
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)
	if limit > 100 {
		limit = 100
	}
	products, err := h.storeSvc.SearchProducts(c.Context(), search, category, brand, limit, offset)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, products, "Products fetched")
}

func (h *Handler) GetProduct(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	product, err := h.storeSvc.GetProductByID(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Product not found")
	}
	return response.JSON(c, fiber.StatusOK, product, "Product details fetched")
}

func (h *Handler) CreateOrder(c *fiber.Ctx) error {
	var body struct {
		Items           []store.OrderItemRequest `json:"items"`
		DeliveryAddress string                   `json:"delivery_address"`
		PharmacyID      uint                     `json:"pharmacy_id"`
		ContactPhone    string                   `json:"contact_phone"`
		ContactEmail    string                   `json:"contact_email"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	userID := c.Locals("user_id").(uint)

	// Default to Hub 1 if not specified
	pharmacyID := body.PharmacyID
	if pharmacyID == 0 {
		pharmacyID = 1
	}

	order, err := h.storeSvc.CreateOrder(c.Context(), userID, body.Items, body.DeliveryAddress, pharmacyID, body.ContactPhone, body.ContactEmail)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, order, "Order created successfully")
}

func (h *Handler) GetMyOrders(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	orders, err := h.storeSvc.GetOrders(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, orders, "User orders fetched")
}

func (h *Handler) GetInventory(c *fiber.Ctx) error {
	products, err := h.storeSvc.SearchProducts(c.Context(), "", "", "", 100, 0)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, products, "Wellness inventory fetched")
}

func (h *Handler) AddToInventory(c *fiber.Ctx) error {
	var body struct {
		ProductID uint   `json:"product_id"`
		Quantity  int    `json:"quantity"`
		Batch     string `json:"batch"`
		Expiry    string `json:"expiry"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"product_id": body.ProductID,
		"quantity":   body.Quantity,
		"batch":      body.Batch,
		"expiry":     body.Expiry,
	}, "Inventory update accepted")
}

func (h *Handler) GetGoals(c *fiber.Ctx) error {
	// Mock goals
	goals := []fiber.Map{
		{"id": 1, "title": "Weight Loss", "target": "5kg", "progress": 20, "deadline": "2026-06-01"},
		{"id": 2, "title": "Daily Steps", "target": "10000", "progress": 65, "deadline": "2026-05-30"},
	}
	return response.JSON(c, fiber.StatusOK, goals, "Wellness goals fetched")
}

func (h *Handler) GetTemplates(c *fiber.Ctx) error {
	// Mock templates
	templates := []fiber.Map{
		{"id": 1, "name": "Yoga for Beginners", "duration": "4 weeks", "duration_days": 28, "intensity": "Low", "description": "A simple mobility and breathing plan for new starters."},
		{"id": 2, "name": "Intermittent Fasting", "duration": "8 weeks", "duration_days": 56, "intensity": "Medium", "description": "A guided eating-window program with habit tracking."},
	}
	return response.JSON(c, fiber.StatusOK, templates, "Program templates fetched")
}

func (h *Handler) GetStats(c *fiber.Ctx) error {
	// Mock stats
	stats := fiber.Map{
		"calories_burned": 1250,
		"sleep_hours":     7.5,
		"water_intake":    "2.5L",
		"active_minutes":  45,
	}
	return response.JSON(c, fiber.StatusOK, stats, "Wellness stats fetched")
}

func (h *Handler) GetMyPrograms(c *fiber.Ctx) error {
	// Mock active programs
	programs := []fiber.Map{
		{"id": 101, "template_id": 1, "name": "Yoga for Beginners", "status": "active", "start_date": "2026-05-01", "progress": 35},
	}
	return response.JSON(c, fiber.StatusOK, programs, "User programs fetched")
}

func (h *Handler) GetProgramDetails(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid program ID")
	}
	program := fiber.Map{
		"id":            id,
		"template_id":   id,
		"name":          "Yoga for Beginners",
		"description":   "A guided wellness program with daily habits, simple tasks, and progress tracking.",
		"status":        "active",
		"progress":      0,
		"duration_days": 28,
		"modules": []fiber.Map{
			{"day": 1, "title": "Baseline check-in", "completed": false},
			{"day": 2, "title": "Breathing and mobility", "completed": false},
		},
	}
	return response.JSON(c, fiber.StatusOK, program, "Program details fetched")
}

func (h *Handler) EnrollProgram(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid program ID")
	}
	program := fiber.Map{
		"id":          id,
		"template_id": id,
		"status":      "active",
		"progress":    0,
	}
	return response.JSON(c, fiber.StatusCreated, program, "Program enrolled")
}

// Legacy global functions for backwards compatibility if needed during migration
func RegisterWellnessRoutes(router fiber.Router, storeSvc store.Service) {
	h := NewHandler(storeSvc)
	h.RegisterRoutes(router)
}

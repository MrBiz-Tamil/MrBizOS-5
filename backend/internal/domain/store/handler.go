package store

import (
	"fmt"
	"strconv"

	"drayya-core/internal/middleware"
	"drayya-core/internal/services/fx"
	"drayya-core/internal/services/razorpay"
	"drayya-core/internal/services/stripe"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	storeGroup := router.Group("/store")
	storeGroup.Get("/products", h.SearchProducts)
	storeGroup.Get("/products/:id", h.GetProduct)
	storeGroup.Get("/categories", h.ListCategories)
	storeGroup.Get("/delivery-check", h.CheckDelivery)

	protected := storeGroup.Group("/", middleware.Protected())
	protected.Post("/orders", h.CreateOrder)
	protected.Get("/orders", h.GetOrders)
	protected.Get("/orders/:id", h.GetOrderByID)
	protected.Post("/orders/:id/pay/initiate", h.InitiateStorePayment)
	protected.Post("/orders/:id/pay/verify", h.VerifyStorePayment)
	protected.Post("/orders/:id/pay/wallet", h.PayStoreFromWallet)
	protected.Post("/orders/:id/pay/stripe/initiate", h.InitiateStoreStripePayment)
	protected.Post("/orders/:id/pay/stripe/verify", h.VerifyStoreStripePayment)
}

// InitiateStoreStripePayment creates a Stripe PaymentIntent (presentment
// currency) for an order the caller owns; amount computed server-side in INR.
func (h *Handler) InitiateStoreStripePayment(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Auth required")
	}
	order, err := h.ownsOrder(c, uint(id), userID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Order not found")
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
	amountMinor, rate, err := fx.ToPresentment(order.TotalAmount, body.Currency)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "FX_FAILED", err.Error())
	}
	intent, err := stripe.CreatePaymentIntent(amountMinor, body.Currency, map[string]string{
		"ref":       fmt.Sprintf("STORE-%d", order.ID),
		"inr_paise": fmt.Sprintf("%d", order.TotalAmount),
		"fx_rate":   fmt.Sprintf("%.4f", rate),
	})
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "STRIPE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, intent, "Stripe payment intent created")
}

// VerifyStoreStripePayment confirms via server-side retrieve, then posts in INR.
func (h *Handler) VerifyStoreStripePayment(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Auth required")
	}
	if _, err := h.ownsOrder(c, uint(id), userID); err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Order not found")
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
	order, err := h.svc.ConfirmStorePayment(c.Context(), uint(id), "stripe", pi.ID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CONFIRM_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, order, "Payment verified")
}

// ownsOrder returns the order if it belongs to the caller, else an error response.
func (h *Handler) ownsOrder(c *fiber.Ctx, id uint, userID uint) (*WellnessOrder, error) {
	orders, err := h.svc.GetOrders(c.Context(), userID)
	if err != nil {
		return nil, err
	}
	for i := range orders {
		if orders[i].ID == id {
			return &orders[i], nil
		}
	}
	return nil, fmt.Errorf("order not found")
}

// InitiateStorePayment creates a Razorpay order for an order the caller owns.
func (h *Handler) InitiateStorePayment(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Auth required")
	}
	order, err := h.ownsOrder(c, uint(id), userID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Order not found")
	}
	if order.PaymentStatus == "paid" {
		return response.Error(c, fiber.StatusConflict, "ALREADY_PAID", "Order already paid")
	}
	rzpOrder, err := razorpay.CreateOrder(order.TotalAmount, fmt.Sprintf("STORE-%d", order.ID))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ORDER_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, rzpOrder, "Payment order created")
}

// VerifyStorePayment verifies the Razorpay signature, then posts the order.
func (h *Handler) VerifyStorePayment(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Auth required")
	}
	if _, err := h.ownsOrder(c, uint(id), userID); err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Order not found")
	}
	var body struct {
		RazorpayOrderID   string `json:"razorpay_order_id"`
		RazorpayPaymentID string `json:"razorpay_payment_id"`
		Signature         string `json:"razorpay_signature"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
	}
	if !razorpay.VerifyPaymentSignature(body.RazorpayOrderID, body.RazorpayPaymentID, body.Signature) {
		return response.Error(c, fiber.StatusBadRequest, "SIGNATURE_INVALID", "Payment signature verification failed")
	}
	order, err := h.svc.ConfirmStorePayment(c.Context(), uint(id), "razorpay", body.RazorpayPaymentID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CONFIRM_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, order, "Payment verified")
}

// PayStoreFromWallet settles an order from the user's wallet.
func (h *Handler) PayStoreFromWallet(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Auth required")
	}
	if _, err := h.ownsOrder(c, uint(id), userID); err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Order not found")
	}
	order, err := h.svc.ConfirmStorePayment(c.Context(), uint(id), "wallet", fmt.Sprintf("wallet-%d", id))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "PAYMENT_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, order, "Paid from wallet")
}

func (h *Handler) CheckDelivery(c *fiber.Ctx) error {
	pincode := c.Query("pincode")
	// Mock delivery check logic — can be extended with DB/3PL lookup
	if len(pincode) == 6 && pincode[0] != '0' {
		return response.JSON(c, fiber.StatusOK, fiber.Map{
			"pincode":        pincode,
			"serviceable":    true,
			"estimated_days": 2,
		}, "Delivery availability fetched")
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"pincode":     pincode,
		"serviceable": false,
		"error":       "Invalid or unserviceable pincode",
	}, "Delivery availability fetched")
}

func (h *Handler) SearchProducts(c *fiber.Ctx) error {
	search := c.Query("search")
	category := c.Query("category")
	brand := c.Query("brand")
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)
	if limit > 100 {
		limit = 100
	}
	products, err := h.svc.SearchProducts(c.Context(), search, category, brand, limit, offset)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, products, "Products fetched")
}

func (h *Handler) GetProduct(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	product, err := h.svc.GetProductByID(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Product not found")
	}
	return response.JSON(c, fiber.StatusOK, product, "Product details fetched")
}

func (h *Handler) ListCategories(c *fiber.Ctx) error {
	// Simple mock or fetch from products
	return response.JSON(c, fiber.StatusOK, []string{"Wellness", "Prescription", "OTC"}, "Categories fetched")
}

func (h *Handler) CreateOrder(c *fiber.Ctx) error {
	var body struct {
		Items           []OrderItemRequest `json:"items"`
		DeliveryAddress string             `json:"delivery_address"`
		PharmacyID      uint               `json:"pharmacy_id"`
		ContactPhone    string             `json:"contact_phone"`
		ContactEmail    string             `json:"contact_email"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	userID := c.Locals("user_id").(uint)

	// Default to Hub 1 if not specified (legacy/simple support)
	pharmacyID := body.PharmacyID
	if pharmacyID == 0 {
		pharmacyID = 1
	}

	order, err := h.svc.CreateOrder(c.Context(), userID, body.Items, body.DeliveryAddress, pharmacyID, body.ContactPhone, body.ContactEmail)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, order, "Order created successfully")
}

func (h *Handler) GetOrders(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	orders, err := h.svc.GetOrders(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, orders, "User orders fetched")
}

func (h *Handler) GetOrderByID(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	userID := c.Locals("user_id").(uint)

	order, err := h.svc.GetOrderByID(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Order not found")
	}

	if order.UserID != userID {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You do not own this order")
	}

	return response.JSON(c, fiber.StatusOK, order, "Order fetched")
}

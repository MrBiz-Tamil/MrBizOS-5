package billing

import (
	"fmt"
	"os"
	"strings"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type RazorpayOrderRequest struct {
	Amount  int    `json:"amount"`
	Receipt string `json:"receipt"`
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	billingGroup := router.Group("/billing", middleware.Protected())
	billingGroup.Get("/config", h.GetBillingConfig)
	billingGroup.Post("/orders", h.CreateOrders)
	billingGroup.Post("/invoices", h.CreateInvoice)
	billingGroup.Post("/invoices/:id/pay", h.PayInvoice)
	billingGroup.Post("/verify", h.VerifyPayment)
	billingGroup.Post("/invoices/:id/stripe/initiate", h.InitiateStripe)
	billingGroup.Post("/invoices/:id/stripe/verify", h.VerifyStripe)
	billingGroup.Post("/corporate-pay", h.CorporatePay)
	billingGroup.Post("/payments/:id/refund", h.RefundPayment)
}

// GetBillingConfig returns the billing configuration
// @Summary Get Billing Config
// @Description Fetch billing configuration including Razorpay keys and test mode status
// @Tags Billing
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} response.StandardResponse{data=map[string]interface{}} "Config fetched"
// @Router /api/v1/billing/config [get]
func (h *Handler) GetBillingConfig(c *fiber.Ctx) error {
	keyID := os.Getenv("RAZORPAY_KEY_ID")
	if keyID == "" {
		keyID = "rzp_test_placeholder"
	}
	isTest := !strings.HasPrefix(keyID, "rzp_live_")
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"razorpay_key_id": keyID,
		"is_test_mode":    isTest,
	}, "Billing configuration fetched")
}

// CreateOrders creates a new Razorpay order
// @Summary Create Payment Order
// @Description Create a new payment order for Razorpay integration
// @Tags Billing
// @Security ApiKeyAuth
// @Param request body RazorpayOrderRequest true "Order Request (amount, receipt)"
// @Success 200 {object} response.StandardResponse{data=map[string]interface{}} "Order created"
// @Router /api/v1/billing/orders [post]
func (h *Handler) CreateOrders(c *fiber.Ctx) error {
	var req RazorpayOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	order, err := h.service.CreateRazorpayOrder(req.Amount, req.Receipt)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ORDER_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, order, "Razorpay order created")
}

type CreateInvoiceRequest struct {
	AppointmentID uint    `json:"appointment_id"`
	PatientID     uint    `json:"patient_id"`
	Amount        float64 `json:"amount"`
}

// CreateInvoice creates a new invoice for an appointment
// @Summary Create Invoice
// @Description Generate a new billing invoice for a specific appointment
// @Tags Billing
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body CreateInvoiceRequest true "Invoice Request"
// @Success 201 {object} response.StandardResponse{data=map[string]interface{}} "Invoice created"
// @Router /api/v1/billing/invoices [post]
func (h *Handler) CreateInvoice(c *fiber.Ctx) error {
	var req CreateInvoiceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	amountCents := int64(req.Amount * 100)
	invoice, err := h.service.CreateInvoice(req.AppointmentID, amountCents, req.PatientID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, invoice, "Invoice created successfully")
}

// PayInvoice processes a mock payment for an invoice
// @Summary Pay Invoice (Mock)
// @Description Process a payment for an invoice using mock credit card data
// @Tags Billing
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Invoice ID"
// @Success 200 {object} response.StandardResponse{data=map[string]interface{}} "Payment successful"
// @Router /api/v1/billing/invoices/{id}/pay [post]
func (h *Handler) PayInvoice(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}

	payment, err := h.service.ProcessPaymentMock(uint(id), "mock_card")
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PAYMENT_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"payment": payment,
	}, "Payment successful (Mock)")
}

type VerifyPaymentRequest struct {
	InvoiceID         uint   `json:"invoice_id"`
	RazorpayOrderID   string `json:"razorpay_order_id"`
	RazorpayPaymentID string `json:"razorpay_payment_id"`
	Signature         string `json:"signature"`
}

// VerifyPayment verifies a Razorpay payment signature
// @Summary Verify Payment
// @Description Verify the authenticity of a completed Razorpay transaction via signature
// @Tags Billing
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body VerifyPaymentRequest true "Verification Request"
// @Success 200 {object} response.StandardResponse{data=map[string]interface{}} "Payment verified"
// @Router /api/v1/billing/verify [post]
func (h *Handler) VerifyPayment(c *fiber.Ctx) error {
	var req VerifyPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	payment, err := h.service.VerifyPayment(req.InvoiceID, req.RazorpayOrderID, req.RazorpayPaymentID, req.Signature)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "VERIFY_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"payment": payment,
	}, "Payment verified successfully")
}

// InitiateStripe creates a Stripe PaymentIntent for a consultation invoice in
// the customer's currency (amount computed server-side from the INR invoice).
func (h *Handler) InitiateStripe(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var body struct {
		Currency string `json:"currency"`
	}
	_ = c.BodyParser(&body)
	intent, err := h.service.CreateStripeIntent(uint(id), body.Currency)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "STRIPE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, intent, "Stripe payment intent created")
}

// VerifyStripe confirms a consultation payment via server-side intent retrieval.
func (h *Handler) VerifyStripe(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var body struct {
		PaymentIntentID string `json:"payment_intent_id"`
	}
	if err := c.BodyParser(&body); err != nil || body.PaymentIntentID == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "payment_intent_id required")
	}
	payment, err := h.service.VerifyStripePayment(uint(id), body.PaymentIntentID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "VERIFY_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"payment": payment}, "Payment verified")
}

// CorporatePay handles corporate billing payments
// @Summary Corporate Payment
// @Description Process a billing payment against a corporate account or credit limit
// @Tags Billing
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body map[string]interface{} true "Corporate Payment Request"
// @Success 200 {object} response.StandardResponse{data=map[string]interface{}} "Payment processed"
// @Router /api/v1/billing/corporate-pay [post]
func (h *Handler) CorporatePay(c *fiber.Ctx) error {
	var req struct {
		Amount    float64 `json:"amount"`
		PatientID uint    `json:"patient_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	amountCents := int64(req.Amount * 100)
	res, err := h.service.ProcessCorporatePayment(amountCents, req.PatientID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PAYMENT_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, res, "Corporate payment processed")
}

// RefundPayment relies heavily on external models like Appointment and Ledger.
// In a fully strictly separated DDD, it would communicate via events or interfaces.
// Kept similar to old handler logic to maintain functionality during Phase 3 scale.
// RefundPayment processes a refund for an appointment payment
// @Summary Refund Payment
// @Description Issues a refund for a previously completed appointment payment back to the patient's wallet
// @Tags Billing
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Appointment ID"
// @Success 200 {object} response.StandardResponse{data=map[string]interface{}} "Refund successful"
// @Router /api/v1/billing/payments/{id}/refund [post]
func (h *Handler) RefundPayment(c *fiber.Ctx) error {
	idStr := c.Params("id")
	appointmentID, err := strconv.Atoi(idStr)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid Appointment ID")
	}

	var appt appointment.Appointment
	if err := database.DB.First(&appt, appointmentID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
	}

	svc := finance.NewService(finance.NewRepository(database.DB))
	patientWallet := fmt.Sprintf("PatientWallet:%d", appt.PatientID)

	walletAcct, err := svc.GetOrCreateAccount(c.Context(), patientWallet, finance.TypeLiability)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "WALLET_ERROR", "Wallet error")
	}

	gatewayAcct, err := svc.GetOrCreateAccount(c.Context(), "Gateway:Razorpay", finance.TypeAsset)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "GATEWAY_ERROR", "Gateway account error")
	}

	var totalAmount int64 = 150000
	refundAmount := totalAmount

	entries := []finance.EntryData{
		{
			AccountID: gatewayAcct.ID,
			Direction: finance.DirectionCredit,
			Amount:    refundAmount,
		},
		{
			AccountID: walletAcct.ID,
			Direction: finance.DirectionCredit,
			Amount:    refundAmount,
		},
	}

	refID := "REFUND-" + strconv.Itoa(appointmentID)
	txn, err := svc.RecordTransaction(c.Context(), refID, "REFUND", "Appointment Cancellation Refund", entries)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "TRANSACTION_FAILED", "Refund transaction failed: "+err.Error())
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"payment_id":     appointmentID,
		"transaction_id": txn.ID,
	}, "Refund processed to wallet successfully")
}

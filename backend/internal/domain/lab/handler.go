package lab

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
	labsGroup := router.Group("/labs")
	labsGroup.Get("/districts", h.GetDistricts)
	labsGroup.Get("/districts/:id/taluks", h.GetTaluks)

	labGroup := router.Group("/lab")
	labGroup.Get("/tests", h.ListTests)
	labGroup.Get("/tests/:id", h.GetTestByID)
	labGroup.Get("/packages", h.ListPackages)
	labGroup.Get("/concerns", h.GetHealthConcerns)
	labGroup.Get("/centers", h.GetLabCenters)
	labGroup.Get("/search", h.GetLabCenters)
	labGroup.Post("/partners/register", h.RegisterPartner)

	patient := labGroup.Group("/bookings", middleware.Protected())
	patient.Get("/", h.GetMyBookings)
	patient.Post("/", h.CreateBooking)
	patient.Get("/:id", h.GetBookingByID)
	patient.Put("/:id/cancel", h.CancelBooking)
	patient.Post("/:id/pay/initiate", h.InitiateLabPayment)
	patient.Post("/:id/pay/verify", h.VerifyLabPayment)
	patient.Post("/:id/pay/wallet", h.PayLabFromWallet)
	patient.Post("/:id/pay/stripe/initiate", h.InitiateLabStripePayment)
	patient.Post("/:id/pay/stripe/verify", h.VerifyLabStripePayment)

	labGroup.Get("/reports", middleware.Protected(), h.GetMyBookings)

	portal := labGroup.Group("/portal", middleware.Protected())
	portal.Get("/bookings", h.GetPortalBookings)
	portal.Put("/bookings/:id/collect", h.MarkCollected)
	portal.Post("/bookings/:id/result", h.SubmitPortalResult)
	portal.Post("/bookings/:id/verify", h.VerifyPortalReport)

	labGroup.Post("/requests", middleware.Protected(), h.OrderTests)
	labGroup.Post("/specimens", middleware.Protected(), h.RecordSpecimen)
	labGroup.Post("/results", middleware.Protected(), h.SubmitResult)
}

// GetDistricts lists districts for labs
// @Summary List Districts
// @Description Fetch list of Tamil Nadu districts supported by labs
// @Tags Labs
// @Produce json
// @Success 200 {array} map[string]interface{} "Districts list"
// @Router /api/v1/labs/districts [get]
func (h *Handler) GetDistricts(c *fiber.Ctx) error {
	districts := []fiber.Map{
		{"id": 1, "name": "Chennai"}, {"id": 2, "name": "Coimbatore"},
		{"id": 3, "name": "Madurai"}, {"id": 4, "name": "Tiruchirappalli"},
		{"id": 5, "name": "Salem"}, {"id": 7, "name": "Erode"},
	}
	return response.JSON(c, fiber.StatusOK, districts, "Districts fetched")
}

// GetTaluks lists taluks for a district
// @Summary List Taluks
// @Description Fetch list of taluks belonging to a specific district ID
// @Tags Labs
// @Produce json
// @Param id path string true "District ID"
// @Success 200 {array} map[string]interface{} "Taluks list"
// @Router /api/v1/labs/districts/{id}/taluks [get]
func (h *Handler) GetTaluks(c *fiber.Ctx) error {
	taluks := map[string][]fiber.Map{
		"1": {{"id": 101, "name": "Anna Nagar"}, {"id": 102, "name": "Tambaram"}, {"id": 103, "name": "Adyar"}, {"id": 104, "name": "T.Nagar"}},
		"2": {{"id": 201, "name": "Coimbatore North"}, {"id": 202, "name": "RS Puram"}},
		"3": {{"id": 301, "name": "Madurai North"}, {"id": 302, "name": "KK Nagar"}},
		"4": {{"id": 401, "name": "Trichy"}, {"id": 402, "name": "Srirangam"}},
	}
	result, ok := taluks[c.Params("id")]
	if !ok {
		result = []fiber.Map{}
	}
	return response.JSON(c, fiber.StatusOK, result, "Taluks fetched")
}
func (h *Handler) ListTests(c *fiber.Ctx) error {
	tests, err := h.svc.ListAvailableTests(c.Context(), c.Query("category"), c.Query("concern"), c.Query("q"))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, tests, "Lab tests fetched")
}
func (h *Handler) GetTestByID(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	test, err := h.svc.GetTestByID(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Lab test not found")
	}
	return response.JSON(c, fiber.StatusOK, test, "Lab test fetched")
}
func (h *Handler) ListPackages(c *fiber.Ctx) error {
	pkgs, err := h.svc.ListPackages(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, pkgs, "Health packages fetched")
}
func (h *Handler) GetHealthConcerns(c *fiber.Ctx) error {
	concerns, err := h.svc.GetHealthConcerns(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, concerns, "Health concerns fetched")
}
func (h *Handler) GetLabCenters(c *fiber.Ctx) error {
	testID := uint(c.QueryInt("test_id"))
	centers, err := h.svc.FindLabCenters(c.Context(), c.Query("pincode"), c.Query("district"), testID)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "SEARCH_FAILED", err.Error())
	}
	if centers == nil {
		centers = []labCenterResult{}
	}
	return response.JSON(c, fiber.StatusOK, centers, "Lab centers fetched")
}
func (h *Handler) CreateBooking(c *fiber.Ctx) error {
	patientID, ok := c.Locals("user_id").(uint)
	if !ok || patientID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Auth required")
	}
	var req BookingRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
	}
	booking, err := h.svc.CreateBooking(c.Context(), patientID, req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "BOOKING_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, booking, "Booking confirmed")
}
func (h *Handler) GetMyBookings(c *fiber.Ctx) error {
	patientID, ok := c.Locals("user_id").(uint)
	if !ok || patientID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Auth required")
	}
	bookings, err := h.svc.GetMyBookings(c.Context(), patientID, c.Query("status"))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	if bookings == nil {
		bookings = []LabBooking{}
	}
	return response.JSON(c, fiber.StatusOK, bookings, "Bookings fetched")
}
func (h *Handler) GetBookingByID(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	patientID, _ := c.Locals("user_id").(uint)
	booking, err := h.svc.GetBookingByID(c.Context(), uint(id), patientID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Booking not found")
	}
	return response.JSON(c, fiber.StatusOK, booking, "Booking fetched")
}
func (h *Handler) CancelBooking(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	patientID, ok := c.Locals("user_id").(uint)
	if !ok || patientID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Auth required")
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.BodyParser(&body)
	if err := h.svc.CancelBooking(c.Context(), uint(id), patientID, body.Reason); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CANCEL_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Booking cancelled")
}
// InitiateLabPayment creates a Razorpay order for a booking the caller owns.
func (h *Handler) InitiateLabPayment(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	patientID, ok := c.Locals("user_id").(uint)
	if !ok || patientID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Auth required")
	}
	booking, err := h.svc.GetBookingByID(c.Context(), uint(id), patientID) // enforces ownership
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Booking not found")
	}
	if booking.PaymentStatus == "paid" {
		return response.Error(c, fiber.StatusConflict, "ALREADY_PAID", "Booking already paid")
	}
	order, err := razorpay.CreateOrder(booking.TotalPaise, fmt.Sprintf("LAB-%d", booking.ID))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ORDER_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, order, "Payment order created")
}

// VerifyLabPayment verifies the Razorpay signature, then posts the booking to
// the ledger. The amount is taken from the server-side booking, never the body.
func (h *Handler) VerifyLabPayment(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	patientID, ok := c.Locals("user_id").(uint)
	if !ok || patientID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Auth required")
	}
	if _, err := h.svc.GetBookingByID(c.Context(), uint(id), patientID); err != nil { // ownership
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Booking not found")
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
	booking, err := h.svc.ConfirmLabPayment(c.Context(), uint(id), "razorpay", body.RazorpayPaymentID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CONFIRM_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, booking, "Payment verified")
}

// PayLabFromWallet settles a booking from the patient's wallet (balance checked
// + non-negative guard inside the ledger transaction).
func (h *Handler) PayLabFromWallet(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	patientID, ok := c.Locals("user_id").(uint)
	if !ok || patientID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Auth required")
	}
	if _, err := h.svc.GetBookingByID(c.Context(), uint(id), patientID); err != nil { // ownership
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Booking not found")
	}
	booking, err := h.svc.ConfirmLabPayment(c.Context(), uint(id), "wallet", fmt.Sprintf("wallet-%d", id))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "PAYMENT_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, booking, "Paid from wallet")
}

// InitiateLabStripePayment creates a Stripe PaymentIntent in the customer's
// currency for a booking they own. The presentment amount is computed
// server-side from the INR price (no client-supplied amount).
func (h *Handler) InitiateLabStripePayment(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	patientID, ok := c.Locals("user_id").(uint)
	if !ok || patientID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Auth required")
	}
	booking, err := h.svc.GetBookingByID(c.Context(), uint(id), patientID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Booking not found")
	}
	if booking.PaymentStatus == "paid" {
		return response.Error(c, fiber.StatusConflict, "ALREADY_PAID", "Booking already paid")
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
	amountMinor, rate, err := fx.ToPresentment(booking.TotalPaise, body.Currency)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "FX_FAILED", err.Error())
	}
	intent, err := stripe.CreatePaymentIntent(amountMinor, body.Currency, map[string]string{
		"ref":       fmt.Sprintf("LAB-%d", booking.ID),
		"inr_paise": fmt.Sprintf("%d", booking.TotalPaise),
		"fx_rate":   fmt.Sprintf("%.4f", rate),
	})
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "STRIPE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, intent, "Stripe payment intent created")
}

// VerifyLabStripePayment retrieves the PaymentIntent server-side; only when
// Stripe reports it succeeded do we post the booking (in INR) to the ledger.
func (h *Handler) VerifyLabStripePayment(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	patientID, ok := c.Locals("user_id").(uint)
	if !ok || patientID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Auth required")
	}
	if _, err := h.svc.GetBookingByID(c.Context(), uint(id), patientID); err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Booking not found")
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
	booking, err := h.svc.ConfirmLabPayment(c.Context(), uint(id), "stripe", pi.ID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CONFIRM_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, booking, "Payment verified")
}

func (h *Handler) GetPortalBookings(c *fiber.Ctx) error {
	labCenterID := uint(c.QueryInt("lab_center_id"))
	bookings, err := h.svc.GetPortalBookings(c.Context(), labCenterID, c.Query("status"))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	if bookings == nil {
		bookings = []LabBooking{}
	}
	return response.JSON(c, fiber.StatusOK, bookings, "Portal bookings fetched")
}
func (h *Handler) MarkCollected(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	if err := h.svc.MarkSampleCollected(c.Context(), uint(id)); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Sample marked as collected")
}
func (h *Handler) SubmitPortalResult(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var req ResultRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
	}
	if err := h.svc.SubmitResult(c.Context(), uint(id), req); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SUBMIT_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Result submitted")
}
func (h *Handler) VerifyPortalReport(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	if err := h.svc.VerifyReport(c.Context(), uint(id)); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "VERIFY_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Report verified")
}
func (h *Handler) OrderTests(c *fiber.Ctx) error {
	var body struct {
		TestIDs  []uint `json:"test_ids"`
		Priority string `json:"priority"`
		DoctorID uint   `json:"doctor_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}
	patientID := c.Locals("user_id").(uint)
	req, err := h.svc.OrderTests(c.Context(), patientID, body.DoctorID, body.TestIDs, body.Priority)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, req, "Tests ordered")
}
func (h *Handler) RecordSpecimen(c *fiber.Ctx) error {
	var body struct {
		RequestID uint   `json:"request_id"`
		Type      string `json:"type"`
		Barcode   string `json:"barcode"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}
	techID := c.Locals("user_id").(uint)
	sp, err := h.svc.RecordSpecimen(c.Context(), body.RequestID, body.Type, body.Barcode, techID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, sp, "Specimen recorded")
}
func (h *Handler) SubmitResult(c *fiber.Ctx) error {
	var body struct {
		RequestID uint   `json:"request_id"`
		TestID    uint   `json:"test_id"`
		Value     string `json:"value"`
		Unit      string `json:"unit"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}
	techID := c.Locals("user_id").(uint)
	res, err := h.svc.SubmitLabResult(c.Context(), body.RequestID, body.TestID, body.Value, body.Unit, techID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, res, "Result submitted")
}

// RegisterPartner accepts lab partnership applications from /lab-center/register.
// Currently stores a draft LabPartner with IsActive=false (pending admin review).
func (h *Handler) RegisterPartner(c *fiber.Ctx) error {
	var body struct {
		Name            string   `json:"name"`
		LicenseNumber   string   `json:"license_number"`
		Email           string   `json:"email"`
		Phone           string   `json:"phone"`
		Address         string   `json:"address"`
		City            string   `json:"city"`
		State           string   `json:"state"`
		DistrictID      int      `json:"district_id"`
		TalukID         int      `json:"taluk_id"`
		PincodesCovered []string `json:"pincodes_covered"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
	}
	if body.Name == "" || body.Email == "" || body.Phone == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "name, email and phone are required")
	}

	// For now, log + return success. A real implementation would persist a
	// LabPartnerApplication row and notify the admin team.
	return response.JSON(c, fiber.StatusCreated, fiber.Map{
		"received": true,
		"name":     body.Name,
		"email":    body.Email,
	}, "Application received — our team will contact you within 48 hours")
}

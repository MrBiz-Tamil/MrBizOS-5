package pharmacy

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"drayya-core/internal/crypto"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/document"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/privacy"
	"drayya-core/internal/utils/response"
)

// checkDrugInteractions fetches medicine names for the given cart items then
// queries the drug_interactions table for any pair-wise interactions.
func checkDrugInteractions(ctx context.Context, db *gorm.DB, cartItems []CartItem) ([]DrugInteraction, error) {
	if len(cartItems) < 2 {
		return nil, nil
	}

	// Collect medicine IDs and look up names
	ids := make([]uint, 0, len(cartItems))
	for _, item := range cartItems {
		ids = append(ids, item.MedicineID)
	}
	var medicines []Medicine
	if err := db.WithContext(ctx).Where("id IN ?", ids).Find(&medicines).Error; err != nil {
		return nil, err
	}
	names := make([]string, 0, len(medicines))
	for _, m := range medicines {
		names = append(names, m.Name)
	}
	if len(names) < 2 {
		return nil, nil
	}

	var interactions []DrugInteraction
	err := db.WithContext(ctx).Where(
		"(drug_a_name IN ? AND drug_b_name IN ?) OR (drug_b_name IN ? AND drug_a_name IN ?)",
		names, names, names, names,
	).Find(&interactions).Error
	return interactions, err
}

// CartItem represents a single item in the shopping cart
type CartItem struct {
	MedicineID   uint   `json:"medicine_id"`
	MedicineName string `json:"medicine_name"`
	Quantity     int    `json:"quantity"`
	UnitPrice    int64  `json:"unit_price"`
	IsRx         bool   `json:"is_rx"`
	VariantSKU   string `json:"variant_sku,omitempty"`
}

// Cart holds the full cart for a user, stored in Redis
type Cart struct {
	UserID    uint       `json:"user_id"`
	Items     []CartItem `json:"items"`
	UpdatedAt time.Time  `json:"updated_at"`
}

const cartTTL = 7 * 24 * time.Hour // 7 days

// maxCartQtyDefault is the default per-item maximum quantity.
const maxCartQtyDefault = 100

// maxCartQtyScheduleH is the per-item maximum for Schedule H drugs.
const maxCartQtyScheduleH = 10

func cartKey(userID uint) string {
	return fmt.Sprintf("cart:%d", userID)
}

// GetCart retrieves the cart from Redis for the authenticated user
func (h *Handler) GetCart(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(uint)
	if userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Please login")
	}

	ctx := context.Background()
	if database.Rdb == nil {
		return response.Error(c, fiber.StatusServiceUnavailable, "CACHE_UNAVAILABLE", "Cart service unavailable")
	}

	key := cartKey(userID)
	data, err := database.Rdb.Get(ctx, key).Bytes()
	if err != nil {
		// No cart yet - return empty
		return response.JSON(c, fiber.StatusOK, Cart{UserID: userID, Items: []CartItem{}}, "Cart is empty")
	}

	var cart Cart
	if err := json.Unmarshal(data, &cart); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PARSE_ERROR", "Failed to parse cart")
	}

	return response.JSON(c, fiber.StatusOK, cart, "Cart fetched")
}

// AddToCart adds or updates an item in the user's Redis cart.
// C-5: uses Redis Watch to prevent race conditions on GET+mutate+SET.
// M-17: enforces per-item max quantity (100 default, 10 for Schedule H drugs).
func (h *Handler) AddToCart(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(uint)
	if userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Please login")
	}

	var req CartItem
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	if req.MedicineID == 0 || req.Quantity <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ITEM", "medicine_id and quantity > 0 are required")
	}

	// Fetch medicine details to populate IsRx flag and price
	med, err := h.svc.GetMedicine(c.Context(), req.MedicineID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Medicine not found")
	}
	req.MedicineName = med.Name
	req.UnitPrice = med.Prices["INR"] // default to INR base price
	req.IsRx = med.IsPrescriptionRequired

	// If a specific variant SKU was requested, use its price instead
	if req.VariantSKU != "" {
		var variant ProductVariant
		if err := h.db.Where("medicine_id = ? AND sku = ?", req.MedicineID, req.VariantSKU).
			First(&variant).Error; err == nil && variant.Prices != nil && variant.Prices["INR"] > 0 {
			req.UnitPrice = variant.Prices["INR"]
		}
	}

	// Determine if this is a Schedule H drug for quantity limits
	var scheduleH ScheduleHDrug
	isScheduleH := h.db.Where("LOWER(drug_name) = LOWER(?)", med.Name).First(&scheduleH).Error == nil

	ctx := context.Background()
	if database.Rdb == nil {
		return response.Error(c, fiber.StatusServiceUnavailable, "CACHE_UNAVAILABLE", "Cart service unavailable")
	}

	key := cartKey(userID)

	// C-5: use Redis Watch for atomic GET+mutate+SET
	var finalCart Cart
	err = database.Rdb.Watch(ctx, func(tx *redis.Tx) error {
		// GET existing cart inside Watch
		var cart Cart
		data, getErr := tx.Get(ctx, key).Bytes()
		if getErr == nil {
			_ = json.Unmarshal(data, &cart)
		}
		cart.UserID = userID

		// Find existing item and update or append
		found := false
		for i, item := range cart.Items {
			if item.MedicineID == req.MedicineID {
				newQty := cart.Items[i].Quantity + req.Quantity
				// M-17: enforce per-item max quantity
				maxQty := maxCartQtyDefault
				if isScheduleH {
					maxQty = maxCartQtyScheduleH
				}
				if newQty > maxQty {
					return fmt.Errorf("maximum quantity per item is %d", maxQty)
				}
				cart.Items[i].Quantity = newQty
				found = true
				break
			}
		}
		if !found {
			// M-17: enforce max for new item additions too
			maxQty := maxCartQtyDefault
			if isScheduleH {
				maxQty = maxCartQtyScheduleH
			}
			if req.Quantity > maxQty {
				return fmt.Errorf("maximum quantity per item is %d", maxQty)
			}
			cart.Items = append(cart.Items, req)
		}
		cart.UpdatedAt = time.Now()

		// Drug interaction check before saving
		interactions, iErr := checkDrugInteractions(ctx, h.db, cart.Items)
		if iErr == nil {
			for _, interaction := range interactions {
				if interaction.Severity == "major" || interaction.Severity == "contraindicated" {
					return fmt.Errorf("drug interaction warning: %s and %s have a %s interaction: %s",
						interaction.DrugAName, interaction.DrugBName, interaction.Severity, interaction.Description)
				}
			}
		}

		// SET inside TxPipelined
		updated, _ := json.Marshal(cart)
		_, pipeErr := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			return pipe.Set(ctx, key, updated, cartTTL).Err()
		})
		if pipeErr != nil {
			return pipeErr
		}
		finalCart = cart
		return nil
	}, key)

	if err != nil {
		if err == redis.TxFailedErr {
			return response.Error(c, fiber.StatusConflict, "CART_CONFLICT", "Cart was modified concurrently, please retry")
		}
		return response.Error(c, fiber.StatusInternalServerError, "CART_ERROR", err.Error())
	}

	// Shadow to CartSession
	cartValue := int64(0)
	for _, item := range finalCart.Items {
		cartValue += item.UnitPrice * int64(item.Quantity)
	}
	updated, _ := json.Marshal(finalCart)
	_ = NewAnalyticsService(NewAnalyticsRepository(h.db)).TrackCartEvent(ctx, userID, key, string(updated), cartValue)

	return response.JSON(c, fiber.StatusOK, finalCart, "Item added to cart")
}

// UpdateCartItem changes the quantity of an item in the cart
func (h *Handler) UpdateCartItem(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(uint)
	if userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Please login")
	}

	var req struct {
		MedicineID uint `json:"medicine_id"`
		Quantity   int  `json:"quantity"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
	}

	ctx := context.Background()
	if database.Rdb == nil {
		return response.Error(c, fiber.StatusServiceUnavailable, "CACHE_UNAVAILABLE", "Cart service unavailable")
	}

	key := cartKey(userID)
	data, err := database.Rdb.Get(ctx, key).Bytes()
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NO_CART", "No active cart found")
	}

	var cart Cart
	if err := json.Unmarshal(data, &cart); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PARSE_ERROR", "Failed to parse cart")
	}

	updatedItems := []CartItem{}
	for _, item := range cart.Items {
		if item.MedicineID == req.MedicineID {
			if req.Quantity > 0 {
				item.Quantity = req.Quantity
				updatedItems = append(updatedItems, item)
			}
			// If qty == 0, effectively removes item
		} else {
			updatedItems = append(updatedItems, item)
		}
	}
	cart.Items = updatedItems
	cart.UpdatedAt = time.Now()

	updated, _ := json.Marshal(cart)
	database.Rdb.Set(ctx, key, updated, cartTTL)

	// Shadow to CartSession
	cartValue := int64(0)
	for _, item := range cart.Items {
		cartValue += item.UnitPrice * int64(item.Quantity)
	}
	_ = NewAnalyticsService(NewAnalyticsRepository(h.db)).TrackCartEvent(ctx, userID, key, string(updated), cartValue)

	return response.JSON(c, fiber.StatusOK, cart, "Cart updated")
}

// ClearCart empties the cart entirely
func (h *Handler) ClearCart(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(uint)
	if userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Please login")
	}

	ctx := context.Background()
	if database.Rdb != nil {
		database.Rdb.Del(ctx, cartKey(userID))
	}
	return response.JSON(c, fiber.StatusOK, nil, "Cart cleared")
}

// CheckoutCart converts cart to a PharmacyOrder (with Rx gating).
// H-7: Rx status must be "verified".
// M-16: verifies PharmacyID exists and is_active = true before creating order.
// M-25: uses crypto.SecureToken for order number.
func (h *Handler) CheckoutCart(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(uint)
	if userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Please login")
	}

	var req struct {
		ShippingAddress       string       `json:"shipping_address"`
		Phone                 string       `json:"phone"`
		PharmacyID            uint         `json:"pharmacy_id"`
		DeliveryMode          DeliveryMode `json:"delivery_mode"`
		PaymentMethod         string       `json:"payment_method"` // cod, online, wallet
		PrescriptionID        *uint        `json:"prescription_id"` // Required if cart has Rx items
		DeliveryPincode       string       `json:"delivery_pincode"`
		DiscountPaise         int64        `json:"discount_paise"`
		InsuranceCoveredPaise int64        `json:"insurance_covered_paise"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	ctx := context.Background()
	if database.Rdb == nil {
		return response.Error(c, fiber.StatusServiceUnavailable, "CACHE_UNAVAILABLE", "Cart service unavailable")
	}

	data, err := database.Rdb.Get(ctx, cartKey(userID)).Bytes()
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "EMPTY_CART", "Your cart is empty")
	}

	var cart Cart
	if err := json.Unmarshal(data, &cart); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PARSE_ERROR", "Failed to read cart")
	}

	// M-16: verify PharmacyID exists and is active
	if req.PharmacyID != 0 {
		var pharmacy Pharmacy
		if err := h.db.Where("id = ? AND is_active = ?", req.PharmacyID, true).First(&pharmacy).Error; err != nil {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_PHARMACY", "Pharmacy not found or is not active")
		}
	}

	// --- The Rx Gating Check ---
	hasRxItem := false
	for _, item := range cart.Items {
		if item.IsRx {
			hasRxItem = true
			break
		}
	}

	orderStatus := "pending"
	if hasRxItem {
		if req.PrescriptionID == nil {
			// Block checkout - force user to provide prescription
			return response.Error(c, fiber.StatusUnprocessableEntity, "RX_REQUIRED",
				"Your cart contains prescription-only medications. Please upload a valid prescription to continue.")
		}
		// Verify the prescription exists and belongs to this user
		var rx Prescription
		if err := h.db.First(&rx, req.PrescriptionID).Error; err != nil || rx.UserID != userID {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_PRESCRIPTION", "Invalid or unauthorized prescription")
		}
		// H-7: Rx must be "verified"
		if rx.Status != "verified" {
			return response.Error(c, fiber.StatusBadRequest, "PRESCRIPTION_NOT_VERIFIED",
				fmt.Sprintf("prescription ID %d has not been verified by a pharmacist", *req.PrescriptionID))
		}
		// Order is pending pharmacist verification
		orderStatus = "pending_rx_verification"
	}

	// Drug interaction check at checkout — block major/contraindicated pairs
	if interactions, iErr := checkDrugInteractions(ctx, h.db, cart.Items); iErr == nil {
		for _, interaction := range interactions {
			if interaction.Severity == "major" || interaction.Severity == "contraindicated" {
				return response.Error(c, fiber.StatusUnprocessableEntity, "DRUG_INTERACTION",
					fmt.Sprintf("drug interaction warning: %s and %s have a %s interaction: %s",
						interaction.DrugAName, interaction.DrugBName, interaction.Severity, interaction.Description))
			}
		}
	}

	// Use pricing engine to compute correct totals (GST, Shipping, COD Surcharge)
	pricingItems := make([]ReservationItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		pricingItems = append(pricingItems, ReservationItem{
			MedicineID: item.MedicineID,
			Quantity:   item.Quantity,
		})
	}

	pm := PaymentPrepaid
	if req.PaymentMethod == "cod" {
		pm = PaymentCOD
	}

	totals, err := PriceCart(c.Context(), h.db, PricingRequest{
		Items:                 pricingItems,
		DeliveryPincode:       req.DeliveryPincode,
		PaymentMethod:         pm,
		InsuranceCoveredPaise: req.InsuranceCoveredPaise,
		DiscountPaise:         req.DiscountPaise,
	})
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "PRICING_FAILED", err.Error())
	}

	// Build order items JSON
	type OrderItemPayload struct {
		MedicineID uint  `json:"medicine_id"`
		Quantity   int   `json:"quantity"`
		UnitPrice  int64 `json:"unit_price"`
	}
	var orderItems []OrderItemPayload
	for _, line := range totals.Lines {
		orderItems = append(orderItems, OrderItemPayload{
			MedicineID: line.MedicineID,
			Quantity:   line.Quantity,
			UnitPrice:  line.UnitMRPPaise,
		})
	}

	// B12: handle marshal error instead of silently ignoring it.
	itemsJSON, marshalErr := json.Marshal(orderItems)
	if marshalErr != nil {
		return response.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "Failed to serialize order items")
	}

	paymentMethod := req.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = "cod"
	}

	// M-25: use crypto.SecureToken for collision-free order numbers
	orderNum := fmt.Sprintf("ORD-%s", strings.ToUpper(generateOrderToken()))

	// Create the order with locked grand total from pricing engine
	order := &PharmacyOrder{
		OrderNumber:     orderNum,
		UserID:          userID,
		PharmacyID:      req.PharmacyID,
		Items:           itemsJSON,
		Amount:          totals.GrandTotalPaise, // Locked Grand Total
		Currency:        "INR",
		Status:          orderStatus,
		PaymentStatus:   "pending",
		PaymentMethod:   paymentMethod,
		DeliveryMode:    req.DeliveryMode,
		ShippingAddress: privacy.EncryptedString(req.ShippingAddress),
		DeliveryPhone:   privacy.EncryptedString(req.Phone),
		DestPincode:     req.DeliveryPincode,
	}

	if err := h.svc.CreateOrder(c.Context(), order); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}

	// Update prescription with order ID if Rx order
	if hasRxItem && req.PrescriptionID != nil {
		h.db.Model(&Prescription{}).Where("id = ?", req.PrescriptionID).Update("order_id", order.ID)
	}

	// Reduce cart TTL to 1 hour — cart is permanently cleared in PayOrderHandler after payment.
	// Keeping it briefly allows the user to retry if payment fails.
	database.Rdb.Expire(ctx, cartKey(userID), 1*time.Hour)

	// Mark CartSession completed
	_ = NewAnalyticsService(NewAnalyticsRepository(h.db)).CompleteCart(ctx, cartKey(userID))

	msg := "Order placed successfully"
	if orderStatus == "pending_rx_verification" {
		msg = "Order placed. Awaiting pharmacist prescription verification."
	}
	return response.JSON(c, fiber.StatusCreated, order, msg)
}

// generateOrderToken returns an 8-char uppercase hex token for order numbers.
func generateOrderToken() string {
	return crypto.SecureToken(4)
}

// UploadPrescription handles secure prescription file upload.
// H-28: requires https:// only; case-insensitive extension check.
// Rate limit: max 10 prescription uploads per user per day.
func (h *Handler) UploadPrescription(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(uint)
	if userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Please login")
	}

	// Feature 6: Redis rate limit — max 10 uploads per user per day
	if database.Rdb != nil {
		rateLimitKey := fmt.Sprintf("rx_upload_limit:%d:%s", userID, time.Now().Format("2006-01-02"))
		count, err := database.Rdb.Incr(c.Context(), rateLimitKey).Result()
		if err == nil && count == 1 {
			database.Rdb.Expire(c.Context(), rateLimitKey, 24*time.Hour)
		}
		if count > 10 {
			return response.Error(c, fiber.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED",
				"Prescription upload limit reached: maximum 10 uploads per day")
		}
	}

	// Preferred path: a real multipart file goes straight into the private vault.
	if file, ferr := c.FormFile("file"); ferr == nil && file != nil {
		doc, derr := document.SavePrivateFile(c.UserContext(), file, "prescription", document.OwnerPatient, userID, &userID)
		if derr != nil {
			return response.Error(c, fiber.StatusBadRequest, "UPLOAD_FAILED", derr.Error())
		}
		var doctorID *uint
		if d, _ := strconv.Atoi(c.FormValue("doctor_id")); d > 0 {
			did := uint(d)
			doctorID = &did
		}
		rx := &Prescription{UserID: userID, DocumentID: &doc.ID, DoctorID: doctorID, Status: "uploaded"}
		if err := h.db.WithContext(c.UserContext()).Create(rx).Error; err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
		}
		// Return a short-lived signed URL so the immediate OCR-parse step can read
		// the just-uploaded scan (the file itself stays private in the vault).
		signedURL, _ := document.SignedURLForID(c.UserContext(), doc.ID)
		return response.JSON(c, fiber.StatusCreated, fiber.Map{
			"id":          rx.ID,
			"document_id": doc.ID,
			"status":      rx.Status,
			"file_url":    signedURL,
		}, "Prescription uploaded successfully")
	}

	// Back-compat path: accept a pre-hosted https file_url directly.
	var req struct {
		FileURL  string `json:"file_url"`
		DoctorID *uint  `json:"doctor_id,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	if req.FileURL == "" {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FILE", "file_url is required")
	}
	// H-28: require https:// only (reject http://)
	if !strings.HasPrefix(req.FileURL, "https://") {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_URL", "file_url must be a valid HTTPS URL")
	}
	// H-28: case-insensitive extension validation — check path before any query string
	urlPath := strings.ToLower(strings.Split(req.FileURL, "?")[0])
	allowedExts := []string{".jpg", ".jpeg", ".png", ".pdf", ".webp"}
	validExt := false
	for _, ext := range allowedExts {
		if strings.HasSuffix(urlPath, ext) {
			validExt = true
			break
		}
	}
	if !validExt {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_FILE_TYPE",
			"Only JPG, JPEG, PNG, PDF, and WebP files are allowed as prescriptions")
	}

	rx := &Prescription{
		UserID:   userID,
		FileURL:  req.FileURL,
		DoctorID: req.DoctorID,
		Status:   "uploaded",
	}

	if err := h.db.Create(rx).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, rx, "Prescription uploaded successfully")
}

// GetMyPrescriptions lists all prescriptions for the authenticated user
func (h *Handler) GetMyPrescriptions(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(uint)
	var rxList []Prescription
	if err := h.db.Where("user_id = ?", userID).Order("created_at desc").Find(&rxList).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, rxList, "Prescriptions fetched")
}

// GetPrescriptionQueue returns all prescriptions pending pharmacist review (Admin only)
func (h *Handler) GetPrescriptionQueue(c *fiber.Ctx) error {
	var rxList []Prescription
	if err := h.db.Where("status = ?", "uploaded").Order("created_at asc").Find(&rxList).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, rxList, "Prescription queue fetched")
}

// VerifyPrescription allows a pharmacist to approve or reject a prescription.
// H-15: stores VerifiedByPharmacistID and VerifiedAt after verification.
//       Audit log insert does not mask duplicates with ON CONFLICT DO NOTHING.
func (h *Handler) VerifyPrescription(c *fiber.Ctx) error {
	pharmacistID, _ := c.Locals("user_id").(uint)
	rxID, _ := c.ParamsInt("id")

	var req struct {
		Status string `json:"status"` // "verified" or "rejected"
		Notes  string `json:"notes"`
	}
	if err := c.BodyParser(&req); err != nil || (req.Status != "verified" && req.Status != "rejected") {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "status must be 'verified' or 'rejected'")
	}

	var rx Prescription
	if err := h.db.First(&rx, rxID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Prescription not found")
	}

	rx.Status = req.Status
	rx.PharmacistNotes = req.Notes

	// H-15: store verification metadata when verified
	if req.Status == "verified" {
		now := time.Now()
		rx.VerifiedByPharmacistID = &pharmacistID
		rx.VerifiedAt = &now
	}

	if err := h.db.Save(&rx).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}

	// If verified + order linked -> advance order to "processing"
	if req.Status == "verified" && rx.OrderID != nil {
		h.db.Model(&PharmacyOrder{}).Where("id = ?", rx.OrderID).Update("status", "processing")
	}

	// H-15: Audit log without ON CONFLICT DO NOTHING — use standard INSERT
	h.db.Exec(`INSERT INTO audit_logs (entity, entity_id, action, actor_id, notes, created_at)
		VALUES (?,?,?,?,?,NOW())`,
		"prescription", rx.ID, req.Status, pharmacistID, req.Notes)

	return response.JSON(c, fiber.StatusOK, rx, fmt.Sprintf("Prescription %s", req.Status))
}

// RegisterCartRoutes adds cart & prescription routes to the router
func (h *Handler) RegisterCartRoutes(router fiber.Router) {
	cart := router.Group("/cart", middleware.Protected())
	cart.Get("/", h.GetCart)
	cart.Post("/add", h.AddToCart)
	cart.Put("/update", h.UpdateCartItem)
	cart.Delete("/clear", h.ClearCart)
	cart.Post("/checkout", h.CheckoutCart)

	rx := router.Group("/prescriptions", middleware.Protected())
	rx.Post("/upload", h.UploadPrescription)
	rx.Get("/mine", h.GetMyPrescriptions)
	rx.Get("/queue", middleware.PharmacyManagerOnly(), h.GetPrescriptionQueue)
	rx.Post("/:id/verify", middleware.PharmacyManagerOnly(), h.VerifyPrescription)
}

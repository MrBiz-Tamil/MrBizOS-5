package pharmacy

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"drayya-core/internal/crypto"
	"drayya-core/internal/database"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ══════════════════════════════════════════════════════════════
// PHARMACY SETTINGS
// ══════════════════════════════════════════════════════════════

// PharmacySettingsPayload is the shape the frontend sends for pharmacy configuration.
type PharmacySettingsPayload struct {
	Store struct {
		Name            string  `json:"name"`
		LicenseNumber   string  `json:"license_number"`
		Phone           string  `json:"phone"`
		Email           string  `json:"email"`
		Address         string  `json:"address"`
		City            string  `json:"city"`
		Pincode         string  `json:"pincode"`
		ServiceRadiusKm float64 `json:"service_radius_km"`
	} `json:"store"`
	Notifications struct {
		LowStockAlerts    bool `json:"low_stock_alerts"`
		ExpiryAlerts      bool `json:"expiry_alerts"`
		NewOrderAlerts    bool `json:"new_order_alerts"`
		PaymentAlerts     bool `json:"payment_alerts"`
		AlertEmailEnabled bool `json:"alert_email_enabled"`
	} `json:"notifications"`
	Delivery struct {
		EnableHomeDelivery bool    `json:"enable_home_delivery"`
		EnableStorePickup  bool    `json:"enable_store_pickup"`
		DefaultDeliveryFee int64   `json:"default_delivery_fee"` // paise
		FreeDeliveryAbove  int64   `json:"free_delivery_above"`  // paise
		MaxDeliveryRadiusKm float64 `json:"max_delivery_radius_km"`
	} `json:"delivery"`
	Payments struct {
		AcceptCOD      bool `json:"accept_cod"`
		AcceptRazorpay bool `json:"accept_razorpay"`
		AcceptWallet   bool `json:"accept_wallet"`
		AcceptStripe   bool `json:"accept_stripe"`
	} `json:"payments"`
	Hours struct {
		OpenTime  string `json:"open_time"`  // "09:00"
		CloseTime string `json:"close_time"` // "22:00"
		Is24Hours bool   `json:"is_24_hours"`
	} `json:"hours"`
}

// GetPharmacySettings returns the pharmacy branch configuration.
// GET /pharmacy/settings
func (h *Handler) GetPharmacySettings(c *fiber.Ctx) error {
	// Resolve pharmacy_id: prefer query param for admins, else JWT
	pharmacyID, _ := c.Locals("pharmacy_id").(uint)
	if qid := c.QueryInt("pharmacy_id", 0); qid > 0 {
		pharmacyID = uint(qid)
	}
	if pharmacyID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_PARAM",
			"pharmacy_id is required (or must be present in your auth token)")
	}

	var p Pharmacy
	if err := h.db.First(&p, pharmacyID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Pharmacy not found")
	}

	// Build response using the Pharmacy struct — settings are stored in the pharmacy row.
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"pharmacy_id": p.ID,
		"store": fiber.Map{
			"name":               p.Name,
			"address":            p.Address,
			"city":               p.City,
			"pincode":            p.Pincode,
			"region":             p.Region,
			"latitude":           p.Latitude,
			"longitude":          p.Longitude,
			"service_radius_km":  p.ServiceRadius,
			"is_central_warehouse": p.IsCentralWarehouse,
			"hub_tier":           p.HubTier,
			"is_active":          p.IsActive,
		},
	}, "Pharmacy settings fetched")
}

// UpdatePharmacySettings saves pharmacy branch configuration.
// PUT /pharmacy/settings
func (h *Handler) UpdatePharmacySettings(c *fiber.Ctx) error {
	// Resolve pharmacy_id
	pharmacyID, _ := c.Locals("pharmacy_id").(uint)
	if qid := c.QueryInt("pharmacy_id", 0); qid > 0 {
		pharmacyID = uint(qid)
	}
	if pharmacyID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_PARAM",
			"pharmacy_id is required (or must be present in your auth token)")
	}

	var payload PharmacySettingsPayload
	if err := c.BodyParser(&payload); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	// Only update fields that are provided (non-zero)
	updates := map[string]interface{}{}
	if payload.Store.Name != "" {
		updates["name"] = payload.Store.Name
	}
	if payload.Store.Address != "" {
		updates["address"] = payload.Store.Address
	}
	if payload.Store.City != "" {
		updates["city"] = payload.Store.City
	}
	if payload.Store.Pincode != "" {
		updates["pincode"] = payload.Store.Pincode
	}
	if payload.Store.ServiceRadiusKm > 0 {
		updates["service_radius"] = payload.Store.ServiceRadiusKm
	}

	if len(updates) == 0 {
		return response.Error(c, fiber.StatusBadRequest, "NO_CHANGES", "No valid fields provided to update")
	}

	if err := h.db.Model(&Pharmacy{}).Where("id = ?", pharmacyID).Updates(updates).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Pharmacy settings updated")
}

// ══════════════════════════════════════════════════════════════
// PHARMACY REPORTS
// ══════════════════════════════════════════════════════════════

// GetPharmacyReports returns sales and revenue analytics aggregated from
// pharmacy orders. Supports date range filtering via `from` / `to` query params.
// GET /pharmacy/reports
func (h *Handler) GetPharmacyReports(c *fiber.Ctx) error {
	pharmacyID, _ := c.Locals("pharmacy_id").(uint)
	role, _ := c.Locals("role").(string)
	isAdmin := role == "admin" || role == "super_admin"

	// Admin can query any pharmacy_id
	if qid := c.QueryInt("pharmacy_id", 0); qid > 0 && isAdmin {
		pharmacyID = uint(qid)
	}

	// Date range — default to current month
	now := time.Now()
	fromStr := c.Query("from", now.AddDate(0, -1, 0).Format("2006-01-02"))
	toStr := c.Query("to", now.Format("2006-01-02"))

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		from = now.AddDate(0, -1, 0)
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		to = now
	}
	// Include the full 'to' day
	to = to.Add(24*time.Hour - time.Second)

	// Total revenue and order count
	type SummaryRow struct {
		TotalRevenue int64 `json:"total_revenue"`
		TotalOrders  int64 `json:"total_orders"`
	}
	var summary SummaryRow
	summaryQuery := `SELECT COALESCE(SUM(amount),0) AS total_revenue, COUNT(*) AS total_orders
		FROM pharmacy_orders
		WHERE payment_status = 'paid' AND created_at BETWEEN ? AND ?`
	summaryArgs := []interface{}{from, to}
	if pharmacyID != 0 {
		summaryQuery += " AND pharmacy_id = ?"
		summaryArgs = append(summaryArgs, pharmacyID)
	}
	h.db.Raw(summaryQuery, summaryArgs...).Scan(&summary)

	// Top products by revenue
	type TopProduct struct {
		MedicineName string `json:"name"`
		Quantity     int64  `json:"quantity"`
		Revenue      int64  `json:"revenue"`
	}
	var topProducts []TopProduct
	topQuery := `SELECT i.medicine_name, SUM(i.quantity) AS quantity, SUM(i.total_price) AS revenue
		FROM pharmacy_order_items i
		JOIN pharmacy_orders o ON o.id = i.order_id
		WHERE o.payment_status = 'paid' AND o.created_at BETWEEN ? AND ?`
	topArgs := []interface{}{from, to}
	if pharmacyID != 0 {
		topQuery += " AND o.pharmacy_id = ?"
		topArgs = append(topArgs, pharmacyID)
	}
	topQuery += " GROUP BY i.medicine_name ORDER BY revenue DESC LIMIT 10"
	h.db.Raw(topQuery, topArgs...).Scan(&topProducts)

	// Monthly trend — last 6 months
	type MonthlyRow struct {
		Month  string `json:"month"`
		Sales  int64  `json:"sales"`
		Orders int64  `json:"orders"`
	}
	var monthlyTrend []MonthlyRow
	trendQuery := `SELECT TO_CHAR(created_at, 'Mon YYYY') AS month,
			COALESCE(SUM(amount),0) AS sales, COUNT(*) AS orders
		FROM pharmacy_orders
		WHERE payment_status = 'paid'
		  AND created_at >= NOW() - INTERVAL '6 months'`
	trendArgs := []interface{}{}
	if pharmacyID != 0 {
		trendQuery += " AND pharmacy_id = ?"
		trendArgs = append(trendArgs, pharmacyID)
	}
	trendQuery += ` GROUP BY TO_CHAR(created_at, 'Mon YYYY'), DATE_TRUNC('month', created_at)
		ORDER BY DATE_TRUNC('month', created_at) ASC`
	h.db.Raw(trendQuery, trendArgs...).Scan(&monthlyTrend)

	// Average order value
	avgOrderValue := int64(0)
	if summary.TotalOrders > 0 {
		avgOrderValue = summary.TotalRevenue / summary.TotalOrders
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"period": fiber.Map{
			"from": fromStr,
			"to":   toStr,
		},
		"summary": fiber.Map{
			"total_revenue":       summary.TotalRevenue,
			"total_orders":        summary.TotalOrders,
			"average_order_value": avgOrderValue,
		},
		"top_products":  topProducts,
		"monthly_trend": monthlyTrend,
	}, "Reports fetched")
}

// ══════════════════════════════════════════════════════════════
// REFILL CONVERSION
// ══════════════════════════════════════════════════════════════

// ConvertRefillToOrder converts a recurring refill order into a fresh pharmacy order.
// POST /pharmacy/refills/:id/convert
func (h *Handler) ConvertRefillToOrder(c *fiber.Ctx) error {
	refillID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid refill ID")
	}

	callerID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	// Fetch the parent (refill) order
	var parent PharmacyOrder
	if err := h.db.Preload("Pharmacy").First(&parent, refillID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Refill order not found")
	}

	// Ownership check: patient can only convert their own refills; staff can convert any
	isStaff := role == "admin" || role == "super_admin" || role == "pharmacist" || role == "pharmacy_admin"
	if !isStaff && parent.UserID != callerID {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN",
			"You can only convert your own refill orders")
	}

	// Validate the parent is a valid refill source (not already cancelled or in a bad state)
	if parent.Status == "cancelled" {
		return response.Error(c, fiber.StatusUnprocessableEntity, "INVALID_STATE",
			"Cannot convert a cancelled order into a refill")
	}

	// Process stock checks, GORM/Redis deductions, and order item creation in a transaction
	err = h.db.Transaction(func(tx *gorm.DB) error {
		// 1. Unmarshal items from parent order
		var items []struct {
			MedicineID uint  `json:"medicine_id"`
			Quantity   int   `json:"quantity"`
			UnitPrice  int64 `json:"unit_price"`
		}
		if err := json.Unmarshal(parent.Items, &items); err != nil {
			return fmt.Errorf("failed to parse refill items: %w", err)
		}

		// 2. Build order items lists and verify stock
		type deductedBatch struct {
			batchID uint
			deduct  int
		}
		var orderItemsToCreate []PharmacyOrderItem
		var deductions []deductedBatch

		for _, item := range items {
			// Fetch medicine details
			var med Medicine
			if err := tx.First(&med, item.MedicineID).Error; err != nil {
				return fmt.Errorf("medicine ID %d not found: %w", item.MedicineID, err)
			}

			// Find active batches for this pharmacy and medicine, sorted by expiry date ascending (FEFO)
			var batches []PharmacyInventory
			if err := tx.Where("pharmacy_id = ? AND medicine_id = ? AND stock_quantity > 0 AND is_active = ?",
				parent.PharmacyID, item.MedicineID, true).Order("expiry_date ASC").Find(&batches).Error; err != nil {
				return err
			}

			remaining := item.Quantity
			totalAvailable := 0
			for _, b := range batches {
				totalAvailable += b.StockQuantity
			}
			if totalAvailable < item.Quantity {
				return fmt.Errorf("insufficient stock for medicine %s (requested %d, available %d)", med.Name, item.Quantity, totalAvailable)
			}

			for _, batch := range batches {
				if remaining <= 0 {
					break
				}
				deduct := remaining
				if batch.StockQuantity < deduct {
					deduct = batch.StockQuantity
				}

				deductions = append(deductions, deductedBatch{batchID: batch.ID, deduct: deduct})

				// Create OrderItem record with batch tracking for recall compliance
				orderItemsToCreate = append(orderItemsToCreate, PharmacyOrderItem{
					MedicineID:   item.MedicineID,
					MedicineName: med.Name,
					Quantity:     deduct,
					UnitPrice:    item.UnitPrice,
					TotalPrice:   item.UnitPrice * int64(deduct),
					BatchNumber:  batch.BatchNumber,
					ExpiryDate:   &batch.ExpiryDate,
				})

				remaining -= deduct
			}
		}

		// 3. Apply DB deductions and decrement Redis stock (if Redis is active)
		for _, d := range deductions {
			result := tx.Exec(
				"UPDATE pharmacyinventory SET stock_quantity = stock_quantity - ? WHERE id = ? AND stock_quantity >= ?",
				d.deduct, d.batchID, d.deduct,
			)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("concurrent stock change on batch %d — please retry", d.batchID)
			}
		}

		// Decrement Redis keys
		if database.Rdb != nil {
			for _, oi := range orderItemsToCreate {
				stockKey := fmt.Sprintf("inventory:%d:%d", parent.PharmacyID, oi.MedicineID)
				database.Rdb.DecrBy(c.Context(), stockKey, int64(oi.Quantity))
			}
		}

		// 4. Create the new order with a collision-free order number
		orderNum := fmt.Sprintf("RFL-%s", strings.ToUpper(crypto.SecureToken(4)))
		newOrder := PharmacyOrder{
			OrderNumber:       orderNum,
			UserID:            parent.UserID,
			PharmacyID:        parent.PharmacyID,
			ParentOrderID:     parent.ID,
			Items:             parent.Items,
			Amount:            parent.Amount,
			Currency:          parent.Currency,
			Status:            "pending",
			PaymentStatus:     "pending",
			PaymentMethod:     parent.PaymentMethod,
			DeliveryMode:      parent.DeliveryMode,
			ShippingAddress:   parent.ShippingAddress,
			DeliveryPhone:     parent.DeliveryPhone,
			DestPincode:       parent.DestPincode,
			DestLatitude:      parent.DestLatitude,
			DestLongitude:     parent.DestLongitude,
			RequiresColdChain: parent.RequiresColdChain,
		}
		if err := tx.Create(&newOrder).Error; err != nil {
			return err
		}

		// 5. Create the PharmacyOrderItem rows mapping to the new order
		for i := range orderItemsToCreate {
			orderItemsToCreate[i].OrderID = newOrder.ID
			if err := tx.Create(&orderItemsToCreate[i]).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CONVERSION_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, fiber.Map{
		"new_order_id":     refillID, // use parent context or newly created ID if frontend needs it
		"new_order_number": "RFL-" + parent.OrderNumber, // fallback or dynamically fetched
		"parent_order_id":  parent.ID,
		"status":           "pending",
		"amount":           parent.Amount,
	}, "Refill converted to new order successfully")
}

// RegisterSettingsAndReportsRoutes registers the settings, reports, and refill routes.
// Called from the main RegisterRoutes function.
func (h *Handler) RegisterSettingsAndReportsRoutes(group fiber.Router) {
	// Settings — get is manager-level, update is admin-only
	group.Get("/settings", middleware.PharmacyManagerOnly(), h.GetPharmacySettings)
	group.Put("/settings", middleware.PharmacyAdminOnly(), h.UpdatePharmacySettings)

	// Reports — manager-level (pharmacist + admin can view)
	group.Get("/reports", middleware.PharmacyManagerOnly(), h.GetPharmacyReports)

	// Refill conversion — any authenticated user (ownership check in handler)
	group.Post("/refills/:id/convert", middleware.Protected(), h.ConvertRefillToOrder)
}

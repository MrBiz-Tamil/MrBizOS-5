package pharmacy

// catalog_publish.go — medicine publishing lifecycle + staff catalog views.
//
// New medicines are drafts. A pharmacist submits a draft for review; a
// pharmacy_admin publishes it (first publish is gated on available stock). The
// public store endpoints only ever show published + public-channel medicines —
// drafts, pending items, archived items, and pharmacy_only items stay hidden.

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"drayya-core/internal/domain/communication"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// publishApproverRoles are the roles that can approve/publish a medicine — also
// the recipients of the "awaiting approval" notification.
var publishApproverRoles = []string{"pharmacy_admin", "admin", "super_admin"}

// notifyAdminsOfPendingReview drops an in-app notification to every publish
// approver when a medicine enters the review queue. Best-effort: a failure here
// must never block the submit itself.
func (h *Handler) notifyAdminsOfPendingReview(ctx context.Context, submitterID uint, med *Medicine) {
	var admins []identity.User
	if err := h.db.Where("role IN ?", publishApproverRoles).Find(&admins).Error; err != nil || len(admins) == 0 {
		return
	}
	var submitter identity.User
	h.db.Select("full_name").First(&submitter, submitterID)
	who := submitter.FullName
	if who == "" {
		who = "A staff member"
	}
	notifs := make([]communication.Notification, 0, len(admins))
	for _, a := range admins {
		notifs = append(notifs, communication.Notification{
			UserID:  a.ID,
			Title:   "Medicine awaiting approval",
			Message: fmt.Sprintf("%s submitted “%s” (%s) for review. Approve it to publish on the store.", who, med.Name, med.Brand),
			Type:    "warning",
		})
	}
	if err := h.db.WithContext(ctx).Create(&notifs).Error; err != nil {
		log.Printf("[catalog] notify admins (pending review, medicine %d): %v", med.ID, err)
	}
}

// publicCatalogScope adds the public-store visibility filter: only medicines
// that are active, published, and on the public sales channel. Applied to every
// unauthenticated catalog endpoint so nothing leaks.
func publicCatalogScope(q *gorm.DB) *gorm.DB {
	return q.Where("is_active = ? AND catalog_status = ? AND sales_channel = ?",
		true, CatalogPublished, ChannelPublic)
}

// hasAvailableStock reports whether a medicine has at least one active inventory
// batch with positive quantity — the gate for a first publish.
func (h *Handler) hasAvailableStock(medicineID uint) bool {
	var n int64
	h.db.Model(&PharmacyInventory{}).
		Where("medicine_id = ? AND stock_quantity > 0 AND is_active = ?", medicineID, true).
		Count(&n)
	return n > 0
}

// SubmitMedicineForReview moves a draft (or archived) medicine to pending_review.
//
// POST /pharmacy/medicines/:id/submit   (PharmacyManagerOnly)
func (h *Handler) SubmitMedicineForReview(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid medicine ID")
	}
	var med Medicine
	if err := h.db.First(&med, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Medicine not found")
	}
	if med.CatalogStatus == CatalogPending {
		return response.Error(c, fiber.StatusConflict, "ALREADY_PENDING", "Medicine is already awaiting review")
	}
	if med.CatalogStatus == CatalogPublished {
		return response.Error(c, fiber.StatusConflict, "ALREADY_PUBLISHED", "Medicine is already live")
	}
	// WithContext so the tamper-evident audit log attributes the actor (compliance).
	if err := h.db.WithContext(c.UserContext()).Model(&Medicine{}).Where("id = ?", id).
		Update("catalog_status", CatalogPending).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	med.CatalogStatus = CatalogPending
	submitterID, _ := c.Locals("user_id").(uint)
	h.notifyAdminsOfPendingReview(c.UserContext(), submitterID, &med)
	return response.JSON(c, fiber.StatusOK, fiber.Map{"id": id, "catalog_status": CatalogPending}, "Submitted for review")
}

// PublishMedicine takes a pending medicine live. The first publish requires
// available stock; re-publishing an archived item that was live before does not.
//
// POST /pharmacy/medicines/:id/publish   (PharmacyAdminOnly)
func (h *Handler) PublishMedicine(c *fiber.Ctx) error {
	adminID, _ := c.Locals("user_id").(uint)
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid medicine ID")
	}
	var med Medicine
	if err := h.db.First(&med, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Medicine not found")
	}
	if med.CatalogStatus == CatalogPublished {
		return response.Error(c, fiber.StatusConflict, "ALREADY_PUBLISHED", "Medicine is already live")
	}

	// First-publish stock gate: a medicine going live for the very first time must
	// have inventory loaded. published_at is set the first time it ever publishes.
	firstPublish := med.PublishedAt == nil
	if firstPublish && !h.hasAvailableStock(med.ID) {
		return response.Error(c, fiber.StatusUnprocessableEntity, "NO_STOCK",
			"Load inventory stock before publishing this medicine for the first time")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"catalog_status": CatalogPublished,
		"published_by":   adminID,
	}
	if firstPublish {
		updates["published_at"] = now
	}
	// WithContext so the audit log records who took this medicine live + when.
	if err := h.db.WithContext(c.UserContext()).Model(&Medicine{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"id": id, "catalog_status": CatalogPublished}, "Medicine published")
}

// UnpublishMedicine delists a live medicine from the public store.
//
// POST /pharmacy/medicines/:id/unpublish   (PharmacyAdminOnly)
func (h *Handler) UnpublishMedicine(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid medicine ID")
	}
	// WithContext so the audit log records who delisted the medicine + when.
	if err := h.db.WithContext(c.UserContext()).Model(&Medicine{}).Where("id = ?", id).
		Update("catalog_status", CatalogArchived).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"id": id, "catalog_status": CatalogArchived}, "Medicine delisted")
}

// ListAdminCatalog returns the full catalog for staff management — every status
// and channel, with optional filters. Authenticated (PharmacyManagerOnly) so it
// never exposes drafts/pharmacy-only items to the public.
//
// GET /pharmacy/admin/catalog?status=&channel=&search=&limit=&offset=
func (h *Handler) ListAdminCatalog(c *fiber.Ctx) error {
	status := strings.TrimSpace(c.Query("status"))
	channel := strings.TrimSpace(c.Query("channel"))
	search := strings.TrimSpace(c.Query("search"))
	category := strings.TrimSpace(c.Query("category"))
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)
	if limit > 200 {
		limit = 200
	}

	q := h.db.Model(&Medicine{})
	if status != "" {
		q = q.Where("catalog_status = ?", status)
	}
	if channel != "" {
		q = q.Where("sales_channel = ?", channel)
	}
	if category != "" {
		q = q.Where("LOWER(category) = ?", strings.ToLower(category))
	}
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("name LIKE ? OR brand LIKE ? OR generic_name LIKE ?", like, like, like)
	}

	var meds []Medicine
	if err := q.Order("updated_at DESC").Limit(limit).Offset(offset).Find(&meds).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	// Pending-review count powers the admin queue badge.
	var pending int64
	h.db.Model(&Medicine{}).Where("catalog_status = ?", CatalogPending).Count(&pending)

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"medicines":     meds,
		"pending_count": pending,
	}, "Catalog fetched")
}

// AdminSearchMedicines is the counter/POS autocomplete. Unlike the public
// search it includes pharmacy_only items (those are sold here), but still only
// published, active ones — you can't sell a draft over the counter.
//
// GET /pharmacy/admin/catalog/search?q=&limit=   (PharmacyManagerOnly)
func (h *Handler) AdminSearchMedicines(c *fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 2 {
		return response.JSON(c, fiber.StatusOK, fiber.Map{"medicines": []interface{}{}, "brands": []interface{}{}}, "Query too short")
	}
	limit := c.QueryInt("limit", 10)
	if limit < 1 || limit > 25 {
		limit = 10
	}
	like := "%" + strings.ToLower(q) + "%"

	type MediResult struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		NameTa   string `json:"name_ta,omitempty"`
		Brand    string `json:"brand"`
		Category string `json:"category"`
		Form     string `json:"form,omitempty"`
		PackSize string `json:"pack_size,omitempty"`
		ImageURL string `json:"image_url,omitempty"`
		IsRx     bool   `json:"is_prescription_required"`
		Channel  string `json:"sales_channel"`
		PriceINR int64  `json:"price_inr"`
		InStock  bool   `json:"in_stock"`
	}
	priceExpr := "(prices->>'INR')::bigint"
	if h.db.Dialector.Name() == "sqlite" {
		priceExpr = "CAST(json_extract(prices, '$.INR') AS INTEGER)"
	}
	stockExpr := "EXISTS (SELECT 1 FROM pharmacyinventory pi WHERE pi.medicine_id = medicine.id AND pi.stock_quantity > 0 AND pi.is_active = true)"

	var meds []MediResult
	h.db.Model(&Medicine{}).
		Select("id, name, name_ta, brand, category, form, pack_size, image_url, is_prescription_required AS is_rx, sales_channel AS channel, "+
			"COALESCE("+priceExpr+", 0) AS price_inr, "+stockExpr+" AS in_stock").
		Where("is_active = ? AND catalog_status = ? AND ("+
			"LOWER(name) LIKE ? OR LOWER(name_ta) LIKE ? OR LOWER(brand) LIKE ? OR LOWER(generic_name) LIKE ? OR LOWER(synonyms) LIKE ?)",
			true, CatalogPublished, like, like, like, like, like).
		Order("is_inhouse_brand DESC, name ASC").Limit(limit).Scan(&meds)

	return response.JSON(c, fiber.StatusOK, fiber.Map{"medicines": meds}, "Search results")
}

package pharmacy

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"
)

// ProductBundle is a curated multi-product kit (e.g. "Diabetes Starter Kit",
// "Heart Health Pack"). Items is a JSON array of {product_id, product_type, qty}.
// BundlePricePaise is the *kit* price — typically lower than the sum of its
// items, that's the savings hook for the customer.
type ProductBundle struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	Name             string         `gorm:"not null" json:"name"`
	Slug             string         `gorm:"uniqueIndex;size:80" json:"slug"`
	ConditionTag     string         `gorm:"index;size:64" json:"condition_tag"` // diabetes | heart | immunity | …
	Description      string         `json:"description"`
	ImageURL         string         `json:"image_url"`
	Items            datatypes.JSON `gorm:"type:jsonb" json:"items"` // [{product_id, product_type, qty}]
	BundlePricePaise int64          `gorm:"not null" json:"bundle_price_paise"`
	IsActive         bool           `gorm:"default:true;index" json:"is_active"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ProductBundle) TableName() string { return "product_bundles" }

// BundleItem is the typed shape of an entry in ProductBundle.Items.
type BundleItem struct {
	ProductID   uint   `json:"product_id"`
	ProductType string `json:"product_type"` // medicine | wellness
	Qty         int    `json:"qty"`
}

// ListBundles returns active condition kits — optionally filtered by condition.
//   GET /pharmacy/bundles?condition=diabetes
func (h *Handler) ListBundles(c *fiber.Ctx) error {
	q := h.db.Where("is_active = ?", true)
	if cond := c.Query("condition"); cond != "" {
		q = q.Where("condition_tag = ?", cond)
	}
	var bundles []ProductBundle
	if err := q.Order("created_at DESC").Find(&bundles).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, bundles, "Bundles fetched")
}

// GetBundleBySlug returns a single kit (slug-based, for SEO-friendly URLs).
func (h *Handler) GetBundleBySlug(c *fiber.Ctx) error {
	var b ProductBundle
	if err := h.db.Where("slug = ? AND is_active = ?", c.Params("slug"), true).First(&b).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Bundle not found")
	}
	return response.JSON(c, fiber.StatusOK, b, "Bundle fetched")
}

// AdminCreateBundle is a thin admin CRUD for ops to author kits.
func (h *Handler) AdminCreateBundle(c *fiber.Ctx) error {
	var b ProductBundle
	if err := c.BodyParser(&b); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
	}
	if b.Slug == "" || b.Name == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "name and slug required")
	}
	if err := h.db.Create(&b).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, b, "Bundle created")
}

// Recommendations returns the medicines most-often co-ordered with the given
// medicine — a simple "patients with this also buy" feed.
//   GET /pharmacy/medicines/:id/recommendations
//
// Powered by pharmacy_order_items co-occurrence: for every order that
// contains medicine X, find the other medicines in that order and rank by
// count. Limited to top 4 to keep the API and the UI tight.
func (h *Handler) Recommendations(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid id")
	}

	type rec struct {
		MedicineID uint `gorm:"column:medicine_id" json:"id"`
		Count      int  `gorm:"column:count" json:"count"`
	}
	var recs []rec
	h.db.Raw(`
		SELECT b.medicine_id, COUNT(*) AS count
		FROM pharmacy_order_items a
		JOIN pharmacy_order_items b ON a.order_id = b.order_id
		WHERE a.medicine_id = ? AND b.medicine_id <> ?
		GROUP BY b.medicine_id
		ORDER BY count DESC
		LIMIT 4
	`, uint(id), uint(id)).Scan(&recs)

	if len(recs) == 0 {
		return response.JSON(c, fiber.StatusOK, []interface{}{}, "No recommendations yet")
	}

	// Hydrate medicine details for the IDs we got.
	ids := make([]uint, len(recs))
	for i, r := range recs {
		ids[i] = r.MedicineID
	}
	var meds []Medicine
	h.db.Where("id IN ? AND is_active = ?", ids, true).Find(&meds)

	return response.JSON(c, fiber.StatusOK, meds, "Recommendations fetched")
}

// RegisterBundleRoutes mounts the kit + recommendations endpoints.
func (h *Handler) RegisterBundleRoutes(pg fiber.Router) {
	pg.Get("/bundles", h.ListBundles)
	pg.Get("/bundles/:slug", h.GetBundleBySlug)
	pg.Post("/bundles", middleware.PharmacyManagerOnly(), h.AdminCreateBundle)
	pg.Get("/medicines/:id/recommendations", h.Recommendations)
}

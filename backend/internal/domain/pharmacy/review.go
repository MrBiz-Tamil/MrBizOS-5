package pharmacy

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"drayya-core/internal/database"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"
)

// ProductReview is a customer's rating of a product (Medicine or wellness item).
// We persist `verified_purchase = true` ONLY when the patient has a delivered
// PharmacyOrder containing this product. Aggregates are computed on read so
// there's no separate "sum/count" field to keep in sync.
type ProductReview struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	ProductID        uint           `gorm:"index;not null" json:"product_id"`
	ProductType      string         `gorm:"index;size:16;not null" json:"product_type"` // medicine | wellness
	PatientID        uint           `gorm:"index;not null" json:"patient_id"`
	Rating           int            `gorm:"not null;check:rating BETWEEN 1 AND 5" json:"rating"`
	Body             string         `gorm:"type:text" json:"body"`
	VerifiedPurchase bool           `gorm:"default:false" json:"verified_purchase"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ProductReview) TableName() string { return "product_reviews" }

// AggregateRating is the response shape for the rating summary endpoint.
type AggregateRating struct {
	AvgRating   float32        `json:"avg_rating"`
	ReviewCount int            `json:"review_count"`
	Distribution map[int]int   `json:"distribution"` // 1..5 → count
}

// hasDeliveredOrderFor returns true iff the patient has a delivered
// PharmacyOrder whose items JSON references this medicine id. For wellness
// products the corresponding check would live in the store domain — kept
// permissive here (verified=true only when explicitly confirmable).
func hasDeliveredOrderFor(patientID, medicineID uint, productType string) bool {
	if productType != "medicine" {
		return false
	}
	// Items is stored as JSON like [{"medicine_id":1,"quantity":2}, …].
	var count int64
	database.DB.
		Table("pharmacy_orders").
		Where("user_id = ? AND status = ?", patientID, "delivered").
		Where("items::text LIKE ?", fmt.Sprintf("%%\"medicine_id\":%d%%", medicineID)).
		Count(&count)
	return count > 0
}

// CreateReviewRequest is the inbound payload.
type CreateReviewRequest struct {
	ProductID   uint   `json:"product_id"`
	ProductType string `json:"product_type"` // medicine | wellness
	Rating      int    `json:"rating"`
	Body        string `json:"body"`
}

// CreateReview lets an authenticated patient submit a review. A patient can
// review a given product at most once (unique on patient_id+product); a
// duplicate becomes an update of the existing row.
func (h *Handler) CreateReview(c *fiber.Ctx) error {
	patientID, ok := c.Locals("user_id").(uint)
	if !ok || patientID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Auth required")
	}
	var req CreateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
	}
	if req.ProductID == 0 || (req.ProductType != "medicine" && req.ProductType != "wellness") {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "product_id and product_type=medicine|wellness required")
	}
	if req.Rating < 1 || req.Rating > 5 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "rating must be 1-5")
	}

	verified := hasDeliveredOrderFor(patientID, req.ProductID, req.ProductType)

	// Upsert by (patient, product, type) — one review per buyer per product.
	var rev ProductReview
	err := h.db.Where("patient_id = ? AND product_id = ? AND product_type = ?",
		patientID, req.ProductID, req.ProductType).First(&rev).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		rev = ProductReview{
			ProductID: req.ProductID, ProductType: req.ProductType,
			PatientID: patientID, Rating: req.Rating, Body: req.Body,
			VerifiedPurchase: verified,
		}
		if err := h.db.Create(&rev).Error; err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
		}
		return response.JSON(c, fiber.StatusCreated, rev, "Review submitted")
	}
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "QUERY_FAILED", err.Error())
	}
	rev.Rating = req.Rating
	rev.Body = req.Body
	rev.VerifiedPurchase = verified || rev.VerifiedPurchase
	if err := h.db.Save(&rev).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, rev, "Review updated")
}

// ListReviews returns recent reviews for a product (newest first).
func (h *Handler) ListReviews(c *fiber.Ctx) error {
	productType := c.Params("type")
	if productType != "medicine" && productType != "wellness" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "type must be medicine|wellness")
	}
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid id")
	}

	var reviews []ProductReview
	h.db.Where("product_id = ? AND product_type = ?", uint(id), productType).
		Order("verified_purchase DESC, created_at DESC").
		Limit(50).Find(&reviews)

	agg := computeAggregate(h.db, uint(id), productType)
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"aggregate": agg,
		"reviews":   reviews,
	}, "Reviews fetched")
}

// GetAggregate returns just the rating summary (used in product cards / lists).
func (h *Handler) GetAggregate(c *fiber.Ctx) error {
	productType := c.Params("type")
	if productType != "medicine" && productType != "wellness" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "type must be medicine|wellness")
	}
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid id")
	}
	return response.JSON(c, fiber.StatusOK, computeAggregate(h.db, uint(id), productType), "Aggregate rating")
}

// computeAggregate returns avg, count, and a 1-5 distribution for a product.
func computeAggregate(db *gorm.DB, productID uint, productType string) AggregateRating {
	type row struct {
		Rating int
		N      int
	}
	var rows []row
	db.Table("product_reviews").
		Select("rating, COUNT(*) AS n").
		Where("product_id = ? AND product_type = ? AND deleted_at IS NULL", productID, productType).
		Group("rating").Scan(&rows)

	dist := map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0}
	var sum, n int
	for _, r := range rows {
		dist[r.Rating] = r.N
		sum += r.Rating * r.N
		n += r.N
	}
	avg := float32(0)
	if n > 0 {
		avg = float32(sum) / float32(n)
	}
	return AggregateRating{AvgRating: avg, ReviewCount: n, Distribution: dist}
}

// RegisterReviewRoutes mounts /pharmacy/reviews/* on the existing pharmacy group.
// Call from RegisterRoutes after the medicine routes are registered.
func (h *Handler) RegisterReviewRoutes(pg fiber.Router) {
	pg.Post("/reviews", middleware.Protected(), h.CreateReview)
	pg.Get("/reviews/:type/:id", h.ListReviews)
	pg.Get("/reviews/:type/:id/aggregate", h.GetAggregate)
}

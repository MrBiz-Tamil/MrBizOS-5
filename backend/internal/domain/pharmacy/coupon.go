package pharmacy

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"
)

// Coupon is a promo code that discounts a cart total. Two flavours:
//   * type=percent → value is 0-100 (e.g. 10 = 10% off)
//   * type=fixed   → value is paise (e.g. 10000 = ₹100 off)
// MaxDiscountPaise caps the absolute discount for percent coupons (so a
// "50% off" doesn't refund the platform on a ₹1cr order).
type Coupon struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	Code             string         `gorm:"uniqueIndex;size:32;not null" json:"code"`
	Description      string         `json:"description"`
	Type             string         `gorm:"size:16;not null" json:"type"` // percent | fixed
	Value            int64          `gorm:"not null" json:"value"`        // % (0-100) or paise
	MinOrderPaise    int64          `json:"min_order_paise"`
	MaxDiscountPaise int64          `json:"max_discount_paise"` // 0 = no cap
	MaxUses          int            `json:"max_uses"`           // 0 = unlimited
	UsedCount        int            `gorm:"default:0" json:"used_count"`
	IsActive         bool           `gorm:"default:true;index" json:"is_active"`
	ExpiresAt        *time.Time     `gorm:"index" json:"expires_at"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Coupon) TableName() string { return "coupons" }

// ValidateCouponRequest is the inbound payload for /coupons/validate.
type ValidateCouponRequest struct {
	Code           string `json:"code"`
	CartTotalPaise int64  `json:"cart_total_paise"`
}

// CouponValidation is the response shape — never mutates state, just answers
// "if this user applied this code right now to this cart, what happens?".
type CouponValidation struct {
	Valid         bool   `json:"valid"`
	Code          string `json:"code"`
	DiscountPaise int64  `json:"discount_paise"`
	NewTotalPaise int64  `json:"new_total_paise"`
	Description   string `json:"description"`
	Reason        string `json:"reason,omitempty"` // why invalid
}

// computeDiscount derives the discount amount in paise, applying the cap.
func (c *Coupon) computeDiscount(cartTotalPaise int64) int64 {
	var d int64
	if c.Type == "percent" {
		d = cartTotalPaise * c.Value / 100
	} else {
		d = c.Value
	}
	if c.MaxDiscountPaise > 0 && d > c.MaxDiscountPaise {
		d = c.MaxDiscountPaise
	}
	if d > cartTotalPaise {
		d = cartTotalPaise
	}
	return d
}

// ValidateCoupon checks a code against a cart total — pure read; never marks
// a coupon as used (that happens at checkout-complete).
func (h *Handler) ValidateCoupon(c *fiber.Ctx) error {
	var req ValidateCouponRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
	}
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code == "" || req.CartTotalPaise <= 0 {
		return response.JSON(c, fiber.StatusOK, CouponValidation{
			Valid: false, Code: code, Reason: "Enter a coupon code",
		}, "")
	}

	var coupon Coupon
	if err := h.db.Where("code = ? AND is_active = ?", code, true).First(&coupon).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.JSON(c, fiber.StatusOK, CouponValidation{
				Valid: false, Code: code, Reason: "Coupon not found",
			}, "")
		}
		return response.Error(c, fiber.StatusInternalServerError, "QUERY_FAILED", err.Error())
	}

	if coupon.ExpiresAt != nil && coupon.ExpiresAt.Before(time.Now()) {
		return response.JSON(c, fiber.StatusOK, CouponValidation{
			Valid: false, Code: code, Reason: "Coupon expired",
		}, "")
	}
	if coupon.MaxUses > 0 && coupon.UsedCount >= coupon.MaxUses {
		return response.JSON(c, fiber.StatusOK, CouponValidation{
			Valid: false, Code: code, Reason: "Coupon usage limit reached",
		}, "")
	}
	if coupon.MinOrderPaise > 0 && req.CartTotalPaise < coupon.MinOrderPaise {
		return response.JSON(c, fiber.StatusOK, CouponValidation{
			Valid: false, Code: code,
			Reason: "Minimum order not met",
		}, "")
	}

	d := coupon.computeDiscount(req.CartTotalPaise)
	return response.JSON(c, fiber.StatusOK, CouponValidation{
		Valid:         true,
		Code:          code,
		DiscountPaise: d,
		NewTotalPaise: req.CartTotalPaise - d,
		Description:   coupon.Description,
	}, "Coupon applied")
}

// RegisterCouponRoutes mounts the validate endpoint on the pharmacy group.
func (h *Handler) RegisterCouponRoutes(pg fiber.Router) {
	pg.Post("/coupons/validate", middleware.Protected(), h.ValidateCoupon)
}

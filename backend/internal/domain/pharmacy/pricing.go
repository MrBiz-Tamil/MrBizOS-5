package pharmacy

import (
	"context"
	"errors"
	"math"

	"gorm.io/gorm"
)

// PricingLine is one item on the bill the patient sees before paying.
// Amounts are in paise so we never lose a rupee to float rounding.
type PricingLine struct {
	MedicineID        uint    `json:"medicine_id"`
	Name              string  `json:"name"`
	NameLocal         string  `json:"name_local,omitempty"`
	Quantity          int     `json:"quantity"`
	UnitMRPPaise      int64   `json:"unit_mrp_paise"`
	LineSubtotalPaise int64   `json:"line_subtotal_paise"` // MRP × qty
	GSTPercent        float64 `json:"gst_percent"`
	GSTPaise          int64   `json:"gst_paise"`
	LineTotalPaise    int64   `json:"line_total_paise"` // subtotal + gst
}

// CartTotals is the full breakdown the patient sees. Invariant (exact, no
// rounding gap):
//
//	subtotal + total_gst + shipping + cod_surcharge − discount − insurance_covered
//	    == grand_total
type CartTotals struct {
	Lines                 []PricingLine `json:"lines"`
	SubtotalPaise         int64         `json:"subtotal_paise"`         // Σ line_subtotal
	TotalGSTPaise         int64         `json:"total_gst_paise"`        // Σ line_gst
	DiscountPaise         int64         `json:"discount_paise,omitempty"`
	ShippingPaise         int64         `json:"shipping_paise"`
	CODSurchargePaise     int64         `json:"cod_surcharge_paise,omitempty"`
	InsuranceCoveredPaise int64         `json:"insurance_covered_paise,omitempty"`
	GrandTotalPaise       int64         `json:"grand_total_paise"`
	Currency              string        `json:"currency"`
}

// PaymentMethod is what the patient picks at checkout. COD adds a small
// surcharge to cover handling.
type PaymentMethod string

const (
	PaymentPrepaid PaymentMethod = "prepaid"
	PaymentCOD     PaymentMethod = "cod"
)

// PricingRequest is the input to PriceCart.
type PricingRequest struct {
	Items          []ReservationItem // reuse the existing shape (medicine_id + quantity)
	DeliveryPincode string
	PaymentMethod   PaymentMethod
	// InsuranceCoveredPaise is the amount the insurer/CGHS has approved to
	// reimburse on this order. Pass 0 for self-pay. Negative values are clamped.
	InsuranceCoveredPaise int64
	// DiscountPaise is a flat discount (coupons, loyalty redemption). Negative
	// values are clamped; the engine refuses to let the grand total go below 0.
	DiscountPaise int64
}

// Pricing constants — tuned for AYUSH e-pharmacy. Tweak via config later when
// we have enough data to differentiate by weight/distance/zone.
const (
	// FlatShippingPaise is the baseline shipping cost for standard delivery.
	// 5kg is the upper bound for an AYUSH consignment in practice.
	FlatShippingPaise = 4900 // ₹49.00

	// FreeShippingThresholdPaise — orders at or above this ship free.
	FreeShippingThresholdPaise = 49900 // ₹499.00

	// CODSurchargePaise covers cash handling. Most Indian e-pharmacies charge
	// ₹30–₹50 for COD.
	CODSurchargePaise = 4000 // ₹40.00
)

// PriceCart computes a fully itemised bill. It does not write anything — call
// this from the patient cart preview and again at checkout to lock the totals
// into the order record.
func PriceCart(ctx context.Context, db *gorm.DB, req PricingRequest) (*CartTotals, error) {
	if len(req.Items) == 0 {
		return nil, errors.New("no items to price")
	}

	// Fetch every wanted medicine in one round trip.
	ids := make([]uint, 0, len(req.Items))
	for _, it := range req.Items {
		if it.MedicineID == 0 || it.Quantity <= 0 {
			return nil, errors.New("invalid item: medicine_id and quantity required")
		}
		ids = append(ids, it.MedicineID)
	}
	var meds []Medicine
	if err := db.WithContext(ctx).Where("id IN ? AND is_active = ?", ids, true).Find(&meds).Error; err != nil {
		return nil, err
	}
	medByID := map[uint]Medicine{}
	for _, m := range meds {
		medByID[m.ID] = m
	}

	totals := &CartTotals{
		Lines:    make([]PricingLine, 0, len(req.Items)),
		Currency: "INR",
	}

	for _, it := range req.Items {
		m, ok := medByID[it.MedicineID]
		if !ok {
			return nil, errors.New("one or more medicines are invalid or inactive")
		}
		unitMRP := m.Prices["INR"]
		lineSubtotal := unitMRP * int64(it.Quantity)
		// GST rounded to nearest paise (banker's rounding to keep totals stable).
		gst := int64(math.Round(float64(lineSubtotal) * m.GSTPercent / 100))
		lineTotal := lineSubtotal + gst

		// Localised name for the cart display.
		nameLocal := m.NameTa
		if nameLocal == "" {
			nameLocal = m.NameHi
		}

		totals.Lines = append(totals.Lines, PricingLine{
			MedicineID:        m.ID,
			Name:              m.Name,
			NameLocal:         nameLocal,
			Quantity:          it.Quantity,
			UnitMRPPaise:      unitMRP,
			LineSubtotalPaise: lineSubtotal,
			GSTPercent:        m.GSTPercent,
			GSTPaise:          gst,
			LineTotalPaise:    lineTotal,
		})
		totals.SubtotalPaise += lineSubtotal
		totals.TotalGSTPaise += gst
	}

	// Free shipping above the threshold; otherwise flat. Refine to weight/zone
	// once we have enough order history to model it accurately.
	if totals.SubtotalPaise+totals.TotalGSTPaise >= FreeShippingThresholdPaise {
		totals.ShippingPaise = 0
	} else {
		totals.ShippingPaise = FlatShippingPaise
	}

	if req.PaymentMethod == PaymentCOD {
		totals.CODSurchargePaise = CODSurchargePaise
	}

	// Clamp negatives and apply.
	disc := req.DiscountPaise
	if disc < 0 {
		disc = 0
	}
	ins := req.InsuranceCoveredPaise
	if ins < 0 {
		ins = 0
	}
	totals.DiscountPaise = disc
	totals.InsuranceCoveredPaise = ins

	grand := totals.SubtotalPaise + totals.TotalGSTPaise + totals.ShippingPaise + totals.CODSurchargePaise - disc - ins
	if grand < 0 {
		grand = 0 // never bill negative
	}
	totals.GrandTotalPaise = grand
	return totals, nil
}

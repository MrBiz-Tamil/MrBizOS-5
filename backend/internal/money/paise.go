// Package money is the canonical representation of currency amounts in this
// codebase. All money in the database is stored as int64 PAISE (1 INR = 100 paise).
//
// The Paise type is a distinct named type — passing a raw int64 won't compile
// where a Paise is required, making "forgot to convert rupees to paise" a
// build-time error rather than a runtime money corruption bug.
//
// Convention rules — these are not optional:
//   1. All DB money columns are BIGINT, named with `_paise` suffix.
//   2. All API JSON fields ending in money end with `_paise`.
//   3. Frontend never displays a raw money number — must go through formatPrice().
//   4. Conversions use math.Round, never raw int64 cast (float precision).
//   5. BeforeSave hooks reject negative or impossibly-large values.
//
// See MONEY_PLAN.md and MONEY_FIELDS.md for the full policy.
package money

import (
	"fmt"
	"math"
)

// Paise is a money amount denominated in Indian paise (1/100 of INR).
// Using a distinct type prevents accidental mixing with raw int64 amounts
// that may be in rupees, percentages, or some other unit.
//
// Deprecated: Use money.Money instead for multi-currency support.
type Paise int64

// ToMoney converts legacy Paise into the new multi-currency Money struct (defaulting to INR).
func (p Paise) ToMoney() Money {
	return Money{
		Amount:   int64(p),
		Currency: "INR",
	}
}

// Common bounds for sanity checks. A single money field of > 1 crore is almost
// certainly a unit-conversion bug (treating rupees as paise).
const (
	MaxReasonablePaise Paise = 100_00_00_000 // ₹1,00,00,000 (1 crore rupees) per field
)

// FromRupees converts a float rupee value to Paise with proper rounding.
// Uses math.Round to avoid floating-point representation errors:
//   FromRupees(499.99) → 49999 (not 49998)
func FromRupees(rupees float64) Paise {
	return Paise(math.Round(rupees * 100))
}

// FromRupeesInt converts an int rupee value to Paise. Safer than FromRupees
// when you know the input is whole rupees.
func FromRupeesInt(rupees int64) Paise {
	return Paise(rupees * 100)
}

// Rupees returns the float64 rupee equivalent (for display or arithmetic).
// Do NOT use this value when writing back to the DB — convert back via FromRupees.
func (p Paise) Rupees() float64 {
	return float64(p) / 100.0
}

// String formats a paise amount as a human-readable INR string.
//   Paise(50000).String() → "₹500.00"
//   Paise(499).String()   → "₹4.99"
func (p Paise) String() string {
	negative := false
	val := int64(p)
	if val < 0 {
		negative = true
		val = -val
	}
	rupees := val / 100
	paise := val % 100
	sign := ""
	if negative {
		sign = "-"
	}
	return fmt.Sprintf("%s₹%d.%02d", sign, rupees, paise)
}

// IsZero reports whether the amount is exactly zero.
func (p Paise) IsZero() bool { return p == 0 }

// IsValid reports whether the amount is non-negative and within sanity bounds.
// Use in BeforeSave hooks:
//   if !m.UnitPricePaise.IsValid() { return errors.New("unit_price out of range") }
func (p Paise) IsValid() bool {
	return p >= 0 && p <= MaxReasonablePaise
}

// Add returns p + other.
func (p Paise) Add(other Paise) Paise { return p + other }

// Sub returns p - other.
func (p Paise) Sub(other Paise) Paise { return p - other }

// Mul returns p multiplied by a scalar (e.g. quantity).
func (p Paise) Mul(n int64) Paise { return Paise(int64(p) * n) }

// PercentOf returns the given percent of p, rounded to nearest paise.
//   Paise(50000).PercentOf(15) → 7500 paise (₹75)
func (p Paise) PercentOf(percent float64) Paise {
	return Paise(math.Round(float64(p) * percent / 100.0))
}

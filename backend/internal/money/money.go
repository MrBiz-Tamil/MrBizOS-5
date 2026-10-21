package money

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

// Money represents a monetary value with its associated currency.
// It replaces the legacy Paise (int64) type to support multi-region global pricing.
type Money struct {
	Amount   int64  `json:"amount"`   // Minor units (cents, paise, etc.)
	Currency string `json:"currency"` // ISO 4217 code e.g., "USD", "INR", "AED"
}

// RegionalPricing is a JSON map used in GORM models to store prices for multiple global regions.
// Example: {"INR": 50000, "USD": 1500, "AED": 6000}
type RegionalPricing map[string]int64

// Value implements driver.Valuer to store as JSON
func (p RegionalPricing) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}
	return json.Marshal(p)
}

// Scan implements sql.Scanner to read JSON from multiple driver representations:
//   - Postgres jsonb            → []byte
//   - SQLite text (no jsonb)    → string
//   - Some MySQL drivers        → []byte
// Previously this only accepted []byte which made queries against jsonb columns
// blow up on SQLite (dev env), so the doctor-pricing page returned an empty list.
func (p *RegionalPricing) Scan(value interface{}) error {
	if value == nil {
		*p = nil
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return errors.New("RegionalPricing.Scan: unsupported value type")
	}
	if len(raw) == 0 {
		*p = nil
		return nil
	}
	return json.Unmarshal(raw, p)
}

// NewMoney creates a new Money instance.
func NewMoney(amount int64, currency string) Money {
	if currency == "" {
		currency = "USD" // Global fallback
	}
	return Money{
		Amount:   amount,
		Currency: currency,
	}
}

// IsZero reports whether the amount is exactly zero.
func (m Money) IsZero() bool {
	return m.Amount == 0
}

// IsValid reports whether the amount is non-negative.
func (m Money) IsValid() bool {
	return m.Amount >= 0
}

// String formats the money amount for display or logging.
func (m Money) String() string {
	val := float64(m.Amount) / 100.0
	return fmt.Sprintf("%s %.2f", m.Currency, val)
}

// CheckCurrency panics if the currencies do not match.
// This guarantees we never accidentally add USD to INR.
func (m Money) CheckCurrency(other Money) {
	if m.Currency != other.Currency {
		panic(fmt.Sprintf("currency mismatch: cannot operate on %s and %s", m.Currency, other.Currency))
	}
}

// Add returns m + other. Panics on currency mismatch.
func (m Money) Add(other Money) Money {
	if !m.IsZero() && !other.IsZero() {
		m.CheckCurrency(other)
	}
	// Inherit currency if one is zero
	curr := m.Currency
	if m.IsZero() && !other.IsZero() {
		curr = other.Currency
	}
	return Money{
		Amount:   m.Amount + other.Amount,
		Currency: curr,
	}
}

// Sub returns m - other. Panics on currency mismatch.
func (m Money) Sub(other Money) Money {
	if !m.IsZero() && !other.IsZero() {
		m.CheckCurrency(other)
	}
	curr := m.Currency
	if m.IsZero() && !other.IsZero() {
		curr = other.Currency
	}
	return Money{
		Amount:   m.Amount - other.Amount,
		Currency: curr,
	}
}

// Mul returns m multiplied by a scalar (e.g. quantity).
func (m Money) Mul(n int64) Money {
	return Money{
		Amount:   m.Amount * n,
		Currency: m.Currency,
	}
}

// PercentOf returns the given percent of m, rounded to nearest minor unit.
func (m Money) PercentOf(percent float64) Money {
	return Money{
		Amount:   int64(math.Round(float64(m.Amount) * percent / 100.0)),
		Currency: m.Currency,
	}
}

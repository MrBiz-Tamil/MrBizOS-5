// Package fx converts our canonical INR (paise) prices into the presentment
// currency a global customer is charged in via Stripe.
//
// IMPORTANT: this is presentment only. The ledger always books the INR price;
// the foreign amount here is just what the customer's card is charged. Stripe
// performs the real settlement conversion, and any FX spread reconciles against
// the bank statement — never against the ledger.
//
// Rates are configurable via env (FX_RATE_USD, FX_RATE_EUR, …) and default to
// conservative placeholders. Wire a real rate feed before going live.
package fx

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

// defaultRates: INR per 1 unit of the foreign currency (e.g. 1 USD = ₹83).
var defaultRates = map[string]float64{
	"USD": 83.0,
	"EUR": 90.0,
	"GBP": 105.0,
	"AED": 22.6,
	"SGD": 62.0,
	"AUD": 55.0,
	"CAD": 61.0,
}

// zeroDecimal currencies have no minor unit (amount is already the integer
// charge); everything else uses 2 decimals (minor = 1/100).
var zeroDecimal = map[string]bool{"JPY": true, "KRW": true}

// Supported reports whether we can present in the given currency.
func Supported(currency string) bool {
	_, ok := defaultRates[strings.ToUpper(currency)]
	return ok
}

// RateINRPerUnit returns the INR-per-unit rate for a currency: env override
// (FX_RATE_<CCY>) first, else the built-in default.
func RateINRPerUnit(currency string) (float64, error) {
	ccy := strings.ToUpper(currency)
	if v := os.Getenv("FX_RATE_" + ccy); v != "" {
		if r, err := strconv.ParseFloat(v, 64); err == nil && r > 0 {
			return r, nil
		}
	}
	r, ok := defaultRates[ccy]
	if !ok || r <= 0 {
		return 0, fmt.Errorf("unsupported presentment currency: %s", ccy)
	}
	return r, nil
}

// ToPresentment converts an INR amount in paise to the minor units of the
// target currency, returning the charge amount and the rate used. INR stays
// INR (no conversion). Unknown currencies error rather than guess.
func ToPresentment(inrPaise int64, currency string) (minorUnits int64, rateINRPerUnit float64, err error) {
	ccy := strings.ToUpper(currency)
	if ccy == "" || ccy == "INR" {
		return inrPaise, 1, nil
	}
	rate, err := RateINRPerUnit(ccy)
	if err != nil {
		return 0, 0, err
	}
	// foreignMinor = round(inrPaise / rate); for zero-decimal currencies the
	// charge is in whole units, so divide the rupee amount by the rate.
	if zeroDecimal[ccy] {
		whole := math.Round((float64(inrPaise) / 100.0) / rate)
		return int64(whole), rate, nil
	}
	return int64(math.Round(float64(inrPaise) / rate)), rate, nil
}

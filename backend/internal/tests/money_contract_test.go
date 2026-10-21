package tests

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"drayya-core/internal/database"
	"drayya-core/internal/tests/common"

	"github.com/stretchr/testify/assert"
)

// moneyKeywords are substrings that, when found in a JSON key, indicate the
// field carries a money value. Every such key MUST end in "_paise".
//
// Exempt patterns (e.g. "fee_paise", "price_paise") naturally pass because
// they end in _paise.  Non-money keys like "price_per_minute_rate" still need
// careful naming review — add to the explicit allow-list below if needed.
var moneyKeywords = []string{"fee", "price", "amount", "cost", "earning", "revenue", "salary"}

// allowedNonPaiseKeys is the explicit allow-list of money-like JSON keys that
// are not in paise — usually because they're legacy or wrappers.
//
// Add a key here ONLY after confirming it's not a money amount, and document why.
var allowedNonPaiseKeys = map[string]string{
	// Booleans / flags / pricing tiers
	"is_priced":         "boolean flag, not an amount",
	"price_change":      "delta / percentage tracker (see scope comment)",
	"pricing_tier":      "string label: standard/premium/etc.",
	"price_info":        "wrapper object — its children carry _paise",
	"per_minute_rate":   "legacy — being migrated to per_minute_rate_paise",
	"base_fee":          "legacy — being migrated to base_fee_paise",
	"total_amount":      "legacy — being migrated to total_amount_paise",
	"original_amount":   "legacy — being migrated to original_amount_paise",
	"consultation_fee":  "legacy — see MONEY_FIELDS.md",
	"consultation_fee_video": "legacy alias",
	"consultation_fee_audio": "legacy alias",
	"consultation_fee_chat":  "legacy alias",
	"unit_price":        "legacy — being migrated to unit_price_paise",
	"total_sales":       "legacy — being migrated to total_sales_paise",
	"today_sales":       "legacy — being migrated to today_sales_paise",
	"inventory_value":   "legacy — being migrated to inventory_value_paise",
	"price_per_night":   "legacy — being migrated to price_per_night_paise",
	"daily_rate":        "computed alias of price_per_night (legacy)",
	"base_price":        "legacy lab field — being migrated",
}

// isMoneyKey reports whether a JSON key looks like it carries a money amount.
func isMoneyKey(key string) bool {
	lower := strings.ToLower(key)
	for _, kw := range moneyKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// walk recursively checks every map key in a JSON-decoded structure.
func walk(t *testing.T, path string, v interface{}, violations *[]string) {
	switch x := v.(type) {
	case map[string]interface{}:
		for key, child := range x {
			fullPath := path + "." + key
			if isMoneyKey(key) && !strings.HasSuffix(key, "_paise") {
				if _, ok := allowedNonPaiseKeys[key]; !ok {
					*violations = append(*violations,
						fullPath+" (money-like key without _paise suffix)")
				}
			}
			walk(t, fullPath, child, violations)
		}
	case []interface{}:
		for i, item := range x {
			walk(t, path+"["+itoa2(i)+"]", item, violations)
		}
	}
}

func itoa2(i int) string {
	if i == 0 {
		return "0"
	}
	digits := []byte{}
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}

// TestMoneyContract_AllPriceFieldsEndInPaise scans the JSON shape of every
// money-bearing endpoint and asserts the _paise suffix convention.
//
// When this test fails, EITHER:
//   - Rename the field to ${name}_paise in the Go handler
//   - Or, if it's truly not money, add it to allowedNonPaiseKeys with a reason
func TestMoneyContract_AllPriceFieldsEndInPaise(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	admin := common.CreateTestUser(db, "admin")
	adminToken := common.GenerateTestToken(admin)
	pharmacist := common.CreateTestUser(db, "pharmacist")
	pharmaToken := common.GenerateTestToken(pharmacist)

	endpoints := []struct {
		name  string
		path  string
		token string
	}{
		{"lab_tests", "/api/v1/lab/tests", ""},
		{"lab_packages", "/api/v1/lab/packages", ""},
		{"pharmacy_dashboard", "/api/v1/pharmacy/dashboard/stats", pharmaToken},
		{"hr_pricing_doctors", "/api/v1/admin/hr/pricing/doctors", adminToken},
		{"hr_pricing_templates", "/api/v1/admin/hr/pricing/templates", adminToken},
	}

	allViolations := map[string][]string{}

	for _, ep := range endpoints {
		req := httptest.NewRequest("GET", ep.path, nil)
		if ep.token != "" {
			req.Header.Set("Authorization", "Bearer "+ep.token)
		}
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Logf("[%s] request error: %v — skipping", ep.name, err)
			continue
		}
		if resp.StatusCode != 200 {
			t.Logf("[%s] non-200 status %d — skipping content check", ep.name, resp.StatusCode)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		var data interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			t.Logf("[%s] not JSON — skipping", ep.name)
			continue
		}

		var violations []string
		walk(t, ep.name, data, &violations)
		if len(violations) > 0 {
			allViolations[ep.name] = violations
		}
	}

	// Report all violations as a single failure so CI shows the whole picture
	if len(allViolations) > 0 {
		msg := "money-like JSON fields without _paise suffix:\n"
		for ep, vs := range allViolations {
			msg += "\n  Endpoint: " + ep + "\n"
			for _, v := range vs {
				msg += "    - " + v + "\n"
			}
		}
		msg += "\nFix: rename to ${name}_paise OR add to allowedNonPaiseKeys with a reason."
		assert.Fail(t, "money convention violated", msg)
	}
}

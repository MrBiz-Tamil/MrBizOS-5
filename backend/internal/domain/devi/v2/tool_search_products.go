package v2

import (
	"context"
	"errors"
	"strings"

	"drayya-core/internal/database"
)

// searchProductsTool finds Siddha/Ayurveda/organic products available in the
// Dr.Ayya pharmacy. Read-only, guest-allowed.
//
// We filter to active, OTC products by default — anything requiring a
// prescription should not be surfaced to a guest browsing for "what's good
// for fever". Patients with a recent prescription can see RX items via
// get_my_prescriptions instead.
var searchProductsTool = Tool{
	Name: "search_products",
	Description: "Search the Dr.Ayya pharmacy catalog for AYUSH medicines, " +
		"Siddha/Ayurveda formulations, and organic wellness products. Use this " +
		"when the user asks 'what's good for X', 'do you have Y', or wants to " +
		"browse products. Returns up to 5 matching OTC products with name, " +
		"brand, category, and price.",
	Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The product name, symptom, or category. Natural language is fine.",
			},
			"category": map[string]any{
				"type":        "string",
				"description": "Optional category filter (e.g. 'syrup', 'tablet', 'oil').",
			},
		},
		"required": []string{"query"},
	},
	Allowed:       []PersonaRole{RoleGuest, RolePatient},
	Confirm:       false,
	MaxPerSession: 10,
	MaxPerMinute:  5,
	Execute:       executeSearchProducts,
}

type productResult struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Brand      string `json:"brand,omitempty"`
	Generic    string `json:"generic_name,omitempty"`
	Category   string `json:"category,omitempty"`
	PriceINR   string `json:"price_inr,omitempty"` // formatted "₹120" string
	IsRX       bool   `json:"prescription_required"`
}

func executeSearchProducts(ctx context.Context, args map[string]any) (any, error) {
	if database.DB == nil {
		return nil, errors.New("database unavailable")
	}
	query, _ := args["query"].(string)
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return map[string]any{"products": []any{}, "total_found": 0}, nil
	}
	category, _ := args["category"].(string)

	type row struct {
		ID                     uint
		Name                   string
		Brand                  string
		GenericName            string
		Category               string
		Prices                 string // raw jsonb text — parsed below
		IsPrescriptionRequired bool
	}
	var rows []row
	q := database.DB.Table("medicine").
		Select(`id, name, brand, generic_name, category, CAST(prices AS TEXT) AS prices,
		         is_prescription_required`).
		Where(`is_active = ? AND deleted_at IS NULL`, true).
		Where(`is_prescription_required = ?`, false). // OTC only for chat-driven search
		Where(`LOWER(name) LIKE ? OR LOWER(brand) LIKE ? OR LOWER(generic_name) LIKE ? OR LOWER(synonyms) LIKE ?`,
			"%"+query+"%", "%"+query+"%", "%"+query+"%", "%"+query+"%")

	if category != "" {
		q = q.Where("LOWER(category) = ?", strings.ToLower(category))
	}
	q.Order("name ASC").Limit(5).Scan(&rows)

	out := make([]productResult, 0, len(rows))
	for _, r := range rows {
		out = append(out, productResult{
			ID:       r.ID,
			Name:     r.Name,
			Brand:    r.Brand,
			Generic:  r.GenericName,
			Category: r.Category,
			PriceINR: extractINRFromJSONB(r.Prices),
			IsRX:     r.IsPrescriptionRequired,
		})
	}
	return map[string]any{
		"products":    out,
		"total_found": len(out),
	}, nil
}

// extractINRFromJSONB parses {"INR":50000} → "₹500.00" (paise → rupees).
// The Prices column is a RegionalPricing jsonb. We don't import money.Money
// because chasing that abstraction here would add an unused dep. A 3-line
// JSON peek is cheaper.
func extractINRFromJSONB(jsonStr string) string {
	if jsonStr == "" {
		return ""
	}
	// Lightweight extraction — look for "INR": followed by digits.
	idx := strings.Index(jsonStr, `"INR":`)
	if idx < 0 {
		return ""
	}
	tail := jsonStr[idx+len(`"INR":`):]
	end := strings.IndexAny(tail, ",}")
	if end < 0 {
		return ""
	}
	paiseStr := strings.TrimSpace(tail[:end])
	// Convert paise to rupees and format as ₹X.XX
	// Use a tiny custom parse instead of strconv to keep this scoped.
	if paiseStr == "" {
		return ""
	}
	var paise int64
	for _, c := range paiseStr {
		if c < '0' || c > '9' {
			return ""
		}
		paise = paise*10 + int64(c-'0')
	}
	rupees := paise / 100
	cents := paise % 100
	out := "₹"
	out += formatInt(rupees)
	if cents > 0 {
		out += "."
		if cents < 10 {
			out += "0"
		}
		out += formatInt(cents)
	}
	return out
}

func formatInt(n int64) string {
	if n == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}

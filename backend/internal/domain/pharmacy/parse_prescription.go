package pharmacy

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"drayya-core/internal/middleware"
	"drayya-core/internal/services/ocr"
	"drayya-core/internal/utils/response"
)

// ParsedMedicine is one line lifted from an OCR'd prescription.
//   Name        — the cleaned medicine name as written (e.g. "Metformin 500mg")
//   Dosage      — frequency string (e.g. "1 tab twice daily")
//   QtyDays     — duration in days when expressible (e.g. 30)
//   MatchedID   — pharmacy.Medicine.ID we found a match for (0 = no match)
//   MatchedName — the matched product's canonical name for display
type ParsedMedicine struct {
	Name        string `json:"name"`
	Dosage      string `json:"dosage"`
	QtyDays     int    `json:"qty_days"`
	MatchedID   uint   `json:"matched_id"`
	MatchedName string `json:"matched_name"`
}

// linePattern matches one Rx line. Examples it handles:
//   "1. Metformin 500mg — 1 tab twice daily — 30 days"
//   "2) Atorvastatin 10mg - 1 tab at bedtime - 30 days"
//   "- Amlodipine 5mg — 1 tab daily — 30 days"
// Capture groups:
//   1: drug body (drug name + strength)
//   2: dosage instruction
//   3: duration (days)
var linePattern = regexp.MustCompile(
	`(?i)^\s*(?:\d+\s*[.\)]\s*|-\s*)?` + // optional list marker
		`([A-Z][A-Za-z0-9 +/'-]+?(?:\s+\d+\s*(?:mg|mcg|ml|g|iu|%))?)` + // 1) drug + optional strength
		`\s*[—\-:]\s*` +
		`([^—\-]+?)` + // 2) dosage instruction
		`(?:\s*[—\-]\s*(\d+)\s*(?:days?|day|d))?` + // 3) optional N days
		`\s*$`,
)

// ParseRxText extracts medicine lines from raw OCR text. It's permissive on
// formatting — anything that doesn't look like a prescription line is skipped.
func ParseRxText(text string) []ParsedMedicine {
	var out []ParsedMedicine
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		m := linePattern.FindStringSubmatch(line)
		if len(m) == 0 {
			continue
		}
		name := strings.TrimSpace(m[1])
		dosage := strings.TrimSpace(m[2])
		// Filter out obvious non-medicine lines (header/footer/signature).
		lower := strings.ToLower(name)
		if strings.HasPrefix(lower, "dr.") || strings.HasPrefix(lower, "rx") ||
			strings.Contains(lower, "signature") || strings.Contains(lower, "clinic") ||
			strings.Contains(lower, "hospital") {
			continue
		}
		// A real medicine line is at least 4 chars and contains at least one letter.
		if len(name) < 4 {
			continue
		}
		days := 0
		if len(m) > 3 && m[3] != "" {
			days, _ = strconv.Atoi(m[3])
		}
		out = append(out, ParsedMedicine{Name: name, Dosage: dosage, QtyDays: days})
	}
	return out
}

// ParsePrescriptionRequest is the inbound payload.
type ParsePrescriptionRequest struct {
	FileURL string `json:"file_url"`
}

// ParsePrescription runs OCR on a prescription file, parses out medicine
// lines, and tries to match each line to a Medicine in our catalog so the
// frontend can auto-populate the cart.
//   POST /pharmacy/prescriptions/parse  { file_url }
func (h *Handler) ParsePrescription(c *fiber.Ctx) error {
	var req ParsePrescriptionRequest
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.FileURL) == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "file_url required")
	}

	text, err := ocr.ExtractText(req.FileURL)
	if err != nil {
		return response.Error(c, fiber.StatusBadGateway, "OCR_FAILED", err.Error())
	}

	parsed := ParseRxText(text)
	// Try to match each line to a catalog medicine by name/brand/generic_name/synonyms.
	for i := range parsed {
		// Strip the strength suffix from the drug name for fuzzy matching:
		// "Metformin 500mg" → "Metformin".
		base := strings.TrimSpace(regexp.MustCompile(`\d+\s*(?i:mg|mcg|ml|g|iu|%)\s*$`).ReplaceAllString(parsed[i].Name, ""))
		if base == "" {
			base = parsed[i].Name
		}
		like := "%" + strings.ToLower(base) + "%"

		var m Medicine
		err := h.db.Where("is_active = ? AND (LOWER(name) LIKE ? OR LOWER(brand) LIKE ? OR LOWER(generic_name) LIKE ? OR LOWER(synonyms) LIKE ?)",
			true, like, like, like, like).
			Order("name ASC").First(&m).Error
		if err == nil {
			parsed[i].MatchedID = m.ID
			parsed[i].MatchedName = m.Name
		}
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"raw_text":  text,
		"medicines": parsed,
	}, "Prescription parsed")
}

// RegisterParsePrescriptionRoute mounts the parse endpoint.
func (h *Handler) RegisterParsePrescriptionRoute(pg fiber.Router) {
	pg.Post("/prescriptions/parse", middleware.Protected(), h.ParsePrescription)
}

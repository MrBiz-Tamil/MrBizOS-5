package pharmacy_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/money"
	"drayya-core/internal/tests/common"

	"github.com/gofiber/fiber/v2"
)

// seedAYUSHCatalog creates a small Siddha + Ayurveda + Wellness catalog used by
// the search tests. Returns the app and the in-house Nilavembu's ID.
func seedAYUSHCatalog(t *testing.T) (*fiber.App, uint) {
	t.Helper()
	app := common.SetupApp()
	db := database.DB

	inhouse := pharmacy.Medicine{
		Name: "Nilavembu Kashayam", NameTa: "நிலவேம்பு கசாயம்",
		Brand: "Dr.Ayya", Category: "siddha", Form: "kashayam", PackSize: "100 ml",
		Prices: money.RegionalPricing{"INR": 12000},
		IsActive: true, IsInhouseBrand: true,
		CatalogStatus: "published", SalesChannel: "public",
	}
	other := pharmacy.Medicine{
		Name: "Nilavembu Kashayam", NameTa: "நிலவேம்பு கசாயம்",
		Brand: "IMPCOPS", Category: "siddha", Form: "kashayam", PackSize: "100 ml",
		Prices: money.RegionalPricing{"INR": 14000},
		IsActive: true, IsInhouseBrand: false,
		CatalogStatus: "published", SalesChannel: "public",
	}
	ayur := pharmacy.Medicine{
		Name: "Triphala Churna", NameHi: "त्रिफला चूर्ण",
		Brand: "Patanjali", Category: "ayurveda", Form: "churna", PackSize: "100 g",
		Prices: money.RegionalPricing{"INR": 8000}, IsActive: true,
		CatalogStatus: "published", SalesChannel: "public",
	}
	wellness := pharmacy.Medicine{
		Name: "Tulsi Drops", Brand: "Dr.Ayya", Category: "wellness",
		Form: "oil", PackSize: "20 ml",
		Prices: money.RegionalPricing{"INR": 9900}, IsActive: true, IsInhouseBrand: true,
		CatalogStatus: "published", SalesChannel: "public",
	}
	inactive := pharmacy.Medicine{
		Name: "Discontinued Kashayam", Brand: "Dr.Ayya", Category: "siddha",
		Form: "kashayam",
		// Note: GORM's `default:true` on Medicine.IsActive treats the zero value
		// (false) as "unset" and applies the column default. Real deactivation
		// happens via Update, which is what we do below — mirrors the admin
		// "toggle medicine" flow rather than an artificial direct insert.
	}
	db.Create(&inhouse)
	db.Create(&other)
	db.Create(&ayur)
	db.Create(&wellness)
	db.Create(&inactive)
	db.Model(&inactive).Update("is_active", false)
	return app, inhouse.ID
}

func searchMedicines(t *testing.T, app *fiber.App, url string) []map[string]interface{} {
	t.Helper()
	req := httptest.NewRequest("GET", url, nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("call %s: %v", url, err)
	}
	body, _ := io.ReadAll(resp.Body)
	var envelope struct {
		Data    map[string]interface{} `json:"data"`
		Success bool                   `json:"success"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode %s: %v\nbody: %s", url, err, string(body))
	}
	if !envelope.Success {
		t.Fatalf("%s returned success=false: %s", url, string(body))
	}
	rawMeds, _ := envelope.Data["medicines"].([]interface{})
	out := make([]map[string]interface{}, 0, len(rawMeds))
	for _, m := range rawMeds {
		out = append(out, m.(map[string]interface{}))
	}
	return out
}

func TestSmartSearch_MatchesTamilAndPrefersInhouseBrand(t *testing.T) {
	app, inhouseID := seedAYUSHCatalog(t)

	meds := searchMedicines(t, app, "/api/v1/pharmacy/medicines/search?q=நிலவேம்பு")
	if len(meds) != 2 {
		t.Fatalf("expected 2 medicines matching the Tamil name, got %d (%+v)", len(meds), meds)
	}
	if uint(meds[0]["id"].(float64)) != inhouseID || meds[0]["is_inhouse_brand"] != true {
		t.Errorf("in-house brand should be ordered first; got %+v", meds[0])
	}
}

func TestSmartSearch_CategoryFilterExcludesOtherCategories(t *testing.T) {
	app, _ := seedAYUSHCatalog(t)

	meds := searchMedicines(t, app, "/api/v1/pharmacy/medicines/search?q=kashayam&category=ayurveda")
	if len(meds) != 0 {
		t.Errorf("category=ayurveda must not return siddha kashayams, got %d", len(meds))
	}

	meds = searchMedicines(t, app, "/api/v1/pharmacy/medicines/search?q=churna&category=ayurveda")
	if len(meds) != 1 {
		t.Fatalf("expected the Triphala churna, got %d", len(meds))
	}
	if !strings.Contains(meds[0]["name_hi"].(string), "त्रिफला") {
		t.Errorf("Hindi name should be surfaced; got %q", meds[0]["name_hi"])
	}
}

func TestSmartSearch_ExcludesInactiveAndShortQueries(t *testing.T) {
	app, _ := seedAYUSHCatalog(t)

	meds := searchMedicines(t, app, "/api/v1/pharmacy/medicines/search?q=kashayam")
	for _, m := range meds {
		if strings.Contains(m["name"].(string), "Discontinued") {
			t.Errorf("inactive medicines must not appear in autocomplete: %+v", m)
		}
	}

	meds = searchMedicines(t, app, "/api/v1/pharmacy/medicines/search?q=k")
	if len(meds) != 0 {
		t.Errorf("query <2 chars should return empty; got %d", len(meds))
	}
}

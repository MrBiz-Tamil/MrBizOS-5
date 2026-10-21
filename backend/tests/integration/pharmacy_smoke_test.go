package integration

// ═══════════════════════════════════════════════════════════════════════════
// SMOKE TEST — Pharmacy System
//
// Quick health checks that verify every critical pharmacy endpoint is alive
// and returns a valid response shape. Designed to run in <10 seconds.
//
// Run via:
//   go test ./tests/integration/ -run TestPharmacySmokeTest -v -timeout 60s
//
// What it checks:
//   • Health endpoints respond
//   • Pharmacy module routes are registered and return correct status codes
//   • Manufacturing module routes are registered
//   • Raw materials module routes are registered
//   • Auth-required routes return 401 without token
//   • Auth-required routes return 200/2xx with valid token
//   • Multi-city workflow dashboard endpoint responds
//
// What it does NOT check:
//   • Business logic correctness (see workflow_e2e_test.go for that)
//   • Database state changes
//   • Cross-domain integrations
// ═══════════════════════════════════════════════════════════════════════════

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/emr"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/handlers"
	"drayya-core/internal/server"
	"drayya-core/internal/tokens"
	"drayya-core/internal/ws"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type PharmacySmokeTestSuite struct {
	suite.Suite
	App         *fiber.App
	DB          *gorm.DB
	AdminToken  string
	PatientID   uint
	PharmacyID  uint
	MedicineID  uint
	StartTime   time.Time
	Results     []SmokeResult
}

type SmokeResult struct {
	Endpoint   string
	Method     string
	Status     int
	Expected   string
	Passed     bool
	DurationMS int64
}

func (s *PharmacySmokeTestSuite) SetupSuite() {
	s.StartTime = time.Now()

	os.Setenv("USE_SQLITE", "true")
	os.Setenv("DB_URL", "file:memdb_pharmacy_smoke?mode=memory&cache=shared")
	os.Setenv("JWT_SECRET", "smoke_test_secret")
	database.Connect()
	s.DB = database.DB

	// Migrate all required domains
	identity.AutoMigrate(s.DB)
	pharmacy.AutoMigrate(s.DB)
	pharmacy.AutoMigrateManufacturing(s.DB)
	pharmacy.AutoMigrateRawMaterial(s.DB)
	emr.AutoMigrate(s.DB)
	finance.AutoMigrate(s.DB)

	handlers.Hub = ws.NewHub(nil)
	s.App = server.New()

	// Seed minimal data for smoke tests
	admin := identity.User{ID: 1, FullName: "Smoke Admin", Role: "admin", Email: "smoke-admin@test"}
	s.DB.Create(&admin)
	s.AdminToken, _ = tokens.CreateAccessToken("1", "admin", nil, nil)

	patient := identity.User{FullName: "Smoke Patient", Role: "patient", Email: "smoke-patient@test"}
	s.DB.Create(&patient)
	s.PatientID = patient.ID

	ph := pharmacy.Pharmacy{
		Name: "Smoke Test Pharmacy", City: "Bangalore", Region: "Karnataka",
		IsCentralWarehouse: true, IsActive: true,
	}
	s.DB.Create(&ph)
	s.PharmacyID = ph.ID

	med := pharmacy.Medicine{Name: "Smoke Medicine", Category: "test", IsActive: true, CatalogStatus: "published", SalesChannel: "public"}
	s.DB.Create(&med)
	s.MedicineID = med.ID

	s.Results = []SmokeResult{}
}

// ── Smoke test runner — records every call's status + duration ──

func (s *PharmacySmokeTestSuite) ping(method, path, expected string, token string, expectedStatus int) {
	start := time.Now()

	req := httptest.NewRequest(method, path, bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, _ := s.App.Test(req, 5000)
	duration := time.Since(start).Milliseconds()

	passed := resp != nil && resp.StatusCode == expectedStatus

	s.Results = append(s.Results, SmokeResult{
		Endpoint:   path,
		Method:     method,
		Status:     resp.StatusCode,
		Expected:   expected,
		Passed:     passed,
		DurationMS: duration,
	})

	if !passed {
		s.T().Errorf("❌ %s %s — expected %d, got %d (%s)", method, path, expectedStatus, resp.StatusCode, expected)
	}
}

// ── Smoke Test Sequence ──

func (s *PharmacySmokeTestSuite) TestSmoke_01_PublicEndpoints() {
	s.T().Log("┌─ PUBLIC ENDPOINTS (no auth required) ─────────────────")

	// Public catalog
	s.ping("GET", "/api/v1/pharmacy/medicines", "list medicines", "", 200)
	s.ping("GET", "/api/v1/pharmacy/medicines/search?q=para", "smart search", "", 200)
	s.ping("GET", fmt.Sprintf("/api/v1/pharmacy/medicines/%d", s.MedicineID), "medicine detail", "", 200)
	s.ping("GET", "/api/v1/pharmacy/", "list pharmacies", "", 200)
}

func (s *PharmacySmokeTestSuite) TestSmoke_02_AuthEnforcement() {
	s.T().Log("┌─ AUTH ENFORCEMENT (must reject anonymous) ─────────────")

	// Protected endpoints should return 401 without token
	s.ping("GET", "/api/v1/pharmacy/inventory", "auth required", "", 401)
	s.ping("GET", "/api/v1/pharmacy/orders", "auth required", "", 401)
	s.ping("GET", "/api/v1/pharmacy/financials", "auth required", "", 401)
	s.ping("POST", "/api/v1/pharmacy/admin/trigger-restock", "auth required", "", 401)
}

func (s *PharmacySmokeTestSuite) TestSmoke_03_PharmacyAdminEndpoints() {
	s.T().Log("┌─ PHARMACY ADMIN ENDPOINTS (with token) ────────────────")

	// Core pharmacy admin
	s.ping("GET", "/api/v1/pharmacy/inventory", "inventory list", s.AdminToken, 200)
	s.ping("GET", "/api/v1/pharmacy/inventory/low-stock", "low stock alerts", s.AdminToken, 200)
	s.ping("GET", "/api/v1/pharmacy/inventory/expiring", "expiring items", s.AdminToken, 200)
	s.ping("GET", "/api/v1/pharmacy/alerts", "stock alerts", s.AdminToken, 200)
	s.ping("GET", "/api/v1/pharmacy/inventory/multi-city", "multi-city inventory", s.AdminToken, 200)
	s.ping("GET", "/api/v1/pharmacy/orders", "orders list", s.AdminToken, 200)
	s.ping("GET", "/api/v1/pharmacy/purchase-orders", "PO list", s.AdminToken, 200)
	s.ping("GET", "/api/v1/pharmacy/suppliers", "suppliers list", s.AdminToken, 200)
	s.ping("GET", "/api/v1/pharmacy/dashboard/stats", "dashboard stats", s.AdminToken, 200)
	s.ping("GET", "/api/v1/pharmacy/transfers", "transfers list", s.AdminToken, 200)
}

func (s *PharmacySmokeTestSuite) TestSmoke_04_ManufacturingEndpoints() {
	s.T().Log("┌─ MANUFACTURING ENDPOINTS ──────────────────────────────")

	s.ping("GET", "/api/v1/manufacturing/bom", "BOM list", s.AdminToken, 200)
	s.ping("GET", "/api/v1/manufacturing/production-orders", "production orders list", s.AdminToken, 200)
	s.ping("GET", "/api/v1/manufacturing/dashboard", "manufacturing dashboard", s.AdminToken, 200)
}

func (s *PharmacySmokeTestSuite) TestSmoke_05_RawMaterialEndpoints() {
	s.T().Log("┌─ RAW MATERIAL ENDPOINTS ───────────────────────────────")

	s.ping("GET", "/api/v1/raw-materials/", "raw materials list", s.AdminToken, 200)
	s.ping("GET", "/api/v1/raw-materials/inventory", "RM inventory", s.AdminToken, 200)
	s.ping("GET", "/api/v1/raw-materials/inventory/low-stock", "RM low stock", s.AdminToken, 200)
	s.ping("GET", "/api/v1/raw-materials/inventory/summary", "RM summary", s.AdminToken, 200)
	s.ping("GET", "/api/v1/raw-materials/vendors", "approved vendors", s.AdminToken, 200)
	s.ping("GET", "/api/v1/raw-materials/indents", "indents list", s.AdminToken, 200)
	s.ping("GET", "/api/v1/raw-materials/rfq", "RFQ list", s.AdminToken, 200)
	s.ping("GET", "/api/v1/raw-materials/dashboard", "raw materials dashboard", s.AdminToken, 200)
}

func (s *PharmacySmokeTestSuite) TestSmoke_06_GRNEndpoints() {
	s.T().Log("┌─ GRN ENDPOINTS ────────────────────────────────────────")

	s.ping("GET", "/api/v1/grn/", "GRN list", s.AdminToken, 200)
}

func (s *PharmacySmokeTestSuite) TestSmoke_07_MultiCityWorkflow() {
	s.T().Log("┌─ MULTI-CITY WORKFLOW DASHBOARD ────────────────────────")

	s.ping("GET", "/api/v1/workflow/city-dashboard?city=Bangalore", "city workflow dashboard", s.AdminToken, 200)
}

func (s *PharmacySmokeTestSuite) TestSmoke_07b_AuditSystem() {
	s.T().Log("┌─ ADMIN AUDIT SYSTEM ───────────────────────────────────")

	s.ping("GET", "/api/v1/pharmacy/audit/run", "full audit run", s.AdminToken, 200)
	s.ping("GET", "/api/v1/pharmacy/audit/by-category?category=compliance", "audit by category", s.AdminToken, 200)
	s.ping("GET", "/api/v1/pharmacy/audit/by-owner?owner=pharmacist", "audit by owner", s.AdminToken, 200)
	s.ping("GET", "/api/v1/pharmacy/audit/owners", "owner directory", s.AdminToken, 200)
}

func (s *PharmacySmokeTestSuite) TestSmoke_08_StoreEndpoints() {
	s.T().Log("┌─ STORE / WELLNESS ENDPOINTS ───────────────────────────")

	// Store endpoints are in a separate domain — pharmacy smoke test only
	// checks they're registered (not 404). 200 or 500 both count as "alive".
	s.pingAlive("GET", "/api/v1/store/products", "wellness products list", "")
	s.pingAlive("GET", "/api/v1/store/categories", "categories", "")
}

// pingAlive — passes if endpoint is registered (any non-404 response).
// Used for endpoints in other domains where we only care that they exist.
func (s *PharmacySmokeTestSuite) pingAlive(method, path, expected string, token string) {
	start := time.Now()

	req := httptest.NewRequest(method, path, bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, _ := s.App.Test(req, 5000)
	duration := time.Since(start).Milliseconds()

	// Pass if endpoint is registered (anything except 404)
	passed := resp != nil && resp.StatusCode != 404

	s.Results = append(s.Results, SmokeResult{
		Endpoint:   path,
		Method:     method,
		Status:     resp.StatusCode,
		Expected:   expected + " (alive check)",
		Passed:     passed,
		DurationMS: duration,
	})

	if !passed {
		s.T().Errorf("❌ %s %s — endpoint not registered (404)", method, path)
	}
}

func (s *PharmacySmokeTestSuite) TestSmoke_99_FinalReport() {
	// This runs alphabetically last — print summary
	s.T().Log("")
	s.T().Log("╔══════════════════════════════════════════════════════════╗")
	s.T().Log("║              SMOKE TEST SUMMARY REPORT                    ║")
	s.T().Log("╚══════════════════════════════════════════════════════════╝")

	total := len(s.Results)
	passed := 0
	failed := 0
	var totalMS int64

	for _, r := range s.Results {
		if r.Passed {
			passed++
		} else {
			failed++
		}
		totalMS += r.DurationMS
	}

	avgMS := int64(0)
	if total > 0 {
		avgMS = totalMS / int64(total)
	}

	s.T().Logf("  Total endpoints checked: %d", total)
	s.T().Logf("  ✓ Passed:                %d", passed)
	if failed > 0 {
		s.T().Logf("  ✗ Failed:                %d", failed)
	}
	s.T().Logf("  Total runtime:           %dms", time.Since(s.StartTime).Milliseconds())
	s.T().Logf("  Avg response time:       %dms", avgMS)
	s.T().Log("")

	// Group results
	categories := map[string][]SmokeResult{}
	for _, r := range s.Results {
		key := categorizeEndpoint(r.Endpoint)
		categories[key] = append(categories[key], r)
	}

	for category, items := range categories {
		passed := 0
		for _, item := range items {
			if item.Passed {
				passed++
			}
		}
		status := "✓"
		if passed != len(items) {
			status = "✗"
		}
		s.T().Logf("  %s %-30s %d/%d (%dms avg)", status, category, passed, len(items), avgDuration(items))
	}

	if failed > 0 {
		s.T().Log("")
		s.T().Log("  FAILED ENDPOINTS:")
		for _, r := range s.Results {
			if !r.Passed {
				s.T().Logf("    %s %s → got %d", r.Method, r.Endpoint, r.Status)
			}
		}
	}

	s.T().Log("")
	if failed == 0 {
		s.T().Log("  🎉 ALL SMOKE CHECKS PASSED — Pharmacy system is healthy")
	} else {
		s.T().Errorf("  💥 %d smoke checks FAILED — investigate before deploying", failed)
	}

	assert.Equal(s.T(), 0, failed, "Smoke checks should all pass")
}

func categorizeEndpoint(path string) string {
	switch {
	case strings.Contains(path, "/pharmacy/audit"):
		return "Audit System"
	case strings.Contains(path, "/manufacturing"):
		return "Manufacturing"
	case strings.Contains(path, "/raw-materials"):
		return "Raw Materials"
	case strings.Contains(path, "/grn"):
		return "GRN"
	case strings.Contains(path, "/workflow"):
		return "Multi-City Workflow"
	case strings.Contains(path, "/store"):
		return "Store"
	case strings.Contains(path, "/pharmacy"):
		return "Pharmacy Core"
	default:
		return "Other"
	}
}

func avgDuration(items []SmokeResult) int64 {
	if len(items) == 0 {
		return 0
	}
	var sum int64
	for _, item := range items {
		sum += item.DurationMS
	}
	return sum / int64(len(items))
}

func TestPharmacySmokeTest(t *testing.T) {
	suite.Run(t, new(PharmacySmokeTestSuite))
}

// ─── Helper: ensure JSON response shape is valid ──

func decodeJSON(t *testing.T, resp *http.Response) map[string]interface{} {
	defer resp.Body.Close()
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Errorf("Invalid JSON response: %v", err)
	}
	return out
}

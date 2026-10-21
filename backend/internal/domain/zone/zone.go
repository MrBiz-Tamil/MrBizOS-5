// Package zone provides geographic service zones for in-person / hospital care.
//
// Distinct from specialty pods (clinical oversight): a service zone maps a set
// of pincodes to the doctors who serve that area, so an in-person booking can
// show same-zone doctors first ("availability zone is the first approach").
package zone

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// pincodesJSON marshals a string slice to a JSONB value.
func pincodesJSON(p []string) datatypes.JSON {
	b, err := json.Marshal(p)
	if err != nil {
		return datatypes.JSON("[]")
	}
	return datatypes.JSON(b)
}

// ServiceZone is the platform's single geographic primitive. Hybrid hierarchy:
// a city-level zone (ParentID nil) can hold pincode-level sub-zones (ParentID
// set). Every operational unit — doctors, labs, pharmacies, stores, hospitals,
// franchises — maps to a zone via its zone_id.
type ServiceZone struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"not null" json:"name"` // "Chennai" or "South Chennai"
	City      string         `gorm:"index" json:"city"`
	State     string         `json:"state"`
	// ParentID nil → city-level zone; set → pincode sub-zone of that city.
	ParentID  *uint          `gorm:"index" json:"parent_id,omitempty"`
	Kind      string         `gorm:"default:'subzone'" json:"kind"` // city | subzone
	Pincodes  datatypes.JSON `gorm:"type:jsonb" json:"pincodes"`    // sub-zones carry pincodes
	CenterLat float64        `json:"center_lat"`
	CenterLng float64        `json:"center_lng"`
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

func (ServiceZone) TableName() string { return "service_zones" }

func AutoMigrate(db *gorm.DB) error { return db.AutoMigrate(&ServiceZone{}) }

// ResolveByPincode finds the sub-zone whose pincode list contains the given
// pincode. Returns the matched zone and true, or false if no zone covers it.
// Shared across domains so every unit resolves geography the same way.
func ResolveByPincode(db *gorm.DB, pincode string) (ServiceZone, bool) {
	pincode = strings.TrimSpace(pincode)
	if pincode == "" {
		return ServiceZone{}, false
	}
	var z ServiceZone
	err := db.Where("is_active = ? AND pincodes LIKE ?", true, "%\""+pincode+"\"%").First(&z).Error
	if err != nil {
		return ServiceZone{}, false
	}
	return z, true
}

// ResolveIDByPincode is the convenience form returning just the zone id (0 if none).
func ResolveIDByPincode(db *gorm.DB, pincode string) uint {
	if z, ok := ResolveByPincode(db, pincode); ok {
		return z.ID
	}
	return 0
}

type Handler struct{ db *gorm.DB }

func NewHandler(db *gorm.DB) *Handler { return &Handler{db: db} }

func (h *Handler) RegisterRoutes(router fiber.Router) {
	g := router.Group("/zones")
	// Booking-time resolution is read-only and patient-facing.
	g.Get("/match", middleware.Protected(), h.MatchByPincode)
	// Management is manager/admin.
	g.Get("/", middleware.ManagerOnly(), h.ListZones)
	g.Post("/", middleware.ManagerOnly(), h.CreateZone)
	g.Put("/:id", middleware.ManagerOnly(), h.UpdateZone)

	// Zone intelligence — scorecard + expansion recommendations (Phase 2/3).
	h.registerScorecardRoutes(g)
}

type zoneRow struct {
	ServiceZone
	// Physical units in the zone. (Online doctors are network-wide, not zonal.)
	Pharmacies int64 `json:"pharmacies"`
	Labs       int64 `json:"labs"`
	Hospitals  int64 `json:"hospitals"`
}

// ListZones returns zones with their physical-unit counts.
func (h *Handler) ListZones(c *fiber.Ctx) error {
	var zones []ServiceZone
	if err := h.db.Order("name ASC").Find(&zones).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	rows := make([]zoneRow, 0, len(zones))
	for _, z := range zones {
		r := zoneRow{ServiceZone: z}
		h.db.Table("pharmacy").Where("zone_id = ?", z.ID).Count(&r.Pharmacies)
		h.db.Table("lab_centers").Where("zone_id = ?", z.ID).Count(&r.Labs)
		h.db.Table("hospitals").Where("zone_id = ?", z.ID).Count(&r.Hospitals)
		rows = append(rows, r)
	}
	return response.JSON(c, fiber.StatusOK, rows, "Zones fetched")
}

func (h *Handler) CreateZone(c *fiber.Ctx) error {
	var req struct {
		Name     string   `json:"name"`
		City     string   `json:"city"`
		State    string   `json:"state"`
		ParentID *uint    `json:"parent_id"`
		Pincodes []string `json:"pincodes"`
		CenterLat float64 `json:"center_lat"`
		CenterLng float64 `json:"center_lng"`
	}
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "name is required")
	}
	kind := "subzone"
	if req.ParentID == nil {
		kind = "city" // top-level zone
	}
	z := ServiceZone{
		Name: strings.TrimSpace(req.Name), City: strings.TrimSpace(req.City), State: strings.TrimSpace(req.State),
		ParentID: req.ParentID, Kind: kind, Pincodes: pincodesJSON(req.Pincodes),
		CenterLat: req.CenterLat, CenterLng: req.CenterLng, IsActive: true,
	}
	if err := h.db.WithContext(c.UserContext()).Create(&z).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, z, "Zone created")
}

func (h *Handler) UpdateZone(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid zone ID")
	}
	var req struct {
		Name     *string   `json:"name"`
		City     *string   `json:"city"`
		Pincodes *[]string `json:"pincodes"`
		IsActive *bool     `json:"is_active"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	updates := map[string]interface{}{}
	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.City != nil {
		updates["city"] = strings.TrimSpace(*req.City)
	}
	if req.Pincodes != nil {
		updates["pincodes"] = pincodesJSON(*req.Pincodes)
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if len(updates) == 0 {
		return response.Error(c, fiber.StatusBadRequest, "NO_FIELDS", "No updatable fields")
	}
	if err := h.db.WithContext(c.UserContext()).Model(&ServiceZone{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"id": id}, "Zone updated")
}

// MatchByPincode resolves a patient's pincode to its service zone and returns
// the PHYSICAL units that serve it — pharmacies, labs, hospitals. This powers
// delivery + in-person ("what's near me"), NOT online consults (those match on
// language + medical system, with no geography).
//
// GET /zones/match?pincode=600001
func (h *Handler) MatchByPincode(c *fiber.Ctx) error {
	pincode := strings.TrimSpace(c.Query("pincode"))
	if pincode == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "pincode is required")
	}

	zone, ok := ResolveByPincode(h.db, pincode)
	if !ok {
		return response.JSON(c, fiber.StatusOK, fiber.Map{
			"zone": nil,
			"note": "No service zone covers this pincode yet.",
		}, "Zone match fetched")
	}

	// The zone + its parent city (for "South Chennai · Chennai" display).
	resp := fiber.Map{"zone": zone}
	zoneIDs := []uint{zone.ID}
	if zone.ParentID != nil {
		var parent ServiceZone
		if h.db.First(&parent, *zone.ParentID).Error == nil {
			resp["city_zone"] = parent
		}
	}

	var pharmacies, labs, hospitals int64
	h.db.Table("pharmacy").Where("zone_id IN ?", zoneIDs).Count(&pharmacies)
	h.db.Table("lab_centers").Where("zone_id IN ?", zoneIDs).Count(&labs)
	h.db.Table("hospitals").Where("zone_id IN ?", zoneIDs).Count(&hospitals)
	resp["serves"] = fiber.Map{"pharmacies": pharmacies, "labs": labs, "hospitals": hospitals}

	return response.JSON(c, fiber.StatusOK, resp, "Zone match fetched")
}

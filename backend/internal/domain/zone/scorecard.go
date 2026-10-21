package zone

// scorecard.go — Phase 2/3: zone intelligence engine.
//
// Per city-zone, blends real supply (doctors, pharmacies, labs, hospitals),
// demand (pharmacy orders to the zone's pincodes), and quality into a health
// score + tier (thriving/healthy/underserved/unserved), and surfaces the
// demand−supply gap that drives expansion (Phase 3).

import (
	"encoding/json"
	"math"

	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

func (h *Handler) registerScorecardRoutes(g fiber.Router) {
	g.Get("/scorecard", middleware.ManagerOnly(), h.Scorecard)
}

// ZoneStat is one city-zone's health rollup. Zones track the PHYSICAL business
// (pharmacies, labs, hospitals, stores) — NOT online doctors, who serve every
// zone and are matched by language + medical system, not geography.
type ZoneStat struct {
	ZoneID       uint   `json:"zone_id"`
	Name         string `json:"name"`
	City         string `json:"city"`
	State        string `json:"state"`
	Pharmacies   int64  `json:"pharmacies"`
	Labs         int64  `json:"labs"`
	Hospitals    int64  `json:"hospitals"`
	Orders       int64  `json:"orders"`
	RevenuePaise int64  `json:"revenue_paise"`
	Score        float64 `json:"score"`
	Tier         string  `json:"tier"` // thriving | healthy | underserved | unserved
	// Recommendation — what the demand-supply gap suggests opening (Phase 3).
	Recommendation string `json:"recommendation"`
}

// computeZoneStat aggregates one city-zone across all its sub-zones.
func (h *Handler) computeZoneStat(city ServiceZone) ZoneStat {
	s := ZoneStat{ZoneID: city.ID, Name: city.Name, City: city.City, State: city.State}

	// Sub-zone ids (incl. the city zone itself) + the union of their pincodes.
	zoneIDs := []uint{city.ID}
	pincodes := []string{}
	var subs []ServiceZone
	h.db.Where("parent_id = ?", city.ID).Find(&subs)
	collect := func(z ServiceZone) {
		var ps []string
		if len(z.Pincodes) > 0 {
			_ = json.Unmarshal(z.Pincodes, &ps)
		}
		pincodes = append(pincodes, ps...)
	}
	collect(city)
	for _, sz := range subs {
		zoneIDs = append(zoneIDs, sz.ID)
		collect(sz)
	}

	// Supply — PHYSICAL business units only. Online doctors are NOT a zone
	// resource (they serve every zone, matched by language + medical system).
	h.db.Table("pharmacy").Where("zone_id IN ?", zoneIDs).Count(&s.Pharmacies)
	h.db.Table("lab_centers").Where("zone_id IN ?", zoneIDs).Count(&s.Labs)
	h.db.Table("hospitals").Where("zone_id IN ?", zoneIDs).Count(&s.Hospitals)

	// Demand — pharmacy orders shipped to this zone's pincodes.
	if len(pincodes) > 0 {
		h.db.Table("pharmacy_orders").Where("dest_pincode IN ?", pincodes).Count(&s.Orders)
		h.db.Table("pharmacy_orders").Where("dest_pincode IN ?", pincodes).
			Select("COALESCE(SUM(amount), 0)").Scan(&s.RevenuePaise)
	}

	s.Score, s.Tier, s.Recommendation = scoreZone(s)
	return s
}

// scoreZone turns the rollup into a 0–100 health score, a tier, and an
// expansion recommendation based on which kind of supply the demand lacks.
func scoreZone(s ZoneStat) (float64, string, string) {
	// Physical supply vs local demand. A zone is "healthy" when its pharmacies/
	// labs/hospitals keep pace with the orders flowing to its pincodes.
	supply := s.Pharmacies + s.Labs + s.Hospitals
	demand := s.Orders

	supplyScore := math.Min(100, float64(supply)*18)
	demandScore := math.Min(100, float64(demand)*5)
	// Coverage = how well supply meets demand (the real signal for expansion).
	coverage := 100.0
	if demand > 0 {
		coverage = math.Min(100, (float64(supply)/float64(demand))*100*5) // ~1 unit per 20 orders = full
	}
	score := math.Round((supplyScore*0.35+demandScore*0.25+coverage*0.40)*10) / 10

	tier := "healthy"
	switch {
	case demand > 0 && supply == 0:
		tier = "unserved" // people want it, nobody serves it
	case demand > int64(supply)*20 && supply > 0:
		tier = "underserved" // demand far outstrips thin physical presence
	case score >= 70:
		tier = "thriving"
	case score < 35 && demand > 0:
		tier = "underserved"
	}

	// Recommendation: what physical unit to open, driven by the gap.
	rec := "Coverage looks balanced."
	if demand > 0 || tier == "underserved" || tier == "unserved" {
		switch {
		case s.Pharmacies == 0 && s.Orders > 0:
			rec = "Open a pharmacy/store here — orders exist with no local pharmacy."
		case s.Labs == 0 && s.Orders > 0:
			rec = "Add a lab or sample-collection point — no diagnostics coverage."
		case s.Hospitals == 0 && tier == "underserved":
			rec = "Consider an own hospital or a franchise — strong demand, no physical centre."
		case tier == "underserved":
			rec = "Grow physical supply (pharmacy/lab) — demand outstrips current coverage."
		}
	}
	return score, tier, rec
}

// Scorecard ranks every city-zone by health, with a distribution header.
//
// GET /zones/scorecard   (manager/admin)
func (h *Handler) Scorecard(c *fiber.Ctx) error {
	var cities []ServiceZone
	if err := h.db.Where("kind = ? AND is_active = ?", "city", true).Order("name ASC").Find(&cities).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	stats := make([]ZoneStat, 0, len(cities))
	dist := map[string]int{"thriving": 0, "healthy": 0, "underserved": 0, "unserved": 0}
	for _, city := range cities {
		st := h.computeZoneStat(city)
		stats = append(stats, st)
		dist[st.Tier]++
	}
	// Rank best score first.
	sortZoneStats(stats)
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"zones":        stats,
		"distribution": dist,
	}, "Zone scorecard fetched")
}

func sortZoneStats(s []ZoneStat) {
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j].Score > s[i].Score {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

package clinical

// scorecard.go — Phase 2: doctor scorecard & tiers.
//
// Blends the real signals already captured (patient rating, completion rate,
// consultation-QA score, volume) into one 0–100 score, bucketed into
// bronze/silver/gold/platinum. Tier drives ranking and payout share. Pod-scoped
// like the rest of clinical oversight.

import (
	"math"
	"time"

	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// Scoring weights — tuned so quality (rating + QA) dominates, with completion
// and volume as supporting signals. Sum = 1.0.
const (
	wRating     = 0.35
	wCompletion = 0.30
	wQA         = 0.25
	wVolume     = 0.10
)

// scoreToTier buckets a 0–100 score.
func scoreToTier(score float64) string {
	switch {
	case score >= 85:
		return "platinum"
	case score >= 70:
		return "gold"
	case score >= 50:
		return "silver"
	default:
		return "bronze"
	}
}

// computeScore blends the signals. rating & qa are 0–5, completion is 0–100,
// volume is a raw consultation count (log-scaled so a handful of consults still
// counts but doesn't dominate). Returns the 0–100 score.
func computeScore(rating, qa, completion float64, volume int) float64 {
	ratingNorm := (rating / 5) * 100
	// No QA reviews yet → fall back to the rating signal so a new doctor isn't
	// punished for the head doctor not having reviewed them.
	qaNorm := ratingNorm
	if qa > 0 {
		qaNorm = (qa / 5) * 100
	}
	// Volume: 0 consults → 0, ramps via log10 so 10→~50, 100→~100, capped.
	volumeScore := 0.0
	if volume > 0 {
		volumeScore = math.Min(100, math.Log10(float64(volume)+1)*50)
	}
	score := wRating*ratingNorm + wCompletion*completion + wQA*qaNorm + wVolume*volumeScore
	return math.Round(score*10) / 10
}

func (h *Handler) registerScorecardRoutes(g fiber.Router) {
	g.Get("/scorecard", h.Scorecard)
	g.Post("/scorecard/recompute", middleware.ManagerOnly(), h.RecomputeScorecards)
}

type scorecardRow struct {
	DoctorID           uint    `json:"doctor_id"`
	Name               string  `json:"name"`
	AverageRating      float64 `json:"average_rating"`
	CompletionRate     float64 `json:"completion_rate"`
	QAAvg              float64 `json:"qa_avg"`
	TotalConsultations int     `json:"total_consultations"`
	Score              float64 `json:"score"`
	Tier               string  `json:"tier"`
}

// scorecardQuery returns the raw signal rows for the caller's scope (pod for a
// head doctor; whole network for manager/admin).
func (h *Handler) scorecardRows(c *fiber.Ctx) ([]scorecardRow, error) {
	q := h.db.Table("doctorprofile AS dp").
		Select(`dp.user_id AS doctor_id, du.full_name AS name,
			dp.average_rating, dp.completion_rate, dp.total_consultations,
			COALESCE((SELECT AVG(quality_score) FROM consultation_reviews cr WHERE cr.doctor_id = dp.user_id), 0) AS qa_avg`).
		Joins(`JOIN "user" du ON du.id = dp.user_id`).
		Where("dp.is_verified = ?", true)
	if ids, scoped := h.scopedDoctorIDs(c); scoped {
		q = q.Where("dp.user_id IN ?", ids)
	}
	var rows []scorecardRow
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].Score = computeScore(rows[i].AverageRating, rows[i].QAAvg, rows[i].CompletionRate, rows[i].TotalConsultations)
		rows[i].Tier = scoreToTier(rows[i].Score)
	}
	return rows, nil
}

// Scorecard returns the live scorecard for the caller's scope, ranked best-first.
//
// GET /clinical/scorecard
func (h *Handler) Scorecard(c *fiber.Ctx) error {
	rows, err := h.scorecardRows(c)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	// Rank best score first.
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[j].Score > rows[i].Score {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
	// Tier distribution for the header.
	dist := map[string]int{"platinum": 0, "gold": 0, "silver": 0, "bronze": 0}
	for _, r := range rows {
		dist[r.Tier]++
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"doctors":      rows,
		"distribution": dist,
	}, "Scorecard fetched")
}

// RecomputeScorecards persists the computed score + tier onto each doctor so
// payouts and listings can read a stable tier between runs.
//
// POST /clinical/scorecard/recompute   (manager / admin)
func (h *Handler) RecomputeScorecards(c *fiber.Ctx) error {
	rows, err := h.scorecardRows(c)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	now := time.Now()
	updated := 0
	for _, r := range rows {
		if err := h.db.WithContext(c.UserContext()).Table("doctorprofile").
			Where("user_id = ?", r.DoctorID).
			Updates(map[string]interface{}{
				"scorecard_score":  r.Score,
				"tier":             r.Tier,
				"tier_computed_at": now,
			}).Error; err == nil {
			updated++
		}
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"updated": updated}, "Scorecards recomputed")
}

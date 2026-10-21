package clinical

// dashboard.go — GET /clinical/dashboard
//
// The head-doctor overview. One real aggregation across consultations, QA
// reviews, the onboarding pipeline, doctor verification, blogs, and doctor
// performance — everything the head-doctor home needs in a single call.

import (
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type perfRow struct {
	DoctorID           uint    `json:"doctor_id"`
	Name               string  `json:"name"`
	AverageRating      float64 `json:"average_rating"`
	CompletionRate     float64 `json:"completion_rate"`
	TotalConsultations int     `json:"total_consultations"`
}

// Dashboard returns the consolidated head-doctor overview.
//
// GET /clinical/dashboard
func (h *Handler) Dashboard(c *fiber.Ctx) error {
	// Pod scoping — a head doctor's dashboard reflects only their pod.
	ids, scoped := h.scopedDoctorIDs(c)
	var podName string
	if uid, _ := c.Locals("user_id").(uint); uid > 0 {
		h.db.Table("pods").Where("lead_doctor_id = ? AND is_active = ?", uid, true).Select("name").Scan(&podName)
	}
	scopeDocCol := func(q *gorm.DB, col string) *gorm.DB {
		if scoped {
			return q.Where(col+" IN ?", ids)
		}
		return q
	}

	// ── Doctor counts ────────────────────────────────────────────────────────
	var onlineDoctors, verifiedDoctors, pendingVerifications int64
	scopeDocCol(h.db.Table("doctorprofile"), "user_id").Where("is_verified = ? AND is_online = ?", true, true).Count(&onlineDoctors)
	scopeDocCol(h.db.Table("doctorprofile"), "user_id").Where("is_verified = ?", true).Count(&verifiedDoctors)
	scopeDocCol(h.db.Table("doctorprofile"), "user_id").Where("is_verified = ?", false).Count(&pendingVerifications)

	// ── Consultation QA ──────────────────────────────────────────────────────
	var qaPending, qaReviewed, qaFlagged int64
	scopeDocCol(h.db.Table("appointments AS a").
		Where("a.status = ?", "completed").
		Where("NOT EXISTS (SELECT 1 FROM consultation_reviews cr WHERE cr.appointment_id = a.id)"), "a.doctor_id").
		Count(&qaPending)
	scopeDocCol(h.db.Model(&ConsultationReview{}), "doctor_id").Count(&qaReviewed)
	scopeDocCol(h.db.Model(&ConsultationReview{}).Where("flagged = ?", true), "doctor_id").Count(&qaFlagged)
	var avgQuality float64
	scopeDocCol(h.db.Model(&ConsultationReview{}).Select("COALESCE(AVG(quality_score), 0)"), "doctor_id").Scan(&avgQuality)

	// ── Onboarding pipeline ──────────────────────────────────────────────────
	type sc struct {
		Stage string
		N     int64
	}
	var stages []sc
	scopeDocCol(h.db.Model(&DoctorOnboarding{}).Select("stage, COUNT(*) AS n"), "doctor_id").Group("stage").Scan(&stages)
	onboardingCounts := map[string]int64{}
	var inOnboarding int64
	for _, s := range stages {
		onboardingCounts[s.Stage] = s.N
		if s.Stage != StageActive && s.Stage != StageRejected {
			inOnboarding += s.N
		}
	}
	readyToActivate := onboardingCounts[StageGoLive]

	// ── Pending blog approvals ───────────────────────────────────────────────
	var pendingBlogs int64
	h.db.Table("blogs").Where("status = ?", "pending").Count(&pendingBlogs)

	// ── Doctor performance (real: from doctorprofile aggregates) ─────────────
	perfBase := func() *gorm.DB {
		q := h.db.Table("doctorprofile AS dp").
			Select("dp.user_id AS doctor_id, du.full_name AS name, dp.average_rating, dp.completion_rate, dp.total_consultations").
			Joins(`JOIN "user" du ON du.id = dp.user_id`).
			Where("dp.is_verified = ? AND dp.total_consultations > 0", true)
		if scoped {
			q = q.Where("dp.user_id IN ?", ids)
		}
		return q
	}
	var topPerformers, needsCoaching []perfRow
	perfBase().Order("dp.average_rating DESC, dp.completion_rate DESC").Limit(5).Scan(&topPerformers)
	perfBase().Where("dp.average_rating > 0").Order("dp.average_rating ASC, dp.completion_rate ASC").Limit(5).Scan(&needsCoaching)

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"pod_name":     podName,   // "" for manager/admin (network-wide) or unassigned head doctor
		"pod_scoped":   scoped,
		"doctors": fiber.Map{
			"online":               onlineDoctors,
			"verified":             verifiedDoctors,
			"pending_verification": pendingVerifications,
		},
		"qa": fiber.Map{
			"pending":     qaPending,
			"reviewed":    qaReviewed,
			"flagged":     qaFlagged,
			"avg_quality": avgQuality,
		},
		"onboarding": fiber.Map{
			"in_pipeline":       inOnboarding,
			"ready_to_activate": readyToActivate,
			"by_stage":          onboardingCounts,
		},
		"pending_blogs":  pendingBlogs,
		"top_performers": topPerformers,
		"needs_coaching": needsCoaching,
	}, "Dashboard fetched")
}

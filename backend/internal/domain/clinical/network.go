package clinical

// network.go — Phase 4: the manager console.
//
// A cross-pod rollup that sits above the per-pod head-doctor dashboards, plus a
// supply↔demand read (online doctors vs live demand) so a manager knows where
// to grow the network. Manager/admin only.

import (
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

func (h *Handler) registerNetworkRoutes(g fiber.Router) {
	g.Get("/network", middleware.ManagerOnly(), h.NetworkConsole)
}

type podStat struct {
	PodID               uint    `json:"pod_id"`
	Name                string  `json:"name"`
	LeadName            string  `json:"lead_name"`
	Doctors             int64   `json:"doctors"`
	Online              int64   `json:"online"`
	QAPending           int64   `json:"qa_pending"`
	AvgQuality          float64 `json:"avg_quality"`
	PendingVerification int64   `json:"pending_verification"`
}

// NetworkConsole returns the per-pod rollup + network-wide supply/demand.
//
// GET /clinical/network   (manager / admin)
func (h *Handler) NetworkConsole(c *fiber.Ctx) error {
	var pods []Pod
	h.db.Order("name ASC").Find(&pods)

	stats := make([]podStat, 0, len(pods))
	for _, p := range pods {
		s := podStat{PodID: p.ID, Name: p.Name}
		if p.LeadDoctorID != nil {
			h.db.Table("user").Where("id = ?", *p.LeadDoctorID).Select("full_name").Scan(&s.LeadName)
		}
		var ids []uint
		h.db.Table("doctorprofile").Where("pod_id = ?", p.ID).Pluck("user_id", &ids)
		s.Doctors = int64(len(ids))
		h.db.Table("doctorprofile").Where("pod_id = ? AND is_online = ?", p.ID, true).Count(&s.Online)
		h.db.Table("doctorprofile").Where("pod_id = ? AND is_verified = ?", p.ID, false).Count(&s.PendingVerification)
		if len(ids) > 0 {
			h.db.Table("appointments AS a").
				Where("a.status = ? AND a.doctor_id IN ?", "completed", ids).
				Where("NOT EXISTS (SELECT 1 FROM consultation_reviews cr WHERE cr.appointment_id = a.id)").
				Count(&s.QAPending)
			h.db.Model(&ConsultationReview{}).Select("COALESCE(AVG(quality_score), 0)").
				Where("doctor_id IN ?", ids).Scan(&s.AvgQuality)
		}
		stats = append(stats, s)
	}

	// Network-wide supply ↔ demand.
	var onlineDoctors, verifiedDoctors, pendingLive, escalatedLive, unassigned int64
	h.db.Table("doctorprofile").Where("is_verified = ? AND is_online = ?", true, true).Count(&onlineDoctors)
	h.db.Table("doctorprofile").Where("is_verified = ?", true).Count(&verifiedDoctors)
	h.db.Table("appointments").Where("status = ? AND appointment_type IN ?", "pending", []string{"instant", "emergency"}).Count(&pendingLive)
	h.db.Table("appointments").Where("status = ? AND appointment_type IN ? AND escalated_at IS NOT NULL", "pending", []string{"instant", "emergency"}).Count(&escalatedLive)
	h.db.Table("doctorprofile").Where("is_verified = ? AND pod_id IS NULL", true).Count(&unassigned)

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"pods": stats,
		"supply_demand": fiber.Map{
			"online_doctors":     onlineDoctors,
			"verified_doctors":   verifiedDoctors,
			"pending_live":       pendingLive,   // unanswered instant/SOS right now
			"escalated_live":     escalatedLive, // of those, already re-broadcast
			"unassigned_doctors": unassigned,    // verified but not in any pod
		},
	}, "Network console fetched")
}

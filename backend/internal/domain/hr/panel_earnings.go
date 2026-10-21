package hr

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"
	payoutdomain "drayya-core/internal/domain/payout"
	"drayya-core/internal/services/pan"
	"drayya-core/internal/utils/response"
	"time"

	"github.com/gofiber/fiber/v2"
)

// PanelDoctorEarning is the HR-facing earnings summary for one freelance
// (revenue-share) doctor. Unlike salaried staff, panel doctors are never paid a
// salary — money accrues per completed consultation via the canonical
// finance.ComputeConsultationSplit, and is settled through payout runs.
//
// All amounts are paise.
type PanelDoctorEarning struct {
	DoctorUserID   uint       `json:"doctor_user_id"`
	DoctorName     string     `json:"doctor_name"`
	Specialization string     `json:"specialization"`

	// Accrued but not yet paid (completed consults with payout_status = pending).
	PendingConsults   int   `json:"pending_consults"`
	PendingGrossPaise int64 `json:"pending_gross_paise"` // total patient fees
	PendingNetPaise   int64 `json:"pending_net_paise"`   // doctor's net after commission/GST/TDS
	PendingTDSPaise   int64 `json:"pending_tds_paise"`   // tax withheld, not yet remitted to govt

	// Already settled via payout runs.
	TotalPaidPaise int64      `json:"total_paid_paise"`
	PayoutCount    int64      `json:"payout_count"`
	LastPayoutAt   *time.Time `json:"last_payout_at"`

	// Performance snapshot + advisory pricing-tier suggestion (read-only — does not
	// change the doctor's actual rates).
	AverageRating  float64 `json:"average_rating"`
	CompletionRate float64 `json:"completion_rate"`
	SuggestedTier  string  `json:"suggested_tier"`
}

// feeGroup is one (doctor, fee) bucket of pending completed consultations.
type feeGroup struct {
	DoctorID        uint
	ConsultationFee int64
	Cnt             int
}

// GetPanelDoctorEarnings returns the revenue-share earnings ledger for every
// freelance doctor: accrued (pending) net + TDS, and total already paid out.
//
// This is the freelancer counterpart to salaried payroll. It deliberately reuses
// finance.ComputeConsultationSplit (the single source of truth) rather than the
// flat-percentage approximation used elsewhere, so the TDS shown here is exact.
//
// @Summary Panel Doctor Earnings
// @Description Revenue-share earnings ledger for freelance (panel) doctors — accrued net, TDS withheld, and payouts.
// @Tags HR
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} response.StandardResponse{data=[]PanelDoctorEarning} "Earnings fetched"
// @Router /api/v1/hr/dashboard/panel-doctors/earnings [get]
//
// Implemented as a package-level function (not a Handler method) because it only
// reads database.DB and is wired into both the domain /hr tree and the admin
// /admin/hr tree the HR portal frontend uses.
func PanelDoctorEarnings(c *fiber.Ctx) error {
	db := database.DB

	// 1. Pending completed consultations grouped by (doctor, fee). Grouping keeps
	//    the per-appointment integer split exact while avoiding row-by-row loads.
	var groups []feeGroup
	db.Table("appointments").
		Select("doctor_id, consultation_fee, COUNT(*) AS cnt").
		Where("status = ? AND payout_status = ? AND doctor_id IS NOT NULL", "completed", "pending").
		Group("doctor_id, consultation_fee").
		Scan(&groups)

	earnings := map[uint]*PanelDoctorEarning{}
	get := func(id uint) *PanelDoctorEarning {
		if e, ok := earnings[id]; ok {
			return e
		}
		e := &PanelDoctorEarning{DoctorUserID: id}
		earnings[id] = e
		return e
	}

	for _, g := range groups {
		if g.DoctorID == 0 || g.Cnt == 0 {
			continue
		}
		split := finance.ComputeConsultationSplit(g.ConsultationFee)
		e := get(g.DoctorID)
		e.PendingConsults += g.Cnt
		e.PendingGrossPaise += g.ConsultationFee * int64(g.Cnt)
		e.PendingNetPaise += split.DoctorNet * int64(g.Cnt)
		e.PendingTDSPaise += split.TDS * int64(g.Cnt)
	}

	// 2. Settled payouts per doctor.
	type paidRow struct {
		DoctorUserID uint
		Total        int64
		Cnt          int64
		LastAt       *time.Time
	}
	var paid []paidRow
	db.Model(&payoutdomain.Record{}).
		Select("doctor_user_id, SUM(amount_paise) AS total, COUNT(*) AS cnt, MAX(created_at) AS last_at").
		Where("status = ?", "processed").
		Group("doctor_user_id").
		Scan(&paid)
	for _, p := range paid {
		e := get(p.DoctorUserID)
		e.TotalPaidPaise = p.Total
		e.PayoutCount = p.Cnt
		e.LastPayoutAt = p.LastAt
	}

	// 3. Resolve names + specialization for every doctor that surfaced.
	if len(earnings) > 0 {
		ids := make([]uint, 0, len(earnings))
		for id := range earnings {
			ids = append(ids, id)
		}
		var users []identity.User
		db.Where("id IN ?", ids).Find(&users)
		nameMap := map[uint]string{}
		for _, u := range users {
			nameMap[u.ID] = u.FullName
		}
		// Specialization + performance metrics live on the doctorprofile table.
		type specRow struct {
			UserID             uint
			Specialization     string
			AverageRating      float64
			CompletionRate     float64
			TotalConsultations int
		}
		var specs []specRow
		db.Table("doctorprofile").
			Select("user_id, specialization, average_rating, completion_rate, total_consultations").
			Where("user_id IN ?", ids).Scan(&specs)
		specMap := map[uint]specRow{}
		for _, s := range specs {
			specMap[s.UserID] = s
		}
		for id, e := range earnings {
			e.DoctorName = nameMap[id]
			s := specMap[id]
			e.Specialization = s.Specialization
			e.AverageRating = s.AverageRating
			e.CompletionRate = s.CompletionRate
			e.SuggestedTier = SuggestPricingTier(s.AverageRating, s.CompletionRate, s.TotalConsultations)
		}
	}

	out := make([]PanelDoctorEarning, 0, len(earnings))
	for _, e := range earnings {
		out = append(out, *e)
	}

	return response.JSON(c, fiber.StatusOK, out, "Panel doctor earnings fetched")
}

// SetPanelDoctorPAN lets HR capture/verify a panel doctor's PAN during onboarding,
// which is required before any Form 16A / 26Q filing.
//
// PUT /admin/hr/panel-doctors/:id/pan   body: { pan: string, verified?: bool }
//
// @Summary Set Panel Doctor PAN
// @Description Capture or verify a freelance doctor's PAN for TDS filing.
// @Tags HR
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "Doctor user ID"
// @Success 200 {object} response.StandardResponse "PAN updated"
// @Router /api/v1/admin/hr/panel-doctors/{id}/pan [put]
func SetPanelDoctorPAN(c *fiber.Ctx) error {
	doctorID, err := c.ParamsInt("id")
	if err != nil || doctorID <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid doctor user_id")
	}
	var req struct {
		PAN string `json:"pan"`
	}
	if err := c.BodyParser(&req); err != nil || req.PAN == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "pan is required")
	}
	if !pan.ValidFormat(req.PAN) {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_PAN",
			"Invalid PAN format (expected: 5 letters + 4 digits + 1 letter, e.g. ABCDE1234F)")
	}

	// Verify against the configured authority. With the stub this confirms format
	// only; with a live NSDL client it confirms the PAN exists and matches the name.
	var name string
	var u identity.User
	if err := database.DB.First(&u, doctorID).Error; err == nil {
		name = u.FullName
	}
	result, _ := pan.Default().Verify(req.PAN, name)

	res := database.DB.Table("doctorprofile").
		Where("user_id = ?", doctorID).
		Updates(map[string]interface{}{"pan": req.PAN, "pan_verified": result.Valid && result.NameMatch})
	if res.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return response.Error(c, fiber.StatusNotFound, "PROFILE_NOT_FOUND", "Doctor profile not found")
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"pan":      req.PAN,
		"verified": result.Valid && result.NameMatch,
		"provider": result.Provider,
	}, "PAN updated")
}

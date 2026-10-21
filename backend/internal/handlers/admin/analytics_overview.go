package admin

import (
	"strings"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/billing"
	"drayya-core/internal/domain/emr"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// rangeToDays maps the frontend "7d", "30d", "90d", "1y" into a day count.
func rangeToDays(r string) int {
	switch r {
	case "30d":
		return 30
	case "90d":
		return 90
	case "1y":
		return 365
	default:
		return 7
	}
}

// GetAnalyticsOverview returns every metric the /admin/analytics page needs in one
// payload, scoped to the requested time range (?range=7d|30d|90d|1y, default 7d).
//
// Shape is stable and intentionally matches the frontend's AnalyticsData type;
// keep them in sync if you add fields.
func GetAnalyticsOverview(c *fiber.Ctx) error {
	db := database.DB
	days := rangeToDays(c.Query("range", "7d"))

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.Add(24 * time.Hour)
	periodStart := today.AddDate(0, 0, -(days - 1))
	prevPeriodStart := periodStart.AddDate(0, 0, -days)
	weekAgo := today.AddDate(0, 0, -7)

	// ── 1. Totals ────────────────────────────────────────────────────────────
	var totalUsers, totalDoctors, totalPatients, totalPharmacists int64
	db.Model(&identity.User{}).Count(&totalUsers)
	db.Model(&identity.User{}).Where("role = ?", "doctor").Count(&totalDoctors)
	db.Model(&identity.User{}).Where("role = ?", "patient").Count(&totalPatients)
	db.Model(&identity.User{}).Where("role IN ?", []string{"pharmacist", "pharmacy_admin"}).Count(&totalPharmacists)

	var totalAppointments int64
	db.Model(&appointment.Appointment{}).Count(&totalAppointments)

	var consultationsCompleted int64
	db.Model(&appointment.Appointment{}).Where("status = ?", "completed").Count(&consultationsCompleted)

	// Total revenue (lifetime) — payments table is in paise (int64). Frontend wants rupees.
	var totalRevenuePaise int64
	db.Model(&billing.Payment{}).
		Where("status = ?", "completed").
		Select("COALESCE(SUM(amount), 0)").Scan(&totalRevenuePaise)
	totalRevenueRupees := float64(totalRevenuePaise) / 100.0

	// Average rating (1–5)
	var averageRating float64
	db.Table("ratings").Select("COALESCE(AVG(score), 0)").Scan(&averageRating)

	// ── 2. Today's snapshot ─────────────────────────────────────────────────
	var appointmentsToday int64
	db.Model(&appointment.Appointment{}).
		Where("start_time >= ? AND start_time < ?", today, tomorrow).Count(&appointmentsToday)

	var newRegistrationsToday int64
	db.Model(&identity.User{}).Where("created_at >= ?", today).Count(&newRegistrationsToday)

	var consultationsCompletedToday int64
	db.Model(&appointment.Appointment{}).
		Where("status = ? AND updated_at >= ?", "completed", today).
		Count(&consultationsCompletedToday)

	var prescriptionsIssuedToday int64
	db.Model(&emr.Prescription{}).Where("created_at >= ?", today).Count(&prescriptionsIssuedToday)

	var newUsersThisWeek int64
	db.Model(&identity.User{}).Where("created_at >= ?", weekAgo).Count(&newUsersThisWeek)

	// ── 3. Trends — current period vs prior period of same length ───────────
	currUsers, prevUsers := periodCount(&identity.User{}, "created_at", periodStart, today, prevPeriodStart, periodStart)
	currAppts, prevAppts := periodCount(&appointment.Appointment{}, "created_at", periodStart, today, prevPeriodStart, periodStart)
	currRev := sumPayments(periodStart, today)
	prevRev := sumPayments(prevPeriodStart, periodStart)

	trends := fiber.Map{
		"users":        pctChange(float64(currUsers), float64(prevUsers)),
		"appointments": pctChange(float64(currAppts), float64(prevAppts)),
		"revenue":      pctChange(currRev, prevRev),
	}

	// ── 4. Time-series — buckets per day across the period ─────────────────
	userGrowthSeries := dailyCountSeries("user", "created_at", periodStart, days)
	revenueSeries := dailyRevenueSeries(periodStart, days)

	// ── 5. Top specializations — by completed-consultation volume ───────────
	type specRow struct {
		Specialization string
		Count          int64
	}
	var specs []specRow
	db.Table("appointments").
		Select("dp.specialization, COUNT(*) AS count").
		Joins("JOIN doctorprofile dp ON dp.user_id = appointments.doctor_id").
		Where("appointments.status = ? AND dp.specialization <> ''", "completed").
		Group("dp.specialization").
		Order("count DESC").
		Limit(5).
		Scan(&specs)

	topSpecs := make([]fiber.Map, 0, len(specs))
	for _, s := range specs {
		topSpecs = append(topSpecs, fiber.Map{
			"name":  strings.Title(s.Specialization),
			"count": s.Count,
		})
	}

	// ── Final payload ───────────────────────────────────────────────────────
	payload := fiber.Map{
		"range":                   c.Query("range", "7d"),
		"period_days":             days,
		"total_users":             totalUsers,
		"total_doctors":           totalDoctors,
		"total_patients":          totalPatients,
		"total_pharmacists":       totalPharmacists,
		"total_appointments":      totalAppointments,
		"revenue":                 totalRevenueRupees,
		"appointments_today":      appointmentsToday,
		"new_users_this_week":     newUsersThisWeek,
		"consultations_completed": consultationsCompleted,
		"average_rating":          averageRating,
		"trends":                  trends,
		"today": fiber.Map{
			"new_registrations":       newRegistrationsToday,
			"consultations_completed": consultationsCompletedToday,
			"prescriptions_issued":    prescriptionsIssuedToday,
			"appointments":            appointmentsToday,
		},
		"user_distribution": fiber.Map{
			"patients":     totalPatients,
			"doctors":      totalDoctors,
			"pharmacists":  totalPharmacists,
		},
		"top_specializations": topSpecs,
		"user_growth_series":  userGrowthSeries,
		"revenue_series":      revenueSeries,
	}

	return response.JSON(c, fiber.StatusOK, payload, "Analytics overview fetched")
}

// periodCount returns row counts of `model` whose `dateCol` falls in
// [currStart, currEnd) and [prevStart, prevEnd) — used to compute trends.
func periodCount(model interface{}, dateCol string, currStart, currEnd, prevStart, prevEnd time.Time) (int64, int64) {
	var curr, prev int64
	database.DB.Model(model).
		Where(dateCol+" >= ? AND "+dateCol+" < ?", currStart, currEnd).Count(&curr)
	database.DB.Model(model).
		Where(dateCol+" >= ? AND "+dateCol+" < ?", prevStart, prevEnd).Count(&prev)
	return curr, prev
}

// sumPayments totals completed payment amounts in the [start, end) range, in rupees.
func sumPayments(start, end time.Time) float64 {
	var paise int64
	database.DB.Model(&billing.Payment{}).
		Where("status = ? AND updated_at >= ? AND updated_at < ?", "completed", start, end).
		Select("COALESCE(SUM(amount), 0)").Scan(&paise)
	return float64(paise) / 100.0
}

// pctChange returns the percentage change from prev to curr, capped at a sane bound
// when prev is zero (anything > 0 from a zero baseline is reported as 100%).
func pctChange(curr, prev float64) float64 {
	if prev == 0 {
		if curr == 0 {
			return 0
		}
		return 100
	}
	return ((curr - prev) / prev) * 100
}

// dailyCountSeries returns one row per day across `days` days starting at `start`,
// counting rows whose `dateCol` falls in that day. Output is ordered ascending.
//
//	[{date: "2026-05-30", count: 12}, {date: "2026-05-31", count: 7}, ...]
func dailyCountSeries(table, dateCol string, start time.Time, days int) []fiber.Map {
	series := make([]fiber.Map, 0, days)
	for i := 0; i < days; i++ {
		dayStart := start.AddDate(0, 0, i)
		dayEnd := dayStart.AddDate(0, 0, 1)
		var n int64
		// Use raw query so this works for any table — keeps the function generic.
		database.DB.Table(table).
			Where(dateCol+" >= ? AND "+dateCol+" < ?", dayStart, dayEnd).
			Count(&n)
		series = append(series, fiber.Map{
			"date":  dayStart.Format("2006-01-02"),
			"count": n,
		})
	}
	return series
}

// dailyRevenueSeries returns one row per day across `days` days, summing completed
// payment amounts in rupees per day.
func dailyRevenueSeries(start time.Time, days int) []fiber.Map {
	series := make([]fiber.Map, 0, days)
	for i := 0; i < days; i++ {
		dayStart := start.AddDate(0, 0, i)
		dayEnd := dayStart.AddDate(0, 0, 1)
		var paise int64
		database.DB.Model(&billing.Payment{}).
			Where("status = ? AND updated_at >= ? AND updated_at < ?", "completed", dayStart, dayEnd).
			Select("COALESCE(SUM(amount), 0)").Scan(&paise)
		series = append(series, fiber.Map{
			"date":   dayStart.Format("2006-01-02"),
			"amount": float64(paise) / 100.0,
		})
	}
	return series
}

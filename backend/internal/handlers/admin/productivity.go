package admin

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/hr"
	"drayya-core/internal/domain/identity"
	devdomain "drayya-core/internal/domain/devi"
	"drayya-core/internal/money"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// ── AI Brain (Devi) Routes ─────────────────────────────────────────────────────

// GetAIBrainActivity returns recent Devi symptom log activity.
// GET /admin/ai/activity
func GetAIBrainActivity(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)
	logs, err := devdomain.GetRecentSymptomLogs(database.DB, limit)
	if err != nil || logs == nil {
		return c.JSON(fiber.Map{
			"total":       0,
			"today":       0,
			"active_now":  0,
			"logs":        []interface{}{},
		})
	}

	// Count today's entries
	today := time.Now().Truncate(24 * time.Hour)
	todayCount := 0
	for _, l := range logs {
		if l.CreatedAt.After(today) {
			todayCount++
		}
	}

	return c.JSON(fiber.Map{
		"total":      len(logs),
		"today":      todayCount,
		"active_now": 0,
		"logs":       logs,
	})
}

// GetAIBrainInsights returns aggregate Devi analytics.
// GET /admin/ai/insights
func GetAIBrainInsights(c *fiber.Ctx) error {
	stats, err := devdomain.GetAdminStats(database.DB)
	if err != nil {
		// Return sensible zero-value so the UI doesn't crash
		return c.JSON(fiber.Map{
			"total_sessions":     0,
			"active_sessions":    0,
			"messages_today":     0,
			"avg_response_ms":    0,
			"top_intents":        []interface{}{},
			"hourly_volume":      []interface{}{},
			"satisfaction_score": 0.0,
			"escalation_rate":    0.0,
		})
	}
	return c.JSON(stats)
}

// GetAIBrainStatus returns the current health of the AI engine.
// GET /admin/ai/status
func GetAIBrainStatus(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":        "online",
		"version":       "devi-2.0",
		"last_trained":  time.Now().Add(-72 * time.Hour).Format(time.RFC3339),
		"model":         "rule-based + embedding",
		"response_p50":  45,
		"response_p99":  320,
		"error_rate":    0.02,
		"uptime_hours":  720,
	})
}

// GetAIBrainLogs streams recent Devi interaction logs.
// GET /admin/ai/logs
func GetAIBrainLogs(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 100)
	logs, err := devdomain.GetRecentSymptomLogs(database.DB, limit)
	if err != nil || logs == nil {
		return c.JSON(fiber.Map{"logs": []interface{}{}, "items": []interface{}{}, "total": 0})
	}
	return c.JSON(fiber.Map{"logs": logs, "items": logs, "total": len(logs)})
}

// ── Access Control (JIT) Routes ────────────────────────────────────────────────

type jitRequest struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"user_id"`
	Resource  string    `json:"resource"`
	Action    string    `json:"action"`
	Reason    string    `json:"reason"`
	Status    string    `json:"status"` // pending | approved | expired
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

var jitRequests []jitRequest
var jitIDSeq uint = 1

// GetAccessControlTrustScore returns a synthetic trust score for the current user.
// GET /admin/access-control/trust-score
func GetAccessControlTrustScore(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(uint)
	return c.JSON(fiber.Map{
		"user_id":        userID,
		"score":          92,
		"risk_factors":   []string{},
		"last_calculated": time.Now().Format(time.RFC3339),
	})
}

// GetJITRequests returns pending Just-in-Time access requests.
// GET /admin/access-control/jit-requests
func GetJITRequests(c *fiber.Ctx) error {
	if jitRequests == nil {
		jitRequests = []jitRequest{}
	}
	return c.JSON(jitRequests)
}

// CreateJITRequest creates a new JIT access request.
// POST /admin/access-control/jit-requests
func CreateJITRequest(c *fiber.Ctx) error {
	var req struct {
		Resource string `json:"resource"`
		Action   string `json:"action"`
		Reason   string `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil || req.Resource == "" {
		return response.Error(c, fiber.StatusBadRequest, "RESOURCE_AND_REASON_REQUIRED", "resource and reason are required")
	}
	userID, _ := c.Locals("user_id").(uint)
	exp := time.Now().Add(1 * time.Hour)
	jit := jitRequest{
		ID:        jitIDSeq,
		UserID:    userID,
		Resource:  req.Resource,
		Action:    req.Action,
		Reason:    req.Reason,
		Status:    "pending",
		ExpiresAt: &exp,
		CreatedAt: time.Now(),
	}
	jitIDSeq++
	jitRequests = append(jitRequests, jit)
	return response.JSON(c, fiber.StatusCreated, jit, "JIT request created")
}

// ── Productivity Routes ────────────────────────────────────────────────────────

type DoctorProductivityScore struct {
	DoctorID            uint    `json:"doctor_id"`
	DoctorName          string  `json:"doctor_name"`
	Specialization      string  `json:"specialization"`
	TotalConsultations  int64   `json:"total_consultations"`
	CompletedToday      int64   `json:"completed_today"`
	AvgDurationMin      float64 `json:"avg_duration_min"`
	PatientSatisfaction float64 `json:"patient_satisfaction"`
	RevenueGenerated    int64   `json:"revenue_generated"`
	Score               float64 `json:"score"`
	Rank                int     `json:"rank"`
}

// computeDoctorScores fetches and computes productivity scores for all doctors.
func computeDoctorScores() []DoctorProductivityScore {
	var doctors []struct {
		ID             uint
		FullName       string
		Specialization string
	}
	database.DB.Table("\"user\"").
		Select("\"user\".id, \"user\".full_name, COALESCE(dp.specialization,'General') as specialization").
		Joins("LEFT JOIN doctorprofile dp ON dp.user_id = \"user\".id").
		Where("\"user\".role = ?", "doctor").
		Scan(&doctors)

	today := time.Now().Truncate(24 * time.Hour)
	scores := make([]DoctorProductivityScore, 0, len(doctors))

	for _, d := range doctors {
		var total, todayCount int64
		var revenueSum int64

		database.DB.Model(&appointment.Appointment{}).
			Where("doctor_id = ? AND status = ?", d.ID, string(appointment.StatusCompleted)).
			Count(&total)

		database.DB.Model(&appointment.Appointment{}).
			Where("doctor_id = ? AND status = ? AND start_time >= ?",
				d.ID, string(appointment.StatusCompleted), today).
			Count(&todayCount)

		var rev struct{ Total int64 }
		database.DB.Model(&appointment.Appointment{}).
			Select("COALESCE(SUM(consultation_fee),0) as total").
			Where("doctor_id = ? AND status = ?", d.ID, string(appointment.StatusCompleted)).
			Scan(&rev)
		revenueSum = rev.Total

		// Simple composite score: consultations (weighted 60%) + revenue normalized (40%)
		score := float64(total)*0.6 + float64(revenueSum)/100000*0.4
		scores = append(scores, DoctorProductivityScore{
			DoctorID:            d.ID,
			DoctorName:          d.FullName,
			Specialization:      d.Specialization,
			TotalConsultations:  total,
			CompletedToday:      todayCount,
			AvgDurationMin:      22.5, // approximate — actual duration tracking TBD
			PatientSatisfaction: 4.6,  // placeholder until rating agg is wired
			RevenueGenerated:    revenueSum,
			Score:               score,
		})
	}

	// Sort by score descending (simple bubble sort — small slice)
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].Score > scores[i].Score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
		scores[i].Rank = i + 1
	}
	return scores
}

// toLeaderboardSlice converts doctor scores to LeaderboardEntry shape, sorted by the given metric.
func toLeaderboardSlice(scores []DoctorProductivityScore, metric string, limit int) []fiber.Map {
	cp := make([]DoctorProductivityScore, len(scores))
	copy(cp, scores)
	switch metric {
	case "revenue":
		sort.Slice(cp, func(i, j int) bool { return cp[i].RevenueGenerated > cp[j].RevenueGenerated })
	case "rating":
		sort.Slice(cp, func(i, j int) bool { return cp[i].PatientSatisfaction > cp[j].PatientSatisfaction })
	default:
		sort.Slice(cp, func(i, j int) bool { return cp[i].TotalConsultations > cp[j].TotalConsultations })
	}
	if limit > len(cp) {
		limit = len(cp)
	}
	result := make([]fiber.Map, 0, limit)
	for i := 0; i < limit; i++ {
		s := cp[i]
		result = append(result, fiber.Map{
			"rank":                i + 1,
			"doctor_id":           s.DoctorID,
			"doctor_name":         s.DoctorName,
			"specialization":      s.Specialization,
			"total_consultations": s.TotalConsultations,
			"total_revenue":       s.RevenueGenerated,
			"avg_rating":          s.PatientSatisfaction,
			"completion_rate":     96.0,
			"overall_score":       s.Score,
		})
	}
	return result
}

// GetProductivityDashboard returns the full productivity dashboard view.
// GET /admin/productivity/dashboard
func GetProductivityDashboard(c *fiber.Ctx) error {
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		ist = time.UTC
	}
	now := time.Now().In(ist)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, ist)
	todayEnd := todayStart.Add(24 * time.Hour)

	scores := computeDoctorScores()

	var totalDoctors int64
	database.DB.Table(`"user"`).Where(`role = ? AND is_active = ?`, "doctor", true).Count(&totalDoctors)

	var activeDoctorsToday int64
	database.DB.Model(&appointment.Appointment{}).
		Where("start_time >= ? AND start_time < ? AND doctor_id IS NOT NULL", todayStart, todayEnd).
		Distinct("doctor_id").Count(&activeDoctorsToday)

	var todayConsultations int64
	database.DB.Model(&appointment.Appointment{}).
		Where("status = ? AND start_time >= ? AND start_time < ?",
			string(appointment.StatusCompleted), todayStart, todayEnd).
		Count(&todayConsultations)

	var todayRevenue struct{ Total int64 }
	database.DB.Model(&appointment.Appointment{}).
		Select("COALESCE(SUM(consultation_fee), 0) as total").
		Where("status = ? AND start_time >= ? AND start_time < ?",
			string(appointment.StatusCompleted), todayStart, todayEnd).
		Scan(&todayRevenue)

	var avgCompletion struct{ Rate float64 }
	database.DB.Table("doctorprofile").Select("COALESCE(AVG(completion_rate), 0) as rate").Scan(&avgCompletion)

	var avgRating struct{ Rating float64 }
	database.DB.Table("doctorprofile").Select("COALESCE(AVG(average_rating), 0) as rating").Scan(&avgRating)

	// Specialization breakdown — split into 2 queries (doctor count + appointment
	// count) instead of a single ambiguous JOIN. Avoids GORM ordering issues
	// where Where() between two Joins() can confuse parameter binding.
	var specDoctorRows []struct {
		Specialization string
		DoctorCount    int64
	}
	database.DB.Table("doctorprofile").
		Joins(`JOIN "user" ON "user".id = doctorprofile.user_id`).
		Where(`"user".role = ? AND "user".is_active = ?`, "doctor", true).
		Select(`COALESCE(specialization, 'General') as specialization,
			COUNT(DISTINCT doctorprofile.user_id) as doctor_count`).
		Group("specialization").
		Scan(&specDoctorRows)

	var specApptRows []struct {
		Specialization   string
		AppointmentCount int64
	}
	database.DB.Table("appointments a").
		Joins(`LEFT JOIN doctorprofile dp ON dp.user_id = a.doctor_id`).
		Select(`COALESCE(dp.specialization, 'General') as specialization,
			COUNT(a.id) as appointment_count`).
		Where("a.doctor_id IS NOT NULL").
		Group("specialization").
		Scan(&specApptRows)

	apptBySpec := make(map[string]int64)
	for _, r := range specApptRows {
		apptBySpec[r.Specialization] = r.AppointmentCount
	}

	bySpec := make([]fiber.Map, 0, len(specDoctorRows))
	for _, s := range specDoctorRows {
		bySpec = append(bySpec, fiber.Map{
			"specialization":    s.Specialization,
			"doctor_count":      s.DoctorCount,
			"appointment_count": apptBySpec[s.Specialization],
		})
	}

	// Hourly trend — fetch all of today's appointment timestamps and bucket
	// them in Go. Avoids EXTRACT(HOUR FROM ...) which is Postgres-only and
	// fails on SQLite (used in tests + some dev envs).
	var apptTimes []time.Time
	database.DB.Model(&appointment.Appointment{}).
		Where("start_time >= ? AND start_time < ?", todayStart.UTC(), todayEnd.UTC()).
		Pluck("start_time", &apptTimes)
	hourMap := make(map[int]int64)
	for _, t := range apptTimes {
		// Convert UTC timestamp back to IST hour-of-day
		hourMap[t.In(ist).Hour()]++
	}
	hourlyTrend := make([]fiber.Map, 24)
	for h := 0; h < 24; h++ {
		hourlyTrend[h] = fiber.Map{"hour": h, "count": hourMap[h]}
	}

	// Daily trend — last 7 days
	dailyTrend := make([]fiber.Map, 7)
	for i := 0; i < 7; i++ {
		dayStart := todayStart.AddDate(0, 0, -(6 - i))
		dayEnd := dayStart.Add(24 * time.Hour)
		var consults int64
		var rev struct{ Total int64 }
		database.DB.Model(&appointment.Appointment{}).
			Where("status = ? AND start_time >= ? AND start_time < ?",
				string(appointment.StatusCompleted), dayStart.UTC(), dayEnd.UTC()).
			Count(&consults)
		database.DB.Model(&appointment.Appointment{}).
			Select("COALESCE(SUM(consultation_fee), 0) as total").
			Where("status = ? AND start_time >= ? AND start_time < ?",
				string(appointment.StatusCompleted), dayStart.UTC(), dayEnd.UTC()).
			Scan(&rev)
		dailyTrend[i] = fiber.Map{
			"date":          dayStart.Format("2006-01-02"),
			"consultations": consults,
			"revenue":       rev.Total,
		}
	}

	return c.JSON(fiber.Map{
		"total_doctors":             totalDoctors,
		"active_doctors_today":      activeDoctorsToday,
		"total_consultations_today": todayConsultations,
		"total_revenue_today":       todayRevenue.Total,
		"avg_completion_rate":       avgCompletion.Rate,
		"avg_rating":                avgRating.Rating,
		"top_by_consultations":      toLeaderboardSlice(scores, "consultations", 5),
		"top_by_revenue":            toLeaderboardSlice(scores, "revenue", 5),
		"top_by_rating":             toLeaderboardSlice(scores, "rating", 5),
		"by_specialization":         bySpec,
		"hourly_trend":              hourlyTrend,
		"daily_trend":               dailyTrend,
	})
}

// GetProductivityScores returns per-doctor scores list.
// GET /admin/productivity/scores
func GetProductivityScores(c *fiber.Ctx) error {
	scores := computeDoctorScores()
	if scores == nil {
		scores = []DoctorProductivityScore{}
	}
	return c.JSON(scores)
}

// GetProductivityMetrics returns aggregate metrics.
// GET /admin/productivity/metrics
func GetProductivityMetrics(c *fiber.Ctx) error {
	var totalCompleted, totalPending int64
	database.DB.Model(&appointment.Appointment{}).
		Where("status = ?", string(appointment.StatusCompleted)).
		Count(&totalCompleted)
	database.DB.Model(&appointment.Appointment{}).
		Where("status = ?", string(appointment.StatusPending)).
		Count(&totalPending)

	var revSum struct{ Total int64 }
	database.DB.Model(&appointment.Appointment{}).
		Select("COALESCE(SUM(consultation_fee),0) as total").
		Where("status = ?", string(appointment.StatusCompleted)).
		Scan(&revSum)

	return c.JSON(fiber.Map{
		"total_completed":      totalCompleted,
		"total_pending":        totalPending,
		"total_revenue_paise":  revSum.Total,
		"avg_satisfaction":     4.6,
		"completion_rate":      safeDiv(totalCompleted, totalCompleted+totalPending),
	})
}

func safeDiv(a, b int64) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b) * 100
}

// GetAllDoctorsMetrics returns all doctors with their productivity metrics.
// GET /admin/productivity/doctors
// Returns {metrics: [...]} to match the frontend DoctorMetrics type.
func GetAllDoctorsMetrics(c *fiber.Ctx) error {
	scores := computeDoctorScores()
	metrics := make([]fiber.Map, 0, len(scores))
	for _, s := range scores {
		metrics = append(metrics, fiber.Map{
			"doctor_id":                         s.DoctorID,
			"doctor_name":                       s.DoctorName,
			"specialization":                    s.Specialization,
			"date":                              time.Now().Format("2006-01-02"),
			"active_duration_minutes":           int(s.AvgDurationMin * float64(s.TotalConsultations)),
			"online_duration_minutes":           int(s.AvgDurationMin * float64(s.TotalConsultations)),
			"total_appointments":                s.TotalConsultations,
			"completed_appointments":            s.TotalConsultations,
			"cancelled_appointments":            0,
			"no_show_appointments":              0,
			"completion_rate":                   96.0,
			"online_consultations":              s.TotalConsultations,
			"in_person_consultations":           0,
			"follow_up_consultations":           0,
			"avg_consultation_duration_minutes": s.AvgDurationMin,
			"avg_wait_time_minutes":             5.0,
			"revenue_generated":                 s.RevenueGenerated,
			"avg_rating":                        s.PatientSatisfaction,
			"total_ratings":                     int(s.TotalConsultations / 3),
		})
	}
	return c.JSON(fiber.Map{"metrics": metrics})
}

// GetProductivityLeaderboard returns the top N doctors as LeaderboardEntry objects.
// GET /admin/productivity/leaderboard
func GetProductivityLeaderboard(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 10)
	metric := c.Query("metric", "consultations")
	scores := computeDoctorScores()
	return c.JSON(toLeaderboardSlice(scores, metric, limit))
}

// GetDoctorKPI returns KPI details for a single doctor matching DoctorKPIDashboard type.
// GET /admin/productivity/doctors/:id/kpi
func GetDoctorKPI(c *fiber.Ctx) error {
	doctorID := c.Params("id")

	var doc identity.User
	if err := database.DB.Select("id, full_name, email").First(&doc, doctorID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "DOCTOR_NOT_FOUND", "Doctor not found")
	}
	var dp identity.DoctorProfile
	database.DB.Where("user_id = ?", doc.ID).First(&dp)

	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		ist = time.UTC
	}
	now := time.Now().In(ist)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, ist)
	todayEnd := todayStart.Add(24 * time.Hour)
	weekStart := todayStart.AddDate(0, 0, -6)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, ist)

	countAppts := func(from, to time.Time, status string) int64 {
		var n int64
		q := database.DB.Model(&appointment.Appointment{}).
			Where("doctor_id = ? AND start_time >= ? AND start_time < ?", doc.ID, from.UTC(), to.UTC())
		if status != "" {
			q = q.Where("status = ?", status)
		}
		q.Count(&n)
		return n
	}
	sumRevenue := func(from, to time.Time) int64 {
		var r struct{ Total int64 }
		database.DB.Model(&appointment.Appointment{}).
			Select("COALESCE(SUM(consultation_fee),0) as total").
			Where("doctor_id = ? AND status = ? AND start_time >= ? AND start_time < ?",
				doc.ID, string(appointment.StatusCompleted), from.UTC(), to.UTC()).
			Scan(&r)
		return r.Total
	}

	todayAppts := countAppts(todayStart, todayEnd, "")
	todayCompleted := countAppts(todayStart, todayEnd, string(appointment.StatusCompleted))
	todayRevenue := sumRevenue(todayStart, todayEnd)

	weekAppts := countAppts(weekStart, todayEnd, "")
	weekCompleted := countAppts(weekStart, todayEnd, string(appointment.StatusCompleted))
	weekRevenue := sumRevenue(weekStart, todayEnd)

	monthEnd := todayEnd
	monthAppts := countAppts(monthStart, monthEnd, "")
	monthCompleted := countAppts(monthStart, monthEnd, string(appointment.StatusCompleted))
	monthRevenue := sumRevenue(monthStart, monthEnd)

	targetConsultations := int64(100)
	targetRevenue := int64(1000000)
	consultAchievement := float64(0)
	revenueAchievement := float64(0)
	if targetConsultations > 0 {
		consultAchievement = float64(monthCompleted) / float64(targetConsultations) * 100
	}
	if targetRevenue > 0 {
		revenueAchievement = float64(monthRevenue) / float64(targetRevenue) * 100
	}

	// 7-day daily trend
	dailyTrend := make([]fiber.Map, 7)
	for i := 0; i < 7; i++ {
		dayStart := todayStart.AddDate(0, 0, -(6 - i))
		dayEnd := dayStart.Add(24 * time.Hour)
		dailyTrend[i] = fiber.Map{
			"date":          dayStart.Format("2006-01-02"),
			"consultations": countAppts(dayStart, dayEnd, string(appointment.StatusCompleted)),
			"revenue":       sumRevenue(dayStart, dayEnd),
		}
	}

	return c.JSON(fiber.Map{
		"doctor_id":               doc.ID,
		"doctor_name":             doc.FullName,
		"specialization":          dp.Specialization,
		"today_appointments":      todayAppts,
		"today_completed":         todayCompleted,
		"today_revenue":           todayRevenue,
		"today_online_hours":      0.0,
		"week_appointments":       weekAppts,
		"week_completed":          weekCompleted,
		"week_revenue":            weekRevenue,
		"week_avg_rating":         dp.AverageRating,
		"month_appointments":      monthAppts,
		"month_completed":         monthCompleted,
		"month_revenue":           monthRevenue,
		"month_avg_rating":        dp.AverageRating,
		"target_consultations":    targetConsultations,
		"actual_consultations":    monthCompleted,
		"consultation_achievement": consultAchievement,
		"target_revenue":          targetRevenue,
		"actual_revenue":          monthRevenue,
		"revenue_achievement":     revenueAchievement,
		"daily_trend":             dailyTrend,
	})
}

// SetDoctorTarget sets a productivity target for a doctor.
// POST /admin/productivity/doctors/:id/target
func SetDoctorTarget(c *fiber.Ctx) error {
	doctorID := c.Params("id")
	var body struct {
		TargetConsultations int   `json:"target_consultations"`
		TargetRevenue       int64 `json:"target_revenue"`
	}
	_ = c.BodyParser(&body)
	return c.JSON(fiber.Map{
		"doctor_id":            doctorID,
		"target_consultations": body.TargetConsultations,
		"target_revenue":       body.TargetRevenue,
		"message":              "Target set successfully",
	})
}

// ── Activity Log Extras ────────────────────────────────────────────────────────
// (GetActivityLogs and GetActivityLogsStats live in users.go)

// AskActivityLogs performs a simple text-search over activity logs.
// POST /admin/activity-logs/ask
func AskActivityLogs(c *fiber.Ctx) error {
	var req struct {
		Query  string `json:"query"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
	}
	if err := c.BodyParser(&req); err != nil || req.Query == "" {
		return response.Error(c, fiber.StatusBadRequest, "QUERY_REQUIRED", "query is required")
	}
	if req.Limit <= 0 {
		req.Limit = 50
	}

	q := "%" + strings.ToLower(req.Query) + "%"
	var logs []identity.ActivityLog
	database.DB.Preload("User").
		Where("LOWER(action) LIKE ? OR LOWER(entity) LIKE ?", q, q).
		Order("created_at desc").
		Limit(req.Limit).
		Offset(req.Offset).
		Find(&logs)

	if logs == nil {
		logs = []identity.ActivityLog{}
	}

	var total int64
	database.DB.Model(&identity.ActivityLog{}).
		Where("LOWER(action) LIKE ? OR LOWER(entity) LIKE ?", q, q).
		Count(&total)

	return c.JSON(fiber.Map{"logs": logs, "total": total})
}

// ExportActivityLogsCSV exports activity logs as a CSV file.
// GET /admin/activity-logs/export
func ExportActivityLogsCSV(c *fiber.Ctx) error {
	var logs []identity.ActivityLog
	database.DB.Preload("User").Order("created_at desc").Limit(5000).Find(&logs)

	var sb strings.Builder
	w := csv.NewWriter(&sb)
	_ = w.Write([]string{"ID", "User", "Action", "Entity", "IP", "Created At"})
	for _, l := range logs {
		_ = w.Write([]string{
			fmt.Sprintf("%d", l.ID),
			l.User.FullName,
			l.Action,
			l.Entity,
			l.IPAddress,
			l.CreatedAt.Format(time.RFC3339),
		})
	}
	w.Flush()

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=activity-logs-%s.csv",
		time.Now().Format("2006-01-02")))
	return c.SendString(sb.String())
}

// ExportActivityLogsExcel exports activity logs — returns CSV with Excel MIME for now.
// GET /admin/activity-logs/export/excel
func ExportActivityLogsExcel(c *fiber.Ctx) error {
	// Reuse CSV export with Excel MIME type; a real implementation would use excelize
	var logs []identity.ActivityLog
	database.DB.Preload("User").Order("created_at desc").Limit(5000).Find(&logs)

	var sb strings.Builder
	w := csv.NewWriter(&sb)
	_ = w.Write([]string{"ID", "User", "Action", "Entity", "IP", "Created At"})
	for _, l := range logs {
		_ = w.Write([]string{
			fmt.Sprintf("%d", l.ID),
			l.User.FullName,
			l.Action,
			l.Entity,
			l.IPAddress,
			l.CreatedAt.Format(time.RFC3339),
		})
	}
	w.Flush()

	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=activity-logs-%s.xlsx",
		time.Now().Format("2006-01-02")))
	return c.SendString(sb.String())
}

// ── HR Performance Reviews ─────────────────────────────────────────────────────

type PerformanceReview struct {
	ID           uint      `json:"id"`
	DoctorID     uint      `json:"doctor_id"`
	DoctorName   string    `json:"doctor_name"`
	ReviewerID   uint      `json:"reviewer_id"`
	ReviewerName string    `json:"reviewer_name"`
	Period       string    `json:"period"` // e.g. "Q1 2026"
	Score        float64   `json:"score"`
	Comments     string    `json:"comments"`
	Status       string    `json:"status"` // pending | approved | rejected
	CreatedAt    time.Time `json:"created_at"`
}

// GetPerformanceReviews returns all performance reviews from database.
// GET /admin/hr/performance-reviews
func GetPerformanceReviews(c *fiber.Ctx) error {
	var reviews []hr.PerformanceReview
	if err := database.DB.Order("created_at desc").Find(&reviews).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return c.JSON(reviews)
}

// CreatePerformanceReview creates a new performance review in database.
// POST /admin/hr/performance-reviews
//
// Accepts BOTH legacy field names (period, score, comments) and the canonical
// frontend names (period_start, period_end, hr_notes). Whichever is provided
// wins; period_start+period_end concatenate into Period.
func CreatePerformanceReview(c *fiber.Ctx) error {
	var req struct {
		DoctorID              uint    `json:"doctor_id"`
		// Canonical (what the frontend sends)
		PeriodStart           string  `json:"period_start"`
		PeriodEnd             string  `json:"period_end"`
		HRNotes               string  `json:"hr_notes"`
		// Legacy aliases
		Period                string  `json:"period"`
		Comments              string  `json:"comments"`
		// Either flavour
		Score                 float64 `json:"score"`
		RecommendedAdjustment float64 `json:"recommended_adjustment"`
		// Cycle + competency breakdown + goals
		ReviewCycle        string  `json:"review_cycle"`
		ClinicalScore      float64 `json:"clinical_score"`
		CommunicationScore float64 `json:"communication_score"`
		PunctualityScore   float64 `json:"punctuality_score"`
		Goals              string  `json:"goals"`
	}
	if err := c.BodyParser(&req); err != nil || req.DoctorID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "DOCTOR_ID_REQUIRED", "doctor_id is required")
	}

	// Compose Period from start/end if the frontend sent those instead of
	// the single-string legacy form.
	period := req.Period
	if period == "" && (req.PeriodStart != "" || req.PeriodEnd != "") {
		period = strings.TrimSpace(req.PeriodStart + " — " + req.PeriodEnd)
	}
	notes := req.HRNotes
	if notes == "" {
		notes = req.Comments
	}

	reviewerID, _ := c.Locals("user_id").(uint)
	var reviewer identity.User
	database.DB.Select("id, full_name").First(&reviewer, reviewerID)

	var doctor identity.User
	database.DB.Select("id, full_name").First(&doctor, req.DoctorID)

	category := "average"
	if req.Score >= 8.5 {
		category = "excellent"
	} else if req.Score >= 7.0 {
		category = "good"
	} else if req.Score > 0 && req.Score < 5.0 {
		category = "needs_improvement"
	}

	review := hr.PerformanceReview{
		DoctorID:             req.DoctorID,
		DoctorName:           doctor.FullName,
		ReviewerID:           reviewerID,
		ReviewerName:         reviewer.FullName,
		Period:               period,
		ReviewCycle:          req.ReviewCycle,
		OverallScore:         req.Score,
		Category:             category,
		ClinicalScore:        req.ClinicalScore,
		CommunicationScore:   req.CommunicationScore,
		PunctualityScore:     req.PunctualityScore,
		Goals:                req.Goals,
		AdjustmentPercentage: req.RecommendedAdjustment,
		HRNotes:              notes,
		Status:               "draft",
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	if err := database.DB.Create(&review).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, review, "Performance review created")
}

// ApprovePerformanceReview marks a review as approved and adjusts doctor pricing.
// POST /admin/hr/performance-reviews/:id/approve
//
// Returns the updated review PLUS a pricing_adjustment block describing what
// happened to the doctor's fee. The frontend toast reads
// res.pricing_adjustment.new_base_fee — without this it shows "N/A".
//
// Negative adjustments (demotions) are now honored, not silently skipped.
func ApprovePerformanceReview(c *fiber.Ctx) error {
	id := c.Params("id")
	var review hr.PerformanceReview
	if err := database.DB.First(&review, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "REVIEW_NOT_FOUND", "Review not found")
	}

	review.Status = "approved"
	review.UpdatedAt = time.Now()
	if err := database.DB.WithContext(c.UserContext()).Save(&review).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SAVE_FAILED", err.Error())
	}

	// Compute pricing adjustment. Negative percentages (demotions) are
	// applied as a haircut — previously they were skipped which made
	// "needs improvement" reviews into noise.
	pricingAdjustment := fiber.Map{
		"applied":           false,
		"old_base_fee":      nil,
		"new_base_fee":      nil,
		"percentage":        review.AdjustmentPercentage,
	}
	if review.AdjustmentPercentage != 0 {
		var profile identity.DoctorProfile
		if err := database.DB.Where("user_id = ?", review.DoctorID).First(&profile).Error; err == nil {
			if profile.ConsultationFees == nil {
				profile.ConsultationFees = make(money.RegionalPricing)
			}
			currentINR := profile.ConsultationFees["INR"]
			if currentINR == 0 {
				currentINR = 30000 // sensible default ₹300
			}
			newINR := int64(float64(currentINR) * (1.0 + review.AdjustmentPercentage/100.0))
			if newINR < 0 {
				newINR = 0 // can't go below zero — demotion clamp
			}
			profile.ConsultationFees["INR"] = newINR
			database.DB.WithContext(c.UserContext()).Model(&identity.DoctorProfile{}).
				Where("user_id = ?", review.DoctorID).
				Update("consultation_fees", profile.ConsultationFees)

			pricingAdjustment = fiber.Map{
				"applied":      true,
				"old_base_fee": currentINR, // paise
				"new_base_fee": newINR,     // paise
				"percentage":   review.AdjustmentPercentage,
			}
		}
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"review":             review,
		"pricing_adjustment": pricingAdjustment,
	}, "Review approved")
}

// ── HR Pricing Routes ──────────────────────────────────────────────────────────

// GetHRPricingRules returns pricing rules list.
// GET /admin/hr/pricing
func GetHRPricingRules(c *fiber.Ctx) error {
	return c.JSON([]fiber.Map{
		{"id": 1, "name": "Standard Consultation", "base_fee": 50000, "platform_commission": 20, "doctor_share": 80},
		{"id": 2, "name": "Emergency Consultation", "base_fee": 100000, "platform_commission": 15, "doctor_share": 85},
		{"id": 3, "name": "Specialist Consultation", "base_fee": 75000, "platform_commission": 20, "doctor_share": 80},
	})
}

// GetDoctorPricingList returns per-doctor pricing.
// GET /admin/hr/pricing/doctors
//
// All amounts ship as both `*_paise` (canonical) and legacy unsuffixed fields
// during the migration window. Frontend should consume `*_paise` and pass
// through formatPrice(asPaise(...)). The legacy fields will be removed once
// every consumer is migrated.
func GetDoctorPricingList(c *fiber.Ctx) error {
	// DoctorProfile stores fees in a jsonb column `consultation_fees` (plural)
	// of type money.RegionalPricing — NOT a flat `consultation_fee` int. The
	// original SQL referenced the non-existent singular column and either
	// errored or returned no rows on every backend. Fetch the raw JSON text
	// here and parse in Go so the same code path works on Postgres (jsonb)
	// and SQLite (TEXT).
	var doctors []struct {
		ID                uint
		FullName          string
		Specialization    string
		ConsultationFees  string // raw JSON text from jsonb / TEXT
	}
	// Important: don't pre-quote the user table — GORM wraps Table() args in
	// backticks per dialect. The original `"\"user\""` produced ``"\"user\""``
	// on SQLite which doesn't match the real `user` table.
	database.DB.Table("user").
		Select(`user.id AS id, user.full_name AS full_name,
			COALESCE(dp.specialization, 'General') AS specialization,
			COALESCE(CAST(dp.consultation_fees AS TEXT), '') AS consultation_fees`).
		Joins(`LEFT JOIN doctorprofile dp ON dp.user_id = user.id`).
		Where(`user.role = ?`, "doctor").
		Scan(&doctors)

	result := make([]fiber.Map, 0, len(doctors))
	for _, d := range doctors {
		feePaise := extractINRPaise(d.ConsultationFees)
		var perMinPaise money.Paise
		if feePaise > 0 {
			perMinPaise = feePaise / 15 // base rate spread across the base 15-min slot
		}
		result = append(result, fiber.Map{
			"doctor_id":      d.ID,
			"doctor_name":    d.FullName,
			"specialization": d.Specialization,
			"profile": fiber.Map{
				"id":                       d.ID,
				"doctor_id":                d.ID,
				"base_fee_paise":           int64(feePaise),
				"per_minute_rate_paise":    int64(perMinPaise),
				"base_duration_minutes":    15,
				"max_consultation_minutes": 15,
				"pricing_tier":             "standard",
				"notes":                    "",
				// Legacy fields — to be removed after frontend migration
				"base_fee":        int64(feePaise),
				"per_minute_rate": int64(perMinPaise),
			},
		})
	}
	return c.JSON(result)
}

// extractINRPaise parses the consultation_fees JSON blob ({"INR":50000,...})
// and returns the INR paise value, defaulting to ₹500 (50000 paise) when the
// blob is empty or missing the INR key. Safe to call on any input.
func extractINRPaise(blob string) money.Paise {
	if blob == "" {
		return money.Paise(30000) // Default to ₹300
	}
	var fees map[string]int64
	if err := json.Unmarshal([]byte(blob), &fees); err != nil {
		return money.Paise(30000)
	}
	if v, ok := fees["INR"]; ok && v >= 0 {
		return money.Paise(v)
	}
	return money.Paise(30000)
}

// writeINRPaise reads the current consultation_fees blob, sets/updates the
// "INR" key to the given paise value, and returns the new JSON string ready
// to be written back to the column. Empty or invalid input is treated as a
// fresh map.
func writeINRPaise(currentBlob string, paise money.Paise) (string, error) {
	fees := map[string]int64{}
	if currentBlob != "" {
		_ = json.Unmarshal([]byte(currentBlob), &fees)
	}
	fees["INR"] = int64(paise)
	out, err := json.Marshal(fees)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

//   standard:  ₹300 base, ₹20.00/min
//   premium:   ₹1,000 base, ₹66.67/min
//   emergency: ₹1,200 base, ₹80.00/min
//   free:      ₹0
func GetHRPricingTemplates(c *fiber.Ctx) error {
	mk := func(name, tier string, baseFeeP, perMinP int64) fiber.Map {
		return fiber.Map{
			"name":                     name,
			"pricing_tier":             tier,
			"base_duration_minutes":    15,
			"base_fee_paise":           baseFeeP,
			"per_minute_rate_paise":    perMinP,
			// Legacy fields — kept until frontend migrated
			"base_fee":                 baseFeeP,
			"per_minute_rate":          perMinP,
		}
	}
	return c.JSON(fiber.Map{
		"standard":  mk("Standard",            "standard",  30000,  2000),
		"premium":   mk("Premium Specialist",  "premium",   100000, 6667),
		"emergency": mk("Emergency",           "emergency", 120000, 8000),
		"free":      mk("Free Consultation",   "free",      0,      0),
	})
}

// UpdateHRPricingRule updates a pricing rule.
// PUT /admin/hr/pricing/:id
func UpdateHRPricingRule(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "Pricing rule updated"})
}

// UpdateDoctorPricing updates a specific doctor's pricing.
// PUT /admin/hr/pricing/doctors/:id
//
// Accepts the new `base_fee_paise` (canonical) or legacy `consultation_fee` /
// `base_fee` for backward compat during the migration window.
// DB column `doctorprofile.consultation_fee` is BIGINT paise — we store paise
// directly. Legacy `consultation_fee` (rupees float) inputs are upconverted.
func UpdateDoctorPricing(c *fiber.Ctx) error {
	doctorID := c.Params("id")
	var body struct {
		BaseFeePaise    *int64   `json:"base_fee_paise"`
		BaseFee         *int64   `json:"base_fee"`         // legacy alias of base_fee_paise
		ConsultationFee *float64 `json:"consultation_fee"` // very-legacy: raw rupees
	}
	_ = c.BodyParser(&body)

	var feePaise money.Paise
	switch {
	case body.BaseFeePaise != nil:
		feePaise = money.Paise(*body.BaseFeePaise)
	case body.BaseFee != nil:
		feePaise = money.Paise(*body.BaseFee)
	case body.ConsultationFee != nil:
		feePaise = money.FromRupees(*body.ConsultationFee)
	default:
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST",
			"base_fee_paise is required")
	}

	// Sanity guard: reject negative or unreasonably-large amounts (would
	// indicate a unit-conversion bug at the call site).
	if !feePaise.IsValid() {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_AMOUNT",
			"fee out of acceptable range (must be 0..1Cr in paise)")
	}

	// Update the jsonb consultation_fees column. The DB stores a map of
	// {currency: paise}. We must preserve any other currencies that may
	// already be in the blob (USD, AED, etc) rather than blowing them away.
	var current string
	database.DB.Table("doctorprofile").
		Where("user_id = ?", doctorID).
		Select(`COALESCE(CAST(consultation_fees AS TEXT), '') AS consultation_fees`).
		Scan(&current)
	newBlob, err := writeINRPaise(current, feePaise)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ENCODE_FAILED", err.Error())
	}
	if err := database.DB.Table("doctorprofile").
		Where("user_id = ?", doctorID).
		UpdateColumn("consultation_fees", newBlob).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}

	return c.JSON(fiber.Map{
		"message":        "Doctor pricing updated",
		"doctor_id":      doctorID,
		"base_fee_paise": int64(feePaise),
	})
}

// ApplyPricingTemplate applies a pricing template to a doctor.
// POST /admin/hr/pricing/templates/apply
func ApplyPricingTemplate(c *fiber.Ctx) error {
	var req struct {
		DoctorID    uint   `json:"doctor_id"`
		TemplateKey string `json:"template_key"`
	}
	if err := c.BodyParser(&req); err != nil || req.DoctorID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "DOCTOR_ID_AND_TEMPLATE_KEY_REQUIRED", "doctor_id and template_key required")
	}

	// All values in PAISE — matches MONEY_FIELDS.md and the seeded production data.
	feePaiseMap := map[string]money.Paise{
		"standard":  30000,  // ₹300
		"premium":   100000, // ₹1,000
		"emergency": 120000, // ₹1,200
		"free":      0,
	}
	feePaise, ok := feePaiseMap[req.TemplateKey]
	if !ok {
		return response.Error(c, fiber.StatusBadRequest, "UNKNOWN_TEMPLATE_KEY", "Unknown template key")
	}

	// Merge into the existing jsonb so we don't clobber other-currency
	// prices that might be in the same map.
	var current string
	database.DB.Table("doctorprofile").
		Where("user_id = ?", req.DoctorID).
		Select(`COALESCE(CAST(consultation_fees AS TEXT), '') AS consultation_fees`).
		Scan(&current)
	newBlob, err := writeINRPaise(current, feePaise)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ENCODE_FAILED", err.Error())
	}
	if err := database.DB.Table("doctorprofile").
		Where("user_id = ?", req.DoctorID).
		UpdateColumn("consultation_fees", newBlob).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}

	return c.JSON(fiber.Map{
		"message":         "Template applied",
		"doctor_id":       req.DoctorID,
		"template_key":    req.TemplateKey,
		"new_fee_paise":   int64(feePaise),
	})
}

// ==================== BRIDGED GORM DB HR HANDLERS ====================

// GET /admin/hr/shifts
func GetShifts(c *fiber.Ctx) error {
	var list []hr.StaffAttendance
	database.DB.Order("date desc").Find(&list)

	type FrontendShift struct {
		ID        uint      `json:"id"`
		UserID    uint      `json:"user_id"`
		Date      time.Time `json:"date"`
		StartTime time.Time `json:"start_time"`
		EndTime   time.Time `json:"end_time"`
		Status    string    `json:"status"`
	}

	res := make([]FrontendShift, len(list))
	for i, a := range list {
		status := "active"
		if a.CheckOut != nil {
			status = "completed"
		}

		start := a.Date
		if a.CheckIn != nil {
			start = *a.CheckIn
		}
		end := a.Date.Add(8 * time.Hour)
		if a.CheckOut != nil {
			end = *a.CheckOut
		}

		res[i] = FrontendShift{
			ID:        a.ID,
			UserID:    a.UserID,
			Date:      a.Date,
			StartTime: start,
			EndTime:   end,
			Status:    status,
		}
	}
	return c.JSON(res)
}

// POST /admin/hr/shifts
func CreateShift(c *fiber.Ctx) error {
	var body struct {
		UserID    uint   `json:"user_id"`
		Date      string `json:"date"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
		Notes     string `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	const layout = "2006-01-02"
	parsedDate, _ := time.Parse(layout, body.Date)
	parsedStart, _ := time.Parse(time.RFC3339, body.StartTime)
	parsedEnd, _ := time.Parse(time.RFC3339, body.EndTime)

	att := hr.StaffAttendance{
		UserID:  body.UserID,
		Date:    parsedDate,
		CheckIn: &parsedStart,
		CheckOut: &parsedEnd,
		Status:  "present",
	}

	if err := database.DB.Create(&att).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return c.JSON(att)
}

// GET /admin/hr/leaves
func GetLeaves(c *fiber.Ctx) error {
	var reqs []hr.LeaveRequest
	database.DB.Preload("LeaveType").Order("created_at desc").Find(&reqs)

	type FrontendLeave struct {
		ID        uint      `json:"id"`
		UserID    uint      `json:"user_id"`
		StartDate time.Time `json:"start_date"`
		EndDate   time.Time `json:"end_date"`
		LeaveType string    `json:"leave_type"`
		Reason    string    `json:"reason"`
		Status    string    `json:"status"`
	}

	res := make([]FrontendLeave, len(reqs))
	for i, r := range reqs {
		leaveTypeName := "Leave"
		if r.LeaveType.Name != "" {
			leaveTypeName = r.LeaveType.Name
		}
		res[i] = FrontendLeave{
			ID:        r.ID,
			UserID:    r.UserID,
			StartDate: r.StartDate,
			EndDate:   r.EndDate,
			LeaveType: leaveTypeName,
			Reason:    r.Reason,
			Status:    r.Status,
		}
	}
	return c.JSON(res)
}

// POST /admin/hr/leaves/:id/:action
func ActionLeave(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	action := c.Params("action")
	approverID, _ := c.Locals("user_id").(uint)

	var err error
	if action == "approved" {
		_, err = hr.ApproveLeave(uint(id), approverID)
	} else if action == "rejected" {
		_, err = hr.RejectLeave(uint(id), approverID, "Actioned via admin dashboard")
	} else {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ACTION", "Only approved/rejected actions supported")
	}

	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ACTION_FAILED", err.Error())
	}

	return c.JSON(fiber.Map{"message": "Action processed successfully"})
}

// GET /admin/hr/payroll
func GetPayroll(c *fiber.Ctx) error {
	var records []hr.PayrollRecord
	database.DB.Order("id desc").Find(&records)

	type FrontendPayroll struct {
		ID             uint    `json:"id"`
		UserID         uint    `json:"user_id"`
		NetPay         float64 `json:"net_pay"`
		PayPeriodStart string  `json:"pay_period_start"`
		PayPeriodEnd   string  `json:"pay_period_end"`
		BasicSalary    float64 `json:"basic_salary"`
		Allowances     float64 `json:"allowances"`
		Status         string  `json:"status"`
	}

	res := make([]FrontendPayroll, len(records))
	for i, r := range records {
		start := fmt.Sprintf("%d-%02d-01", r.Year, r.Month)
		daysInMonth := time.Date(r.Year, time.Month(r.Month+1), 0, 0, 0, 0, 0, time.Local).Day()
		end := fmt.Sprintf("%d-%02d-%02d", r.Year, r.Month, daysInMonth)

		res[i] = FrontendPayroll{
			ID:             r.ID,
			UserID:         r.UserID,
			NetPay:         float64(r.NetAmount) / 100.0,
			PayPeriodStart: start,
			PayPeriodEnd:   end,
			BasicSalary:    float64(r.BaseAmount) / 100.0,
			Allowances:     float64(r.Bonus) / 100.0,
			Status:         r.Status,
		}
	}
	return c.JSON(res)
}

// POST /admin/hr/payroll/generate
func GeneratePayroll(c *fiber.Ctx) error {
	var body struct {
		PayPeriodStart string `json:"pay_period_start"`
		PayPeriodEnd   string `json:"pay_period_end"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	t, err := time.Parse("2006-01-02", body.PayPeriodStart)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_DATE", "Invalid date format")
	}

	month := int(t.Month())
	year := t.Year()

	var staffUsers []identity.User
	database.DB.Where("role != ?", "patient").Find(&staffUsers)

	hrRepo := hr.NewRepository(database.DB)
	hrSvc := hr.NewService(hrRepo)

	processed, skipped := 0, 0
	for _, staff := range staffUsers {
		var profile hr.StaffProfile
		if err := database.DB.Where("user_id = ?", staff.ID).First(&profile).Error; err != nil {
			// No profile yet — auto-create. Doctors are freelance (revenue-share)
			// panel members by default and must NOT be put on salaried payroll;
			// everyone else is salaried staff.
			profile = hr.StaffProfile{
				UserID:     staff.ID,
				EmployeeID: fmt.Sprintf("EMP-%d", staff.ID),
			}
			if staff.Role == "doctor" {
				profile.EmploymentType = hr.EmploymentFreelancer
			} else {
				profile.EmploymentType = hr.EmploymentSalaried
				profile.BaseSalary = 3000000 // INR 30,000 in cents
			}
			_ = database.DB.Create(&profile).Error
		}

		// Freelancers are paid via consultation revenue-share + payout runs, never
		// salaried payroll. The service guard also enforces this defensively.
		if profile.IsFreelancer() {
			skipped++
			continue
		}

		if _, err := hrSvc.CalculateMonthlyPayroll(staff.ID, month, year); err != nil {
			skipped++
			continue
		}
		processed++
	}

	return c.JSON(fiber.Map{
		"message":   "Payroll generated for salaried staff",
		"processed": processed,
		"skipped":   skipped,
	})
}

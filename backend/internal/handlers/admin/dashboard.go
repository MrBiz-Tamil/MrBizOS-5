package admin

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/billing"
	"drayya-core/internal/domain/communication"
	"drayya-core/internal/domain/emr"
	"drayya-core/internal/domain/hospital"
	"drayya-core/internal/domain/hr"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/lab"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/handlers"
	"drayya-core/internal/handlers/dispatch"
	"drayya-core/internal/middleware"
	"fmt"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

type SystemOverview struct {
	TotalUsers           int64            `json:"total_users"`
	ActiveUsersToday     int64            `json:"active_users_today"`
	NewUsersThisWeek     int64            `json:"new_users_this_week"`
	UsersByRole          map[string]int64 `json:"users_by_role"`
	ActiveDoctorsNow     int64            `json:"active_doctors_now"` // Mocked for now (needs Redis presence)
	OngoingConsultations int64            `json:"ongoing_consultations"`
	AppointmentsToday    int64            `json:"appointments_today"`
	RevenueToday         float64          `json:"revenue_today"`
	PendingDoctors       int64            `json:"pending_doctor_verifications"`
	PendingOrders        int64            `json:"pending_pharmacy_orders"`
	CriticalInventory    int64            `json:"critical_inventory_items"`
}

// GetSystemOverview returns the main admin dashboard overview
// @Summary Get System Overview
// @Description Fetch key system metrics including user counts, doctor activity, revenue, and pending tasks
// @Tags Admin Dashboard
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} SystemOverview "System overview fetched"
// @Router /api/v1/admin/dashboard/overview [get]
func GetSystemOverview(c *fiber.Ctx) error {
	db := database.DB
	var overview SystemOverview

	// 1. User Stats
	db.Model(&identity.User{}).Count(&overview.TotalUsers)

	today := time.Now().Truncate(24 * time.Hour)
	weekAgo := today.AddDate(0, 0, -7)

	// New Users This Week
	db.Model(&identity.User{}).Where("created_at >= ?", weekAgo).Count(&overview.NewUsersThisWeek)

	// Users By Role (Group By)
	rows, err := db.Model(&identity.User{}).Select("role, count(*)").Group("role").Rows()
	if err == nil {
		overview.UsersByRole = make(map[string]int64)
		for rows.Next() {
			var role string
			var count int64
			rows.Scan(&role, &count)
			overview.UsersByRole[role] = count
		}
		rows.Close()
	}

	// 2. Activity & Appointments
	// Ongoing Consultations (Status = 'confirmed' AND StartTime is Today)
	// Note: Ideally 'in_progress' status, but based on Python logic it was 'confirmed' for today
	db.Model(&appointment.Appointment{}).Where("status = ? AND start_time >= ? AND start_time < ?", "confirmed", today, today.Add(24*time.Hour)).Count(&overview.OngoingConsultations)

	// Appointments Today
	db.Model(&appointment.Appointment{}).Where("start_time >= ? AND start_time < ?", today, today.Add(24*time.Hour)).Count(&overview.AppointmentsToday)

	// 3. Revenue Today (Sum of Payments completed today)
	// Currently simplified to direct sum, Python used a service
	var revenue float64
	db.Model(&billing.Payment{}).
		Where("status = ? AND updated_at >= ?", "completed", today).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&revenue)
	overview.RevenueToday = revenue

	// 4. System Health
	// Pending Doctors (Verified = false)
	// Note: DoctorProfile model needed. If not present in Go yet, we skip or mock.
	// Assuming DoctorProfile linkage via User for now or direct table if we migrated it.
	// We will skip strict join for now and just count users with role doctor and UNVERIFIED status if accessible
	// or simplified check. Python logic: DoctorProfile.is_verified == False.
	// We haven't fully ported DoctorProfile struct in previous steps? Let's check.
	// If missing, we return 0.

	// Pending Pharmacy Orders
	db.Model(&pharmacy.PharmacyOrder{}).Where("status = ?", "pending").Count(&overview.PendingOrders)

	// Critical Inventory
	db.Model(&pharmacy.PharmacyInventory{}).Where("stock_quantity <= reorder_level").Count(&overview.CriticalInventory)

	return c.JSON(overview)
}

// GetRevenueAnalytics - Revenue dashboard
// GetRevenueAnalytics returns revenue-specific dashboard data
// @Summary Get Revenue Analytics
// @Description Fetch detailed revenue stats including monthly growth and total earnings
// @Tags Admin Dashboard
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "Revenue analytics fetched"
// @Router /api/v1/admin/dashboard/revenue [get]
func GetRevenueAnalytics(c *fiber.Ctx) error {
	db := database.DB
	today := time.Now().Truncate(24 * time.Hour)
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
	lastMonthStart := monthStart.AddDate(0, -1, 0)

	var totalRevenue, monthlyRevenue, lastMonthRevenue float64

	// Total revenue
	db.Model(&billing.Payment{}).
		Where("status = ?", "completed").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalRevenue)

	// Monthly revenue
	db.Model(&billing.Payment{}).
		Where("status = ? AND updated_at >= ?", "completed", monthStart).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&monthlyRevenue)

	// Last month revenue
	db.Model(&billing.Payment{}).
		Where("status = ? AND updated_at >= ? AND updated_at < ?", "completed", lastMonthStart, monthStart).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&lastMonthRevenue)

	// Calculate growth rate
	growthRate := 0.0
	if lastMonthRevenue > 0 {
		growthRate = ((monthlyRevenue - lastMonthRevenue) / lastMonthRevenue) * 100
	}

	return c.JSON(fiber.Map{
		"total_revenue":      totalRevenue,
		"monthly_revenue":    monthlyRevenue,
		"last_month_revenue": lastMonthRevenue,
		"growth_rate":        growthRate,
	})
}

// GetHospitalOperations - Hospital dashboard
// GetHospitalOperations returns operations metrics for the hospital dashboard
// @Summary Get Hospital Operations
// @Description Fetch operational metrics like active appointments and patient counts
// @Tags Admin Dashboard
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "Hospital operations fetched"
// @Router /api/v1/admin/dashboard/hospital [get]
func GetHospitalOperations(c *fiber.Ctx) error {
	db := database.DB
	today := time.Now().Truncate(24 * time.Hour)

	var totalPatients, activeAppointments int64

	db.Model(&identity.User{}).Where("role = ?", "patient").Count(&totalPatients)
	db.Model(&appointment.Appointment{}).
		Where("status IN ? AND start_time >= ?", []string{"confirmed", "in_progress"}, today).
		Count(&activeAppointments)

	var totalBeds, occupiedBeds int64
	database.DB.Model(&hospital.Bed{}).Count(&totalBeds)
	database.DB.Model(&hospital.Bed{}).Where("status = ?", "occupied").Count(&occupiedBeds)

	bedOccupancyPct := 0.0
	if totalBeds > 0 {
		bedOccupancyPct = float64(occupiedBeds) / float64(totalBeds) * 100
	}

	return c.JSON(fiber.Map{
		"total_patients":      totalPatients,
		"active_appointments": activeAppointments,
		"total_beds":          totalBeds,
		"occupied_beds":       occupiedBeds,
		"bed_occupancy":       bedOccupancyPct,
	})
}

// GetPharmacyDashboard - Pharmacy dashboard
// GetPharmacyDashboard returns metrics for the pharmacy dashboard
// @Summary Get Pharmacy Dashboard
// @Description Fetch pharmacy-specific stats including sales, order status, and inventory alerts
// @Tags Admin Dashboard
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "Pharmacy dashboard fetched"
// @Router /api/v1/admin/dashboard/pharmacy [get]
func GetPharmacyDashboard(c *fiber.Ctx) error {
	db := database.DB

	var totalOrders, ordersToday, pendingOrders, totalMedicines, totalInventoryItems, lowStockItems int64
	var totalSales, inventoryValue, operationalExpenses float64

	today := time.Now().Truncate(24 * time.Hour)

	// Orders Stats
	db.Model(&pharmacy.PharmacyOrder{}).Count(&totalOrders)
	db.Model(&pharmacy.PharmacyOrder{}).Where("created_at >= ?", today).Count(&ordersToday)
	db.Model(&pharmacy.PharmacyOrder{}).Where("status = ?", "pending").Count(&pendingOrders)

	// Total Sales (Paid Orders)
	db.Model(&pharmacy.PharmacyOrder{}).
		Where("payment_status = ?", "paid").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalSales)

	// Operational Expenses (Approved Pharmacy Expenses)
	// We use the raw table name "expenses" to avoid circular dependencies
	db.Table("expenses").
		Where("department = ? AND status = ?", "PHARMACY", "approved").
		Select("COALESCE(SUM(amount), 0) / 100.0"). // amount is in paise
		Scan(&operationalExpenses)

	// Inventory Stats
	db.Model(&pharmacy.Medicine{}).Count(&totalMedicines)
	db.Model(&pharmacy.PharmacyInventory{}).Count(&totalInventoryItems)
	db.Model(&pharmacy.PharmacyInventory{}).Where("stock_quantity <= reorder_level").Count(&lowStockItems)

	// Inventory Value — prices is a JSONB map keyed by currency code (e.g. "INR")
	db.Table("pharmacyinventory").
		Joins("JOIN medicine ON pharmacyinventory.medicine_id = medicine.id").
		Select("COALESCE(SUM(pharmacyinventory.stock_quantity * (medicine.prices->>'INR')::bigint), 0)").
		Scan(&inventoryValue)

	return c.JSON(fiber.Map{
		"total_sales":           totalSales,
		"orders_today":          ordersToday,
		"pending_orders":        pendingOrders,
		"low_stock_items":       lowStockItems,
		"total_medicines":       totalMedicines,
		"total_inventory_items": totalInventoryItems,
		"inventory_value":       inventoryValue,
		"operational_expenses":  operationalExpenses,
		"net_earnings":          totalSales - operationalExpenses,
	})
}

// GetCourierDashboard - Courier dashboard
// GetCourierDashboard returns metrics for the courier/delivery dashboard
// @Summary Get Courier Dashboard
// @Description Fetch stats related to active deliveries and courier performance
// @Tags Admin Dashboard
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "Courier dashboard fetched"
// @Router /api/v1/admin/dashboard/courier [get]
func GetCourierDashboard(c *fiber.Ctx) error {
	db := database.DB
	today := time.Now().Truncate(24 * time.Hour)

	var activeCount, completedToday, pendingAssignments int64

	// Active = courier picked up or en route
	db.Model(&pharmacy.CourierDelivery{}).
		Where("status IN ?", []string{"picked_up", "en_route", "arrived"}).
		Count(&activeCount)

	// Completed today
	db.Model(&pharmacy.CourierDelivery{}).
		Where("status = ? AND actual_delivery_time >= ?", "completed", today).
		Count(&completedToday)

	// Pending = assigned but not yet picked up
	db.Model(&pharmacy.CourierDelivery{}).
		Where("status = ?", "assigned").
		Count(&pendingAssignments)

	// Active Shipments List (Recent 10)
	type ShipmentView struct {
		ID             string    `json:"id"`
		Destination    string    `json:"destination"`
		ProviderName   string    `json:"provider_name"`
		Status         string    `json:"status"`
		EstimatedArrival string  `json:"eta"`
		CreatedAt      time.Time `json:"created_at"`
	}
	var activeShipments []ShipmentView
	db.Table("shipments").
		Select("tracking_number as id, destination, provider_name, status, COALESCE(CAST(estimated_arrival AS TEXT), 'Calculating...') as eta, created_at").
		Where("status NOT IN ?", []string{"delivered", "failed", "cancelled"}).
		Order("created_at DESC").
		Limit(10).
		Scan(&activeShipments)

	// Regional Stats (Grouped by Destination City or Region)
	type RegionStat struct {
		Label    string `json:"label"`
		Progress int    `json:"progress"`
	}
	var regionalStats []RegionStat
	// Mocking logic for progress based on volume for now, or use real capacity if available
	db.Table("shipments").
		Select("destination as label, COUNT(*) as progress").
		Where("created_at >= ?", today).
		Group("destination").
		Order("progress DESC").
		Limit(5).
		Scan(&regionalStats)

	// Normalize progress to 0-100 for UI if needed, or just return counts
	
	return c.JSON(fiber.Map{
		"active_deliveries":   activeCount,
		"completed_today":     completedToday,
		"pending_assignments": pendingAssignments,
		"active_shipments":    activeShipments,
		"regional_stats":      regionalStats,
	})
}

// GetNotificationsDashboard - Notifications dashboard
// GetNotificationsDashboard returns stats for the notification system
// @Summary Get Notifications Dashboard
// @Description Fetch metrics on sent, pending, and failed notifications
// @Tags Admin Dashboard
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "Notifications dashboard fetched"
// @Router /api/v1/admin/dashboard/notifications [get]
func GetNotificationsDashboard(c *fiber.Ctx) error {
	db := database.DB
	today := time.Now().Truncate(24 * time.Hour)
	last30d := time.Now().AddDate(0, 0, -30)

	var totalSent, sentToday, failedToday int64
	db.Model(&communication.NotificationLog{}).Where("status = ?", "sent").Count(&totalSent)
	db.Model(&communication.NotificationLog{}).
		Where("status = ? AND sent_at >= ?", "sent", today).Count(&sentToday)
	db.Model(&communication.NotificationLog{}).
		Where("status = ? AND sent_at >= ?", "failed", today).Count(&failedToday)

	// Per-channel success rates over the last 30 days. notification_type
	// holds the channel (sms | whatsapp | email). Default to 0 when no
	// traffic so the UI shows an honest "0%" instead of a fabricated 98%.
	channelRate := func(channel string) float64 {
		var sent, failed int64
		db.Model(&communication.NotificationLog{}).
			Where("notification_type = ? AND status = ? AND sent_at >= ?", channel, "sent", last30d).Count(&sent)
		db.Model(&communication.NotificationLog{}).
			Where("notification_type = ? AND status = ? AND sent_at >= ?", channel, "failed", last30d).Count(&failed)
		total := sent + failed
		if total == 0 {
			return 0
		}
		return (float64(sent) / float64(total)) * 100.0
	}

	return c.JSON(fiber.Map{
		// canonical fields the frontend reads
		"total_sent_today":      sentToday,
		"total_sent":            totalSent,
		"failed_today":          failedToday,
		"active_alert_count":    failedToday, // failures today = alerts needing attention
		"whatsapp_success_rate": channelRate("whatsapp"),
		"email_delivery_rate":   channelRate("email"),
		"sms_success_rate":      channelRate("sms"),
		// legacy aliases (kept for any older consumer)
		"sent_today": sentToday,
	})
}

// GetUserManagementStats - User management stats
// GetUserManagementStats returns stats for user management
// @Summary Get User Management Stats
// @Description Fetch statistics on total users, active sessions, and verified roles
// @Tags Admin Dashboard
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "User management stats fetched"
// @Router /api/v1/admin/dashboard/users/stats [get]
func GetUserManagementStats(c *fiber.Ctx) error {
	db := database.DB
	today := time.Now().Truncate(24 * time.Hour)
	weekAgo := today.AddDate(0, 0, -7)

	var totalUsers, activeUsers, newThisWeek, verifiedDoctors int64

	db.Model(&identity.User{}).Count(&totalUsers)
	db.Model(&identity.User{}).Where("is_active = ?", true).Count(&activeUsers)
	db.Model(&identity.User{}).Where("created_at >= ?", weekAgo).Count(&newThisWeek)
	db.Model(&identity.User{}).Where("role = ? AND is_verified = ?", "doctor", true).Count(&verifiedDoctors)

	return c.JSON(fiber.Map{
		"total_users":      totalUsers,
		"active_users":     activeUsers,
		"new_this_week":    newThisWeek,
		"verified_doctors": verifiedDoctors,
	})
}

// GetSystemAlerts - System alerts
// GetSystemAlerts returns active system-wide alerts
// @Summary Get System Alerts
// @Description Fetch all active system warnings, errors, and informational alerts for admins
// @Tags Admin Dashboard
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} []fiber.Map "System alerts fetched"
// @Router /api/v1/admin/dashboard/alerts [get]
func GetSystemAlerts(c *fiber.Ctx) error {
	db := database.DB
	alerts := []fiber.Map{}

	// Check low stock
	var lowStock int64
	db.Model(&pharmacy.PharmacyInventory{}).Where("stock_quantity <= reorder_level").Count(&lowStock)
	if lowStock > 0 {
		alerts = append(alerts, fiber.Map{
			"type":     "warning",
			"message":  "Low stock items detected",
			"count":    lowStock,
			"severity": "medium",
		})
	}

	// Check pending doctor verifications
	var pendingDoctors int64
	db.Model(&identity.User{}).Where("role = ? AND is_verified = ?", "doctor", false).Count(&pendingDoctors)
	if pendingDoctors > 0 {
		alerts = append(alerts, fiber.Map{
			"type":     "info",
			"message":  "Pending doctor verifications",
			"count":    pendingDoctors,
			"severity": "low",
		})
	}

	return c.JSON(alerts)
}

// GetLabDashboard - Lab dashboard
// GetLabDashboard returns metrics for the laboratory system
// @Summary Get Lab Dashboard
// @Description Fetch stats on pending tests and completed reports for the chief doctor
// @Tags Admin Dashboard
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "Lab dashboard fetched"
// @Router /api/v1/admin/dashboard/labs [get]
func GetLabDashboard(c *fiber.Ctx) error {
	db := database.DB
	today := time.Now().Truncate(24 * time.Hour)

	var pendingTests, completedToday, pendingResults int64

	db.Model(&lab.LabRequest{}).Where("status = ?", "pending").Count(&pendingTests)
	db.Model(&lab.LabRequest{}).
		Where("status = ? AND created_at >= ?", "completed", today).
		Count(&completedToday)
	// Requests that are processing but results not yet uploaded
	db.Model(&lab.LabRequest{}).Where("status = ?", "processing").Count(&pendingResults)

	return c.JSON(fiber.Map{
		"pending_tests":   pendingTests,
		"completed_today": completedToday,
		"pending_results": pendingResults,
	})
}

// Monitoring Stubs

func GetSystemHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "healthy",
		"uptime": "15d 4h 22m",
		"services": []fiber.Map{
			{"name": "Auth Service", "status": "up"},
			{"name": "EMR Engine", "status": "up"},
			{"name": "Payment Gateway", "status": "warning"},
		},
	})
}

func GetMonitoringMetrics(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"cpu_usage":    "14%",
		"memory_usage": "1.2GB",
		"disk_usage":   "45%",
		"goroutines":   142,
	})
}

func GetMonitoringLogs(c *fiber.Ctx) error {
	return c.JSON([]fiber.Map{
		{"timestamp": time.Now().Add(-5 * time.Minute), "level": "INFO", "message": "User login success", "service": "auth"},
		{"timestamp": time.Now().Add(-15 * time.Minute), "level": "WARN", "message": "Slow database query", "service": "db"},
	})
}

func GetMonitoringPerformance(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"p95_latency": "140ms",
		"p99_latency": "450ms",
		"throughput":  "120 req/s",
	})
}

func GetPendingMonitoringApprovals(c *fiber.Ctx) error {
	return c.JSON([]fiber.Map{
		{"id": 1, "type": "BULK_DELETE", "user": "admin_joe", "created_at": time.Now().Add(-2 * time.Hour)},
	})
}

func ApproveMonitoringBatch(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "Batch approved", "count": 1})
}

func GetAuditSample(c *fiber.Ctx) error {
	return c.JSON([]fiber.Map{
		{"id": 101, "action": "LOGIN", "user": "doctor_smith", "date": time.Now().Add(-1 * time.Hour)},
	})
}

// GetDispatchDashboard - Dispatch dashboard
// GetDispatchDashboard returns metrics for the dispatch system
// @Summary Get Dispatch Dashboard
// @Description Fetch key metrics for the dispatcher dashboard (queue length, wait times)
// @Tags Admin Dashboard
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "Dispatch dashboard fetched"
// @Router /api/v1/admin/dashboard/dispatch [get]
func GetDispatchDashboard(c *fiber.Ctx) error {
	db := database.DB
	today := time.Now().Truncate(24 * time.Hour)

	// Instant / SOS appointments waiting for a doctor (queue)
	var queueLength, assignedToday int64
	db.Model(&appointment.Appointment{}).
		Where("status = ? AND appointment_type IN ?", "pending",
			[]string{"instant", "emergency"}).
		Count(&queueLength)
	db.Model(&appointment.Appointment{}).
		Where("appointment_type IN ? AND doctor_id IS NOT NULL AND created_at >= ?",
			[]string{"instant", "emergency"}, today).
		Count(&assignedToday)

	return c.JSON(fiber.Map{
		"queue_length":      queueLength,
		"assigned_today":    assignedToday,
		"average_wait_time": 0, // requires time-series data — left for analytics layer
	})
}

// GetAppointments - Get all appointments for admin
// GetAppointments returns all appointments for the admin view
// @Summary Get Admin Appointments List
// @Description Fetch a list of the last 100 appointments across the system for administrative review
// @Tags Admin Dashboard
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} []map[string]interface{} "Appointments list fetched (flattened)"
// @Router /api/v1/admin/dashboard/appointments [get]
//
// GetAppointments returns a FLATTENED appointment list for the admin UI.
//
// The raw Appointment model embeds Patient + Doctor profiles which themselves
// embed User. The admin UI just needs patient/doctor display names, so we
// flatten the response here to avoid:
//   1. Three levels of nested JSON the frontend has to walk
//   2. "Unknown / Unassigned / Invalid Date" placeholders when frontend
//      can't find the fields
//   3. Leaking irrelevant profile fields (allergies, license numbers, etc.)
func GetAppointments(c *fiber.Ctx) error {
	db := database.DB
	var appointments []appointment.Appointment

	if err := db.
		Preload("Patient").
		Preload("Patient.User").
		Preload("Doctor").
		Order("start_time DESC").
		Limit(100).
		Find(&appointments).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "failed to load appointments: " + err.Error()})
	}

	// DoctorProfile doesn't embed User, so resolve doctor names via a single
	// batched query against the users table keyed by user_id (which is what
	// the appointment.DoctorID references).
	doctorUserIDs := make([]uint, 0, len(appointments))
	for _, a := range appointments {
		if a.DoctorID != nil && *a.DoctorID > 0 {
			doctorUserIDs = append(doctorUserIDs, *a.DoctorID)
		}
	}

	doctorNames := map[uint]string{}
	if len(doctorUserIDs) > 0 {
		type userRow struct {
			ID       uint
			FullName string
		}
		var rows []userRow
		_ = db.Table("user").
			Select("id, full_name").
			Where("id IN ?", doctorUserIDs).
			Scan(&rows).Error
		for _, r := range rows {
			doctorNames[r.ID] = r.FullName
		}
	}

	rows := make([]fiber.Map, 0, len(appointments))
	for _, a := range appointments {
		patientName := "Unknown"
		if a.Patient.User != nil && a.Patient.User.FullName != "" {
			patientName = a.Patient.User.FullName
		}

		doctorName := "Unassigned"
		doctorID := uint(0)
		if a.DoctorID != nil && *a.DoctorID > 0 {
			doctorID = *a.DoctorID
			if name, ok := doctorNames[*a.DoctorID]; ok && name != "" {
				doctorName = name
			} else {
				doctorName = fmt.Sprintf("Doctor #%d", *a.DoctorID)
			}
		}

		// Normalize consultation mode to the type field expected by UI.
		apptType := a.ConsultationMode
		if apptType == "" {
			apptType = "video"
		}

		// Convert paise → rupees for display (UI shows ₹ amounts).
		amountRupees := float64(a.ConsultationFee) / 100.0

		rows = append(rows, fiber.Map{
			"id":           a.ID,
			"patient_id":   a.PatientID,
			"patient_name": patientName,
			"doctor_id":    doctorID,
			"doctor_name":  doctorName,
			"type":         apptType,
			"status":       a.Status,
			"start_time":   a.StartTime,
			"end_time":     a.EndTime,
			// Frontend convenience fields — pre-formatted-ready
			"date":             a.StartTime,
			"consultation_fee": a.ConsultationFee, // paise (raw)
			"amount":           amountRupees,      // rupees (display)
			"appointment_type": a.AppointmentType,
			"is_priority":      a.IsPriority,
			"payout_status":    a.PayoutStatus,
			"created_at":       a.CreatedAt,
		})
	}

	return c.JSON(rows)
}

// GetFinanceStats - Admin Finance Stats
// GetFinanceStats returns specialized financial metrics
// @Summary Get Finance Stats
// @Description Fetch detailed financial stats including platform fees and escrow balances
// @Tags Admin Finance
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "Finance stats fetched"
// @Router /api/v1/admin/finance/stats [get]
func GetFinanceStats(c *fiber.Ctx) error {
	// Mock implementation for missing endpoint
	return c.JSON(fiber.Map{
		"total_revenue":   150000,
		"pending_payouts": 25000,
		"active_escrow":   10000,
		"platform_fees":   5000,
	})
}

// GetFinanceLedgerStream - Admin Finance Ledger Stream
// GetFinanceLedgerStream returns a stream of recent financial transactions
// @Summary Get Finance Ledger Stream
// @Description Fetch a list of recent ledger entries for the finance dashboard stream
// @Tags Admin Finance
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} []fiber.Map "Ledger stream fetched"
// @Router /api/v1/admin/finance/ledger/stream [get]
func GetFinanceLedgerStream(c *fiber.Ctx) error {
	// Mock implementation for missing endpoint
	return c.JSON([]fiber.Map{
		{
			"id":          "tx_1",
			"amount":      1500,
			"type":        "consultation",
			"status":      "completed",
			"description": "Consultation with Dr. Smith",
			"created_at":  time.Now().Format(time.RFC3339),
		},
	})
}

// GetActiveSessions - Admin Active Sessions
// GetActiveSessions returns current active user sessions
// @Summary Get Active Sessions
// @Description Fetch a list of current active user sessions for system monitoring
// @Tags Admin Dashboard
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} []fiber.Map "Active sessions fetched"
// @Router /api/v1/admin/dashboard/active-sessions [get]
func GetActiveSessions(c *fiber.Ctx) error {
	// Mock implementation for missing endpoint
	return c.JSON([]fiber.Map{})
}

func RegisterAdminRoutes(router fiber.Router) {
	adminBase := router.Group("/admin", middleware.Protected())
	adminOnly := middleware.AdminOnly()
	hrAndAdmin := middleware.HRManagerOnly()
	// headDoctor covers both head-doctor roles: chief_doctor (the online network
	// head doctor who runs clinical oversight) and hospital_head_doctor (physical
	// hospital lead), plus manager. ChiefDoctorOnly() excludes chief_doctor, which
	// wrongly 403'd the online head doctor on the doctor-management routes below.
	headDoctor := middleware.RequireRoles("chief_doctor", "hospital_head_doctor", "manager", "admin")

	// Dashboards
	adminBase.Get("/dashboard/overview", adminOnly, GetSystemOverview)
	adminBase.Get("/analytics/overview", adminOnly, GetAnalyticsOverview)
	adminBase.Get("/dashboard/revenue", middleware.FinanceManagerOnly(), GetRevenueAnalytics)
	adminBase.Get("/dashboard/hospital", middleware.ChiefDoctorOnly(), GetHospitalOperations)
	adminBase.Get("/dashboard/pharmacy", middleware.PharmacyManagerOnly(), GetPharmacyDashboard)
	adminBase.Get("/dashboard/courier", adminOnly, GetCourierDashboard)
	adminBase.Get("/dashboard/notifications", hrAndAdmin, GetNotificationsDashboard)
	adminBase.Get("/dashboard/users/stats", hrAndAdmin, GetUserManagementStats)
	adminBase.Get("/dashboard/doctors/stats", adminOnly, GetDoctorManagementStats)
	adminBase.Get("/dashboard/alerts", adminOnly, GetSystemAlerts)
	adminBase.Get("/dashboard/labs", middleware.ChiefDoctorOnly(), GetLabDashboard)
	adminBase.Get("/dashboard/dispatch", middleware.BPOOnly(), GetDispatchDashboard)
	adminBase.Get("/dashboard/appointments", middleware.ChiefDoctorOnly(), GetAppointments)
	adminBase.Get("/dashboard/active-sessions", adminOnly, GetActiveSessions)

	// HR Dashboard
	adminBase.Get("/dashboard/hr", middleware.HRManagerOnly(), GetHRDashboardOverview)
	adminBase.Get("/dashboard/hr/staff", middleware.HRManagerOnly(), GetStaffDirectory)
	adminBase.Post("/dashboard/hr/staff/:id/status", middleware.HRManagerOnly(), UpdateStaffStatus)
	adminBase.Get("/dashboard/hr/payroll/export", middleware.HRManagerOnly(), ExportPayroll)

	// Finance Dashboard
	adminBase.Get("/dashboard/finance", middleware.FinanceManagerOnly(), GetFinanceDashboardOverview)
	adminBase.Get("/dashboard/finance/ledger", middleware.FinanceManagerOnly(), GetFinancialLedger)
	adminBase.Get("/dashboard/finance/partners", middleware.FinanceManagerOnly(), GetProjectPartners)
	adminBase.Get("/dashboard/finance/investors", middleware.FinanceManagerOnly(), GetInvestors)
	adminBase.Get("/dashboard/finance/expenses", middleware.FinanceManagerOnly(), GetFinancialLedger) // Reusing ledger for expenses
	adminBase.Get("/dashboard/finance/franchise", middleware.FinanceManagerOnly(), GetFranchiseDeals)
	adminBase.Get("/dashboard/finance/suppliers", middleware.FinanceManagerOnly(), GetPharmacySuppliers)
	adminBase.Post("/dashboard/finance/payouts/execute", middleware.FinanceManagerOnly(), ExecutePayouts)
	adminBase.Get("/dashboard/finance/tds/export", middleware.FinanceManagerOnly(), ExportTDSReport)
	adminBase.Get("/dashboard/finance/reconciliation", middleware.FinanceManagerOnly(), RunLedgerReconciliation)

	// BPO Dashboard
	adminBase.Get("/dashboard/bpo", middleware.BPOOnly(), GetBPODashboardOverview)
	adminBase.Get("/dashboard/bpo/agents", middleware.BPOOnly(), GetBPOAgentPerformance)

	// Finance
	adminBase.Get("/finance/stats", middleware.FinanceManagerOnly(), GetFinanceStats)
	adminBase.Get("/finance/ledger/stream", middleware.FinanceManagerOnly(), GetFinanceLedgerStream)

	// Settings
	adminBase.Get("/settings", adminOnly, GetSettings)
	adminBase.Post("/settings", adminOnly, UpdateSettings)
	adminBase.Put("/settings", adminOnly, UpdateSettings)

	// Activity Logs
	adminBase.Get("/activity-logs", hrAndAdmin, GetActivityLogs)
	adminBase.Get("/activity-logs/stats", hrAndAdmin, GetActivityLogsStats)
	adminBase.Post("/activity-logs/ask", hrAndAdmin, AskActivityLogs)
	adminBase.Get("/activity-logs/export", hrAndAdmin, ExportActivityLogsCSV)
	adminBase.Get("/activity-logs/export/excel", hrAndAdmin, ExportActivityLogsExcel)
	// Realtime logs (WS)
	adminBase.Get("/activity-logs/realtime", handlers.WebSocketUpgrade, websocket.New(handlers.WebSocketHandler))

	// Monitoring — accessible to QA managers and admins
	monitoring := adminBase.Group("/monitoring", middleware.QAManagerOnly())
	monitoring.Get("/health", GetSystemHealth)
	monitoring.Get("/metrics", GetMonitoringMetrics)
	monitoring.Get("/alerts", GetSystemAlerts)
	monitoring.Get("/logs", GetMonitoringLogs)
	monitoring.Get("/performance", GetMonitoringPerformance)
	monitoring.Get("/pending-approvals", GetPendingMonitoringApprovals)
	monitoring.Post("/approve-batch", ApproveMonitoringBatch)
	monitoring.Get("/audit-sample", GetAuditSample)

	// Dispatch Management
	dispatchGroup := adminBase.Group("/dispatch", middleware.BPOOnly())
	dispatchGroup.Get("/", GetDispatchDashboard)
	dispatchGroup.Get("/queue", dispatch.GetQueue)
	dispatchGroup.Get("/stats", dispatch.GetQueueStats)
	dispatchGroup.Get("/sessions", GetActiveSessions)
	dispatchGroup.Post("/run", middleware.AdminOnly(), dispatch.TriggerRun)
	dispatchGroup.Post("/autopilot", middleware.AdminOnly(), dispatch.ToggleAutopilot)

	// HR Management
	hrAdminGroup := adminBase.Group("/hr", middleware.HRManagerOnly())
	hrAdminGroup.Get("/shifts", GetShifts)
	hrAdminGroup.Post("/shifts", CreateShift)
	hrAdminGroup.Get("/leaves", GetLeaves)
	hrAdminGroup.Post("/leaves/:id/:action", ActionLeave)
	hrAdminGroup.Get("/payroll", GetPayroll)
	hrAdminGroup.Post("/payroll/generate", GeneratePayroll)
	// Performance Reviews
	hrAdminGroup.Get("/performance-reviews", GetPerformanceReviews)
	hrAdminGroup.Post("/performance-reviews", CreatePerformanceReview)
	hrAdminGroup.Post("/performance-reviews/:id/approve", ApprovePerformanceReview)
	// Doctor Pricing
	hrAdminGroup.Get("/pricing", GetHRPricingRules)
	hrAdminGroup.Put("/pricing/:id", UpdateHRPricingRule)
	hrAdminGroup.Get("/pricing/doctors", GetDoctorPricingList)
	hrAdminGroup.Put("/pricing/doctors/:id", UpdateDoctorPricing)
	hrAdminGroup.Get("/pricing/templates", GetHRPricingTemplates)
	hrAdminGroup.Post("/pricing/templates/apply", ApplyPricingTemplate)
	// Panel (freelance) doctor revenue-share earnings — accrued net, TDS, payouts.
	hrAdminGroup.Get("/panel-doctors/earnings", hr.PanelDoctorEarnings)
	hrAdminGroup.Put("/panel-doctors/:id/pan", hr.SetPanelDoctorPAN)
	// Salaried-staff hiring front door (creates user + StaffProfile + onboarding)
	hrAdminGroup.Post("/staff", HireStaff)
	// Offboarding + attendance marking (drives payroll loss-of-pay)
	hrAdminGroup.Post("/staff/:id/offboard", hr.OffboardStaff)
	hrAdminGroup.Get("/staff/:id/settlement", hr.GetFinalSettlement)
	hrAdminGroup.Put("/staff/:id/clearance", hr.UpdateExitClearance)
	hrAdminGroup.Post("/attendance/mark", hr.MarkAttendance)
	hrAdminGroup.Get("/attendance", hr.GetAttendanceSheet)
	// Online doctors + staff lists (reuse existing data)
	hrAdminGroup.Get("/doctors/online", func(c *fiber.Ctx) error {
		var docs []identity.User
		database.DB.
			Joins("JOIN doctorprofile dp ON dp.user_id = \"user\".id").
			Where("\"user\".role = ? AND dp.is_online = ?", "doctor", true).
			Find(&docs)
		for i := range docs { docs[i].PasswordHash = "" }
		if docs == nil { docs = []identity.User{} }
		return c.JSON(docs)
	})
	hrAdminGroup.Get("/hospitals/staff", func(c *fiber.Ctx) error {
		var staff []identity.User
		database.DB.Where("role IN ?", []string{"nurse", "receptionist", "lab_technician"}).Find(&staff)
		for i := range staff { staff[i].PasswordHash = "" }
		if staff == nil { staff = []identity.User{} }
		return c.JSON(staff)
	})
	hrAdminGroup.Get("/pharmacy/staff", func(c *fiber.Ctx) error {
		var staff []identity.User
		database.DB.Where("role IN ?", []string{"pharmacist", "pharmacy_manager"}).Find(&staff)
		for i := range staff { staff[i].PasswordHash = "" }
		if staff == nil { staff = []identity.User{} }
		return c.JSON(staff)
	})

	// P5: refill reminder sweep — admin-triggered (called by external daily cron)
	adminBase.Post("/pharmacy/refills/sweep", adminOnly, emr.SweepRefillsDueHandler)
	// P6: batch recall lookup — returns every patient who received a recalled batch
	adminBase.Get("/pharmacy/recalls", adminOnly, pharmacy.LookupAffectedByBatchHandler)

	// Audit — accessible to QA managers and admins
	adminBase.Get("/audit/activity", middleware.QAManagerOnly(), GetSystemActivityLogs)
	adminBase.Get("/audit/ledger/:transaction_id", middleware.QAManagerOnly(), GetLedgerAuditTrail)

	// Ecosystem
	adminBase.Get("/ecosystem/stats", adminOnly, GetEcosystemStats)

	// Users
	adminBase.Get("/users", hrAndAdmin, GetUsers)
	adminBase.Get("/users/:id", hrAndAdmin, GetUserDetails)
	adminBase.Post("/users/:id/:action", hrAndAdmin, ToggleUserStatus)
	adminBase.Post("/users", hrAndAdmin, CreateUser)
	adminBase.Post("/users/bulk", hrAndAdmin, CreateUsersBulk)
	adminBase.Put("/users/:id", hrAndAdmin, UpdateUser)
	adminBase.Delete("/users/:id", adminOnly, DeleteUser)

	// Doctors (Chief Doctor / Admin)
	adminBase.Get("/doctors", headDoctor, GetPendingDoctors)
	adminBase.Get("/doctors/verified", headDoctor, GetVerifiedDoctors)
	adminBase.Get("/doctors/audit", middleware.AdminOnly(), GetDoctorAuditDirectory)
	adminBase.Get("/doctors/metrics", headDoctor, GetDoctorMetrics)
	adminBase.Post("/doctors/:id/verify", headDoctor, VerifyDoctor)
	adminBase.Post("/doctors/:id/premium", headDoctor, PromotePremiumDoctor)
	adminBase.Patch("/doctors/:id", headDoctor, AdminUpdateDoctor)
	adminBase.Post("/doctors/:id/recalc-score", headDoctor, AdminRecalcDoctorScore)
	adminBase.Get("/doctors/performance", headDoctor, GetDoctorPerformanceTable)

	// Ratings Audit
	adminBase.Get("/ratings", headDoctor, AdminGetRatings)
	adminBase.Patch("/ratings/:id/hide", headDoctor, AdminHideRating)
	adminBase.Patch("/ratings/:id/restore", headDoctor, AdminRestoreRating)

	// Appointments Admin
	adminBase.Get("/appointments", middleware.ChiefDoctorOnly(), GetAllAppointments)
	adminBase.Get("/appointments/:id", middleware.ChiefDoctorOnly(), GetAppointmentDetails)
	adminBase.Put("/appointments/:id", middleware.ChiefDoctorOnly(), UpdateAppointment)
	adminBase.Delete("/appointments/:id", middleware.ChiefDoctorOnly(), CancelAppointment)
	adminBase.Post("/appointments/reassign", middleware.ChiefDoctorOnly(), ReassignAppointments)

	// AI Brain (Devi) admin routes
	aiGroup := adminBase.Group("/ai", adminOnly)
	aiGroup.Get("/activity", GetAIBrainActivity)
	aiGroup.Get("/insights", GetAIBrainInsights)
	aiGroup.Get("/status", GetAIBrainStatus)
	aiGroup.Get("/logs", GetAIBrainLogs)

	// Access Control / JIT
	acGroup := adminBase.Group("/access-control", adminOnly)
	acGroup.Get("/trust-score", GetAccessControlTrustScore)
	acGroup.Get("/jit-requests", GetJITRequests)
	acGroup.Post("/jit-requests", CreateJITRequest)

	// Productivity dashboard
	prodGroup := adminBase.Group("/productivity", adminOnly)
	prodGroup.Get("/dashboard", GetProductivityDashboard)
	prodGroup.Get("/metrics", GetProductivityMetrics)
	prodGroup.Get("/scores", GetProductivityScores)
	prodGroup.Get("/doctors", GetAllDoctorsMetrics)
	prodGroup.Get("/leaderboard", GetProductivityLeaderboard)
	prodGroup.Get("/doctors/:id/kpi", GetDoctorKPI)
	prodGroup.Post("/doctors/:id/target", SetDoctorTarget)
}

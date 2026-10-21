package hr

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

type ClockInRequest struct {
	UserID  uint `json:"user_id"`
	ShiftID uint `json:"shift_id"`
}

type ClockOutRequest struct {
	UserID uint `json:"user_id"`
}

type Handler struct {
	hrService         *Service
	onboardingHandler *OnboardingHandler
}

func NewHandler(s *Service, oh *OnboardingHandler) *Handler {
	return &Handler{
		hrService:         s,
		onboardingHandler: oh,
	}
}

// ClockIn records a staff member's punch-in time
// @Summary Staff Clock-In
// @Description Record the start of a staff member's shift
// @Tags HR
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body ClockInRequest true "Clock-in Request"
// @Success 200 {object} response.StandardResponse "Clocked in"
// @Router /api/v1/hr/attendance/clock-in [post]
func (h *Handler) ClockIn(c *fiber.Ctx) error {
	var req ClockInRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	err := h.hrService.ClockIn(req.UserID, req.ShiftID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CLOCK_IN_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Clocked in successfully")
}

// ClockOut records a staff member's punch-out time
// @Summary Staff Clock-Out
// @Description Record the end of a staff member's shift
// @Tags HR
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param attendanceId path string true "Attendance Entry ID"
// @Param request body ClockOutRequest true "Clock-out Request"
// @Success 200 {object} response.StandardResponse "Clocked out"
// @Router /api/v1/hr/attendance/clock-out/{attendanceId} [post]
func (h *Handler) ClockOut(c *fiber.Ctx) error {
	attendanceID, _ := strconv.Atoi(c.Params("attendanceId"))

	var req ClockOutRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	err := h.hrService.ClockOut(req.UserID, uint(attendanceID))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CLOCK_OUT_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Clocked out successfully")
}

// CalculatePayroll generates payroll for a staff member
// @Summary Calculate Payroll
// @Description Calculate monthly earnings for a specific staff member
// @Tags HR
// @Produce json
// @Security ApiKeyAuth
// @Param userId path string true "User ID"
// @Param month query int true "Month (1-12)"
// @Param year query int true "Year"
// @Success 200 {object} response.StandardResponse{data=map[string]interface{}} "Payroll calculated"
// @Router /api/v1/hr/payroll/{userId} [get]
func (h *Handler) CalculatePayroll(c *fiber.Ctx) error {
	userID, _ := strconv.Atoi(c.Params("userId"))
	month, _ := strconv.Atoi(c.Query("month"))
	year, _ := strconv.Atoi(c.Query("year"))

	result, err := h.hrService.CalculateMonthlyPayroll(uint(userID), month, year)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CALC_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, result, "Payroll calculated successfully")
}

// GetHRRoster fetches the staff roster for a hospital
// @Summary Get HR Roster
// @Description Fetch shift assignments and availability for all staff in a hospital
// @Tags HR
// @Produce json
// @Security ApiKeyAuth
// @Param hospitalId query int true "Hospital ID"
// @Success 200 {object} response.StandardResponse{data=map[string]interface{}} "Roster fetched"
// @Router /api/v1/hr/roster [get]
func (h *Handler) GetHRRoster(c *fiber.Ctx) error {
	hospitalID, _ := strconv.Atoi(c.Query("hospitalId"))

	roster, err := h.hrService.GetStaffRoster(uint(hospitalID))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, roster, "Staff roster fetched")
}

type RecentStaff struct {
	ID        uint      `json:"id"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// HRDashboardOverview represents the overview statistics for the HR dashboard
type HRDashboardOverview struct {
	TotalStaff        int64            `json:"total_staff"`
	StaffByDepartment map[string]int64 `json:"staff_by_department"`
	PendingLeaves     int              `json:"pending_leaves"`
	ActiveRecruitment int              `json:"active_recruitment"`
	AttendanceToday   float64          `json:"attendance_today"`
	ActiveDoctors     int64            `json:"active_doctors"`
	ActiveNurses      int64            `json:"active_nurses"`
	ActiveLabTechs    int64            `json:"active_lab_techs"`
	ShiftsToday       int              `json:"shifts_today"`
	RecentStaff       []RecentStaff    `json:"recent_staff"`

	// Employment-type split — the salaried/freelancer fork made visible.
	// SalariedStaff have a StaffProfile and run through payroll; PanelDoctors are
	// freelancers paid via consultation revenue-share (no payroll).
	SalariedStaff int64 `json:"salaried_staff"`
	PanelDoctors  int64 `json:"panel_doctors"`
}

// GetDashboardOverview returns overview statistics for the HR dashboard
// GetDashboardOverview returns overview statistics for the HR dashboard
// @Summary Get HR Dashboard Overview
// @Description Fetch key HR metrics including staff counts, department distribution, and attendance stats
// @Tags HR
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} response.StandardResponse{data=HRDashboardOverview} "Overview fetched"
// @Router /api/v1/hr/dashboard/overview [get]
func (h *Handler) GetDashboardOverview(c *fiber.Ctx) error {
	db := database.DB
	var overview HRDashboardOverview

	// 1. Total Staff (Non-patient users)
	db.Model(&identity.User{}).Where("role != ?", "patient").Count(&overview.TotalStaff)

	// 2. Counts by Role
	db.Model(&identity.User{}).Where("role = ?", "doctor").Count(&overview.ActiveDoctors)
	db.Model(&identity.User{}).Where("role = ?", "nurse").Count(&overview.ActiveNurses)
	db.Model(&identity.User{}).Where("role = ?", "lab_tech").Count(&overview.ActiveLabTechs)

	// 3. Staff by Role (as a proxy for department for now)
	rows, err := db.Model(&identity.User{}).
		Where("role != ?", "patient").
		Select("role, count(*)").
		Group("role").Rows()

	if err == nil {
		overview.StaffByDepartment = make(map[string]int64)
		for rows.Next() {
			var role string
			var count int64
			rows.Scan(&role, &count)
			overview.StaffByDepartment[role] = count
		}
		rows.Close()
	}

	// 4. Recent Staff
	var staff []identity.User
	db.Where("role != ?", "patient").Order("created_at desc").Limit(5).Find(&staff)
	overview.RecentStaff = make([]RecentStaff, len(staff))
	for i, s := range staff {
		overview.RecentStaff[i] = RecentStaff{
			ID:        s.ID,
			FullName:  s.FullName,
			Role:      s.Role,
			CreatedAt: s.CreatedAt,
		}
	}

	// 5. Real operational metrics (previously hardcoded mocks).

	// Pending leave requests awaiting HR approval.
	var pendingLeaves int64
	db.Model(&LeaveRequest{}).Where("status = ?", "pending").Count(&pendingLeaves)
	overview.PendingLeaves = int(pendingLeaves)

	// Active job openings (career module). Queried by table name to avoid a
	// package dependency from hr → career.
	var activeRecruitment int64
	db.Table("job_openings").Where("is_active = ?", true).Count(&activeRecruitment)
	overview.ActiveRecruitment = int(activeRecruitment)

	// Attendance today: staff marked present/late/half_day as a % of total staff.
	today := time.Now().Format("2006-01-02")
	var presentToday int64
	db.Model(&StaffAttendance{}).
		Where("date = ? AND status IN ?", today, []string{"present", "late", "half_day"}).
		Count(&presentToday)
	overview.ShiftsToday = int(presentToday)
	if overview.TotalStaff > 0 {
		overview.AttendanceToday = float64(presentToday) / float64(overview.TotalStaff) * 100.0
	}

	// 6. Employment-type split (salaried vs freelancer panel doctors).
	db.Model(&StaffProfile{}).
		Where("employment_type = ? OR employment_type = '' OR employment_type IS NULL", EmploymentSalaried).
		Count(&overview.SalariedStaff)
	db.Model(&StaffProfile{}).
		Where("employment_type = ?", EmploymentFreelancer).
		Count(&overview.PanelDoctors)

	return response.JSON(c, fiber.StatusOK, overview, "HR dashboard overview fetched")
}

// GetOnlineDoctors returns a list of online doctors
// GetOnlineDoctors returns a list of online doctors
// @Summary Get Online Doctors
// @Description Fetch all doctors currently marked as 'online' for video consultations
// @Tags HR
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} response.StandardResponse{data=[]identity.User} "Doctors fetched"
// @Router /api/v1/hr/dashboard/doctors/online [get]
func (h *Handler) GetOnlineDoctors(c *fiber.Ctx) error {
	var doctors []identity.User
	if err := database.DB.Where("role = ? AND status = ?", "doctor", "online").Find(&doctors).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch online doctors")
	}

	return response.JSON(c, fiber.StatusOK, doctors, "Online doctors fetched")
}

// GetHospitalStaffDistribution returns staff counts by hospital
// GetHospitalStaffDistribution returns staff counts by hospital
// @Summary Get Hospital Staff Distribution
// @Description Fetch a breakdown of staff counts across different hospital IDs
// @Tags HR
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} response.StandardResponse{data=map[string]interface{}} "Distribution fetched"
// @Router /api/v1/hr/dashboard/hospitals/staff [get]
func (h *Handler) GetHospitalStaffDistribution(c *fiber.Ctx) error {
	var distribution []struct {
		HospitalID uint  `json:"hospital_id"`
		StaffCount int64 `json:"staff_count"`
	}

	database.DB.Model(&identity.User{}).
		Where("role != ?", "patient").
		Select("hospital_id, count(*) as staff_count").
		Group("hospital_id").
		Scan(&distribution)

	return response.JSON(c, fiber.StatusOK, distribution, "Hospital staff distribution fetched")
}

// GetPharmacyStaff returns a list of pharmacy staff
// GetPharmacyStaff returns a list of pharmacy staff
// @Summary Get Pharmacy Staff
// @Description Fetch all staff members assigned to pharmacy roles (managers, pharmacists)
// @Tags HR
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} response.StandardResponse{data=[]identity.User} "Pharmacy staff fetched"
// @Router /api/v1/hr/dashboard/pharmacy/staff [get]
func (h *Handler) GetPharmacyStaff(c *fiber.Ctx) error {
	var staff []identity.User
	if err := database.DB.Where("role IN ?", []string{"pharmacy_manager", "pharmacist"}).Find(&staff).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch pharmacy staff")
	}

	return response.JSON(c, fiber.StatusOK, staff, "Pharmacy staff fetched")
}

// GetStaffDirectory returns a list of all non-patient users
// GetStaffDirectory returns a list of all non-patient users
// @Summary Get Staff Directory
// @Description Fetch a complete directory of all non-patient users (staff members)
// @Tags HR
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} response.StandardResponse{data=[]identity.User} "Staff directory fetched"
// @Router /api/v1/hr/dashboard/staff [get]
func (h *Handler) GetStaffDirectory(c *fiber.Ctx) error {
	var users []identity.User
	if err := database.DB.Where("role != ?", "patient").Find(&users).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch staff directory")
	}

	// Remove password hashes
	for i := range users {
		users[i].PasswordHash = ""
	}

	return response.JSON(c, fiber.StatusOK, users, "Staff directory fetched")
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	hrGroup := router.Group("/hr", middleware.Protected())

	hrGroup.Post("/attendance/clock-in", h.ClockIn)
	hrGroup.Post("/attendance/clock-out/:attendanceId", h.ClockOut)
	hrGroup.Get("/payroll/:userId", middleware.HRManagerOnly(), h.CalculatePayroll)
	hrGroup.Get("/roster", middleware.HRManagerOnly(), h.GetHRRoster)

	// Onboarding Module
	onboarding := hrGroup.Group("/onboarding", middleware.HRManagerOnly())
	if h.onboardingHandler != nil {
		onboarding.Get("/", h.onboardingHandler.GetAll)
		onboarding.Get("/:id", h.onboardingHandler.GetByID)
		onboarding.Post("/initiate", h.onboardingHandler.InitiateOnboarding)
		onboarding.Put("/:id/documents", h.onboardingHandler.UpdateDocument)
		onboarding.Put("/:id/tasks", h.onboardingHandler.UpdateTask)
		onboarding.Put("/:id/status", h.onboardingHandler.UpdateStatus)
		onboarding.Post("/:id/it-request", h.onboardingHandler.SendITRequest)
		onboarding.Post("/:id/document-reminder", h.onboardingHandler.SendDocumentReminder)
	}

	// Dashboard Features (secured for HR Manager)
	dashboard := hrGroup.Group("/dashboard", middleware.HRManagerOnly())
	dashboard.Get("/overview", h.GetDashboardOverview)
	dashboard.Get("/staff", h.GetStaffDirectory)
	dashboard.Get("/doctors/online", h.GetOnlineDoctors)
	dashboard.Get("/panel-doctors/earnings", PanelDoctorEarnings)
	dashboard.Get("/hospitals/staff", h.GetHospitalStaffDistribution)
	dashboard.Get("/pharmacy/staff", h.GetPharmacyStaff)
}

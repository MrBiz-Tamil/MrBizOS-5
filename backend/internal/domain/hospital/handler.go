package hospital

import (
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetHospitalDashboard(c *fiber.Ctx) error {
	hospID := uint(1)
	if uid, ok := c.Locals("hospital_id").(uint); ok && uid > 0 {
		hospID = uid
	}
	stats, err := h.svc.GetDashboardStats(c.Context(), hospID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, stats, "Hospital dashboard stats fetched")
}

func (h *Handler) GetOPDQueue(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	queue, err := h.svc.GetOPDQueue(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, queue, "OPD queue fetched")
}

func (h *Handler) AddWalkInToQueue(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id")) // Hospital ID
	var req struct {
		PatientID uint   `json:"patient_id"`
		DoctorID  uint   `json:"doctor_id"`
		Notes     string `json:"notes"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	appt, err := h.svc.AddWalkInToQueue(c.Context(), uint(id), req.PatientID, req.DoctorID, req.Notes)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, appt, "Walk-in added to queue")
}

func (h *Handler) AdmitPatient(c *fiber.Ctx) error {
	hospitalID, _ := strconv.Atoi(c.Params("id"))
	var req struct {
		PatientID uint `json:"patient_id"`
		DoctorID  uint `json:"doctor_id"`
		BedID     uint `json:"bed_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	admission, err := h.svc.AdmitPatient(c.Context(), uint(hospitalID), req.PatientID, req.DoctorID, req.BedID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ADMISSION_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, admission, "Patient admitted successfully")
}

func (h *Handler) DischargePatient(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id")) // Admission ID
	var req struct {
		Summary string `json:"summary"`
	}
	if err := c.BodyParser(&req); err != nil {
		// Summary might be optional for some flows
	}

	err := h.svc.DischargePatient(c.Context(), uint(id), req.Summary)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DISCHARGE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Patient discharged successfully")
}

func (h *Handler) GetIPDAdmissions(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	admissions, err := h.svc.GetIPDAdmissions(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, admissions, "IPD admissions fetched")
}

func (h *Handler) GetHospitalInventory(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	inventory, err := h.svc.GetHospitalInventory(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, inventory, "Hospital inventory fetched")
}

func (h *Handler) GetAvailableBeds(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	beds, err := h.svc.GetAvailableBeds(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, beds, "Available beds fetched")
}

func (h *Handler) UpdateHospitalInventory(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var req struct {
		QuantityChange int    `json:"quantity_change"`
		Reason         string `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	err := h.svc.UpdateHospitalInventory(c.Context(), uint(id), req.QuantityChange, req.Reason)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Stock updated successfully")
}

func (h *Handler) CreateWardServiceRequest(c *fiber.Ctx) error {
	hospitalID, _ := strconv.Atoi(c.Params("id"))
	var req struct {
		PatientID     *uint  `json:"patient_id"`
		RequestedByID uint   `json:"requested_by_id"`
		BedID         *uint  `json:"bed_id"`
		Items         []byte `json:"items"`
		Priority      string `json:"priority"`
		Notes         string `json:"notes"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	wardReq, err := h.svc.CreateWardServiceRequest(c.Context(), uint(hospitalID), req.PatientID, req.RequestedByID, req.BedID, req.Items, req.Priority, req.Notes)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, wardReq, "Ward service request created")
}

func (h *Handler) GetWardServiceRequests(c *fiber.Ctx) error {
	hospitalID, _ := strconv.Atoi(c.Params("id"))
	requests, err := h.svc.GetWardServiceRequests(c.Context(), uint(hospitalID))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, requests, "Ward service requests fetched")
}

func (h *Handler) UpdateWardServiceRequest(c *fiber.Ctx) error {
	requestID, _ := strconv.Atoi(c.Params("req_id"))
	var req struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	err := h.svc.UpdateWardServiceRequest(c.Context(), uint(requestID), req.Status)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Status updated successfully")
}

// --- Admin CRUD Handlers ---

func (h *Handler) ListHospitals(c *fiber.Ctx) error {
	hType := c.Query("type", "all")
	hospitals, err := h.svc.ListAllHospitals(c.Context(), hType)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, hospitals, "Hospitals listed")
}

func (h *Handler) CreateHospitalHandler(c *fiber.Ctx) error {
	var req struct {
		Name         string `json:"name"`
		Address      string `json:"address"`
		ContactInfo  string `json:"contact_info"`
		Type         string `json:"type"`
		HospitalType string `json:"hospital_type"`
		FranchiseID  *uint  `json:"franchise_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	hosp, err := h.svc.CreateHospital(c.Context(), req.Name, req.Address, req.ContactInfo, req.Type, req.HospitalType, req.FranchiseID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, hosp, "Hospital created")
}

func (h *Handler) UpdateHospitalHandler(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var req struct {
		Name         string `json:"name"`
		Address      string `json:"address"`
		ContactInfo  string `json:"contact_info"`
		Type         string `json:"type"`
		HospitalType string `json:"hospital_type"`
		IsActive     bool   `json:"is_active"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	hosp, err := h.svc.UpdateHospital(c.Context(), uint(id), req.Name, req.Address, req.ContactInfo, req.Type, req.HospitalType, req.IsActive)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, hosp, "Hospital updated")
}

func (h *Handler) ToggleHospitalHandler(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	hosp, err := h.svc.ToggleHospital(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, hosp, "Hospital status toggled")
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	hospGroup := router.Group("/hospital", middleware.EnforceTenancy())
	hospGroup.Get("/dashboard", h.GetHospitalDashboard)
	hospGroup.Get("/wards", h.ListWards)
	hospGroup.Get("/beds", h.ListBeds)
	hospGroup.Get("/tasks", middleware.Protected(), h.ListTasks)
	hospGroup.Post("/tasks", middleware.Protected(), h.CreateTask)
	hospGroup.Post("/tasks/:id/complete", middleware.Protected(), h.CompleteTask)
	hospGroup.Get("/shifts", middleware.Protected(), h.ListShifts)
	hospGroup.Post("/shifts", middleware.Protected(), h.CreateShift)

	hospitalsGroup := router.Group("/hospitals")
	hospitalsGroup.Get("/", h.ListHospitals)
	hospitalsGroup.Post("/", middleware.AdminOnly(), h.CreateHospitalHandler)

	tenantHosp := hospitalsGroup.Group("/:id", middleware.Protected(), middleware.EnforceTenancy())
	tenantHosp.Put("/", middleware.HospitalAdminOnly(), h.UpdateHospitalHandler)
	tenantHosp.Patch("/toggle", middleware.AdminOnly(), h.ToggleHospitalHandler)
	tenantHosp.Get("/opd", h.GetOPDQueue)
	tenantHosp.Get("/ipd", h.GetIPDAdmissions)
	tenantHosp.Get("/beds/available", h.GetAvailableBeds)
	tenantHosp.Post("/admit", h.AdmitPatient)
	tenantHosp.Post("/opd/call-next", h.CallNextPatient)
	tenantHosp.Post("/opd/complete", h.CompleteOPDConsultation)
	tenantHosp.Post("/walkin", h.AddWalkInToQueue)
	tenantHosp.Post("/ward-services", h.CreateWardServiceRequest)
	tenantHosp.Get("/ward-services", h.GetWardServiceRequests)

	// Ward Service Requests and Inventory under tenant checks
	hospitalsGroup.Put("/ward-services/:req_id/status", middleware.Protected(), h.UpdateWardServiceRequest)

	// Inventory
	inventoryGroup := router.Group("/inventory", middleware.Protected(), middleware.EnforceTenancy())
	inventoryGroup.Get("/hospital/:id", h.GetHospitalInventory)
	inventoryGroup.Put("/:id/stock", h.UpdateHospitalInventory)
}

// ── New handlers ──────────────────────────────────────────────────────────────

func (h *Handler) ListWards(c *fiber.Ctx) error {
	hospitalID := uint(c.QueryInt("hospital_id"))
	if uid, ok := c.Locals("hospital_id").(uint); ok && uid > 0 {
		hospitalID = uid
	}
	if hospitalID == 0 {
		hospitalID = 1
	}
	wards, err := h.svc.ListWards(c.Context(), hospitalID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	if wards == nil {
		wards = []Ward{}
	}
	return response.JSON(c, fiber.StatusOK, wards, "Wards fetched")
}

func (h *Handler) ListBeds(c *fiber.Ctx) error {
	wardID := uint(c.QueryInt("ward_id"))
	hospitalID := uint(c.QueryInt("hospital_id"))
	if uid, ok := c.Locals("hospital_id").(uint); ok && uid > 0 {
		hospitalID = uid
	}
	if hospitalID == 0 {
		hospitalID = 1
	}

	var beds []Bed
	var err error
	if wardID > 0 {
		beds, err = h.svc.ListBedsByWard(c.Context(), wardID)
	} else {
		beds, err = h.svc.ListBedsByHospital(c.Context(), hospitalID)
	}
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	if beds == nil {
		beds = []Bed{}
	}
	return response.JSON(c, fiber.StatusOK, beds, "Beds fetched")
}

func (h *Handler) CallNextPatient(c *fiber.Ctx) error {
	hospitalID, _ := strconv.Atoi(c.Params("id"))
	next, err := h.svc.CallNextPatient(c.Context(), uint(hospitalID))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NO_WAITING", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, next, "Patient called")
}

func (h *Handler) CompleteOPDConsultation(c *fiber.Ctx) error {
	var body struct {
		PatientID uint `json:"patient_id"`
		OPDID     uint `json:"opd_id"`
	}
	_ = c.BodyParser(&body)
	opdID := body.OPDID
	if opdID == 0 {
		opdID = body.PatientID // backward compat — frontend passes queueId as patient_id
	}
	if opdID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "opd_id is required")
	}
	if err := h.svc.CompleteOPDConsultation(c.Context(), opdID); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "COMPLETE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "OPD consultation completed")
}

func (h *Handler) ListTasks(c *fiber.Ctx) error {
	hospitalID := uint(c.QueryInt("hospital_id"))
	if uid, ok := c.Locals("hospital_id").(uint); ok && uid > 0 {
		hospitalID = uid
	}
	if hospitalID == 0 {
		hospitalID = 1
	}
	tasks, err := h.svc.ListTasks(c.Context(), hospitalID, c.Query("status"))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	if tasks == nil {
		tasks = []NursingTask{}
	}
	return response.JSON(c, fiber.StatusOK, tasks, "Tasks fetched")
}

func (h *Handler) CreateTask(c *fiber.Ctx) error {
	var t NursingTask
	if err := c.BodyParser(&t); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if uid, ok := c.Locals("user_id").(uint); ok {
		t.CreatedByID = uid
	}
	if uid, ok := c.Locals("hospital_id").(uint); ok && uid > 0 {
		t.HospitalID = uid
	}
	if t.HospitalID == 0 {
		t.HospitalID = 1
	}
	if err := h.svc.CreateTask(c.Context(), &t); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, t, "Task created")
}

func (h *Handler) CompleteTask(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var body struct {
		Notes string `json:"notes"`
	}
	_ = c.BodyParser(&body)
	if err := h.svc.CompleteTask(c.Context(), uint(id), body.Notes); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "COMPLETE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Task completed")
}

func (h *Handler) ListShifts(c *fiber.Ctx) error {
	hospitalID := uint(c.QueryInt("hospital_id"))
	if uid, ok := c.Locals("hospital_id").(uint); ok && uid > 0 {
		hospitalID = uid
	}
	if hospitalID == 0 {
		hospitalID = 1
	}
	shifts, err := h.svc.ListShifts(c.Context(), hospitalID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	if shifts == nil {
		shifts = []ShiftSchedule{}
	}
	return response.JSON(c, fiber.StatusOK, shifts, "Shifts fetched")
}

func (h *Handler) CreateShift(c *fiber.Ctx) error {
	var s ShiftSchedule
	if err := c.BodyParser(&s); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if uid, ok := c.Locals("hospital_id").(uint); ok && uid > 0 {
		s.HospitalID = uid
	}
	if s.HospitalID == 0 {
		s.HospitalID = 1
	}
	if err := h.svc.CreateShift(c.Context(), &s); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, s, "Shift created")
}

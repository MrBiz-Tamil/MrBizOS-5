//go:build ignore
// +build ignore

package handlers

import (
	"drayya-core/internal/hospital"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// GetHospitalDashboard godoc
// @Summary Get hospital dashboard stats
// @Description Fetch statistics for the hospital dashboard
// @Tags Hospital
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /hospital/dashboard [get]
func GetHospitalDashboard(c *fiber.Ctx) error {
	stats, err := hospital.GetDashboardStats(1)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DASHBOARD_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, stats, "Fetched")
}

func GetOPDQueue(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	queue, err := hospital.GetOPDQueue(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "OPD_QUEUE_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, queue, "Fetched")
}

func AddWalkInToQueue(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id")) // Hospital ID
	var req struct {
		PatientID uint   `json:"patient_id"`
		DoctorID  uint   `json:"doctor_id"`
		Notes     string `json:"notes"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	appt, err := hospital.AddWalkInToQueue(uint(id), req.PatientID, req.DoctorID, req.Notes)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ADD_WALK_IN_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, appt, "Created")
}

func AdmitPatient(c *fiber.Ctx) error {
	hospitalID, _ := strconv.Atoi(c.Params("id"))
	var req struct {
		PatientID uint `json:"patient_id"`
		DoctorID  uint `json:"doctor_id"`
		BedID     uint `json:"bed_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	admission, err := hospital.AdmitPatient(uint(hospitalID), req.PatientID, req.DoctorID, req.BedID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ADMIT_PATIENT_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, admission, "Created")
}

func DischargePatient(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id")) // Admission ID
	var req struct {
		Summary string `json:"summary"`
	}
	if err := c.BodyParser(&req); err != nil {
		// Summary might be optional for some flows
	}

	err := hospital.DischargePatient(uint(id), req.Summary)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DISCHARGE_PATIENT_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Patient discharged successfully")
}

func GetIPDAdmissions(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	admissions, err := hospital.GetIPDAdmissions(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "IPD_ADMISSIONS_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, admissions, "Fetched")
}

func GetHospitalInventory(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	inventory, err := hospital.GetHospitalInventory(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "INVENTORY_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, inventory, "Fetched")
}

func GetAvailableBeds(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	beds, err := hospital.GetAvailableBeds(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "BEDS_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, beds, "Fetched")
}

func UpdateHospitalInventory(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var req struct {
		QuantityChange int    `json:"quantity_change"`
		Reason         string `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	err := hospital.UpdateHospitalInventory(uint(id), req.QuantityChange, req.Reason)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "INVENTORY_UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Stock updated successfully")
}

func CreateWardServiceRequest(c *fiber.Ctx) error {
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

	wardReq, err := hospital.CreateWardServiceRequest(uint(hospitalID), req.PatientID, req.RequestedByID, req.BedID, req.Items, req.Priority, req.Notes)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "WARD_SERVICE_CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, wardReq, "Created")
}

func GetWardServiceRequests(c *fiber.Ctx) error {
	hospitalID, _ := strconv.Atoi(c.Params("id"))
	requests, err := hospital.GetWardServiceRequests(uint(hospitalID))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "WARD_SERVICE_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, requests, "Fetched")
}

func UpdateWardServiceRequest(c *fiber.Ctx) error {
	requestID, _ := strconv.Atoi(c.Params("req_id"))
	var req struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	err := hospital.UpdateWardServiceRequest(uint(requestID), req.Status)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "WARD_SERVICE_UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Status updated successfully")
}

// --- Admin CRUD Handlers ---

// ListHospitals godoc
// @Summary List all hospitals
// @Description Fetch a list of all hospitals, optionally filtered by type
// @Tags Hospital
// @Produce json
// @Param type query string false "Hospital Type (e.g., all, private, public)"
// @Success 200 {array} object
// @Router /hospitals [get]
func ListHospitals(c *fiber.Ctx) error {
	hType := c.Query("type", "all")
	hospitals, err := hospital.ListAllHospitals(hType)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "HOSPITALS_FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, hospitals, "Fetched")
}

func CreateHospitalHandler(c *fiber.Ctx) error {
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
	h, err := hospital.CreateHospital(req.Name, req.Address, req.ContactInfo, req.Type, req.HospitalType, req.FranchiseID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_HOSPITAL_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, h, "Created")
}

func UpdateHospitalHandler(c *fiber.Ctx) error {
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
	h, err := hospital.UpdateHospital(uint(id), req.Name, req.Address, req.ContactInfo, req.Type, req.HospitalType, req.IsActive)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_HOSPITAL_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, h, "Fetched")
}

func ToggleHospitalHandler(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	h, err := hospital.ToggleHospital(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "TOGGLE_HOSPITAL_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, h, "Fetched")
}

func RegisterHospitalRoutes(router fiber.Router) {
	hospGroup := router.Group("/hospital")
	hospGroup.Get("/dashboard", GetHospitalDashboard)

	hospitalsGroup := router.Group("/hospitals")
	hospitalsGroup.Get("/", ListHospitals)
	hospitalsGroup.Post("/", middleware.Protected(), CreateHospitalHandler)
	hospitalsGroup.Put("/:id", middleware.Protected(), UpdateHospitalHandler)
	hospitalsGroup.Patch("/:id/toggle", middleware.Protected(), ToggleHospitalHandler)
	hospitalsGroup.Get("/:id/opd", GetOPDQueue)
	hospitalsGroup.Get("/:id/ipd", GetIPDAdmissions)
	hospitalsGroup.Get("/:id/beds/available", GetAvailableBeds)
	hospitalsGroup.Post("/:id/admit", middleware.Protected(), AdmitPatient)
	hospitalsGroup.Post("/ipd/:id/discharge", middleware.Protected(), DischargePatient)

	// Ward Service Requests
	hospitalsGroup.Post("/:id/ward-services", middleware.Protected(), CreateWardServiceRequest)
	hospitalsGroup.Get("/:id/ward-services", middleware.Protected(), GetWardServiceRequests)
	hospitalsGroup.Put("/ward-services/:req_id/status", middleware.Protected(), UpdateWardServiceRequest)

	// Inventory
	inventoryGroup := router.Group("/inventory")
	inventoryGroup.Get("/hospital/:id", GetHospitalInventory)
	inventoryGroup.Put("/:id/stock", UpdateHospitalInventory)
}

package travel

import (
	"strconv"

	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r fiber.Router) {
	tr := r.Group("/travel")
	tr.Get("/destinations", h.ListDestinations)
	tr.Post("/destinations", h.CreateDestination)
	tr.Get("/packages", h.ListTourPackages)
	tr.Post("/packages", h.CreateTourPackage)
	tr.Get("/packages/:id", h.GetTourPackage)
	tr.Post("/bookings", h.CreateBooking)
	tr.Get("/bookings", h.ListBookings)

	v := r.Group("/visa")
	v.Get("/checklists/:country", h.GetVisaChecklist)
	v.Post("/checklists", h.CreateVisaChecklist)
	v.Post("/applications", h.SubmitVisaApplication)
	v.Get("/applications", h.ListVisaApplications)
	v.Put("/applications/:id/status", h.UpdateVisaStatus)
}

func (h *Handler) ListDestinations(c *fiber.Ctx) error {
	dests, err := h.svc.ListDestinations()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, dests, "Destinations fetched successfully")
}

func (h *Handler) CreateDestination(c *fiber.Ctx) error {
	var dest Destination
	if err := c.BodyParser(&dest); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	created, err := h.svc.CreateDestination(&dest)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, created, "Destination created successfully")
}

func (h *Handler) ListTourPackages(c *fiber.Ctx) error {
	var destID uint
	if d := c.Query("destination_id"); d != "" {
		if id, err := strconv.ParseUint(d, 10, 32); err == nil {
			destID = uint(id)
		}
	}

	pkgs, err := h.svc.ListTourPackages(destID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, pkgs, "Tour packages fetched successfully")
}

func (h *Handler) CreateTourPackage(c *fiber.Ctx) error {
	var pkg TourPackage
	if err := c.BodyParser(&pkg); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	created, err := h.svc.CreateTourPackage(&pkg)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, created, "Tour package created successfully")
}

func (h *Handler) GetTourPackage(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Package ID must be an integer")
	}

	pkg, err := h.svc.GetTourPackage(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Tour package not found")
	}
	return response.JSON(c, fiber.StatusOK, pkg, "Tour package fetched successfully")
}

func (h *Handler) CreateBooking(c *fiber.Ctx) error {
	var req struct {
		CustomerID    uint   `json:"customer_id"`
		TourPackageID uint   `json:"tour_package_id"`
		Passengers    int    `json:"passengers"`
		TravelDate    string `json:"travel_date"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	if uid, ok := c.Locals("user_id").(uint); ok {
		req.CustomerID = uid
	} else if req.CustomerID == 0 {
		req.CustomerID = 1
	}

	booking, err := h.svc.CreateBooking(req.CustomerID, req.TourPackageID, req.Passengers, req.TravelDate)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "BOOKING_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, booking, "Travel booking created successfully")
}

func (h *Handler) ListBookings(c *fiber.Ctx) error {
	var custID uint
	if cid := c.Query("customer_id"); cid != "" {
		if id, err := strconv.ParseUint(cid, 10, 32); err == nil {
			custID = uint(id)
		}
	}

	bookings, err := h.svc.ListBookings(custID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, bookings, "Travel bookings fetched successfully")
}

func (h *Handler) GetVisaChecklist(c *fiber.Ctx) error {
	country := c.Params("country")
	visaType := c.Query("type", "TOURIST")

	checklist, err := h.svc.GetVisaChecklist(country, visaType)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Visa checklist not found")
	}
	return response.JSON(c, fiber.StatusOK, checklist, "Visa checklist fetched successfully")
}

func (h *Handler) CreateVisaChecklist(c *fiber.Ctx) error {
	var checklist VisaChecklist
	if err := c.BodyParser(&checklist); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	created, err := h.svc.CreateVisaChecklist(&checklist)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, created, "Visa checklist created successfully")
}

func (h *Handler) SubmitVisaApplication(c *fiber.Ctx) error {
	var req struct {
		ApplicantID    uint   `json:"applicant_id"`
		CountryCode    string `json:"country_code"`
		VisaType       string `json:"visa_type"`
		PassportNumber string `json:"passport_number"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	if uid, ok := c.Locals("user_id").(uint); ok {
		req.ApplicantID = uid
	} else if req.ApplicantID == 0 {
		req.ApplicantID = 1
	}

	app, err := h.svc.SubmitVisaApplication(req.ApplicantID, req.CountryCode, req.VisaType, req.PassportNumber)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "SUBMISSION_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, app, "Visa application submitted successfully")
}

func (h *Handler) ListVisaApplications(c *fiber.Ctx) error {
	var applicantID uint
	if aid := c.Query("applicant_id"); aid != "" {
		if id, err := strconv.ParseUint(aid, 10, 32); err == nil {
			applicantID = uint(id)
		}
	}

	apps, err := h.svc.ListVisaApplications(applicantID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, apps, "Visa applications fetched successfully")
}

func (h *Handler) UpdateVisaStatus(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Application ID must be an integer")
	}

	var req struct {
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	updated, err := h.svc.UpdateVisaStatus(uint(id), req.Status, req.Notes)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, updated, "Visa application status updated successfully")
}

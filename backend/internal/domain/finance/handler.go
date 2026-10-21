package finance

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
	f := r.Group("/finance")
	f.Get("/profitability", h.GetProfitabilityReport)
	f.Get("/cost-centers", h.ListCostCenters)
	f.Post("/cost-centers", h.CreateCostCenter)
	f.Get("/entries", h.ListEntries)
	f.Post("/entries", h.RecordEntry)
}

func (h *Handler) GetProfitabilityReport(c *fiber.Ctx) error {
	report, err := h.svc.GetGroupProfitabilityReport()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "REPORT_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, report, "Group profitability report generated successfully")
}

func (h *Handler) ListCostCenters(c *fiber.Ctx) error {
	ccs, err := h.svc.ListCostCenters()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, ccs, "Cost centers fetched successfully")
}

func (h *Handler) CreateCostCenter(c *fiber.Ctx) error {
	var cc CostCenter
	if err := c.BodyParser(&cc); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	created, err := h.svc.CreateCostCenter(&cc)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, created, "Cost center created successfully")
}

func (h *Handler) ListEntries(c *fiber.Ctx) error {
	var buID uint
	if b := c.Query("business_unit_id"); b != "" {
		if id, err := strconv.ParseUint(b, 10, 32); err == nil {
			buID = uint(id)
		}
	}

	entries, err := h.svc.ListEntries(buID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, entries, "Financial ledger entries fetched successfully")
}

func (h *Handler) RecordEntry(c *fiber.Ctx) error {
	var entry FinancialLedgerEntry
	if err := c.BodyParser(&entry); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	recorded, err := h.svc.RecordEntry(&entry)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "RECORD_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, recorded, "Financial ledger entry recorded successfully")
}

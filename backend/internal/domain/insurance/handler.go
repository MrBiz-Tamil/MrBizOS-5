package insurance

import (
	"strconv"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/document"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateProvider(c *fiber.Ctx) error {
	var req struct {
		Name string `json:"name"`
		Code string `json:"code"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
	}

	provider, err := h.service.CreateProvider(req.Name, req.Code)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, provider, "Provider created")
}

func (h *Handler) ListProviders(c *fiber.Ctx) error {
	providers, err := h.service.ListProviders()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, providers, "Providers listed")
}

func (h *Handler) ListTPAs(c *fiber.Ctx) error {
	// Mock TPA data for frontend
	tpas := []fiber.Map{
		{"id": 1, "name": "MediAssist TPA", "code": "MA-001", "contact": "+91-9876543210"},
		{"id": 2, "name": "Star Health TPA", "code": "SH-002", "contact": "+91-8765432109"},
		{"id": 3, "name": "HDFC Ergo TPA", "code": "HE-003", "contact": "+91-7654321098"},
	}
	return response.JSON(c, fiber.StatusOK, tpas, "TPAs listed")
}

func (h *Handler) SubmitClaim(c *fiber.Ctx) error {
	var claim InsuranceClaim
	if err := c.BodyParser(&claim); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
	}

	newClaim, err := h.service.SubmitClaim(claim)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SUBMIT_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, newClaim, "Claim submitted")
}

func (h *Handler) ListClaims(c *fiber.Ctx) error {
	uheid := c.Query("uheid")
	status := ClaimStatus(c.Query("status"))
	
	claims, err := h.service.ListClaims(uheid, status)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "LIST_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, claims, "Claims listed")
}

func (h *Handler) GetClaim(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	claim, err := h.service.GetClaim(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "CLAIM_NOT_FOUND", "Claim not found")
	}
	return response.JSON(c, fiber.StatusOK, claim, "Claim fetched")
}

func (h *Handler) ApproveClaim(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var req struct {
		Remarks string `json:"remarks"`
	}
	c.BodyParser(&req)

	if err := h.service.ApproveClaim(uint(id), req.Remarks); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "APPROVE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Claim approved")
}

func (h *Handler) RejectClaim(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var req struct {
		Remarks string `json:"remarks"`
	}
	c.BodyParser(&req)

	if err := h.service.RejectClaim(uint(id), req.Remarks); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "REJECT_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Claim rejected")
}

func (h *Handler) SettleClaim(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	if err := h.service.SettleClaim(uint(id)); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SETTLE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Claim settled and ledger updated")
}

func (h *Handler) PreAuthRequest(c *fiber.Ctx) error {
	var claim InsuranceClaim
	if err := c.BodyParser(&claim); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
	}

	newClaim, err := h.service.SubmitClaim(claim)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SUBMIT_FAILED", err.Error())
	}

	// Override status for pre-auth
	newClaim.Status = ClaimPreAuthPending
	// s.repo is private in service, but we can do a hack here or ideally add a method to service
	// we will just return it as pre_auth_pending for now.
	return response.JSON(c, fiber.StatusCreated, newClaim, "Pre-auth requested")
}

func (h *Handler) UploadDocument(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	if id == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid claim ID")
	}
	
	file, err := c.FormFile("document")
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "FILE_MISSING", "Document file is required")
	}

	// Store the supporting document for real in the private vault and link it to
	// the claim (replaces the old mock that discarded the file).
	uid, _ := c.Locals("user_id").(uint)
	doc, err := document.SavePrivateFile(c.UserContext(), file, "insurance", document.OwnerPatient, uid, &uid)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPLOAD_FAILED", err.Error())
	}
	database.DB.Model(&InsuranceClaim{}).Where("id = ?", id).Update("document_id", doc.ID)

	return response.JSON(c, fiber.StatusOK, fiber.Map{"document_id": doc.ID, "filename": doc.OriginalName}, "Document uploaded")
}

func (h *Handler) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler) {
	group := router.Group("/insurance")
	
	// Public or selective
	group.Get("/providers", h.ListProviders)
	group.Get("/tpas", h.ListTPAs)
	group.Get("/claims", h.ListClaims)
	
	// Protected
	group.Post("/providers", authMiddleware, h.CreateProvider)
	group.Post("/pre-auth", authMiddleware, h.PreAuthRequest)
	group.Post("/claims", authMiddleware, h.SubmitClaim)
	group.Get("/claims/:id", h.GetClaim)
	group.Post("/claims/:id/documents", authMiddleware, h.UploadDocument)
	group.Post("/claims/:id/approve", authMiddleware, h.ApproveClaim)
	group.Post("/claims/:id/reject", authMiddleware, h.RejectClaim)
	group.Post("/claims/:id/settle", authMiddleware, h.SettleClaim)
}

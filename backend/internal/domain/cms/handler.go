package cms

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
	c := r.Group("/cms")
	c.Get("/sites", h.ListSites)
	c.Post("/sites", h.CreateSite)
	c.Get("/sites/resolve", h.ResolveSite)
	c.Get("/pages", h.ListPages)
	c.Post("/pages", h.CreatePage)
	c.Get("/pages/resolve", h.ResolvePage)
}

func (h *Handler) ListSites(c *fiber.Ctx) error {
	sites, err := h.svc.ListSites()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, sites, "Sites fetched successfully")
}

func (h *Handler) CreateSite(c *fiber.Ctx) error {
	var site SiteConfig
	if err := c.BodyParser(&site); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	created, err := h.svc.CreateSite(&site)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, created, "Site configuration created successfully")
}

func (h *Handler) ResolveSite(c *fiber.Ctx) error {
	domain := c.Query("domain")
	if domain == "" {
		domain = c.Hostname()
	}

	site, err := h.svc.GetSiteByDomain(domain)
	if err != nil {
		// Fallback to default healthcare site configuration for localhost/dev
		defaultSite := &SiteConfig{
			ID:             1,
			OrganizationID: 1,
			BusinessUnitID: 2,
			Domain:         domain,
			BrandName:      "Dr.Ayya Healthcare & Group Business OS",
			PrimaryColor:   "#10b981",
			SecondaryColor: "#0f172a",
			FontFamily:     "Inter",
			IsActive:       true,
		}
		return response.JSON(c, fiber.StatusOK, defaultSite, "Default site resolved")
	}
	return response.JSON(c, fiber.StatusOK, site, "Site resolved successfully")
}

func (h *Handler) ListPages(c *fiber.Ctx) error {
	var siteID uint
	if s := c.Query("site_id"); s != "" {
		if id, err := strconv.ParseUint(s, 10, 32); err == nil {
			siteID = uint(id)
		}
	}

	pages, err := h.svc.ListPages(siteID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, pages, "CMS pages fetched successfully")
}

func (h *Handler) CreatePage(c *fiber.Ctx) error {
	var page CMSPage
	if err := c.BodyParser(&page); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	created, err := h.svc.CreatePage(&page)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, created, "CMS page created successfully")
}

func (h *Handler) ResolvePage(c *fiber.Ctx) error {
	slug := c.Query("slug", "/")
	var siteID uint = 1
	if s := c.Query("site_id"); s != "" {
		if id, err := strconv.ParseUint(s, 10, 32); err == nil {
			siteID = uint(id)
		}
	}

	page, err := h.svc.GetPageBySlug(siteID, slug)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "CMS page not found")
	}
	return response.JSON(c, fiber.StatusOK, page, "CMS page fetched successfully")
}

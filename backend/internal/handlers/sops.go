package handlers

import (
	"strconv"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SOP is a Standard Operating Procedure document — used by hospital, clinic
// and pharmacy staff for clinical protocols, compliance and onboarding.
type SOP struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Title        string         `gorm:"not null" json:"title"`
	Content      string         `gorm:"type:text" json:"content"`
	Department   string         `gorm:"index" json:"department"` // clinical, pharmacy, lab, hr, admin
	Tags         string         `json:"tags"`                    // JSON-encoded array string
	IsPublished  bool           `gorm:"default:false" json:"is_published"`
	AccessRoles  string         `json:"access_roles"` // comma-separated roles
	CreatedBy    uint           `json:"created_by"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (SOP) TableName() string { return "sops" }

// AutoMigrateSOPs runs the migration for the sops table.
func AutoMigrateSOPs(db *gorm.DB) error {
	return db.AutoMigrate(&SOP{})
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func ListSOPs(c *fiber.Ctx) error {
	var sops []SOP
	qry := database.DB.Model(&SOP{})

	if dept := c.Query("department"); dept != "" {
		qry = qry.Where("department = ?", dept)
	}
	if search := c.Query("search"); search != "" {
		qry = qry.Where("LOWER(title) LIKE ? OR LOWER(content) LIKE ?",
			"%"+search+"%", "%"+search+"%")
	}

	if err := qry.Order("updated_at DESC").Find(&sops).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	if sops == nil {
		sops = []SOP{}
	}
	return response.JSON(c, fiber.StatusOK, sops, "SOPs fetched")
}

func GetSOPHandler(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid SOP ID")
	}
	var sop SOP
	if err := database.DB.First(&sop, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "SOP not found")
	}
	return response.JSON(c, fiber.StatusOK, sop, "SOP fetched")
}

func CreateSOP(c *fiber.Ctx) error {
	var sop SOP
	if err := c.BodyParser(&sop); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
	}
	if sop.Title == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "title is required")
	}
	if userID, ok := c.Locals("user_id").(uint); ok {
		sop.CreatedBy = userID
	}
	if err := database.DB.Create(&sop).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, sop, "SOP created")
}

func UpdateSOPHandler(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid SOP ID")
	}
	var existing SOP
	if err := database.DB.First(&existing, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "SOP not found")
	}
	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
	}
	delete(updates, "id")
	delete(updates, "created_at")
	delete(updates, "created_by")
	if err := database.DB.Model(&existing).Updates(updates).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	database.DB.First(&existing, id) // re-fetch with new values
	return response.JSON(c, fiber.StatusOK, existing, "SOP updated")
}

func DeleteSOPHandler(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid SOP ID")
	}
	if err := database.DB.Delete(&SOP{}, id).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "SOP deleted")
}

// RegisterSOPRoutes mounts SOP CRUD under /sops.
func RegisterSOPRoutes(router fiber.Router) {
	g := router.Group("/sops")
	g.Get("/", ListSOPs)
	g.Get("/:id", GetSOPHandler)
	g.Post("/", middleware.Protected(), CreateSOP)
	g.Put("/:id", middleware.Protected(), UpdateSOPHandler)
	g.Delete("/:id", middleware.Protected(), DeleteSOPHandler)
}

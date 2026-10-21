package pharmacy

// medicine_image.go — POST /pharmacy/medicines/:id/image
//
// Uploads a product photo for a medicine into the document vault as a public
// (CDN-served) asset. The returned stable URL is saved to medicine.image_url so
// every storefront surface (cards, search, detail page) can render it.

import (
	"strconv"

	"drayya-core/internal/domain/document"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// UploadMedicineImage saves an uploaded product photo and points the medicine
// at it. Multipart field name: "image".
//
// POST /pharmacy/medicines/:id/image
// Auth: PharmacyAdminOnly
func (h *Handler) UploadMedicineImage(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid medicine ID")
	}

	var med Medicine
	if err := h.db.First(&med, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Medicine not found")
	}

	file, err := c.FormFile("image")
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "FILE_MISSING", "image file is required")
	}

	// Store the product photo as a public (CDN-served) asset in the vault. The
	// vault validates type/size and returns a stable public URL. Replaces the
	// old local-disk write, which wasn't safe across multiple replicas.
	doc, err := document.SavePublicAsset(c.UserContext(), file, "product_image", "medicine", uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "UPLOAD_FAILED", err.Error())
	}

	if err := h.db.WithContext(c.UserContext()).Model(&Medicine{}).Where("id = ?", id).Update("image_url", doc.PublicURL).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Image saved but could not be linked")
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"id":        id,
		"image_url": doc.PublicURL,
	}, "Image uploaded")
}

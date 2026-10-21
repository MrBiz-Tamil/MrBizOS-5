package admin

import (
	"strconv"
	"strings"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// ── Doctor Types ──────────────────────────────────────────────────────────────

// ListDoctorTypes returns all 4 AYUSH doctor types.
// GET /doctor-types  (public)
// GET /admin/doctor-types  (admin)
func ListDoctorTypes(c *fiber.Ctx) error {
	var types []identity.DoctorType
	database.DB.Where("is_active = ?", true).Order("id").Find(&types)
	return response.JSON(c, fiber.StatusOK, types, "Doctor types fetched")
}

// UpdateDoctorType allows admin to edit description/icon/color of a type.
// PUT /admin/doctor-types/:id
func UpdateDoctorType(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil || id == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid doctor type ID")
	}
	var existing identity.DoctorType
	if err := database.DB.First(&existing, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Doctor type not found")
	}
	var req struct {
		Description string `json:"description"`
		Icon        string `json:"icon"`
		Color       string `json:"color"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	updates := map[string]interface{}{}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Icon != "" {
		updates["icon"] = req.Icon
	}
	if req.Color != "" {
		updates["color"] = req.Color
	}
	database.DB.Model(&existing).Updates(updates)
	database.DB.First(&existing, id)
	return response.JSON(c, fiber.StatusOK, existing, "Doctor type updated")
}

// ── Sub-Specialties CRUD (admin only) ────────────────────────────────────────

// ListSubSpecialties lists sub-specialties, optionally filtered by type_id.
// GET /admin/doctor-sub-specialties?type_id=1
// GET /doctor-types/:slug/sub-specialties  (public)
func ListSubSpecialties(c *fiber.Ctx) error {
	typeID := c.QueryInt("type_id", 0)
	slug := c.Params("slug")

	q := database.DB.Preload("DoctorType").Where("doctor_sub_specialties.is_active = ?", true)
	if typeID > 0 {
		q = q.Where("doctor_type_id = ?", typeID)
	} else if slug != "" {
		var dt identity.DoctorType
		if err := database.DB.Where("slug = ?", slug).First(&dt).Error; err != nil {
			return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Doctor type not found")
		}
		q = q.Where("doctor_type_id = ?", dt.ID)
	}
	var subs []identity.DoctorSubSpecialty
	q.Order("name").Find(&subs)
	return response.JSON(c, fiber.StatusOK, subs, "Sub-specialties fetched")
}

// CreateSubSpecialty creates a new sub-specialty (admin only).
// POST /admin/doctor-sub-specialties
func CreateSubSpecialty(c *fiber.Ctx) error {
	var req identity.DoctorSubSpecialty
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	if req.Name == "" || req.DoctorTypeID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELDS", "name and doctor_type_id are required")
	}
	if req.Slug == "" {
		req.Slug = strings.ToLower(strings.ReplaceAll(req.Name, " ", "_"))
	}
	userID, _ := c.Locals("user_id").(uint)
	req.CreatedBy = userID
	if err := database.DB.Create(&req).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DB_ERROR", err.Error())
	}
	database.DB.Preload("DoctorType").First(&req, req.ID)
	return response.JSON(c, fiber.StatusCreated, req, "Sub-specialty created")
}

// UpdateSubSpecialty updates a sub-specialty (admin only).
// PUT /admin/doctor-sub-specialties/:id
func UpdateSubSpecialty(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil || id == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	var existing identity.DoctorSubSpecialty
	if err := database.DB.First(&existing, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Sub-specialty not found")
	}
	if err := c.BodyParser(&existing); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	existing.ID = uint(id)
	database.DB.Save(&existing)
	return response.JSON(c, fiber.StatusOK, existing, "Sub-specialty updated")
}

// DeactivateSubSpecialty soft-deletes a sub-specialty (admin only).
// DELETE /admin/doctor-sub-specialties/:id
func DeactivateSubSpecialty(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil || id == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid ID")
	}
	if err := database.DB.Model(&identity.DoctorSubSpecialty{}).Where("id = ?", id).Update("is_active", false).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DB_ERROR", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Sub-specialty deactivated")
}

// ── Doctor Assignment ─────────────────────────────────────────────────────────

// GetDoctorSpecialties returns a doctor's assigned type and sub-specialties.
// GET /admin/doctors/:user_id/specialties
func GetDoctorSpecialties(c *fiber.Ctx) error {
	uid, err := strconv.ParseUint(c.Params("user_id"), 10, 32)
	if err != nil || uid == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid user_id")
	}
	var profile identity.DoctorProfile
	database.DB.Where("user_id = ?", uid).First(&profile)

	var assignments []identity.DoctorSubSpecialtyAssignment
	database.DB.Preload("SubSpecialty.DoctorType").
		Where("doctor_user_id = ? AND is_active = ?", uid, true).
		Find(&assignments)

	var doctorType *identity.DoctorType
	if profile.DoctorTypeID != nil {
		var dt identity.DoctorType
		if err := database.DB.First(&dt, *profile.DoctorTypeID).Error; err == nil {
			doctorType = &dt
		}
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"doctor_user_id":  uid,
		"doctor_type":     doctorType,
		"sub_specialties": assignments,
	}, "Doctor specialties fetched")
}

// SetDoctorType sets a doctor's primary AYUSH type (admin only).
// PUT /admin/doctors/:user_id/type
func SetDoctorType(c *fiber.Ctx) error {
	uid, err := strconv.ParseUint(c.Params("user_id"), 10, 32)
	if err != nil || uid == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid user_id")
	}
	var req struct {
		DoctorTypeID uint `json:"doctor_type_id"`
	}
	if err := c.BodyParser(&req); err != nil || req.DoctorTypeID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "doctor_type_id required")
	}
	var dt identity.DoctorType
	if err := database.DB.First(&dt, req.DoctorTypeID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Doctor type not found")
	}
	result := database.DB.Model(&identity.DoctorProfile{}).
		Where("user_id = ?", uid).
		Updates(map[string]interface{}{
			"doctor_type_id":  req.DoctorTypeID,
			"specialization": dt.Name,
		})
	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DB_ERROR", result.Error.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"doctor_user_id": uid,
		"doctor_type":    dt,
	}, "Doctor type updated")
}

// AssignSubSpecialty assigns a sub-specialty to a doctor.
// POST /admin/doctors/:user_id/sub-specialties
// POST /chief-doctor/doctors/:user_id/sub-specialties
func AssignSubSpecialty(c *fiber.Ctx) error {
	uid, err := strconv.ParseUint(c.Params("user_id"), 10, 32)
	if err != nil || uid == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid user_id")
	}
	var req struct {
		SubSpecialtyID uint   `json:"sub_specialty_id"`
		CertificateURL string `json:"certificate_url"`
	}
	if err := c.BodyParser(&req); err != nil || req.SubSpecialtyID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "sub_specialty_id required")
	}

	// Verify sub-specialty exists
	var sub identity.DoctorSubSpecialty
	if err := database.DB.First(&sub, req.SubSpecialtyID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Sub-specialty not found")
	}

	// Chief doctor role check — can only assign sub-specialties of their own type
	callerRole, _ := c.Locals("role").(string)
	if callerRole == "chief_doctor" {
		callerID, _ := c.Locals("user_id").(uint)
		var callerProfile identity.DoctorProfile
		if err := database.DB.Where("user_id = ?", callerID).First(&callerProfile).Error; err != nil {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Chief doctor profile not found")
		}
		if callerProfile.DoctorTypeID == nil || *callerProfile.DoctorTypeID != sub.DoctorTypeID {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You can only assign sub-specialties within your own AYUSH type")
		}
	}

	assignerID, _ := c.Locals("user_id").(uint)
	assignment := identity.DoctorSubSpecialtyAssignment{
		DoctorUserID:   uint(uid),
		SubSpecialtyID: req.SubSpecialtyID,
		AssignedBy:     assignerID,
		AssignedByRole: callerRole,
		CertificateURL: req.CertificateURL,
		IsActive:       true,
	}

	// Upsert — reactivate if previously deactivated
	var existing identity.DoctorSubSpecialtyAssignment
	err2 := database.DB.Where("doctor_user_id = ? AND sub_specialty_id = ?", uid, req.SubSpecialtyID).First(&existing).Error
	if err2 == nil {
		database.DB.Model(&existing).Updates(map[string]interface{}{
			"is_active": true, "certificate_url": req.CertificateURL,
			"assigned_by": assignerID, "assigned_by_role": callerRole,
		})
	} else {
		if err := database.DB.Create(&assignment).Error; err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "DB_ERROR", err.Error())
		}
	}

	database.DB.Preload("SubSpecialty.DoctorType").Where("doctor_user_id = ? AND sub_specialty_id = ?", uid, req.SubSpecialtyID).First(&assignment)
	return response.JSON(c, fiber.StatusCreated, assignment, "Sub-specialty assigned")
}

// RemoveSubSpecialty removes a sub-specialty assignment.
// DELETE /admin/doctors/:user_id/sub-specialties/:id
// DELETE /chief-doctor/doctors/:user_id/sub-specialties/:id
func RemoveSubSpecialty(c *fiber.Ctx) error {
	uid, err := strconv.ParseUint(c.Params("user_id"), 10, 32)
	subID, err2 := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil || err2 != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid IDs")
	}

	// Chief doctor type check
	callerRole, _ := c.Locals("role").(string)
	if callerRole == "chief_doctor" {
		var sub identity.DoctorSubSpecialty
		if err := database.DB.First(&sub, subID).Error; err != nil {
			return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Sub-specialty not found")
		}
		callerID, _ := c.Locals("user_id").(uint)
		var callerProfile identity.DoctorProfile
		database.DB.Where("user_id = ?", callerID).First(&callerProfile)
		if callerProfile.DoctorTypeID == nil || *callerProfile.DoctorTypeID != sub.DoctorTypeID {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You can only manage sub-specialties within your own AYUSH type")
		}
	}

	database.DB.Model(&identity.DoctorSubSpecialtyAssignment{}).
		Where("doctor_user_id = ? AND sub_specialty_id = ?", uid, subID).
		Update("is_active", false)
	return response.JSON(c, fiber.StatusOK, nil, "Sub-specialty removed")
}

// ListAssignmentAudit returns audit log of all specialty assignments.
// GET /admin/doctor-sub-specialties/audit?page=1&limit=50
func ListAssignmentAudit(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)
	page := c.QueryInt("page", 1)
	offset := (page - 1) * limit

	var assignments []identity.DoctorSubSpecialtyAssignment
	var total int64
	database.DB.Model(&identity.DoctorSubSpecialtyAssignment{}).Count(&total)
	database.DB.Preload("SubSpecialty.DoctorType").
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&assignments)

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"assignments": assignments,
		"total":       total,
		"page":        page,
	}, "Audit log fetched")
}

// ── Public endpoint ───────────────────────────────────────────────────────────

// GetDoctorPublicSpecialties returns a doctor's specialties (public).
// GET /doctors/:id/specialties
func GetDoctorPublicSpecialties(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil || id == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid doctor ID")
	}
	var assignments []identity.DoctorSubSpecialtyAssignment
	database.DB.Preload("SubSpecialty").
		Where("doctor_user_id = ? AND is_active = ?", id, true).
		Find(&assignments)

	subs := make([]identity.DoctorSubSpecialty, 0, len(assignments))
	for _, a := range assignments {
		subs = append(subs, a.SubSpecialty)
	}
	return response.JSON(c, fiber.StatusOK, subs, "Doctor specialties fetched")
}

// ── Route Registration ────────────────────────────────────────────────────────

// RegisterSpecializationRoutes registers all specialty endpoints.
// Call from app.go: RegisterSpecializationRoutes(v1)
func RegisterSpecializationRoutes(router fiber.Router) {
	// Public routes
	//
	// NOTE: GET /doctor-types is intentionally NOT registered here.
	// The rich version (with doctor_count + online_count + nested
	// sub_specialties) lives in handlers/doctors.go::GetDoctorTypes
	// and is registered by handlers.RegisterDoctorRoutes. Registering
	// the bare ListDoctorTypes here would shadow that rich response
	// depending on registration order — caused production "0 online"
	// + empty count issues on /doctors page.
	router.Get("/doctor-types/:slug/sub-specialties", ListSubSpecialties)
	router.Get("/doctors/:id/specialties", GetDoctorPublicSpecialties)

	// Admin-only catalog management
	adminGroup := router.Group("/admin",
		middleware.Protected(),
		middleware.AdminOnly(),
	)
	adminGroup.Put("/doctor-types/:id", UpdateDoctorType)
	adminGroup.Post("/doctor-sub-specialties", CreateSubSpecialty)
	adminGroup.Put("/doctor-sub-specialties/:id", UpdateSubSpecialty)
	adminGroup.Delete("/doctor-sub-specialties/:id", DeactivateSubSpecialty)
	adminGroup.Put("/doctors/:user_id/type", SetDoctorType)

	// Catalog Read Access - allowed for admins, managers, and head doctors
	readGroup := router.Group("/admin",
		middleware.Protected(),
		middleware.RequireRoles("admin", "super_admin", "manager", "chief_doctor", "hospital_head_doctor"),
	)
	readGroup.Get("/doctor-types", ListDoctorTypes)
	readGroup.Get("/doctor-sub-specialties", ListSubSpecialties)
	readGroup.Get("/doctor-sub-specialties/audit", ListAssignmentAudit)
	readGroup.Get("/doctors/:user_id/specialties", GetDoctorSpecialties)
	readGroup.Post("/doctors/:user_id/sub-specialties", AssignSubSpecialty)
	readGroup.Delete("/doctors/:user_id/sub-specialties/:id", RemoveSubSpecialty)

	// Chief doctor routes (assignment only).
	//
	// Both head-doctor roles are allowed: chief_doctor (the online network head
	// doctor who runs clinical onboarding) and hospital_head_doctor (physical
	// hospital lead), plus manager. The AssignSubSpecialty handler additionally
	// restricts a chief_doctor to sub-specialties within their own AYUSH type.
	chiefGroup := router.Group("/chief-doctor",
		middleware.Protected(),
		middleware.RequireRoles("chief_doctor", "hospital_head_doctor", "manager", "admin"),
	)
	chiefGroup.Get("/doctors/:user_id/specialties", GetDoctorSpecialties)
	chiefGroup.Post("/doctors/:user_id/sub-specialties", AssignSubSpecialty)
	chiefGroup.Delete("/doctors/:user_id/sub-specialties/:id", RemoveSubSpecialty)
}

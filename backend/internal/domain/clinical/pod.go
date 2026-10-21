package clinical

// pod.go — specialty pods: the org structure for a scaling freelance doctor
// network. Each pod is a clinical specialty cluster led by a head doctor
// (chief_doctor). Doctors are assigned to a pod; head-doctor oversight views
// (dashboard, QA queue, onboarding) scope to the pod the chief doctor leads.

import (
	"strconv"
	"strings"
	"time"

	"drayya-core/internal/domain/identity"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Pod is a specialty cluster of doctors with a head-doctor lead.
type Pod struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"not null" json:"name"`        // "Cardiology", "AYUSH / General"
	Specialty    string    `gorm:"index" json:"specialty"`      // canonical key
	LeadDoctorID *uint     `gorm:"index" json:"lead_doctor_id"` // head doctor (chief_doctor user id)
	// DoctorTypeID links a pod to one AYUSH doctor type (Siddha/Ayurveda/…), so a
	// pod is the org home for an entire AYUSH system. Doctors are auto-routed to
	// the pod matching their type at onboard; the pod's lead is that system's
	// head doctor. Nullable for legacy/ad-hoc pods not tied to a type.
	DoctorTypeID *uint     `gorm:"index" json:"doctor_type_id"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Pod) TableName() string { return "pods" }

// SeedAYUSHPods creates one pod per AYUSH doctor type (idempotent) and routes any
// existing doctors with no pod into the pod matching their type. Run at startup
// after SeedDoctorTypes. This is what makes "a separate head doctor per AYUSH
// system" possible: assign a chief_doctor as each pod's lead and they own that
// system's doctors. At launch one head doctor can lead all four pods.
func SeedAYUSHPods(db *gorm.DB) {
	type dtRow struct {
		ID   uint
		Name string
		Slug string
	}
	var types []dtRow
	db.Table("doctor_types").Select("id, name, slug").Find(&types)
	for _, t := range types {
		tid := t.ID
		var existing int64
		db.Model(&Pod{}).Where("doctor_type_id = ?", tid).Count(&existing)
		if existing == 0 {
			db.Create(&Pod{
				Name:         t.Name + " — Online",
				Specialty:    t.Slug,
				DoctorTypeID: &tid,
				IsActive:     true,
			})
		}
	}
	// Backfill: route existing doctors with no pod into their type's pod. Done in
	// Go (not UPDATE…FROM) so it works on both Postgres and SQLite.
	var pods []Pod
	db.Where("is_active = ? AND doctor_type_id IS NOT NULL", true).Find(&pods)
	for _, p := range pods {
		if p.DoctorTypeID == nil {
			continue
		}
		db.Table("doctorprofile").
			Where("doctor_type_id = ? AND (pod_id IS NULL OR pod_id = 0)", *p.DoctorTypeID).
			Update("pod_id", p.ID)
	}
}

// scopedDoctorIDs resolves which doctors a request may see. Admin / manager /
// super_admin see everyone (scoped=false). A chief_doctor who leads a pod is
// scoped to that pod's doctors; a chief_doctor with no pod keeps network-wide
// visibility (backward compatible). The bool reports whether a filter applies.
func (h *Handler) scopedDoctorIDs(c *fiber.Ctx) (ids []uint, scoped bool) {
	role, _ := c.Locals("role").(string)
	if role == "admin" || role == "super_admin" || role == "manager" {
		return nil, false
	}
	userID, _ := c.Locals("user_id").(uint)
	// Both head-doctor roles (chief_doctor and hospital_head_doctor) are pod leads
	// and are scoped to the doctors in the pods they lead.
	if (role != "chief_doctor" && role != "hospital_head_doctor") || userID == 0 {
		return nil, false
	}
	// A head doctor may lead more than one pod (at launch, one head doctor leads
	// all four AYUSH pods). Scope to the union of every pod they lead, so the
	// model supports "one head doctor now, four later" with no code change —
	// splitting is just reassigning a pod's lead.
	var podIDs []uint
	h.db.Model(&Pod{}).Where("lead_doctor_id = ? AND is_active = ?", userID, true).Pluck("id", &podIDs)
	if len(podIDs) == 0 {
		return nil, false // not a pod lead → network-wide (legacy behaviour)
	}
	h.db.Table("doctorprofile").Where("pod_id IN ?", podIDs).Pluck("user_id", &ids)
	if len(ids) == 0 {
		// Leads with empty pods — scope to a sentinel so they see nothing rather
		// than everything.
		return []uint{0}, true
	}
	return ids, true
}

// ── Pod management (manager / admin) ─────────────────────────────────────────

func (h *Handler) registerPodRoutes(g fiber.Router) {
	pods := g.Group("/pods")
	pods.Get("/", h.ListPods)                  // chief_doctor + manager + admin
	pods.Get("/assignable", h.PodAssignables)  // people picker for lead/doctor assignment
	pods.Post("/", middleware.ManagerOnly(), h.CreatePod)
	pods.Put("/:id", middleware.ManagerOnly(), h.UpdatePod)
	pods.Post("/:id/lead", middleware.ManagerOnly(), h.AssignPodLead)
	pods.Post("/:id/doctors", middleware.ManagerOnly(), h.AssignPodDoctors)
	pods.Delete("/:id/doctors/:doctorId", middleware.ManagerOnly(), h.RemovePodDoctor)
}

type podRow struct {
	Pod
	LeadName    string `json:"lead_name"`
	DoctorCount int64  `json:"doctor_count"`
}

// PodAssignables is the people picker for pod assignment. kind=lead returns head
// doctors (chief_doctor); anything else returns doctors, with their current pod.
//
// GET /clinical/pods/assignable?kind=lead|doctor&q=
func (h *Handler) PodAssignables(c *fiber.Ctx) error {
	var roles []string
	if c.Query("kind") == "lead" {
		roles = []string{"doctor", "chief_doctor", "hospital_head_doctor"}
	} else {
		roles = []string{"doctor"}
	}
	type person struct {
		ID    uint   `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
		PodID *uint  `json:"pod_id,omitempty"`
	}
	q := h.db.Table("user AS u").
		Select("u.id AS id, u.full_name AS name, u.email AS email, dp.pod_id AS pod_id").
		Joins("LEFT JOIN doctorprofile dp ON dp.user_id = u.id").
		Where("u.role IN ?", roles)
	if term := strings.TrimSpace(c.Query("q")); term != "" {
		like := "%" + strings.ToLower(term) + "%"
		q = q.Where("LOWER(u.full_name) LIKE ? OR LOWER(u.email) LIKE ?", like, like)
	}
	var rows []person
	if err := q.Order("u.full_name ASC").Limit(20).Scan(&rows).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, rows, "Assignables fetched")
}

// ListPods returns every pod with its lead's name and current doctor count.
//
// GET /clinical/pods
func (h *Handler) ListPods(c *fiber.Ctx) error {
	var pods []Pod
	if err := h.db.Order("name ASC").Find(&pods).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	rows := make([]podRow, 0, len(pods))
	for _, p := range pods {
		r := podRow{Pod: p}
		if p.LeadDoctorID != nil {
			h.db.Table("user").Where("id = ?", *p.LeadDoctorID).Select("full_name").Scan(&r.LeadName)
		}
		h.db.Table("doctorprofile").Where("pod_id = ?", p.ID).Count(&r.DoctorCount)
		rows = append(rows, r)
	}
	return response.JSON(c, fiber.StatusOK, rows, "Pods fetched")
}

// CreatePod creates a specialty pod.
//
// POST /clinical/pods   body: { name, specialty, lead_doctor_id? }
func (h *Handler) CreatePod(c *fiber.Ctx) error {
	var req struct {
		Name         string `json:"name"`
		Specialty    string `json:"specialty"`
		LeadDoctorID *uint  `json:"lead_doctor_id"`
	}
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "name is required")
	}
	pod := Pod{Name: strings.TrimSpace(req.Name), Specialty: strings.TrimSpace(req.Specialty), LeadDoctorID: req.LeadDoctorID, IsActive: true}
	if err := h.db.WithContext(c.UserContext()).Create(&pod).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, pod, "Pod created")
}

// UpdatePod edits a pod's name / specialty / active flag.
//
// PUT /clinical/pods/:id
func (h *Handler) UpdatePod(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid pod ID")
	}
	var req struct {
		Name      *string `json:"name"`
		Specialty *string `json:"specialty"`
		IsActive  *bool   `json:"is_active"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	updates := map[string]interface{}{}
	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Specialty != nil {
		updates["specialty"] = strings.TrimSpace(*req.Specialty)
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if len(updates) == 0 {
		return response.Error(c, fiber.StatusBadRequest, "NO_FIELDS", "No updatable fields")
	}
	if err := h.db.WithContext(c.UserContext()).Model(&Pod{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"id": id}, "Pod updated")
}

// AssignPodLead sets (or clears) the head doctor for a pod. The lead must be a
// chief_doctor; assigning also pulls them into the pod's doctor set if they
// consult.
//
// POST /clinical/pods/:id/lead   body: { lead_doctor_id }
func (h *Handler) AssignPodLead(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid pod ID")
	}
	var req struct {
		LeadDoctorID *uint `json:"lead_doctor_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	if req.LeadDoctorID != nil {
		var u identity.User
		if err := h.db.First(&u, *req.LeadDoctorID).Error; err != nil {
			return response.Error(c, fiber.StatusNotFound, "DOCTOR_NOT_FOUND", "Selected doctor not found")
		}
		if u.Role != "chief_doctor" && u.Role != "hospital_head_doctor" {
			if u.Role == "doctor" {
				if err := h.db.Model(&u).Update("role", "chief_doctor").Error; err != nil {
					return response.Error(c, fiber.StatusInternalServerError, "ROLE_UPDATE_FAILED", "Failed to assign head doctor role: "+err.Error())
				}
			} else {
				return response.Error(c, fiber.StatusUnprocessableEntity, "NOT_HEAD_DOCTOR", "Pod lead must be a doctor or a head doctor")
			}
		}
	}
	if err := h.db.WithContext(c.UserContext()).Model(&Pod{}).Where("id = ?", id).
		Update("lead_doctor_id", req.LeadDoctorID).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"id": id, "lead_doctor_id": req.LeadDoctorID}, "Pod lead assigned")
}

// AssignPodDoctors moves doctors into the pod (sets doctorprofile.pod_id).
//
// POST /clinical/pods/:id/doctors   body: { doctor_ids: [..] }   (user ids)
func (h *Handler) AssignPodDoctors(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid pod ID")
	}
	var req struct {
		DoctorIDs []uint `json:"doctor_ids"`
	}
	if err := c.BodyParser(&req); err != nil || len(req.DoctorIDs) == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "doctor_ids[] is required")
	}
	if err := h.db.WithContext(c.UserContext()).Table("doctorprofile").
		Where("user_id IN ?", req.DoctorIDs).Update("pod_id", id).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"pod_id": id, "assigned": len(req.DoctorIDs)}, "Doctors assigned")
}

// RemovePodDoctor unassigns a doctor from the pod (pod_id → NULL).
//
// DELETE /clinical/pods/:id/doctors/:doctorId
func (h *Handler) RemovePodDoctor(c *fiber.Ctx) error {
	doctorID, err := strconv.Atoi(c.Params("doctorId"))
	if err != nil || doctorID <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid doctor ID")
	}
	if err := h.db.WithContext(c.UserContext()).Table("doctorprofile").
		Where("user_id = ?", doctorID).Update("pod_id", gorm.Expr("NULL")).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"doctor_id": doctorID}, "Doctor removed from pod")
}

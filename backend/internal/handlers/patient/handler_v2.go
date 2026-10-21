package patient

import (
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// ─── Profile ─────────────────────────────────────────────────────────────────

func (h *Handler) GetProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var profile identity.PatientProfile

	err := h.db.Where("user_id = ?", userID).First(&profile).Error
	if err == gorm.ErrRecordNotFound {
		return response.Error(c, fiber.StatusNotFound, "PROFILE_NOT_FOUND", "Profile not found")
	}

	return response.JSON(c, fiber.StatusOK, profile, "Patient profile fetched")
}

func (h *Handler) UpdateProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var req identity.PatientProfile
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	req.UserID = userID

	var profile identity.PatientProfile
	err := h.db.Where("user_id = ?", userID).First(&profile).Error

	if err == gorm.ErrRecordNotFound {
		if err := h.db.Create(&req).Error; err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create profile")
		}
		return response.JSON(c, fiber.StatusCreated, req, "Profile created")
	}

	req.ID = profile.ID
	h.db.Save(&req)

	return response.JSON(c, fiber.StatusOK, req, "Profile updated")
}

// ─── Health Goals ─────────────────────────────────────────────────────────────

func (h *Handler) GetHealthGoals(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var goals []HealthGoal
	h.db.Where("patient_id = ?", userID).Find(&goals)
	return response.JSON(c, fiber.StatusOK, goals, "Health goals fetched")
}

func (h *Handler) CreateHealthGoal(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var goal HealthGoal
	if err := c.BodyParser(&goal); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	goal.PatientID = userID
	if err := h.db.Create(&goal).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create health goal")
	}
	return response.JSON(c, fiber.StatusCreated, goal, "Health goal created")
}

func (h *Handler) UpdateHealthGoal(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	id := c.Params("id")
	var goal HealthGoal
	if h.db.Where("id = ? AND patient_id = ?", id, userID).First(&goal).Error != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Health goal not found")
	}
	if err := c.BodyParser(&goal); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
	}
	h.db.Save(&goal)
	return response.JSON(c, fiber.StatusOK, goal, "Health goal updated")
}

func (h *Handler) DeleteHealthGoal(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	id := c.Params("id")
	h.db.Where("id = ? AND patient_id = ?", id, userID).Delete(&HealthGoal{})
	return response.JSON(c, fiber.StatusOK, nil, "Health goal deleted")
}

// ─── Medication Reminders ─────────────────────────────────────────────────────

func (h *Handler) GetMedicationReminders(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var reminders []MedicationReminder
	h.db.Where("patient_id = ?", userID).Find(&reminders)
	return response.JSON(c, fiber.StatusOK, reminders, "Medication reminders fetched")
}

func (h *Handler) CreateMedicationReminder(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var reminder MedicationReminder
	if err := c.BodyParser(&reminder); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	reminder.PatientID = userID
	if err := h.db.Create(&reminder).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create medication reminder")
	}
	return response.JSON(c, fiber.StatusCreated, reminder, "Medication reminder created")
}

func (h *Handler) UpdateMedicationReminder(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	id := c.Params("id")
	var reminder MedicationReminder
	if h.db.Where("id = ? AND patient_id = ?", id, userID).First(&reminder).Error != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Medication reminder not found")
	}
	if err := c.BodyParser(&reminder); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
	}
	h.db.Save(&reminder)
	return response.JSON(c, fiber.StatusOK, reminder, "Medication reminder updated")
}

func (h *Handler) DeleteMedicationReminder(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	id := c.Params("id")
	h.db.Where("id = ? AND patient_id = ?", id, userID).Delete(&MedicationReminder{})
	return response.JSON(c, fiber.StatusOK, nil, "Medication reminder deleted")
}

// ─── Second Opinions ─────────────────────────────────────────────────────────

func (h *Handler) GetSecondOpinions(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var opinions []SecondOpinion
	h.db.Where("patient_id = ?", userID).Find(&opinions)
	return response.JSON(c, fiber.StatusOK, opinions, "Second opinions fetched")
}

func (h *Handler) SubmitSecondOpinion(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var opinion SecondOpinion
	if err := c.BodyParser(&opinion); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	opinion.PatientID = userID
	opinion.Status = "pending"
	if err := h.db.Create(&opinion).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to submit second opinion")
	}
	return response.JSON(c, fiber.StatusCreated, opinion, "Second opinion submitted")
}

// ─── Vaccinations ─────────────────────────────────────────────────────────────

func (h *Handler) GetVaccinations(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var records []VaccinationRecord
	h.db.Where("patient_id = ?", userID).Find(&records)
	return response.JSON(c, fiber.StatusOK, records, "Vaccination records fetched")
}

func (h *Handler) AddVaccination(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var record VaccinationRecord
	if err := c.BodyParser(&record); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	record.PatientID = userID
	if err := h.db.Create(&record).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to add vaccination record")
	}
	return response.JSON(c, fiber.StatusCreated, record, "Vaccination record added")
}

// ─── Health Timeline ─────────────────────────────────────────────────────────

func (h *Handler) GetHealthTimeline(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var events []HealthTimelineEvent
	h.db.Where("patient_id = ?", userID).Order("date DESC").Find(&events)
	return response.JSON(c, fiber.StatusOK, events, "Health timeline fetched")
}

func (h *Handler) AddTimelineEvent(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var event HealthTimelineEvent
	if err := c.BodyParser(&event); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	event.PatientID = userID
	if err := h.db.Create(&event).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to add timeline event")
	}
	return response.JSON(c, fiber.StatusCreated, event, "Timeline event added")
}

// ─── Route Registration ───────────────────────────────────────────────────────

func (h *Handler) RegisterRoutes(router fiber.Router) {
	group := router.Group("/patients", middleware.Protected())

	// Dashboard Summary
	group.Get("/dashboard/summary", GetDashboardSummary)

	// Profile
	group.Get("/profile", h.GetProfile)
	group.Post("/profile", h.UpdateProfile)
	group.Put("/profile", h.UpdateProfile)

	// Health Goals
	group.Get("/health-goals", h.GetHealthGoals)
	group.Post("/health-goals", h.CreateHealthGoal)
	group.Put("/health-goals/:id", h.UpdateHealthGoal)
	group.Delete("/health-goals/:id", h.DeleteHealthGoal)

	// Medication Reminders
	group.Get("/medication-reminders", h.GetMedicationReminders)
	group.Post("/medication-reminders", h.CreateMedicationReminder)
	group.Put("/medication-reminders/:id", h.UpdateMedicationReminder)
	group.Delete("/medication-reminders/:id", h.DeleteMedicationReminder)

	// Second Opinions
	group.Get("/second-opinions", h.GetSecondOpinions)
	group.Post("/second-opinions", h.SubmitSecondOpinion)

	// Vaccinations
	group.Get("/vaccinations", h.GetVaccinations)
	group.Post("/vaccinations", h.AddVaccination)

	// Health Timeline
	group.Get("/health-timeline", h.GetHealthTimeline)
	group.Post("/health-timeline", h.AddTimelineEvent)

	// Patient-initiated Lab Bookings (Android-friendly format)
	group.Get("/lab-bookings", h.GetLabBookings)
	group.Post("/lab-bookings", h.CreateLabBooking)
}

// ─── Patient Lab Bookings ─────────────────────────────────────────────────────

func (h *Handler) GetLabBookings(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var bookings []PatientLabBooking
	h.db.Where("patient_id = ?", userID).Order("created_at desc").Find(&bookings)
	return response.JSON(c, fiber.StatusOK, bookings, "Lab bookings fetched")
}

func (h *Handler) CreateLabBooking(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var booking PatientLabBooking
	if err := c.BodyParser(&booking); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	booking.PatientID = userID
	booking.Status = "booked"
	if err := h.db.Create(&booking).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create lab booking")
	}
	return response.JSON(c, fiber.StatusCreated, booking, "Lab test booked successfully")
}

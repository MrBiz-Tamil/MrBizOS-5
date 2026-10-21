package handlers

import (
	"drayya-core/internal/domain/rating"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type RatingHandler struct {
	service rating.Service
}

func NewRatingHandler(service rating.Service) *RatingHandler {
	return &RatingHandler{service: service}
}

// SubmitRating — POST /ratings
// Body: { doctor_id, appointment_id, score (1-5), comment? }
func (h *RatingHandler) SubmitRating(c *fiber.Ctx) error {
	// Fix: middleware sets "user_id" (not "userID")
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	var r rating.Rating
	if err := c.BodyParser(&r); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	if r.Score < 1 || r.Score > 5 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_SCORE", "Score must be between 1 and 5")
	}
	if r.DoctorID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_DOCTOR", "doctor_id required")
	}
	if r.AppointmentID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_APPOINTMENT", "appointment_id required")
	}

	// Prevent double-rating
	already, _ := h.service.HasRatedAppointment(userID, r.AppointmentID)
	if already {
		return response.Error(c, fiber.StatusConflict, "ALREADY_RATED", "You have already rated this appointment")
	}

	r.UserID = userID
	if err := h.service.Submit(&r); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SUBMIT_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, r, "Rating submitted successfully")
}

// GetDoctorRatings — GET /ratings/doctor/:id
func (h *RatingHandler) GetDoctorRatings(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid doctor ID")
	}

	ratings, err := h.service.DoctorRatings(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, ratings, "Doctor ratings fetched")
}

// GetMyRatings — GET /ratings/mine
func (h *RatingHandler) GetMyRatings(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	ratings, err := h.service.MyRatings(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, ratings, "Your ratings fetched")
}

// CheckRated — GET /ratings/check?appointment_id=X
// Returns { rated: true/false } — used by frontend to decide whether to show modal.
func (h *RatingHandler) CheckRated(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	apptIDStr := c.Query("appointment_id")
	apptID, err := strconv.ParseUint(apptIDStr, 10, 32)
	if err != nil || apptID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_PARAMS", "appointment_id query param required")
	}

	rated, _ := h.service.HasRatedAppointment(userID, uint(apptID))
	return response.JSON(c, fiber.StatusOK, fiber.Map{"rated": rated}, "")
}

func (h *RatingHandler) RegisterRoutes(router fiber.Router) {
	r := router.Group("/ratings")
	// Public
	r.Get("/doctor/:id", h.GetDoctorRatings)
	// Protected
	r.Post("/", middleware.Protected(), h.SubmitRating)
	r.Get("/mine", middleware.Protected(), h.GetMyRatings)
	r.Get("/check", middleware.Protected(), h.CheckRated)
}

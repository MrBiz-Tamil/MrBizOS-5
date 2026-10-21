package abdm

import (
	"drayya-core/internal/middleware"
	"drayya-core/internal/services/abdm"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

var client = abdm.NewClient()

// InitiateABHACreation starts the ABHA creation flow via Aadhaar OTP
// POST /api/v1/abha/create/aadhaar/otp
func InitiateABHACreation(c *fiber.Ctx) error {
	var req struct {
		Aadhaar string `json:"aadhaar"`
	}
	if err := c.BodyParser(&req); err != nil || req.Aadhaar == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Aadhaar number required")
	}

	resp, err := client.SendAadhaarOTP(req.Aadhaar)
	if err != nil {
		return response.Error(c, fiber.StatusBadGateway, "ABDM_ERROR", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"txn_id":  resp.TxnID,
		"message": resp.Message,
	}, "OTP sent to Aadhaar-linked mobile")
}

// VerifyABHAOTP verifies the OTP and returns the ABHA profile
// POST /api/v1/abha/create/aadhaar/verify
func VerifyABHAOTP(c *fiber.Ctx) error {
	var req struct {
		TxnID string `json:"txn_id"`
		OTP   string `json:"otp"`
	}
	if err := c.BodyParser(&req); err != nil || req.TxnID == "" || req.OTP == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "txn_id and otp required")
	}

	profile, err := client.VerifyAadhaarOTP(req.TxnID, req.OTP)
	if err != nil {
		return response.Error(c, fiber.StatusBadGateway, "ABDM_OTP_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, profile, "ABHA profile created/retrieved")
}

// SendMobileOTP sends OTP to a mobile number for ABHA login
// POST /api/v1/abha/login/mobile/otp
func SendMobileOTP(c *fiber.Ctx) error {
	var req struct {
		Mobile string `json:"mobile"`
	}
	if err := c.BodyParser(&req); err != nil || req.Mobile == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Mobile number required")
	}

	resp, err := client.SendMobileOTP(req.Mobile)
	if err != nil {
		return response.Error(c, fiber.StatusBadGateway, "ABDM_ERROR", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"txn_id":  resp.TxnID,
		"message": resp.Message,
	}, "OTP sent")
}

// GetABHAProfile returns the authenticated user's ABHA profile
// GET /api/v1/abha/profile
func GetABHAProfile(c *fiber.Ctx) error {
	healthID := c.Query("health_id")
	profile, err := client.GetABHAProfile(healthID)
	if err != nil {
		return response.Error(c, fiber.StatusBadGateway, "ABDM_ERROR", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, profile, "ABHA profile fetched")
}

// GetFHIRBundle returns a FHIR-formatted health record bundle for the patient
// GET /api/v1/abha/fhir/patient/:id
func GetFHIRBundle(c *fiber.Ctx) error {
	// Build a demo FHIR bundle for the patient
	// In production, fetch real data from the database
	patientID := c.Params("id")

	patient := abdm.MapPatientToFHIR(0, "Patient "+patientID, "unknown", "", "", "")
	bundle := abdm.BuildPatientBundle(patient, nil, nil)

	return c.Status(200).JSON(bundle)
}

// RegisterABHARoutes registers all ABHA/ABDM routes
func RegisterABHARoutes(router fiber.Router) {
	abha := router.Group("/abha")

	// Public — OTP flows (require rate limiting in production)
	abha.Post("/create/aadhaar/otp", InitiateABHACreation)
	abha.Post("/create/aadhaar/verify", VerifyABHAOTP)
	abha.Post("/login/mobile/otp", SendMobileOTP)

	// Protected — profile access
	abha.Get("/profile", middleware.Protected(), GetABHAProfile)
	abha.Get("/fhir/patient/:id", middleware.Protected(), GetFHIRBundle)
}

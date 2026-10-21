package validator

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"drayya-core/internal/utils/response"
)

var validate = validator.New()

// ErrorResponse represents a single validation error
type ErrorResponse struct {
	Field string `json:"field"`
	Tag   string `json:"tag"`
	Value string `json:"value,omitempty"`
}

// ValidateStruct validates a struct and returns a standardized error response if validation fails.
// IMPORTANT: this function writes the HTTP response AND returns a non-nil sentinel error so
// callers can do `if err := ValidateStruct(c, &req); err != nil { return err }` and the
// handler will stop executing. Previously response.Error() returned nil on a successful write,
// so the handler silently continued past validation failures.
var errValidationFailed = errors.New("validation failed")

func ValidateStruct(c *fiber.Ctx, s interface{}) error {
	err := validate.Struct(s)
	if err != nil {
		var errs []ErrorResponse
		for _, ve := range err.(validator.ValidationErrors) {
			errs = append(errs, ErrorResponse{
				Field: strings.ToLower(ve.Field()),
				Tag:   ve.Tag(),
				Value: ve.Param(),
			})
		}
		// Write the 400 response to the client, then return a non-nil sentinel so
		// the caller's `if err != nil { return err }` guard actually fires.
		_ = response.Error(c, fiber.StatusBadRequest, "VALIDATION_FAILED", "Input validation failed", errs)
		return errValidationFailed
	}
	return nil
}

// Setup custom validation rules if needed
func init() {
	// Add custom validations here if required, e.g., phone numbers, etc.
}

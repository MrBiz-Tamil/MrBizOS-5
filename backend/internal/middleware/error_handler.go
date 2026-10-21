package middleware

import (
	"fmt"
	"drayya-core/internal/utils/response"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
)

// ErrorHandler is a global middleware to catch errors and panics
func ErrorHandler(c *fiber.Ctx, err error) error {
	if err != nil && err.Error() == "validation failed" {
		// ValidateStruct already wrote the 400 response with detailed fields
		return nil
	}

	// Status code defaults to 500
	code := fiber.StatusInternalServerError

	// Check if it's a fiber.Error
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	// Log the error (and stack trace if it's a panic/internal error)
	if code >= 500 {
		fmt.Printf("API Error: %v\nStack: %s\n", err, debug.Stack())
	}

	// Map error types to standard codes
	errCode := "INTERNAL_SERVER_ERROR"
	switch code {
	case fiber.StatusNotFound:
		errCode = "RESOURCE_NOT_FOUND"
	case fiber.StatusUnauthorized:
		errCode = "UNAUTHORIZED"
	case fiber.StatusForbidden:
		errCode = "FORBIDDEN"
	case fiber.StatusBadRequest:
		errCode = "BAD_REQUEST"
	}

	return response.Error(c, code, errCode, err.Error())
}

// Recovery middleware specifically to ensure we return JSON on panics
func Recovery() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}
				_ = ErrorHandler(c, err)
			}
		}()
		return c.Next()
	}
}

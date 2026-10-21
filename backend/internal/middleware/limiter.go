package middleware

import (
	"os"
	"time"

	"drayya-core/internal/ratelimit"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// AuthLimiter - Strict rate limiting for authentication endpoints (5 attempts per 5 minutes)
func AuthLimiter() fiber.Handler {
	if os.Getenv("SKIP_LIMITER") == "true" {
		return func(c *fiber.Ctx) error { return c.Next() }
	}
	// Redis-backed storage so the auth attempt counter is GLOBAL across api
	// replicas — without it, an attacker gets N×10 attempts at N replicas. The
	// adapter returns nil when Redis isn't up so the limiter still works.
	return limiter.New(limiter.Config{
		Next: func(c *fiber.Ctx) bool {
			return c.Method() == "OPTIONS"
		},
		Max:        10,
		Expiration: 5 * time.Minute,
		Storage:    ratelimit.New(),
		KeyGenerator: func(c *fiber.Ctx) string {
			return "auth:" + c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many login attempts. Please try again after 5 minutes.",
			})
		},
	})
}

// APILimiter - Standard rate limiting for general API routes (100 requests per minute)
func APILimiter() fiber.Handler {
	if os.Getenv("SKIP_LIMITER") == "true" {
		return func(c *fiber.Ctx) error { return c.Next() }
	}
	return limiter.New(limiter.Config{
		Next: func(c *fiber.Ctx) bool {
			return c.Method() == "OPTIONS"
		},
		Max:        150,
		Expiration: 1 * time.Minute,
		Storage:    ratelimit.New(),
		KeyGenerator: func(c *fiber.Ctx) string {
			return "api:" + c.IP()
		},
	})
}

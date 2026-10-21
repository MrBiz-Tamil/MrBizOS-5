package telemetry

import (
	"log/slog"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

// InitLogger initializes the global structured logger using slog
func InitLogger() {
	opts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false, // Set to true for debugging to see file and line numbers
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("Logger Initialized", "version", "v1.0.2-fixed-logging", "status", "ready")
}

// SlogMiddleware creates a Fiber middleware that logs HTTP requests as structured JSON using slog
func SlogMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		duration := time.Since(start)

		statusCode := c.Response().StatusCode()
		method := c.Method()
		path := c.Path()
		ip := c.IP()

		if err != nil {
			// Fiber error handler usually sets the status code, but let's be safe
			if customErr, ok := err.(*fiber.Error); ok {
				statusCode = customErr.Code
			} else {
				statusCode = fiber.StatusInternalServerError
			}
			slog.Error("HTTP Request Failed",
				slog.Int("status", statusCode),
				slog.String("method", method),
				slog.String("path", path),
				slog.String("ip", ip),
				slog.Duration("duration", duration),
				slog.String("error", err.Error()),
			)
			return err
		}

		// Skip health check hits — fire every 15s from Docker and flood logs
		if path == "/health" {
			return nil
		}

		slog.Info("HTTP Request",
			slog.Int("status", statusCode),
			slog.String("method", method),
			slog.String("path", path),
			slog.String("ip", ip),
			slog.Duration("duration", duration),
		)

		return nil
	}
}

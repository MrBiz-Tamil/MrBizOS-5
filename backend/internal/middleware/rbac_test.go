package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestRequireRoles(t *testing.T) {
	app := fiber.New()

	// Route that requires pharmacy_admin or admin
	app.Get("/pharmacy-admin-only", func(c *fiber.Ctx) error {
		c.Locals("role", c.Get("Test-Role"))
		return c.Next()
	}, RequireRoles("pharmacy_admin", "admin"), func(c *fiber.Ctx) error {
		return c.SendString("success")
	})

	tests := []struct {
		name       string
		role       string
		expectCode int
	}{
		{"Admin Access", "admin", 200},
		{"Pharmacy Admin Access", "pharmacy_admin", 200},
		{"Patient Access Forbidden", "patient", 403},
		{"Supplier Access Forbidden", "supplier_agent", 403},
		{"No Role Forbidden", "", 403},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/pharmacy-admin-only", nil)
			req.Header.Set("Test-Role", tt.role)
			resp, _ := app.Test(req)
			assert.Equal(t, tt.expectCode, resp.StatusCode)
		})
	}
}

func TestConvenienceMiddlewares(t *testing.T) {
	app := fiber.New()

	app.Get("/supplier-only", func(c *fiber.Ctx) error {
		c.Locals("role", c.Get("Test-Role"))
		return c.Next()
	}, SupplierOnly(), func(c *fiber.Ctx) error {
		return c.SendString("success")
	})

	t.Run("Supplier Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/supplier-only", nil)
		req.Header.Set("Test-Role", "supplier_agent")
		resp, _ := app.Test(req)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("Supplier Forbidden", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/supplier-only", nil)
		req.Header.Set("Test-Role", "accountant")
		resp, _ := app.Test(req)
		assert.Equal(t, 403, resp.StatusCode)
	})
}

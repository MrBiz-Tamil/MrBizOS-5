package support

import (
	"testing"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

func TestGetDepartmentForRole(t *testing.T) {
	tests := []struct {
		role     string
		expected string
	}{
		{"finance_manager", "billing"},
		{"chief_doctor", "medical"},
		{"doctor", "medical"},
		{"pharmacy_manager", "pharmacy"},
		{"pharmacist", "pharmacy"},
		{"hr_manager", "hr"},
		{"inventory_manager", "logistics"},
		{"admin", "all"},
		{"super_admin", "all"},
		{"random_user", "none"},
		{"patient", "none"},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			result := getDepartmentForRole(tt.role)
			if result != tt.expected {
				t.Errorf("getDepartmentForRole(%q) = %q, want %q", tt.role, result, tt.expected)
			}
		})
	}
}

func TestRoleFromCtx(t *testing.T) {
	app := fiber.New()
	
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("role", "finance_manager")
		role := roleFromCtx(c)
		if role != "finance_manager" {
			t.Errorf("Expected role 'finance_manager', got %q", role)
		}
		
		c.Locals("user_id", uint(42))
		uid := userIDFromCtx(c)
		if uid != 42 {
			t.Errorf("Expected user_id 42, got %d", uid)
		}
		return c.SendStatus(200)
	})

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/test")
	ctx.Request.Header.SetMethod("GET")
	
	app.Handler()(ctx)
}

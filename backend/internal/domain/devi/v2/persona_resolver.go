package v2

import (
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"github.com/gofiber/fiber/v2"
)

// ResolvePersona inspects the Fiber request and returns the resolved
// caller identity. It reads:
//
//   - c.Locals("user_id") set by the JWT middleware (uint, 0 for guests)
//   - c.Locals("role")    set by JWT middleware (string)
//   - body.GuestBrowserID for anonymous users
//
// For authenticated patients it does ONE small DB hit to fetch tier and
// signup date — the cost of that lookup is recovered many times over by
// the personalised system prompt that flows from it.
func ResolvePersona(c *fiber.Ctx, guestBrowserID string) Persona {
	userID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	if userID == 0 {
		// Pure guest.
		bid := guestBrowserID
		if bid == "" {
			bid = "anonymous"
		}
		return Persona{
			Role:           RoleGuest,
			GuestBrowserID: bid,
		}
	}

	// Authenticated. Map role.
	persona := Persona{
		Role:   PersonaRole(role),
		UserID: &userID,
	}

	// Hydrate tier + IsNew flag for patients.
	if persona.Role == RolePatient {
		var u identity.User
		if database.DB != nil {
			database.DB.Select("tier, created_at").First(&u, userID)
			persona.Tier = u.Tier
			persona.IsNew = time.Since(u.CreatedAt) < 24*time.Hour
		}
	}

	// Unknown / unmapped roles get treated as staff (deferred).
	if persona.Role != RolePatient && persona.Role != RoleDoctor &&
		persona.Role != RoleGuest && persona.Role != RoleAdmin {
		persona.Role = RoleStaff
	}
	return persona
}

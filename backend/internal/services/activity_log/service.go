// Package activity_log writes auth/security events to the activity_logs table.
//
// While the GORM audit plugin captures every DATA mutation automatically,
// AUTH events (login, logout, MFA, password reset) are not data mutations
// — they're security signals. This package captures them explicitly via
// short async writes so the 🔐 Auth Log tab has real content.
//
// IMPORTANT — to avoid an import cycle with the identity package (which
// also imports this package now), we declare a LOCAL struct here that
// maps to the same `activity_logs` table that identity.ActivityLog
// already uses. GORM is happy with multiple Go types pointing at one
// table; both can coexist forever.
package activity_log

import (
	"encoding/json"
	"time"

	"drayya-core/internal/database"

	"github.com/gofiber/fiber/v2"
)

// authEvent is the local mirror of identity.ActivityLog used only for
// INSERTs by this package. We intentionally skip the FK preload — we just
// need to write a row.
type authEvent struct {
	ID        uint           `gorm:"primaryKey"`
	UserID    uint           `gorm:"index"`
	Action    string         `gorm:"index"`
	Entity    string
	EntityID  string
	Details   []byte         `gorm:"type:text"`
	IPAddress string
	UserAgent string
	CreatedAt time.Time
}

func (authEvent) TableName() string { return "activity_logs" }

// AuditSkip prevents the GORM audit plugin from logging our own writes —
// that would be circular noise. Auth events are surfaced via this table,
// not via audit_logs.
func (authEvent) AuditSkip() bool { return true }

// Categories used in the Action column. Keep this list short and stable —
// the frontend's Auth Log tab filters/colors by these values.
const (
	ActionLogin           = "login"
	ActionLoginFailed     = "login_failed"
	ActionLogout          = "logout"
	ActionMFAVerified     = "mfa_verified"
	ActionMFAFailed       = "mfa_failed"
	ActionPasswordReset   = "password_reset_requested"
	ActionPasswordChanged = "password_changed"
	ActionRegister        = "register"
	ActionTokenRefresh    = "token_refresh"
)

// LogAsync writes an activity log entry in a background goroutine.
// Safe to call from request handlers — never blocks.
//
// userID == 0 is allowed for events with no authenticated user yet
// (failed login, password-reset request before lookup).
func LogAsync(userID uint, action, entity, entityID string, details interface{}, c *fiber.Ctx) {
	ip := ""
	userAgent := ""
	if c != nil {
		ip = c.IP()
		userAgent = c.Get("User-Agent")
	}
	go func() {
		entry := authEvent{
			UserID:    userID,
			Action:    action,
			Entity:    entity,
			EntityID:  entityID,
			IPAddress: ip,
			UserAgent: userAgent,
			CreatedAt: time.Now(),
		}
		if details != nil {
			if data, err := json.Marshal(details); err == nil {
				entry.Details = data
			}
		}
		_ = database.DB.Create(&entry).Error
	}()
}


// LogAuthSuccess is a convenience for successful login/MFA/register events.
func LogAuthSuccess(c *fiber.Ctx, userID uint, action string, extras map[string]interface{}) {
	LogAsync(userID, action, "auth", "", extras, c)
}

// LogAuthFailure captures failed-login attempts — userID is 0 because
// the credentials never validated. We keep the email as entity_id so
// admins can search by it.
func LogAuthFailure(c *fiber.Ctx, action, email, reason string) {
	LogAsync(0, action, "auth", email, map[string]interface{}{
		"email":  email,
		"reason": reason,
	}, c)
}

// LogMiddleware (legacy) — kept for callers that may already import it.
// Use the explicit LogAuthSuccess/LogAuthFailure helpers in new code.
func LogMiddleware(action, entity string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		if err == nil && c.Response().StatusCode() < 400 {
			if userID, ok := c.Locals("user_id").(uint); ok {
				LogAsync(userID, action, entity, "", nil, c)
			}
		}
		return err
	}
}

package clinical

// compliance.go — Phase 3: compliance monitor.
//
// A live HR-facing queue of doctors with credential issues (license expiry,
// missing/unverified PAN needed for TDS on freelance earnings) plus a daily
// background scanner that pushes a digest to HR + admins.

import (
	"context"
	"fmt"
	"log"
	"time"

	"drayya-core/internal/domain/communication"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

const licenseExpiryWindow = 30 * 24 * time.Hour

type complianceIssue struct {
	Type     string `json:"type"`     // license_expired | license_expiring | pan_missing | pan_unverified
	Severity string `json:"severity"` // high | medium
	Detail   string `json:"detail"`
}

type complianceRow struct {
	DoctorID uint              `json:"doctor_id"`
	Name     string            `json:"name"`
	Email    string            `json:"email"`
	Issues   []complianceIssue `json:"issues"`
}

// registerComplianceRoutes mounts the compliance queue under its own gate — HR
// and management own this, plus a pod's head doctor for their own doctors.
func (h *Handler) registerComplianceRoutes(router fiber.Router) {
	comp := router.Group("/clinical/compliance",
		middleware.RequireRoles("hr_manager", "hr", "manager", "chief_doctor", "admin"))
	comp.Get("/", h.ComplianceQueue)
}

type docComplianceRow struct {
	DoctorID      uint
	Name          string
	Email         string
	LicenseExpiry *time.Time
	PAN           string
	PANVerified   bool
}

// computeIssues turns one doctor's credential state into a list of issues.
func computeIssues(d docComplianceRow, now, soon time.Time) []complianceIssue {
	var issues []complianceIssue
	if d.LicenseExpiry != nil {
		if d.LicenseExpiry.Before(now) {
			issues = append(issues, complianceIssue{"license_expired", "high",
				fmt.Sprintf("License lapsed %s", d.LicenseExpiry.Format("02 Jan 2006"))})
		} else if d.LicenseExpiry.Before(soon) {
			issues = append(issues, complianceIssue{"license_expiring", "medium",
				fmt.Sprintf("License expires %s", d.LicenseExpiry.Format("02 Jan 2006"))})
		}
	}
	if d.PAN == "" {
		issues = append(issues, complianceIssue{"pan_missing", "high", "PAN missing — TDS cannot be filed"})
	} else if !d.PANVerified {
		issues = append(issues, complianceIssue{"pan_unverified", "medium", "PAN not yet verified"})
	}
	return issues
}

// ComplianceQueue returns flagged doctors for the caller's scope.
//
// GET /clinical/compliance
func (h *Handler) ComplianceQueue(c *fiber.Ctx) error {
	q := h.db.Table("doctorprofile AS dp").
		Select(`dp.user_id AS doctor_id, du.full_name AS name, du.email AS email,
			dp.license_expiry, dp.pan, dp.pan_verified`).
		Joins(`JOIN "user" du ON du.id = dp.user_id`).
		Where("dp.is_verified = ?", true)
	if ids, scoped := h.scopedDoctorIDs(c); scoped {
		q = q.Where("dp.user_id IN ?", ids)
	}
	var docs []docComplianceRow
	if err := q.Scan(&docs).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	now := time.Now()
	soon := now.Add(licenseExpiryWindow)
	rows := []complianceRow{}
	counts := map[string]int{"license_expired": 0, "license_expiring": 0, "pan_missing": 0, "pan_unverified": 0}
	for _, d := range docs {
		issues := computeIssues(d, now, soon)
		for _, is := range issues {
			counts[is.Type]++
		}
		if len(issues) > 0 {
			rows = append(rows, complianceRow{d.DoctorID, d.Name, d.Email, issues})
		}
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"flagged":       rows,
		"counts":        counts,
		"total_flagged": len(rows),
	}, "Compliance queue fetched")
}

// StartComplianceScanner runs a daily scan, pushing a digest notification to HR
// and admins whenever there are open credential issues. Started from app.go.
func StartComplianceScanner(db *gorm.DB) {
	if db == nil {
		return
	}
	go func() {
		// Small delay so it doesn't run during the migration window at boot.
		time.Sleep(2 * time.Minute)
		for {
			scanCompliance(context.Background(), db)
			time.Sleep(24 * time.Hour)
		}
	}()
}

func scanCompliance(ctx context.Context, db *gorm.DB) {
	var docs []docComplianceRow
	db.WithContext(ctx).Table("doctorprofile AS dp").
		Select(`dp.user_id AS doctor_id, du.full_name AS name, du.email AS email, dp.license_expiry, dp.pan, dp.pan_verified`).
		Joins(`JOIN "user" du ON du.id = dp.user_id`).
		Where("dp.is_verified = ?", true).Scan(&docs)

	now := time.Now()
	soon := now.Add(licenseExpiryWindow)
	expired, expiring, panIssues := 0, 0, 0
	for _, d := range docs {
		for _, is := range computeIssues(d, now, soon) {
			switch is.Type {
			case "license_expired":
				expired++
			case "license_expiring":
				expiring++
			case "pan_missing", "pan_unverified":
				panIssues++
			}
		}
	}
	total := expired + expiring + panIssues
	if total == 0 {
		return
	}

	var recipients []identity.User
	db.WithContext(ctx).Where("role IN ?", []string{"hr_manager", "hr", "admin", "super_admin"}).Find(&recipients)
	if len(recipients) == 0 {
		return
	}
	msg := fmt.Sprintf("Compliance: %d expired licence(s), %d expiring within 30 days, %d PAN/TDS issue(s). Review the compliance queue.",
		expired, expiring, panIssues)
	notifs := make([]communication.Notification, 0, len(recipients))
	for _, r := range recipients {
		notifs = append(notifs, communication.Notification{
			UserID: r.ID, Title: "Doctor compliance review", Message: msg,
			Type: ternary(expired > 0, "error", "warning"),
		})
	}
	if err := db.WithContext(ctx).Create(&notifs).Error; err != nil {
		log.Printf("[Compliance] digest notify failed: %v", err)
	} else {
		log.Printf("[Compliance] digest sent to %d HR/admins (%d issues)", len(recipients), total)
	}
}

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

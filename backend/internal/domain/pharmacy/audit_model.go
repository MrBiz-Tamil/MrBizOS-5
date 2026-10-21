package pharmacy

import "time"

// ═══════════════════════════════════════════════════════════════════════════
// PHARMACY ADMIN AUDIT SYSTEM
//
// A self-contained audit module that runs automated health checks across the
// pharmacy admin surface and routes each finding to the responsible person/role
// for resolution.
//
// Design principles:
//   - Every check has a SEVERITY (critical → info)
//   - Every check has an OWNER (which role should fix it)
//   - Every check has an ACTION (what to do to resolve it)
//   - Every check is RUNNABLE multiple times safely (idempotent reads only)
//   - Results are EPHEMERAL (computed on-demand, not stored)
// ═══════════════════════════════════════════════════════════════════════════

// Severity classifies how urgent an issue is.
type AuditSeverity string

const (
	SeverityCritical AuditSeverity = "critical" // System broken or compliance violation — fix immediately
	SeverityHigh     AuditSeverity = "high"     // Major operational impact — fix today
	SeverityMedium   AuditSeverity = "medium"   // Workflow degradation — fix this week
	SeverityLow      AuditSeverity = "low"      // Quality issue — schedule fix
	SeverityInfo     AuditSeverity = "info"     // FYI / opportunity — no action required
)

// Owner identifies which role is responsible for fixing an issue.
type AuditOwner string

const (
	OwnerBackendDev       AuditOwner = "backend_developer"    // API issues, migrations, code bugs
	OwnerFrontendDev      AuditOwner = "frontend_developer"   // UI bugs, broken pages
	OwnerPharmacyManager  AuditOwner = "pharmacy_manager"     // Operational config (vendors, BOM, etc.)
	OwnerPharmacist       AuditOwner = "pharmacist"           // Daily workflow (Rx queue, dispense)
	OwnerComplianceOfficer AuditOwner = "compliance_officer"  // Regulatory issues (recalls, audit, Schedule H)
	OwnerFinanceManager   AuditOwner = "finance_manager"      // Financial reconciliation, payments
	OwnerSystemAdmin      AuditOwner = "system_admin"         // Permissions, security, config
	OwnerCentralAdmin     AuditOwner = "central_admin"        // Cross-network/strategic
)

// Category groups related checks for filtering in the UI.
type AuditCategory string

const (
	CategoryFrontend     AuditCategory = "frontend"
	CategoryBackend      AuditCategory = "backend"
	CategoryDataIntegrity AuditCategory = "data_integrity"
	CategoryOperations   AuditCategory = "operations"
	CategoryCompliance   AuditCategory = "compliance"
	CategorySecurity     AuditCategory = "security"
	CategoryFinancial    AuditCategory = "financial"
	CategoryPerformance  AuditCategory = "performance"
)

// AuditIssue is a single finding from a check.
type AuditIssue struct {
	ID          string         `json:"id"`           // Stable check ID, e.g. "no-active-pharmacy"
	Severity    AuditSeverity  `json:"severity"`
	Owner       AuditOwner     `json:"owner"`
	Category    AuditCategory  `json:"category"`
	Title       string         `json:"title"`        // Short human-readable title
	Description string         `json:"description"`  // Full explanation
	Action      string         `json:"action"`       // What to do to fix it
	Impact      string         `json:"impact"`       // What happens if not fixed
	Count       int            `json:"count"`        // How many items affected (e.g. 12 vendors expiring)
	Items       []AuditItem    `json:"items,omitempty"` // Affected entities (medicines, orders, vendors, etc.)
	FixURL      string         `json:"fix_url,omitempty"` // Deep link to the page where it's fixed
	Metadata    map[string]any `json:"metadata,omitempty"` // Free-form context
	DetectedAt  time.Time      `json:"detected_at"`
}

// AuditItem is one affected entity for an issue.
type AuditItem struct {
	Type        string `json:"type"`        // "medicine", "vendor", "order", "prescription", etc.
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// AuditReport is the result of running all checks.
type AuditReport struct {
	GeneratedAt    time.Time                       `json:"generated_at"`
	DurationMS     int64                           `json:"duration_ms"`
	TotalChecks    int                             `json:"total_checks"`
	TotalIssues    int                             `json:"total_issues"`
	BySeverity     map[AuditSeverity]int           `json:"by_severity"`
	ByOwner        map[AuditOwner]int              `json:"by_owner"`
	ByCategory     map[AuditCategory]int           `json:"by_category"`
	Issues         []AuditIssue                    `json:"issues"`
	HealthScore    int                             `json:"health_score"`    // 0-100
	Status         string                          `json:"status"`          // "healthy" | "degraded" | "critical"
}

// OwnerDirectory provides display info + contact routing per owner role.
type OwnerInfo struct {
	Owner       AuditOwner `json:"owner"`
	Label       string     `json:"label"`        // Human-readable role name
	Description string     `json:"description"`  // What this role does
	Email       string     `json:"email"`        // Mailing list / on-call email
	Slack       string     `json:"slack"`        // Slack channel
	Escalation  string     `json:"escalation"`   // Who to escalate to if unresponsive
}

// GetOwnerInfo returns the directory entry for an owner role.
// In production, this would be loaded from config/DB.
func GetOwnerInfo(o AuditOwner) OwnerInfo {
	switch o {
	case OwnerBackendDev:
		return OwnerInfo{
			Owner: o, Label: "Backend Developer",
			Description: "Owns API endpoints, services, database migrations, and integrations.",
			Email: "backend@example.com", Slack: "#eng-backend",
			Escalation: "Engineering Manager",
		}
	case OwnerFrontendDev:
		return OwnerInfo{
			Owner: o, Label: "Frontend Developer",
			Description: "Owns React/Next.js pages, components, and UI bugs.",
			Email: "frontend@example.com", Slack: "#eng-frontend",
			Escalation: "Engineering Manager",
		}
	case OwnerPharmacyManager:
		return OwnerInfo{
			Owner: o, Label: "Pharmacy Manager",
			Description: "Owns city-level operations: vendors, BOMs, financials, forecasting.",
			Email: "pharmacy-managers@example.com", Slack: "#pharmacy-ops",
			Escalation: "Regional Head",
		}
	case OwnerPharmacist:
		return OwnerInfo{
			Owner: o, Label: "Pharmacist (Shift)",
			Description: "Handles day-to-day: prescriptions, dispense queue, POS sales.",
			Email: "pharmacists@example.com", Slack: "#pharmacy-floor",
			Escalation: "Pharmacy Manager",
		}
	case OwnerComplianceOfficer:
		return OwnerInfo{
			Owner: o, Label: "Compliance Officer",
			Description: "Owns regulatory: recalls, Schedule H, audit, cold chain compliance.",
			Email: "compliance@example.com", Slack: "#compliance",
			Escalation: "Quality Head",
		}
	case OwnerFinanceManager:
		return OwnerInfo{
			Owner: o, Label: "Finance Manager",
			Description: "Owns financial reconciliation, supplier payments, refunds.",
			Email: "finance@example.com", Slack: "#finance",
			Escalation: "CFO",
		}
	case OwnerSystemAdmin:
		return OwnerInfo{
			Owner: o, Label: "System Administrator",
			Description: "Owns permissions, security config, API keys, infrastructure.",
			Email: "sysadmin@example.com", Slack: "#it-ops",
			Escalation: "CTO",
		}
	case OwnerCentralAdmin:
		return OwnerInfo{
			Owner: o, Label: "Central Admin",
			Description: "Owns cross-network strategy: hub expansion, network policies.",
			Email: "admin@example.com", Slack: "#leadership",
			Escalation: "COO",
		}
	}
	return OwnerInfo{Owner: o, Label: string(o)}
}

// ListOwners returns all owner directory entries (for the audit UI to display).
func ListOwners() []OwnerInfo {
	return []OwnerInfo{
		GetOwnerInfo(OwnerBackendDev),
		GetOwnerInfo(OwnerFrontendDev),
		GetOwnerInfo(OwnerPharmacyManager),
		GetOwnerInfo(OwnerPharmacist),
		GetOwnerInfo(OwnerComplianceOfficer),
		GetOwnerInfo(OwnerFinanceManager),
		GetOwnerInfo(OwnerSystemAdmin),
		GetOwnerInfo(OwnerCentralAdmin),
	}
}

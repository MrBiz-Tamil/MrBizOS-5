package database

import (
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// Tenant scoping (T3) — data-layer isolation helpers.
//
// Domain models in this codebase scope rows by a uint OrganizationID column (see
// internal/domain/task, internal/domain/finance). These helpers convert the
// authorized tenant string carried on the request context ("tenant-<n>") into
// that organization id and apply it as a GORM query scope, so a handler cannot
// accidentally read or write another tenant's rows.
//
// NOTE: full defense-in-depth also requires PostgreSQL row-level security with a
// per-request tenant GUC (ADR-5). That is the next T3 increment and complements —
// does not replace — this application-layer scoping.

// OrgIDFromTenant parses "tenant-<n>" into its organization id. Returns 0 for an
// empty or malformed value so callers can fail closed (refuse rather than read
// everything).
func OrgIDFromTenant(tenantID string) uint {
	s := strings.TrimPrefix(strings.TrimSpace(tenantID), "tenant-")
	if s == "" {
		return 0
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return uint(n)
}

// OrgScope returns a GORM scope that restricts a query to a single organization.
// Apply it with db.Scopes(OrgScope(orgID)). An orgID of 0 is treated as
// unsatisfiable (organization_id = 0) so a missing tenant never widens access.
func OrgScope(orgID uint) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("organization_id = ?", orgID)
	}
}

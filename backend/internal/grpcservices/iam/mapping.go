package iam

import (
	"fmt"
	"strconv"
	"strings"
)

// This file holds the (documented, deterministic) mappings between the persisted
// identity.User model — which scopes users by uint OrganizationID / BusinessUnitID —
// and the IAM proto world, which uses string tenant IDs and business-unit slugs.
//
// These mappings are intentionally simple and lossless for the current
// single-org deployment. Full first-class multi-tenant / multi-BU *membership*
// persistence (string tenant records, per-user BU membership rows) is tracked as
// T4 (org-model) in docs/prd. Until then we derive scope conservatively from the
// authenticated user's own OrganizationID/BusinessUnitID — never from client input.

// buSlugByID maps the numeric BusinessUnitID used across the domain models
// (see internal/domain/task/model.go and internal/domain/finance) to the BU
// slug used by the IAM proto and the frontend workspace switcher.
var buSlugByID = map[uint]string{
	1: "travel",
	2: "healthcare",
	3: "realestate",
	4: "ecommerce",
	5: "consulting",
}

// buIDBySlug is the reverse of buSlugByID.
var buIDBySlug = map[string]uint{
	"travel":     1,
	"healthcare": 2,
	"realestate": 3,
	"ecommerce":  4,
	"consulting": 5,
}

// allBUSlugs is the full set of business units a platform super-admin may act in.
// Order matches ListAccessibleUnits.
var allBUSlugs = []string{"travel", "healthcare", "realestate", "ecommerce", "consulting"}

// tenantStringForOrg renders the stable tenant identifier for an organization id.
// org 1 -> "tenant-1". Deterministic; no fabrication.
func tenantStringForOrg(orgID uint) string {
	if orgID == 0 {
		orgID = 1
	}
	return fmt.Sprintf("tenant-%d", orgID)
}

// orgIDFromTenant parses "tenant-<n>" back to the organization id. Returns 0 when
// the string is empty or malformed, so callers can fail closed.
func orgIDFromTenant(tenantID string) uint {
	s := strings.TrimSpace(tenantID)
	if s == "" {
		return 0
	}
	s = strings.TrimPrefix(s, "tenant-")
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return uint(n)
}

// buSlug returns the slug for a BusinessUnitID, defaulting to healthcare (2) which
// is the legacy default on the identity.User model.
func buSlug(buID uint) string {
	if slug, ok := buSlugByID[buID]; ok {
		return slug
	}
	return "healthcare"
}

// isPlatformSuperAdmin reports whether a role may cross tenant boundaries.
// Only the explicit platform role "superadmin" qualifies — a tenant-level "admin"
// is NOT a platform super-admin (that distinction is the T2 fix for the previous
// behaviour where any admin could read any tenant).
func isPlatformSuperAdmin(role string) bool {
	return strings.EqualFold(strings.TrimSpace(role), "superadmin")
}

// allowedBUsForRole returns the business units a user may switch into. A platform
// super-admin may act in all BUs; every other user is restricted to their own BU
// until per-user BU membership is persisted (T4). This is strictly more
// restrictive than the previous behaviour (which granted every caller all BUs).
func allowedBUsForRole(role string, ownBU uint) []string {
	if isPlatformSuperAdmin(role) {
		out := make([]string, len(allBUSlugs))
		copy(out, allBUSlugs)
		return out
	}
	return []string{buSlug(ownBU)}
}

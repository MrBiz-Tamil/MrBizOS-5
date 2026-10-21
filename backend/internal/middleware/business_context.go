package middleware

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// BusinessUnit IDs corresponding to Group Business OS architecture.
const (
	BusinessUnitTravel      uint = 1
	BusinessUnitHealthcare  uint = 2
	BusinessUnitRealEstate  uint = 3
	BusinessUnitEcommerce   uint = 4
	BusinessUnitConsulting uint = 5
)

// BusinessContext extracts organizational, business unit, and site scoping
// context from request headers, query parameters, or JWT claims and sets them in Fiber locals.
func BusinessContext() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Business Unit ID
		buHeader := c.Get("X-Business-Unit-Id")
		if buHeader != "" {
			if buID, err := strconv.ParseUint(buHeader, 10, 32); err == nil {
				c.Locals("business_unit_id", uint(buID))
			}
		}
		if c.Locals("business_unit_id") == nil {
			if buQuery := c.Query("business_unit_id"); buQuery != "" {
				if buID, err := strconv.ParseUint(buQuery, 10, 32); err == nil {
					c.Locals("business_unit_id", uint(buID))
				}
			}
		}
		// Default to Healthcare (2) for backward compatibility if unassigned
		if c.Locals("business_unit_id") == nil {
			c.Locals("business_unit_id", BusinessUnitHealthcare)
		}

		// 2. Organization ID
		orgHeader := c.Get("X-Organization-Id")
		if orgHeader != "" {
			if orgID, err := strconv.ParseUint(orgHeader, 10, 32); err == nil {
				c.Locals("organization_id", uint(orgID))
			}
		}
		if c.Locals("organization_id") == nil {
			c.Locals("organization_id", uint(1)) // Default Group Organization
		}

		// 3. Site ID
		siteHeader := c.Get("X-Site-Id")
		if siteHeader != "" {
			if siteID, err := strconv.ParseUint(siteHeader, 10, 32); err == nil {
				c.Locals("site_id", uint(siteID))
			}
		}
		if c.Locals("site_id") == nil {
			c.Locals("site_id", uint(1)) // Default Primary Site
		}

		return c.Next()
	}
}

// GetBusinessUnitID retrieves the active BusinessUnitID from Fiber locals.
func GetBusinessUnitID(c *fiber.Ctx) uint {
	if buID, ok := c.Locals("business_unit_id").(uint); ok {
		return buID
	}
	return BusinessUnitHealthcare
}

// GetOrganizationID retrieves the active OrganizationID from Fiber locals.
func GetOrganizationID(c *fiber.Ctx) uint {
	if orgID, ok := c.Locals("organization_id").(uint); ok {
		return orgID
	}
	return 1
}

// GetSiteID retrieves the active SiteID from Fiber locals.
func GetSiteID(c *fiber.Ctx) uint {
	if siteID, ok := c.Locals("site_id").(uint); ok {
		return siteID
	}
	return 1
}

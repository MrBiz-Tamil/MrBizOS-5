package content

import "github.com/microcosm-cc/bluemonday"

// blogPolicy is the allowlist applied to all blog HTML before it is stored and
// (later) rendered via dangerouslySetInnerHTML. It permits rich formatting,
// images, tables and callouts, but strips <script>, event handlers (on*),
// javascript: URLs and anything else not explicitly allowed — closing the XSS
// vector that a rich (TipTap) editor would otherwise open.
var blogPolicy = buildBlogPolicy()

func buildBlogPolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowImages()
	p.AllowTables()
	p.AllowStandardURLs()
	// Callout / alert blocks and editor styling hooks.
	p.AllowAttrs("class").Globally()
	p.AllowElements("figure", "figcaption")
	// Safe link hardening.
	p.RequireNoReferrerOnLinks(true)
	p.AddTargetBlankToFullyQualifiedLinks(true)
	return p
}

// SanitizeHTML returns XSS-safe HTML for storage/render.
func SanitizeHTML(in string) string {
	return blogPolicy.Sanitize(in)
}

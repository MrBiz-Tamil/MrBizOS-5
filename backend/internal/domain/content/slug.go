package content

import (
	"regexp"
	"strings"
)

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// slugify turns a title into a URL-safe slug ("Siddha for Diabetes!" → "siddha-for-diabetes").
func slugify(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = nonSlugChars.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 80 {
		s = strings.Trim(s[:80], "-")
	}
	if s == "" {
		s = "post"
	}
	return s
}

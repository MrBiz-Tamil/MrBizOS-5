package v2

import (
	"sort"
	"strconv"
)

// uintStr is strconv.FormatUint with the common base for Redis keys.
func uintStr(n uint) string {
	return strconv.FormatUint(uint64(n), 10)
}

// sortToolsByName mutates the slice into deterministic order so the prompt
// payload is byte-identical for identical persona+catalog combinations.
// That stability is what lets Gemini's prompt cache do useful work for us.
func sortToolsByName(t []Tool) {
	sort.Slice(t, func(i, j int) bool { return t[i].Name < t[j].Name })
}

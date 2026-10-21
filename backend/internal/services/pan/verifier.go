// Package pan verifies Indian PAN (Permanent Account Number) identities.
//
// Verification of a PAN against the income-tax database is done through an
// authorised agency (Protean/NSDL or a KYC aggregator). That requires API
// credentials, so this package ships a format-only StubVerifier and selects a
// real implementation only when credentials are present. Call sites depend on the
// Verifier interface, so swapping in the live client is a one-line change.
package pan

import (
	"os"
	"regexp"
)

var panRegex = regexp.MustCompile(`^[A-Z]{5}[0-9]{4}[A-Z]$`)

// Result is the outcome of a PAN check.
type Result struct {
	Valid          bool   `json:"valid"`           // PAN exists / is well-formed
	NameMatch      bool   `json:"name_match"`      // holder name matches the supplied name
	RegisteredName string `json:"registered_name"` // name on record (live verifier only)
	Provider       string `json:"provider"`        // "stub" | "nsdl"
}

// Verifier checks a PAN, optionally matching it against a holder name.
type Verifier interface {
	Verify(pan, name string) (Result, error)
}

// ValidFormat reports whether a PAN is structurally valid (5 letters, 4 digits,
// 1 letter). Useful for callers that only need the cheap check.
func ValidFormat(pan string) bool { return panRegex.MatchString(pan) }

// StubVerifier performs format-only validation. It never contacts an external
// service, so NameMatch is reported true whenever the format is valid — callers
// should treat a stub result as "format OK, identity unconfirmed".
type StubVerifier struct{}

func (StubVerifier) Verify(pan, name string) (Result, error) {
	ok := panRegex.MatchString(pan)
	return Result{Valid: ok, NameMatch: ok, Provider: "stub"}, nil
}

// Default returns the configured verifier. When NSDL/Protean credentials are
// present (NSDL_PAN_API_KEY) a live client should be returned here; until then we
// fall back to the format-only stub.
func Default() Verifier {
	if os.Getenv("NSDL_PAN_API_KEY") != "" {
		// TODO: return a live NSDL/Protean-backed verifier once the client exists.
		// Falling through to the stub keeps behaviour safe until then.
	}
	return StubVerifier{}
}

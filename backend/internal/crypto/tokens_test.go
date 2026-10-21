package crypto_test

import (
	"strconv"
	"strings"
	"testing"

	"drayya-core/internal/crypto"
)

// TestOTPFormat verifies that every generated OTP is a zero-padded 6-digit string.
func TestOTPFormat(t *testing.T) {
	for i := 0; i < 1000; i++ {
		otp := crypto.OTP()
		if len(otp) != 6 {
			t.Errorf("iteration %d: OTP length %d, want 6: %q", i, len(otp), otp)
		}
		n, err := strconv.Atoi(otp)
		if err != nil {
			t.Errorf("iteration %d: OTP %q is not numeric: %v", i, otp, err)
		}
		if n < 0 || n > 999999 {
			t.Errorf("iteration %d: OTP %d out of range [0, 999999]", i, n)
		}
	}
}

// TestOTPEntropy verifies that generated OTPs are not all identical (i.e., the
// PRNG produces distinct values rather than returning the same number every call).
func TestOTPEntropy(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		seen[crypto.OTP()] = true
	}
	// With 1,000,000 possible values and 1,000 draws we expect ~1,000 unique values.
	// Requiring ≥900 unique values is a very conservative threshold.
	if len(seen) < 900 {
		t.Errorf("OTP entropy too low: only %d unique values in 1000 calls (expected ≥900)", len(seen))
	}
}

// TestOTPZeroPadded verifies the zero-padding of small values.
func TestOTPZeroPadded(t *testing.T) {
	// We can't force a specific value from crypto/rand, but we can verify the
	// format contract by checking that any returned OTP with a leading zero is
	// still 6 characters long.
	for i := 0; i < 10000; i++ {
		otp := crypto.OTP()
		if len(otp) != 6 {
			t.Errorf("OTP %q has length %d, want 6 (not zero-padded?)", otp, len(otp))
		}
	}
}

// TestSecureTokenLength verifies that SecureToken(n) returns a hex string of
// exactly 2n characters.
func TestSecureTokenLength(t *testing.T) {
	cases := []int{16, 32, 64, 128}
	for _, byteLen := range cases {
		tok := crypto.SecureToken(byteLen)
		wantLen := byteLen * 2 // hex encodes each byte as 2 chars
		if len(tok) != wantLen {
			t.Errorf("SecureToken(%d) length %d, want %d", byteLen, len(tok), wantLen)
		}
	}
}

// TestSecureTokenIsHex verifies that every SecureToken result is a valid lower-case hex string.
func TestSecureTokenIsHex(t *testing.T) {
	const hexChars = "0123456789abcdef"
	for i := 0; i < 100; i++ {
		tok := crypto.SecureToken(32)
		for j, ch := range tok {
			if !strings.ContainsRune(hexChars, ch) {
				t.Errorf("SecureToken contains non-hex character %q at position %d in %q", ch, j, tok)
				break
			}
		}
	}
}

// TestSecureTokenUniqueness verifies that SecureToken never returns duplicate values
// across 1,000 calls (the probability of a collision on 256-bit tokens is negligible).
func TestSecureTokenUniqueness(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		tok := crypto.SecureToken(32)
		if seen[tok] {
			t.Errorf("duplicate token generated at iteration %d: %s", i, tok)
		}
		seen[tok] = true
	}
}

package crypto

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
)

// OTP generates a cryptographically random 6-digit numeric OTP string.
func OTP() string {
	n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(1000000))
	if err != nil {
		// crypto/rand failure is catastrophic — panic
		panic("crypto/rand unavailable: " + err.Error())
	}
	return fmt.Sprintf("%06d", n.Int64())
}

// SecureToken generates a cryptographically random URL-safe hex token of byteLen bytes.
// byteLen=32 produces a 64-char hex string (256 bits).
func SecureToken(byteLen int) string {
	b := make([]byte, byteLen)
	if _, err := cryptorand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// chainPayload is the canonical byte sequence used to compute the hash of one
// AuditLog row. ANY change to fields here is a breaking change to the chain —
// existing chains can no longer be verified. Keep this list frozen.
//
// We deliberately exclude `HashChain` itself (chicken-and-egg) and `ID`
// (set by the DB after insert).
func chainPayload(l *AuditLog) []byte {
	// Stable string concatenation — no map iteration, no JSON ordering surprises.
	// Each field is followed by a null byte separator so "foo|bar" and "fo|obar"
	// produce different hashes.
	return []byte(fmt.Sprintf(
		"%d\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%d",
		l.ActorID,
		l.ActorRole,
		l.Action,
		l.Table,
		l.RowID,
		string(l.Before),
		string(l.After),
		string(l.Diff),
		l.RequestID,
		l.IPAddress,
		l.UserAgent,
		l.Path,
		l.Method,
		l.CreatedAt.UnixNano(),
	))
}

// computeChainHash returns hex(sha256(prevHashHex || canonicalPayload)).
// `prevHashHex` is the empty string for the very first row in the chain.
func computeChainHash(prevHashHex string, l *AuditLog) string {
	h := sha256.New()
	h.Write([]byte(prevHashHex))
	h.Write(chainPayload(l))
	return hex.EncodeToString(h.Sum(nil))
}

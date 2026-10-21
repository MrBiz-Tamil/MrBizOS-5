package audit

import (
	"time"

	"gorm.io/datatypes"
)

// AuditLog is the canonical record of a data mutation in the system.
// One row per Create / Update / Delete operation, written automatically by
// the GORM audit plugin (see plugin.go).
//
// IMPORTANT operational property: this table is APPEND-ONLY in production.
// Apply this SQL after migration to enforce it:
//
//	REVOKE UPDATE, DELETE ON audit_logs FROM medihub_app;
//
// Day 1 ships CREATE/UPDATE/DELETE capture. Day 3 adds the hash_chain column
// for tamper detection.
type AuditLog struct {
	ID         uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	ActorID    uint           `gorm:"index" json:"actor_id"`
	ActorRole  string         `gorm:"index;size:32" json:"actor_role"`
	Action     string         `gorm:"index;size:10" json:"action"`     // CREATE | UPDATE | DELETE
	Table      string         `gorm:"index;size:64;column:table_name" json:"table_name"`
	RowID      string         `gorm:"index;size:64" json:"row_id"`     // string so it works with UUID/composite PKs
	Before     datatypes.JSON `json:"before,omitempty"`                 // nil on CREATE
	After      datatypes.JSON `json:"after,omitempty"`                  // nil on DELETE
	Diff       datatypes.JSON `json:"diff,omitempty"`                   // {field: [old, new]} — populated Day 2
	// RequestID can be a UUID (36 chars) OR a prefixed system id like
	// "sys-<uuid>" emitted by cron jobs (40 chars). Size bumped to 64 to
	// accommodate both without truncation errors (SQLSTATE 22001).
	RequestID  string         `gorm:"index;size:64" json:"request_id"`
	IPAddress  string         `gorm:"size:45" json:"ip_address"`
	UserAgent  string         `gorm:"size:512" json:"user_agent"`
	Path       string         `gorm:"size:255" json:"path"`
	Method     string         `gorm:"size:10" json:"method"`
	// HashChain is sha256(prev_hash_hex || canonical_row_json). It binds this
	// row to every row before it — tampering with any historic row invalidates
	// every chain hash from that point forward. Verified periodically by
	// /admin/audit-logs/verify-chain.
	HashChain  string         `gorm:"size:64" json:"hash_chain"`
	CreatedAt  time.Time      `gorm:"index" json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }

// AuditSkippable is the opt-OUT marker. Tables NOT implementing it are
// audited by default. To skip a table, give the model:
//
//	func (T) AuditSkip() bool { return true }
//
// Use sparingly — only for high-volume tables that are not records of business
// truth (e.g. ws_presence, rate-limit counters, the audit_logs table itself).
type AuditSkippable interface {
	AuditSkip() bool
}

// AuditSkip on AuditLog prevents infinite recursion (the plugin would otherwise
// try to audit its own writes).
func (AuditLog) AuditSkip() bool { return true }

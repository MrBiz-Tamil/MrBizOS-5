package database

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

// safeIdent allows only the characters PostgreSQL accepts in unquoted
// identifiers (letters, digits, underscores). Used to validate table and
// constraint names before they are inlined into DDL — prevents SQL injection
// because Postgres DO blocks cannot accept parameterized placeholders.
var safeIdent = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// ApplyConstraints adds CHECK constraints that GORM AutoMigrate cannot create.
// Safe to call on every startup — checks information_schema before adding.
//
// IMPORTANT: PostgreSQL's DO $$ ... $$ blocks cannot bind query parameters
// ($1, $2, $3), so identifiers are inlined directly. To stay safe we whitelist
// table/constraint names against [a-zA-Z_][a-zA-Z0-9_]* — any DDL-defined
// constraint must satisfy that regex. CHECK expressions are also defined
// in-code (never user input) so inlining them is safe.
func ApplyConstraints(db *gorm.DB) {
	constraints := []struct {
		table      string
		name       string
		expression string
	}{
		{
			table:      "doctor_availability",
			name:       "chk_day_of_week",
			expression: "day_of_week BETWEEN 0 AND 6",
		},
		{
			table:      "doctor_availability",
			name:       "chk_time_order",
			expression: "start_time < end_time",
		},
		{
			table:      "appointments",
			name:       "chk_appt_payout_status",
			expression: "payout_status IN ('pending', 'processed', 'failed', 'skipped')",
		},
		{
			table: "appointments",
			name:  "chk_appt_status",
			// Includes legacy values ('scheduled', 'doctor_arrived') that exist
			// in production data from before this constraint was introduced.
			// Adding these matches the AppointmentStatus enum in the model.
			expression: "status IN ('pending', 'scheduled', 'confirmed', 'in_consultation', 'doctor_arrived', 'completed', 'cancelled', 'no_show')",
		},
		{
			table:      "otp_records",
			name:       "chk_otp_length",
			expression: "LENGTH(otp) = 6",
		},
		{
			table:      "pharmacy_orders",
			name:       "chk_pharmacy_order_status",
			expression: "status IN ('pending','pending_rx_verification','processing','confirmed','packed','dispensed','shipped','delivered','cancelled','refunded')",
		},
		{
			table:      "pharmacy_orders",
			name:       "chk_pharmacy_order_payment_status",
			expression: "payment_status IN ('pending','paid','failed','refunded')",
		},
		{
			table:      "pharmacy_purchase_orders",
			name:       "chk_purchase_order_status",
			expression: "status IN ('draft','approved','rejected','ordered','received','cancelled')",
		},
		{
			table:      "pharmacyinventory",
			name:       "chk_pharmacy_inventory_stock_non_negative",
			expression: "stock_quantity >= 0",
		},
		{
			table:      "refill_schedules",
			name:       "chk_refill_status",
			expression: "status IN ('active','paused','cancelled','completed')",
		},
		{
			table: "prescriptions",
			name:  "chk_prescription_status",
			// Includes legacy 'pending' and 'active' which exist in production
			// data from earlier schema versions. Adding here prevents
			// "constraint violated by existing row" startup errors.
			expression: "status IN ('uploaded','under_review','verified','rejected','expired','pending','active','draft')",
		},
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("[constraints] failed to get sql.DB: %v", err)
		return
	}

	// Drop legacy FK constraints that conflict with intentional sentinel values.
	// activity_logs.user_id = 0 is used for failed-login events where no user
	// has been authenticated yet. The FK to user(id) rejects those inserts.
	// Dropping the FK is safe — we keep the index for query performance.
	dropFKs := []struct{ table, name string }{
		{"activity_logs", "fk_activity_logs_user"},
	}
	for _, fk := range dropFKs {
		if !safeIdent.MatchString(fk.table) || !safeIdent.MatchString(fk.name) {
			continue
		}
		_, _ = sqlDB.Exec(fmt.Sprintf(
			`ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s`,
			quoteIdent(fk.table), quoteIdent(fk.name),
		))
	}

	// The DoctorProfile model migrated from a single `consultation_fee` bigint
	// column to a `consultation_fees` jsonb column (regional pricing). Many
	// legacy queries still reference the old column name (visibility service,
	// devi booking, productivity dashboard, etc.). To keep them working on
	// fresh deployments where AutoMigrate doesn't create the legacy column,
	// we add it back as a derived bigint that mirrors INR price.
	// This is a no-op once the column already exists.
	_, _ = sqlDB.Exec(`
		ALTER TABLE doctorprofile
		ADD COLUMN IF NOT EXISTS consultation_fee BIGINT DEFAULT 0
	`)
	// Backfill from the jsonb column for any rows still at default 0.
	_, _ = sqlDB.Exec(`
		UPDATE doctorprofile
		SET consultation_fee = COALESCE((consultation_fees->>'INR')::bigint, 0)
		WHERE consultation_fee = 0 AND consultation_fees IS NOT NULL
	`)

	applied := 0
	skipped := 0
	for _, c := range constraints {
		// Validate identifiers — must match [a-zA-Z_][a-zA-Z0-9_]* (safe for inlining).
		if !safeIdent.MatchString(c.table) || !safeIdent.MatchString(c.name) {
			log.Printf("[constraints] skip %s.%s: unsafe identifier", c.table, c.name)
			continue
		}

		// 1) Check existence using a parameterized SELECT (safe).
		var count int
		row := sqlDB.QueryRow(
			`SELECT COUNT(*) FROM information_schema.table_constraints
			 WHERE table_name = $1 AND constraint_name = $2`,
			c.table, c.name,
		)
		if err := row.Scan(&count); err != nil {
			log.Printf("[constraints] failed to check %s.%s: %v", c.table, c.name, err)
			continue
		}
		if count > 0 {
			skipped++
			continue
		}

		// 2) Inline DDL — Postgres ALTER TABLE doesn't accept placeholders for
		//    identifiers, but our values are whitelisted via safeIdent above
		//    and expressions are static strings defined in this file.
		ddl := fmt.Sprintf(
			`ALTER TABLE %s ADD CONSTRAINT %s CHECK (%s)`,
			quoteIdent(c.table), quoteIdent(c.name), c.expression,
		)
		if _, err := sqlDB.Exec(ddl); err != nil {
			log.Printf("[constraints] failed to apply %s.%s: %v", c.table, c.name, err)
		} else {
			applied++
		}
	}
	log.Printf("[constraints] applied %d new, skipped %d existing", applied, skipped)
}

// quoteIdent wraps an identifier in double quotes, escaping any embedded quotes.
// safeIdent should be checked beforehand; this is defense in depth.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

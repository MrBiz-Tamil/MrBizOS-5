package emr

import (
	"drayya-core/internal/utils/privacy"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// raw is a thin row used to read prescription PHI columns as cleartext, bypassing
// the GORM EncryptedString Scan that would auto-decrypt them.
type raw struct {
	ID           uint
	Diagnosis    string
	Symptoms     string
	Instructions string
}

func setupBackfillDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE prescriptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			diagnosis TEXT,
			symptoms TEXT,
			instructions TEXT
		)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}

func TestBackfillEncryptedFields_EncryptsPlaintextAndIsIdempotent(t *testing.T) {
	db := setupBackfillDB(t)

	// Insert one plaintext row, one already-encrypted row, and one empty row.
	plainDiag := "Type 2 Diabetes"
	plainInst := "Take 1 tablet daily after breakfast"
	encSym, err := privacy.Encrypt("fatigue, polyuria")
	if err != nil {
		t.Fatalf("seed encrypt: %v", err)
	}
	db.Exec(`INSERT INTO prescriptions (diagnosis, symptoms, instructions) VALUES (?, ?, ?)`,
		plainDiag, encSym, plainInst)
	db.Exec(`INSERT INTO prescriptions (diagnosis, symptoms, instructions) VALUES (?, ?, ?)`,
		"", "", "")

	BackfillEncryptedFields(db)

	var rows []raw
	if err := db.Table("prescriptions").Find(&rows).Error; err != nil {
		t.Fatalf("read after backfill: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	// Row 1 — plaintext fields should now decrypt back to their originals; the
	// pre-encrypted symptom column should be untouched.
	got, err := privacy.Decrypt(rows[0].Diagnosis)
	if err != nil || got != plainDiag {
		t.Errorf("diagnosis not encrypted-then-decrypted correctly: got %q err=%v", got, err)
	}
	if rows[0].Symptoms != encSym {
		t.Error("already-encrypted symptoms column was rewritten; backfill is not idempotent")
	}
	got, err = privacy.Decrypt(rows[0].Instructions)
	if err != nil || got != plainInst {
		t.Errorf("instructions not encrypted-then-decrypted correctly: got %q err=%v", got, err)
	}

	// Row 2 — empty fields stay empty.
	if rows[1].Diagnosis != "" || rows[1].Symptoms != "" || rows[1].Instructions != "" {
		t.Error("empty fields should not be modified by the backfill")
	}

	// Second run — must be a no-op. Capture state, run, compare.
	snapshot := make([]raw, len(rows))
	copy(snapshot, rows)
	BackfillEncryptedFields(db)

	var after []raw
	db.Table("prescriptions").Find(&after)
	for i := range after {
		if after[i] != snapshot[i] {
			t.Errorf("row %d changed on second backfill run (not idempotent): before=%+v after=%+v",
				i, snapshot[i], after[i])
		}
	}
}

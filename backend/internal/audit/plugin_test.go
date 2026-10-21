package audit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"drayya-core/internal/middleware"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Test fixture — a simple model with sensitive fields to verify redaction.
type testWidget struct {
	ID           uint   `gorm:"primaryKey"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash" audit:"skip"`
	AadhaarLast4 string `json:"aadhaar_last4" audit:"redact"`
}

// A model that opts OUT of auditing entirely
type quietCounter struct {
	ID    uint `gorm:"primaryKey"`
	Value int
}

func (quietCounter) AuditSkip() bool { return true }

func setupTestDB(t *testing.T) (*gorm.DB, *Plugin) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Pin to 1 connection (SQLite :memory: is per-connection)
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(&testWidget{}, &quietCounter{}); err != nil {
		t.Fatal(err)
	}
	p := NewPlugin(100)
	if err := db.Use(p); err != nil {
		t.Fatal(err)
	}
	return db, p
}

// waitForAudit blocks until either `min` audit rows are present or `timeout`
// elapses. Necessary because the worker is async.
func waitForAudit(t *testing.T, db *gorm.DB, min int, timeout time.Duration) []AuditLog {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var rows []AuditLog
		if err := db.Order("id").Find(&rows).Error; err == nil && len(rows) >= min {
			return rows
		}
		time.Sleep(20 * time.Millisecond)
	}
	var rows []AuditLog
	db.Find(&rows)
	t.Fatalf("expected ≥%d audit rows within %v, got %d", min, timeout, len(rows))
	return rows
}

func TestAuditPlugin_CapturesCreate(t *testing.T) {
	db, p := setupTestDB(t)
	defer p.Stop()

	ac := &middleware.AuditContext{
		ActorID:   42,
		ActorRole: "doctor",
		RequestID: "req-xyz",
		IP:        "127.0.0.1",
	}
	ctx := middleware.WithAuditContext(context.Background(), ac)

	w := &testWidget{Name: "Stethoscope", Email: "doc@test.com"}
	if err := db.WithContext(ctx).Create(w).Error; err != nil {
		t.Fatal(err)
	}

	rows := waitForAudit(t, db, 1, 3*time.Second)
	assert.Len(t, rows, 1)
	got := rows[0]
	assert.Equal(t, "CREATE", got.Action)
	assert.Equal(t, "test_widgets", got.Table)
	assert.Equal(t, uint(42), got.ActorID)
	assert.Equal(t, "doctor", got.ActorRole)
	assert.Equal(t, "req-xyz", got.RequestID)
	assert.NotEmpty(t, got.After)
	assert.Empty(t, got.Before)
}

func TestAuditPlugin_CapturesUpdate(t *testing.T) {
	db, p := setupTestDB(t)
	defer p.Stop()

	ac := &middleware.AuditContext{ActorID: 1, ActorRole: "admin"}
	ctx := middleware.WithAuditContext(context.Background(), ac)

	w := &testWidget{Name: "Initial"}
	db.WithContext(ctx).Create(w)

	w.Name = "Updated"
	if err := db.WithContext(ctx).Save(w).Error; err != nil {
		t.Fatal(err)
	}

	rows := waitForAudit(t, db, 2, 3*time.Second)
	assert.GreaterOrEqual(t, len(rows), 2)
	// Last row should be the UPDATE
	last := rows[len(rows)-1]
	assert.Equal(t, "UPDATE", last.Action)
	assert.NotEmpty(t, last.After)
}

func TestAuditPlugin_CapturesDelete(t *testing.T) {
	db, p := setupTestDB(t)
	defer p.Stop()

	ac := &middleware.AuditContext{ActorID: 99, ActorRole: "admin"}
	ctx := middleware.WithAuditContext(context.Background(), ac)

	w := &testWidget{Name: "ToDelete"}
	db.WithContext(ctx).Create(w)

	if err := db.WithContext(ctx).Delete(w).Error; err != nil {
		t.Fatal(err)
	}

	rows := waitForAudit(t, db, 2, 3*time.Second)
	last := rows[len(rows)-1]
	assert.Equal(t, "DELETE", last.Action)
	// Before should contain the row state at the time of deletion
	assert.NotEmpty(t, last.Before)
}

func TestAuditPlugin_RedactsSensitiveFields(t *testing.T) {
	db, p := setupTestDB(t)
	defer p.Stop()

	ctx := middleware.WithAuditContext(context.Background(), &middleware.AuditContext{ActorID: 1})

	w := &testWidget{
		Name:         "PII Test",
		Email:        "user@test.com",
		PasswordHash: "supersecret-bcrypt-hash",
		AadhaarLast4: "1234",
	}
	db.WithContext(ctx).Create(w)

	rows := waitForAudit(t, db, 1, 3*time.Second)
	var after map[string]interface{}
	json.Unmarshal(rows[0].After, &after)

	// password_hash MUST be entirely absent — never even a placeholder
	_, hasPassword := after["password_hash"]
	assert.False(t, hasPassword, "password_hash must not appear in audit log")

	// aadhaar_last4 MUST be redacted (rdct: prefix)
	aadhaar, _ := after["aadhaar_last4"].(string)
	assert.True(t, len(aadhaar) > 0 && aadhaar[:5] == "rdct:",
		"aadhaar_last4 must be redacted, got %q", aadhaar)

	// Non-sensitive fields still present
	assert.Equal(t, "user@test.com", after["email"])
	assert.Equal(t, "PII Test", after["name"])
}

func TestAuditPlugin_SkipsOptedOutTables(t *testing.T) {
	db, p := setupTestDB(t)
	defer p.Stop()

	ctx := middleware.WithAuditContext(context.Background(), &middleware.AuditContext{ActorID: 1})

	q := &quietCounter{Value: 1}
	db.WithContext(ctx).Create(q)
	q.Value = 2
	db.WithContext(ctx).Save(q)

	// Give the worker a beat to drain if anything sneaked in
	time.Sleep(200 * time.Millisecond)

	var rows []AuditLog
	db.Find(&rows)
	for _, r := range rows {
		assert.NotEqual(t, "quiet_counters", r.Table,
			"quietCounter implements AuditSkip — must not be audited")
	}
}

func TestAuditPlugin_DoesNotRecurseOnSelf(t *testing.T) {
	// The audit_logs table must NEVER appear in audit_logs (infinite loop)
	db, p := setupTestDB(t)
	defer p.Stop()

	ctx := middleware.WithAuditContext(context.Background(), &middleware.AuditContext{ActorID: 1})

	db.WithContext(ctx).Create(&testWidget{Name: "Trigger"})
	waitForAudit(t, db, 1, 3*time.Second)

	var selfRefCount int64
	db.Model(&AuditLog{}).Where("table_name = ?", "audit_logs").Count(&selfRefCount)
	assert.Equal(t, int64(0), selfRefCount, "audit_logs must never reference itself")
}

func TestAuditPlugin_BulkFlagSkipsPerRow(t *testing.T) {
	db, p := setupTestDB(t)
	defer p.Stop()

	ac := &middleware.AuditContext{ActorID: 1, Bulk: true}
	ctx := middleware.WithAuditContext(context.Background(), ac)

	db.WithContext(ctx).Create(&testWidget{Name: "Bulk1"})
	db.WithContext(ctx).Create(&testWidget{Name: "Bulk2"})

	time.Sleep(200 * time.Millisecond)

	var count int64
	db.Model(&AuditLog{}).Count(&count)
	assert.Equal(t, int64(0), count, "Bulk flag should suppress per-row audit logs")
}

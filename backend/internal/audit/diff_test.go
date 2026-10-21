package audit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"drayya-core/internal/middleware"

	"github.com/stretchr/testify/assert"
)

func TestComputeDiff_BasicChanges(t *testing.T) {
	before := map[string]interface{}{
		"name":   "Old",
		"status": "pending",
		"count":  5,
	}
	after := map[string]interface{}{
		"name":   "New",
		"status": "pending", // unchanged
		"count":  7,
	}
	diff := computeDiff(before, after)

	assert.Len(t, diff, 2, "only 2 changed fields")
	assert.Equal(t, []interface{}{"Old", "New"}, diff["name"])
	assert.Equal(t, []interface{}{5, 7}, diff["count"])
	_, statusChanged := diff["status"]
	assert.False(t, statusChanged, "status unchanged must not appear in diff")
}

func TestComputeDiff_AddedAndRemovedFields(t *testing.T) {
	before := map[string]interface{}{"a": 1, "b": 2}
	after := map[string]interface{}{"b": 2, "c": 3}
	diff := computeDiff(before, after)

	assert.Equal(t, []interface{}{nil, 3}, diff["c"], "added field has old=nil")
	assert.Equal(t, []interface{}{1, nil}, diff["a"], "removed field has new=nil")
	_, bChanged := diff["b"]
	assert.False(t, bChanged, "b unchanged")
}

func TestComputeDiff_EmptyInputsReturnNil(t *testing.T) {
	assert.Nil(t, computeDiff(nil, nil))
	assert.Nil(t, computeDiff(map[string]interface{}{}, map[string]interface{}{}))
}

func TestAuditPlugin_UpdateProducesDiff(t *testing.T) {
	db, p := setupTestDB(t)
	defer p.Stop()

	ctx := middleware.WithAuditContext(context.Background(), &middleware.AuditContext{ActorID: 1, ActorRole: "admin"})

	w := &testWidget{Name: "Original", Email: "before@test.com"}
	if err := db.WithContext(ctx).Create(w).Error; err != nil {
		t.Fatal(err)
	}

	// Update one field
	w.Name = "Renamed"
	if err := db.WithContext(ctx).Save(w).Error; err != nil {
		t.Fatal(err)
	}

	rows := waitForAudit(t, db, 2, 3*time.Second)
	var updateRow *AuditLog
	for i := range rows {
		if rows[i].Action == "UPDATE" {
			updateRow = &rows[i]
			break
		}
	}
	if updateRow == nil {
		t.Fatal("no UPDATE row found in audit_logs")
	}

	// Diff should mention name field change
	var diff map[string]interface{}
	if err := json.Unmarshal(updateRow.Diff, &diff); err != nil {
		t.Fatalf("diff not valid JSON: %v", err)
	}

	change, ok := diff["name"].([]interface{})
	assert.True(t, ok, "diff must include name change, got: %v", diff)
	assert.Equal(t, "Original", change[0])
	assert.Equal(t, "Renamed", change[1])

	// email did not change — must not appear in diff
	_, emailChanged := diff["email"]
	assert.False(t, emailChanged, "unchanged field must not appear in diff")
}

package audit

import (
	"context"
	"testing"
	"time"

	"drayya-core/internal/middleware"

	"github.com/stretchr/testify/assert"
)

func TestChainHash_DeterministicForSameInput(t *testing.T) {
	row := AuditLog{
		ActorID: 1, Action: "CREATE", Table: "widgets",
		RowID: "5", CreatedAt: time.Unix(1700000000, 0),
	}
	h1 := computeChainHash("prev", &row)
	h2 := computeChainHash("prev", &row)
	assert.Equal(t, h1, h2, "same input must produce same hash")
}

func TestChainHash_ChangesWithAnyField(t *testing.T) {
	base := AuditLog{
		ActorID: 1, Action: "CREATE", Table: "widgets",
		RowID: "5", CreatedAt: time.Unix(1700000000, 0),
	}
	h := computeChainHash("", &base)

	mutated := base
	mutated.ActorID = 2
	assert.NotEqual(t, h, computeChainHash("", &mutated), "actor change must alter hash")

	mutated = base
	mutated.Action = "UPDATE"
	assert.NotEqual(t, h, computeChainHash("", &mutated), "action change must alter hash")

	mutated = base
	mutated.RowID = "6"
	assert.NotEqual(t, h, computeChainHash("", &mutated), "row_id change must alter hash")
}

func TestChainHash_PrevHashPropagates(t *testing.T) {
	row := AuditLog{
		ActorID: 1, Action: "CREATE", Table: "widgets", RowID: "5",
		CreatedAt: time.Unix(1700000000, 0),
	}
	h1 := computeChainHash("a", &row)
	h2 := computeChainHash("b", &row)
	assert.NotEqual(t, h1, h2, "different prev hash must produce different result")
}

func TestVerifyChain_IntactAfterNormalWrites(t *testing.T) {
	db, p := setupTestDB(t)
	defer p.Stop()

	ctx := middleware.WithAuditContext(context.Background(), &middleware.AuditContext{ActorID: 1})

	// Produce a few CREATE events
	for i := 0; i < 5; i++ {
		db.WithContext(ctx).Create(&testWidget{Name: "w"})
	}
	// Force a flush so they're persisted with chain hashes
	waitForAudit(t, db, 5, 3*time.Second)

	total, broken, err := VerifyChain(db)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(5))
	assert.Equal(t, uint64(0), broken, "chain must be intact, got break at %d", broken)
}

func TestVerifyChain_DetectsTampering(t *testing.T) {
	db, p := setupTestDB(t)
	defer p.Stop()

	ctx := middleware.WithAuditContext(context.Background(), &middleware.AuditContext{ActorID: 1})

	for i := 0; i < 3; i++ {
		db.WithContext(ctx).Create(&testWidget{Name: "honest"})
	}
	waitForAudit(t, db, 3, 3*time.Second)

	// Simulate tampering: modify the middle row directly (bypassing the worker)
	db.Exec(`UPDATE audit_logs SET actor_id = 999 WHERE id = (SELECT id FROM audit_logs ORDER BY id LIMIT 1 OFFSET 1)`)

	_, broken, err := VerifyChain(db)
	assert.NoError(t, err)
	assert.NotEqual(t, uint64(0), broken, "tampering must be detected")
}

package workflow

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	workflowv1 "drayya-core/gen/workflow/v1"
	"drayya-core/internal/authctx"
	taskdomain "drayya-core/internal/domain/task"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(&taskdomain.Task{}); err != nil {
		t.Fatalf("migrate task: %v", err)
	}
	db.Exec("DELETE FROM tasks")
	return db
}

// T3 regression: ListTasks must only ever return rows for the authenticated
// caller's organization, and must return nothing when the context carries no
// authorized tenant (fail closed).
func TestListTasksTenantIsolation(t *testing.T) {
	db := newTestDB(t)
	// Two tenants, one task each.
	if err := db.Create(&taskdomain.Task{OrganizationID: 1, Title: "ACME task", Status: "TODO", Priority: "HIGH", OwnerID: 1}).Error; err != nil {
		t.Fatalf("seed org1: %v", err)
	}
	if err := db.Create(&taskdomain.Task{OrganizationID: 2, Title: "Globex task", Status: "TODO", Priority: "HIGH", OwnerID: 9}).Error; err != nil {
		t.Fatalf("seed org2: %v", err)
	}
	srv := NewServer(db)

	listFor := func(tenant string) []*workflowv1.TaskProto {
		ctx := context.Background()
		if tenant != "" {
			ctx = authctx.WithTenantID(ctx, tenant)
		}
		res, err := srv.ListTasks(ctx, connect.NewRequest(&workflowv1.ListTasksRequest{}))
		if err != nil {
			t.Fatalf("ListTasks(%q): %v", tenant, err)
		}
		return res.Msg.Tasks
	}

	t.Run("tenant-1 sees only its own task", func(t *testing.T) {
		got := listFor("tenant-1")
		if len(got) != 1 || got[0].Title != "ACME task" {
			t.Fatalf("tenant-1 tasks = %+v, want only [ACME task]", got)
		}
	})

	t.Run("tenant-2 sees only its own task", func(t *testing.T) {
		got := listFor("tenant-2")
		if len(got) != 1 || got[0].Title != "Globex task" {
			t.Fatalf("tenant-2 tasks = %+v, want only [Globex task]", got)
		}
	})

	t.Run("no tenant context returns nothing", func(t *testing.T) {
		got := listFor("")
		if len(got) != 0 {
			t.Fatalf("unauthenticated tasks = %+v, want empty", got)
		}
	})
}

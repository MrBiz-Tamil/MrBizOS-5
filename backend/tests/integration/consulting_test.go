package integration

import (
	"testing"
	"time"

	"drayya-core/internal/domain/consulting"
	"drayya-core/internal/domain/journey"
	"drayya-core/internal/domain/task"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupConsultingTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to memory db: %v", err)
	}

	if err := task.AutoMigrate(db); err != nil {
		t.Fatalf("task automigrate failed: %v", err)
	}
	if err := journey.AutoMigrate(db); err != nil {
		t.Fatalf("journey automigrate failed: %v", err)
	}
	if err := consulting.AutoMigrate(db); err != nil {
		t.Fatalf("consulting automigrate failed: %v", err)
	}

	return db
}

func TestSoftwareConsultingWorkflow(t *testing.T) {
	db := setupConsultingTestDB(t)

	taskSvc := task.NewService(task.NewRepository(db))
	journeySvc := journey.NewService(db)
	consultingSvc := consulting.NewService(consulting.NewRepository(db), journeySvc, taskSvc)

	// 1. Create B2B Client Account
	client, err := consultingSvc.CreateClient(&consulting.ClientAccount{
		CompanyName:    "Acme Health Systems Corp",
		PrimaryContact: "Sarah Jenkins",
		Email:          "sarah@acmehealth.com",
		Industry:       "Healthcare IT",
	})
	if err != nil || client.ID == 0 {
		t.Fatalf("CreateClient failed: %v", err)
	}

	// 2. Create Software Project (Auto-creates engineering task)
	proj, err := consultingSvc.CreateProject(client.ID, "Group EMR Integration Platform", "PRJ-ACME-01", 120000.00)
	if err != nil || proj.ID == 0 {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Verify project kickoff task
	tasks, _, err := taskSvc.ListTasks(task.Filter{BusinessUnitID: 5})
	if err != nil || len(tasks) == 0 {
		t.Fatalf("expected software engineering task, got count=%d", len(tasks))
	}

	// 3. Create Milestone (Auto-creates milestone task)
	targetDate := time.Now().Add(30 * 24 * time.Hour)
	ms, err := consultingSvc.CreateMilestone(proj.ID, "Phase 1 Architecture & API Specifications", 40000.00, &targetDate)
	if err != nil || ms.ID == 0 {
		t.Fatalf("CreateMilestone failed: %v", err)
	}

	// 4. Log Developer Timesheet & verify project budget tracking
	ts, err := consultingSvc.LogTimesheet(proj.ID, 501, 8.0, "Built FHIR API integration gateway")
	if err != nil || ts.ID == 0 {
		t.Fatalf("LogTimesheet failed: %v", err)
	}

	// Verify updated spent budget ($85 * 8h = $680)
	updatedProj, _ := consultingSvc.GetProject(proj.ID)
	if updatedProj.Spent != 680.00 {
		t.Fatalf("expected spent budget 680.00, got %.2f", updatedProj.Spent)
	}
}

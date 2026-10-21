package integration

import (
	"testing"

	"drayya-core/internal/domain/customer"
	"drayya-core/internal/domain/journey"
	"drayya-core/internal/domain/task"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
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
	if err := customer.AutoMigrate(db); err != nil {
		t.Fatalf("customer automigrate failed: %v", err)
	}

	return db
}

func TestTaskManagementEngine(t *testing.T) {
	db := setupTestDB(t)
	repo := task.NewRepository(db)
	svc := task.NewService(repo)

	// Create task
	newTask, err := svc.CreateTask(&task.Task{
		OrganizationID: 1,
		BusinessUnitID: 1, // Travel
		OwnerID:        1,
		Title:          "Review Passport & Visa Documents",
		Priority:       task.PriorityHigh,
	})
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	if newTask.ID == 0 {
		t.Fatalf("expected non-zero task ID")
	}

	// List tasks
	tasks, total, err := svc.ListTasks(task.Filter{
		BusinessUnitID: 1,
	})
	if err != nil || total != 1 || len(tasks) != 1 {
		t.Fatalf("ListTasks failed: err=%v total=%d tasksCount=%d", err, total, len(tasks))
	}

	// Complete task
	completed, err := svc.CompleteTask(newTask.ID)
	if err != nil || completed.Status != task.StatusCompleted {
		t.Fatalf("CompleteTask failed: err=%v status=%s", err, completed.Status)
	}
}

func TestJourneyWorkflowEngine(t *testing.T) {
	db := setupTestDB(t)
	svc := journey.NewService(db)

	// Create Journey Template
	_, err := svc.CreateTemplate(&journey.JourneyTemplate{
		OrganizationID: 1,
		BusinessUnitID: 3, // Real Estate
		Code:           "REAL_ESTATE_DEAL",
		Name:           "Real Estate Buyer Deal Flow",
		Stages: []journey.JourneyStage{
			{StageName: "Inquiry", StageOrder: 1},
			{StageName: "Site Visit", StageOrder: 2},
			{StageName: "Offer & Negotiation", StageOrder: 3},
			{StageName: "Closed", StageOrder: 4},
		},
	})
	if err != nil {
		t.Fatalf("CreateTemplate failed: %v", err)
	}

	fetchedTmpl, err := svc.GetTemplateByCode("REAL_ESTATE_DEAL")
	if err != nil || len(fetchedTmpl.Stages) < 2 {
		t.Fatalf("GetTemplateByCode failed: %v", err)
	}

	// Start Journey Instance
	instance, err := svc.StartJourney(fetchedTmpl.ID, 1001, nil, 3, 1)
	if err != nil {
		t.Fatalf("StartJourney failed: %v", err)
	}
	if instance.Status != "ACTIVE" {
		t.Fatalf("expected ACTIVE status, got %s", instance.Status)
	}

	nextStage := fetchedTmpl.Stages[1]
	// Transition Stage
	updated, err := svc.TransitionStage(instance.ID, nextStage.ID, 1, "Scheduled site visit for property #402")
	if err != nil {
		t.Fatalf("TransitionStage failed: %v", err)
	}
	if updated.CurrentStageID != nextStage.ID {
		t.Fatalf("expected stage ID %d, got %d", nextStage.ID, updated.CurrentStageID)
	}
}

func TestCustomer360ClinicalPHIIsolation(t *testing.T) {
	db := setupTestDB(t)
	svc := customer.NewService(db)

	// Create Lead
	lead, err := svc.CreateLead(&customer.Lead{
		OrganizationID: 1,
		BusinessUnitID: 2, // Healthcare
		Name:           "John Doe",
		Email:          "john@example.com",
		Phone:          "+1234567890",
		Source:         "Web Registration",
		Value:          150.0,
	})
	if err != nil {
		t.Fatalf("CreateLead failed: %v", err)
	}

	// Fetch Customer 360 without clinical access
	profileNonClinical, err := svc.GetCustomer360(lead.ID, false)
	if err != nil {
		t.Fatalf("GetCustomer360 failed: %v", err)
	}
	if profileNonClinical.Name != "John Doe" {
		t.Fatalf("expected John Doe, got %s", profileNonClinical.Name)
	}

	// Verify clinical timeline events are sanitized for non-clinical user
	for _, evt := range profileNonClinical.Timeline {
		if evt.IsClinical {
			t.Fatalf("clinical PHI event leaked to non-clinical user context!")
		}
	}
}

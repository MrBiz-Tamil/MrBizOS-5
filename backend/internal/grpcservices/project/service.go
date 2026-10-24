package project

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"connectrpc.com/connect"
	projectv1 "drayya-core/gen/project/v1"
	"drayya-core/gen/project/v1/projectv1connect"
	"drayya-core/internal/domain/consulting"
	"drayya-core/internal/domain/task"
	"gorm.io/gorm"
)

type ProjectServiceServer struct {
	projectv1connect.UnimplementedProjectServiceHandler
	db *gorm.DB
	mu sync.RWMutex

	// In-memory operational store with fast cache
	projects   map[string]*projectv1.Project
	tasks      map[string]*projectv1.UTMSTask
	milestones map[string][]*projectv1.Milestone
	timesheets map[string][]*projectv1.TimesheetEntry
}

func NewService(db *gorm.DB) *ProjectServiceServer {
	s := &ProjectServiceServer{
		db:         db,
		projects:   make(map[string]*projectv1.Project),
		tasks:      make(map[string]*projectv1.UTMSTask),
		milestones: make(map[string][]*projectv1.Milestone),
		timesheets: make(map[string][]*projectv1.TimesheetEntry),
	}

	// Seed sample projects & UTMS tasks
	s.seedDefaults()
	return s
}

func (s *ProjectServiceServer) seedDefaults() {
	s.projects["PRJ-001"] = &projectv1.Project{
		Id:             "PRJ-001",
		Code:           "PRJ-SOW-2026",
		Name:           "FinTech Enterprise ERP Rollout",
		Description:    "Complete cloud transformation, accounting integration, and automated bank feeds",
		BusinessUnit:   "CONSULTING",
		ClientName:     "Acme International Corp",
		Status:         "IN_PROGRESS",
		Budget:         1250000,
		Spent:          420000,
		StartDate:      "2026-09-01",
		TargetDate:     "2026-12-15",
		TotalTasks:     14,
		CompletedTasks: 8,
	}

	s.projects["PRJ-002"] = &projectv1.Project{
		Id:             "PRJ-002",
		Code:           "PRJ-MED-SHIELD",
		Name:           "Apollo Multi-Specialty Clinic Deployment",
		Description:    "EMR onboarding, doctor consultation scheduling, and laboratory HL7 bridge",
		BusinessUnit:   "HEALTH",
		ClientName:     "Apollo Diagnostics Network",
		Status:         "IN_PROGRESS",
		Budget:         850000,
		Spent:          210000,
		StartDate:      "2026-10-01",
		TargetDate:     "2026-11-30",
		TotalTasks:     9,
		CompletedTasks: 4,
	}

	s.tasks["TSK-101"] = &projectv1.UTMSTask{
		Id:                "TSK-101",
		ProjectId:         "PRJ-001",
		Title:             "Deploy Automated Bank Reconciliation Stubs",
		Description:       "Configure OFX/CSV statement parsers and verify ledger tolerance thresholds",
		BusinessUnit:      "CONSULTING",
		AssignedTo:        "Arun Kumar (Principal Architect)",
		Priority:          "HIGH",
		Status:            "IN_PROGRESS",
		DueDate:           "2026-10-25",
		SlaHoursRemaining: 36,
		EntityResource:    "project",
		EntityId:          "PRJ-001",
		EstimatedHours:    16,
		ActualHours:       10,
	}

	s.tasks["TSK-102"] = &projectv1.UTMSTask{
		Id:                "TSK-102",
		ProjectId:         "PRJ-001",
		Title:             "Client Review of 3-Way Procurement Workflow",
		Description:       "Present automated Purchase Order vs GRN vs Vendor Bill reconciliation bench",
		BusinessUnit:      "FINANCE",
		AssignedTo:        "Priya Sundaram (Financial Controller)",
		Priority:          "URGENT",
		Status:            "TODO",
		DueDate:           "2026-10-23",
		SlaHoursRemaining: 12,
		EntityResource:    "procurement",
		EntityId:          "PO-2026-0001",
		EstimatedHours:    4,
		ActualHours:       2,
	}

	s.tasks["TSK-103"] = &projectv1.UTMSTask{
		Id:                "TSK-103",
		ProjectId:         "PRJ-002",
		Title:             "Configure LiveKit Telehealth Audio/Video Rooms",
		Description:       "Test WebRTC latency and secure tokens for doctor-patient consultations",
		BusinessUnit:      "HEALTH",
		AssignedTo:        "Karthik S (DevOps Lead)",
		Priority:          "HIGH",
		Status:            "REVIEW",
		DueDate:           "2026-10-24",
		SlaHoursRemaining: 20,
		EntityResource:    "telehealth",
		EntityId:          "ROOM-APOLLO-01",
		EstimatedHours:    12,
		ActualHours:       11,
	}

	s.milestones["PRJ-001"] = []*projectv1.Milestone{
		{
			Id:                   "MS-01",
			ProjectId:            "PRJ-001",
			Title:                "Architecture Sign-off & Database Migration",
			TargetDate:           "2026-09-30",
			InvoiceTriggerAmount: 400000,
			Status:               "ACHIEVED",
		},
		{
			Id:                   "MS-02",
			ProjectId:            "PRJ-001",
			Title:                "Core Accounting & Bank Feeds Integration",
			TargetDate:           "2026-10-31",
			InvoiceTriggerAmount: 450000,
			Status:               "PENDING",
		},
	}
}

func (s *ProjectServiceServer) ListProjects(ctx context.Context, req *connect.Request[projectv1.ListProjectsRequest]) (*connect.Response[projectv1.ListProjectsResponse], error) {
	if s.db != nil {
		var swProjects []consulting.SoftwareProject
		if err := s.db.WithContext(ctx).Preload("Client").Find(&swProjects).Error; err == nil && len(swProjects) > 0 {
			var list []*projectv1.Project
			for _, sp := range swProjects {
				clientName := "Corporate Client"
				if sp.Client != nil && sp.Client.CompanyName != "" {
					clientName = sp.Client.CompanyName
				}
				startDate := ""
				if sp.StartDate != nil {
					startDate = sp.StartDate.Format("2006-01-02")
				}
				targetDate := ""
				if sp.TargetDate != nil {
					targetDate = sp.TargetDate.Format("2006-01-02")
				}
				list = append(list, &projectv1.Project{
					Id:           strconv.FormatUint(uint64(sp.ID), 10),
					Code:         sp.Code,
					Name:         sp.Title,
					Description:  sp.Title,
					BusinessUnit: "CONSULTING",
					ClientName:   clientName,
					Status:       sp.Status,
					Budget:       sp.Budget,
					Spent:        sp.Spent,
					StartDate:    startDate,
					TargetDate:   targetDate,
				})
			}
			return connect.NewResponse(&projectv1.ListProjectsResponse{Projects: list}), nil
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*projectv1.Project
	for _, p := range s.projects {
		if req.Msg.BusinessUnit != "" && p.BusinessUnit != req.Msg.BusinessUnit {
			continue
		}
		if req.Msg.Status != "" && p.Status != req.Msg.Status {
			continue
		}
		list = append(list, p)
	}

	return connect.NewResponse(&projectv1.ListProjectsResponse{Projects: list}), nil
}

func (s *ProjectServiceServer) CreateProject(ctx context.Context, req *connect.Request[projectv1.CreateProjectRequest]) (*connect.Response[projectv1.CreateProjectResponse], error) {
	id := fmt.Sprintf("PRJ-%03d", len(s.projects)+1)
	code := fmt.Sprintf("PRJ-%d", time.Now().Unix()%10000)

	var targetTime *time.Time
	if t, err := time.Parse("2006-01-02", req.Msg.TargetDate); err == nil {
		targetTime = &t
	}
	startTime := time.Now()

	if s.db != nil {
		sp := &consulting.SoftwareProject{
			OrganizationID: 1,
			BusinessUnitID: 5,
			Title:          req.Msg.Name,
			Code:           code,
			Budget:         req.Msg.Budget,
			Spent:          0,
			Status:         "PLANNING",
			StartDate:      &startTime,
			TargetDate:     targetTime,
		}
		s.db.WithContext(ctx).Create(sp)
		id = strconv.FormatUint(uint64(sp.ID), 10)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	p := &projectv1.Project{
		Id:             id,
		Code:           code,
		Name:           req.Msg.Name,
		Description:    req.Msg.Description,
		BusinessUnit:   req.Msg.BusinessUnit,
		ClientName:     req.Msg.ClientName,
		Status:         "PLANNING",
		Budget:         req.Msg.Budget,
		Spent:          0,
		StartDate:      time.Now().Format("2006-01-02"),
		TargetDate:     req.Msg.TargetDate,
		TotalTasks:     0,
		CompletedTasks: 0,
	}
	s.projects[id] = p

	return connect.NewResponse(&projectv1.CreateProjectResponse{Project: p}), nil
}

func (s *ProjectServiceServer) GetProject(ctx context.Context, req *connect.Request[projectv1.GetProjectRequest]) (*connect.Response[projectv1.GetProjectResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.projects[req.Msg.Id]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("project %s not found", req.Msg.Id))
	}

	var pTasks []*projectv1.UTMSTask
	for _, t := range s.tasks {
		if t.ProjectId == req.Msg.Id {
			pTasks = append(pTasks, t)
		}
	}

	return connect.NewResponse(&projectv1.GetProjectResponse{
		Project:    p,
		Tasks:      pTasks,
		Milestones: s.milestones[req.Msg.Id],
	}), nil
}

func (s *ProjectServiceServer) ListTasks(ctx context.Context, req *connect.Request[projectv1.ListTasksRequest]) (*connect.Response[projectv1.ListTasksResponse], error) {
	if s.db != nil {
		var dbTasks []task.Task
		q := s.db.WithContext(ctx).Order("created_at DESC")
		if req.Msg.Status != "" {
			q = q.Where("status = ?", req.Msg.Status)
		}
		if err := q.Find(&dbTasks).Error; err == nil && len(dbTasks) > 0 {
			var list []*projectv1.UTMSTask
			for _, t := range dbTasks {
				dueDate := ""
				if t.DueDate != nil {
					dueDate = t.DueDate.Format("2006-01-02")
				}
				projID := ""
				if t.EntityID != nil {
					projID = strconv.FormatUint(uint64(*t.EntityID), 10)
				}
				list = append(list, &projectv1.UTMSTask{
					Id:             strconv.FormatUint(uint64(t.ID), 10),
					ProjectId:      projID,
					Title:          t.Title,
					Description:    t.Description,
					BusinessUnit:   "CONSULTING",
					AssignedTo:     t.Team,
					Priority:       t.Priority,
					Status:         t.Status,
					DueDate:        dueDate,
					EntityResource: t.EntityResource,
				})
			}
			return connect.NewResponse(&projectv1.ListTasksResponse{Tasks: list}), nil
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*projectv1.UTMSTask
	for _, t := range s.tasks {
		if req.Msg.ProjectId != "" && t.ProjectId != req.Msg.ProjectId {
			continue
		}
		if req.Msg.BusinessUnit != "" && t.BusinessUnit != req.Msg.BusinessUnit {
			continue
		}
		if req.Msg.Status != "" && t.Status != req.Msg.Status {
			continue
		}
		list = append(list, t)
	}

	return connect.NewResponse(&projectv1.ListTasksResponse{Tasks: list}), nil
}

func (s *ProjectServiceServer) CreateTask(ctx context.Context, req *connect.Request[projectv1.CreateTaskRequest]) (*connect.Response[projectv1.CreateTaskResponse], error) {
	id := fmt.Sprintf("TSK-%03d", len(s.tasks)+1)
	if s.db != nil {
		var dueTime *time.Time
		if t, err := time.Parse("2006-01-02", req.Msg.DueDate); err == nil {
			dueTime = &t
		}
		dbTask := &task.Task{
			OrganizationID: 1,
			BusinessUnitID: 5,
			Title:          req.Msg.Title,
			Description:    req.Msg.Description,
			Status:         "TODO",
			Priority:       req.Msg.Priority,
			DueDate:        dueTime,
			EntityResource: req.Msg.EntityResource,
			Team:           req.Msg.AssignedTo,
		}
		s.db.WithContext(ctx).Create(dbTask)
		id = strconv.FormatUint(uint64(dbTask.ID), 10)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	t := &projectv1.UTMSTask{
		Id:                id,
		ProjectId:         req.Msg.ProjectId,
		Title:             req.Msg.Title,
		Description:       req.Msg.Description,
		BusinessUnit:      req.Msg.BusinessUnit,
		AssignedTo:        req.Msg.AssignedTo,
		Priority:          req.Msg.Priority,
		Status:            "TODO",
		DueDate:           req.Msg.DueDate,
		SlaHoursRemaining: 48,
		EntityResource:    req.Msg.EntityResource,
		EntityId:          req.Msg.EntityId,
		EstimatedHours:    req.Msg.EstimatedHours,
		ActualHours:       0,
	}
	s.tasks[id] = t

	// Update project counter if attached
	if p, ok := s.projects[req.Msg.ProjectId]; ok {
		p.TotalTasks++
	}

	return connect.NewResponse(&projectv1.CreateTaskResponse{Task: t}), nil
}

func (s *ProjectServiceServer) UpdateTaskStatus(ctx context.Context, req *connect.Request[projectv1.UpdateTaskStatusRequest]) (*connect.Response[projectv1.UpdateTaskStatusResponse], error) {
	if s.db != nil {
		s.db.WithContext(ctx).Model(&task.Task{}).Where("id = ?", req.Msg.TaskId).Update("status", req.Msg.Status)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[req.Msg.TaskId]
	if !ok {
		return connect.NewResponse(&projectv1.UpdateTaskStatusResponse{
			Task: &projectv1.UTMSTask{
				Id:     req.Msg.TaskId,
				Status: req.Msg.Status,
			},
		}), nil
	}
	t.Status = req.Msg.Status

	if req.Msg.Status == "DONE" {
		if p, ok := s.projects[t.ProjectId]; ok {
			p.CompletedTasks++
		}
	}

	return connect.NewResponse(&projectv1.UpdateTaskStatusResponse{Task: t}), nil
}

func (s *ProjectServiceServer) DelegateTask(ctx context.Context, req *connect.Request[projectv1.DelegateTaskRequest]) (*connect.Response[projectv1.DelegateTaskResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[req.Msg.TaskId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("task %s not found", req.Msg.TaskId))
	}

	t.BusinessUnit = req.Msg.TargetBusinessUnit
	t.AssignedTo = req.Msg.NewAssignee
	t.Description = fmt.Sprintf("%s\n\n[Delegation Note]: %s", t.Description, req.Msg.DelegationNote)

	return connect.NewResponse(&projectv1.DelegateTaskResponse{Task: t}), nil
}

func (s *ProjectServiceServer) ListMilestones(ctx context.Context, req *connect.Request[projectv1.ListMilestonesRequest]) (*connect.Response[projectv1.ListMilestonesResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return connect.NewResponse(&projectv1.ListMilestonesResponse{Milestones: s.milestones[req.Msg.ProjectId]}), nil
}

func (s *ProjectServiceServer) CreateMilestone(ctx context.Context, req *connect.Request[projectv1.CreateMilestoneRequest]) (*connect.Response[projectv1.CreateMilestoneResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m := &projectv1.Milestone{
		Id:                   fmt.Sprintf("MS-%02d", len(s.milestones[req.Msg.ProjectId])+1),
		ProjectId:            req.Msg.ProjectId,
		Title:                req.Msg.Title,
		TargetDate:           req.Msg.TargetDate,
		InvoiceTriggerAmount: req.Msg.InvoiceTriggerAmount,
		Status:               "PENDING",
	}

	s.milestones[req.Msg.ProjectId] = append(s.milestones[req.Msg.ProjectId], m)
	return connect.NewResponse(&projectv1.CreateMilestoneResponse{Milestone: m}), nil
}

func (s *ProjectServiceServer) LogTimesheet(ctx context.Context, req *connect.Request[projectv1.LogTimesheetRequest]) (*connect.Response[projectv1.LogTimesheetResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e := &projectv1.TimesheetEntry{
		Id:           fmt.Sprintf("TS-%04d", time.Now().Unix()%10000),
		ProjectId:    req.Msg.ProjectId,
		TaskId:       req.Msg.TaskId,
		EmployeeName: req.Msg.EmployeeName,
		Hours:        req.Msg.Hours,
		WorkDate:     req.Msg.WorkDate,
		Notes:        req.Msg.Notes,
		Billable:     req.Msg.Billable,
	}

	s.timesheets[req.Msg.ProjectId] = append(s.timesheets[req.Msg.ProjectId], e)
	if t, ok := s.tasks[req.Msg.TaskId]; ok {
		t.ActualHours += req.Msg.Hours
	}

	return connect.NewResponse(&projectv1.LogTimesheetResponse{Entry: e}), nil
}

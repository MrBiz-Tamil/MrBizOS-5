package consulting

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	consultingv1 "drayya-core/gen/consulting/v1"
	"drayya-core/gen/consulting/v1/consultingv1connect"
	"gorm.io/gorm"
)

type ConsultingServiceServer struct {
	consultingv1connect.UnimplementedConsultingServiceHandler
	db *gorm.DB
	mu sync.RWMutex

	projects   map[string]*consultingv1.ProjectProto
	timesheets map[string]*consultingv1.TimesheetProto
	contracts  map[string]*consultingv1.SOWContractProto
}

func NewService(db *gorm.DB) *ConsultingServiceServer {
	s := &ConsultingServiceServer{
		db:         db,
		projects:   make(map[string]*consultingv1.ProjectProto),
		timesheets: make(map[string]*consultingv1.TimesheetProto),
		contracts:  make(map[string]*consultingv1.SOWContractProto),
	}
	s.seedDefaults()
	return s
}

func (s *ConsultingServiceServer) seedDefaults() {
	s.projects["SOW-PRJ-01"] = &consultingv1.ProjectProto{
		Id:         "SOW-PRJ-01",
		TenantId:   "tenant-1",
		Name:       "FinTech Cloud Microservices Architecture",
		ClientName: "Axis Financial Holdings",
		StartDate:  "2026-09-15",
		EndDate:    "2026-12-15",
		Budget:     "1500000.00",
		Status:     "ACTIVE",
	}

	s.contracts["SOW-01"] = &consultingv1.SOWContractProto{
		Id:            "SOW-01",
		TenantId:      "tenant-1",
		ProjectId:     "SOW-PRJ-01",
		ClientName:    "Axis Financial Holdings",
		ContractValue: "1500000.00",
		Terms:         "Net 30 milestone billing, 3 full-stack Go engineers and 1 Kubernetes lead",
		Status:        "SIGNED",
	}

	s.timesheets["TS-01"] = &consultingv1.TimesheetProto{
		Id:              "TS-01",
		TenantId:        "tenant-1",
		ProjectId:       "SOW-PRJ-01",
		ConsultantName:  "Arun Kumar (Principal Architect)",
		Date:            "2026-10-20",
		Hours:           8.0,
		TaskDescription: "Migrated ledger to Go ConnectRPC with zero memory leaks",
		IsBillable:      true,
	}
}

func (s *ConsultingServiceServer) ListProjects(ctx context.Context, req *connect.Request[consultingv1.ListProjectsRequest]) (*connect.Response[consultingv1.ListProjectsResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*consultingv1.ProjectProto
	for _, p := range s.projects {
		if req.Msg.Status != "" && p.Status != req.Msg.Status {
			continue
		}
		list = append(list, p)
	}
	return connect.NewResponse(&consultingv1.ListProjectsResponse{Projects: list}), nil
}

func (s *ConsultingServiceServer) GetProject(ctx context.Context, req *connect.Request[consultingv1.GetProjectRequest]) (*connect.Response[consultingv1.GetProjectResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.projects[req.Msg.Id]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("project %s not found", req.Msg.Id))
	}
	return connect.NewResponse(&consultingv1.GetProjectResponse{Project: p}), nil
}

func (s *ConsultingServiceServer) CreateProject(ctx context.Context, req *connect.Request[consultingv1.CreateProjectRequest]) (*connect.Response[consultingv1.CreateProjectResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("SOW-PRJ-%02d", len(s.projects)+1)
	p := &consultingv1.ProjectProto{
		Id:         id,
		TenantId:   "tenant-1",
		Name:       req.Msg.Name,
		ClientName: req.Msg.ClientName,
		StartDate:  req.Msg.StartDate,
		EndDate:    req.Msg.EndDate,
		Budget:     req.Msg.Budget,
		Status:     "PLANNING",
	}
	s.projects[id] = p

	return connect.NewResponse(&consultingv1.CreateProjectResponse{Project: p}), nil
}

func (s *ConsultingServiceServer) LogTimesheet(ctx context.Context, req *connect.Request[consultingv1.LogTimesheetRequest]) (*connect.Response[consultingv1.LogTimesheetResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("TS-%04d", time.Now().Unix()%10000)
	t := &consultingv1.TimesheetProto{
		Id:              id,
		TenantId:        "tenant-1",
		ProjectId:       req.Msg.ProjectId,
		ConsultantName:  "Lead Consultant",
		Date:            req.Msg.Date,
		Hours:           req.Msg.Hours,
		TaskDescription: req.Msg.TaskDescription,
		IsBillable:      req.Msg.IsBillable,
	}
	s.timesheets[id] = t

	return connect.NewResponse(&consultingv1.LogTimesheetResponse{Timesheet: t}), nil
}

func (s *ConsultingServiceServer) ListTimesheets(ctx context.Context, req *connect.Request[consultingv1.ListTimesheetsRequest]) (*connect.Response[consultingv1.ListTimesheetsResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*consultingv1.TimesheetProto
	for _, t := range s.timesheets {
		if req.Msg.ProjectId != "" && t.ProjectId != req.Msg.ProjectId {
			continue
		}
		list = append(list, t)
	}
	return connect.NewResponse(&consultingv1.ListTimesheetsResponse{Timesheets: list}), nil
}

func (s *ConsultingServiceServer) CreateSOWContract(ctx context.Context, req *connect.Request[consultingv1.CreateSOWContractRequest]) (*connect.Response[consultingv1.CreateSOWContractResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("SOW-%02d", len(s.contracts)+1)
	c := &consultingv1.SOWContractProto{
		Id:            id,
		TenantId:      "tenant-1",
		ProjectId:     req.Msg.ProjectId,
		ClientName:    req.Msg.ClientName,
		ContractValue: req.Msg.ContractValue,
		Terms:         req.Msg.Terms,
		Status:        "DRAFT",
	}
	s.contracts[id] = c

	return connect.NewResponse(&consultingv1.CreateSOWContractResponse{Sow: c}), nil
}

func (s *ConsultingServiceServer) ListSOWContracts(ctx context.Context, req *connect.Request[consultingv1.ListSOWContractsRequest]) (*connect.Response[consultingv1.ListSOWContractsResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*consultingv1.SOWContractProto
	for _, c := range s.contracts {
		list = append(list, c)
	}
	return connect.NewResponse(&consultingv1.ListSOWContractsResponse{Sows: list}), nil
}

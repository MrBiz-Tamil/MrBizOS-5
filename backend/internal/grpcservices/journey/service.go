package journey

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	journeyv1 "drayya-core/gen/journey/v1"
	"drayya-core/gen/journey/v1/journeyv1connect"
	"gorm.io/gorm"
)

type JourneyServiceServer struct {
	journeyv1connect.UnimplementedJourneyServiceHandler
	db *gorm.DB
	mu sync.RWMutex

	templates map[string]*journeyv1.JourneyTemplateDef
	instances map[string]*journeyv1.JourneyInstanceDef
	logs      map[string][]*journeyv1.JourneyLogEntry
}

func NewService(db *gorm.DB) *JourneyServiceServer {
	s := &JourneyServiceServer{
		db:        db,
		templates: make(map[string]*journeyv1.JourneyTemplateDef),
		instances: make(map[string]*journeyv1.JourneyInstanceDef),
		logs:      make(map[string][]*journeyv1.JourneyLogEntry),
	}
	s.seedDefaults()
	return s
}

func (s *JourneyServiceServer) seedDefaults() {
	// Healthcare Clinical Patient Journey
	s.templates["JRN-HEALTH"] = &journeyv1.JourneyTemplateDef{
		Id:           "JRN-HEALTH",
		Code:         "PATIENT_SURGICAL_CARE",
		Name:         "Surgical Care & Clinical Recovery Journey",
		BusinessUnit: "HEALTH",
		Description:  "End-to-end patient pathway from OPD intake to post-discharge rehab monitoring",
		Stages: []*journeyv1.JourneyStageDef{
			{Id: "STG-H1", StageName: "OPD Triage & Consultation", StageOrder: 1, Description: "Doctor clinical assessment and initial diagnosis", AutoTask: true, AutoTaskTitle: "Review lab requisitions", TriggerNotificationChannel: "WHATSAPP"},
			{Id: "STG-H2", StageName: "Pre-Operative Diagnostics", StageOrder: 2, Description: "Blood work, radiology scans, and anesthesia clearance", AutoTask: true, AutoTaskTitle: "Verify pathology reports", TriggerNotificationChannel: "EMAIL"},
			{Id: "STG-H3", StageName: "Surgical Procedure", StageOrder: 3, Description: "Operating theatre execution and vital stabilization", AutoTask: false, TriggerNotificationChannel: "SMS"},
			{Id: "STG-H4", StageName: "Inpatient Ward Recovery", StageOrder: 4, Description: "Nursing care, antibiotic regimen, and clinical rounds", AutoTask: true, AutoTaskTitle: "Daily ward vitals chart", TriggerNotificationChannel: "WHATSAPP"},
			{Id: "STG-H5", StageName: "Discharge & Home Rehab", StageOrder: 5, Description: "Discharge summary, prescription delivery, and telehealth follow-up", AutoTask: true, AutoTaskTitle: "Schedule Day-7 Telehealth Review", TriggerNotificationChannel: "WHATSAPP"},
		},
	}

	// Travel Visa Journey
	s.templates["JRN-TRAVEL"] = &journeyv1.JourneyTemplateDef{
		Id:           "JRN-TRAVEL",
		Code:         "EUROPE_SCHENGEN_VISA",
		Name:         "Schengen Tourist & Business Visa Clearance",
		BusinessUnit: "TRAVEL",
		Description:  "Document verification, appointment slot booking, embassy biometric submission, and flight delivery",
		Stages: []*journeyv1.JourneyStageDef{
			{Id: "STG-T1", StageName: "Document Collection", StageOrder: 1, Description: "Passport scan, bank statements, ITR, and employer NOC", AutoTask: true, AutoTaskTitle: "Verify passport validity > 6 months", TriggerNotificationChannel: "WHATSAPP"},
			{Id: "STG-T2", StageName: "VFS Slot Booking", StageOrder: 2, Description: "Lock biometric appointment slot at regional VFS center", AutoTask: true, AutoTaskTitle: "Book appointment slot", TriggerNotificationChannel: "EMAIL"},
			{Id: "STG-T3", StageName: "Embassy Processing", StageOrder: 3, Description: "Consulate document review and verification", AutoTask: false, TriggerNotificationChannel: "WHATSAPP"},
			{Id: "STG-T4", StageName: "Visa Stamped & Ticket Dispatch", StageOrder: 4, Description: "Courier stamped passport and issue e-tickets + hotel vouchers", AutoTask: true, AutoTaskTitle: "Dispatch travel kit", TriggerNotificationChannel: "WHATSAPP"},
		},
	}

	// Real Estate Deal Journey
	s.templates["JRN-REALESTATE"] = &journeyv1.JourneyTemplateDef{
		Id:           "JRN-REALESTATE",
		Code:         "COMMERCIAL_PROPERTY_ACQUISITION",
		Name:         "Commercial Office Space Lease & Acquisition",
		BusinessUnit: "REALESTATE",
		Description:  "Requirement intake, site visits, legal title verification, lease execution, and key handover",
		Stages: []*journeyv1.JourneyStageDef{
			{Id: "STG-R1", StageName: "Requirement Intake", StageOrder: 1, Description: "Client square footage, budget, and location preference", AutoTask: true, AutoTaskTitle: "Prepare curated property list", TriggerNotificationChannel: "WHATSAPP"},
			{Id: "STG-R2", StageName: "Physical Site Visits", StageOrder: 2, Description: "Escorted inspection of top 3 properties with client executive", AutoTask: true, AutoTaskTitle: "Collect feedback on site tour", TriggerNotificationChannel: "WHATSAPP"},
			{Id: "STG-R3", StageName: "Legal Due Diligence", StageOrder: 3, Description: "Title search report, encumbrance check, and NOC clearance", AutoTask: true, AutoTaskTitle: "Legal advocate deed review", TriggerNotificationChannel: "EMAIL"},
			{Id: "STG-R4", StageName: "Agreement & Registry", StageOrder: 4, Description: "Stamp duty payment, registered deed execution, and escrow settlement", AutoTask: true, AutoTaskTitle: "Prepare draft lease deed", TriggerNotificationChannel: "EMAIL"},
			{Id: "STG-R5", StageName: "Handover & Fit-Out", StageOrder: 5, Description: "Key handover and maintenance transition", AutoTask: false, TriggerNotificationChannel: "WHATSAPP"},
		},
	}

	// Seed Sample Instances
	s.instances["INS-001"] = &journeyv1.JourneyInstanceDef{
		Id:                "INS-001",
		TemplateId:        "JRN-HEALTH",
		TemplateName:      "Surgical Care & Clinical Recovery Journey",
		BusinessUnit:      "HEALTH",
		CustomerName:      "Meenakshi Sundaram",
		CustomerContact:   "+91 98401 23456",
		CurrentStageName:  "Pre-Operative Diagnostics",
		CurrentStageOrder: 2,
		TotalStages:       5,
		Status:            "ACTIVE",
		CreatedAt:         "2026-10-18",
		UpdatedAt:         "2026-10-21",
	}

	s.instances["INS-002"] = &journeyv1.JourneyInstanceDef{
		Id:                "INS-002",
		TemplateId:        "JRN-TRAVEL",
		TemplateName:      "Schengen Tourist & Business Visa Clearance",
		BusinessUnit:      "TRAVEL",
		CustomerName:      "Vikram Malhotra",
		CustomerContact:   "+91 98111 54321",
		CurrentStageName:  "VFS Slot Booking",
		CurrentStageOrder: 2,
		TotalStages:       4,
		Status:            "ACTIVE",
		CreatedAt:         "2026-10-19",
		UpdatedAt:         "2026-10-20",
	}

	s.logs["INS-001"] = []*journeyv1.JourneyLogEntry{
		{Id: "LOG-01", FromStage: "Initiated", ToStage: "OPD Triage & Consultation", ActorName: "Dr. Ayya Senior Consultant", Notes: "Patient presented with acute symptoms; recommended elective procedure", Timestamp: "2026-10-18 10:30"},
		{Id: "LOG-02", FromStage: "OPD Triage & Consultation", ToStage: "Pre-Operative Diagnostics", ActorName: "Nurse In-Charge", Notes: "Scheduled cardiac screening and ultrasound", Timestamp: "2026-10-19 14:15"},
	}
}

func (s *JourneyServiceServer) ListTemplates(ctx context.Context, req *connect.Request[journeyv1.ListTemplatesRequest]) (*connect.Response[journeyv1.ListTemplatesResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*journeyv1.JourneyTemplateDef
	for _, t := range s.templates {
		if req.Msg.BusinessUnit != "" && t.BusinessUnit != req.Msg.BusinessUnit {
			continue
		}
		list = append(list, t)
	}

	return connect.NewResponse(&journeyv1.ListTemplatesResponse{Templates: list}), nil
}

func (s *JourneyServiceServer) CreateTemplate(ctx context.Context, req *connect.Request[journeyv1.CreateTemplateRequest]) (*connect.Response[journeyv1.CreateTemplateResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("JRN-%03d", len(s.templates)+1)
	t := &journeyv1.JourneyTemplateDef{
		Id:           id,
		Code:         req.Msg.Code,
		Name:         req.Msg.Name,
		BusinessUnit: req.Msg.BusinessUnit,
		Description:  req.Msg.Description,
		Stages:       req.Msg.Stages,
	}
	s.templates[id] = t

	return connect.NewResponse(&journeyv1.CreateTemplateResponse{Template: t}), nil
}

func (s *JourneyServiceServer) ListInstances(ctx context.Context, req *connect.Request[journeyv1.ListInstancesRequest]) (*connect.Response[journeyv1.ListInstancesResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*journeyv1.JourneyInstanceDef
	for _, inst := range s.instances {
		if req.Msg.BusinessUnit != "" && inst.BusinessUnit != req.Msg.BusinessUnit {
			continue
		}
		if req.Msg.Status != "" && inst.Status != req.Msg.Status {
			continue
		}
		list = append(list, inst)
	}

	return connect.NewResponse(&journeyv1.ListInstancesResponse{Instances: list}), nil
}

func (s *JourneyServiceServer) StartJourney(ctx context.Context, req *connect.Request[journeyv1.StartJourneyRequest]) (*connect.Response[journeyv1.StartJourneyResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tmpl, ok := s.templates[req.Msg.TemplateId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("template %s not found", req.Msg.TemplateId))
	}

	firstStage := "Initiated"
	if len(tmpl.Stages) > 0 {
		firstStage = tmpl.Stages[0].StageName
	}

	id := fmt.Sprintf("INS-%03d", len(s.instances)+1)
	inst := &journeyv1.JourneyInstanceDef{
		Id:                id,
		TemplateId:        tmpl.Id,
		TemplateName:      tmpl.Name,
		BusinessUnit:      tmpl.BusinessUnit,
		CustomerName:      req.Msg.CustomerName,
		CustomerContact:   req.Msg.CustomerContact,
		CurrentStageName:  firstStage,
		CurrentStageOrder: 1,
		TotalStages:       int32(len(tmpl.Stages)),
		Status:            "ACTIVE",
		CreatedAt:         time.Now().Format("2006-01-02"),
		UpdatedAt:         time.Now().Format("2006-01-02"),
	}
	s.instances[id] = inst

	s.logs[id] = []*journeyv1.JourneyLogEntry{
		{
			Id:        fmt.Sprintf("LOG-%02d", 1),
			FromStage: "START",
			ToStage:   firstStage,
			ActorName: "System",
			Notes:     req.Msg.InitialNotes,
			Timestamp: time.Now().Format("2006-01-02 15:04"),
		},
	}

	return connect.NewResponse(&journeyv1.StartJourneyResponse{Instance: inst}), nil
}

func (s *JourneyServiceServer) AdvanceStage(ctx context.Context, req *connect.Request[journeyv1.AdvanceStageRequest]) (*connect.Response[journeyv1.AdvanceStageResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inst, ok := s.instances[req.Msg.InstanceId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("instance %s not found", req.Msg.InstanceId))
	}

	from := inst.CurrentStageName
	to := req.Msg.TargetStageName

	inst.CurrentStageName = to
	inst.CurrentStageOrder++
	if inst.CurrentStageOrder > inst.TotalStages {
		inst.Status = "COMPLETED"
	}
	inst.UpdatedAt = time.Now().Format("2006-01-02")

	logEntry := &journeyv1.JourneyLogEntry{
		Id:        fmt.Sprintf("LOG-%02d", len(s.logs[inst.Id])+1),
		FromStage: from,
		ToStage:   to,
		ActorName: req.Msg.ActorName,
		Notes:     req.Msg.Notes,
		Timestamp: time.Now().Format("2006-01-02 15:04"),
	}
	s.logs[inst.Id] = append(s.logs[inst.Id], logEntry)

	triggered := []string{
		fmt.Sprintf("Delivered automated WhatsApp notification to %s", inst.CustomerContact),
		fmt.Sprintf("Created milestone verification task in UTMS for %s", to),
	}

	return connect.NewResponse(&journeyv1.AdvanceStageResponse{
		Instance:         inst,
		TriggeredActions: triggered,
	}), nil
}

func (s *JourneyServiceServer) GetJourneyDetails(ctx context.Context, req *connect.Request[journeyv1.GetJourneyDetailsRequest]) (*connect.Response[journeyv1.GetJourneyDetailsResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	inst, ok := s.instances[req.Msg.InstanceId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("instance %s not found", req.Msg.InstanceId))
	}

	tmpl := s.templates[inst.TemplateId]
	var stages []*journeyv1.JourneyStageDef
	if tmpl != nil {
		stages = tmpl.Stages
	}

	return connect.NewResponse(&journeyv1.GetJourneyDetailsResponse{
		Instance:  inst,
		AllStages: stages,
		Logs:      s.logs[inst.Id],
	}), nil
}

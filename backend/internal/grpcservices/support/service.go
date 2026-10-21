package support

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	supportv1 "drayya-core/gen/support/v1"
	"drayya-core/gen/support/v1/supportv1connect"
	"gorm.io/gorm"
)

type SupportServiceServer struct {
	supportv1connect.UnimplementedSupportServiceHandler
	db *gorm.DB
	mu sync.RWMutex

	tickets  map[string]*supportv1.TicketItem
	messages map[string][]*supportv1.TicketMessageItem
}

func NewService(db *gorm.DB) *SupportServiceServer {
	s := &SupportServiceServer{
		db:       db,
		tickets:  make(map[string]*supportv1.TicketItem),
		messages: make(map[string][]*supportv1.TicketMessageItem),
	}
	s.seedDefaults()
	return s
}

func (s *SupportServiceServer) seedDefaults() {
	s.tickets["TIK-001"] = &supportv1.TicketItem{
		Id:                  "TIK-001",
		TicketNumber:        "TIK-2026-0891",
		CustomerName:        "Dr. Ayya Clinic (Chennai Branch)",
		CustomerContact:     "+91 98401 99887",
		Subject:             "HL7 Lab Analyzer bridge connection timeout",
		Category:            "TECHNICAL",
		Priority:            "URGENT",
		Status:              "IN_PROGRESS",
		Channel:             "EMAIL",
		AssignedTo:          "Karthik S (DevOps Lead)",
		BusinessUnit:        "HEALTH",
		SlaMinutesRemaining: 42,
		IsSlaBreached:       false,
		CreatedAt:           "2026-10-21 09:30",
	}

	s.messages["TIK-001"] = []*supportv1.TicketMessageItem{
		{
			Id:         "MSG-01",
			TicketId:   "TIK-001",
			SenderName: "Chennai Branch Admin",
			SenderType: "CUSTOMER",
			Message:    "Our Mindray BC-5000 blood analyzer is returning socket error 10060 when syncing CBC reports to EMR.",
			SentAt:     "09:30 AM",
		},
		{
			Id:         "MSG-02",
			TicketId:   "TIK-001",
			SenderName: "Karthik S",
			SenderType: "AGENT",
			Message:    "Investigating the local gateway firewall. Re-routing HL7 MLLP port 2575 through the local proxy.",
			SentAt:     "09:45 AM",
		},
	}

	s.tickets["TIK-002"] = &supportv1.TicketItem{
		Id:                  "TIK-002",
		TicketNumber:        "TIK-2026-0892",
		CustomerName:        "Global Horizon Logistics",
		CustomerContact:     "ops@globalhorizon.com",
		Subject:             "GST E-Way bill generation failure on export shipment",
		Category:            "BILLING",
		Priority:            "HIGH",
		Status:              "NEW",
		Channel:             "WHATSAPP",
		AssignedTo:          "Priya (Finance)",
		BusinessUnit:        "CONSULTING",
		SlaMinutesRemaining: 110,
		IsSlaBreached:       false,
		CreatedAt:           "2026-10-21 11:15",
	}

	s.messages["TIK-002"] = []*supportv1.TicketMessageItem{
		{
			Id:         "MSG-03",
			TicketId:   "TIK-002",
			SenderName: "Ops Lead",
			SenderType: "CUSTOMER",
			Message:    "We are receiving error 248: Pin-to-Pin distance exceeds 4000km on E-Way portal for container MB-990.",
			SentAt:     "11:15 AM",
		},
	}

	s.tickets["TIK-003"] = &supportv1.TicketItem{
		Id:                  "TIK-003",
		TicketNumber:        "TIK-2026-0893",
		CustomerName:        "Vikram Malhotra",
		CustomerContact:     "+91 98111 54321",
		Subject:             "Urgent VFS Biometric slot confirmation for France Visa",
		Category:            "GENERAL",
		Priority:            "HIGH",
		Status:              "WAITING_CUSTOMER",
		Channel:             "VOICE_BOT",
		AssignedTo:          "Travel Desk Lead",
		BusinessUnit:        "TRAVEL",
		SlaMinutesRemaining: 180,
		IsSlaBreached:       false,
		CreatedAt:           "2026-10-21 10:00",
	}
}

func (s *SupportServiceServer) ListTickets(ctx context.Context, req *connect.Request[supportv1.ListTicketsRequest]) (*connect.Response[supportv1.ListTicketsResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*supportv1.TicketItem
	totalOpen := int32(0)
	totalBreached := int32(0)

	for _, t := range s.tickets {
		if t.Status != "RESOLVED" && t.Status != "CLOSED" {
			totalOpen++
		}
		if t.IsSlaBreached {
			totalBreached++
		}
		if req.Msg.Status != "" && t.Status != req.Msg.Status {
			continue
		}
		if req.Msg.Priority != "" && t.Priority != req.Msg.Priority {
			continue
		}
		if req.Msg.BusinessUnit != "" && t.BusinessUnit != req.Msg.BusinessUnit {
			continue
		}
		list = append(list, t)
	}

	return connect.NewResponse(&supportv1.ListTicketsResponse{
		Tickets:            list,
		TotalOpen:          totalOpen,
		TotalBreachedSla:   totalBreached,
		AvgResolutionHours: 2.4,
	}), nil
}

func (s *SupportServiceServer) CreateTicket(ctx context.Context, req *connect.Request[supportv1.CreateTicketRequest]) (*connect.Response[supportv1.CreateTicketResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("TIK-%03d", len(s.tickets)+1)
	num := fmt.Sprintf("TIK-2026-%04d", time.Now().Unix()%10000)

	t := &supportv1.TicketItem{
		Id:                  id,
		TicketNumber:        num,
		CustomerName:        req.Msg.CustomerName,
		CustomerContact:     req.Msg.CustomerContact,
		Subject:             req.Msg.Subject,
		Category:            req.Msg.Category,
		Priority:            req.Msg.Priority,
		Status:              "NEW",
		Channel:             req.Msg.Channel,
		AssignedTo:          "Support Pool (Auto-Triage)",
		BusinessUnit:        req.Msg.BusinessUnit,
		SlaMinutesRemaining: 120,
		IsSlaBreached:       false,
		CreatedAt:           time.Now().Format("2006-01-02 15:04"),
	}
	s.tickets[id] = t

	s.messages[id] = []*supportv1.TicketMessageItem{
		{
			Id:         fmt.Sprintf("MSG-%d", time.Now().Unix()%1000),
			TicketId:   id,
			SenderName: req.Msg.CustomerName,
			SenderType: "CUSTOMER",
			Message:    req.Msg.Description,
			SentAt:     time.Now().Format("03:04 PM"),
		},
	}

	return connect.NewResponse(&supportv1.CreateTicketResponse{Ticket: t}), nil
}

func (s *SupportServiceServer) GetTicketDetails(ctx context.Context, req *connect.Request[supportv1.GetTicketDetailsRequest]) (*connect.Response[supportv1.GetTicketDetailsResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tickets[req.Msg.TicketId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("ticket %s not found", req.Msg.TicketId))
	}

	return connect.NewResponse(&supportv1.GetTicketDetailsResponse{
		Ticket:   t,
		Messages: s.messages[t.Id],
	}), nil
}

func (s *SupportServiceServer) ReplyTicket(ctx context.Context, req *connect.Request[supportv1.ReplyTicketRequest]) (*connect.Response[supportv1.ReplyTicketResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tickets[req.Msg.TicketId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("ticket %s not found", req.Msg.TicketId))
	}

	m := &supportv1.TicketMessageItem{
		Id:         fmt.Sprintf("MSG-%04d", time.Now().Unix()%10000),
		TicketId:   t.Id,
		SenderName: req.Msg.SenderName,
		SenderType: "AGENT",
		Message:    req.Msg.Message,
		SentAt:     time.Now().Format("03:04 PM"),
	}
	s.messages[t.Id] = append(s.messages[t.Id], m)

	if req.Msg.CloseTicket {
		t.Status = "RESOLVED"
	} else if t.Status == "NEW" {
		t.Status = "IN_PROGRESS"
	}

	return connect.NewResponse(&supportv1.ReplyTicketResponse{Message: m}), nil
}

func (s *SupportServiceServer) UpdateTicketStatus(ctx context.Context, req *connect.Request[supportv1.UpdateTicketStatusRequest]) (*connect.Response[supportv1.UpdateTicketStatusResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tickets[req.Msg.TicketId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("ticket %s not found", req.Msg.TicketId))
	}
	t.Status = req.Msg.Status

	return connect.NewResponse(&supportv1.UpdateTicketStatusResponse{Ticket: t}), nil
}

func (s *SupportServiceServer) AssignTicket(ctx context.Context, req *connect.Request[supportv1.AssignTicketRequest]) (*connect.Response[supportv1.AssignTicketResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tickets[req.Msg.TicketId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("ticket %s not found", req.Msg.TicketId))
	}
	t.AssignedTo = req.Msg.AssignedTo
	t.Status = "ASSIGNED"

	return connect.NewResponse(&supportv1.AssignTicketResponse{Ticket: t}), nil
}

func (s *SupportServiceServer) SubmitCSAT(ctx context.Context, req *connect.Request[supportv1.SubmitCSATRequest]) (*connect.Response[supportv1.SubmitCSATResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if t, ok := s.tickets[req.Msg.TicketId]; ok {
		t.Status = "CLOSED"
	}
	return connect.NewResponse(&supportv1.SubmitCSATResponse{Success: true}), nil
}

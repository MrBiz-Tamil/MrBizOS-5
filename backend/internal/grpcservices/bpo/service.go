package bpo

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	bpov1 "drayya-core/gen/bpo/v1"
	"drayya-core/gen/bpo/v1/bpov1connect"
	"gorm.io/gorm"
)

type VoiceBPOServiceServer struct {
	bpov1connect.UnimplementedVoiceBPOServiceHandler
	db *gorm.DB
	mu sync.RWMutex

	calls map[string]*bpov1.CallSession
}

func NewService(db *gorm.DB) *VoiceBPOServiceServer {
	s := &VoiceBPOServiceServer{
		db:    db,
		calls: make(map[string]*bpov1.CallSession),
	}
	s.seedDefaults()
	return s
}

func (s *VoiceBPOServiceServer) seedDefaults() {
	s.calls["CALL-901"] = &bpov1.CallSession{
		CallId:                   "CALL-901",
		CallerPhone:              "+91 98450 11223",
		CustomerName:             "Rajesh Kumar",
		Status:                   "AI_HANDLING",
		Intent:                   "SALES_INQUIRY",
		AssignedAgent:            "Autonomous AI Voice Bot",
		TargetDepartment:         "REALESTATE",
		LivekitRoomToken:         "mock_token_lk_room_901",
		LiveTranscript:           "[Bot]: Hello Rajesh! Thanks for calling MrBiz-OS Real Estate desk.\n[Caller]: Hi, I visited the Guindy tech park office yesterday. What is the final lease price?\n[Bot]: The Guindy commercial space is 4,200 sqft at ₹65/sqft all-inclusive. Would you like me to connect you with Divya from our sales team?",
		AiSummary:                "Customer inquiring about Guindy 4,200 sqft tech park office lease pricing. Expressed ready purchase intent; recommended warm transfer to Divya (Sales Lead).",
		CallDurationSeconds:      45,
		CustomerKhataBalance:     "₹0.00 (Active Prospect)",
		ActiveDealOrAppointment:  "Property Visit #PV-2026-88",
	}

	s.calls["CALL-902"] = &bpov1.CallSession{
		CallId:                   "CALL-902",
		CallerPhone:              "+91 98401 23456",
		CustomerName:             "Meenakshi Sundaram",
		Status:                   "QUEUED_FOR_AGENT",
		Intent:                   "MEDICAL_TRIAGE",
		AssignedAgent:            "Connecting to Duty Doctor...",
		TargetDepartment:         "CLINICAL",
		LivekitRoomToken:         "mock_token_lk_room_902",
		LiveTranscript:           "[Bot]: Namaste Meenakshi! Calling about your cardiology consultation?\n[Caller]: Yes, my father has slight chest heaviness after his evening walk.\n[Bot]: Please have him sit down and remain calm. Transferring your audio directly to our Senior Clinical Consultant right now.",
		AiSummary:                "EMERGENCY/URGENT: Patient's father experiencing chest heaviness post-exertion. Priority 1 clinical triage transfer initiated to Dr. Ayya consultant.",
		CallDurationSeconds:      30,
		CustomerKhataBalance:     "₹1,500 Consultation Paid",
		ActiveDealOrAppointment:  "Cardiology Review #APP-2026-41",
	}

	s.calls["CALL-903"] = &bpov1.CallSession{
		CallId:                   "CALL-903",
		CallerPhone:              "+91 98111 54321",
		CustomerName:             "Vikram Malhotra",
		Status:                   "IN_CALL_WITH_AGENT",
		Intent:                   "SUPPORT_COMPLAINT",
		AssignedAgent:            "Arun (Travel Desk)",
		TargetDepartment:         "TRAVEL",
		LivekitRoomToken:         "mock_token_lk_room_903",
		LiveTranscript:           "[Bot]: Verified caller Vikram Malhotra.\n[Caller]: I need my France visa appointment letter urgently.\n[Bot]: Transferring to Arun with your visa dossier open.\n[Arun]: Hi Vikram! I have your VFS receipt on my screen. Dispatching the PDF now.",
		AiSummary:                "Warm transfer completed. Arun resolving VFS appointment letter dispatch for Schengen visa.",
		CallDurationSeconds:      125,
		CustomerKhataBalance:     "₹24,000 Package Advance",
		ActiveDealOrAppointment:  "Journey #INS-002 (VFS Slot)",
	}
}

func (s *VoiceBPOServiceServer) StartInboundCallSession(ctx context.Context, req *connect.Request[bpov1.StartCallRequest]) (*connect.Response[bpov1.StartCallResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	callId := fmt.Sprintf("CALL-%03d", len(s.calls)+1)
	greeting := fmt.Sprintf("Hello! Welcome to MrBiz-OS Enterprise Desk. How can our AI voice agent assist you today?")

	session := &bpov1.CallSession{
		CallId:                  callId,
		CallerPhone:             req.Msg.CallerPhone,
		CustomerName:            "Caller (" + req.Msg.CallerPhone + ")",
		Status:                  "AI_HANDLING",
		Intent:                  "GENERAL_INQUIRY",
		AssignedAgent:           "MrBiz Autonomous Voice Bot",
		TargetDepartment:        "SUPPORT",
		LivekitRoomToken:        fmt.Sprintf("lk_token_%s", callId),
		LiveTranscript:          "[Bot]: " + greeting,
		AiSummary:               "Call initiated from " + req.Msg.CallerPhone,
		CallDurationSeconds:     1,
		CustomerKhataBalance:    "₹0.00",
		ActiveDealOrAppointment: "None",
	}
	s.calls[callId] = session

	return connect.NewResponse(&bpov1.StartCallResponse{
		Session:         session,
		InitialGreeting: greeting,
	}), nil
}

func (s *VoiceBPOServiceServer) ProcessVoiceTurn(ctx context.Context, req *connect.Request[bpov1.VoiceTurnRequest]) (*connect.Response[bpov1.VoiceTurnResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.calls[req.Msg.CallId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("call %s not found", req.Msg.CallId))
	}

	// Append caller utterance to transcript
	session.LiveTranscript += fmt.Sprintf("\n[Caller]: %s", req.Msg.CallerUtterance)
	session.CallDurationSeconds += 12

	botReply := "I understand. I am accessing your account records right now."
	intent := "SUPPORT_COMPLAINT"
	shouldHandoff := false
	dept := "SUPPORT"

	text := req.Msg.CallerUtterance
	if len(text) > 0 {
		// Rule-based fast intent classifier
		if containsAny(text, []string{"buy", "purchase", "price", "quote", "cost", "lease"}) {
			intent = "SALES_INQUIRY"
			dept = "SALES"
			botReply = "I have located your requirement in our system. Let me transfer you directly to our Sales Specialist who has your quotation on screen."
			shouldHandoff = true
		} else if containsAny(text, []string{"pain", "doctor", "medicine", "emergency", "fever", "chest"}) {
			intent = "MEDICAL_TRIAGE"
			dept = "CLINICAL"
			botReply = "I am prioritizing your call for clinical assistance. Transferring immediately to our Consultant on duty."
			shouldHandoff = true
		} else if containsAny(text, []string{"agent", "human", "person", "operator", "speak to someone"}) {
			intent = "SUPPORT_COMPLAINT"
			dept = "SUPPORT"
			botReply = "Certainly! Connecting you to an available support specialist."
			shouldHandoff = true
		} else {
			botReply = "Thank you for the details. I have updated your file and sent a verification link to your WhatsApp."
		}
	}

	session.LiveTranscript += fmt.Sprintf("\n[Bot]: %s", botReply)
	session.Intent = intent
	session.TargetDepartment = dept
	if shouldHandoff {
		session.Status = "QUEUED_FOR_AGENT"
	}

	return connect.NewResponse(&bpov1.VoiceTurnResponse{
		BotReplyText:      botReply,
		DetectedIntent:    intent,
		ShouldHandoff:     shouldHandoff,
		HandoffDepartment: dept,
		AiSummaryUpdate:   session.AiSummary,
	}), nil
}

func (s *VoiceBPOServiceServer) RequestHumanHandoff(ctx context.Context, req *connect.Request[bpov1.HandoffRequest]) (*connect.Response[bpov1.HandoffResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.calls[req.Msg.CallId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("call %s not found", req.Msg.CallId))
	}

	session.Status = "IN_CALL_WITH_AGENT"
	session.TargetDepartment = req.Msg.TargetDepartment
	if req.Msg.PreferredAgent != "" {
		session.AssignedAgent = req.Msg.PreferredAgent
	} else {
		session.AssignedAgent = "Assigned to " + req.Msg.TargetDepartment + " Rep"
	}

	// Generate real-time Screen Pop Payload
	screenPop := map[string]interface{}{
		"call_id":                 session.CallId,
		"customer_name":           session.CustomerName,
		"caller_phone":            session.CallerPhone,
		"department":              session.TargetDepartment,
		"intent":                  session.Intent,
		"ai_summary":              session.AiSummary,
		"transcript_snippet":      session.LiveTranscript,
		"khata_balance":           session.CustomerKhataBalance,
		"active_deal_appointment": session.ActiveDealOrAppointment,
		"livekit_room":            session.LivekitRoomToken,
	}
	bytes, _ := json.Marshal(screenPop)

	return connect.NewResponse(&bpov1.HandoffResponse{
		Session:           session,
		TransferStatus:    "BRIDGED",
		ScreenPopPayload:  string(bytes),
	}), nil
}

func (s *VoiceBPOServiceServer) ListActiveCalls(ctx context.Context, req *connect.Request[bpov1.ListCallsRequest]) (*connect.Response[bpov1.ListCallsResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*bpov1.CallSession
	activeCount := int32(0)
	for _, c := range s.calls {
		if c.Status != "COMPLETED" {
			activeCount++
		}
		if req.Msg.Status != "" && c.Status != req.Msg.Status {
			continue
		}
		list = append(list, c)
	}

	return connect.NewResponse(&bpov1.ListCallsResponse{
		Calls:                 list,
		TotalActiveCalls:      activeCount,
		TotalAiHandledToday:   184,
		TotalTransferredToday: 26,
	}), nil
}

func (s *VoiceBPOServiceServer) LaunchOutboundCampaign(ctx context.Context, req *connect.Request[bpov1.LaunchCampaignRequest]) (*connect.Response[bpov1.LaunchCampaignResponse], error) {
	campaignId := fmt.Sprintf("CMP-%d", time.Now().Unix()%10000)
	count := int32(len(req.Msg.TargetPhoneNumbers))
	if count == 0 {
		count = 25 // default batch demo
	}

	return connect.NewResponse(&bpov1.LaunchCampaignResponse{
		CampaignId:     campaignId,
		CallsScheduled: count,
		Status:         "SCHEDULED_FOR_AUTONOMOUS_DIALING",
	}), nil
}

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

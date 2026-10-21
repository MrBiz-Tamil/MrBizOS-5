package omnichannel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	omnichannelv1 "drayya-core/gen/omnichannel/v1"
	"drayya-core/gen/omnichannel/v1/omnichannelv1connect"
	"gorm.io/gorm"
)

type OmnichannelServiceServer struct {
	omnichannelv1connect.UnimplementedOmnichannelServiceHandler
	db *gorm.DB
	mu sync.RWMutex

	conversations map[string]*omnichannelv1.ConversationSummary
	messages      map[string][]*omnichannelv1.MessageItem
}

func NewService(db *gorm.DB) *OmnichannelServiceServer {
	s := &OmnichannelServiceServer{
		db:            db,
		conversations: make(map[string]*omnichannelv1.ConversationSummary),
		messages:      make(map[string][]*omnichannelv1.MessageItem),
	}
	s.seedDefaults()
	return s
}

func (s *OmnichannelServiceServer) seedDefaults() {
	// WhatsApp Thread
	s.conversations["CONV-WA-001"] = &omnichannelv1.ConversationSummary{
		Id:                 "CONV-WA-001",
		Channel:            omnichannelv1.ChannelType_CHANNEL_TYPE_WHATSAPP,
		CustomerName:       "Rajesh Kumar",
		CustomerIdentifier: "+91 98450 11223",
		LastMessage:        "Can you send the revised quotation for the office space?",
		LastMessageTime:    "10 mins ago",
		UnreadCount:        2,
		Status:             "OPEN",
		AssignedAgent:      "Divya (Sales Lead)",
		BusinessUnit:       "REALESTATE",
		Sentiment:          "URGENT",
	}

	s.messages["CONV-WA-001"] = []*omnichannelv1.MessageItem{
		{
			Id:             "MSG-01",
			ConversationId: "CONV-WA-001",
			SenderType:     "CUSTOMER",
			SenderName:     "Rajesh Kumar",
			Content:        "Hello, we visited the Guindy tech park office yesterday. Location looks good.",
			SentAt:         "10:15 AM",
			DeliveryStatus: "READ",
		},
		{
			Id:             "MSG-02",
			ConversationId: "CONV-WA-001",
			SenderType:     "AGENT",
			SenderName:     "Divya (Sales Lead)",
			Content:        "Glad you liked it Rajesh! The carpet area is 4,200 sqft with 50 dedicated workstations.",
			SentAt:         "10:18 AM",
			DeliveryStatus: "READ",
		},
		{
			Id:             "MSG-03",
			ConversationId: "CONV-WA-001",
			SenderType:     "CUSTOMER",
			SenderName:     "Rajesh Kumar",
			Content:        "Can you send the revised quotation for the office space including maintenance?",
			SentAt:         "10:25 AM",
			DeliveryStatus: "DELIVERED",
		},
	}

	// Instagram DM Thread
	s.conversations["CONV-IG-002"] = &omnichannelv1.ConversationSummary{
		Id:                 "CONV-IG-002",
		Channel:            omnichannelv1.ChannelType_CHANNEL_TYPE_INSTAGRAM,
		CustomerName:       "Ananya Roy",
		CustomerIdentifier: "@ananya_travels",
		LastMessage:        "Do you have a package for Switzerland in December?",
		LastMessageTime:    "25 mins ago",
		UnreadCount:        1,
		Status:             "OPEN",
		AssignedAgent:      "Vikram (Travel Desk)",
		BusinessUnit:       "TRAVEL",
		Sentiment:          "POSITIVE",
	}

	s.messages["CONV-IG-002"] = []*omnichannelv1.MessageItem{
		{
			Id:             "MSG-IG-01",
			ConversationId: "CONV-IG-002",
			SenderType:     "CUSTOMER",
			SenderName:     "Ananya Roy",
			Content:        "Loved your latest reel on Zurich! Do you have a package for Switzerland in December?",
			SentAt:         "09:55 AM",
			DeliveryStatus: "DELIVERED",
		},
	}

	// Facebook Messenger Thread
	s.conversations["CONV-FB-003"] = &omnichannelv1.ConversationSummary{
		Id:                 "CONV-FB-003",
		Channel:            omnichannelv1.ChannelType_CHANNEL_TYPE_FACEBOOK,
		CustomerName:       "Dr. Ramesh Babu",
		CustomerIdentifier: "fb_user_897123",
		LastMessage:        "Is Dr. Ayya available for cardiology video consult today?",
		LastMessageTime:    "1 hour ago",
		UnreadCount:        0,
		Status:             "PENDING",
		AssignedAgent:      "AI Copilot (Auto-Triage)",
		BusinessUnit:       "HEALTH",
		Sentiment:          "NEUTRAL",
	}

	s.messages["CONV-FB-003"] = []*omnichannelv1.MessageItem{
		{
			Id:             "MSG-FB-01",
			ConversationId: "CONV-FB-003",
			SenderType:     "CUSTOMER",
			SenderName:     "Dr. Ramesh Babu",
			Content:        "Is Dr. Ayya available for cardiology video consult today?",
			SentAt:         "09:10 AM",
			DeliveryStatus: "READ",
		},
		{
			Id:             "MSG-FB-02",
			SenderType:     "AI_BOT",
			ConversationId: "CONV-FB-003",
			SenderName:     "MrBiz-OS AI Copilot",
			Content:        "Dr. Ayya has open slots today at 3:30 PM and 5:00 PM IST. Would you like me to reserve a 30-min consultation slot for you?",
			SentAt:         "09:11 AM",
			DeliveryStatus: "READ",
		},
	}

	// Email Thread
	s.conversations["CONV-EM-004"] = &omnichannelv1.ConversationSummary{
		Id:                 "CONV-EM-004",
		Channel:            omnichannelv1.ChannelType_CHANNEL_TYPE_EMAIL,
		CustomerName:       "Finance Dept (TechVision)",
		CustomerIdentifier: "billing@techvision.io",
		LastMessage:        "Wire transfer completed for invoice INV-2026-042",
		LastMessageTime:    "2 hours ago",
		UnreadCount:        0,
		Status:             "RESOLVED",
		AssignedAgent:      "Priya (Finance)",
		BusinessUnit:       "CONSULTING",
		Sentiment:          "POSITIVE",
	}

	s.messages["CONV-EM-004"] = []*omnichannelv1.MessageItem{
		{
			Id:             "MSG-EM-01",
			ConversationId: "CONV-EM-004",
			SenderType:     "CUSTOMER",
			SenderName:     "Finance Dept (TechVision)",
			Content:        "Dear MrBiz-OS Team, We have processed the wire transfer of $14,500 for INV-2026-042. Please find the bank advice slip attached.",
			SentAt:         "08:20 AM",
			DeliveryStatus: "READ",
		},
	}
}

func (s *OmnichannelServiceServer) ListConversations(ctx context.Context, req *connect.Request[omnichannelv1.ListConversationsRequest]) (*connect.Response[omnichannelv1.ListConversationsResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*omnichannelv1.ConversationSummary
	totalOpen := int32(0)
	totalUnread := int32(0)

	for _, c := range s.conversations {
		if c.Status == "OPEN" {
			totalOpen++
		}
		totalUnread += c.UnreadCount

		if req.Msg.Channel != omnichannelv1.ChannelType_CHANNEL_TYPE_UNSPECIFIED && c.Channel != req.Msg.Channel {
			continue
		}
		if req.Msg.Status != "" && c.Status != req.Msg.Status {
			continue
		}
		if req.Msg.BusinessUnit != "" && c.BusinessUnit != req.Msg.BusinessUnit {
			continue
		}
		list = append(list, c)
	}

	return connect.NewResponse(&omnichannelv1.ListConversationsResponse{
		Conversations: list,
		TotalOpen:     totalOpen,
		TotalUnread:   totalUnread,
	}), nil
}

func (s *OmnichannelServiceServer) GetConversation(ctx context.Context, req *connect.Request[omnichannelv1.GetConversationRequest]) (*connect.Response[omnichannelv1.GetConversationResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c, ok := s.conversations[req.Msg.ConversationId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("conversation %s not found", req.Msg.ConversationId))
	}

	// Clear unread on open
	c.UnreadCount = 0

	return connect.NewResponse(&omnichannelv1.GetConversationResponse{
		Summary:             c,
		Messages:            s.messages[req.Msg.ConversationId],
		CustomerEmail:       "client@mrbiz.os",
		CustomerPhone:       c.CustomerIdentifier,
		ActiveJourneysCount: 1,
		OpenInvoicesCount:   1,
	}), nil
}

func (s *OmnichannelServiceServer) SendMessage(ctx context.Context, req *connect.Request[omnichannelv1.SendMessageRequest]) (*connect.Response[omnichannelv1.SendMessageResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.conversations[req.Msg.ConversationId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("conversation %s not found", req.Msg.ConversationId))
	}

	msgId := fmt.Sprintf("MSG-%04d", time.Now().Unix()%10000)
	m := &omnichannelv1.MessageItem{
		Id:             msgId,
		ConversationId: c.Id,
		SenderType:     "AGENT",
		SenderName:     "Agent On-Duty",
		Content:        req.Msg.Content,
		MediaUrl:       req.Msg.MediaUrl,
		MediaType:      req.Msg.MediaType,
		SentAt:         time.Now().Format("03:04 PM"),
		DeliveryStatus: "SENT",
	}

	s.messages[c.Id] = append(s.messages[c.Id], m)
	c.LastMessage = req.Msg.Content
	c.LastMessageTime = "Just now"

	return connect.NewResponse(&omnichannelv1.SendMessageResponse{
		Message:           m,
		ExternalMessageId: fmt.Sprintf("ext_%s_%d", c.Id, time.Now().Unix()),
	}), nil
}

func (s *OmnichannelServiceServer) UpdateConversationStatus(ctx context.Context, req *connect.Request[omnichannelv1.UpdateConversationStatusRequest]) (*connect.Response[omnichannelv1.UpdateConversationStatusResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.conversations[req.Msg.ConversationId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("conversation %s not found", req.Msg.ConversationId))
	}

	if req.Msg.Status != "" {
		c.Status = req.Msg.Status
	}
	if req.Msg.AssignedAgent != "" {
		c.AssignedAgent = req.Msg.AssignedAgent
	}

	return connect.NewResponse(&omnichannelv1.UpdateConversationStatusResponse{Success: true}), nil
}

func (s *OmnichannelServiceServer) ConvertToLeadOrTask(ctx context.Context, req *connect.Request[omnichannelv1.ConvertToLeadOrTaskRequest]) (*connect.Response[omnichannelv1.ConvertToLeadOrTaskResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c, ok := s.conversations[req.Msg.ConversationId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("conversation %s not found", req.Msg.ConversationId))
	}

	createdId := fmt.Sprintf("%s-%d", req.Msg.TargetType, time.Now().Unix()%10000)
	msg := fmt.Sprintf("Successfully converted %s conversation with %s into %s #%s",
		c.Channel.String(), c.CustomerName, req.Msg.TargetType, createdId)

	return connect.NewResponse(&omnichannelv1.ConvertToLeadOrTaskResponse{
		CreatedId: createdId,
		Message:   msg,
	}), nil
}

func (s *OmnichannelServiceServer) SuggestAIReply(ctx context.Context, req *connect.Request[omnichannelv1.SuggestAIReplyRequest]) (*connect.Response[omnichannelv1.SuggestAIReplyResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c, ok := s.conversations[req.Msg.ConversationId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("conversation %s not found", req.Msg.ConversationId))
	}

	var suggestion string
	var sources []string

	switch c.BusinessUnit {
	case "REALESTATE":
		suggestion = fmt.Sprintf("Hi %s, thank you for your visit to Guindy Tech Park! I have prepared the revised quotation with maintenance at ₹65/sqft. Can I send the PDF over WhatsApp?", c.CustomerName)
		sources = []string{"Property: Guindy Tech Park (4,200 sqft)", "Base Rate: ₹58/sqft + ₹7 maintenance"}
	case "TRAVEL":
		suggestion = fmt.Sprintf("Hello %s! Yes, our December Switzerland 7D/6N Glacier Express tour starts at ₹1,45,000 including 4-star hotels, Swiss Travel Pass, and visa appointment assistance. Would you like the complete itinerary?", c.CustomerName)
		sources = []string{"Tour Package: Winter Swiss Explorer 2026", "Available Departure: Dec 14 & Dec 21"}
	default:
		suggestion = fmt.Sprintf("Hello %s! Thank you for contacting MrBiz-OS. How can our team assist you today?", c.CustomerName)
		sources = []string{"General FAQ Knowledge Base"}
	}

	return connect.NewResponse(&omnichannelv1.SuggestAIReplyResponse{
		SuggestedReply:  suggestion,
		ConfidenceScore: 0.94,
		ReferenceSources: sources,
	}), nil
}

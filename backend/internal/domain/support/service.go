package support

import (
	"fmt"
	"time"
)

type Service interface {
	// User-facing
	GetUserTickets(userID uint) ([]Ticket, error)
	CreateTicket(userID uint, subject, category, priority, description string) (*Ticket, error)
	GetTicketDetails(ticketID uint) (*Ticket, error)
	AddMessage(ticketID, senderID uint, message string, isStaff bool) (*TicketMessage, error)
	UpdateStatus(ticketID uint, status string) (*Ticket, error)
	// Admin-facing
	GetAllTickets(status, department string) ([]Ticket, error)
	AdminReply(ticketID, agentID uint, senderName, message string) (*TicketMessage, error)
	AssignTicket(ticketID, agentID uint) error
	SetDepartment(ticketID uint, dept string) error
	GetStats() (map[string]int64, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// ── User-facing ───────────────────────────────────────────────────────────────

func (s *service) GetUserTickets(userID uint) ([]Ticket, error) {
	return s.repo.GetTicketsByUserID(userID)
}

func (s *service) CreateTicket(userID uint, subject, category, priority, description string) (*Ticket, error) {
	if subject == "" {
		return nil, fmt.Errorf("subject is required")
	}
	if priority == "" {
		priority = "normal"
	}
	if category == "" {
		category = "general"
	}
	if !ValidDepartments[category] {
		category = "general"
	}

	ticket := &Ticket{
		UserID:      userID,
		Subject:     subject,
		Description: description,
		Category:    category,
		Department:  category, // preliminary; AI will refine
		Priority:    priority,
		Status:      "open",
	}

	if err := s.repo.CreateTicket(ticket); err != nil {
		return nil, err
	}

	// Save the opening message
	if description != "" {
		msg := &TicketMessage{
			TicketID: ticket.ID,
			SenderID: userID,
			Message:  description,
			IsStaff:  false,
		}
		_ = s.repo.CreateMessage(msg)
	}

	// Fire AI classification + auto-reply asynchronously
	processTicketAsync(s.repo, ticket)

	return ticket, nil
}

func (s *service) GetTicketDetails(ticketID uint) (*Ticket, error) {
	return s.repo.GetTicketWithMessages(ticketID)
}

func (s *service) AddMessage(ticketID, senderID uint, message string, isStaff bool) (*TicketMessage, error) {
	if message == "" {
		return nil, fmt.Errorf("message cannot be empty")
	}
	// Re-open ticket if user replies to a resolved/closed one
	if !isStaff {
		ticket, err := s.repo.GetTicketByID(ticketID)
		if err == nil && (ticket.Status == "resolved" || ticket.Status == "closed") {
			_ = s.repo.SetStatus(ticketID, "in_progress")
		}
	}
	msg := &TicketMessage{
		TicketID: ticketID,
		SenderID: senderID,
		Message:  message,
		IsStaff:  isStaff,
	}
	if err := s.repo.CreateMessage(msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (s *service) UpdateStatus(ticketID uint, status string) (*Ticket, error) {
	ticket, err := s.repo.GetTicketByID(ticketID)
	if err != nil {
		return nil, err
	}
	ticket.Status = status
	if status == "closed" || status == "resolved" {
		now := time.Now()
		ticket.ClosedAt = &now
	}
	if err := s.repo.UpdateTicket(ticket); err != nil {
		return nil, err
	}
	return ticket, nil
}

// ── Admin-facing ──────────────────────────────────────────────────────────────

func (s *service) GetAllTickets(status, department string) ([]Ticket, error) {
	return s.repo.GetAllTickets(status, department)
}

func (s *service) AdminReply(ticketID, agentID uint, senderName, message string) (*TicketMessage, error) {
	if message == "" {
		return nil, fmt.Errorf("message cannot be empty")
	}
	msg := &TicketMessage{
		TicketID:   ticketID,
		SenderID:   agentID,
		SenderName: senderName,
		Message:    message,
		IsStaff:    true,
		IsAIReply:  false,
	}
	if err := s.repo.CreateMessage(msg); err != nil {
		return nil, err
	}
	// Move to in_progress when human agent first replies
	_ = s.repo.SetStatus(ticketID, "in_progress")
	return msg, nil
}

func (s *service) AssignTicket(ticketID, agentID uint) error {
	return s.repo.AssignTicket(ticketID, agentID)
}

func (s *service) SetDepartment(ticketID uint, dept string) error {
	if !ValidDepartments[dept] {
		return fmt.Errorf("invalid department: %s", dept)
	}
	return s.repo.SetDepartment(ticketID, dept)
}

func (s *service) GetStats() (map[string]int64, error) {
	return s.repo.GetStats()
}

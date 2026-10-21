package support

import (
	"fmt"
	"gorm.io/gorm"
)

type Repository interface {
	AutoMigrate() error
	// Tickets
	CreateTicket(ticket *Ticket) error
	GetTicketByID(id uint) (*Ticket, error)
	GetTicketWithMessages(id uint) (*Ticket, error)
	GetTicketsByUserID(userID uint) ([]Ticket, error)
	GetAllTickets(status, department string) ([]Ticket, error)
	UpdateTicket(ticket *Ticket) error
	SetStatus(ticketID uint, status string) error
	SetDepartment(ticketID uint, dept string) error
	SetAIHandled(ticketID uint, handled bool) error
	AssignTicket(ticketID, agentID uint) error
	// Messages
	CreateMessage(msg *TicketMessage) error
	// Stats
	GetStats() (map[string]int64, error)
	EscalateStaleTickets() (int64, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) AutoMigrate() error {
	err := r.db.AutoMigrate(&Ticket{}, &TicketMessage{}, &TicketAuditLog{})
	if err != nil {
		return err
	}
	
	// Enforce strict Database State Machine constraints (Zero-Error method)
	// We use IF NOT EXISTS equivalents by wrapping in a safe execute, or just executing directly 
	// (GORM doesn't natively do check constraints during AutoMigrate without tags, but raw SQL is safer)
	r.db.Exec(`ALTER TABLE tickets ADD CONSTRAINT chk_ticket_status CHECK (status IN ('open', 'in_progress', 'escalated', 'resolved', 'closed'))`)
	r.db.Exec(`ALTER TABLE tickets ADD CONSTRAINT chk_ticket_department CHECK (department IN ('billing', 'medical', 'pharmacy', 'technical', 'general', 'hr', 'logistics'))`)
	r.db.Exec(`ALTER TABLE tickets ADD CONSTRAINT chk_ticket_priority CHECK (priority IN ('low', 'normal', 'high', 'urgent'))`)
	
	return nil
}

func (r *repository) CreateTicket(ticket *Ticket) error {
	return r.db.Create(ticket).Error
}

func (r *repository) GetTicketByID(id uint) (*Ticket, error) {
	var ticket Ticket
	if err := r.db.First(&ticket, id).Error; err != nil {
		return nil, err
	}
	return &ticket, nil
}

func (r *repository) GetTicketWithMessages(id uint) (*Ticket, error) {
	var ticket Ticket
	if err := r.db.Preload("Messages", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at ASC")
	}).First(&ticket, id).Error; err != nil {
		return nil, err
	}
	return &ticket, nil
}

func (r *repository) GetTicketsByUserID(userID uint) ([]Ticket, error) {
	var tickets []Ticket
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&tickets).Error
	return tickets, err
}

func (r *repository) GetAllTickets(status, department string) ([]Ticket, error) {
	var tickets []Ticket
	q := r.db.Order("created_at DESC")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if department != "" {
		q = q.Where("department = ?", department)
	}
	err := q.Find(&tickets).Error
	return tickets, err
}

func (r *repository) UpdateTicket(ticket *Ticket) error {
	var oldTicket Ticket
	r.db.First(&oldTicket, ticket.ID)

	err := r.db.Save(ticket).Error
	if err == nil && oldTicket.ID > 0 && oldTicket.Status != ticket.Status {
		r.db.Create(&TicketAuditLog{
			TicketID: ticket.ID, 
			Action:   "status_changed", 
			OldValue: oldTicket.Status, 
			NewValue: ticket.Status,
		})
	}
	return err
}

func (r *repository) SetStatus(ticketID uint, status string) error {
	err := r.db.Model(&Ticket{}).Where("id = ?", ticketID).Update("status", status).Error
	if err == nil {
		r.db.Create(&TicketAuditLog{TicketID: ticketID, Action: "status_changed", NewValue: status})
	}
	return err
}

func (r *repository) SetDepartment(ticketID uint, dept string) error {
	err := r.db.Model(&Ticket{}).Where("id = ?", ticketID).Update("department", dept).Error
	if err == nil {
		r.db.Create(&TicketAuditLog{TicketID: ticketID, Action: "department_changed", NewValue: dept})
	}
	return err
}

func (r *repository) SetAIHandled(ticketID uint, handled bool) error {
	return r.db.Model(&Ticket{}).Where("id = ?", ticketID).Update("ai_handled", handled).Error
}

func (r *repository) AssignTicket(ticketID, agentID uint) error {
	err := r.db.Model(&Ticket{}).Where("id = ?", ticketID).Update("assigned_to", agentID).Error
	if err == nil {
		// Log the assignment
		importFmt := "0"
		if agentID > 0 {
			importFmt = fmt.Sprintf("%d", agentID)
		}
		r.db.Create(&TicketAuditLog{TicketID: ticketID, Action: "assigned", ActorID: agentID, NewValue: importFmt})
	}
	return err
}

func (r *repository) CreateMessage(msg *TicketMessage) error {
	return r.db.Create(msg).Error
}

func (r *repository) GetStats() (map[string]int64, error) {
	stats := make(map[string]int64)
	statuses := []string{"open", "in_progress", "resolved", "closed"}
	for _, s := range statuses {
		var count int64
		if err := r.db.Model(&Ticket{}).Where("status = ?", s).Count(&count).Error; err != nil {
			return nil, err
		}
		stats[s] = count
	}
	var aiHandled int64
	r.db.Model(&Ticket{}).Where("ai_handled = ?", true).Count(&aiHandled)
	stats["ai_handled"] = aiHandled
	return stats, nil
}

func (r *repository) EscalateStaleTickets() (int64, error) {
	// Find tickets that have been open for > 24 hours
	result := r.db.Model(&Ticket{}).
		Where("status = ?", "open").
		Where("created_at <= NOW() - INTERVAL '24 hours'").
		Updates(map[string]interface{}{"status": "escalated", "priority": "high"})
		
	if result.Error == nil && result.RowsAffected > 0 {
		// Note: Audit logs for bulk updates usually require a loop or a DB trigger.
		// For simplicity, we just rely on the updated status. A robust system would use a trigger or iterate.
	}
	return result.RowsAffected, result.Error
}

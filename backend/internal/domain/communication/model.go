package communication

import (
	"drayya-core/internal/domain/identity"
	"time"

	"gorm.io/gorm"
)

type ChatType string

const (
	ChatTypeConsultation ChatType = "consultation"
	ChatTypeDirect       ChatType = "direct"
	ChatTypeSupport      ChatType = "support"
)

type ChatUrgency string

const (
	ChatUrgencyNormal ChatUrgency = "normal"
	ChatUrgencyHigh   ChatUrgency = "high"
	ChatUrgencySOS    ChatUrgency = "sos"
)

type ChatRoom struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	AppointmentID  *uint          `gorm:"index" json:"appointment_id,omitempty"`
	DoctorID       uint           `gorm:"index" json:"doctor_id"`
	PatientID      uint           `gorm:"index" json:"patient_id"`
	FamilyMemberID *uint          `gorm:"index" json:"family_member_id,omitempty"` // set when booking for a family member
	OnBehalfNote   *string        `json:"on_behalf_note,omitempty"`                // e.g. "For son Arjun (age 8)"
	ChatType       string         `gorm:"default:'consultation'" json:"chat_type"`
	Urgency        string         `gorm:"default:'normal'" json:"urgency"`
	IsActive       bool           `gorm:"default:true" json:"is_active"`
	ClosedAt       *time.Time     `json:"closed_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Doctor  *identity.User `gorm:"foreignKey:DoctorID" json:"doctor,omitempty"`
	Patient *identity.User `gorm:"foreignKey:PatientID" json:"patient,omitempty"`
}

func (ChatRoom) TableName() string {
	return "chat_rooms"
}

type ChatMessage struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	RoomID          uint           `gorm:"index" json:"room_id"`
	SenderID        uint           `gorm:"index" json:"sender_id"`
	ReceiverID      uint           `gorm:"index" json:"receiver_id"`
	Message         string         `json:"message"`
	MessageType     string         `gorm:"default:'text'" json:"message_type"` // text, image, file, system
	FileURL         *string        `json:"file_url,omitempty"`
	IsRead          bool           `gorm:"default:false" json:"is_read"`
	ReadAt          *time.Time     `json:"read_at,omitempty"`
	EditedAt        *time.Time     `json:"edited_at,omitempty"`
	ParentMessageID *uint          `gorm:"index" json:"parent_message_id,omitempty"` // for threaded replies
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`

	// Relations
	Sender *identity.User `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
}

func (ChatMessage) TableName() string {
	return "chat_messages"
}

// ChatRoomWithMeta enriches a ChatRoom with computed fields for the room list UI.
type ChatRoomWithMeta struct {
	ChatRoom
	UnreadCount int64        `json:"unread_count"`
	LastMessage *ChatMessage `json:"last_message,omitempty"`
}

type NotificationTemplate struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	TemplateType   string         `gorm:"uniqueIndex;not null" json:"template_type"`
	MessageContent string         `gorm:"type:text;not null" json:"message_content"`
	Variables      string         `json:"variables"` // Comma separated list of vars e.g. "name,date"
	IsActive       bool           `gorm:"default:true" json:"is_active"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (NotificationTemplate) TableName() string {
	return "notification_templates"
}

type NotificationLog struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"index" json:"user_id"`
	AppointmentID    *uint     `gorm:"index" json:"appointment_id,omitempty"`
	NotificationType string    `json:"notification_type"` // sms, whatsapp, email
	TemplateType     string    `json:"template_type"`
	Status           string    `json:"status"` // sent, failed
	ErrorMessage     *string   `json:"error_message,omitempty"`
	SentAt           time.Time `json:"sent_at"`

	// Relations
	User *identity.User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (NotificationLog) TableName() string {
	return "notification_logs"
}

type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Title     string    `gorm:"not null" json:"title"`
	Message   string    `gorm:"type:text;not null" json:"message"`
	Type      string    `gorm:"not null" json:"type"` // info, success, warning, error
	IsRead    bool      `gorm:"default:false" json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	User *identity.User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Notification) TableName() string {
	return "notifications"
}

type VideoConsultation struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	SessionID     string    `gorm:"uniqueIndex" json:"session_id"`
	AppointmentID uint      `gorm:"index" json:"appointment_id"`
	DoctorID      uint      `gorm:"index" json:"doctor_id"`
	PatientID     uint      `gorm:"index" json:"patient_id"`
	Status        string    `json:"status"` // in_progress, completed, failed
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (VideoConsultation) TableName() string {
	return "video_consultations"
}

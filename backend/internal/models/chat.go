//go:build ignore
// +build ignore

package models

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
	ID            uint           `gorm:"primaryKey" json:"id"`
	AppointmentID *uint          `gorm:"index" json:"appointment_id,omitempty"` // Nullable for direct chats
	DoctorID      uint           `gorm:"index" json:"doctor_id"`
	PatientID     uint           `gorm:"index" json:"patient_id"`
	ChatType      string         `gorm:"default:'consultation'" json:"chat_type"`
	Urgency       string         `gorm:"default:'normal'" json:"urgency"`
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	ClosedAt      *time.Time     `json:"closed_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Doctor  *identity.User `gorm:"foreignKey:DoctorID" json:"doctor,omitempty"`
	Patient *identity.User `gorm:"foreignKey:PatientID" json:"patient,omitempty"`
}

func (ChatRoom) TableName() string {
	return "chat_rooms"
}

type ChatMessage struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	RoomID      uint       `gorm:"index" json:"room_id"`
	SenderID    uint       `gorm:"index" json:"sender_id"`
	ReceiverID  uint       `gorm:"index" json:"receiver_id"`
	Message     string     `json:"message"`
	MessageType string     `gorm:"default:'text'" json:"message_type"` // text, image, file, system
	FileURL     *string    `json:"file_url,omitempty"`
	IsRead      bool       `gorm:"default:false" json:"is_read"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Relations
	Sender *identity.User `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
}

func (ChatMessage) TableName() string {
	return "chat_messages"
}

package communication

import (
	"context"
	"fmt"
	"drayya-core/internal/domain/identity"
	"time"
)

type Service interface {
	// Chat rooms
	GetOrCreateChatRoom(ctx context.Context, doctorID, patientID uint, chatType string) (*ChatRoom, error)
	GetRoomByID(ctx context.Context, id uint) (*ChatRoom, error)
	GetRoomsForUser(ctx context.Context, userID uint) ([]ChatRoomWithMeta, error)
	GetActiveConsultations(ctx context.Context, doctorID uint) ([]ChatRoom, error)

	// Chat messages
	SendMessage(ctx context.Context, roomID, senderID, receiverID uint, message string, msgType string, fileURL *string) (*ChatMessage, error)
	GetChatHistory(ctx context.Context, roomID uint, limit int) ([]ChatMessage, error)
	SearchMessages(ctx context.Context, roomID uint, query string, limit int) ([]ChatMessage, error)
	DeleteMessage(ctx context.Context, messageID, callerID uint) error
	GetUnreadCount(ctx context.Context, roomID, userID uint) (int64, error)
	MarkMessagesRead(ctx context.Context, roomID, userID uint) error

	// Notifications
	SendNotification(ctx context.Context, userID uint, title, message, nType string) error
	LogExternalNotification(ctx context.Context, nType, templateType string, userID uint, appointmentID *uint, status string, errMsg *string) error
	GetUserNotifications(ctx context.Context, userID uint) ([]Notification, error)
	MarkNotificationRead(ctx context.Context, id uint) error
	SendEmail(ctx context.Context, to, subject, body string) error
}

type service struct {
	repo         Repository
	identityRepo identity.Repository
}

func NewService(repo Repository, iRepo identity.Repository) Service {
	return &service{repo: repo, identityRepo: iRepo}
}

func (s *service) GetOrCreateChatRoom(ctx context.Context, doctorID, patientID uint, chatType string) (*ChatRoom, error) {
	// Simple lookup for existing active room for this pair, or create new
	rooms, err := s.repo.GetActiveChatRoomsByDoctor(ctx, doctorID)
	if err == nil {
		for _, r := range rooms {
			if r.PatientID == patientID && r.ChatType == chatType {
				return &r, nil
			}
		}
	}

	room := &ChatRoom{
		DoctorID:  doctorID,
		PatientID: patientID,
		ChatType:  chatType,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateChatRoom(ctx, room); err != nil {
		return nil, err
	}
	return room, nil
}

func (s *service) SendMessage(ctx context.Context, roomID, senderID, receiverID uint, message string, msgType string, fileURL *string) (*ChatMessage, error) {
	msg := &ChatMessage{
		RoomID:      roomID,
		SenderID:    senderID,
		ReceiverID:  receiverID,
		Message:     message,
		MessageType: msgType,
		FileURL:     fileURL,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.CreateChatMessage(ctx, msg); err != nil {
		return nil, err
	}

	// Update room touched time
	room, _ := s.repo.GetChatRoomByID(ctx, roomID)
	if room != nil {
		room.UpdatedAt = time.Now()
		s.repo.UpdateChatRoom(ctx, room)
	}

	return msg, nil
}

func (s *service) GetChatHistory(ctx context.Context, roomID uint, limit int) ([]ChatMessage, error) {
	return s.repo.GetChatMessages(ctx, roomID, limit)
}

func (s *service) SearchMessages(ctx context.Context, roomID uint, query string, limit int) ([]ChatMessage, error) {
	return s.repo.SearchMessages(ctx, roomID, query, limit)
}

func (s *service) DeleteMessage(ctx context.Context, messageID, callerID uint) error {
	msg, err := s.repo.GetMessageByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("message not found")
	}
	if msg.SenderID != callerID {
		return fmt.Errorf("forbidden: you can only delete your own messages")
	}
	return s.repo.SoftDeleteMessage(ctx, messageID)
}

func (s *service) GetUnreadCount(ctx context.Context, roomID, userID uint) (int64, error) {
	return s.repo.GetUnreadCount(ctx, roomID, userID)
}

func (s *service) MarkMessagesRead(ctx context.Context, roomID, userID uint) error {
	return s.repo.MarkMessagesRead(ctx, roomID, userID)
}

func (s *service) GetRoomByID(ctx context.Context, id uint) (*ChatRoom, error) {
	return s.repo.GetChatRoomByID(ctx, id)
}

func (s *service) GetRoomsForUser(ctx context.Context, userID uint) ([]ChatRoomWithMeta, error) {
	rooms, err := s.repo.GetRoomsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]ChatRoomWithMeta, 0, len(rooms))
	for _, r := range rooms {
		meta := ChatRoomWithMeta{ChatRoom: r}
		if count, cerr := s.repo.GetUnreadCount(ctx, r.ID, userID); cerr == nil {
			meta.UnreadCount = count
		}
		if last, lerr := s.repo.GetLatestMessage(ctx, r.ID); lerr == nil {
			meta.LastMessage = last
		}
		result = append(result, meta)
	}
	return result, nil
}

func (s *service) GetActiveConsultations(ctx context.Context, doctorID uint) ([]ChatRoom, error) {
	return s.repo.GetActiveChatRoomsByDoctor(ctx, doctorID)
}

func (s *service) SendNotification(ctx context.Context, userID uint, title, message, nType string) error {
	notification := &Notification{
		UserID:    userID,
		Title:     title,
		Message:   message,
		Type:      nType,
		IsRead:    false,
		CreatedAt: time.Now(),
	}
	return s.repo.CreateNotification(ctx, notification)
}

func (s *service) LogExternalNotification(ctx context.Context, nType, templateType string, userID uint, appointmentID *uint, status string, errMsg *string) error {
	log := &NotificationLog{
		UserID:           userID,
		AppointmentID:    appointmentID,
		NotificationType: nType,
		TemplateType:     templateType,
		Status:           status,
		ErrorMessage:     errMsg,
		SentAt:           time.Now(),
	}
	return s.repo.CreateNotificationLog(ctx, log)
}

func (s *service) GetUserNotifications(ctx context.Context, userID uint) ([]Notification, error) {
	return s.repo.GetNotificationsByUser(ctx, userID)
}

func (s *service) MarkNotificationRead(ctx context.Context, id uint) error {
	return s.repo.MarkNotificationRead(ctx, id)
}

func (s *service) SendEmail(ctx context.Context, to, subject, body string) error {
	// Simulated email sending
	fmt.Printf("\n--- [SIMULATED EMAIL] ---\nTo: %s\nSubject: %s\nBody: %s\n-------------------------\n\n", to, subject, body)
	
	// Also log it in notification_log
	return s.repo.CreateNotificationLog(ctx, &NotificationLog{
		UserID:           0, // System
		NotificationType: "email",
		TemplateType:     "manual",
		Status:           "sent",
		SentAt:           time.Now(),
	})
}

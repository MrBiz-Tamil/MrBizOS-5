package communication

import (
	"context"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&ChatRoom{},
		&ChatMessage{},
		&NotificationTemplate{},
		&NotificationLog{},
		&Notification{},
		&VideoConsultation{},
	)
}

type Repository interface {
	// Chat rooms
	CreateChatRoom(ctx context.Context, room *ChatRoom) error
	GetChatRoomByID(ctx context.Context, id uint) (*ChatRoom, error)
	GetActiveChatRoomsByDoctor(ctx context.Context, doctorID uint) ([]ChatRoom, error)
	GetActiveChatRoomsByPatient(ctx context.Context, patientID uint) ([]ChatRoom, error)
	GetRoomsForUser(ctx context.Context, userID uint) ([]ChatRoom, error)
	UpdateChatRoom(ctx context.Context, room *ChatRoom) error

	// Chat messages
	CreateChatMessage(ctx context.Context, msg *ChatMessage) error
	GetChatMessages(ctx context.Context, roomID uint, limit int) ([]ChatMessage, error)
	GetMessageByID(ctx context.Context, id uint) (*ChatMessage, error)
	GetLatestMessage(ctx context.Context, roomID uint) (*ChatMessage, error)
	SoftDeleteMessage(ctx context.Context, id uint) error
	GetUnreadCount(ctx context.Context, roomID, receiverID uint) (int64, error)
	MarkMessagesRead(ctx context.Context, roomID, receiverID uint) error
	SearchMessages(ctx context.Context, roomID uint, query string, limit int) ([]ChatMessage, error)

	// Notifications
	CreateNotificationTemplate(ctx context.Context, t *NotificationTemplate) error
	GetNotificationTemplate(ctx context.Context, templateType string) (*NotificationTemplate, error)
	CreateNotificationLog(ctx context.Context, log *NotificationLog) error
	CreateNotification(ctx context.Context, n *Notification) error
	GetNotificationsByUser(ctx context.Context, userID uint) ([]Notification, error)
	MarkNotificationRead(ctx context.Context, id uint) error
}

type defaultRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &defaultRepository{db: db}
}


func (r *defaultRepository) CreateChatRoom(ctx context.Context, room *ChatRoom) error {
	return r.db.WithContext(ctx).Create(room).Error
}

func (r *defaultRepository) GetChatRoomByID(ctx context.Context, id uint) (*ChatRoom, error) {
	var room ChatRoom
	err := r.db.WithContext(ctx).Preload("Doctor").Preload("Patient").First(&room, id).Error
	return &room, err
}

func (r *defaultRepository) GetActiveChatRoomsByDoctor(ctx context.Context, doctorID uint) ([]ChatRoom, error) {
	var rooms []ChatRoom
	err := r.db.WithContext(ctx).Where("doctor_id = ? AND is_active = ?", doctorID, true).Order("created_at desc").Find(&rooms).Error
	return rooms, err
}

func (r *defaultRepository) GetActiveChatRoomsByPatient(ctx context.Context, patientID uint) ([]ChatRoom, error) {
	var rooms []ChatRoom
	err := r.db.WithContext(ctx).Where("patient_id = ? AND is_active = ?", patientID, true).Order("created_at desc").Find(&rooms).Error
	return rooms, err
}

func (r *defaultRepository) GetRoomsForUser(ctx context.Context, userID uint) ([]ChatRoom, error) {
	var rooms []ChatRoom
	err := r.db.WithContext(ctx).
		Preload("Doctor").
		Preload("Patient").
		Where("doctor_id = ? OR patient_id = ?", userID, userID).
		Order("updated_at desc").
		Find(&rooms).Error
	return rooms, err
}

func (r *defaultRepository) UpdateChatRoom(ctx context.Context, room *ChatRoom) error {
	return r.db.WithContext(ctx).Save(room).Error
}

func (r *defaultRepository) CreateChatMessage(ctx context.Context, msg *ChatMessage) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

func (r *defaultRepository) GetChatMessages(ctx context.Context, roomID uint, limit int) ([]ChatMessage, error) {
	var msgs []ChatMessage
	err := r.db.WithContext(ctx).Where("room_id = ?", roomID).Order("created_at desc").Limit(limit).Find(&msgs).Error
	return msgs, err
}

func (r *defaultRepository) GetMessageByID(ctx context.Context, id uint) (*ChatMessage, error) {
	var msg ChatMessage
	err := r.db.WithContext(ctx).First(&msg, id).Error
	return &msg, err
}

func (r *defaultRepository) GetLatestMessage(ctx context.Context, roomID uint) (*ChatMessage, error) {
	var msg ChatMessage
	err := r.db.WithContext(ctx).Where("room_id = ?", roomID).Order("created_at desc").First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (r *defaultRepository) SoftDeleteMessage(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&ChatMessage{}, id).Error
}

func (r *defaultRepository) GetUnreadCount(ctx context.Context, roomID, receiverID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&ChatMessage{}).
		Where("room_id = ? AND receiver_id = ? AND is_read = ?", roomID, receiverID, false).
		Count(&count).Error
	return count, err
}

func (r *defaultRepository) MarkMessagesRead(ctx context.Context, roomID, receiverID uint) error {
	now := r.db.NowFunc()
	return r.db.WithContext(ctx).Model(&ChatMessage{}).
		Where("room_id = ? AND receiver_id = ? AND is_read = ?", roomID, receiverID, false).
		Updates(map[string]interface{}{"is_read": true, "read_at": now}).Error
}

func (r *defaultRepository) SearchMessages(ctx context.Context, roomID uint, query string, limit int) ([]ChatMessage, error) {
	var msgs []ChatMessage
	err := r.db.WithContext(ctx).
		Where("room_id = ? AND message LIKE ?", roomID, "%"+query+"%").
		Order("created_at desc").
		Limit(limit).
		Find(&msgs).Error
	return msgs, err
}

func (r *defaultRepository) CreateNotificationTemplate(ctx context.Context, t *NotificationTemplate) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *defaultRepository) GetNotificationTemplate(ctx context.Context, templateType string) (*NotificationTemplate, error) {
	var t NotificationTemplate
	err := r.db.WithContext(ctx).Where("template_type = ? AND is_active = ?", templateType, true).First(&t).Error
	return &t, err
}

func (r *defaultRepository) CreateNotificationLog(ctx context.Context, log *NotificationLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *defaultRepository) CreateNotification(ctx context.Context, n *Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *defaultRepository) GetNotificationsByUser(ctx context.Context, userID uint) ([]Notification, error) {
	var ns []Notification
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Find(&ns).Error
	return ns, err
}

func (r *defaultRepository) MarkNotificationRead(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&Notification{}).Where("id = ?", id).Update("is_read", true).Error
}

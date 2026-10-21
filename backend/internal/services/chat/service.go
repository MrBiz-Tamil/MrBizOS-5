package chat

import (
	"errors"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/communication"
	"time"
)

// GetOrCreateConsultationRoom finds an existing room for an appointment or creates one
func GetOrCreateConsultationRoom(appointmentID, doctorID, patientID uint) (*communication.ChatRoom, error) {
	db := database.DB
	var room communication.ChatRoom

	// Check if exists
	err := db.Where("appointment_id = ?", appointmentID).First(&room).Error
	if err == nil {
		return &room, nil
	}

	// Create new
	newRoom := communication.ChatRoom{
		AppointmentID: &appointmentID,
		DoctorID:      doctorID,
		PatientID:     patientID,
		ChatType:      string(communication.ChatTypeConsultation),
		IsActive:      true,
		CreatedAt:     time.Now(),
	}

	if err := db.Create(&newRoom).Error; err != nil {
		return nil, err
	}

	return &newRoom, nil
}

// GetOrCreateDirectRoom finds or creates a direct chat between two users
func GetOrCreateDirectRoom(user1ID, user2ID uint) (*communication.ChatRoom, error) {
	db := database.DB
	var room communication.ChatRoom

	// Check for existing room (generic approach: either user can be doctor or patient field in direct chat)
	// We check both combinations
	err := db.Where(
		"(doctor_id = ? AND patient_id = ? AND chat_type = ?) OR (doctor_id = ? AND patient_id = ? AND chat_type = ?)",
		user1ID, user2ID, communication.ChatTypeDirect,
		user2ID, user1ID, communication.ChatTypeDirect,
	).First(&room).Error

	if err == nil {
		return &room, nil
	}

	// Create new. Arbitrarily assign DoctorID/PatientID for storage.
	// Logic: If one is a doctor, put in DoctorID slot, else just ID1=Doc, ID2=Pat
	// Ideally we fetch user roles, but for speed we just store.
	newRoom := communication.ChatRoom{
		DoctorID:  user1ID,
		PatientID: user2ID,
		ChatType:  string(communication.ChatTypeDirect),
		IsActive:  true,
		CreatedAt: time.Now(),
	}

	if err := db.Create(&newRoom).Error; err != nil {
		return nil, err
	}

	return &newRoom, nil
}

// SaveMessage persists a chat message
func SaveMessage(roomID, senderID, receiverID uint, message, msgType string, fileURL *string) (*communication.ChatMessage, error) {
	db := database.DB

	msg := communication.ChatMessage{
		RoomID:      roomID,
		SenderID:    senderID,
		ReceiverID:  receiverID,
		Message:     message,
		MessageType: msgType,
		FileURL:     fileURL,
		IsRead:      false,
		CreatedAt:   time.Now(),
	}

	if err := db.Create(&msg).Error; err != nil {
		return nil, err
	}

	return &msg, nil
}

// GetRoomMessages retrieves history with pagination
func GetRoomMessages(roomID uint, limit, offset int) ([]communication.ChatMessage, error) {
	db := database.DB
	var messages []communication.ChatMessage

	err := db.Where("room_id = ?", roomID).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error

	return messages, err
}

// GetUserRooms retrieves all rooms for a user
func GetUserRooms(userID uint) ([]communication.ChatRoom, error) {
	db := database.DB
	var rooms []communication.ChatRoom

	err := db.Where("doctor_id = ? OR patient_id = ?", userID, userID).
		Preload("Doctor").
		Preload("Patient").
		Order("updated_at desc").
		Find(&rooms).Error

	return rooms, err
}

// CloseRoom marks a room as inactive
func CloseRoom(roomID, doctorID uint) error {
	db := database.DB
	var room communication.ChatRoom

	if err := db.First(&room, roomID).Error; err != nil {
		return err
	}

	// Only doctor owner can close (as per business logic)
	// But in Direct chats, maybe either? Following Python logic: `if current_user.id != room.doctor_id`
	if room.DoctorID != doctorID {
		return errors.New("unauthorized: only doctor can close room")
	}

	now := time.Now()
	room.IsActive = false
	room.ClosedAt = &now

	return db.Save(&room).Error
}

// DeleteMessage removes a message
func DeleteMessage(messageID, senderID uint) error {
	db := database.DB
	var msg communication.ChatMessage

	if err := db.First(&msg, messageID).Error; err != nil {
		return err
	}

	if msg.SenderID != senderID {
		return errors.New("unauthorized")
	}

	return db.Delete(&msg).Error
}

// UpdateRoom updates room metadata
func UpdateRoom(roomID uint, updates map[string]interface{}) error {
	return database.DB.Model(&communication.ChatRoom{}).Where("id = ?", roomID).Updates(updates).Error
}

// GetRoomByID returns a ChatRoom by its primary key — used by the WS client to resolve ReceiverID.
func GetRoomByID(roomID uint) (*communication.ChatRoom, error) {
	var room communication.ChatRoom
	err := database.DB.First(&room, roomID).Error
	return &room, err
}

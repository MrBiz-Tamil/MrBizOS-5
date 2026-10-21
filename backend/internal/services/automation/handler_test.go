package automation

import (
	"bytes"
	"context"
	"encoding/json"
	"drayya-core/internal/domain/communication"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/mock"
)

// MockCommService implements communication.Service for testing.
type MockCommService struct {
	mock.Mock
}

// ── Chat rooms ─────────────────────────────────────────────────────────────────

func (m *MockCommService) GetOrCreateChatRoom(ctx context.Context, doctorID, patientID uint, chatType string) (*communication.ChatRoom, error) {
	args := m.Called(ctx, doctorID, patientID, chatType)
	return args.Get(0).(*communication.ChatRoom), args.Error(1)
}
func (m *MockCommService) GetRoomByID(ctx context.Context, id uint) (*communication.ChatRoom, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*communication.ChatRoom), args.Error(1)
}
func (m *MockCommService) GetRoomsForUser(ctx context.Context, userID uint) ([]communication.ChatRoomWithMeta, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]communication.ChatRoomWithMeta), args.Error(1)
}
func (m *MockCommService) GetActiveConsultations(ctx context.Context, doctorID uint) ([]communication.ChatRoom, error) {
	args := m.Called(ctx, doctorID)
	return args.Get(0).([]communication.ChatRoom), args.Error(1)
}

// ── Chat messages ──────────────────────────────────────────────────────────────

func (m *MockCommService) SendMessage(ctx context.Context, roomID, senderID, receiverID uint, message string, msgType string, fileURL *string) (*communication.ChatMessage, error) {
	args := m.Called(ctx, roomID, senderID, receiverID, message, msgType, fileURL)
	return args.Get(0).(*communication.ChatMessage), args.Error(1)
}
func (m *MockCommService) GetChatHistory(ctx context.Context, roomID uint, limit int) ([]communication.ChatMessage, error) {
	args := m.Called(ctx, roomID, limit)
	return args.Get(0).([]communication.ChatMessage), args.Error(1)
}
func (m *MockCommService) SearchMessages(ctx context.Context, roomID uint, query string, limit int) ([]communication.ChatMessage, error) {
	args := m.Called(ctx, roomID, query, limit)
	return args.Get(0).([]communication.ChatMessage), args.Error(1)
}
func (m *MockCommService) DeleteMessage(ctx context.Context, messageID, callerID uint) error {
	return m.Called(ctx, messageID, callerID).Error(0)
}
func (m *MockCommService) GetUnreadCount(ctx context.Context, roomID, userID uint) (int64, error) {
	args := m.Called(ctx, roomID, userID)
	return args.Get(0).(int64), args.Error(1)
}
func (m *MockCommService) MarkMessagesRead(ctx context.Context, roomID, userID uint) error {
	return m.Called(ctx, roomID, userID).Error(0)
}

// ── Notifications ──────────────────────────────────────────────────────────────

func (m *MockCommService) SendNotification(ctx context.Context, userID uint, title, message, nType string) error {
	return m.Called(ctx, userID, title, message, nType).Error(0)
}
func (m *MockCommService) LogExternalNotification(ctx context.Context, nType, templateType string, userID uint, appointmentID *uint, status string, errMsg *string) error {
	return m.Called(ctx, nType, templateType, userID, appointmentID, status, errMsg).Error(0)
}
func (m *MockCommService) GetUserNotifications(ctx context.Context, userID uint) ([]communication.Notification, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]communication.Notification), args.Error(1)
}
func (m *MockCommService) MarkNotificationRead(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockCommService) SendEmail(ctx context.Context, to, subject, body string) error {
	return m.Called(ctx, to, subject, body).Error(0)
}

// MockHub captures broadcast calls.
type MockHub struct {
	LastRoomID string
	LastData   interface{}
}

func (m *MockHub) BroadcastMessage(roomID string, data interface{}) {
	m.LastRoomID = roomID
	m.LastData = data
}

// TestBotRespond verifies the admin BotRespond endpoint broadcasts to the hub.
func TestBotRespond(t *testing.T) {
	app := fiber.New()
	mockSvc := new(MockCommService)
	mockHub := &MockHub{}
	handler := NewHandler(mockSvc, mockHub)

	app.Post("/bot/respond", handler.BotRespond)

	reqBody := BotResponseRequest{
		RoomID:  "devi-user-1",
		UserID:  "1",
		Message: "Hello from admin",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/bot/respond", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if mockHub.LastRoomID != "devi-user-1" {
		t.Errorf("expected broadcast to devi-user-1, got %q", mockHub.LastRoomID)
	}
}

// TestDeviChat verifies the HTTP chat endpoint requires a non-empty message.
func TestDeviChat_EmptyMessage(t *testing.T) {
	app := fiber.New()
	mockSvc := new(MockCommService)
	mockHub := &MockHub{}
	handler := NewHandler(mockSvc, mockHub)

	app.Post("/devi/chat", handler.DeviChat)

	reqBody := ChatRequest{SessionID: "devi-guest", GuestBrowserID: "test-browser-uuid", Message: ""}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/devi/chat", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400 for empty message, got %d", resp.StatusCode)
	}
}

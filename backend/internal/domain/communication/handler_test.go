package communication

import (
	"bytes"
	"context"
	"drayya-core/internal/domain/career"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ─── Mock Service ─────────────────────────────────────────────────────────────

type mockCommSvc struct {
	mock.Mock
}

// ── Chat rooms ────────────────────────────────────────────────────────────────

func (m *mockCommSvc) GetOrCreateChatRoom(ctx context.Context, doctorID, patientID uint, chatType string) (*ChatRoom, error) {
	args := m.Called(ctx, doctorID, patientID, chatType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ChatRoom), args.Error(1)
}
func (m *mockCommSvc) GetRoomByID(ctx context.Context, id uint) (*ChatRoom, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ChatRoom), args.Error(1)
}
func (m *mockCommSvc) GetRoomsForUser(ctx context.Context, userID uint) ([]ChatRoomWithMeta, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]ChatRoomWithMeta), args.Error(1)
}
func (m *mockCommSvc) GetActiveConsultations(ctx context.Context, doctorID uint) ([]ChatRoom, error) {
	args := m.Called(ctx, doctorID)
	return args.Get(0).([]ChatRoom), args.Error(1)
}

// ── Chat messages ─────────────────────────────────────────────────────────────

func (m *mockCommSvc) SendMessage(ctx context.Context, roomID, senderID, receiverID uint, message, msgType string, fileURL *string) (*ChatMessage, error) {
	args := m.Called(ctx, roomID, senderID, receiverID, message, msgType, fileURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ChatMessage), args.Error(1)
}
func (m *mockCommSvc) GetChatHistory(ctx context.Context, roomID uint, limit int) ([]ChatMessage, error) {
	args := m.Called(ctx, roomID, limit)
	return args.Get(0).([]ChatMessage), args.Error(1)
}
func (m *mockCommSvc) SearchMessages(ctx context.Context, roomID uint, query string, limit int) ([]ChatMessage, error) {
	args := m.Called(ctx, roomID, query, limit)
	return args.Get(0).([]ChatMessage), args.Error(1)
}
func (m *mockCommSvc) DeleteMessage(ctx context.Context, messageID, callerID uint) error {
	return m.Called(ctx, messageID, callerID).Error(0)
}
func (m *mockCommSvc) GetUnreadCount(ctx context.Context, roomID, userID uint) (int64, error) {
	args := m.Called(ctx, roomID, userID)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockCommSvc) MarkMessagesRead(ctx context.Context, roomID, userID uint) error {
	return m.Called(ctx, roomID, userID).Error(0)
}

// ── Notifications ─────────────────────────────────────────────────────────────

func (m *mockCommSvc) SendNotification(ctx context.Context, userID uint, title, message, nType string) error {
	return m.Called(ctx, userID, title, message, nType).Error(0)
}
func (m *mockCommSvc) LogExternalNotification(ctx context.Context, nType, templateType string, userID uint, appointmentID *uint, status string, errMsg *string) error {
	return m.Called(ctx, nType, templateType, userID, appointmentID, status, errMsg).Error(0)
}
func (m *mockCommSvc) GetUserNotifications(ctx context.Context, userID uint) ([]Notification, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]Notification), args.Error(1)
}
func (m *mockCommSvc) MarkNotificationRead(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockCommSvc) SendEmail(ctx context.Context, to, subject, body string) error {
	return m.Called(ctx, to, subject, body).Error(0)
}

// ─── In-memory SQLite DB for handler tests ────────────────────────────────────

func newTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	db.AutoMigrate(&ChatRoom{}, &career.RecruitmentTemplate{})
	return db
}

// ─── SendMessage tests ────────────────────────────────────────────────────────

func TestSendMessage_SenderIDForcedFromToken(t *testing.T) {
	svc := &mockCommSvc{}
	db := newTestDB(t)
	h := &Handler{svc: svc, db: db}

	// Token says user 42, body claims sender 99 — 42 should win
	svc.On("SendMessage", mock.Anything, uint(1), uint(42), uint(10), "hello", "text", (*string)(nil)).
		Return(&ChatMessage{ID: 1, RoomID: 1, SenderID: 42, Message: "hello"}, nil)

	app := fiber.New()
	app.Post("/chat/messages", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(42)) // token
		return h.SendMessage(c)
	})

	payload := map[string]interface{}{
		"room_id":      1,
		"sender_id":    99, // should be overridden
		"receiver_id":  10,
		"message":      "hello",
		"message_type": "text",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/chat/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	assert.Equal(t, 201, resp.StatusCode)
	// Verify the service was called with senderID=42, not 99
	svc.AssertCalled(t, "SendMessage", mock.Anything, uint(1), uint(42), uint(10), "hello", "text", (*string)(nil))
}

func TestSendMessage_UnauthenticatedReturns401(t *testing.T) {
	svc := &mockCommSvc{}
	db := newTestDB(t)
	h := &Handler{svc: svc, db: db}

	app := fiber.New()
	app.Post("/chat/messages", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(0)) // no valid user
		return h.SendMessage(c)
	})

	payload := map[string]interface{}{"room_id": 1, "message": "hi"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/chat/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	assert.Equal(t, 401, resp.StatusCode)
	svc.AssertNotCalled(t, "SendMessage")
}

// ─── GetChatHistory membership tests ─────────────────────────────────────────

func TestGetChatHistory_ParticipantCanAccess(t *testing.T) {
	svc := &mockCommSvc{}
	db := newTestDB(t)
	h := &Handler{svc: svc, db: db}

	// Seed a room with doctor=5, patient=10
	room := ChatRoom{ID: 1, DoctorID: 5, PatientID: 10, ChatType: "consultation", IsActive: true}
	db.Create(&room)

	svc.On("GetChatHistory", mock.Anything, uint(1), 50).Return([]ChatMessage{}, nil)

	app := fiber.New()
	app.Get("/chat/history/:roomID", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(10)) // patient is a participant
		c.Locals("role", "patient")
		return h.GetChatHistory(c)
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/chat/history/1", nil))
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetChatHistory_NonParticipantForbidden(t *testing.T) {
	svc := &mockCommSvc{}
	db := newTestDB(t)
	h := &Handler{svc: svc, db: db}

	// Seed a room with doctor=5, patient=10
	room := ChatRoom{ID: 2, DoctorID: 5, PatientID: 10, ChatType: "consultation", IsActive: true}
	db.Create(&room)

	app := fiber.New()
	app.Get("/chat/history/:roomID", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(99)) // unrelated user
		c.Locals("role", "patient")
		return h.GetChatHistory(c)
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/chat/history/2", nil))
	assert.Equal(t, 403, resp.StatusCode)
	svc.AssertNotCalled(t, "GetChatHistory")
}

func TestGetChatHistory_AdminCanAccess(t *testing.T) {
	svc := &mockCommSvc{}
	db := newTestDB(t)
	h := &Handler{svc: svc, db: db}

	// Seed a room
	room := ChatRoom{ID: 3, DoctorID: 5, PatientID: 10, ChatType: "consultation", IsActive: true}
	db.Create(&room)

	svc.On("GetChatHistory", mock.Anything, uint(3), 50).Return([]ChatMessage{}, nil)

	app := fiber.New()
	app.Get("/chat/history/:roomID", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(1))
		c.Locals("role", "admin") // admin bypasses membership check
		return h.GetChatHistory(c)
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/chat/history/3", nil))
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetChatHistory_RoomNotFoundReturns404(t *testing.T) {
	svc := &mockCommSvc{}
	db := newTestDB(t)
	h := &Handler{svc: svc, db: db}

	app := fiber.New()
	app.Get("/chat/history/:roomID", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(10))
		c.Locals("role", "patient")
		return h.GetChatHistory(c)
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/chat/history/9999", nil))
	assert.Equal(t, 404, resp.StatusCode)
}

// ─── GetOrCreateChatRoom participant validation ───────────────────────────────

func TestGetOrCreateChatRoom_ParticipantCanCreate(t *testing.T) {
	svc := &mockCommSvc{}
	db := newTestDB(t)
	h := &Handler{svc: svc, db: db}

	svc.On("GetOrCreateChatRoom", mock.Anything, uint(5), uint(10), "consultation").
		Return(&ChatRoom{ID: 1, DoctorID: 5, PatientID: 10}, nil)

	app := fiber.New()
	app.Post("/chat/rooms", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(10)) // patient is participant
		c.Locals("role", "patient")
		return h.GetOrCreateChatRoom(c)
	})

	payload := map[string]interface{}{"doctor_id": 5, "patient_id": 10, "chat_type": "consultation"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/chat/rooms", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestGetOrCreateChatRoom_NonParticipantForbidden(t *testing.T) {
	svc := &mockCommSvc{}
	db := newTestDB(t)
	h := &Handler{svc: svc, db: db}

	app := fiber.New()
	app.Post("/chat/rooms", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(99)) // unrelated user
		c.Locals("role", "patient")
		return h.GetOrCreateChatRoom(c)
	})

	payload := map[string]interface{}{"doctor_id": 5, "patient_id": 10, "chat_type": "consultation"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/chat/rooms", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, 403, resp.StatusCode)
	svc.AssertNotCalled(t, "GetOrCreateChatRoom")
}

// ─── MarkRoomRead tests ───────────────────────────────────────────────────────

func TestMarkRoomRead_AuthenticatedCallsService(t *testing.T) {
	svc := &mockCommSvc{}
	db := newTestDB(t)
	h := &Handler{svc: svc, db: db}

	svc.On("MarkMessagesRead", mock.Anything, uint(1), uint(42)).Return(nil)

	app := fiber.New()
	app.Post("/chat/rooms/:roomID/read", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(42))
		return h.MarkRoomRead(c)
	})

	resp, _ := app.Test(httptest.NewRequest("POST", "/chat/rooms/1/read", nil))
	assert.Equal(t, 200, resp.StatusCode)
	svc.AssertCalled(t, "MarkMessagesRead", mock.Anything, uint(1), uint(42))
}

func TestMarkRoomRead_UnauthenticatedReturns401(t *testing.T) {
	svc := &mockCommSvc{}
	db := newTestDB(t)
	h := &Handler{svc: svc, db: db}

	app := fiber.New()
	app.Post("/chat/rooms/:roomID/read", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(0))
		return h.MarkRoomRead(c)
	})

	resp, _ := app.Test(httptest.NewRequest("POST", "/chat/rooms/1/read", nil))
	assert.Equal(t, 401, resp.StatusCode)
	svc.AssertNotCalled(t, "MarkMessagesRead")
}

// ─── DeleteMessage tests ──────────────────────────────────────────────────────

func TestDeleteMessage_SenderCanDelete(t *testing.T) {
	svc := &mockCommSvc{}
	db := newTestDB(t)
	h := &Handler{svc: svc, db: db}

	svc.On("DeleteMessage", mock.Anything, uint(10), uint(42)).Return(nil)

	app := fiber.New()
	app.Delete("/chat/messages/:id", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(42))
		return h.DeleteMessage(c)
	})

	resp, _ := app.Test(httptest.NewRequest("DELETE", "/chat/messages/10", nil))
	assert.Equal(t, 200, resp.StatusCode)
}

func TestDeleteMessage_NonSenderForbidden(t *testing.T) {
	svc := &mockCommSvc{}
	db := newTestDB(t)
	h := &Handler{svc: svc, db: db}

	svc.On("DeleteMessage", mock.Anything, uint(10), uint(99)).
		Return(errors.New("forbidden: you can only delete your own messages"))

	app := fiber.New()
	app.Delete("/chat/messages/:id", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(99))
		return h.DeleteMessage(c)
	})

	resp, _ := app.Test(httptest.NewRequest("DELETE", "/chat/messages/10", nil))
	assert.Equal(t, 403, resp.StatusCode)
}

func TestGetOrCreateChatRoom_MissingIDs(t *testing.T) {
	svc := &mockCommSvc{}
	db := newTestDB(t)
	h := &Handler{svc: svc, db: db}

	app := fiber.New()
	app.Post("/chat/rooms", func(c *fiber.Ctx) error {
		c.Locals("user_id", uint(10))
		c.Locals("role", "patient")
		return h.GetOrCreateChatRoom(c)
	})

	payload := map[string]interface{}{"chat_type": "consultation"} // no IDs
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/chat/rooms", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestNotificationTemplates_CRUD(t *testing.T) {
	svc := &mockCommSvc{}
	db := newTestDB(t)
	h := &Handler{svc: svc, db: db}

	app := fiber.New()
	app.Get("/notifications-enhanced/templates", h.GetNotifications) // wait, the handler routes are registered in RegisterRoutes
	// Let's call h.RegisterRoutes instead or mount the specific endpoints we want to test:
	api := app.Group("/api/v1")
	h.RegisterRoutes(api)

	// 1. Create a template (POST)
	createPayload := map[string]interface{}{
		"name":          "Welcome WhatsApp Alert",
		"channel":       "whatsapp",
		"body_template": "Hello {{name}}, welcome to our platform!",
		"variables":     "name",
	}
	bodyBytes, _ := json.Marshal(createPayload)
	reqPost := httptest.NewRequest("POST", "/api/v1/notifications-enhanced/templates", bytes.NewReader(bodyBytes))
	reqPost.Header.Set("Content-Type", "application/json")
	respPost, _ := app.Test(reqPost)
	assert.Equal(t, 201, respPost.StatusCode)

	var createResp map[string]interface{}
	json.NewDecoder(respPost.Body).Decode(&createResp)
	dataMap := createResp["data"].(map[string]interface{})
	assert.Equal(t, "Welcome WhatsApp Alert", dataMap["name"])
	assert.Equal(t, "whatsapp", dataMap["channel"])
	assert.Equal(t, "Hello {{name}}, welcome to our platform!", dataMap["body_template"])

	templateID := int(dataMap["id"].(float64))

	// 2. Fetch all templates (GET)
	reqGet := httptest.NewRequest("GET", "/api/v1/notifications-enhanced/templates", nil)
	respGet, _ := app.Test(reqGet)
	assert.Equal(t, 200, respGet.StatusCode)

	var getResp map[string]interface{}
	json.NewDecoder(respGet.Body).Decode(&getResp)
	templatesList := getResp["data"].([]interface{})
	assert.Len(t, templatesList, 1)

	// 3. Fetch specific template (GET /:id)
	reqGetSingle := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/notifications-enhanced/templates/%d", templateID), nil)
	respGetSingle, _ := app.Test(reqGetSingle)
	assert.Equal(t, 200, respGetSingle.StatusCode)

	var getSingleResp map[string]interface{}
	json.NewDecoder(respGetSingle.Body).Decode(&getSingleResp)
	singleTemplate := getSingleResp["data"].(map[string]interface{})
	assert.Equal(t, float64(templateID), singleTemplate["id"])

	// 4. Update the template (PUT)
	updatePayload := map[string]interface{}{
		"body_template": "Hello {{name}}, welcome to our updated platform!",
	}
	updateBytes, _ := json.Marshal(updatePayload)
	reqPut := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/notifications-enhanced/templates/%d", templateID), bytes.NewReader(updateBytes))
	reqPut.Header.Set("Content-Type", "application/json")
	respPut, _ := app.Test(reqPut)
	assert.Equal(t, 200, respPut.StatusCode)

	// 5. Delete the template (DELETE)
	reqDelete := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/notifications-enhanced/templates/%d", templateID), nil)
	respDelete, _ := app.Test(reqDelete)
	assert.Equal(t, 200, respDelete.StatusCode)

	// 6. Verify it is deleted
	reqGetAfterDelete := httptest.NewRequest("GET", "/api/v1/notifications-enhanced/templates", nil)
	respGetAfterDelete, _ := app.Test(reqGetAfterDelete)
	var getAfterDeleteResp map[string]interface{}
	json.NewDecoder(respGetAfterDelete.Body).Decode(&getAfterDeleteResp)
	templatesListAfter := getAfterDeleteResp["data"].([]interface{})
	assert.Len(t, templatesListAfter, 0)
}

package automation

import (
	"context"
	"drayya-core/internal/domain/communication"
	"drayya-core/internal/domain/devi"
	"testing"

	"github.com/stretchr/testify/mock"
)

// MockDeviService implements devi.Service for testing.
type MockDeviService struct {
	devi.Service
	mock.Mock
}

func (m *MockDeviService) ProcessLegacy(ctx context.Context, userID, sessionID, message string) string {
	args := m.Called(ctx, userID, sessionID, message)
	return args.String(0)
}

func (m *MockDeviService) Process(ctx context.Context, uc *devi.UserContext, message string) devi.FlowResult {
	args := m.Called(ctx, uc, message)
	return args.Get(0).(devi.FlowResult)
}

func TestProcessMessage_WithMock(t *testing.T) {
	mockEngine := new(MockDeviService)
	svc := NewService(mockEngine)

	ctx := context.Background()
	msg := &communication.ChatMessage{Message: "book appointment"}

	mockEngine.On("ProcessLegacy", ctx, "123", "sess_1", "book appointment").Return("Booking flow started")

	reply := svc.ProcessMessage(ctx, msg, "User", "123", "sess_1")

	if reply != "Booking flow started" {
		t.Errorf("expected 'Booking flow started', got %q", reply)
	}
	mockEngine.AssertExpectations(t)
}

func TestProcessMessageWithContext_WithMock(t *testing.T) {
	mockEngine := new(MockDeviService)
	svc := NewService(mockEngine)

	ctx := context.Background()
	msg := &communication.ChatMessage{Message: "hello"}
	uc := &devi.UserContext{SessionKey: "sess_123"}

	expectedResult := devi.FlowResult{Message: "Hello there!"}
	mockEngine.On("Process", ctx, uc, "hello").Return(expectedResult)

	reply := svc.ProcessMessageWithContext(ctx, msg, uc)

	if reply.Message != "Hello there!" {
		t.Errorf("expected 'Hello there!', got %q", reply.Message)
	}
	mockEngine.AssertExpectations(t)
}

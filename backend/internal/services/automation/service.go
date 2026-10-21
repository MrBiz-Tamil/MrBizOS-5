// Package automation bridges the WebSocket layer with the Devi engine.
// n8n has been removed. All processing is in-process via the devi package.
package automation

import (
	"context"

	"drayya-core/internal/domain/devi"
	"drayya-core/internal/domain/communication"
)

// Service wraps the Devi engine and is held by the WebSocket Hub.
type Service struct {
	engine devi.Service
}

func NewService(engine devi.Service) *Service {
	return &Service{engine: engine}
}

// ProcessMessage runs the Devi engine and returns the bot's response text.
// Called from ws/client.go in a goroutine for every user message in a Devi room.
// Uses ProcessLegacy so callers that only have (userID, sessionID) strings still work.
func (s *Service) ProcessMessage(ctx context.Context, msg *communication.ChatMessage, userName, userID, sessionID string) string {
	if s == nil || s.engine == nil {
		return ""
	}
	return s.engine.ProcessLegacy(ctx, userID, sessionID, msg.Message)
}

// ProcessWhatsApp runs the Devi engine for a message received via WhatsApp.
// It uses phone-based lookup to establish context.
func (s *Service) ProcessWhatsApp(ctx context.Context, phone, sessionID, message string) string {
	if s == nil || s.engine == nil {
		return ""
	}
	return s.engine.ProcessWhatsApp(ctx, phone, sessionID, message)
}

// ProcessMessageWithContext runs the Devi engine with a fully-scoped UserContext.
// Preferred over ProcessMessage for new call-sites that have participant type info.
func (s *Service) ProcessMessageWithContext(ctx context.Context, msg *communication.ChatMessage, uc *devi.UserContext) *devi.FlowResult {
	if s == nil || s.engine == nil {
		return &devi.FlowResult{Message: ""}
	}
	res := s.engine.Process(ctx, uc, msg.Message)
	return &res
}

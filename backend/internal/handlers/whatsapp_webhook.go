package handlers

// WhatsApp Cloud API Webhook
//
// Meta sends two kinds of events to this endpoint:
//  1. GET  /webhooks/whatsapp — one-time verification challenge during app setup
//  2. POST /webhooks/whatsapp — delivery status updates + incoming user messages
//
// Env vars:
//   WHATSAPP_VERIFY_TOKEN — must match the token you entered in Meta dashboard
//   WHATSAPP_APP_SECRET   — used to validate X-Hub-Signature-256 on inbound events
//
// Delivery status flow:
//   sent → delivered → read  (or failed)
// Each status update is logged to the communication.NotificationLog table so
// the admin dashboard can show real delivery metrics.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/communication"
	"drayya-core/internal/services/notification"
	notifSvc "drayya-core/internal/services/notification"

	"github.com/gofiber/fiber/v2"
)

// ─── Webhook payload types ────────────────────────────────────────────────────

type waWebhookEnvelope struct {
	Object string    `json:"object"`
	Entry  []waEntry `json:"entry"`
}

type waEntry struct {
	ID      string    `json:"id"`
	Changes []waChange `json:"changes"`
}

type waChange struct {
	Value waChangeValue `json:"value"`
	Field string        `json:"field"`
}

type waChangeValue struct {
	MessagingProduct string      `json:"messaging_product"`
	Metadata         waMetadata  `json:"metadata"`
	Contacts         []waContact `json:"contacts,omitempty"`
	Messages         []waInMsg   `json:"messages,omitempty"`
	Statuses         []waStatus  `json:"statuses,omitempty"`
	Errors           []waError   `json:"errors,omitempty"`
}

type waMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type waContact struct {
	Profile waProfile `json:"profile"`
	WaID    string    `json:"wa_id"`
}

type waProfile struct {
	Name string `json:"name"`
}

// waInMsg is an inbound message from a user.
type waInMsg struct {
	From      string  `json:"from"`      // sender's phone (E.164 without +)
	ID        string  `json:"id"`        // wamid.xxx
	Timestamp string  `json:"timestamp"` // unix epoch string
	Type      string  `json:"type"`      // text | image | audio | ...
	Text      *waInText `json:"text,omitempty"`
}

type waInText struct {
	Body string `json:"body"`
}

// waStatus is a delivery/read receipt.
type waStatus struct {
	ID           string         `json:"id"`           // wamid.xxx
	RecipientID  string         `json:"recipient_id"` // phone number
	Status       string         `json:"status"`       // sent | delivered | read | failed
	Timestamp    string         `json:"timestamp"`
	Errors       []waError      `json:"errors,omitempty"`
	Conversation *waConversation `json:"conversation,omitempty"`
}

type waConversation struct {
	ID     string `json:"id"`
	Origin struct {
		Type string `json:"type"` // business_initiated | user_initiated | referral_conversion
	} `json:"origin"`
}

type waError struct {
	Code    int    `json:"code"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

// ─── Handler ──────────────────────────────────────────────────────────────────

// WAWebhookVerify handles Meta's one-time GET challenge for webhook subscription.
// GET /webhooks/whatsapp
func WAWebhookVerify(c *fiber.Ctx) error {
	mode      := c.Query("hub.mode")
	token     := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if mode != "subscribe" {
		return c.Status(fiber.StatusBadRequest).SendString("invalid hub.mode")
	}

	expected := notification.WAVerifyToken()
	if expected == "" || token != expected {
		log.Printf("[WhatsApp Webhook] Verification failed: token mismatch (got %q)", token)
		return c.Status(fiber.StatusForbidden).SendString("verification failed")
	}

	log.Println("[WhatsApp Webhook] Webhook verified successfully")
	return c.Status(fiber.StatusOK).SendString(challenge)
}

// WAWebhookReceive processes incoming Meta webhook POST events.
// POST /webhooks/whatsapp
func WAWebhookReceive(c *fiber.Ctx) error {
	// Validate X-Hub-Signature-256 when app secret is configured.
	// Meta signs the raw body with HMAC-SHA256 using WHATSAPP_APP_SECRET.
	if appSecret := os.Getenv("WHATSAPP_APP_SECRET"); appSecret != "" {
		sigHeader := c.Get("X-Hub-Signature-256") // "sha256=<hex>"
		if sigHeader == "" {
			log.Printf("[WhatsApp Webhook] Missing X-Hub-Signature-256 — rejecting")
			return c.SendStatus(fiber.StatusForbidden)
		}
		// Strip "sha256=" prefix
		sigHex := strings.TrimPrefix(sigHeader, "sha256=")
		mac := hmac.New(sha256.New, []byte(appSecret))
		mac.Write(c.Body())
		expected := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(expected), []byte(sigHex)) {
			log.Printf("[WhatsApp Webhook] Signature mismatch — rejecting (got %s, want %s)", sigHex, expected)
			return c.SendStatus(fiber.StatusForbidden)
		}
	}

	// Always acknowledge immediately — Meta retries if we don't return 200 fast
	var envelope waWebhookEnvelope
	if err := json.Unmarshal(c.Body(), &envelope); err != nil {
		log.Printf("[WhatsApp Webhook] Bad payload: %v", err)
		return c.SendStatus(fiber.StatusOK) // still 200 so Meta doesn't retry garbage
	}

	if envelope.Object != "whatsapp_business_account" {
		return c.SendStatus(fiber.StatusOK)
	}

	for _, entry := range envelope.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}
			val := change.Value
			for _, status := range val.Statuses {
				go handleDeliveryStatus(status)
			}
			for _, msg := range val.Messages {
				go handleIncomingMessage(msg, val.Metadata.PhoneNumberID)
			}
		}
	}

	return c.SendStatus(fiber.StatusOK)
}

// RegisterWhatsAppWebhookRoutes mounts the two webhook endpoints.
// These are public — Meta sends unauthenticated requests.
func RegisterWhatsAppWebhookRoutes(app fiber.Router) {
	app.Get("/webhooks/whatsapp", WAWebhookVerify)
	app.Post("/webhooks/whatsapp", WAWebhookReceive)
}

// ─── Event processors (run in goroutines) ────────────────────────────────────

func handleDeliveryStatus(s waStatus) {
	db := database.DB
	if db == nil {
		return
	}

	var errMsg *string
	if len(s.Errors) > 0 {
		msg := s.Errors[0].Title
		errMsg = &msg
	}

	logEntry := communication.NotificationLog{
		NotificationType: "whatsapp",
		TemplateType:     "delivery_status",
		Status:           s.Status, // sent | delivered | read | failed
		ErrorMessage:     errMsg,
		SentAt:           time.Now(),
	}

	if err := db.Create(&logEntry).Error; err != nil {
		log.Printf("[WhatsApp Webhook] Failed to log delivery status: %v", err)
		return
	}

	if s.Status == "failed" {
		errStr := ""
		if errMsg != nil {
			errStr = *errMsg
		}
		log.Printf("[WhatsApp Webhook] Message %s FAILED to %s: %s", s.ID, s.RecipientID, errStr)
	} else {
		log.Printf("[WhatsApp Webhook] Message %s %s (recipient: %s)", s.ID, s.Status, s.RecipientID)
	}
}

func handleIncomingMessage(msg waInMsg, phoneNumberID string) {
	body := ""
	if msg.Text != nil {
		body = msg.Text.Body
	}
	if body == "" {
		// Non-text message (image, audio, etc.) — acknowledge but skip bot processing
		log.Printf("[WhatsApp Webhook] Inbound non-text from %s (type=%s) — skipping", msg.From, msg.Type)
		return
	}

	log.Printf("[WhatsApp Webhook] Inbound from %s: %q", msg.From, body)

	// Log the inbound message for audit
	db := database.DB
	if db != nil {
		logEntry := communication.NotificationLog{
			NotificationType: "whatsapp_inbound",
			TemplateType:     "user_reply",
			Status:           "received",
			SentAt:           time.Now(),
		}
		if err := db.Create(&logEntry).Error; err != nil {
			log.Printf("[WhatsApp Webhook] Failed to log inbound message: %v", err)
		}
	}

	// Route through Devi engine if the automation handler is wired up
	if waDeviProcessor != nil {
		sessionID := "whatsapp-" + msg.From
		inMsg := &communication.ChatMessage{Message: body}
		reply := waDeviProcessor(inMsg, msg.From, sessionID)
		if reply == "" {
			return
		}
		// Send reply back via Meta Graph API
		if err := notifSvc.SendText(msg.From, reply); err != nil {
			log.Printf("[WhatsApp Webhook] Failed to send reply to %s: %v", msg.From, err)
		} else {
			log.Printf("[WhatsApp Webhook] Replied to %s: %q", msg.From, reply)
		}
	}
}

// waDeviProcessor is set by the automation package to inject the Devi engine
// without creating an import cycle. Set via RegisterWADeviProcessor.
var waDeviProcessor func(msg *communication.ChatMessage, phone, sessionID string) string

// RegisterWADeviProcessor allows the automation handler to provide its
// ProcessMessage callback to the webhook handler.
func RegisterWADeviProcessor(fn func(msg *communication.ChatMessage, phone, sessionID string) string) {
	waDeviProcessor = fn
}

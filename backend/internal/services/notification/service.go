// Package notification provides a unified, reliable outbox-backed dispatcher
// for all outbound notifications in Dr.Ayya.
//
// Architecture summary
// ────────────────────
//
//	┌─────────────────────────────────────────────────────────────────┐
//	│  Transactional (must not be lost)        Conversational         │
//	│  ─────────────────────────────────────   ─────────────────────  │
//	│  email OTP, WhatsApp OTP template    →   Devi free-text reply   │
//	│  welcome, appointment confirmed …        (fire-and-forget OK)   │
//	│                                                                  │
//	│  EnqueueXxx()  →  notification_jobs  →  worker (retry+backoff)  │
//	│                                          SendDeviReply() goroutine│
//	└─────────────────────────────────────────────────────────────────┘
//
// Usage
// ─────
//  svc := notification.NewService(db)
//  svc.AutoMigrate()                                  // once at startup
//  notification.StartWorker(ctx, db, svc.Senders(), 0) // background goroutine
//  svc.EnqueueMFAOTP(ctx, email, phone, name, otp)   // called by identity svc
package notification

import (
	"context"
	"log"

	emailSvc "drayya-core/internal/services/email"

	"gorm.io/gorm"
)

// pkg is the package-level singleton set by Init.
// All package-level helper wrappers (EnqueueEmail, etc.) use it.
// nil = outbox not ready; callers fall back to direct fire-and-forget.
var pkg *Service

// Init registers the shared Service instance used by package-level helpers
// such as EnqueueEmail and the upgraded WhatsApp* wrappers in whatsapp.go.
// Call this once from server/app.go after the Service is constructed.
func Init(svc *Service) { pkg = svc }

// Service is the unified notification dispatcher.
// Construct with NewService; call AutoMigrate once; then start the worker.
type Service struct {
	db *gorm.DB
}

// NewService returns a Service backed by the given GORM database connection.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// AutoMigrate creates or updates the notification_jobs table in Postgres.
// Call this from server/app.go RunMigrations.
func (s *Service) AutoMigrate() error {
	return AutoMigrateOutbox(s.db)
}

// Senders returns the channel-to-sender map expected by StartWorker.
func (s *Service) Senders() map[Channel]Sender {
	return map[Channel]Sender{
		ChannelEmail:    NewEmailSender(),
		ChannelWhatsApp: NewWhatsAppTemplateSender(),
	}
}

// ── Transactional enqueue methods ────────────────────────────────────────────

// EnqueueMFAOTP enqueues both an email OTP and (if phone is non-empty) a
// WhatsApp OTP template message for a staff MFA login event.
// The worker delivers them with automatic retry on failure.
func (s *Service) EnqueueMFAOTP(ctx context.Context, recipientEmail, phone, name, otp string) error {
	// Email leg — always required
	if err := enqueue(ctx, s.db, &NotificationJob{
		Channel:     ChannelEmail,
		JobType:     "mfa_otp",
		Recipient:   recipientEmail,
		Subject:     "Your Dr.Ayya Login Code",
		Body:        emailSvc.StaffMFAEmail(name, otp),
		MaxAttempts: 3,
	}); err != nil {
		return err
	}

	// WhatsApp leg — optional; a missing phone is not an error
	if phone != "" {
		if err := enqueue(ctx, s.db, &NotificationJob{
			Channel:     ChannelWhatsApp,
			JobType:     "mfa_otp",
			Recipient:   phone,
			TemplateID:  "drayya_otp",
			Params:      SliceToParams([]string{otp}),
			MaxAttempts: 3,
		}); err != nil {
			// Non-fatal: email leg already queued; log and continue
			log.Printf("[notif] EnqueueMFAOTP: failed to queue WhatsApp leg for %s: %v", phone, err)
		}
	}
	log.Printf("[notif] EnqueueMFAOTP: successfully queued email (and WhatsApp if phone set) for %s", recipientEmail)
	return nil
}

// EnqueueEmailJob adds a single email delivery to the outbox.
// Prefer the package-level EnqueueEmail wrapper which falls back gracefully.
func (s *Service) EnqueueEmailJob(ctx context.Context, to, subject, body string) error {
	return enqueue(ctx, s.db, &NotificationJob{
		Channel:     ChannelEmail,
		JobType:     "email",
		Recipient:   to,
		Subject:     subject,
		Body:        body,
		MaxAttempts: 3,
	})
}

// enqueueWATemplate adds a WhatsApp template job to the outbox.
// Used internally by the WhatsApp* domain helpers in whatsapp.go.
func (s *Service) enqueueWATemplate(ctx context.Context, phone, jobType, templateID string, params []string) error {
	return enqueue(ctx, s.db, &NotificationJob{
		Channel:     ChannelWhatsApp,
		JobType:     jobType,
		Recipient:   phone,
		TemplateID:  templateID,
		Params:      SliceToParams(params),
		MaxAttempts: 3,
	})
}

// ── Package-level wrappers (used by domain callers that don't hold *Service) ──

// EnqueueEmail enqueues an email through the outbox when Init has been called;
// otherwise delivers fire-and-forget via a goroutine (legacy fallback).
// All call sites may use this without checking whether the outbox is ready.
func EnqueueEmail(ctx context.Context, to, subject, body string) {
	if pkg != nil {
		if err := pkg.EnqueueEmailJob(ctx, to, subject, body); err != nil {
			log.Printf("[notif] EnqueueEmail DB write failed for %s: %v — falling back to direct send", to, err)
			// fall through to goroutine fallback
		} else {
			return
		}
	}
	go func() {
		if err := emailSvc.SendSimple(to, subject, body); err != nil {
			log.Printf("[notif] direct email to %s failed: %v", to, err)
		}
	}()
}

// ── Conversational / Devi path ────────────────────────────────────────────────

// SendDeviReply sends a free-text WhatsApp reply from the Devi chatbot in a
// fire-and-forget goroutine.
//
// Rationale: conversational replies must not be retried.  If Devi replies
// "Hi! How can I help?" and the send fails, queuing the same text 30 minutes
// later would confuse the user.  The 24-h customer-service window also makes
// template semantics inapplicable.  Transient WhatsApp outages are acceptable
// for chat; they are NOT acceptable for OTPs — hence the split design.
func (s *Service) SendDeviReply(phone, message string) {
	go func() {
		if err := SendText(phone, message); err != nil {
			log.Printf("[notif] Devi reply to %s failed (not retried): %v", phone, err)
		}
	}()
}

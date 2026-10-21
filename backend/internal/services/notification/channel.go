package notification

// Channel identifies the delivery transport for a notification job.
type Channel string

const (
	// ChannelEmail delivers via Resend (transactional HTML email).
	ChannelEmail Channel = "email"
	// ChannelWhatsApp delivers via Meta WhatsApp Cloud API template messages.
	// Free-text (conversational) replies are NOT routed through the outbox —
	// they go directly through SendText / SendDeviReply.
	ChannelWhatsApp Channel = "whatsapp"
)

// Sender is implemented by channel-specific adapters (EmailSender,
// WhatsAppTemplateSender).  Send is called by the worker goroutine and must
// be safe for concurrent use.
type Sender interface {
	Send(job *NotificationJob) error
}

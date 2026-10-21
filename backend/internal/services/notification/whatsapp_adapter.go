package notification

// whatsapp_adapter.go — Sender implementation for the WhatsApp template channel.
//
// Only pre-approved Meta Business template messages go through the outbox.
// Conversational free-text replies (Devi chatbot) remain fire-and-forget via
// SendDeviReply and are never retried.

// WhatsAppTemplateSender implements Sender for ChannelWhatsApp.
type WhatsAppTemplateSender struct{}

// NewWhatsAppTemplateSender returns a ready-to-use WhatsAppTemplateSender.
func NewWhatsAppTemplateSender() *WhatsAppTemplateSender { return &WhatsAppTemplateSender{} }

// Send dispatches a WhatsApp template message using the pre-approved template
// stored in job.TemplateID with the body parameters from job.Params.
// Returns a non-nil error on any API or network failure; the worker retries.
func (s *WhatsAppTemplateSender) Send(job *NotificationJob) error {
	params := ParamsToSlice(job.Params)
	_, err := SendTemplate(job.Recipient, job.TemplateID, params)
	return err
}

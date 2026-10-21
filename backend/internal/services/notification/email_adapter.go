package notification

// email_adapter.go — Sender implementation for the email channel.
// Delegates to the Resend-backed email.SendSimple function.

import (
	emailSvc "drayya-core/internal/services/email"
)

// EmailSender implements Sender for ChannelEmail.
type EmailSender struct{}

// NewEmailSender returns a ready-to-use EmailSender.
func NewEmailSender() *EmailSender { return &EmailSender{} }

// Send delivers job.Body as an HTML email to job.Recipient with job.Subject.
// Returns a non-nil error on any API or network failure; the worker will
// retry the job according to the back-off schedule.
func (s *EmailSender) Send(job *NotificationJob) error {
	return emailSvc.SendSimple(job.Recipient, job.Subject, job.Body)
}

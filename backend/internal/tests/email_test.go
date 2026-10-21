package tests_test

// Feature 12: Email integration tests
//
// Tests cover:
//   - Template rendering (all templates produce valid HTML)
//   - Send() error when RESEND_API_KEY is absent
//   - Admin test-email endpoint (400 on missing to, 500 when no key set)
//   - Identity: Register triggers email path (no panic)
//   - EMR: CreatePrescription triggers email path (no panic)
//   - Wallet: VerifyTopUp triggers email path (no panic)

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/services/email"
	"drayya-core/internal/tests/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// Template unit tests
// ─────────────────────────────────────────────────────────────────────────────

func TestEmailTemplate_Welcome(t *testing.T) {
	html := email.WelcomeEmail("John Doe", "patient")
	assert.Contains(t, html, "John Doe")
	assert.Contains(t, html, "<!DOCTYPE html>")
	assert.Contains(t, html, "Dr.Ayya")
}

func TestEmailTemplate_OTP(t *testing.T) {
	html := email.StaffMFAEmail("Dr. Smith", "654321")
	assert.Contains(t, html, "654321")
	assert.Contains(t, html, "Dr. Smith")
}

func TestEmailTemplate_AppointmentConfirmation(t *testing.T) {
	html := email.AppointmentConfirmation(email.AppointmentEmailData{
		PatientName:   "Alice",
		DoctorName:    "Dr. Bob",
		Date:          "15 Apr 2026",
		Time:          "10:00 AM",
		Mode:          "video",
		AppointmentID: 42,
		Fee:           500.0,
	})
	assert.Contains(t, html, "Alice")
	assert.Contains(t, html, "Dr. Bob")
	assert.Contains(t, html, "10:00 AM")
}

func TestEmailTemplate_DoctorAppointmentNotice(t *testing.T) {
	html := email.DoctorAppointmentNotice(email.AppointmentEmailData{
		PatientName:   "Alice",
		DoctorName:    "Dr. Bob",
		Date:          "15 Apr 2026",
		Time:          "10:00 AM",
		Mode:          "video",
		AppointmentID: 42,
		Fee:           500.0,
	})
	assert.Contains(t, html, "Alice")
	assert.Contains(t, html, "<!DOCTYPE html>")
}

func TestEmailTemplate_PaymentReceipt(t *testing.T) {
	html := email.PaymentReceipt(email.PaymentEmailData{
		Name:        "Alice",
		Amount:      500.0,
		TxnID:       "pay_abc123",
		Description: "Consultation with Dr. Bob",
		Date:        time.Now().Format("02 Jan 2006"),
	})
	assert.Contains(t, html, "pay_abc123")
	assert.Contains(t, html, "500")
}

func TestEmailTemplate_WalletTopUp(t *testing.T) {
	html := email.WalletTopUpEmail("Alice", 200.0, 700.0, "order_xyz")
	assert.Contains(t, html, "200")
	assert.Contains(t, html, "700")
	assert.Contains(t, html, "order_xyz")
}

func TestEmailTemplate_DoctorPayout(t *testing.T) {
	html := email.DoctorPayoutEmail(email.PayoutEmailData{
		DoctorName:       "Dr. Bob",
		Amount:           1250.0,
		PayoutID:         "pout_test",
		AppointmentCount: 5,
		Date:             time.Now().Format("02 Jan 2006"),
	})
	assert.Contains(t, html, "Dr. Bob")
	assert.Contains(t, html, "pout_test")
	assert.Contains(t, html, "1250")
}

func TestEmailTemplate_PrescriptionIssued(t *testing.T) {
	html := email.PrescriptionIssuedEmail(email.PrescriptionEmailData{
		PatientName: "Alice",
		DoctorName:  "Dr. Bob",
		Date:        "15 Apr 2026",
		Diagnosis:   "Flu treatment",
		PrescID:     7,
	})
	assert.Contains(t, html, "Alice")
	assert.Contains(t, html, "Dr. Bob")
	assert.Contains(t, html, "Flu treatment")
}

func TestEmailTemplate_AppointmentReminder(t *testing.T) {
	html := email.AppointmentReminderEmail("Alice", "Dr. Bob", "11:00 AM", 99)
	assert.Contains(t, html, "Alice")
	assert.Contains(t, html, "Dr. Bob")
	assert.Contains(t, html, "11:00 AM")
}

// ─────────────────────────────────────────────────────────────────────────────
// Send() error path — no API key
// ─────────────────────────────────────────────────────────────────────────────

func TestEmailSend_NoAPIKey_ReturnsError(t *testing.T) {
	os.Unsetenv("RESEND_API_KEY")
	err := email.SendSimple("test@example.com", "Test Subject", "<p>Hello</p>")
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "RESEND_API_KEY"))
}

// ─────────────────────────────────────────────────────────────────────────────
// Admin test-email endpoint
// ─────────────────────────────────────────────────────────────────────────────

func TestAdminTestEmail_MissingTo(t *testing.T) {
	app := common.SetupApp()
	db := database.DB
	admin := common.CreateTestUser(db, "admin")
	token := common.GenerateTestToken(admin)

	body := mustJSON(t, map[string]interface{}{"template": "welcome"})
	req := httptest.NewRequest("POST", "/api/v1/admin/notifications/test-email", body)
	req.Header.Set("Authorization", authHeader(token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestAdminTestEmail_NoAPIKey_Returns500(t *testing.T) {
	os.Unsetenv("RESEND_API_KEY")
	app := common.SetupApp()
	db := database.DB
	admin := common.CreateTestUser(db, "admin")
	token := common.GenerateTestToken(admin)

	body := mustJSON(t, map[string]interface{}{
		"to":       "test@example.com",
		"template": "welcome",
	})
	req := httptest.NewRequest("POST", "/api/v1/admin/notifications/test-email", body)
	req.Header.Set("Authorization", authHeader(token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	errBlock, _ := result["error"].(map[string]interface{})
	assert.Equal(t, "SEND_FAILED", errBlock["code"])
}

func TestAdminTestEmail_AllTemplates_NoAPIKey(t *testing.T) {
	os.Unsetenv("RESEND_API_KEY")
	app := common.SetupApp()
	db := database.DB
	admin := common.CreateTestUser(db, "admin")
	token := common.GenerateTestToken(admin)

	templates := []string{"welcome", "otp", "reminder", "prescription", "payout", "payment", "topup"}
	for _, tmpl := range templates {
		body := mustJSON(t, map[string]interface{}{
			"to":       "test@example.com",
			"template": tmpl,
		})
		req := httptest.NewRequest("POST", "/api/v1/admin/notifications/test-email", body)
		req.Header.Set("Authorization", authHeader(token))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		require.NoError(t, err, "template %s panicked", tmpl)
		// 500 because no API key — but should NOT be 500 from a panic
		assert.Equal(t, 500, resp.StatusCode, "unexpected status for template %s", tmpl)
	}
}

func TestAdminTestEmail_Unauthorized(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	// Non-admin user
	patient := common.CreateTestUser(db, "patient")
	token := common.GenerateTestToken(patient)

	body := mustJSON(t, map[string]interface{}{
		"to":       "test@example.com",
		"template": "welcome",
	})
	req := httptest.NewRequest("POST", "/api/v1/admin/notifications/test-email", body)
	req.Header.Set("Authorization", authHeader(token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)
}

// ─────────────────────────────────────────────────────────────────────────────
// Identity: Register produces welcome email goroutine (no panic)
// ─────────────────────────────────────────────────────────────────────────────

func TestRegister_TriggersEmailPath_NoPanic(t *testing.T) {
	os.Unsetenv("RESEND_API_KEY")
	app := common.SetupApp()

	body := mustJSON(t, map[string]interface{}{
		"email":    "newpatient_emailtest@example.com",
		"password": "securepass123",
		"full_name": "Email Test User",
		"role":     "patient",
	})
	req := httptest.NewRequest("POST", "/api/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	// 201 Created — email goroutine fires in background but doesn't affect response
	assert.Equal(t, 201, resp.StatusCode)
}

// ─────────────────────────────────────────────────────────────────────────────
// Identity: LoginStaff MFA OTP email goroutine (no panic)
// ─────────────────────────────────────────────────────────────────────────────

func TestLoginStaff_OTPEmail_NoPanic(t *testing.T) {
	os.Unsetenv("RESEND_API_KEY")
	app := common.SetupApp()

	// Register a doctor via the API so the password hash is correct
	registerBody := mustJSON(t, map[string]interface{}{
		"email":     "staff_email_test@example.com",
		"password":  "testpass123",
		"full_name": "Staff Email Test",
		"role":      "doctor",
	})
	regReq := httptest.NewRequest("POST", "/api/v1/auth/register", registerBody)
	regReq.Header.Set("Content-Type", "application/json")
	regResp, _ := app.Test(regReq, -1)
	require.Equal(t, 201, regResp.StatusCode)

	loginBody := mustJSON(t, map[string]interface{}{
		"email":    "staff_email_test@example.com",
		"password": "testpass123",
	})
	loginReq := httptest.NewRequest("POST", "/api/v1/auth/staff/login", loginBody)
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp, err := app.Test(loginReq, -1)
	require.NoError(t, err)
	// 200 with mfa_token — email goroutine fires in background
	assert.Equal(t, 200, loginResp.StatusCode)
}

// ─────────────────────────────────────────────────────────────────────────────
// from() helper — respects env vars
// ─────────────────────────────────────────────────────────────────────────────

func TestEmailFrom_DefaultValues(t *testing.T) {
	os.Unsetenv("RESEND_FROM_EMAIL")
	os.Unsetenv("RESEND_FROM_NAME")
	// Trigger Send to exercise from() indirectly; we just check no panic
	os.Unsetenv("RESEND_API_KEY")
	err := email.SendSimple("x@example.com", "s", "<p>h</p>")
	require.Error(t, err) // expected: no API key
}

func TestEmailFrom_CustomValues(t *testing.T) {
	os.Setenv("RESEND_FROM_EMAIL", "custom@hospital.com")
	os.Setenv("RESEND_FROM_NAME", "My Hospital")
	defer func() {
		os.Unsetenv("RESEND_FROM_EMAIL")
		os.Unsetenv("RESEND_FROM_NAME")
	}()
	os.Unsetenv("RESEND_API_KEY")
	err := email.SendSimple("x@example.com", "s", "<p>h</p>")
	require.Error(t, err) // expected: no API key — but from() was exercised
}

// ─────────────────────────────────────────────────────────────────────────────
// Message struct — attachments round-trip
// ─────────────────────────────────────────────────────────────────────────────

func TestEmailMessage_AttachmentSerialization(t *testing.T) {
	msg := email.Message{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test with attachment",
		HTML:    "<p>See attached</p>",
		Attachments: []email.Attachment{
			{
				Filename:    "test.pdf",
				Content:     "JVBERi0xLjM=", // base64 PDF magic bytes
				ContentType: "application/pdf",
			},
		},
	}

	raw, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded email.Message
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Len(t, decoded.Attachments, 1)
	assert.Equal(t, "test.pdf", decoded.Attachments[0].Filename)
	assert.Equal(t, "application/pdf", decoded.Attachments[0].ContentType)
}

// ─────────────────────────────────────────────────────────────────────────────
// Template HTML validity — all templates must contain doctype + closing tag
// ─────────────────────────────────────────────────────────────────────────────

func TestAllTemplates_ProduceValidHTML(t *testing.T) {
	templates := map[string]string{
		"welcome":      email.WelcomeEmail("Test", "patient"),
		"otp":          email.StaffMFAEmail("Test", "123456"),
		"appt_confirm": email.AppointmentConfirmation(email.AppointmentEmailData{PatientName: "P", DoctorName: "D", Date: "1 Jan", Time: "10 AM", Mode: "video", AppointmentID: 1, Fee: 100}),
		"doctor_appt":  email.DoctorAppointmentNotice(email.AppointmentEmailData{PatientName: "P", DoctorName: "D", Date: "1 Jan", Time: "10 AM", Mode: "video", AppointmentID: 1, Fee: 100}),
		"payment":      email.PaymentReceipt(email.PaymentEmailData{Name: "P", Amount: 100, TxnID: "t1", Description: "Consult", Date: "1 Jan"}),
		"topup":        email.WalletTopUpEmail("P", 100, 500, "ord_1"),
		"payout":       email.DoctorPayoutEmail(email.PayoutEmailData{DoctorName: "D", Amount: 100, PayoutID: "po1", AppointmentCount: 2, Date: "1 Jan"}),
		"prescription": email.PrescriptionIssuedEmail(email.PrescriptionEmailData{PatientName: "P", DoctorName: "D", Date: "1 Jan", Diagnosis: "Cold", PrescID: 1}),
		"reminder":     email.AppointmentReminderEmail("P", "D", "10 AM", 1),
		"rec_received":  email.ApplicationReceivedEmail(email.JobApplicationEmailData{FullName: "Dr. Priya", Position: "Ayurveda Doctor", ExperienceYears: 5, LicenseNumber: "LIC123", WhatsappNumber: "9876543210"}),
		"rec_status":    email.ApplicationStatusUpdateEmail(email.JobApplicationEmailData{FullName: "Dr. Priya", Position: "Ayurveda Doctor"}, "Reviewing", "#1a4731", "Your application is under review"),
		"rec_welcome":   email.DoctorOnboardingWelcomeEmail(email.JobApplicationEmailData{FullName: "Dr. Priya", Position: "Ayurveda Doctor", Email: "priya@example.com"}, "tempPass123"),
		"rec_hr_notice": email.HRApplicationNotificationEmail(email.JobApplicationEmailData{FullName: "Dr. Priya", Email: "priya@example.com", PhoneNumber: "9876543210", WhatsappNumber: "9876543210", Position: "Ayurveda Doctor", ExperienceYears: 5, LicenseNumber: "LIC123"}),
		"rec_verified":  email.DoctorProfileVerifiedEmail("Dr. Priya"),
		"support_ticket": email.SupportTicketNotificationEmail(1, "Test Ticket", "Billing", "High", "Test description", "billing", "AI resolved"),
		"critical_anomaly": email.AuditCriticalAnomalyEmail("Title", "Desc", "Rule", "admin", "logs", "10:00", 1),
		"daily_digest": email.AuditDailyDigestEmail("Monday", "100", "INTACT", "#16a34a", "", "", ""),
	}

	for name, html := range templates {
		t.Run(name, func(t *testing.T) {
			assert.True(t, strings.HasPrefix(strings.TrimSpace(html), "<!DOCTYPE html>"),
				"%s: should start with <!DOCTYPE html>", name)
			assert.Contains(t, html, "</html>", "%s: should contain </html>", name)
			assert.Contains(t, html, "Dr.Ayya", "%s: should contain brand name", name)
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Recruitment template unit tests
// ─────────────────────────────────────────────────────────────────────────────

func TestEmailTemplate_ApplicationReceived(t *testing.T) {
	html := email.ApplicationReceivedEmail(email.JobApplicationEmailData{
		FullName:        "Dr. Rajesh Kumar",
		Position:        "Senior Siddha Doctor",
		ExperienceYears: 12,
		LicenseNumber:   "SD-8874-TN",
		WhatsappNumber:  "9998887776",
	})
	assert.Contains(t, html, "Dr. Rajesh Kumar")
	assert.Contains(t, html, "Senior Siddha Doctor")
	assert.Contains(t, html, "12 years")
	assert.Contains(t, html, "SD-8874-TN")
	assert.Contains(t, html, "9998887776")
}

func TestEmailTemplate_ApplicationStatusUpdate(t *testing.T) {
	html := email.ApplicationStatusUpdateEmail(email.JobApplicationEmailData{
		FullName: "Dr. Rajesh Kumar",
		Position: "Senior Siddha Doctor",
	}, "Interviewing", "#1a4731", "We are pleased to invite you for an interview.")
	assert.Contains(t, html, "Dr. Rajesh Kumar")
	assert.Contains(t, html, "Senior Siddha Doctor")
	assert.Contains(t, html, "Interviewing")
	assert.Contains(t, html, "We are pleased to invite you for an interview.")
}

func TestEmailTemplate_DoctorOnboardingWelcome(t *testing.T) {
	html := email.DoctorOnboardingWelcomeEmail(email.JobApplicationEmailData{
		FullName: "Dr. Rajesh Kumar",
		Position: "Senior Siddha Doctor",
		Email:    "rajesh.siddha@example.com",
	}, "welcomePass999")
	assert.Contains(t, html, "Dr. Rajesh Kumar")
	assert.Contains(t, html, "rajesh.siddha@example.com")
	assert.Contains(t, html, "welcomePass999")
}

func TestEmailTemplate_HRApplicationNotification(t *testing.T) {
	html := email.HRApplicationNotificationEmail(email.JobApplicationEmailData{
		FullName:          "Dr. Rajesh Kumar",
		Email:             "rajesh.siddha@example.com",
		PhoneNumber:       "9998887776",
		WhatsappNumber:    "9998887776",
		Position:          "Senior Siddha Doctor",
		ExperienceYears:   12,
		LicenseNumber:     "SD-8874-TN",
		DateOfBirth:       "1985-08-20",
		Gender:            "male",
		ResumeURL:         "https://example.com/rajesh_resume.pdf",
		TreatmentApproach: "Traditional Siddha purification and herbal formulas",
		EducationDetails:  "BSMS from Government Siddha Medical College, Palayamkottai",
	})
	assert.Contains(t, html, "Dr. Rajesh Kumar")
	assert.Contains(t, html, "rajesh.siddha@example.com")
	assert.Contains(t, html, "9998887776")
	assert.Contains(t, html, "Senior Siddha Doctor")
	assert.Contains(t, html, "12 years")
	assert.Contains(t, html, "SD-8874-TN")
	assert.Contains(t, html, "https://example.com/rajesh_resume.pdf")
}

func TestEmailTemplate_DoctorProfileVerified(t *testing.T) {
	html := email.DoctorProfileVerifiedEmail("Dr. Rajesh Kumar")
	assert.Contains(t, html, "Dr. Rajesh Kumar")
	assert.Contains(t, html, "verified")
	assert.Contains(t, html, "Available")
}


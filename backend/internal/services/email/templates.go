package email

import (
	"fmt"
	"strings"
	"time"
	"drayya-core/internal/database"
)

type dbTemplate struct {
	ID      uint   `gorm:"primaryKey"`
	Type    string `gorm:"not null"`
	Title   string `gorm:"not null"`
	Content string `gorm:"type:text;not null"`
}

func (dbTemplate) TableName() string {
	return "recruitment_templates"
}

func getDBTemplate(title string, fallback string, vars map[string]string) string {
	var tmpl dbTemplate
	if database.DB != nil {
		if err := database.DB.Where("type = ? AND title = ?", "Email", title).First(&tmpl).Error; err == nil && tmpl.Content != "" {
			content := tmpl.Content
			for k, v := range vars {
				content = strings.ReplaceAll(content, "{{"+k+"}}", v)
			}
			return layout(title, "", content)
		}
	}
	for k, v := range vars {
		fallback = strings.ReplaceAll(fallback, "{{"+k+"}}", v)
	}
	return fallback
}

type JobApplicationEmailData struct {
	FullName          string
	Email             string
	PhoneNumber       string
	WhatsappNumber    string
	Position          string
	ExperienceYears   int
	LicenseNumber     string
	DateOfBirth       string
	Gender            string
	ResumeURL         string
	TreatmentApproach string
	EducationDetails  string
}

func ApplicationReceivedEmail(d JobApplicationEmailData) string {
	body := `
<div style="font-family:sans-serif;max-width:560px;margin:0 auto;padding:32px;background:#f9fafb;border-radius:12px">
  <div style="background-color:#059669;background:linear-gradient(135deg,#059669,#0d9488);padding:24px;border-radius:8px;text-align:center;margin-bottom:24px">
    <h1 style="color:#fff;margin:0;font-size:22px">Application Received ✅</h1>
    <p style="color:rgba(255,255,255,.8);margin:8px 0 0;font-size:14px">Dr.Ayya / Dr.Ayya Recruitment</p>
  </div>
  <p style="color:#374151;font-size:15px;line-height:1.6">Dear <strong>{{name}}</strong>,</p>
  <p style="color:#374151;font-size:15px;line-height:1.6">
    Thank you for applying for the <strong>{{position}}</strong> position at Dr.Ayya.
    Your application has been received and is currently under review by our recruitment team.
  </p>
  <div style="background:#fff;border:1px solid #e5e7eb;border-radius:8px;padding:16px;margin:20px 0">
    <p style="margin:0 0 8px;font-size:13px;color:#6b7280;font-weight:600;text-transform:uppercase;letter-spacing:.5px">Application Summary</p>
    <table style="width:100%;font-size:14px;color:#374151">
      <tr><td style="padding:4px 0;color:#6b7280">Position</td><td style="font-weight:600">{{position}}</td></tr>
      <tr><td style="padding:4px 0;color:#6b7280">Experience</td><td style="font-weight:600">{{experience}} years</td></tr>
      <tr><td style="padding:4px 0;color:#6b7280">License No.</td><td style="font-weight:600">{{license}}</td></tr>
    </table>
  </div>
  <p style="color:#374151;font-size:15px;line-height:1.6">
    Our team will contact you via <strong>Email</strong> or <strong>WhatsApp ({{whatsapp}})</strong> for the next steps.
    The process typically takes 3–5 business days.
  </p>
  <p style="color:#6b7280;font-size:13px;margin-top:24px">— Dr.Ayya Recruitment Team</p>
</div>`
	fallback := layout("Application Received", "Your application has been received", body)
	return getDBTemplate("Application Received Confirmation", fallback, map[string]string{
		"name":       d.FullName,
		"position":   d.Position,
		"experience": fmt.Sprintf("%d", d.ExperienceYears),
		"license":    d.LicenseNumber,
		"whatsapp":   d.WhatsappNumber,
	})
}

func ApplicationStatusUpdateEmail(d JobApplicationEmailData, statusLine, statusColor, friendlyMsg string) string {
	body := `
<div style="font-family:sans-serif;max-width:560px;margin:0 auto;padding:32px;background:#f9fafb;border-radius:12px">
  <div style="background-color:#059669;background:linear-gradient(135deg,#059669,#0d9488);padding:24px;border-radius:8px;text-align:center;margin-bottom:24px">
    <h1 style="color:#fff;margin:0;font-size:22px">Application Status Update</h1>
    <p style="color:rgba(255,255,255,.8);margin:8px 0 0;font-size:14px">Dr.Ayya / Dr.Ayya Recruitment</p>
  </div>
  <p style="color:#374151;font-size:15px;line-height:1.6">Dear <strong>{{name}}</strong>,</p>
  <p style="color:#374151;font-size:15px;line-height:1.6">
    We have an update regarding your application for the <strong>{{position}}</strong> position.
  </p>
  <div style="background:#fff;border:1px solid #e5e7eb;border-radius:8px;padding:20px;margin:20px 0;text-align:center">
    <p style="margin:0 0 8px;font-size:13px;color:#6b7280;font-weight:600;text-transform:uppercase;letter-spacing:.5px">Current Status</p>
    <span style="display:inline-block;padding:8px 24px;background:{{status_color}};color:#fff;border-radius:99px;font-size:15px;font-weight:700">{{status_label}}</span>
  </div>
  <p style="color:#374151;font-size:15px;line-height:1.6">{{message}}</p>
  <p style="color:#6b7280;font-size:13px;margin-top:24px">— Dr.Ayya Recruitment Team</p>
</div>`
	fallback := layout("Application Status Update", "Status update regarding your application", body)
	return getDBTemplate("Application Status Update", fallback, map[string]string{
		"name":         d.FullName,
		"position":     d.Position,
		"status_label": statusLine,
		"status_color": statusColor,
		"message":      friendlyMsg,
	})
}

func DoctorOnboardingWelcomeEmail(d JobApplicationEmailData, tempPassword string) string {
	body := `
<div style="font-family:sans-serif;max-width:560px;margin:0 auto;padding:32px;background:#f9fafb;border-radius:12px">
  <div style="background-color:#059669;background:linear-gradient(135deg,#059669,#0d9488);padding:24px;border-radius:8px;text-align:center;margin-bottom:24px">
    <h1 style="color:#fff;margin:0;font-size:22px">Welcome to Dr.Ayya 🎉</h1>
    <p style="color:rgba(255,255,255,.8);margin:8px 0 0;font-size:14px">Your doctor account is ready</p>
  </div>
  <p style="color:#374151;font-size:15px;line-height:1.6">Dear <strong>{{name}}</strong>,</p>
  <p style="color:#374151;font-size:15px;line-height:1.6">
    Your application for the <strong>{{position}}</strong> position has been approved and your account has been created.
    Use the credentials below to log in for the first time.
  </p>
  <div style="background:#fff;border:2px solid #059669;border-radius:8px;padding:20px;margin:20px 0">
    <p style="margin:0 0 12px;font-size:13px;color:#6b7280;font-weight:600;text-transform:uppercase;letter-spacing:.5px">Login Credentials</p>
    <table style="width:100%;font-size:15px;color:#374151">
      <tr><td style="padding:6px 0;color:#6b7280;width:100px">Email</td><td style="font-weight:700">{{email}}</td></tr>
      <tr><td style="padding:6px 0;color:#6b7280">Password</td><td style="font-family:monospace;font-weight:700;font-size:16px;letter-spacing:1px">{{password}}</td></tr>
    </table>
  </div>
  <p style="color:#dc2626;font-size:13px;font-weight:600">⚠ Please change your password immediately after your first login.</p>
  <p style="color:#374151;font-size:15px;line-height:1.6">
    Once logged in, please complete your profile (specialization, consultation fee, availability) and
    await admin verification before you can start accepting appointments.
  </p>
  <p style="color:#6b7280;font-size:13px;margin-top:24px">— Dr.Ayya HR Team</p>
</div>`
	fallback := layout("Welcome to Dr.Ayya", "Your doctor account is ready — let's get started!", body)
	return getDBTemplate("Doctor Onboarding Welcome", fallback, map[string]string{
		"name":     d.FullName,
		"position": d.Position,
		"email":    d.Email,
		"password": tempPassword,
	})
}

func HRApplicationNotificationEmail(d JobApplicationEmailData) string {
	body := `
<div style="font-family:sans-serif;max-width:640px;margin:0 auto;padding:24px">
  <h2 style="color:#1f2937">New Career Application</h2>
  <table style="width:100%;border-collapse:collapse;font-size:14px">
    <tr style="background:#f3f4f6"><td style="padding:8px 12px;font-weight:600;width:160px">Name</td><td style="padding:8px 12px">{{name}}</td></tr>
    <tr><td style="padding:8px 12px;font-weight:600">Email</td><td style="padding:8px 12px">{{email}}</td></tr>
    <tr style="background:#f3f4f6"><td style="padding:8px 12px;font-weight:600">Phone</td><td style="padding:8px 12px">{{phone}}</td></tr>
    <tr><td style="padding:8px 12px;font-weight:600">WhatsApp</td><td style="padding:8px 12px">{{whatsapp}}</td></tr>
    <tr style="background:#f3f4f6"><td style="padding:8px 12px;font-weight:600">Position</td><td style="padding:8px 12px">{{position}}</td></tr>
    <tr><td style="padding:8px 12px;font-weight:600">Experience</td><td style="padding:8px 12px">{{experience}} years</td></tr>
    <tr style="background:#f3f4f6"><td style="padding:8px 12px;font-weight:600">License No.</td><td style="padding:8px 12px">{{license}}</td></tr>
    <tr><td style="padding:8px 12px;font-weight:600">DOB</td><td style="padding:8px 12px">{{dob}}</td></tr>
    <tr style="background:#f3f4f6"><td style="padding:8px 12px;font-weight:600">Gender</td><td style="padding:8px 12px">{{gender}}</td></tr>
    <tr><td style="padding:8px 12px;font-weight:600">Resume</td><td style="padding:8px 12px"><a href="{{resume}}">View Resume</a></td></tr>
    <tr style="background:#f3f4f6"><td style="padding:8px 12px;font-weight:600">Treatment Approach</td><td style="padding:8px 12px">{{approach}}</td></tr>
    <tr><td style="padding:8px 12px;font-weight:600">Education</td><td style="padding:8px 12px"><pre style="font-size:12px;margin:0">{{education}}</pre></td></tr>
  </table>
</div>`
	fallback := layout("New Application Received", "New job application alert", body)
	return getDBTemplate("HR Application Notification", fallback, map[string]string{
		"name":       d.FullName,
		"email":      d.Email,
		"phone":      d.PhoneNumber,
		"whatsapp":   d.WhatsappNumber,
		"position":   d.Position,
		"experience": fmt.Sprintf("%d", d.ExperienceYears),
		"license":    d.LicenseNumber,
		"dob":        d.DateOfBirth,
		"gender":     d.Gender,
		"resume":     d.ResumeURL,
		"approach":   d.TreatmentApproach,
		"education":  d.EducationDetails,
	})
}

func DoctorProfileVerifiedEmail(fullName string) string {
	body := `
<div style="font-family:sans-serif;max-width:560px;margin:0 auto;padding:32px;background:#f9fafb;border-radius:12px">
  <div style="background-color:#059669;background:linear-gradient(135deg,#059669,#0d9488);padding:24px;border-radius:8px;text-align:center;margin-bottom:24px">
    <h1 style="color:#fff;margin:0;font-size:22px">Profile Verified ✅</h1>
    <p style="color:rgba(255,255,255,.8);margin:8px 0 0;font-size:14px">Dr.Ayya Medical Platform</p>
  </div>
  <p style="color:#374151;font-size:15px;line-height:1.6">Dear <strong>{{name}}</strong>,</p>
  <p style="color:#374151;font-size:15px;line-height:1.6">
    Great news! Your doctor profile on Dr.Ayya has been reviewed and <strong>verified</strong> by our admin team.
  </p>
  <div style="background:#fff;border:1px solid #d1fae5;border-radius:8px;padding:16px;margin:20px 0;text-align:center">
    <p style="margin:0;font-size:15px;color:#059669;font-weight:700">You can now go online and accept patient appointments.</p>
  </div>
  <p style="color:#374151;font-size:15px;line-height:1.6">
    Log in to your dashboard, set your availability, and toggle your status to <strong>Available</strong> to start receiving consultations.
  </p>
  <p style="color:#6b7280;font-size:13px;margin-top:24px">— Dr.Ayya Admin Team</p>
</div>`
	fallback := layout("Profile Verified", "Your doctor profile is verified", body)
	return getDBTemplate("Doctor Profile Verified", fallback, map[string]string{
		"name": fullName,
	})
}

// ─── shared layout ────────────────────────────────────────────────────────────

// WrapInLayout wraps a partial HTML body with the full Dr.Ayya branded email shell.
// This is the public equivalent of the internal layout() function, intended for
// use by the admin email-template preview/send-test handlers.
func WrapInLayout(title, preheader, bodyHTML string) string {
	return layout(title, preheader, bodyHTML)
}

func layout(title, preheader, bodyHTML string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>%s</title>
<style>
  body { font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background-color: #f4f1ea; margin: 0; padding: 0; -webkit-font-smoothing: antialiased; }
  .wrapper { width: 100%%; table-layout: fixed; background-color: #f4f1ea; padding-bottom: 40px; }
  .main { background-color: #ffffff; margin: 0 auto; width: 100%%; max-width: 600px; border-spacing: 0; color: #1e293b; border-radius: 16px; overflow: hidden; margin-top: 40px; box-shadow: 0 4px 12px rgba(26, 71, 49, 0.08), 0 2px 4px rgba(0, 0, 0, 0.04); border: 1px solid #e6e2d8; }
  .header { background-color: #1a4731; background: linear-gradient(135deg, #1a4731 0%%, #123825 100%%); padding: 40px 32px; text-align: center; }
  .header h1 { color: #ffffff; margin: 0; font-size: 28px; font-weight: 800; letter-spacing: -0.025em; }
  .header p { color: rgba(255, 255, 255, 0.85); margin: 8px 0 0; font-size: 14px; font-weight: 500; }
  .content { padding: 40px 32px; line-height: 1.6; }
  .content h2 { margin-top: 0; color: #1a4731; font-size: 20px; font-weight: 700; }
  .content p { margin-bottom: 20px; color: #475569; font-size: 16px; }
  .card { background-color: #fcfbfa; border-radius: 12px; padding: 24px; margin: 24px 0; border: 1px solid #e6e2d8; }
  .card-item { display: flex; justify-content: space-between; margin-bottom: 12px; font-size: 14px; }
  .card-item:last-child { margin-bottom: 0; }
  .label { color: #64748b; font-weight: 500; }
  .value { color: #1e293b; font-weight: 700; }
  .otp-container { background: #fdfbf7; border: 2px dashed #d4af37; border-radius: 12px; padding: 32px; text-align: center; margin: 32px 0; }
  .otp-code { font-size: 42px; font-weight: 800; color: #1a4731; letter-spacing: 12px; margin: 0; font-family: 'Courier New', Courier, monospace; }
  .btn { display: inline-block; background-color: #1a4731; background: linear-gradient(135deg, #1a4731 0%%, #123825 100%%); color: #ffffff !important; text-decoration: none; padding: 16px 32px; border-radius: 12px; font-weight: 700; font-size: 16px; text-align: center; margin: 10px 0; box-shadow: 0 8px 16px -3px rgba(26, 71, 49, 0.3); }
  .footer { text-align: center; padding: 32px; font-size: 13px; color: #94a3b8; }
  .footer a { color: #1a4731; text-decoration: none; font-weight: 600; }
  .badge { display: inline-block; padding: 4px 12px; border-radius: 9999px; font-size: 12px; font-weight: 700; text-transform: uppercase; }
  .badge-success { background-color: #d1fae5; color: #065f46; }
  .badge-info { background-color: #fef3c7; color: #92400e; }
  @media only screen and (max-width: 600px) {
    .main { border-radius: 0; margin-top: 0; }
    .content { padding: 32px 20px; }
  }
</style>
</head>
<body>
  <div style="display:none;max-height:0;overflow:hidden;mso-hide:all;">%s</div>
  <div class="wrapper">
    <table class="main">
      <tr>
        <td class="header">
          <h1>🏥 Dr.Ayya</h1>
          <p>Your Health, Our Priority</p>
        </td>
      </tr>
      <tr>
        <td class="content">
          %s
        </td>
      </tr>
      <tr>
        <td class="footer">
          <p>&copy; %d Dr.Ayya pvt ltd , Kanniya Kumari,TamilNadu, India</p>
          <p>
            <a href="https://medihub.mrbiz.in">Dashboard</a> &nbsp;•&nbsp; 
            <a href="https://medihub.mrbiz.in/privacy">Privacy Policy</a> &nbsp;•&nbsp; 
            <a href="https://medihub.mrbiz.in/support">Support</a>
          </p>
        </td>
      </tr>
    </table>
  </div>
</body>
</html>`, title, preheader, bodyHTML, time.Now().Year())
}

// ─── Welcome / Registration ───────────────────────────────────────────────────

func WelcomeEmail(name, role string) string {
	roleLabel := strings.Title(role)
	body := `
<h2>Welcome to the family, {{name}}! 🎉</h2>
<p>Your <strong>{{role}}</strong> account has been successfully created. We're excited to have you on board with Dr.Ayya.</p>
<p>Here's what you can do right now:</p>
<div class="card">
  <div class="card-item"><span class="label">👨‍⚕️ Digital Consultations</span><span class="badge badge-info">Available 24/7</span></div>
  <div class="card-item"><span class="label">💊 Medicine Delivery</span><span class="badge badge-success">Express</span></div>
  <div class="card-item"><span class="label">🔬 Lab Test Booking</span><span class="badge badge-success">Home Collection</span></div>
</div>
<p>Click the button below to sign in and explore your new health dashboard.</p>
<div style="text-align: center;">
  <a href="https://medihub.mrbiz.in/login" class="btn">Get Started Now</a>
</div>
<p style="margin-top:32px; font-size:14px; color:#94a3b8;">If you did not create this account, please contact our security team immediately.</p>
`
	fallback := layout("Welcome to Dr.Ayya", "Your account is ready — let's get started!", body)
	return getDBTemplate("Welcome Email", fallback, map[string]string{
		"name": name,
		"role": roleLabel,
	})
}

// ─── OTP / MFA ───────────────────────────────────────────────────────────────

func OTPEmail(name, otp string) string {
	body := `
<h2>Security Verification</h2>
<p>Hello <strong>{{name}}</strong>,</p>
<p>To keep your account secure, please use the following One-Time Password (OTP) to complete your action.</p>
<div class="otp-container">
  <h1 class="otp-code">{{otp}}</h1>
</div>
<p style="text-align:center; font-weight:600; color:#4f46e5;">Valid for the next 10 minutes.</p>
<div class="card">
  <div class="card-item"><span class="label">Requested at</span><span class="value">{{time}} IST</span></div>
  <div class="card-item"><span class="label">Action</span><span class="value">Account Login / Verification</span></div>
</div>
<p style="font-size:14px; color:#94a3b8;">Security Note: Never share this code with anyone, including Dr.Ayya staff. If you didn't request this, please ignore this email.</p>
`
	fallback := layout("Your Dr.Ayya OTP", "Your verification code: {{otp}}", body)
	return getDBTemplate("OTP Verification Email", fallback, map[string]string{
		"name": name,
		"otp":  otp,
		"time": time.Now().In(ist()).Format("3:04 PM"),
	})
}

// ─── Password Reset ──────────────────────────────────────────────────────────

func PasswordResetEmail(name, resetURL string) string {
	body := `
<h2>Reset Your Password</h2>
<p>Hello <strong>{{name}}</strong>,</p>
<p>We received a request to reset your Dr.Ayya account password. Click the button below to choose a new one.</p>
<div style="text-align: center; margin: 32px 0;">
  <a href="{{resetURL}}" class="btn">Reset Password</a>
</div>
<p>This link is valid for **30 minutes**. After that, you'll need to request a new one.</p>
<p style="font-size:14px; color:#94a3b8;">If you didn't request a password reset, you can safely ignore this email. Your password will remain unchanged.</p>
`
	fallback := layout("Reset Your Password", "Secure link to reset your Dr.Ayya password", body)
	return getDBTemplate("Password Reset Email", fallback, map[string]string{
		"name":     name,
		"resetURL": resetURL,
	})
}

// ─── Email Verification ──────────────────────────────────────────────────────

func EmailVerificationEmail(name, verifyURL string) string {
	body := `
<h2>Verify Your Email</h2>
<p>Hello <strong>{{name}}</strong>,</p>
<p>Thank you for joining Dr.Ayya! Please verify your email address to activate all features of your account.</p>
<div style="text-align: center; margin: 32px 0;">
  <a href="{{verifyURL}}" class="btn">Verify Email Address</a>
</div>
<p>This link is valid for **24 hours**.</p>
<p style="font-size:14px; color:#94a3b8;">Once verified, you'll be able to book appointments and order medicines instantly.</p>
`
	fallback := layout("Verify Your Email", "Complete your registration by verifying your email", body)
	return getDBTemplate("Email Verification Email", fallback, map[string]string{
		"name":      name,
		"verifyURL": verifyURL,
	})
}

// ─── Appointment Confirmation ─────────────────────────────────────────────────

type AppointmentEmailData struct {
	PatientName   string
	DoctorName    string
	Date          string // e.g. "Mon, 14 Apr 2026"
	Time          string // e.g. "10:30 AM"
	Mode          string // video | chat | in_person
	AppointmentID uint
	Fee           float64
}

func AppointmentConfirmation(d AppointmentEmailData) string {
	modeLabel := map[string]string{
		"video":     "📹 Video Consultation",
		"chat":      "💬 Chat Consultation",
		"in_person": "🏥 In-Person Visit",
	}[d.Mode]
	if modeLabel == "" {
		modeLabel = d.Mode
	}
	body := `
<h2>Appointment Confirmed! ✅</h2>
<p>Hi <strong>{{patientName}}</strong>, your appointment with <strong>Dr. {{doctorName}}</strong> has been successfully booked.</p>
<div class="card">
  <div class="card-item"><span class="label">📅 Date</span><span class="value">{{date}}</span></div>
  <div class="card-item"><span class="label">⏰ Time</span><span class="value">{{time}} IST</span></div>
  <div class="card-item"><span class="label">🖥️ Mode</span><span class="value">{{mode}}</span></div>
  <div class="card-item"><span class="label">💳 Fee Paid</span><span class="value">₹{{fee}}</span></div>
  <div class="card-item"><span class="label">🔖 Booking ID</span><span class="value">APT-{{appointmentID}}</span></div>
</div>
<div style="text-align: center;">
  <a href="https://medihub.mrbiz.in/appointments/{{appointmentID}}" class="btn">View Appointment Details</a>
</div>
<p style="margin-top:32px; font-size:14px; color:#64748b;">A reminder will be sent to you 1 hour before the start time.</p>
`
	fallback := layout("Appointment Confirmed", "Dr. {{doctorName}} on {{date}} at {{time}}", body)
	return getDBTemplate("Appointment Confirmation", fallback, map[string]string{
		"patientName":   d.PatientName,
		"doctorName":    d.DoctorName,
		"date":          d.Date,
		"time":          d.Time,
		"mode":          modeLabel,
		"fee":           fmt.Sprintf("%.0f", d.Fee),
		"appointmentID": fmt.Sprintf("%d", d.AppointmentID),
	})
}

// DoctorAppointmentNotice is sent to the doctor when an appointment is booked.
func DoctorAppointmentNotice(d AppointmentEmailData) string {
	body := `
<h2>New Appointment Scheduled 👨‍⚕️</h2>
<p>Hello Dr. <strong>{{doctorName}}</strong>, you have a new appointment booking.</p>
<div class="card">
  <div class="card-item"><span class="label">🧑‍💼 Patient</span><span class="value">{{patientName}}</span></div>
  <div class="card-item"><span class="label">📅 Date</span><span class="value">{{date}}</span></div>
  <div class="card-item"><span class="label">⏰ Time</span><span class="value">{{time}} IST</span></div>
  <div class="card-item"><span class="label">🖥️ Mode</span><span class="value">{{mode}}</span></div>
</div>
<p>Please review the patient's records on your dashboard before the consultation.</p>
<div style="text-align: center;">
  <a href="https://medihub.mrbiz.in/doctor/appointments/{{appointmentID}}" class="btn">Open Doctor Dashboard</a>
</div>
`
	fallback := layout("New Appointment — Dr.Ayya", "Appointment with {{patientName}} on {{date}}", body)
	return getDBTemplate("Doctor Appointment Notice", fallback, map[string]string{
		"doctorName":    d.DoctorName,
		"patientName":   d.PatientName,
		"date":          d.Date,
		"time":          d.Time,
		"mode":          d.Mode,
		"appointmentID": fmt.Sprintf("%d", d.AppointmentID),
	})
}

// ─── Payouts & Payments ───────────────────────────────────────────────────────

type PaymentEmailData struct {
	Name        string
	Amount      float64
	TxnID       string
	Description string
	Date        string
}

func PaymentReceipt(d PaymentEmailData) string {
	body := `
<h2>Payment Receipt</h2>
<p>Hi <strong>{{name}}</strong>, thank you for your payment. Your transaction has been processed successfully.</p>
<div class="card">
  <div class="card-item"><span class="label">💰 Amount Paid</span><span class="value">₹{{amount}}</span></div>
  <div class="card-item"><span class="label">📋 Description</span><span class="value">{{description}}</span></div>
  <div class="card-item"><span class="label">📅 Date</span><span class="value">{{date}}</span></div>
  <div class="card-item"><span class="label">🔖 Transaction ID</span><span class="value">{{txnID}}</span></div>
  <div class="card-item"><span class="label">✅ Status</span><span class="badge badge-success">Successful</span></div>
</div>
<p style="font-size:14px; color:#94a3b8;">Please keep this email for your records. If you have any questions, reply to this email or contact support.</p>
`
	fallback := layout("Payment Receipt — Dr.Ayya", "₹{{amount}} payment confirmed", body)
	return getDBTemplate("Payment Receipt", fallback, map[string]string{
		"name":        d.Name,
		"amount":      fmt.Sprintf("%.2f", d.Amount),
		"description": d.Description,
		"date":        d.Date,
		"txnID":       d.TxnID,
	})
}

// WalletTopUpEmail is a helper for wallet top-up notifications.
func WalletTopUpEmail(name string, amount float64, newBalance float64, paymentID string) string {
	return PaymentReceipt(PaymentEmailData{
		Name:        name,
		Amount:      amount,
		TxnID:       paymentID,
		Description: "Dr.Ayya Wallet Top-up",
		Date:        time.Now().In(ist()).Format("02 Jan 2006, 3:04 PM"),
	})
}

// DoctorPayoutEmail is sent to doctors when they receive a payout.
type PayoutEmailData struct {
	DoctorName       string
	Amount           float64
	PayoutID         string
	AppointmentCount int
	Date             string
}

func DoctorPayoutEmail(d PayoutEmailData) string {
	body := `
<h2>Earnings Transferred! 💸</h2>
<p>Hi Dr. <strong>{{doctorName}}</strong>, your consultation earnings have been transferred to your registered bank account.</p>
<div class="card">
  <div class="card-item"><span class="label">💰 Amount</span><span class="value">₹{{amount}}</span></div>
  <div class="card-item"><span class="label">📊 Appointments</span><span class="value">{{appointmentCount}}</span></div>
  <div class="card-item"><span class="label">📅 Date</span><span class="value">{{date}}</span></div>
  <div class="card-item"><span class="label">🔖 Payout ID</span><span class="value">{{payoutID}}</span></div>
</div>
<p>The funds should reflect in your account within 24–48 business hours.</p>
`
	fallback := layout("Payout Processed — Dr.Ayya", "Earnings of ₹{{amount}} transferred", body)
	return getDBTemplate("Doctor Payout Email", fallback, map[string]string{
		"doctorName":       d.DoctorName,
		"amount":           fmt.Sprintf("%.2f", d.Amount),
		"appointmentCount": fmt.Sprintf("%d", d.AppointmentCount),
		"date":             d.Date,
		"payoutID":         d.PayoutID,
	})
}

// ─── Prescriptions ───────────────────────────────────────────────────────────

type PrescriptionEmailData struct {
	PatientName string
	DoctorName  string
	Date        string
	Diagnosis   string
	PrescID     uint
}

func PrescriptionIssuedEmail(d PrescriptionEmailData) string {
	body := `
<h2>New Prescription Issued 📋</h2>
<p>Hi <strong>{{patientName}}</strong>, Dr. <strong>{{doctorName}}</strong> has issued a digital prescription for your recent consultation.</p>
<div class="card">
  <div class="card-item"><span class="label">🩺 Diagnosis</span><span class="value">{{diagnosis}}</span></div>
  <div class="card-item"><span class="label">👨‍⚕️ Issued by</span><span class="value">Dr. {{doctorName}}</span></div>
  <div class="card-item"><span class="label">📅 Date</span><span class="value">{{date}}</span></div>
</div>
<p>You can download the PDF prescription directly from the Dr.Ayya mobile app.</p>
<div style="text-align: center;">
  <a href="https://medihub.mrbiz.in/prescriptions" class="btn">View Prescription</a>
</div>
`
	fallback := layout("New Prescription — Dr.Ayya", "Prescription from Dr. {{doctorName}}", body)
	return getDBTemplate("Prescription Issued Email", fallback, map[string]string{
		"patientName": d.PatientName,
		"doctorName":  d.DoctorName,
		"diagnosis":   d.Diagnosis,
		"date":        d.Date,
	})
}

// ─── Staff MFA Convenience ───────────────────────────────────────────────────

func StaffMFAEmail(name, code string) string {
	return OTPEmail(name, code)
}

// AppointmentReminderEmail is sent to patients 1 hour before their appointment.
func AppointmentReminderEmail(patientName, doctorName, timeStr string, apptID uint) string {
	body := `
<h2>Appointment Reminder ⏰</h2>
<p>Hi <strong>{{patientName}}</strong>, this is a friendly reminder that your appointment with <strong>Dr. {{doctorName}}</strong> starts in 1 hour.</p>
<div class="card">
  <div class="card-item"><span class="label">⏰ Time</span><span class="value">{{time}} IST</span></div>
  <div class="card-item"><span class="label">🔖 Booking ID</span><span class="value">APT-{{appointmentID}}</span></div>
</div>
<p>Please ensure you have a stable internet connection if this is a video consultation.</p>
<div style="text-align: center;">
  <a href="https://medihub.mrbiz.in/appointments/{{appointmentID}}" class="btn">Join Consultation</a>
</div>
`
	fallback := layout("Appointment Reminder — Dr.Ayya", "Your appointment with Dr. {{doctorName}} is in 1 hour", body)
	return getDBTemplate("Appointment Reminder Email", fallback, map[string]string{
		"patientName":   patientName,
		"doctorName":    doctorName,
		"time":          timeStr,
		"appointmentID": fmt.Sprintf("%d", apptID),
	})
}

// ─── IST helper ──────────────────────────────────────────────────────────────

func ist() *time.Location {
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return time.UTC
	}
	return loc
}

// ─── IT Provisioning ─────────────────────────────────────────────────────────

func ITProvisioningEmail(employeeID, department, designation, customMessage string) string {
	body := `
<div style="font-family:sans-serif;max-width:560px;margin:0 auto;padding:32px;background:#f9fafb;border-radius:12px">
  <div style="background-color:#1a4731;background:linear-gradient(135deg,#1a4731,#123825);padding:24px;border-radius:8px;text-align:center;margin-bottom:24px">
    <h1 style="color:#d4af37;margin:0;font-size:22px">IT Provisioning Request 🖥️</h1>
  </div>
  <p style="color:#374151;font-size:15px;line-height:1.6">Hello IT Support,</p>
  <p style="color:#374151;font-size:15px;line-height:1.6">Please process the following provisioning request for a new employee:</p>
  
  <div style="background:#fff;border:1px solid #e5e7eb;border-radius:8px;padding:16px;margin:20px 0">
    <table style="width:100%;font-size:14px;color:#374151">
      <tr><td style="padding:4px 0;color:#6b7280">🆔 Employee ID</td><td style="font-weight:600">{{employee_id}}</td></tr>
      <tr><td style="padding:4px 0;color:#6b7280">🏢 Department</td><td style="font-weight:600">{{department}}</td></tr>
      <tr><td style="padding:4px 0;color:#6b7280">👨‍💼 Designation</td><td style="font-weight:600">{{designation}}</td></tr>
    </table>
  </div>
  
  <div style="background:#fff;border:1px solid #e5e7eb;border-left:4px solid #d4af37;border-radius:8px;padding:16px;margin:20px 0;">
    <p style="margin:0;font-size:14px;color:#374151;"><strong>Message from HR:</strong><br/><br/>{{message}}</p>
  </div>
  
  <p style="color:#374151;font-size:15px;line-height:1.6">Please ensure all system accounts (Dr.Ayya portal, Email) are created and configured.</p>
</div>`
	
	fallback := layout("IT Provisioning Request", "New employee provisioning required", body)
	return getDBTemplate("IT Provisioning Request", fallback, map[string]string{
		"employee_id": employeeID,
		"department":  department,
		"designation": designation,
		"message":     customMessage,
	})
}

// ─── HR Onboarding — Employee-Facing Emails ──────────────────────────────────

// OnboardingWelcomeEmail is sent to the new employee when HR initiates their onboarding.
func OnboardingWelcomeEmail(name, employeeID, department, designation, joiningDate string) string {
	body := `
<h2>Welcome to Dr.Ayya! 🎉</h2>
<p>Dear <strong>{{name}}</strong>,</p>
<p>
  We are thrilled to welcome you to the Dr.Ayya family! Your employee profile has been created
  and your onboarding journey begins today.
</p>
<div class="card">
  <div class="card-item"><span class="label">🆔 Employee ID</span><span class="value">{{employee_id}}</span></div>
  <div class="card-item"><span class="label">🏢 Department</span><span class="value">{{department}}</span></div>
  <div class="card-item"><span class="label">👨‍💼 Designation</span><span class="value">{{designation}}</span></div>
  <div class="card-item"><span class="label">📅 Joining Date</span><span class="value">{{joining_date}}</span></div>
</div>
<p><strong>Next steps:</strong></p>
<p>
  Please log in to the Dr.Ayya portal and upload your required documents (Aadhar ID, PAN Card,
  Education Certificate, Bank Account details, and Passport Photo) at the earliest.
</p>
<div style="text-align: center; margin: 32px 0;">
  <a href="{{portal_url}}" class="btn">Access Your Portal</a>
</div>
<p style="font-size:14px; color:#94a3b8;">
  If you have any questions, please reach out to your HR Manager. We look forward to having you on board!
</p>
`
	fallback := layout("Welcome to Dr.Ayya!", "Your employee onboarding has begun", body)
	return getDBTemplate("Onboarding Welcome Email", fallback, map[string]string{
		"name":        name,
		"employee_id": employeeID,
		"department":  department,
		"designation": designation,
		"joining_date": joiningDate,
		"portal_url":  "https://medihub.mrbiz.in/login",
	})
}

// DocumentReminderEmail is sent to remind the employee to upload pending documents.
func DocumentReminderEmail(name, employeeID string, pendingDocs []string, deadline string) string {
	body := `
<h2>Document Submission Reminder 📄</h2>
<p>Dear <strong>{{name}}</strong>,</p>
<p>
  This is a friendly reminder that your onboarding documents are still pending. Please submit
  them as soon as possible to avoid any delay in your joining process.
</p>
<div class="card">
  <div class="card-item"><span class="label">🆔 Employee ID</span><span class="value">{{employee_id}}</span></div>
  <div class="card-item"><span class="label">⏰ Deadline</span><span class="value">{{deadline}}</span></div>
</div>
<p><strong>Pending Documents:</strong></p>
<div style="background:#fef3c7;border:1px solid #fde68a;border-radius:10px;padding:16px;margin:16px 0;">
  <p style="margin:0;font-size:15px;color:#92400e;">{{pending_docs}}</p>
</div>
<div style="text-align: center; margin: 32px 0;">
  <a href="{{portal_url}}" class="btn">Upload Documents Now</a>
</div>
<p style="font-size:14px; color:#94a3b8;">
  If you have already submitted these documents, please ignore this email or contact your HR Manager.
</p>
`
	fallback := layout("Document Submission Reminder", "Pending documents require your attention", body)
	pendingList := strings.Join(pendingDocs, " • ")
	if pendingList == "" {
		pendingList = "Documents pending — please check your portal"
	}
	return getDBTemplate("Onboarding Document Reminder", fallback, map[string]string{
		"name":         name,
		"employee_id":  employeeID,
		"pending_docs": pendingList,
		"deadline":     deadline,
		"portal_url":   "https://medihub.mrbiz.in/login",
	})
}

// DocumentVerifiedEmail is sent when HR marks one of the employee's documents as verified.
func DocumentVerifiedEmail(name, docName, verifiedBy, date string) string {
	body := `
<h2>Document Verified ✅</h2>
<p>Dear <strong>{{name}}</strong>,</p>
<p>
  Great news! One of your onboarding documents has been reviewed and successfully verified by our HR team.
</p>
<div class="card">
  <div class="card-item"><span class="label">📄 Document</span><span class="value">{{doc_name}}</span></div>
  <div class="card-item"><span class="label">✅ Status</span><span class="value" style="color:#059669;font-weight:700;">Verified</span></div>
  <div class="card-item"><span class="label">👤 Verified By</span><span class="value">{{verified_by}}</span></div>
  <div class="card-item"><span class="label">📅 Date</span><span class="value">{{date}}</span></div>
</div>
<p>
  Please continue uploading any remaining documents. You will be notified as each document is verified.
</p>
<p style="font-size:14px; color:#94a3b8;">
  If you believe this verification is incorrect, please contact your HR Manager.
</p>
`
	fallback := layout("Document Verified", "Your {{doc_name}} has been verified", body)
	return getDBTemplate("Onboarding Document Verified", fallback, map[string]string{
		"name":        name,
		"doc_name":    docName,
		"verified_by": verifiedBy,
		"date":        date,
	})
}

// TrainingStartedEmail is sent to the employee when their status moves to "in_training".
func TrainingStartedEmail(name, employeeID, department, designation, trainingDate string) string {
	body := `
<h2>Your Orientation & Training Begins! 🎓</h2>
<p>Dear <strong>{{name}}</strong>,</p>
<p>
  All your documents have been verified and you are now ready to begin your official orientation and training.
  Welcome to the active team!
</p>
<div class="card">
  <div class="card-item"><span class="label">🆔 Employee ID</span><span class="value">{{employee_id}}</span></div>
  <div class="card-item"><span class="label">🏢 Department</span><span class="value">{{department}}</span></div>
  <div class="card-item"><span class="label">👨‍💼 Designation</span><span class="value">{{designation}}</span></div>
  <div class="card-item"><span class="label">📅 Training Start</span><span class="value">{{training_date}}</span></div>
</div>
<p>
  Your IT accounts, system access, and department credentials will be configured shortly by our IT team.
  Please log in to the portal to review your orientation tasks and handbook.
</p>
<div style="text-align: center; margin: 32px 0;">
  <a href="{{portal_url}}" class="btn">Go to Your Dashboard</a>
</div>
<p style="font-size:14px; color:#94a3b8;">
  Reach out to your department head if you need any assistance during the training period.
</p>
`
	fallback := layout("Orientation & Training Started", "Your Dr.Ayya orientation begins today", body)
	return getDBTemplate("Onboarding Training Started", fallback, map[string]string{
		"name":          name,
		"employee_id":   employeeID,
		"department":    department,
		"designation":   designation,
		"training_date": trainingDate,
		"portal_url":    "https://medihub.mrbiz.in/login",
	})
}

// OnboardingCompletedEmail is sent to the employee when their onboarding is fully completed.
func OnboardingCompletedEmail(name, employeeID, department, designation, joiningDate string) string {
	body := `
<h2>Onboarding Complete — Welcome to the Team! 🏅</h2>
<p>Dear <strong>{{name}}</strong>,</p>
<p>
  Congratulations! Your onboarding process is now <strong>fully complete</strong>. You are officially
  a valued member of the Dr.Ayya family. We are excited to have you with us!
</p>
<div class="card">
  <div class="card-item"><span class="label">🆔 Employee ID</span><span class="value">{{employee_id}}</span></div>
  <div class="card-item"><span class="label">🏢 Department</span><span class="value">{{department}}</span></div>
  <div class="card-item"><span class="label">👨‍💼 Role</span><span class="value">{{designation}}</span></div>
  <div class="card-item"><span class="label">📅 Official Joining Date</span><span class="value">{{joining_date}}</span></div>
  <div class="card-item"><span class="label">🎯 Status</span><span class="value" style="color:#059669;font-weight:700;">Active Employee</span></div>
</div>
<p>
  All your documents are verified, your system access is configured, and your orientation is complete.
  You are fully set up to contribute to our mission of delivering excellent healthcare.
</p>
<div style="text-align: center; margin: 32px 0;">
  <a href="{{portal_url}}" class="btn">Go to Your Dashboard</a>
</div>
<p style="font-size:14px; color:#94a3b8;">
  If you need any support, your HR Manager and department head are always available to help.
  Welcome aboard! 🌟
</p>
`
	fallback := layout("Onboarding Complete", "Welcome to the Dr.Ayya team, {{name}}!", body)
	return getDBTemplate("Onboarding Completed Email", fallback, map[string]string{
		"name":         name,
		"employee_id":  employeeID,
		"department":   department,
		"designation":  designation,
		"joining_date": joiningDate,
		"portal_url":   "https://medihub.mrbiz.in/login",
	})
}

// SupportTicketNotificationEmail is sent to a department's group email on a new support ticket.
func SupportTicketNotificationEmail(id uint, subject, category, priority, description, dept, reason string) string {
	body := `
<div style="font-family:sans-serif;max-width:640px;margin:0 auto;padding:24px">
  <h2 style="color:#1f2937">🎫 New Support Ticket — {{department}} Department</h2>
  <table style="width:100%;border-collapse:collapse;font-size:14px;margin-top:16px">
    <tr style="background:#f3f4f6"><td style="padding:6px 12px;font-weight:600;width:120px">Ticket #</td><td style="padding:6px 12px">{{ticket_id}}</td></tr>
    <tr><td style="padding:6px 12px;font-weight:600">Subject</td><td style="padding:6px 12px">{{subject}}</td></tr>
    <tr style="background:#f3f4f6"><td style="padding:6px 12px;font-weight:600">Category</td><td style="padding:6px 12px">{{category}}</td></tr>
    <tr><td style="padding:6px 12px;font-weight:600">Priority</td><td style="padding:6px 12px">{{priority}}</td></tr>
    <tr style="background:#f3f4f6"><td style="padding:6px 12px;font-weight:600">Description</td><td style="padding:6px 12px">{{description}}</td></tr>
    {{ai_reason}}
  </table>
  <p style="margin-top:20px;color:#6b7280;font-size:13px">Please log into the admin portal to respond.</p>
</div>`

	aiReasonLine := ""
	if reason != "" {
		aiReasonLine = fmt.Sprintf(`<tr><td style="padding:6px 12px;font-weight:600;color:#6b7280">AI Note</td><td style="padding:6px 12px;color:#374151">%s</td></tr>`, reason)
	}

	fallback := layout("New Support Ticket", "New support ticket received", body)
	return getDBTemplate("Support Ticket Notification", fallback, map[string]string{
		"ticket_id":   fmt.Sprintf("%d", id),
		"subject":     subject,
		"category":    category,
		"priority":    priority,
		"description": description,
		"department":  strings.Title(dept),
		"ai_reason":   aiReasonLine,
	})
}

// AuditCriticalAnomalyEmail sends a real-time warning to the security administrator.
func AuditCriticalAnomalyEmail(title, description, ruleName, actorRole, tableName, detectedTime string, actorID uint) string {
	body := `
<div style="font-family:-apple-system,sans-serif;background:#f7f7f9;padding:24px;">
  <table style="max-width:560px;margin:auto;background:#fff;border-radius:12px;overflow:hidden;border-top:6px solid #dc2626;">
    <tr><td style="padding:24px;">
      <h2 style="color:#dc2626;margin:0 0 16px;">🔴 CRITICAL Audit Anomaly</h2>
      <p style="font-size:16px;line-height:1.4;margin:0 0 8px;"><strong>{{title}}</strong></p>
      <p style="color:#555;font-size:14px;line-height:1.5;margin:0 0 20px;">{{description}}</p>
      <table style="width:100%;font-size:13px;color:#444;border-collapse:collapse;">
        <tr><td style="padding:4px 8px;color:#888;">Rule</td><td><code>{{rule}}</code></td></tr>
        <tr><td style="padding:4px 8px;color:#888;">Severity</td><td><strong style="color:#dc2626;">CRITICAL</strong></td></tr>
        <tr><td style="padding:4px 8px;color:#888;">Actor</td><td>#{{actor_id}} ({{actor_role}})</td></tr>
        <tr><td style="padding:4px 8px;color:#888;">Table</td><td>{{table}}</td></tr>
        <tr><td style="padding:4px 8px;color:#888;">Detected</td><td>{{detected_time}} IST</td></tr>
      </table>
      <p style="margin-top:24px;">
        <a href="https://medihub.mrbiz.in/admin/activity-logs"
           style="display:inline-block;background:#dc2626;color:#fff;padding:10px 20px;border-radius:6px;text-decoration:none;font-weight:600;">
          Investigate now →
        </a>
      </p>
      <p style="font-size:11px;color:#999;margin-top:24px;border-top:1px solid #eee;padding-top:12px;">
        This alert was generated automatically by MediHub audit anomaly detector.
        Critical anomalies are routed via real-time email; warnings appear only in the daily digest.
      </p>
    </td></tr>
  </table>
</div>`

	fallback := layout("Critical Audit Anomaly", "Critical audit anomaly detected", body)
	return getDBTemplate("Audit Critical Anomaly", fallback, map[string]string{
		"title":         title,
		"description":   description,
		"rule":          ruleName,
		"actor_id":      fmt.Sprintf("%d", actorID),
		"actor_role":    actorRole,
		"table":         tableName,
		"detected_time": detectedTime,
	})
}

// AuditDailyDigestEmail sends a summary report of daily database activity.
func AuditDailyDigestEmail(dayStr, totalEvents, chainStatus, chainColor, volumesTable, severityTable, anomaliesList string) string {
	body := `
<div style="font-family:-apple-system,sans-serif;background:#f7f7f9;padding:24px;">
  <table style="max-width:680px;margin:auto;background:#fff;border-radius:12px;overflow:hidden;border-top:6px solid #2563eb;">
    <tr><td style="padding:24px;">
      <h1 style="margin:0 0 4px;color:#111;font-size:22px;">📊 MediHub Daily Audit Digest</h1>
      <p style="color:#666;margin:0 0 24px;font-size:14px;">{{day}}</p>
      
      <div style="background:#f3f4f6;border-radius:8px;padding:16px;margin-bottom:24px;display:flex;justify-content:space-between;">
        <div style="text-align:center;flex:1;">
          <div style="font-size:28px;font-weight:700;color:#2563eb;">{{total_events}}</div>
          <div style="font-size:12px;color:#666;">Audit events</div>
        </div>
        <div style="text-align:center;flex:1;border-left:1px solid #e5e7eb;">
          <div style="font-size:18px;font-weight:700;color:{{chain_color}};">{{chain_status}}</div>
          <div style="font-size:12px;color:#666;">Chain status</div>
        </div>
      </div>
      
      {{severity_table}}
      {{anomalies_list}}
      {{volumes_table}}
      
      <p style="margin-top:32px;text-align:center;">
        <a href="https://medihub.mrbiz.in/admin/activity-logs" style="display:inline-block;background:#2563eb;color:#fff;padding:10px 24px;border-radius:6px;text-decoration:none;font-weight:600;">Open Audit Trail →</a>
      </p>
      <p style="font-size:11px;color:#999;margin-top:32px;border-top:1px solid #eee;padding-top:12px;text-align:center;">
        Daily digest generated by MediHub audit system. Critical anomalies are also sent immediately via separate email.
      </p>
    </td></tr>
  </table>
</div>`

	fallback := layout("Daily Audit Digest", "Daily digest regarding MediHub audit logs", body)
	return getDBTemplate("Audit Daily Digest", fallback, map[string]string{
		"day":            dayStr,
		"total_events":   totalEvents,
		"chain_status":   chainStatus,
		"chain_color":    chainColor,
		"volumes_table":  volumesTable,
		"severity_table": severityTable,
		"anomalies_list": anomaliesList,
	})
}

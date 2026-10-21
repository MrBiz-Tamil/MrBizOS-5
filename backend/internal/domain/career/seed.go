package career

import (
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

func SeedTemplates(db *gorm.DB) {
	// One-time auto-healing for the OTP template bug in existing databases
	if db.Migrator().HasTable(&RecruitmentTemplate{}) {
		var tmpl RecruitmentTemplate
		if err := db.Where("title = ?", "OTP Verification Email").First(&tmpl).Error; err == nil {
			if strings.Contains(tmpl.Content, "%s") {
				tmpl.Content = strings.ReplaceAll(tmpl.Content, "%s", "{{otp}}")
				db.Save(&tmpl)
				log.Println("[seed] Auto-healed legacy OTP template in database")
			}
		}

		// Auto-heal background gradient for Outlook compatibility in existing DB
		var tmpls []RecruitmentTemplate
		if err := db.Find(&tmpls).Error; err == nil {
			for _, t := range tmpls {
				updated := false
				if strings.Contains(t.Content, "background:linear-gradient(135deg,#059669,#0d9488)") {
					t.Content = strings.ReplaceAll(t.Content, "background:linear-gradient(135deg,#059669,#0d9488)", "background-color:#059669;background:linear-gradient(135deg,#059669,#0d9488)")
					updated = true
				}
				if updated {
					db.Save(&t)
					log.Printf("[seed] Auto-healed template background gradient in database: %s", t.Title)
				}
			}
		}
	}

	var count int64
	db.Model(&RecruitmentTemplate{}).Count(&count)
	if count > 0 {
		return // Already seeded
	}

	defaultTemplates := []RecruitmentTemplate{
		{
			Role:      "Doctors",
			Type:      "Offer Letter",
			Title:     "Standard Doctor Offer",
			Content:   "Dear {{name}},\n\nWe are pleased to offer you the position of {{position}} at MediHub. Your basic consultation fee will be initialized to the standard rate of ₹300 per 15-minute slot.\n\nWarm regards,\nHR Team",
			Variables: "name,position",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "HR",
			Type:      "Policy",
			Title:     "Code of Conduct",
			Content:   "Dear {{name}},\n\nAs part of your onboarding, please review the standard MediHub Code of Conduct. We maintain a professional, patient-first environment.\n\nBest regards,\nHR Department",
			Variables: "name",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "Finance",
			Type:      "Contract",
			Title:     "Payout Agreement",
			Content:   "This agreement outlines the payout terms for {{name}} in the role of {{position}}.\n\nConsultation revenue split is configured at 80% to the doctor and 20% to the platform.",
			Variables: "name,position",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "Notification",
			Title:     "Welcome Email",
			Content:   "Welcome {{name}} to MediHub! We are excited to have you on board as a {{role}}.",
			Variables: "name,role",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "SMS",
			Title:     "Interview Reminder",
			Content:   "Hi {{name}}, this is a reminder for your upcoming interview for the {{position}} role on {{date}}.",
			Variables: "name,position,date",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "WhatsApp",
			Title:     "Offer Accepted Confirmation",
			Content:   "Hello {{name}},\n\nThank you for accepting our offer for the {{position}} position. Welcome to the team!",
			Variables: "name,position",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "Email",
			Title:     "Application Received Confirmation",
			Content:   `<div style="font-family:sans-serif;max-width:560px;margin:0 auto;padding:32px;background:#f9fafb;border-radius:12px">
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
</div>`,
			Variables: "name,position,experience,license,whatsapp",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "WhatsApp",
			Title:     "Application Received Confirmation",
			Content:   "Dear {{name}}, thank you for applying for the {{position}} position at Dr.Ayya. Your application has been received and is under review.",
			Variables: "name,position",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "Email",
			Title:     "Application Status Update",
			Content:   `<div style="font-family:sans-serif;max-width:560px;margin:0 auto;padding:32px;background:#f9fafb;border-radius:12px">
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
</div>`,
			Variables: "name,position,status_label,status_color,message",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "WhatsApp",
			Title:     "Application Status Update",
			Content:   "Application Update — Dr.Ayya\nDear {{name}},\n\n{{message}}",
			Variables: "name,message",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "Doctors",
			Type:      "Email",
			Title:     "Doctor Onboarding Welcome",
			Content:   `<div style="font-family:sans-serif;max-width:560px;margin:0 auto;padding:32px;background:#f9fafb;border-radius:12px">
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
</div>`,
			Variables: "name,position,email,password",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "Doctors",
			Type:      "WhatsApp",
			Title:     "Doctor Onboarding Welcome",
			Content:   "Welcome to Dr.Ayya!\nYour doctor account is ready.\nLogin Email: {{email}}\nTemporary Password: {{password}}\nPlease change your password on first login.",
			Variables: "email,password",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "Email",
			Title:     "Welcome Email",
			Content:   `<h2>Welcome to the family, {{name}}! 🎉</h2>
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
<p style="margin-top:32px; font-size:14px; color:#94a3b8;">If you did not create this account, please contact our security team immediately.</p>`,
			Variables: "name,role",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "Email",
			Title:     "OTP Verification Email",
			Content:   `<h2>Security Verification</h2>
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
<p style="font-size:14px; color:#94a3b8;">Security Note: Never share this code with anyone, including Dr.Ayya staff. If you didn't request this, please ignore this email.</p>`,
			Variables: "name,otp,time",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "Email",
			Title:     "Password Reset Email",
			Content:   `<h2>Reset Your Password</h2>
<p>Hello <strong>{{name}}</strong>,</p>
<p>We received a request to reset your Dr.Ayya account password. Click the button below to choose a new one.</p>
<div style="text-align: center; margin: 32px 0;">
  <a href="{{resetURL}}" class="btn">Reset Password</a>
</div>
<p>This link is valid for **30 minutes**. After that, you'll need to request a new one.</p>
<p style="font-size:14px; color:#94a3b8;">If you didn't request a password reset, you can safely ignore this email. Your password will remain unchanged.</p>`,
			Variables: "name,resetURL",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "Email",
			Title:     "Email Verification Email",
			Content:   `<h2>Verify Your Email</h2>
<p>Hello <strong>{{name}}</strong>,</p>
<p>Thank you for joining Dr.Ayya! Please verify your email address to activate all features of your account.</p>
<div style="text-align: center; margin: 32px 0;">
  <a href="{{verifyURL}}" class="btn">Verify Email Address</a>
</div>
<p>This link is valid for **24 hours**.</p>
<p style="font-size:14px; color:#94a3b8;">Once verified, you'll be able to book appointments and order medicines instantly.</p>`,
			Variables: "name,verifyURL",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "Email",
			Title:     "Appointment Confirmation",
			Content:   `<h2>Appointment Confirmed! ✅</h2>
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
<p style="margin-top:32px; font-size:14px; color:#64748b;">A reminder will be sent to you 1 hour before the start time.</p>`,
			Variables: "patientName,doctorName,date,time,mode,fee,appointmentID",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "Doctors",
			Type:      "Email",
			Title:     "Doctor Appointment Notice",
			Content:   `<h2>New Appointment Scheduled 👨‍⚕️</h2>
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
</div>`,
			Variables: "doctorName,patientName,date,time,mode,appointmentID",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "Email",
			Title:     "Payment Receipt",
			Content:   `<h2>Payment Receipt</h2>
<p>Hi <strong>{{name}}</strong>, thank you for your payment. Your transaction has been processed successfully.</p>
<div class="card">
  <div class="card-item"><span class="label">💰 Amount Paid</span><span class="value">₹{{amount}}</span></div>
  <div class="card-item"><span class="label">📋 Description</span><span class="value">{{description}}</span></div>
  <div class="card-item"><span class="label">📅 Date</span><span class="value">{{date}}</span></div>
  <div class="card-item"><span class="label">🔖 Transaction ID</span><span class="value">{{txnID}}</span></div>
  <div class="card-item"><span class="label">✅ Status</span><span class="badge badge-success">Successful</span></div>
</div>
<p style="font-size:14px; color:#94a3b8;">Please keep this email for your records. If you have any questions, reply to this email or contact support.</p>`,
			Variables: "name,amount,description,date,txnID",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "Doctors",
			Type:      "Email",
			Title:     "Doctor Payout Email",
			Content:   `<h2>Earnings Transferred! 💸</h2>
<p>Hi Dr. <strong>{{doctorName}}</strong>, your consultation earnings have been transferred to your registered bank account.</p>
<div class="card">
  <div class="card-item"><span class="label">💰 Amount</span><span class="value">₹{{amount}}</span></div>
  <div class="card-item"><span class="label">📊 Appointments</span><span class="value">{{appointmentCount}}</span></div>
  <div class="card-item"><span class="label">📅 Date</span><span class="value">{{date}}</span></div>
  <div class="card-item"><span class="label">🔖 Payout ID</span><span class="value">{{payoutID}}</span></div>
</div>
<p>The funds should reflect in your account within 24–48 business hours.</p>`,
			Variables: "doctorName,amount,appointmentCount,date,payoutID",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "Email",
			Title:     "Prescription Issued Email",
			Content:   `<h2>New Prescription Issued 📋</h2>
<p>Hi <strong>{{patientName}}</strong>, Dr. <strong>{{doctorName}}</strong> has issued a digital prescription for your recent consultation.</p>
<div class="card">
  <div class="card-item"><span class="label">🩺 Diagnosis</span><span class="value">{{diagnosis}}</span></div>
  <div class="card-item"><span class="label">👨‍⚕️ Issued by</span><span class="value">Dr. {{doctorName}}</span></div>
  <div class="card-item"><span class="label">📅 Date</span><span class="value">{{date}}</span></div>
</div>
<p>You can download the PDF prescription directly from the Dr.Ayya mobile app.</p>
<div style="text-align: center;">
  <a href="https://medihub.mrbiz.in/prescriptions" class="btn">View Prescription</a>
</div>`,
			Variables: "patientName,doctorName,diagnosis,date",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "Email",
			Title:     "Appointment Reminder Email",
			Content:   `<h2>Appointment Reminder ⏰</h2>
<p>Hi <strong>{{patientName}}</strong>, this is a friendly reminder that your appointment with <strong>Dr. {{doctorName}}</strong> starts in 1 hour.</p>
<div class="card">
  <div class="card-item"><span class="label">⏰ Time</span><span class="value">{{time}} IST</span></div>
  <div class="card-item"><span class="label">🔖 Booking ID</span><span class="value">APT-{{appointmentID}}</span></div>
</div>
<p>Please ensure you have a stable internet connection if this is a video consultation.</p>
<div style="text-align: center;">
  <a href="https://medihub.mrbiz.in/appointments/{{appointmentID}}" class="btn">Join Consultation</a>
</div>`,
			Variables: "patientName,doctorName,time,appointmentID",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "HR",
			Type:      "Email",
			Title:     "HR Application Notification",
			Content:   `<div style="font-family:sans-serif;max-width:640px;margin:0 auto;padding:24px">
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
</div>`,
			Variables: "name,email,phone,whatsapp,position,experience,license,dob,gender,resume,approach,education",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "Doctors",
			Type:      "Email",
			Title:     "Doctor Profile Verified",
			Content:   `<div style="font-family:sans-serif;max-width:560px;margin:0 auto;padding:32px;background:#f9fafb;border-radius:12px">
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
</div>`,
			Variables: "name",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:  "HR",
			Type:  "Email",
			Title: "IT Provisioning Request",
			Content: `<h2>IT Provisioning Request 🖥️</h2>
<p>Hello IT Support,</p>
<p>Please process the following provisioning request for a new employee joining the team:</p>
<div class="card">
  <div class="card-item"><span class="label">🆔 Employee ID</span><span class="value">{{employee_id}}</span></div>
  <div class="card-item"><span class="label">🏢 Department</span><span class="value">{{department}}</span></div>
  <div class="card-item"><span class="label">👨‍💼 Designation</span><span class="value">{{designation}}</span></div>
</div>
<div style="background:#fdfbf7;border-left:4px solid #d4af37;border-radius:8px;padding:16px;margin:20px 0;">
  <p style="margin:0;"><strong>Message from HR:</strong><br/><br/>{{message}}</p>
</div>
<p>Please ensure all system accounts (Dr.Ayya portal, email, access cards) are created and configured by end of day.</p>`,
			Variables: "employee_id,department,designation,message",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "HR",
			Type:      "Email",
			Title:     "Onboarding Welcome Email",
			Variables: "name,employee_id,department,designation,joining_date,portal_url",
			Content: `<h2>Welcome to Dr.Ayya! 🎉</h2>
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
</p>`,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "HR",
			Type:      "Email",
			Title:     "Onboarding Document Reminder",
			Variables: "name,employee_id,pending_docs,deadline,portal_url",
			Content: `<h2>Document Submission Reminder 📄</h2>
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
</p>`,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "HR",
			Type:      "Email",
			Title:     "Onboarding Document Verified",
			Variables: "name,doc_name,verified_by,date",
			Content: `<h2>Document Verified ✅</h2>
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
</p>`,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "HR",
			Type:      "Email",
			Title:     "Onboarding Training Started",
			Variables: "name,employee_id,department,designation,training_date,portal_url",
			Content: `<h2>Your Orientation & Training Begins! 🎓</h2>
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
</p>`,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "HR",
			Type:      "Email",
			Title:     "Onboarding Completed Email",
			Variables: "name,employee_id,department,designation,joining_date,portal_url",
			Content: `<h2>Onboarding Complete — Welcome to the Team! 🏅</h2>
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
</p>`,
		},
		{
			Role:      "All Users",
			Type:      "WhatsApp",
			Title:     "drayya_otp",
			Variables: "otp",
			Content:   "Your Dr.Ayya OTP is {{otp}}. Valid for 5 minutes.",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "WhatsApp",
			Title:     "welcome_drayya",
			Variables: "name,role",
			Content:   "Welcome {{name}}! Your Dr.Ayya {{role}} account is ready.",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "WhatsApp",
			Title:     "appointment_confirmed",
			Variables: "patient_name,doctor_name,date,time",
			Content:   "Hi {{patient_name}}, your appointment with Dr. {{doctor_name}} on {{date}} at {{time}} is confirmed.",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "Doctors",
			Type:      "WhatsApp",
			Title:     "doctor_new_appointment",
			Variables: "doctor_name,patient_name,date,time",
			Content:   "Dr. {{doctor_name}}, a new appointment has been booked by {{patient_name}} on {{date}} at {{time}}.",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "WhatsApp",
			Title:     "appointment_reminder_1hr",
			Variables: "patient_name,doctor_name,time",
			Content:   "Reminder: Hi {{patient_name}}, your consultation with Dr. {{doctor_name}} starts in 1 hour at {{time}}.",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "WhatsApp",
			Title:     "appointment_cancelled",
			Variables: "patient_name,doctor_name,date",
			Content:   "Hi {{patient_name}}, your appointment with Dr. {{doctor_name}} on {{date}} has been cancelled.",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "WhatsApp",
			Title:     "payment_receipt",
			Variables: "patient_name,amount,description",
			Content:   "Hi {{patient_name}}, ₹{{amount}} paid successfully for {{description}}. Your appointment is confirmed.",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "WhatsApp",
			Title:     "wallet_topup",
			Variables: "name,amount,new_balance",
			Content:   "Hi {{name}}, ₹{{amount}} has been added to your Dr.Ayya wallet. New balance: ₹{{new_balance}}.",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "WhatsApp",
			Title:     "call_started",
			Variables: "doctor_name",
			Content:   "Dr. {{doctor_name}} has started your consultation. Please join the call.",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "WhatsApp",
			Title:     "prescription_issued",
			Variables: "patient_name,doctor_name,date",
			Content:   "Hi {{patient_name}}, Dr. {{doctor_name}} has issued a prescription for you on {{date}}. View it in the Dr.Ayya app.",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "Doctors",
			Type:      "WhatsApp",
			Title:     "doctor_payout_processed",
			Variables: "doctor_name,amount,payout_ref",
			Content:   "Hi Dr. {{doctor_name}}, ₹{{amount}} has been transferred to your bank account. Ref: {{payout_ref}}.",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "WhatsApp",
			Title:     "instant_timeout_refund",
			Variables: "patient_name,amount",
			Content:   "Hi {{patient_name}}, we couldn't match you with a doctor. ₹{{amount}} has been refunded to your wallet.",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "Email",
			Title:     "Support Ticket Notification",
			Variables: "ticket_id,subject,category,priority,description,department,ai_reason",
			Content: `
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
</div>`,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "Email",
			Title:     "Audit Critical Anomaly",
			Variables: "title,description,rule,actor_id,actor_role,table,detected_time",
			Content: `
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
</div>`,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Role:      "All Users",
			Type:      "Email",
			Title:     "Audit Daily Digest",
			Variables: "day,total_events,chain_status,chain_color,volumes_table,severity_table,anomalies_list",
			Content: `
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
</div>`,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, t := range defaultTemplates {
		if err := db.Create(&t).Error; err != nil {
			log.Printf("[seed] failed to create template: %v", err)
		}
	}

	log.Printf("[seed] successfully seeded %d recruitment templates", len(defaultTemplates))
}

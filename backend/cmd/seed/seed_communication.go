package main

import (
	"log"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/communication"
	"drayya-core/internal/domain/identity"
	"time"

	"gorm.io/gorm"
)

func seedCommunication(db *gorm.DB) {
	// Chat rooms for completed appointments
	var completed []appointment.Appointment
	db.Where("status = ?", "completed").Preload("Patient").Find(&completed)

	for _, apt := range completed {
		if apt.DoctorID == nil {
			continue
		}
		var existing communication.ChatRoom
		if db.Where("appointment_id = ?", apt.ID).First(&existing).Error == nil {
			continue
		}
		room := communication.ChatRoom{
			AppointmentID: &apt.ID,
			DoctorID:      *apt.DoctorID,
			PatientID:     apt.PatientID,
			ChatType:      "consultation",
			Urgency:       "normal",
			IsActive:      false,
		}
		if apt.AppointmentType == "sos" {
			room.Urgency = "sos"
		}
		closedAt := apt.EndTime
		room.ClosedAt = &closedAt
		db.Create(&room)

		// Messages
		msgs := []communication.ChatMessage{
			{RoomID: room.ID, SenderID: apt.PatientID, ReceiverID: *apt.DoctorID, Message: "Hello Doctor, I need your help.", MessageType: "text", IsRead: true, CreatedAt: apt.StartTime},
			{RoomID: room.ID, SenderID: *apt.DoctorID, ReceiverID: apt.PatientID, Message: "Hello! Please describe your symptoms.", MessageType: "text", IsRead: true, CreatedAt: apt.StartTime.Add(1 * time.Minute)},
			{RoomID: room.ID, SenderID: apt.PatientID, ReceiverID: *apt.DoctorID, Message: "I have been having fever and body ache for 2 days.", MessageType: "text", IsRead: true, CreatedAt: apt.StartTime.Add(2 * time.Minute)},
			{RoomID: room.ID, SenderID: *apt.DoctorID, ReceiverID: apt.PatientID, Message: "I see. I will prescribe some medicines. Please take rest and stay hydrated.", MessageType: "text", IsRead: true, CreatedAt: apt.StartTime.Add(5 * time.Minute)},
		}
		for i := range msgs {
			db.Create(&msgs[i])
		}
	}

	// Notification templates
	templates := []communication.NotificationTemplate{
		{TemplateType: "appointment_confirmed", MessageContent: "Your appointment with Dr. {{doctor_name}} is confirmed for {{date}} at {{time}}.", Variables: "doctor_name,date,time", IsActive: true},
		{TemplateType: "appointment_reminder", MessageContent: "Reminder: Your appointment with Dr. {{doctor_name}} is in 1 hour.", Variables: "doctor_name", IsActive: true},
		{TemplateType: "prescription_ready", MessageContent: "Your prescription from Dr. {{doctor_name}} is ready. View it in the app.", Variables: "doctor_name", IsActive: true},
		{TemplateType: "lab_results_ready", MessageContent: "Your lab results are ready. Please check the reports section.", Variables: "", IsActive: true},
		{TemplateType: "payment_received", MessageContent: "Payment of    {{amount}} received for appointment #{{appointment_id}}.", Variables: "amount,appointment_id", IsActive: true},
		{TemplateType: "onboarding_welcome", MessageContent: "Welcome to Dr.Ayya! Your employee onboarding has begun. Employee ID: {{employee_id}}.", Variables: "employee_id", IsActive: true},
		{TemplateType: "onboarding_doc_reminder", MessageContent: "Friendly reminder: Please upload your pending onboarding documents.", Variables: "", IsActive: true},
		{TemplateType: "onboarding_doc_verified", MessageContent: "Document Verified: Your {{doc_name}} has been verified successfully.", Variables: "doc_name", IsActive: true},
		{TemplateType: "onboarding_training_started", MessageContent: "Your training and orientation has started! Welcome to the active team.", Variables: "", IsActive: true},
		{TemplateType: "onboarding_completed", MessageContent: "Congratulations! Your onboarding is complete. Welcome to the team!", Variables: "", IsActive: true},
	}
	for i := range templates {
		db.FirstOrCreate(&templates[i], "template_type = ?", templates[i].TemplateType)
	}

	// Notifications for patients
	var patients []identity.User
	db.Where("role = ?", "patient").Limit(3).Find(&patients)
	for _, p := range patients {
		notifs := []communication.Notification{
			{UserID: p.ID, Title: "Welcome to Dr.Ayya!", Message: "Your account has been created successfully. Book your first appointment today.", Type: "info"},
			{UserID: p.ID, Title: "Appointment Confirmed", Message: "Your appointment has been confirmed.", Type: "success"},
		}
		for _, n := range notifs {
			db.FirstOrCreate(&n, "user_id = ? AND title = ?", n.UserID, n.Title)
		}
	}

	log.Printf("       Created chat rooms, messages, and notification templates")
}


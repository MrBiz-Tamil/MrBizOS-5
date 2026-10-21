package main

import (
	"log"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/identity"
	"time"

	"gorm.io/gorm"
)

func seedAppointment(db *gorm.DB) {
	// Fetch users
	var doctors []identity.User
	db.Where("role = ?", "doctor").Find(&doctors)
	var patients []identity.User
	db.Where("role = ?", "patient").Find(&patients)

	if len(doctors) == 0 || len(patients) == 0 {
		log.Println("          No doctors/patients found, skipping appointments")
		return
	}

	now := time.Now()

	appointments := []appointment.Appointment{
		// Completed appointments (past)
		{PatientID: patients[0].ID, DoctorID: &doctors[0].ID, StartTime: now.Add(-72 * time.Hour), EndTime: now.Add(-71 * time.Hour), Status: "completed", AppointmentType: "consultation", ConsultationMode: "video", ConsultationFee: 50000, Notes: "General checkup completed"},
		{PatientID: patients[1].ID, DoctorID: &doctors[1].ID, StartTime: now.Add(-48 * time.Hour), EndTime: now.Add(-47 * time.Hour), Status: "completed", AppointmentType: "consultation", ConsultationMode: "in_person", ConsultationFee: 80000, Notes: "Cardiology follow-up"},
		{PatientID: patients[2].ID, DoctorID: &doctors[0].ID, StartTime: now.Add(-24 * time.Hour), EndTime: now.Add(-23 * time.Hour), Status: "completed", AppointmentType: "consultation", ConsultationMode: "video", ConsultationFee: 50000, Notes: "Diabetes review"},
		// Today's appointments (confirmed/in-progress)
		{PatientID: patients[0].ID, DoctorID: &doctors[2].ID, StartTime: now.Add(2 * time.Hour), EndTime: now.Add(3 * time.Hour), Status: "confirmed", AppointmentType: "consultation", ConsultationMode: "video", ConsultationFee: 60000, Notes: "Child vaccination"},
		{PatientID: patients[3].ID, DoctorID: &doctors[3].ID, StartTime: now.Add(4 * time.Hour), EndTime: now.Add(5 * time.Hour), Status: "confirmed", AppointmentType: "consultation", ConsultationMode: "in_person", ConsultationFee: 70000, Notes: "Skin rash checkup"},
		// Pending appointments (future)
		{PatientID: patients[4].ID, DoctorID: &doctors[0].ID, StartTime: now.Add(24 * time.Hour), EndTime: now.Add(25 * time.Hour), Status: "pending", AppointmentType: "consultation", ConsultationMode: "chat", ConsultationFee: 50000, Notes: "First consultation"},
		{PatientID: patients[1].ID, DoctorID: &doctors[2].ID, StartTime: now.Add(48 * time.Hour), EndTime: now.Add(49 * time.Hour), Status: "scheduled", AppointmentType: "follow_up", ConsultationMode: "video", ConsultationFee: 40000, Notes: "Follow-up for asthma"},
		// Cancelled
		{PatientID: patients[2].ID, DoctorID: &doctors[1].ID, StartTime: now.Add(-96 * time.Hour), EndTime: now.Add(-95 * time.Hour), Status: "cancelled", AppointmentType: "consultation", ConsultationMode: "in_person", ConsultationFee: 80000, Notes: "Patient cancelled"},
		// SOS emergency
		{PatientID: patients[0].ID, DoctorID: &doctors[0].ID, StartTime: now.Add(-5 * time.Hour), EndTime: now.Add(-4 * time.Hour), Status: "completed", AppointmentType: "sos", ConsultationMode: "video", ConsultationFee: 100000, Notes: "Emergency SOS - chest discomfort", IsPriority: true},
		// Family member appointment
		{PatientID: patients[0].ID, DoctorID: &doctors[2].ID, StartTime: now.Add(72 * time.Hour), EndTime: now.Add(73 * time.Hour), Status: "pending", AppointmentType: "consultation", ConsultationMode: "in_person", ConsultationFee: 60000, Notes: "Child fever checkup"},
	}

	for i := range appointments {
		var existing appointment.Appointment
		if db.Where("patient_id = ? AND doctor_id = ? AND start_time = ?",
			appointments[i].PatientID, appointments[i].DoctorID, appointments[i].StartTime).
			First(&existing).Error == nil {
			continue
		}
		db.Create(&appointments[i])
	}

	log.Printf("       Created %d appointments", len(appointments))
}


package main

import (
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/billing"
	"drayya-core/internal/domain/communication"
	"drayya-core/internal/domain/emr"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/franchise"
	"drayya-core/internal/domain/hospital"
	"drayya-core/internal/domain/hr"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/lab"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/domain/store"
	"drayya-core/internal/domain/support"

	"gorm.io/gorm"
)

func dropAllTables(db *gorm.DB) {
	tables := []string{
		"ledger_entry_v2", "ledger_transaction_v2", "ledger_account_v2",
		"courier_deliveries", "delivery_zones",
		"wellness_order_items", "wellness_orders", "wellness_products",
		"payroll_records", "staff_attendance", "staff_shifts", "staff_profiles",
		"notifications", "notification_logs", "notification_templates",
		"chat_messages", "chat_rooms", "video_consultations",
		"lab_specimens", "lab_results", "lab_request_items", "lab_requests", "lab_tests",
		"payment", "billing",
		"health_timeline_events", "patient_vitals", "diagnosis",
		"prescription_templates", "prescriptions", "medical_records",
		"appointment",
		"pharmacy_purchase_orders", "pharmacy_inventory_transfers",
		"cold_chain_metrics", "courier_profiles",
		"pharmacy_order_items", "pharmacy_orders",
		"pharmacyinventory", "medicine", "pharmacy",
		"pharmacy_suppliers",
		"hospital_ipd_admissions", "hospital_ward_service_requests",
		"hospital_beds", "hospital_departments", "hospitals",
		"franchise_royalty_logs", "franchises",
		"family_members", "doctor_availability", "doctorprofile",
		"patient_profiles", "activity_logs", "user",
		"tickets", "ticket_messages", "blogs",
	}
	for _, t := range tables {
		db.Exec("DROP TABLE IF EXISTS " + t)
	}
}

func runMigrations(db *gorm.DB) {
	identity.AutoMigrate(db)
	db.AutoMigrate(&franchise.Franchise{}, &franchise.FranchiseRoyaltyLog{})
	db.AutoMigrate(&hospital.Hospital{}, &hospital.HospitalDepartment{}, &hospital.Bed{},
		&hospital.WardServiceRequest{}, &hospital.IPDAdmission{})
	pharmacy.AutoMigrate(db)
	db.AutoMigrate(&appointment.Appointment{})
	emr.AutoMigrate(db)
	db.AutoMigrate(&billing.Billing{}, &billing.Payment{})
	db.AutoMigrate(&lab.LabTest{}, &lab.LabRequest{}, &lab.LabResult{}, &lab.Specimen{})
	db.AutoMigrate(&communication.ChatRoom{}, &communication.ChatMessage{},
		&communication.NotificationTemplate{}, &communication.NotificationLog{},
		&communication.Notification{}, &communication.VideoConsultation{})
	hr.AutoMigrate(db)
	store.AutoMigrate(db)
	finance.AutoMigrate(db)
	db.AutoMigrate(&support.Ticket{}, &support.TicketMessage{})
}

func printCredentials() {
	creds := `
     Test Credentials:
                                                                                                                                                
Admin:        admin@Dr.Ayya.com       / Test@123
Finance Mgr:  finance@Dr.Ayya.com     / Test@123
HR Manager:   hr@Dr.Ayya.com          / Test@123
Doctor 1:     dr.sharma@Dr.Ayya.com   / Test@123  (General Medicine)
Doctor 2:     dr.patel@Dr.Ayya.com    / Test@123  (Cardiology)
Doctor 3:     dr.gupta@Dr.Ayya.com    / Test@123  (Pediatrics)
Doctor 4:     dr.kumar@Dr.Ayya.com    / Test@123  (Dermatology)
Nurse 1:      nurse1@Dr.Ayya.com      / Test@123
Nurse 2:      nurse2@Dr.Ayya.com      / Test@123
Lab Tech:     labtech@Dr.Ayya.com     / Test@123
Pharmacist:   pharmacist@Dr.Ayya.com  / Test@123
Patient 1:    patient1@Dr.Ayya.com    / Test@123
Patient 2:    patient2@Dr.Ayya.com    / Test@123
Patient 3:    patient3@Dr.Ayya.com    / Test@123
Patient 4:    patient4@Dr.Ayya.com    / Test@123
Patient 5:    patient5@Dr.Ayya.com    / Test@123
Courier 1:    courier1@Dr.Ayya.com    / Test@123
Courier 2:    courier2@Dr.Ayya.com    / Test@123
                                                                                                                                                `
	println(creds)
}


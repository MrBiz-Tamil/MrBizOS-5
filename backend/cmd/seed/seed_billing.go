package main

import (
	"log"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/billing"
	"time"

	"gorm.io/gorm"
)

func seedBilling(db *gorm.DB) {
	var completed []appointment.Appointment
	db.Where("status = ?", "completed").Find(&completed)

	for i, apt := range completed {
		// Billing record
		var existingBill billing.Billing
		if db.Where("appointment_id = ?", apt.ID).First(&existingBill).Error == nil {
			continue
		}
		status := "paid"
		if i == len(completed)-1 {
			status = "pending"
		}
		b := billing.Billing{
			AppointmentID: &apt.ID,
			PatientID:     apt.PatientID,
			Amount:        apt.ConsultationFee,
			PaymentStatus: status,
			PaymentMethod: "upi",
			InvoiceDate:   apt.StartTime,
		}
		db.Create(&b)

		// Payment record for paid bills
		if status == "paid" {
			platformFee := float64(apt.ConsultationFee) * 0.15
			db.Create(&billing.Payment{
				AppointmentID:         &apt.ID,
				PatientID:             apt.PatientID,
				Amount:                apt.ConsultationFee,
				ConsultationFee:       apt.ConsultationFee,
				PlatformFeePercentage: 15.0,
				PlatformCommission:    int64(platformFee),
				DoctorEarnings:        apt.ConsultationFee - int64(platformFee),
				PaymentMethod:         "upi",
				RazorpayPaymentID:     "pay_test_" + time.Now().Format("20060102") + "_" + string(rune('A'+i)),
				RazorpayOrderID:       "order_test_" + time.Now().Format("20060102") + "_" + string(rune('A'+i)),
				Status:                "completed",
				PaidAt:                apt.EndTime,
			})
		}
	}

	log.Printf("       Created billing & payment records for %d appointments", len(completed))
}


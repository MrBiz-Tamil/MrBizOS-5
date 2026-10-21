package handlers

import (
	"context"
	"fmt"
	"os"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/middleware"
	"drayya-core/internal/services/email"
	"drayya-core/internal/services/notification"
	rz "drayya-core/internal/services/razorpay"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type WalletHandler struct {
	finSvc finance.Service
}

func NewWalletHandler(finSvc finance.Service) *WalletHandler {
	return &WalletHandler{finSvc: finSvc}
}

// GetBalance returns the authenticated user's wallet balance
func (h *WalletHandler) GetBalance(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	ctx := context.Background()
	walletName := fmt.Sprintf("PatientWallet:%d", userID)
	acct, err := h.finSvc.GetOrCreateAccount(ctx, walletName, finance.TypeLiability)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "WALLET_ERROR", "Failed to fetch wallet")
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"balance":      float64(acct.Balance) / 100.0,
		"currency":     acct.Currency,
		"wallet_id":    acct.ID,
		"last_updated": acct.UpdatedAt,
		"is_frozen":    false,
	}, "Wallet balance fetched")
}

type TopUpRequest struct {
	Amount float64 `json:"amount"`
	Source string  `json:"source"`
	TxnID  string  `json:"txn_id"`
}

// TopUp adds funds to the user's wallet
func (h *WalletHandler) TopUp(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	role, _ := c.Locals("role").(string)
	if role != "admin" && role != "super_admin" {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Use the verified payment flow to top up wallet")
	}
	if os.Getenv("ALLOW_DEBUG_WALLET_TOPUP") != "true" {
		return response.Error(c, fiber.StatusForbidden, "DEBUG_TOPUP_DISABLED", "Direct wallet top-up is disabled")
	}
	var req TopUpRequest
	if err := c.BodyParser(&req); err != nil || req.Amount <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Amount must be positive")
	}
	ctx := context.Background()
	amountPaise := int64(req.Amount * 100)
	walletName := fmt.Sprintf("PatientWallet:%d", userID)

	walletAcct, err := h.finSvc.GetOrCreateAccount(ctx, walletName, finance.TypeLiability)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "WALLET_ERROR", "Failed to fetch wallet")
	}
	gatewayAcct, err := h.finSvc.GetOrCreateAccount(ctx, "Gateway:"+req.Source, finance.TypeAsset)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "GATEWAY_ERROR", "Failed to fetch gateway account")
	}

	refID := req.TxnID
	if refID == "" {
		refID = "TOPUP-" + uuid.New().String()
	}
	_, err = h.finSvc.RecordTransaction(ctx, refID, "TOPUP", "Wallet top-up", []finance.EntryData{
		{AccountID: gatewayAcct.ID, Direction: finance.DirectionDebit, Amount: amountPaise},
		{AccountID: walletAcct.ID, Direction: finance.DirectionCredit, Amount: amountPaise},
	})
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "TOPUP_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"new_balance": float64(walletAcct.Balance+amountPaise) / 100.0,
	}, "Wallet topped up successfully")
}

// InitiateTopUp creates a Razorpay order and returns payment details to the frontend.
// POST /wallet/topup/initiate   body: { amount: number }
func (h *WalletHandler) InitiateTopUp(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	var req struct {
		Amount float64 `json:"amount"`
	}
	if err := c.BodyParser(&req); err != nil || req.Amount < 1 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_AMOUNT", "amount must be ≥ 1")
	}

	amountPaise := int64(req.Amount * 100)
	receipt := fmt.Sprintf("wallet-topup-%d-%d", userID, time.Now().Unix())

	order, err := rz.CreateOrder(amountPaise, receipt)
	if err != nil {
		return response.Error(c, fiber.StatusBadGateway, "RAZORPAY_ERROR", err.Error())
	}

	// Return everything the frontend Razorpay SDK needs
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"order_id": order["id"],
		"amount":   amountPaise,
		"currency": "INR",
		"key_id":   razorpayPublicKeyID(),
		"user_id":  userID,
		"receipt":  receipt,
	}, "Razorpay order created")
}

// VerifyTopUp validates the Razorpay payment signature and credits the wallet.
// POST /wallet/topup/verify   body: { order_id, payment_id, signature, amount }
func (h *WalletHandler) VerifyTopUp(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	var req struct {
		OrderID   string  `json:"order_id"`
		PaymentID string  `json:"payment_id"`
		Signature string  `json:"signature"`
		Amount    float64 `json:"amount"` // in INR
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if req.OrderID == "" || req.PaymentID == "" || req.Signature == "" {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELDS", "order_id, payment_id, signature required")
	}

	if !rz.VerifyPaymentSignature(req.OrderID, req.PaymentID, req.Signature) {
		return response.Error(c, fiber.StatusUnprocessableEntity, "INVALID_SIGNATURE", "Payment verification failed")
	}

	ctx := context.Background()
	amountPaise := int64(req.Amount * 100)
	walletName := fmt.Sprintf("PatientWallet:%d", userID)

	walletAcct, err := h.finSvc.GetOrCreateAccount(ctx, walletName, finance.TypeLiability)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "WALLET_ERROR", "Failed to fetch wallet")
	}
	gatewayAcct, err := h.finSvc.GetOrCreateAccount(ctx, "Gateway:Razorpay", finance.TypeAsset)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "GATEWAY_ERROR", "Internal error")
	}

	_, err = h.finSvc.RecordTransaction(ctx, req.PaymentID, "TOPUP_RAZORPAY",
		fmt.Sprintf("Razorpay top-up | order: %s", req.OrderID),
		[]finance.EntryData{
			{AccountID: gatewayAcct.ID, Direction: finance.DirectionDebit, Amount: amountPaise},
			{AccountID: walletAcct.ID, Direction: finance.DirectionCredit, Amount: amountPaise},
		})
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "TOPUP_FAILED", err.Error())
	}

	// Send FCM + WhatsApp + Email confirmation
	var user identity.User
	if err := database.DB.Select("fcm_token, phone_number, full_name, email").First(&user, userID).Error; err == nil {
		newBalance := float64(walletAcct.Balance+amountPaise) / 100.0
		notification.SendToToken(user.FCMToken, notification.FCMMessage{
			Title: "Wallet Top-up Successful",
			Body:  fmt.Sprintf("₹%.0f added to your Dr.Ayya wallet.", req.Amount),
			Data:  map[string]string{"type": "topup_success"},
		})
		if user.PhoneNumber != "" {
			notification.WhatsAppWalletTopUp(user.PhoneNumber, user.FullName,
				fmt.Sprintf("%.0f", req.Amount),
				fmt.Sprintf("%.0f", newBalance))
		}
		notification.EnqueueEmail(context.Background(),
			user.Email,
			"Wallet Top-up Successful — Dr.Ayya",
			email.WalletTopUpEmail(user.FullName, req.Amount, newBalance, req.PaymentID),
		)
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"payment_id":  req.PaymentID,
		"amount":      req.Amount,
		"new_balance": float64(walletAcct.Balance+amountPaise) / 100.0,
	}, "Wallet topped up successfully")
}

// razorpayPublicKeyID returns the public Razorpay key ID to send to the frontend SDK.
func razorpayPublicKeyID() string {
	return os.Getenv("RAZORPAY_KEY_ID")
}

// PayConsultation deducts the consultation fee from the patient's wallet and marks appointment confirmed.
// POST /wallet/pay-consultation  body: {appointment_id: number}
func (h *WalletHandler) PayConsultation(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	var req struct {
		AppointmentID uint `json:"appointment_id"`
	}
	if err := c.BodyParser(&req); err != nil || req.AppointmentID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "appointment_id required")
	}

	ctx := context.Background()

	var appt appointment.Appointment
	if err := database.DB.First(&appt, req.AppointmentID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
	}
	if appt.PatientID != userID {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You can only pay for your own appointments")
	}
	if appt.Status != appointment.StatusPending {
		return response.Error(c, fiber.StatusConflict, "INVALID_STATUS", "Appointment is not awaiting payment")
	}
	fee := appt.ConsultationFee
	if fee <= 0 {
		database.DB.Model(&appointment.Appointment{}).
			Where("id = ? AND patient_id = ? AND status = ?", req.AppointmentID, userID, appointment.StatusPending).
			Update("status", string(appointment.StatusConfirmed))
		return response.JSON(c, fiber.StatusOK, fiber.Map{
			"appointment_id": req.AppointmentID,
			"amount_paid":    0,
		}, "Appointment confirmed")
	}
	walletName := fmt.Sprintf("PatientWallet:%d", userID)
	walletAcct, err := h.finSvc.GetOrCreateAccount(ctx, walletName, finance.TypeLiability)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "WALLET_ERROR", "Failed to fetch wallet")
	}
	if walletAcct.Balance < fee {
		return response.Error(c, fiber.StatusPaymentRequired, "INSUFFICIENT_BALANCE",
			fmt.Sprintf("Insufficient balance. Need ₹%.2f, have ₹%.2f", float64(fee)/100, float64(walletAcct.Balance)/100))
	}

	heldAcct, err := h.finSvc.GetOrCreateAccount(ctx, "HeldFunds", finance.TypeLiability)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "HELD_FUNDS_ERROR", "Internal error")
	}
	claim := database.DB.Model(&appointment.Appointment{}).
		Where("id = ? AND patient_id = ? AND status = ?", req.AppointmentID, userID, appointment.StatusPending).
		Update("status", "payment_processing")
	if claim.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PAYMENT_FAILED", "Could not reserve appointment for payment")
	}
	if claim.RowsAffected == 0 {
		return response.Error(c, fiber.StatusConflict, "INVALID_STATUS", "Appointment is no longer awaiting payment")
	}

	refID := fmt.Sprintf("CONSULT-%d-%d", userID, req.AppointmentID)
	_, err = h.finSvc.RecordTransaction(ctx, refID, "CONSULTATION_PAYMENT",
		fmt.Sprintf("Payment for appointment #%d", req.AppointmentID),
		[]finance.EntryData{
			{AccountID: walletAcct.ID, Direction: finance.DirectionDebit, Amount: fee},
			{AccountID: heldAcct.ID, Direction: finance.DirectionCredit, Amount: fee},
		})
	if err != nil {
		database.DB.Model(&appointment.Appointment{}).
			Where("id = ? AND patient_id = ? AND status = ?", req.AppointmentID, userID, "payment_processing").
			Update("status", string(appointment.StatusPending))
		return response.Error(c, fiber.StatusInternalServerError, "PAYMENT_FAILED", err.Error())
	}

	// Mark appointment as confirmed
	result := database.DB.Model(&appointment.Appointment{}).
		Where("id = ? AND patient_id = ? AND status = ?", req.AppointmentID, userID, "payment_processing").
		Update("status", string(appointment.StatusConfirmed))
	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CONFIRM_FAILED", "Payment recorded but appointment confirmation failed")
	}
	if result.RowsAffected == 0 {
		return response.Error(c, fiber.StatusConflict, "INVALID_STATUS", "Appointment is no longer awaiting payment")
	}

	// Notify patient via FCM + WhatsApp + Email
	go func() {
		var user identity.User
		if err := database.DB.Select("fcm_token, phone_number, full_name, email").First(&user, userID).Error; err != nil {
			return
		}
		// Resolve appointment + doctor details for email
		doctorName := "your doctor"
		doctorEmail := ""
		if database.DB.Preload("Doctor").First(&appt, req.AppointmentID).Error == nil && appt.DoctorID != nil {
			var doc identity.User
			if database.DB.Select("full_name, email").First(&doc, *appt.DoctorID).Error == nil {
				doctorName = doc.FullName
				doctorEmail = doc.Email
			}
		}
		notification.NotifyPaymentSuccess(user.FCMToken, float64(fee)/100.0, req.AppointmentID)
		notification.WhatsAppPaymentReceipt(user.PhoneNumber, user.FullName,
			fmt.Sprintf("%.0f", float64(fee)/100.0), doctorName)

		// Payment receipt email to patient
		notification.EnqueueEmail(context.Background(),
			user.Email,
			"Payment Confirmed — Dr.Ayya",
			email.PaymentReceipt(email.PaymentEmailData{
				Name:        user.FullName,
				Amount:      float64(fee) / 100.0,
				TxnID:       fmt.Sprintf("CONSULT-%d-%d", userID, req.AppointmentID),
				Description: "Consultation with Dr. " + doctorName,
				Date:        time.Now().Format("02 Jan 2006, 3:04 PM"),
			}),
		)
		// Appointment notice email to doctor
		if doctorEmail != "" && appt.ID != 0 {
			dateStr := appt.StartTime.Format("Mon, 02 Jan 2006")
			timeStr := appt.StartTime.Format("3:04 PM")
			notification.EnqueueEmail(context.Background(),
				doctorEmail,
				"New Appointment — Dr.Ayya",
				email.DoctorAppointmentNotice(email.AppointmentEmailData{
					PatientName:   user.FullName,
					DoctorName:    doctorName,
					Date:          dateStr,
					Time:          timeStr,
					Mode:          appt.ConsultationMode,
					AppointmentID: appt.ID,
					Fee:           float64(fee) / 100.0,
				}),
			)
		}
	}()

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"appointment_id": req.AppointmentID,
		"amount_paid":    float64(fee) / 100.0,
		"new_balance":    float64(walletAcct.Balance-fee) / 100.0,
	}, "Payment successful")
}

// GetTransactions returns the last 50 wallet transactions
func (h *WalletHandler) GetTransactions(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	ctx := context.Background()
	walletName := fmt.Sprintf("PatientWallet:%d", userID)
	walletAcct, err := h.finSvc.GetOrCreateAccount(ctx, walletName, finance.TypeLiability)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "WALLET_ERROR", "Failed to fetch wallet")
	}
	entries, err := h.finSvc.GetTransactions(walletAcct.ID, 50)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, entries, "Transactions fetched")
}

// Withdraw requests a withdrawal from the wallet
func (h *WalletHandler) Withdraw(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	var req struct {
		Amount float64 `json:"amount"`
		Notes  string  `json:"notes"`
	}
	if err := c.BodyParser(&req); err != nil || req.Amount <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Amount must be positive")
	}
	ctx := context.Background()
	amountPaise := int64(req.Amount * 100)
	walletName := fmt.Sprintf("PatientWallet:%d", userID)
	walletAcct, err := h.finSvc.GetOrCreateAccount(ctx, walletName, finance.TypeLiability)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "WALLET_ERROR", "Failed to fetch wallet")
	}
	if walletAcct.Balance < amountPaise {
		return response.Error(c, fiber.StatusPaymentRequired, "INSUFFICIENT_BALANCE", "Insufficient wallet balance")
	}
	payoutAcct, err := h.finSvc.GetOrCreateAccount(ctx, "Internal:PendingPayouts", finance.TypeLiability)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PAYOUT_ERROR", "Internal error")
	}
	refID := "WITHDRAW-" + uuid.New().String()
	_, err = h.finSvc.RecordTransaction(ctx, refID, "WITHDRAWAL", req.Notes, []finance.EntryData{
		{AccountID: walletAcct.ID, Direction: finance.DirectionDebit, Amount: amountPaise},
		{AccountID: payoutAcct.ID, Direction: finance.DirectionCredit, Amount: amountPaise},
	})
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "WITHDRAW_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"new_balance": float64(walletAcct.Balance-amountPaise) / 100.0,
	}, "Withdrawal request processed")
}

// GetStatement returns a wallet summary
func (h *WalletHandler) GetStatement(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	ctx := context.Background()
	walletName := fmt.Sprintf("PatientWallet:%d", userID)
	walletAcct, err := h.finSvc.GetOrCreateAccount(ctx, walletName, finance.TypeLiability)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "WALLET_ERROR", "Failed to fetch wallet")
	}
	entries, _ := h.finSvc.GetTransactions(walletAcct.ID, 100)
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"current_balance": float64(walletAcct.Balance) / 100.0,
		"currency":        walletAcct.Currency,
		"statement_date":  time.Now(),
		"transactions":    entries,
	}, "Statement fetched")
}

func RegisterWalletRoutes(router fiber.Router, finSvc finance.Service) {
	h := NewWalletHandler(finSvc)
	walletGroup := router.Group("/wallet", middleware.Protected())
	walletGroup.Get("/balance", h.GetBalance)
	walletGroup.Post("/topup", h.TopUp)                  // legacy direct top-up (debug/admin)
	walletGroup.Post("/topup/debug", h.TopUp)            // dev alias
	walletGroup.Post("/topup/initiate", h.InitiateTopUp) // Razorpay: create order
	walletGroup.Post("/topup/verify", h.VerifyTopUp)     // Razorpay: verify + credit
	walletGroup.Post("/pay-consultation", h.PayConsultation)
	walletGroup.Get("/transactions", h.GetTransactions)
	walletGroup.Post("/withdraw", h.Withdraw)
	walletGroup.Get("/statement", h.GetStatement)
}

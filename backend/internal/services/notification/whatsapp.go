package notification

// Meta WhatsApp Cloud API (Business API)
// Docs: https://developers.facebook.com/docs/whatsapp/cloud-api/messages
//
// Required env vars:
//   WHATSAPP_PHONE_NUMBER_ID  — numeric phone number ID from Meta dashboard
//   WHATSAPP_ACCESS_TOKEN     — permanent or temporary system user access token
//   WHATSAPP_TEMPLATE_LANG    — template language code, default "en"
//   WHATSAPP_VERIFY_TOKEN     — shared secret for webhook challenge verification
//
// Template names referenced here must be pre-approved in your Meta Business
// account before they can be sent to any phone number.
//
// Template catalogue used by Dr.Ayya:
//
//   Name                      Category         Body params
//   ──────────────────────────────────────────────────────────────────────
//   drayya_otp               AUTHENTICATION   {{1}} = OTP code
//   appointment_confirmed     UTILITY          {{1}} = patient, {{2}} = doctor, {{3}} = date, {{4}} = time
//   appointment_reminder_1hr  UTILITY          {{1}} = patient, {{2}} = doctor, {{3}} = time
//   appointment_cancelled     UTILITY          {{1}} = patient, {{2}} = doctor, {{3}} = date
//   doctor_new_appointment    UTILITY          {{1}} = doctor, {{2}} = patient, {{3}} = date, {{4}} = time
//   payment_receipt           UTILITY          {{1}} = patient, {{2}} = amount, {{3}} = description
//   wallet_topup              UTILITY          {{1}} = name, {{2}} = amount, {{3}} = new_balance
//   doctor_payout_processed   UTILITY          {{1}} = doctor, {{2}} = amount, {{3}} = payout_ref
//   prescription_issued       UTILITY          {{1}} = patient, {{2}} = doctor, {{3}} = date
//   welcome_drayya           MARKETING        {{1}} = name, {{2}} = role
//   instant_timeout_refund    UTILITY          {{1}} = patient, {{2}} = amount

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"drayya-core/internal/database"
)

const waAPIBase = "https://graph.facebook.com/v20.0"

// WAResult is returned by every send call.
type WAResult struct {
	MessageID string `json:"message_id"` // wamid.xxx
}

// ─── Wire types ──────────────────────────────────────────────────────────────

type waTextMessage struct {
	MessagingProduct string `json:"messaging_product"`
	To               string `json:"to"`
	Type             string `json:"type"`
	Text             waText `json:"text"`
}
type waText struct {
	PreviewURL bool   `json:"preview_url,omitempty"`
	Body       string `json:"body"`
}

type waTemplateMessage struct {
	MessagingProduct string     `json:"messaging_product"`
	To               string     `json:"to"`
	Type             string     `json:"type"`
	Template         waTemplate `json:"template"`
}
type waTemplate struct {
	Name       string        `json:"name"`
	Language   waLangCode    `json:"language"`
	Components []waComponent `json:"components,omitempty"`
}
type waLangCode struct {
	Code string `json:"code"`
}
type waComponent struct {
	Type       string    `json:"type"`            // body | header | button
	SubType    string    `json:"sub_type,omitempty"` // url | quick_reply (for buttons)
	Index      string    `json:"index,omitempty"` // button index
	Parameters []waParam `json:"parameters"`
}
type waParam struct {
	Type     string   `json:"type"`               // text | image | document | video
	Text     string   `json:"text,omitempty"`
	Image    *waMedia `json:"image,omitempty"`
	Document *waMedia `json:"document,omitempty"`
}
type waMedia struct {
	Link    string `json:"link,omitempty"`    // public URL
	Caption string `json:"caption,omitempty"`
}

// waInteractiveMessage supports reply-button interactive messages.
type waInteractiveMessage struct {
	MessagingProduct string        `json:"messaging_product"`
	To               string        `json:"to"`
	Type             string        `json:"type"` // "interactive"
	Interactive      waInteractive `json:"interactive"`
}
type waInteractive struct {
	Type   string       `json:"type"` // "button"
	Header *waIHeader   `json:"header,omitempty"`
	Body   waIBody      `json:"body"`
	Footer *waIFooter   `json:"footer,omitempty"`
	Action waIAction    `json:"action"`
}
type waIHeader struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}
type waIBody struct {
	Text string `json:"text"`
}
type waIFooter struct {
	Text string `json:"text"`
}
type waIAction struct {
	Buttons []waIButton `json:"buttons"`
}
type waIButton struct {
	Type  string   `json:"type"` // "reply"
	Reply waIReply `json:"reply"`
}
type waIReply struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// waAPIResponse is the raw shape returned by Meta on success.
type waAPIResponse struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

// ─── Config helpers ───────────────────────────────────────────────────────────

func waPhoneID() string { return os.Getenv("WHATSAPP_PHONE_NUMBER_ID") }
func waToken() string   { return os.Getenv("WHATSAPP_ACCESS_TOKEN") }
func waLang() string {
	if l := os.Getenv("WHATSAPP_TEMPLATE_LANG"); l != "" {
		return l
	}
	return "en"
}

// WAVerifyToken returns the webhook verify token.
func WAVerifyToken() string { return os.Getenv("WHATSAPP_VERIFY_TOKEN") }

// normalizePhone converts a phone number to E.164 without the leading '+'.
// Supports 10-digit Indian numbers (adds 91 prefix).
func normalizePhone(phone string) string {
	if len(phone) == 10 {
		return "91" + phone
	}
	if len(phone) > 0 && phone[0] == '+' {
		return phone[1:]
	}
	return phone
}

// ─── Core transport ───────────────────────────────────────────────────────────

// waPost marshals payload, POSTs to the Cloud API, returns WAResult.
// Returns nil result (no error) when the integration is not configured so
// callers can treat the notification as optional.
func waPost(payload interface{}) (*WAResult, error) {
	phoneID := waPhoneID()
	token := waToken()
	if phoneID == "" || token == "" {
		log.Printf("[WhatsApp] Not configured (WHATSAPP_PHONE_NUMBER_ID / WHATSAPP_ACCESS_TOKEN missing), skipping")
		return nil, nil
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/%s/messages", waAPIBase, phoneID)
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("whatsapp: request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("whatsapp: API error %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp waAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err == nil && len(apiResp.Messages) > 0 {
		return &WAResult{MessageID: apiResp.Messages[0].ID}, nil
	}
	return &WAResult{}, nil
}

// ─── Public send primitives ───────────────────────────────────────────────────

// SendText sends a free-form text message (only valid within 24-h customer service window).
func SendText(phone, message string) error {
	_, err := waPost(waTextMessage{
		MessagingProduct: "whatsapp",
		To:               normalizePhone(phone),
		Type:             "text",
		Text:             waText{Body: message},
	})
	return err
}

// SendTemplate sends a pre-approved WhatsApp template message.
// bodyParams substitutes {{1}}, {{2}}, … in the template body.
func SendTemplate(phone, templateName string, bodyParams []string) (*WAResult, error) {
	params := make([]waParam, len(bodyParams))
	for i, p := range bodyParams {
		params[i] = waParam{Type: "text", Text: p}
	}
	components := []waComponent{}
	if len(params) > 0 {
		components = append(components, waComponent{Type: "body", Parameters: params})
	}
	return waPost(waTemplateMessage{
		MessagingProduct: "whatsapp",
		To:               normalizePhone(phone),
		Type:             "template",
		Template: waTemplate{
			Name:       templateName,
			Language:   waLangCode{Code: waLang()},
			Components: components,
		},
	})
}

// SendTemplateWithButton sends a template that has a URL button whose dynamic
// suffix is provided as buttonParam (e.g. a deep-link appointment ID).
func SendTemplateWithButton(phone, templateName string, bodyParams []string, buttonParam string) (*WAResult, error) {
	params := make([]waParam, len(bodyParams))
	for i, p := range bodyParams {
		params[i] = waParam{Type: "text", Text: p}
	}
	components := []waComponent{}
	if len(params) > 0 {
		components = append(components, waComponent{Type: "body", Parameters: params})
	}
	if buttonParam != "" {
		components = append(components, waComponent{
			Type:    "button",
			SubType: "url",
			Index:   "0",
			Parameters: []waParam{
				{Type: "text", Text: buttonParam},
			},
		})
	}
	return waPost(waTemplateMessage{
		MessagingProduct: "whatsapp",
		To:               normalizePhone(phone),
		Type:             "template",
		Template: waTemplate{
			Name:       templateName,
			Language:   waLangCode{Code: waLang()},
			Components: components,
		},
	})
}

// SendInteractiveButtons sends an interactive reply-button message (max 3 buttons).
// Use for in-conversation prompts within the 24-h customer service window.
func SendInteractiveButtons(phone, headerText, bodyText, footerText string, buttons []waIButton) (*WAResult, error) {
	interactive := waInteractive{
		Type:   "button",
		Body:   waIBody{Text: bodyText},
		Action: waIAction{Buttons: buttons},
	}
	if headerText != "" {
		interactive.Header = &waIHeader{Type: "text", Text: headerText}
	}
	if footerText != "" {
		interactive.Footer = &waIFooter{Text: footerText}
	}
	return waPost(waInteractiveMessage{
		MessagingProduct: "whatsapp",
		To:               normalizePhone(phone),
		Type:             "interactive",
		Interactive:      interactive,
	})
}

// MakeButton is a helper to build a waIButton for SendInteractiveButtons.
func MakeButton(id, title string) waIButton {
	return waIButton{Type: "reply", Reply: waIReply{ID: id, Title: title}}
}

type dbWATemplate struct {
	ID        uint   `gorm:"primaryKey"`
	Type      string `gorm:"not null"`
	Title     string `gorm:"not null"`
	Content   string `gorm:"type:text;not null"`
	Variables string `gorm:"not null"`
}

func (dbWATemplate) TableName() string {
	return "recruitment_templates"
}

func resolveDBWhatsAppTemplate(templateName string, fallbackParams []string, vars map[string]string) []string {
	if database.DB == nil {
		return fallbackParams
	}
	var tmpl dbWATemplate
	err := database.DB.Where("type = ? AND title = ?", "WhatsApp", templateName).First(&tmpl).Error
	if err != nil || tmpl.Variables == "" {
		return fallbackParams
	}

	varParts := strings.Split(tmpl.Variables, ",")
	params := make([]string, 0, len(varParts))
	for _, rawVar := range varParts {
		vName := strings.TrimSpace(rawVar)
		if val, ok := vars[vName]; ok {
			params = append(params, val)
		} else {
			params = append(params, "")
		}
	}
	return params
}

// ─── WhatsApp config validation ───────────────────────────────────────────────

// ValidateWAConfig checks that WhatsApp Cloud API credentials are present at
// startup. Call this alongside emailSvc.ValidateConfig() in server/app.go.
func ValidateWAConfig() {
	if waPhoneID() == "" {
		log.Println("[whatsapp] WARNING: WHATSAPP_PHONE_NUMBER_ID not set — all WhatsApp messages will be skipped")
	}
	if waToken() == "" {
		log.Println("[whatsapp] WARNING: WHATSAPP_ACCESS_TOKEN not set — all WhatsApp messages will be skipped")
	}
}

// ─── Domain-specific helpers ─────────────────────────────────────────────────
//
// Each helper uses the outbox (reliable, retried) when the package has been
// initialised via notification.Init; otherwise it falls back to the original
// fire-and-forget goroutine so nothing breaks in test environments or before
// the DB is available.
//
// Call signatures are intentionally unchanged so no call site needs updating.

func waOutbox(jobType, phone, templateID string, params []string) bool {
	if pkg == nil {
		return false
	}
	if err := pkg.enqueueWATemplate(context.Background(), phone, jobType, templateID, params); err != nil {
		log.Printf("[WhatsApp] outbox enqueue failed (%s → %s): %v — falling back to direct send", jobType, phone, err)
		return false
	}
	return true
}

// WhatsAppOTP sends a one-time password to a staff member via an authentication
// template. Template: "drayya_otp" — body: "Your Dr.Ayya OTP is {{1}}. Valid for 5 minutes."
func WhatsAppOTP(phone, name, otp string) {
	params := resolveDBWhatsAppTemplate("drayya_otp", []string{otp}, map[string]string{
		"otp": otp,
	})
	if waOutbox("mfa_otp", phone, "drayya_otp", params) {
		return
	}
	go func() {
		if _, err := SendTemplate(phone, "drayya_otp", params); err != nil {
			log.Printf("[WhatsApp] OTP to %s: %v", phone, err)
		}
	}()
}

// WhatsAppWelcome sends a welcome message on new user registration.
// Template: "welcome_drayya" — body: "Welcome {{1}}! Your Dr.Ayya {{2}} account is ready."
func WhatsAppWelcome(phone, name, role string) {
	params := resolveDBWhatsAppTemplate("welcome_drayya", []string{name, role}, map[string]string{
		"name": name,
		"role": role,
	})
	if waOutbox("welcome", phone, "welcome_drayya", params) {
		return
	}
	go func() {
		if _, err := SendTemplate(phone, "welcome_drayya", params); err != nil {
			log.Printf("[WhatsApp] Welcome to %s: %v", phone, err)
		}
	}()
}

// WhatsAppAppointmentBooked notifies a patient when their appointment is confirmed.
// Template: "appointment_confirmed"
// Body: "Hi {{1}}, your appointment with Dr. {{2}} on {{3}} at {{4}} is confirmed."
func WhatsAppAppointmentBooked(phone, patientName, doctorName, date, timeStr string) {
	params := resolveDBWhatsAppTemplate("appointment_confirmed", []string{patientName, doctorName, date, timeStr}, map[string]string{
		"patient_name": patientName,
		"doctor_name":  doctorName,
		"date":         date,
		"time":         timeStr,
	})
	if waOutbox("appointment_confirmed", phone, "appointment_confirmed", params) {
		return
	}
	go func() {
		if _, err := SendTemplate(phone, "appointment_confirmed", params); err != nil {
			log.Printf("[WhatsApp] AppointmentBooked: %v", err)
		}
	}()
}

// WhatsAppDoctorNewAppointment notifies a doctor of a newly booked appointment.
// Template: "doctor_new_appointment"
// Body: "Dr. {{1}}, a new appointment has been booked by {{2}} on {{3}} at {{4}}."
func WhatsAppDoctorNewAppointment(phone, doctorName, patientName, date, timeStr string) {
	params := resolveDBWhatsAppTemplate("doctor_new_appointment", []string{doctorName, patientName, date, timeStr}, map[string]string{
		"doctor_name":  doctorName,
		"patient_name": patientName,
		"date":         date,
		"time":         timeStr,
	})
	if waOutbox("doctor_new_appointment", phone, "doctor_new_appointment", params) {
		return
	}
	go func() {
		if _, err := SendTemplate(phone, "doctor_new_appointment", params); err != nil {
			log.Printf("[WhatsApp] DoctorNewAppointment: %v", err)
		}
	}()
}

// WhatsAppAppointmentReminder sends a 1-hour reminder to the patient.
// Template: "appointment_reminder_1hr"
// Body: "Reminder: Hi {{1}}, your consultation with Dr. {{2}} starts in 1 hour at {{3}}."
func WhatsAppAppointmentReminder(phone, patientName, doctorName, timeStr string) {
	params := resolveDBWhatsAppTemplate("appointment_reminder_1hr", []string{patientName, doctorName, timeStr}, map[string]string{
		"patient_name": patientName,
		"doctor_name":  doctorName,
		"time":         timeStr,
	})
	if waOutbox("appointment_reminder", phone, "appointment_reminder_1hr", params) {
		return
	}
	go func() {
		if _, err := SendTemplate(phone, "appointment_reminder_1hr", params); err != nil {
			log.Printf("[WhatsApp] AppointmentReminder: %v", err)
		}
	}()
}

// WhatsAppAppointmentCancelled notifies the patient that their appointment was cancelled.
// Template: "appointment_cancelled"
// Body: "Hi {{1}}, your appointment with Dr. {{2}} on {{3}} has been cancelled."
func WhatsAppAppointmentCancelled(phone, patientName, doctorName, date string) {
	params := resolveDBWhatsAppTemplate("appointment_cancelled", []string{patientName, doctorName, date}, map[string]string{
		"patient_name": patientName,
		"doctor_name":  doctorName,
		"date":         date,
	})
	if waOutbox("appointment_cancelled", phone, "appointment_cancelled", params) {
		return
	}
	go func() {
		if _, err := SendTemplate(phone, "appointment_cancelled", params); err != nil {
			log.Printf("[WhatsApp] AppointmentCancelled: %v", err)
		}
	}()
}

// WhatsAppPaymentReceipt notifies a patient after a consultation payment.
// Template: "payment_receipt"
// Body: "Hi {{1}}, ₹{{2}} paid successfully for {{3}}. Your appointment is confirmed."
func WhatsAppPaymentReceipt(phone, patientName, amount, description string) {
	params := resolveDBWhatsAppTemplate("payment_receipt", []string{patientName, amount, description}, map[string]string{
		"patient_name": patientName,
		"amount":       amount,
		"description":  description,
	})
	if waOutbox("payment_receipt", phone, "payment_receipt", params) {
		return
	}
	go func() {
		if _, err := SendTemplate(phone, "payment_receipt", params); err != nil {
			log.Printf("[WhatsApp] PaymentReceipt: %v", err)
		}
	}()
}

// WhatsAppWalletTopUp notifies a patient after a successful wallet top-up.
// Template: "wallet_topup"
// Body: "Hi {{1}}, ₹{{2}} has been added to your Dr.Ayya wallet. New balance: ₹{{3}}."
func WhatsAppWalletTopUp(phone, name, amount, newBalance string) {
	params := resolveDBWhatsAppTemplate("wallet_topup", []string{name, amount, newBalance}, map[string]string{
		"name":        name,
		"amount":      amount,
		"new_balance": newBalance,
	})
	if waOutbox("wallet_topup", phone, "wallet_topup", params) {
		return
	}
	go func() {
		if _, err := SendTemplate(phone, "wallet_topup", params); err != nil {
			log.Printf("[WhatsApp] WalletTopUp: %v", err)
		}
	}()
}

// WhatsAppCallStarted notifies the patient that the doctor has started the session.
// Template: "call_started" + optional deep-link text follow-up.
func WhatsAppCallStarted(phone, doctorName, roomLink string) {
	params := resolveDBWhatsAppTemplate("call_started", []string{doctorName}, map[string]string{
		"doctor_name": doctorName,
	})
	go func() {
		if _, err := SendTemplate(phone, "call_started", params); err != nil {
			log.Printf("[WhatsApp] CallStarted template: %v", err)
		}
		if roomLink != "" {
			time.Sleep(1 * time.Second)
			_ = SendText(phone, "Click here to join: "+roomLink)
		}
	}()
}

// WhatsAppPrescriptionIssued notifies a patient when a prescription is issued.
// Template: "prescription_issued"
// Body: "Hi {{1}}, Dr. {{2}} has issued a prescription for you on {{3}}. View it in the Dr.Ayya app."
func WhatsAppPrescriptionIssued(phone, patientName, doctorName, date string) {
	params := resolveDBWhatsAppTemplate("prescription_issued", []string{patientName, doctorName, date}, map[string]string{
		"patient_name": patientName,
		"doctor_name":  doctorName,
		"date":         date,
	})
	if waOutbox("prescription_issued", phone, "prescription_issued", params) {
		return
	}
	go func() {
		if _, err := SendTemplate(phone, "prescription_issued", params); err != nil {
			log.Printf("[WhatsApp] PrescriptionIssued: %v", err)
		}
	}()
}

// WhatsAppDoctorPayout notifies a doctor when a payout is processed.
// Template: "doctor_payout_processed"
// Body: "Hi Dr. {{1}}, ₹{{2}} has been transferred to your bank account. Ref: {{3}}."
func WhatsAppDoctorPayout(phone, doctorName, amount, payoutRef string) {
	params := resolveDBWhatsAppTemplate("doctor_payout_processed", []string{doctorName, amount, payoutRef}, map[string]string{
		"doctor_name": doctorName,
		"amount":      amount,
		"payout_ref":  payoutRef,
	})
	if waOutbox("doctor_payout", phone, "doctor_payout_processed", params) {
		return
	}
	go func() {
		if _, err := SendTemplate(phone, "doctor_payout_processed", params); err != nil {
			log.Printf("[WhatsApp] DoctorPayout: %v", err)
		}
	}()
}

// WhatsAppInstantTimeout notifies the patient when an instant consultation
// times out and they receive an automatic refund.
// Template: "instant_timeout_refund"
// Body: "Hi {{1}}, we couldn't match you with a doctor. ₹{{2}} has been refunded to your wallet."
func WhatsAppInstantTimeout(phone, patientName, amount string) {
	params := resolveDBWhatsAppTemplate("instant_timeout_refund", []string{patientName, amount}, map[string]string{
		"patient_name": patientName,
		"amount":       amount,
	})
	if waOutbox("instant_timeout_refund", phone, "instant_timeout_refund", params) {
		return
	}
	go func() {
		if _, err := SendTemplate(phone, "instant_timeout_refund", params); err != nil {
			log.Printf("[WhatsApp] InstantTimeout: %v", err)
		}
	}()
}

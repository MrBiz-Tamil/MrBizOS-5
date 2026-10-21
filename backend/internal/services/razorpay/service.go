package razorpay

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	baseURL    = "https://api.razorpay.com/v1"
	payoutBase = "https://api.razorpay.com/v1"
)

func keyID() string     { return os.Getenv("RAZORPAY_KEY_ID") }
func keySecret() string { return os.Getenv("RAZORPAY_KEY_SECRET") }
func payoutKeyID() string {
	if v := os.Getenv("RAZORPAY_PAYOUT_KEY_ID"); v != "" {
		return v
	}
	return keyID()
}
func payoutKeySecret() string {
	if v := os.Getenv("RAZORPAY_PAYOUT_KEY_SECRET"); v != "" {
		return v
	}
	return keySecret()
}

func basicAuth(id, secret string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(id+":"+secret))
}

func do(method, url, auth string, body interface{}) (map[string]interface{}, error) {
	// Mock mode for development with placeholder keys
	if keyID() == "rzp_test_xxxxxxxxxxxx" || keyID() == "" {
		// Return mock success responses for critical endpoints
		if strings.HasSuffix(url, "/orders") && method == "POST" {
			return map[string]interface{}{
				"id":       "order_mock_" + time.Now().Format("20060102150405"),
				"entity":   "order",
				"amount":   body.(map[string]interface{})["amount"],
				"currency": "INR",
				"receipt":  body.(map[string]interface{})["receipt"],
				"status":   "created",
			}, nil
		}
		// Default mock response for other calls
		return map[string]interface{}{
			"id":     "mock_id_" + time.Now().Format("20060102150405"),
			"status": "mock_success",
		}, nil
	}

	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("razorpay: invalid response: %s", string(respBody))
	}
	if resp.StatusCode >= 400 {
		errMsg := "razorpay API error"
		if e, ok := result["error"].(map[string]interface{}); ok {
			if desc, ok := e["description"].(string); ok {
				errMsg = desc
			}
		}
		return nil, fmt.Errorf("razorpay: %s (status %d)", errMsg, resp.StatusCode)
	}
	return result, nil
}

// -------------------------------------------------------------------
// Orders
// -------------------------------------------------------------------

// CreateOrder creates a Razorpay order and returns the raw response map.
// amount is in paise (INR × 100).
func CreateOrder(amountPaise int64, receipt string) (map[string]interface{}, error) {
	return do("POST", baseURL+"/orders", basicAuth(keyID(), keySecret()), map[string]interface{}{
		"amount":   amountPaise,
		"currency": "INR",
		"receipt":  receipt,
	})
}

// VerifyPaymentSignature validates a Razorpay payment callback.
// Returns true when the HMAC is valid.
func VerifyPaymentSignature(orderID, paymentID, signature string) bool {
	data := orderID + "|" + paymentID
	mac := hmac.New(sha256.New, []byte(keySecret()))
	mac.Write([]byte(data))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// -------------------------------------------------------------------
// Payouts  (requires Razorpay Route / X account)
// -------------------------------------------------------------------

type ContactInput struct {
	Name   string // doctor full name
	Email  string
	Phone  string
	Type   string // "vendor" for doctor payouts
	RefID  string // unique doctor reference (doctor:userID)
}

type FundAccountInput struct {
	ContactID   string
	AccountType string // "bank_account" or "vpa"
	// Bank account fields
	Name        string
	IFSC        string
	AccountNo   string
	// VPA (UPI) field
	VPA         string
}

type PayoutInput struct {
	FundAccountID string
	AmountPaise   int64
	Currency      string // "INR"
	Mode          string // "IMPS" | "NEFT" | "UPI"
	Purpose       string // "payout"
	Narration     string
	RefID         string // idempotency
}

// CreateContact creates a Razorpay contact (for doctor).
func CreateContact(in ContactInput) (string, error) {
	res, err := do("POST", payoutBase+"/contacts", basicAuth(payoutKeyID(), payoutKeySecret()), map[string]interface{}{
		"name":         in.Name,
		"email":        in.Email,
		"contact":      in.Phone,
		"type":         in.Type,
		"reference_id": in.RefID,
	})
	if err != nil {
		return "", err
	}
	id, _ := res["id"].(string)
	return id, nil
}

// CreateFundAccount registers a bank account or UPI VPA for a contact.
func CreateFundAccount(in FundAccountInput) (string, error) {
	body := map[string]interface{}{
		"contact_id":   in.ContactID,
		"account_type": in.AccountType,
	}
	if in.AccountType == "bank_account" {
		body["bank_account"] = map[string]string{
			"name":           in.Name,
			"ifsc":           in.IFSC,
			"account_number": in.AccountNo,
		}
	} else {
		body["vpa"] = map[string]string{"address": in.VPA}
	}

	res, err := do("POST", payoutBase+"/fund_accounts", basicAuth(payoutKeyID(), payoutKeySecret()), body)
	if err != nil {
		return "", err
	}
	id, _ := res["id"].(string)
	return id, nil
}

// CreatePayout transfers money to a fund account.
func CreatePayout(in PayoutInput) (string, error) {
	currency := in.Currency
	if currency == "" {
		currency = "INR"
	}
	mode := in.Mode
	if mode == "" {
		mode = "IMPS"
	}
	purpose := in.Purpose
	if purpose == "" {
		purpose = "payout"
	}

	res, err := do("POST", payoutBase+"/payouts", basicAuth(payoutKeyID(), payoutKeySecret()), map[string]interface{}{
		"account_number":  os.Getenv("RAZORPAY_PAYOUT_ACCOUNT_NUMBER"),
		"fund_account_id": in.FundAccountID,
		"amount":          in.AmountPaise,
		"currency":        currency,
		"mode":            mode,
		"purpose":         purpose,
		"narration":       in.Narration,
		"reference_id":    in.RefID,
		"queue_if_low_balance": true,
	})
	if err != nil {
		return "", err
	}
	id, _ := res["id"].(string)
	return id, nil
}

// -------------------------------------------------------------------
// Subscriptions
// -------------------------------------------------------------------

type PlanInput struct {
	Period   string // "daily", "weekly", "monthly", "yearly"
	Interval int    // e.g. 1
	Item     struct {
		Name        string
		Amount      int64  // in paise
		Currency    string // "INR"
		Description string
	}
}

// CreatePlan creates a Razorpay billing plan.
func CreatePlan(in PlanInput) (string, error) {
	body := map[string]interface{}{
		"period":   in.Period,
		"interval": in.Interval,
		"item": map[string]interface{}{
			"name":        in.Item.Name,
			"amount":      in.Item.Amount,
			"currency":    in.Item.Currency,
			"description": in.Item.Description,
		},
	}
	res, err := do("POST", baseURL+"/plans", basicAuth(keyID(), keySecret()), body)
	if err != nil {
		return "", err
	}
	id, _ := res["id"].(string)
	return id, nil
}

type SubscriptionInput struct {
	PlanID        string
	TotalCount    int // Number of billing cycles
	Quantity      int // Optional, defaults to 1
	StartAt       int64 // Unix timestamp, optional
	ExpireBy      int64 // Unix timestamp, optional
	CustomerNotify int  // 1 to notify customer via email/sms
}

// CreateSubscription creates a subscription for a plan.
func CreateSubscription(in SubscriptionInput) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"plan_id":         in.PlanID,
		"total_count":     in.TotalCount,
		"customer_notify": in.CustomerNotify,
	}
	if in.Quantity > 0 {
		body["quantity"] = in.Quantity
	}
	if in.StartAt > 0 {
		body["start_at"] = in.StartAt
	}
	if in.ExpireBy > 0 {
		body["expire_by"] = in.ExpireBy
	}

	return do("POST", baseURL+"/subscriptions", basicAuth(keyID(), keySecret()), body)
}

// CancelSubscription cancels an active subscription.
func CancelSubscription(subscriptionID string, cancelAtCycleEnd bool) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"cancel_at_cycle_end": cancelAtCycleEnd,
	}
	return do("POST", baseURL+"/subscriptions/"+subscriptionID+"/cancel", basicAuth(keyID(), keySecret()), body)
}

// VerifyWebhookSignature verifies Razorpay webhook signature.
func VerifyWebhookSignature(payloadBody []byte, signature, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payloadBody)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

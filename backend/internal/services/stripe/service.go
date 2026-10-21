// Package stripe is a thin client for Stripe global card payments, mirroring the
// razorpay service shape (env-keyed, mock mode for dev). We use PaymentIntents:
// create one in the customer's presentment currency, the client confirms it with
// Stripe.js, and the server verifies by retrieving the intent (status=succeeded)
// before posting to the ledger.
package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.stripe.com/v1"

func secretKey() string     { return os.Getenv("STRIPE_SECRET_KEY") }
func webhookSecret() string { return os.Getenv("STRIPE_WEBHOOK_SECRET") }

// mockMode is on when no real key is configured, so dev/tests don't hit Stripe.
func mockMode() bool {
	k := secretKey()
	return k == "" || strings.HasPrefix(k, "sk_test_xxxx")
}

func do(method, path string, form url.Values) (map[string]interface{}, error) {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequest(method, baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+secretKey())
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("stripe: invalid response: %s", string(raw))
	}
	if resp.StatusCode >= 400 {
		msg := "stripe API error"
		if e, ok := result["error"].(map[string]interface{}); ok {
			if d, ok := e["message"].(string); ok {
				msg = d
			}
		}
		return nil, fmt.Errorf("stripe: %s (status %d)", msg, resp.StatusCode)
	}
	return result, nil
}

// CreatePaymentIntent creates a PaymentIntent for amountMinor in the given
// (lowercase) currency, attaching metadata (e.g. our reference + INR price).
// Returns the raw intent map (includes id + client_secret).
func CreatePaymentIntent(amountMinor int64, currency string, metadata map[string]string) (map[string]interface{}, error) {
	if mockMode() {
		id := "pi_mock_" + time.Now().Format("20060102150405")
		return map[string]interface{}{
			"id":            id,
			"object":        "payment_intent",
			"amount":        amountMinor,
			"currency":      strings.ToLower(currency),
			"status":        "requires_payment_method",
			"client_secret": id + "_secret_mock",
		}, nil
	}
	form := url.Values{}
	form.Set("amount", fmt.Sprintf("%d", amountMinor))
	form.Set("currency", strings.ToLower(currency))
	form.Set("automatic_payment_methods[enabled]", "true")
	for k, v := range metadata {
		form.Set("metadata["+k+"]", v)
	}
	return do("POST", "/payment_intents", form)
}

// PaymentIntent is the verified subset we care about.
type PaymentIntent struct {
	ID       string
	Status   string // "succeeded" when paid
	Amount   int64  // minor units, presentment currency
	Currency string
	Metadata map[string]string
}

// RetrievePaymentIntent fetches an intent server-side (authenticated by the
// secret key) so we can trust its status without relying on the client.
func RetrievePaymentIntent(id string) (*PaymentIntent, error) {
	if mockMode() {
		// In dev/test, treat any retrieved intent as succeeded.
		return &PaymentIntent{ID: id, Status: "succeeded"}, nil
	}
	res, err := do("GET", "/payment_intents/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}
	pi := &PaymentIntent{ID: id, Metadata: map[string]string{}}
	if s, ok := res["status"].(string); ok {
		pi.Status = s
	}
	if a, ok := res["amount"].(float64); ok {
		pi.Amount = int64(a)
	}
	if c, ok := res["currency"].(string); ok {
		pi.Currency = c
	}
	if m, ok := res["metadata"].(map[string]interface{}); ok {
		for k, v := range m {
			if vs, ok := v.(string); ok {
				pi.Metadata[k] = vs
			}
		}
	}
	return pi, nil
}

// WebhookEvent is the slim view of a Stripe event we act on.
type WebhookEvent struct {
	Type            string            // e.g. "payment_intent.succeeded"
	PaymentIntentID string            // the pi_... id
	Metadata        map[string]string // includes our "ref" (LAB-5, STORE-3, INV-9, PHARM-2)
}

// VerifyWebhook validates the Stripe-Signature header (HMAC-SHA256 over
// "timestamp.payload" with the webhook secret) and returns the parsed event.
// In mock mode (no webhook secret) it parses without verifying so dev works.
func VerifyWebhook(payload []byte, sigHeader string) (*WebhookEvent, error) {
	secret := webhookSecret()
	if secret != "" {
		if !validSignature(payload, sigHeader, secret) {
			return nil, fmt.Errorf("stripe webhook signature verification failed")
		}
	}

	var evt struct {
		Type string `json:"type"`
		Data struct {
			Object struct {
				ID       string            `json:"id"`
				Metadata map[string]string `json:"metadata"`
			} `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &evt); err != nil {
		return nil, fmt.Errorf("stripe webhook: bad payload: %w", err)
	}
	return &WebhookEvent{
		Type:            evt.Type,
		PaymentIntentID: evt.Data.Object.ID,
		Metadata:        evt.Data.Object.Metadata,
	}, nil
}

// validSignature implements Stripe's signature scheme: the header is
// "t=<ts>,v1=<hmac>"; the signed payload is "<ts>.<body>".
func validSignature(payload []byte, sigHeader, secret string) bool {
	var ts, v1 string
	for _, part := range strings.Split(sigHeader, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			ts = kv[1]
		case "v1":
			v1 = kv[1]
		}
	}
	if ts == "" || v1 == "" {
		return false
	}
	// Reject stale timestamps (> 5 min) to blunt replay attacks.
	if n, err := strconv.ParseInt(ts, 10, 64); err == nil {
		if time.Since(time.Unix(n, 0)) > 5*time.Minute {
			return false
		}
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "." + string(payload)))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(v1))
}

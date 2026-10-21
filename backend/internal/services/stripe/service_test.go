package stripe

import "testing"

// With no STRIPE_SECRET_KEY, the service runs in mock mode so dev/tests don't
// hit the network.
func TestCreatePaymentIntent_MockMode(t *testing.T) {
	intent, err := CreatePaymentIntent(1205, "usd", map[string]string{"ref": "LAB-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if intent["id"] == nil || intent["client_secret"] == nil {
		t.Fatalf("mock intent missing id/client_secret: %+v", intent)
	}
	if intent["amount"] != int64(1205) {
		t.Fatalf("amount mismatch: %v", intent["amount"])
	}
}

func TestRetrievePaymentIntent_MockSucceeds(t *testing.T) {
	pi, err := RetrievePaymentIntent("pi_test_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pi.Status != "succeeded" {
		t.Fatalf("expected succeeded in mock mode, got %q", pi.Status)
	}
}

// Without a webhook secret, VerifyWebhook parses the event (no signature check).
func TestVerifyWebhook_ParsesEvent(t *testing.T) {
	payload := []byte(`{"type":"payment_intent.succeeded","data":{"object":{"id":"pi_9","metadata":{"ref":"STORE-3"}}}}`)
	evt, err := VerifyWebhook(payload, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.Type != "payment_intent.succeeded" || evt.PaymentIntentID != "pi_9" || evt.Metadata["ref"] != "STORE-3" {
		t.Fatalf("parsed event wrong: %+v", evt)
	}
}

// A bad signature is rejected when a webhook secret IS configured.
func TestVerifyWebhook_RejectsBadSignature(t *testing.T) {
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")
	_, err := VerifyWebhook([]byte(`{"type":"x"}`), "t=123,v1=deadbeef")
	if err == nil {
		t.Fatal("expected signature rejection")
	}
}

package lab

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"drayya-core/internal/domain/finance"
)

// Drives the REAL HTTP workflow through the actual lab handlers + routes:
//   create booking → Stripe initiate → Stripe verify → ledger posted.
// Stripe runs in mock mode (no keys), so verify treats the intent as succeeded
// — exactly the path a live payment takes after Stripe confirms the card.
func TestLabStripeWorkflow_HTTP(t *testing.T) {
	svc, db := setupLabTest(t) // seeds partner, center, test, center-pricing (₹1000)
	h := NewHandler(svc)

	// Minimal app with a fake-auth middleware injecting the patient's user_id.
	app := fiber.New()
	auth := func(c *fiber.Ctx) error { c.Locals("user_id", uint(42)); return c.Next() }
	app.Post("/lab/bookings", auth, h.CreateBooking)
	app.Post("/lab/bookings/:id/pay/stripe/initiate", auth, h.InitiateLabStripePayment)
	app.Post("/lab/bookings/:id/pay/stripe/verify", auth, h.VerifyLabStripePayment)

	post := func(path, body string) map[string]interface{} {
		req := httptest.NewRequest("POST", path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		raw, _ := io.ReadAll(resp.Body)
		require.Equalf(t, 200, statusOK(resp.StatusCode), "path %s -> %d: %s", path, resp.StatusCode, string(raw))
		var env map[string]interface{}
		require.NoError(t, json.Unmarshal(raw, &env))
		return env
	}

	// 1. Patient books a test (price resolved server-side → ₹1000).
	env := post("/lab/bookings", `{"lab_center_id":1,"lab_test_id":1,"booking_type":"lab_visit","preferred_date":"2026-06-10","test_price":1}`)
	booking := dataMap(t, env)
	bookingID := uint(booking["id"].(float64))
	assert.Equal(t, float64(100000), booking["total_paise"], "server pricing wins (₹1000, not the ₹0.01 the client sent)")

	// 2. Initiate a Stripe payment in USD.
	env = post("/lab/bookings/"+itoa(bookingID)+"/pay/stripe/initiate", `{"currency":"USD"}`)
	intent := dataMap(t, env)
	intentID, _ := intent["id"].(string)
	require.NotEmpty(t, intentID, "intent id returned")

	// 3. Verify (Stripe mock reports succeeded) → posts to the ledger.
	post("/lab/bookings/"+itoa(bookingID)+"/pay/stripe/verify", `{"payment_intent_id":"`+intentID+`"}`)

	// 4. Booking is paid and the INR amount is in Asset:Stripe with the split.
	var b LabBooking
	require.NoError(t, db.First(&b, bookingID).Error)
	assert.Equal(t, "paid", b.PaymentStatus)
	assert.Equal(t, int64(100000), acctBalance(t, db, finance.AcctStripe))
	assert.Equal(t, int64(19047), acctBalance(t, db, finance.RevLab))

	// 5. The ledger reconciles cleanly after the workflow.
	report := finance.NewService(finance.NewRepository(db)).Reconcile(db)
	assert.True(t, report.AllPassed, "ledger must reconcile after a real payment: %+v", report.Results)
}

func statusOK(code int) int {
	if code == 201 {
		return 200
	}
	return code
}

func dataMap(t *testing.T, env map[string]interface{}) map[string]interface{} {
	if d, ok := env["data"].(map[string]interface{}); ok {
		return d
	}
	return env
}

func itoa(u uint) string {
	if u == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for u > 0 {
		i--
		b[i] = byte('0' + u%10)
		u /= 10
	}
	return string(b[i:])
}

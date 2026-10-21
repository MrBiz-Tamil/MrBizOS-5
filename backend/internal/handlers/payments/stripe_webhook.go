// Package payments hosts the Stripe webhook — the resilient confirmation path.
// Even if a customer closes the tab right after paying, Stripe re-delivers the
// payment_intent.succeeded event here; we verify its signature and post the
// payment to the ledger by reading the "ref" metadata we set at intent creation.
package payments

import (
	"context"
	"log"
	"strconv"
	"strings"

	"drayya-core/internal/domain/lab"
	"drayya-core/internal/domain/store"
	"drayya-core/internal/services/stripe"

	"github.com/gofiber/fiber/v2"
)

type StripeWebhookHandler struct {
	lab   lab.Service
	store store.Service
}

func NewStripeWebhookHandler(labSvc lab.Service, storeSvc store.Service) *StripeWebhookHandler {
	return &StripeWebhookHandler{lab: labSvc, store: storeSvc}
}

// RegisterRoutes mounts the webhook UNAUTHENTICATED (Stripe calls it directly)
// but it is signature-verified inside Handle.
func (h *StripeWebhookHandler) RegisterRoutes(app *fiber.App) {
	app.Post("/webhooks/stripe", h.Handle)
}

func (h *StripeWebhookHandler) Handle(c *fiber.Ctx) error {
	evt, err := stripe.VerifyWebhook(c.Body(), c.Get("Stripe-Signature"))
	if err != nil {
		// Untrusted/garbled — reject so Stripe shows a delivery failure.
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	if evt.Type != "payment_intent.succeeded" {
		return c.SendStatus(fiber.StatusOK) // ack and ignore
	}

	ref := evt.Metadata["ref"] // e.g. "LAB-5", "STORE-3"
	parts := strings.SplitN(ref, "-", 2)
	if len(parts) != 2 {
		return c.SendStatus(fiber.StatusOK)
	}
	id64, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return c.SendStatus(fiber.StatusOK)
	}
	id := uint(id64)
	ctx := context.Background()

	// Confirm* calls are idempotent (reference-keyed), so a re-delivered webhook
	// or a double client+webhook confirmation posts to the ledger only once.
	switch parts[0] {
	case "LAB":
		if _, e := h.lab.ConfirmLabPayment(ctx, id, "stripe", evt.PaymentIntentID); e != nil {
			log.Printf("[stripe:webhook] LAB-%d confirm failed: %v", id, e)
		}
	case "STORE":
		if _, e := h.store.ConfirmStorePayment(ctx, id, "stripe", evt.PaymentIntentID); e != nil {
			log.Printf("[stripe:webhook] STORE-%d confirm failed: %v", id, e)
		}
	default:
		log.Printf("[stripe:webhook] unhandled ref prefix %q", parts[0])
	}
	return c.SendStatus(fiber.StatusOK)
}

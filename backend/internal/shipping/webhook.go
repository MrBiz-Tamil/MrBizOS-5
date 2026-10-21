package shipping

// webhook.go — Inbound tracking webhooks from 3PL providers.
//
// Shiprocket sends a POST to our webhook URL whenever a shipment status
// changes (picked up, out for delivery, delivered, RTO, etc.).
//
// India Post does not send webhooks natively; the endpoint here accepts the
// same normalised schema from any polling bridge you build.
//
// Routes (public — no auth, verified by secret token in header):
//
//   POST /api/v1/shipping/webhook/shiprocket
//   POST /api/v1/shipping/webhook/indiapost
//
// Webhook secret: SHIPPING_WEBHOOK_SECRET env var.
// Shiprocket sends it as the `X-Shiprocket-Hmac-Sha256` header (SHA-256 HMAC
// of the raw body using your secret).  We do a constant-time comparison.
//
// Status mapping (Shiprocket → internal):
//   "Pickup Scheduled"       → "assigned"
//   "Picked Up"              → "picked_up"
//   "In Transit"             → "en_route"
//   "Out For Delivery"       → "arrived"
//   "Delivered"              → "completed"
//   "RTO Initiated"          → "failed"
//   "RTO Delivered"          → "failed"
//   "Cancelled"              → "cancelled"

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/utils/response"
)

// RegisterWebhookRoutes wires the 3PL webhook endpoints.
// These are intentionally placed outside the /api/v1 auth middleware since
// Shiprocket calls them from their servers (no JWT).
func RegisterWebhookRoutes(app fiber.Router) {
	wh := app.Group("/api/v1/shipping/webhook")
	wh.Post("/shiprocket", ShiprocketWebhookHandler)
	wh.Post("/indiapost", IndiaPostWebhookHandler)
}

// ─── Shiprocket webhook ───────────────────────────────────────────────────────

// ShiprocketWebhookPayload is the subset of fields Shiprocket sends on each
// status-change event.  Full schema: https://apiv2.shiprocket.in/webhooks
type ShiprocketWebhookPayload struct {
	AWB           string `json:"awb"`
	OrderID       string `json:"order_id"`       // our order_id we sent during creation
	CurrentStatus string `json:"current_status"` // e.g. "Delivered"
	StatusCode    string `json:"status_code"`
	Location      string `json:"location"`
	Timestamp     string `json:"scans_timestamp"`
	EDD           string `json:"etd"` // estimated delivery date
}

// ShiprocketWebhookHandler processes inbound delivery status events from Shiprocket.
// POST /api/v1/shipping/webhook/shiprocket
func ShiprocketWebhookHandler(c *fiber.Ctx) error {
	// 1. Verify HMAC signature
	if err := verifyWebhookHMAC(c, "X-Shiprocket-Hmac-Sha256"); err != nil {
		log.Printf("[Webhook/Shiprocket] HMAC verification failed: %v", err)
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	var payload ShiprocketWebhookPayload
	if err := c.BodyParser(&payload); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_PAYLOAD", "Cannot parse webhook body")
	}
	if payload.AWB == "" {
		return c.SendStatus(fiber.StatusOK) // Shiprocket sends test pings with empty AWB — ack and ignore
	}

	log.Printf("[Webhook/Shiprocket] AWB=%s status=%q location=%q", payload.AWB, payload.CurrentStatus, payload.Location)

	processTrackingUpdate(c.Context(), payload.AWB, payload.CurrentStatus, payload.Location, "shiprocket")

	return c.SendStatus(fiber.StatusOK)
}

// ─── India Post webhook ───────────────────────────────────────────────────────

// IndiaPostWebhookPayload uses the same normalised schema so the processing
// logic is shared.  A polling bridge script can POST to this endpoint with
// the same fields after calling India Post's tracking API.
type IndiaPostWebhookPayload struct {
	AWB           string `json:"awb"`
	CurrentStatus string `json:"current_status"`
	Location      string `json:"location"`
}

// IndiaPostWebhookHandler processes inbound (or bridge-polled) tracking events
// from India Post.
// POST /api/v1/shipping/webhook/indiapost
func IndiaPostWebhookHandler(c *fiber.Ctx) error {
	if err := verifyWebhookHMAC(c, "X-Webhook-Secret"); err != nil {
		log.Printf("[Webhook/IndiaPost] HMAC verification failed: %v", err)
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	var payload IndiaPostWebhookPayload
	if err := c.BodyParser(&payload); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_PAYLOAD", "Cannot parse webhook body")
	}
	if payload.AWB == "" {
		return c.SendStatus(fiber.StatusOK)
	}

	log.Printf("[Webhook/IndiaPost] AWB=%s status=%q location=%q", payload.AWB, payload.CurrentStatus, payload.Location)

	processTrackingUpdate(c.Context(), payload.AWB, payload.CurrentStatus, payload.Location, "indiapost")

	return c.SendStatus(fiber.StatusOK)
}

// ─── Shared processing ────────────────────────────────────────────────────────

// statusMap normalises 3PL status strings to internal delivery statuses.
var shiprocketStatusMap = map[string]string{
	"pickup scheduled":   "assigned",
	"pickup generated":   "assigned",
	"picked up":          "picked_up",
	"in transit":         "en_route",
	"out for delivery":   "arrived",
	"delivered":          "completed",
	"rto initiated":      "failed",
	"rto delivered":      "failed",
	"rto acknowledged":   "failed",
	"cancelled":          "cancelled",
	"lost":               "failed",
	"damaged":            "failed",
}

var indiaPostStatusMap = map[string]string{
	"item booked":       "assigned",
	"item received":     "picked_up",
	"item dispatched":   "en_route",
	"out for delivery":  "arrived",
	"item delivered":    "completed",
	"item returned":     "failed",
}

// processTrackingUpdate looks up the delivery by AWB, maps the 3PL status to
// our internal status, updates the DB, and pushes WS + FCM to the patient.
func processTrackingUpdate(ctx context.Context, awb, rawStatus, location, provider string) {
	// Find the delivery row by matching tracking_id on the parent order
	var order pharmacy.PharmacyOrder
	if err := database.DB.WithContext(ctx).
		Where("tracking_id = ?", awb).
		First(&order).Error; err != nil {
		log.Printf("[Webhook/%s] AWB %s: no matching order found", provider, awb)
		return
	}

	// Find the associated CourierDelivery row
	var delivery pharmacy.CourierDelivery
	if err := database.DB.WithContext(ctx).
		Where("order_id = ?", order.ID).
		Order("created_at DESC").
		First(&delivery).Error; err != nil {
		log.Printf("[Webhook/%s] AWB %s: no delivery row for order %d", provider, awb, order.ID)
		return
	}

	// Map the 3PL status based on provider
	var internalStatus string
	var ok bool
	if provider == "shiprocket" {
		internalStatus, ok = shiprocketStatusMap[strings.ToLower(rawStatus)]
	} else if provider == "indiapost" {
		internalStatus, ok = indiaPostStatusMap[strings.ToLower(rawStatus)]
	}
	if !ok {
		// Unknown status — store the raw value and skip state machine updates
		log.Printf("[Webhook/%s] AWB %s: unknown status %q — storing as note", provider, awb, rawStatus)
		database.DB.WithContext(ctx).Model(&delivery).Update("delivery_notes",
			fmt.Sprintf("3PL update: %s @ %s [%s]", rawStatus, location, provider))
		return
	}

	// Skip if already at a terminal state or moving backwards
	terminalStates := map[string]bool{"completed": true, "failed": true, "cancelled": true}
	if terminalStates[delivery.Status] {
		log.Printf("[Webhook/%s] AWB %s: delivery %d already terminal (%s), skipping", provider, awb, delivery.ID, delivery.Status)
		return
	}

	now := time.Now()

	// Build the update map
	updates := map[string]interface{}{
		"status":         internalStatus,
		"current_address": location,
		"delivery_notes": fmt.Sprintf("3PL: %s @ %s [%s]", rawStatus, location, provider),
		"updated_at":     now,
	}
	if internalStatus == "completed" {
		updates["actual_delivery_time"] = now
	}

	if err := database.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&delivery).Updates(updates).Error; err != nil {
			return err
		}
		// Keep parent order status in sync
		orderStatus := internalStatus
		if internalStatus == "en_route" || internalStatus == "arrived" {
			orderStatus = "in_transit"
		}
		return tx.Model(&pharmacy.PharmacyOrder{}).
			Where("id = ?", order.ID).
			Update("status", orderStatus).Error
	}); err != nil {
		log.Printf("[Webhook/%s] AWB %s: DB update failed: %v", provider, awb, err)
		return
	}

	log.Printf("[Webhook/%s] order %d delivery %d: %s → %s (location: %s)",
		provider, order.ID, delivery.ID, delivery.Status, internalStatus, location)

	// Push WS + FCM to patient
	delivery.Status = internalStatus
	SendDeliveryStatusNotification(ctx, &delivery, order.UserID)

	// Alert ops when 3PL marks an order as failed/RTO
	if internalStatus == "failed" {
		alertOps(ctx, fmt.Sprintf(
			"[3PL %s] 🔴 Order #%s (AWB %s) returned/failed — manual action required",
			strings.ToUpper(provider), order.OrderNumber, awb,
		))
	}
}

// ─── HMAC verification ────────────────────────────────────────────────────────

// verifyWebhookHMAC validates the HMAC-SHA256 signature on the raw request body
// using SHIPPING_WEBHOOK_SECRET.  Returns nil if valid or if no secret is configured
// (so dev environments without a secret skip verification gracefully).
func verifyWebhookHMAC(c *fiber.Ctx, headerName string) error {
	secret := os.Getenv("SHIPPING_WEBHOOK_SECRET")
	if secret == "" {
		return nil // no secret configured — skip verification (dev/staging only)
	}

	sig := c.Get(headerName)
	if sig == "" {
		return fmt.Errorf("missing %s header", headerName)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(c.Body())
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return fmt.Errorf("HMAC mismatch: got %s want %s", sig, expected)
	}
	return nil
}

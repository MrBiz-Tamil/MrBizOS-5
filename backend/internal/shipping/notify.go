package shipping

// notify.go — Delivery lifecycle notifications.
//
// Provides two public surfaces:
//   publishDeliveryEvent  — low-level Redis pub/sub used by every file in this
//                           package (batcher, sla, autoassign) to push JSON
//                           events to WS rooms without an import cycle.
//   SendDeliveryStatusNotification — high-level helper that fans out a FCM
//                           push + patient WS event + courier WS event on each
//                           delivery status transition.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/services/notification"
)

// publishDeliveryEvent marshals payload, attaches room_id, and publishes to
// the Redis "chat:global" channel so the WS hub routes the message to every
// client subscribed to that room.  A nil Redis client is silently ignored so
// the caller never has to guard.
func publishDeliveryEvent(ctx context.Context, roomID string, payload map[string]interface{}) {
	if database.Rdb == nil {
		return
	}
	payload["room_id"] = roomID
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Shipping] publishDeliveryEvent marshal error: %v", err)
		return
	}
	if err := database.Rdb.Publish(ctx, "chat:global", data).Err(); err != nil {
		log.Printf("[Shipping] Redis publish error (room %s): %v", roomID, err)
	}
}

// statusMessages holds the patient-facing copy for each delivery transition.
var statusMessages = map[string]string{
	"assigned":  "Your order has been assigned to a courier and will be picked up shortly.",
	"picked_up": "Your order has been picked up by the courier and is on its way!",
	"en_route":  "Your courier is en route — arriving soon.",
	"arrived":   "Your courier has arrived at the delivery address.",
	"completed": "Your order has been delivered. Enjoy!",
	"failed":    "We were unable to deliver your order. A refund will be processed shortly.",
}

// statusTitles holds the FCM notification title per status.
var statusTitles = map[string]string{
	"assigned":  "Courier assigned",
	"picked_up": "Order picked up",
	"en_route":  "On the way",
	"arrived":   "Courier arrived",
	"completed": "Order delivered",
	"failed":    "Delivery failed",
}

// SendDeliveryStatusNotification fires:
//  1. A WS event to room:user:{patientUserID}    (patient-facing update)
//  2. A WS event to room:courier:{courierUserID} (courier confirmation/ACK)
//  3. An FCM push notification to the patient's device (if token exists)
//
// All errors are logged; the function is intentionally non-fatal.
func SendDeliveryStatusNotification(ctx context.Context, delivery *pharmacy.CourierDelivery, patientUserID uint) {
	status := delivery.Status
	msg, ok := statusMessages[status]
	if !ok {
		msg = fmt.Sprintf("Your order status has been updated to: %s", status)
	}
	title, ok := statusTitles[status]
	if !ok {
		title = "Order update"
	}

	// 1. Patient-facing WS event
	publishDeliveryEvent(ctx, fmt.Sprintf("room:user:%d", patientUserID), map[string]interface{}{
		"type":        "order_update",
		"status":      status,
		"delivery_id": delivery.ID,
		"order_id":    delivery.OrderID,
		"message":     msg,
	})

	// 2. Courier confirmation WS event (echo back so the courier app can update
	//    its local state without waiting for the next poll).
	publishDeliveryEvent(ctx, fmt.Sprintf("room:courier:%d", delivery.CourierID), map[string]interface{}{
		"type":        "delivery_status_ack",
		"delivery_id": delivery.ID,
		"order_id":    delivery.OrderID,
		"status":      status,
	})

	// 3. FCM push to patient (fire-and-forget goroutine to avoid blocking the
	//    caller; any failure is only logged).
	go sendPatientFCM(ctx, patientUserID, title, msg, delivery)
}

// sendPatientFCM looks up the patient's FCM token from identity.User and fires
// a push notification.  Non-fatal on any error.
func sendPatientFCM(ctx context.Context, patientUserID uint, title, body string, delivery *pharmacy.CourierDelivery) {
	if database.DB == nil {
		return
	}

	// Look up the FCM token stored on the User row
	var user identity.User
	if err := database.DB.WithContext(ctx).
		Select("fcm_token").
		First(&user, patientUserID).Error; err != nil {
		return // user not found — silently skip
	}
	if user.FCMToken == "" {
		return // no push token registered
	}

	err := notification.SendToToken(user.FCMToken, notification.FCMMessage{
		Title: title,
		Body:  body,
		Data: map[string]string{
			"delivery_id": fmt.Sprintf("%d", delivery.ID),
			"order_id":    fmt.Sprintf("%d", delivery.OrderID),
			"status":      delivery.Status,
		},
	})
	if err != nil {
		log.Printf("[Shipping] FCM push failed for user %d: %v", patientUserID, err)
	}
}

package events

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// StreamName is the single Redis Stream used for all domain events.
const StreamName = "Dr.Ayya:events"

// DeadLetterStream receives events that exhausted all retries.
const DeadLetterStream = "Dr.Ayya:events:dead"

// MaxRetries before an event is moved to the dead-letter stream.
const MaxRetries = 3

// EventType enumerates all domain event kinds.
type EventType string

const (
	EventDoctorOnline         EventType = "doctor.online"
	EventDoctorOffline        EventType = "doctor.offline"
	EventDoctorPing           EventType = "doctor.ping"
	EventAppointmentAccepted  EventType = "appointment.accepted"
	EventAppointmentStarted   EventType = "appointment.started"
	EventAppointmentCompleted EventType = "appointment.completed"
	EventAppointmentCancelled EventType = "appointment.cancelled"
	EventPayoutTriggered      EventType = "payout.triggered"
	EventOTPSent              EventType = "otp.sent"
	EventUserRegistered       EventType = "user.registered"
	EventDoctorProfileUpdated EventType = "doctor.profile.updated"
	EventTicketCreated        EventType = "ticket.created"
	EventTicketStatusUpdated  EventType = "ticket.status.updated"
	EventTicketAssigned       EventType = "ticket.assigned"
	// ERP Autonomous Events
	EventStockLow             EventType = "stock.low"
	EventInvoiceOverdue       EventType = "invoice.overdue"
	EventPOApprovalNeeded     EventType = "procurement.po.approval_needed"
	EventBankDiscrepancy      EventType = "banking.discrepancy"
	EventPayrollDue           EventType = "hrms.payroll.due"
)

// DomainEvent is the envelope for every event on the stream.
type DomainEvent struct {
	ID         string                 `json:"id"`
	Type       EventType              `json:"type"`
	OccurredAt time.Time              `json:"occurred_at"`
	ActorID    uint                   `json:"actor_id"` // user ID who triggered the event
	Payload    map[string]interface{} `json:"payload"`
	Retries    int                    `json:"retries"`
}

// Bus publishes and consumes domain events via Redis Streams.
type Bus struct {
	rdb *redis.Client
}

// New creates a new Bus.
func New(rdb *redis.Client) *Bus {
	return &Bus{rdb: rdb}
}

// Publish enqueues a domain event. Non-blocking; errors are logged but not returned.
func (b *Bus) Publish(ctx context.Context, eventType EventType, actorID uint, payload map[string]interface{}) {
	evt := DomainEvent{
		Type:       eventType,
		OccurredAt: time.Now().UTC(),
		ActorID:    actorID,
		Payload:    payload,
		Retries:    0,
	}
	data, err := json.Marshal(evt)
	if err != nil {
		log.Printf("[events] marshal error for %s: %v", eventType, err)
		return
	}
	if err := b.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamName,
		Values: map[string]interface{}{"event": string(data)},
	}).Err(); err != nil {
		log.Printf("[events] publish error for %s: %v", eventType, err)
	}
}

// HandlerFunc is called for each event. Return an error to trigger retry.
type HandlerFunc func(ctx context.Context, evt DomainEvent) error

// Subscribe starts a blocking consumer loop for the given group+consumer name.
// It reads events from the stream, calls the handler, and acks on success.
// On handler error it retries up to MaxRetries, then moves to dead-letter stream.
func (b *Bus) Subscribe(ctx context.Context, groupName, consumerName string, handler HandlerFunc) {
	// Create the consumer group if it doesn't exist
	b.rdb.XGroupCreateMkStream(ctx, StreamName, groupName, "0")

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msgs, err := b.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    groupName,
			Consumer: consumerName,
			Streams:  []string{StreamName, ">"},
			Count:    10,
			Block:    5 * time.Second,
		}).Result()

		if err != nil && err != redis.Nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[events] XReadGroup error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		for _, msg := range msgs {
			for _, xmsg := range msg.Messages {
				b.processMessage(ctx, groupName, xmsg, handler)
			}
		}
	}
}

func (b *Bus) processMessage(ctx context.Context, group string, msg redis.XMessage, handler HandlerFunc) {
	raw, ok := msg.Values["event"].(string)
	if !ok {
		b.rdb.XAck(ctx, StreamName, group, msg.ID)
		return
	}

	var evt DomainEvent
	if err := json.Unmarshal([]byte(raw), &evt); err != nil {
		log.Printf("[events] unmarshal error msg %s: %v", msg.ID, err)
		b.rdb.XAck(ctx, StreamName, group, msg.ID)
		return
	}
	evt.ID = msg.ID

	if err := handler(ctx, evt); err != nil {
		evt.Retries++
		if evt.Retries >= MaxRetries {
			log.Printf("[events] dead-letter event %s type=%s after %d retries: %v", msg.ID, evt.Type, evt.Retries, err)
			data, _ := json.Marshal(evt)
			b.rdb.XAdd(ctx, &redis.XAddArgs{
				Stream: DeadLetterStream,
				Values: map[string]interface{}{"event": string(data)},
			})
		} else {
			// Re-publish with incremented retry count
			data, _ := json.Marshal(evt)
			b.rdb.XAdd(ctx, &redis.XAddArgs{
				Stream: StreamName,
				Values: map[string]interface{}{"event": string(data)},
			})
		}
		// Ack the original so it doesn't stay as pending
		b.rdb.XAck(ctx, StreamName, group, msg.ID)
		return
	}

	b.rdb.XAck(ctx, StreamName, group, msg.ID)
}

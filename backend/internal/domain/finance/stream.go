package finance

import (
	"context"
	"encoding/json"
	"log"
	"sync"
)

// FinanceEvent is a slim, socket-friendly view of a ledger transaction, pushed
// to admin dashboards the instant a transaction commits. Heavy detail (full
// entry list, hashes) stays queryable via the REST endpoints — we keep frames
// small so a busy ledger doesn't flood clients.
type FinanceEvent struct {
	Ref         string `json:"reference_id"`
	Type        string `json:"type"`        // CONSULTATION, PAYOUT_SPLIT, DOCTOR_PAYOUT, ...
	Description string `json:"description"`
	Amount      int64  `json:"amount"`      // total debits == total credits, in paise
	CreatedAt   string `json:"created_at"`  // RFC3339
}

// financeHub holds the set of subscribed WebSocket clients. One global instance
// per process; goroutine-safe. Mirrors the proven audit stream hub.
type financeHub struct {
	mu   sync.RWMutex
	subs map[chan FinanceEvent]struct{}
}

var fhub = &financeHub{subs: map[chan FinanceEvent]struct{}{}}

// Subscribe returns a channel that receives newly-committed finance events.
// Caller MUST call Unsubscribe when done (typically on WebSocket close).
//
// Buffer of 32: a slow client doesn't block the recorder. If the buffer fills,
// that client's event is dropped silently — they see a gap in the live view but
// the ledger row itself is safely persisted.
func Subscribe() chan FinanceEvent {
	ch := make(chan FinanceEvent, 32)
	fhub.mu.Lock()
	fhub.subs[ch] = struct{}{}
	fhub.mu.Unlock()
	return ch
}

// Unsubscribe removes the channel from the hub and closes it. Idempotent.
func Unsubscribe(ch chan FinanceEvent) {
	fhub.mu.Lock()
	defer fhub.mu.Unlock()
	if _, ok := fhub.subs[ch]; ok {
		delete(fhub.subs, ch)
		close(ch)
	}
}

// PublishFinanceEvent fan-outs an event to every subscriber. Called by the
// service layer AFTER a transaction has committed, so subscribers only ever see
// transactions that actually landed in the ledger. Drops events for slow
// clients rather than blocking the caller.
func PublishFinanceEvent(ev FinanceEvent) {
	fhub.mu.RLock()
	defer fhub.mu.RUnlock()
	for ch := range fhub.subs {
		select {
		case ch <- ev:
		default:
			// Drop — subscriber too slow. Their live view will be incomplete;
			// the ledger row is still safe in the DB.
		}
	}
}

// SubscriberCount returns how many WebSocket clients are currently listening.
// Handy for observability ("why is my live ledger silent?" → zero subscribers).
func SubscriberCount() int {
	fhub.mu.RLock()
	defer fhub.mu.RUnlock()
	return len(fhub.subs)
}

// LiveStreamSession reads from a fresh subscription until ctx is done, writing
// JSON frames via `write`. Handles its own cleanup. Used by the WebSocket layer.
func LiveStreamSession(ctx context.Context, write func([]byte) error) {
	ch := Subscribe()
	defer Unsubscribe(ch)

	// Initial hello so the frontend knows the connection is alive.
	hello, _ := json.Marshal(map[string]interface{}{
		"type": "ready",
		"msg":  "finance stream connected",
	})
	if err := write(hello); err != nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			payload, err := json.Marshal(map[string]interface{}{
				"type":  "finance_event",
				"event": ev,
			})
			if err != nil {
				continue
			}
			if err := write(payload); err != nil {
				log.Printf("[finance:stream] write failed: %v", err)
				return
			}
		}
	}
}

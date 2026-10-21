package audit

import (
	"context"
	"encoding/json"
	"log"
	"sync"
)

// broadcastEvent is a slim view of an AuditLog suited for live UI streaming.
// We deliberately omit `before`/`after` JSON to keep socket frames small —
// admins click through to /admin/audit-logs/:id for the heavy detail.
type broadcastEvent struct {
	ID         uint64 `json:"id"`
	ActorID    uint   `json:"actor_id"`
	ActorRole  string `json:"actor_role"`
	Action     string `json:"action"`
	TableName  string `json:"table_name"`
	RowID      string `json:"row_id"`
	IPAddress  string `json:"ip_address"`
	Path       string `json:"path"`
	CreatedAt  string `json:"created_at"`
}

// liveHub holds the set of subscribed WebSocket clients. One global instance
// in the package; goroutine-safe.
type liveHub struct {
	mu  sync.RWMutex
	subs map[chan broadcastEvent]struct{}
}

var hub = &liveHub{subs: map[chan broadcastEvent]struct{}{}}

// Subscribe returns a channel that receives newly-written AuditLog events.
// Caller MUST call Unsubscribe when done — typically on WebSocket close.
//
// Buffer of 32: a slow client doesn't block the worker. If the buffer fills,
// the broadcast for that client is dropped silently — they'll see a gap in
// the live view but the audit row itself is still persisted in audit_logs.
func Subscribe() chan broadcastEvent {
	ch := make(chan broadcastEvent, 32)
	hub.mu.Lock()
	hub.subs[ch] = struct{}{}
	hub.mu.Unlock()
	return ch
}

// Unsubscribe removes the channel from the hub and closes it. Idempotent.
func Unsubscribe(ch chan broadcastEvent) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	if _, ok := hub.subs[ch]; ok {
		delete(hub.subs, ch)
		close(ch)
	}
}

// publish fan-outs an event to every subscriber. Called by the audit worker
// right after a row is persisted. Drops events for slow clients rather than
// blocking.
func (h *liveHub) publish(ev broadcastEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs {
		select {
		case ch <- ev:
		default:
			// Drop — subscriber is too slow. Live view will be incomplete
			// for them; the audit_log row is still safe in the DB.
		}
	}
}

// publishMany is invoked from the worker AFTER CreateInBatches succeeds, so
// only events that actually landed in audit_logs are streamed. This keeps
// what subscribers see consistent with what's queryable.
func (h *liveHub) publishMany(rows []AuditLog) {
	if len(rows) == 0 {
		return
	}
	for i := range rows {
		h.publish(broadcastEvent{
			ID:        rows[i].ID,
			ActorID:   rows[i].ActorID,
			ActorRole: rows[i].ActorRole,
			Action:    rows[i].Action,
			TableName: rows[i].Table,
			RowID:     rows[i].RowID,
			IPAddress: rows[i].IPAddress,
			Path:      rows[i].Path,
			CreatedAt: rows[i].CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
}

// SubscriberCount returns how many WebSocket clients are currently listening.
// Exposed for /health-style observability — handy when investigating "why
// is my live tab silent" (answer: zero subscribers means nothing is connected
// vs. broken broadcast vs. genuinely quiet system).
func SubscriberCount() int {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	return len(hub.subs)
}

// LiveStreamSession is a small helper used by handlers/websocket layer.
// It reads from the subscriber channel until ctx is done and writes JSON
// frames via `write`. Handles cleanup on exit.
func LiveStreamSession(ctx context.Context, write func([]byte) error) {
	ch := Subscribe()
	defer Unsubscribe(ch)

	// Initial hello so the frontend knows the connection is alive
	hello, _ := json.Marshal(map[string]interface{}{
		"type": "ready",
		"msg":  "audit stream connected",
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
				"type":  "audit_event",
				"event": ev,
			})
			if err != nil {
				continue
			}
			if err := write(payload); err != nil {
				log.Printf("[audit:stream] write failed: %v", err)
				return
			}
		}
	}
}

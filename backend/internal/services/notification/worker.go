package notification

// worker.go — background goroutine that drains the notification_jobs outbox.
//
// StartWorker launches a single goroutine that ticks every pollInterval and
// claims + sends all currently-due jobs.  The goroutine exits cleanly when ctx
// is cancelled (server shutdown).  It is safe to call multiple times — each
// call starts an independent goroutine, so keep to one per process.

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// DefaultPollInterval is how often the worker checks for pending jobs.
const DefaultPollInterval = 10 * time.Second

var (
	workerCancel context.CancelFunc
)

// StopWorker cancels the active background worker if one is running.
func StopWorker() {
	if workerCancel != nil {
		workerCancel()
		workerCancel = nil
	}
}

// StartWorker starts the outbox processor.  senders is the map produced by
// Service.Senders().  Pass context.Background() if no lifecycle context is
// available; the goroutine will run for the lifetime of the process.
func StartWorker(ctx context.Context, db *gorm.DB, senders map[Channel]Sender, pollInterval time.Duration) {
	if pollInterval <= 0 {
		pollInterval = DefaultPollInterval
	}

	StopWorker()

	var cancelCtx context.Context
	cancelCtx, workerCancel = context.WithCancel(ctx)

	go func() {
		log.Printf("[notif] worker started — poll interval %s", pollInterval)
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-cancelCtx.Done():
				log.Println("[notif] worker stopped")
				return
			case <-ticker.C:
				processAll(cancelCtx, db, senders)
			}
		}
	}()
}

// processAll drains all currently-due jobs in a tight loop so a burst of
// notifications does not wait for the next tick.
func processAll(ctx context.Context, db *gorm.DB, senders map[Channel]Sender) {
	for {
		// Stop iterating if the context was cancelled mid-burst.
		if ctx.Err() != nil {
			return
		}

		job, err := claimNextJob(ctx, db)
		if err != nil {
			// FIX 13: ErrRecordNotFound simply means the queue is empty — not an error condition.
			// Only log at error level for genuine failures.
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return // queue drained — not an error
			}
			log.Printf("[notif] claimNextJob error: %v", err)
			return
		}

		sender, ok := senders[job.Channel]
		if !ok {
			log.Printf("[notif] no sender for channel %q — killing %s", job.Channel, jobSummary(job))
			markFailed(ctx, db, job, fmt.Errorf("no sender registered for channel %s", job.Channel))
			continue
		}

		log.Printf("[notif] sending %s", jobSummary(job))
		if err := sender.Send(job); err != nil {
			log.Printf("[notif] FAILED %s: %v", jobSummary(job), err)
			markFailed(ctx, db, job, err)
		} else {
			log.Printf("[notif] sent   %s", jobSummary(job))
			markSent(ctx, db, job.ID)
		}
	}
}

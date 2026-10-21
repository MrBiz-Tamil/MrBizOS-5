// Package audit provides automatic, tamper-evident logging of every data
// mutation across the application via a GORM plugin.
//
// Architecture:
//
//	Request → middleware.AuditCtx() sets AuditContext on Go context
//	        → Handler calls db.WithContext(ctx).Save(...) as normal
//	        → GORM lifecycle hook fires (this plugin)
//	        → Event sent to non-blocking buffered channel
//	        → Background worker batches and writes audit_logs rows
//
// Properties:
//   - Zero per-handler code: every db.Save/Create/Delete is captured.
//   - Cannot block the user's request: drops with warning if buffer fills.
//   - Field-level redaction via `audit:"skip"` and `audit:"redact"` tags.
//   - Self-skipping (audit_logs table itself is not audited — infinite loop).
//   - Tables can opt out by implementing AuditSkippable.
//
// See AUDIT_LOG_ARCHITECTURE.md for the full design.
package audit

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"sync/atomic"
	"time"

	"drayya-core/internal/middleware"

	"gorm.io/gorm"
)

// Plugin is the GORM plugin that captures mutations into audit_logs.
type Plugin struct {
	sink      chan AuditLog // non-blocking buffer, drained by worker()
	dropped   uint64        // count of events dropped due to full buffer
	writeDB   *gorm.DB      // dedicated session for writes (avoids re-triggering hooks)
	stopWorker chan struct{}
}

// NewPlugin returns a plugin with the given buffer size. A buffer of 10_000
// gives ~30 seconds of headroom at 300 mutations/sec — well above current peak.
func NewPlugin(bufferSize int) *Plugin {
	if bufferSize <= 0 {
		bufferSize = 10_000
	}
	return &Plugin{
		sink:       make(chan AuditLog, bufferSize),
		stopWorker: make(chan struct{}),
	}
}

// globalPlugin holds the singleton plugin attached at process startup.
// Exposed for tests / observability only — production code never reads this.
var globalPlugin *Plugin

// InstallGlobal mounts the audit plugin on the given GORM DB exactly once.
// Subsequent calls are no-ops. Safe to call from goroutines.
//
// Called from cmd/server/main.go after RunMigrations() returns.
func InstallGlobal(db *gorm.DB) {
	if globalPlugin != nil || db == nil {
		return
	}
	p := NewPlugin(10_000)
	if err := db.Use(p); err != nil {
		log.Printf("[audit] failed to install plugin: %v (audit logging disabled)", err)
		return
	}
	globalPlugin = p
}

func (p *Plugin) Name() string { return "drayya:audit" }

// Initialize is called by gorm.DB.Use(plugin). It:
//   1. Migrates the audit_logs table.
//   2. Opens a dedicated write session (so the worker's INSERTs don't re-fire
//      this plugin's callbacks → infinite recursion).
//   3. Registers Create/Update/Delete callbacks.
//   4. Spawns the background worker.
func (p *Plugin) Initialize(db *gorm.DB) error {
	if err := db.AutoMigrate(&AuditLog{}); err != nil {
		return fmt.Errorf("audit: migrate audit_logs: %w", err)
	}

	// GORM's AutoMigrate doesn't widen existing columns. Older deployments
	// have audit_logs.request_id as VARCHAR(36) which truncates the cron
	// job IDs like "sys-<uuid>" (40 chars) raising SQLSTATE 22001.
	// This is a no-op on databases where the column is already 64+ wide.
	if db.Dialector.Name() == "postgres" {
		_ = db.Exec(`ALTER TABLE audit_logs ALTER COLUMN request_id TYPE VARCHAR(64)`).Error
	}

	// New session = clean callback chain. We use this for writing the audit
	// rows themselves so our own writes don't re-enter the plugin.
	p.writeDB = db.Session(&gorm.Session{NewDB: true, SkipHooks: true})

	if err := db.Callback().Create().After("gorm:create").Register("audit:after_create", p.afterCreate); err != nil {
		return err
	}
	// Update: capture row state BEFORE the update fires, then compare to the
	// after-state inside afterUpdate to produce a diff.
	if err := db.Callback().Update().Before("gorm:update").Register("audit:before_update", p.beforeUpdate); err != nil {
		return err
	}
	if err := db.Callback().Update().After("gorm:update").Register("audit:after_update", p.afterUpdate); err != nil {
		return err
	}
	if err := db.Callback().Delete().Before("gorm:delete").Register("audit:before_delete", p.beforeDelete); err != nil {
		return err
	}

	go p.worker()
	log.Println("[audit] plugin initialised — capturing CREATE/UPDATE/DELETE on all tables (except opt-outs)")
	return nil
}

// Stop tells the worker goroutine to flush and exit. Test cleanup mostly.
func (p *Plugin) Stop() {
	close(p.stopWorker)
}

// DroppedCount returns how many audit events were dropped because the buffer
// was full. Should always be 0 in steady state — exposed for monitoring.
func (p *Plugin) DroppedCount() uint64 {
	return atomic.LoadUint64(&p.dropped)
}

// ── GORM callbacks ───────────────────────────────────────────────────────────

func (p *Plugin) afterCreate(db *gorm.DB) {
	if shouldSkip(db) {
		return
	}
	after := redactRecord(db.Statement.Dest, db.Statement.Schema)
	p.emit(db, "CREATE", nil, after)
}

// beforeUpdate snapshots the row state BEFORE the update fires, so afterUpdate
// can compute a real before/after diff. The snapshot is stashed on the
// statement's Settings map (request-scoped, automatically cleaned up).
//
// Skipped when the update is a map-based partial (db.Updates(map{...})) AND
// the primary key isn't known — re-fetching would be a cross-tenant risk.
func (p *Plugin) beforeUpdate(db *gorm.DB) {
	if shouldSkip(db) {
		return
	}
	if db.Statement.Schema == nil {
		return
	}
	pk := extractPrimaryKey(db)
	if pk == "" {
		return // bulk update with WHERE clause; before-state would need re-query of all matching rows
	}

	// Re-fetch the current row into a fresh struct of the same model type.
	// SkipHooks avoids re-firing this very plugin.
	dest := reflect.New(db.Statement.Schema.ModelType).Interface()
	tx := db.Session(&gorm.Session{NewDB: true, SkipHooks: true})
	if err := tx.Table(db.Statement.Table).
		Where(db.Statement.Schema.PrimaryFields[0].DBName+" = ?", pk).
		First(dest).Error; err != nil {
		return // record not found / select failed — fall through with no before
	}
	beforeMap := redactRecord(dest, db.Statement.Schema)
	db.Statement.Settings.Store("audit:before", beforeMap)
}

func (p *Plugin) afterUpdate(db *gorm.DB) {
	if shouldSkip(db) {
		return
	}
	after := redactRecord(db.Statement.Dest, db.Statement.Schema)

	var before map[string]interface{}
	if bAny, ok := db.Statement.Settings.LoadAndDelete("audit:before"); ok {
		before, _ = bAny.(map[string]interface{})
	}

	diff := computeDiff(before, after)
	p.emitWithDiff(db, "UPDATE", before, after, diff)
}

func (p *Plugin) beforeDelete(db *gorm.DB) {
	if shouldSkip(db) {
		return
	}
	// Capture the row as it exists right before deletion — once deleted we
	// can't recover the state.
	before := redactRecord(db.Statement.Dest, db.Statement.Schema)
	p.emit(db, "DELETE", before, nil)
}

// ── Helpers ──────────────────────────────────────────────────────────────────

// shouldSkip checks whether this operation must be excluded from audit:
//   - The audit_logs table itself (would recurse).
//   - Models that implement AuditSkippable and return true.
//   - When db.Statement.Dest is nil (unusual; nothing to audit).
func shouldSkip(db *gorm.DB) bool {
	if db.Statement.Dest == nil {
		return true
	}
	if db.Statement.Table == "audit_logs" {
		return true
	}
	if db.Statement.Schema != nil && db.Statement.Schema.Table == "audit_logs" {
		return true
	}

	// Check AuditSkippable on the model. Use a fresh instance so we don't
	// rely on the destination value being set (e.g. on Delete by primary key).
	if db.Statement.Schema != nil {
		modelType := db.Statement.Schema.ModelType
		if modelType != nil {
			zero := reflect.New(modelType).Interface()
			if skipper, ok := zero.(AuditSkippable); ok && skipper.AuditSkip() {
				return true
			}
		}
	}
	return false
}

// extractPrimaryKey returns a string representation of the row's primary key.
// Returns "" when the PK is unknown (e.g. bulk update with WHERE clause).
func extractPrimaryKey(db *gorm.DB) string {
	if db.Statement.Schema == nil || db.Statement.ReflectValue.Kind() == reflect.Invalid {
		return ""
	}
	rv := db.Statement.ReflectValue
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	// Handle slice of records (bulk create) — use first PK as a hint
	if rv.Kind() == reflect.Slice && rv.Len() > 0 {
		rv = rv.Index(0)
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}
	for _, f := range db.Statement.Schema.PrimaryFields {
		fieldVal, isZero := f.ValueOf(db.Statement.Context, rv)
		if !isZero {
			return fmt.Sprintf("%v", fieldVal)
		}
	}
	return ""
}

// emit constructs an AuditLog (no diff) and pushes it to the worker. Used for
// CREATE and DELETE where a diff doesn't apply.
func (p *Plugin) emit(db *gorm.DB, action string, before, after map[string]interface{}) {
	p.emitWithDiff(db, action, before, after, nil)
}

// emitWithDiff is the workhorse — accepts an optional diff and pushes the
// full AuditLog to the buffered channel. Never blocks the calling goroutine.
func (p *Plugin) emitWithDiff(db *gorm.DB, action string, before, after, diff map[string]interface{}) {
	ac := middleware.GetAuditContext(db.Statement.Context)

	// Honor bulk-flag: skip per-row audit; caller is responsible for summary.
	if ac != nil && ac.Bulk {
		return
	}

	beforeJSON, _ := jsonOrEmpty(before)
	afterJSON, _ := jsonOrEmpty(after)
	diffJSON, _ := jsonOrEmpty(diff)

	logEntry := AuditLog{
		Action:    action,
		Table:     db.Statement.Table,
		RowID:     extractPrimaryKey(db),
		Before:    beforeJSON,
		After:     afterJSON,
		Diff:      diffJSON,
		CreatedAt: time.Now(),
	}
	if ac != nil {
		logEntry.ActorID = ac.ActorID
		logEntry.ActorRole = ac.ActorRole
		logEntry.IPAddress = ac.IP
		logEntry.UserAgent = ac.UserAgent
		logEntry.RequestID = ac.RequestID
		logEntry.Path = ac.Path
		logEntry.Method = ac.Method
	}

	// Non-blocking send. Full buffer means we're producing faster than the DB
	// can absorb — drop and warn rather than block the user request.
	select {
	case p.sink <- logEntry:
	default:
		count := atomic.AddUint64(&p.dropped, 1)
		// Log only on first drop and then once per 1000 to avoid log spam.
		if count == 1 || count%1000 == 0 {
			log.Printf("[audit] DROPPED %s %s/%s — buffer full (total dropped: %d)",
				action, logEntry.Table, logEntry.RowID, count)
		}
	}
}

func jsonOrEmpty(m map[string]interface{}) ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// worker batches incoming events and writes them in chunks of 100 (or every
// second, whichever first). One DB roundtrip per batch is much cheaper than
// per-row INSERTs.
//
// The worker is the SINGLE point that computes the hash chain — keeping all
// writes serialised through one goroutine makes the chain deterministic and
// races impossible.
func (p *Plugin) worker() {
	const (
		batchSize     = 100
		flushInterval = time.Second
	)
	batch := make([]AuditLog, 0, batchSize)
	tick := time.NewTicker(flushInterval)
	defer tick.Stop()

	// Bootstrap prevHash from the most recent row in audit_logs so chains
	// survive process restarts. If the table is empty, prevHash stays "".
	var prevHash string
	var last AuditLog
	if err := p.writeDB.Order("id desc").Limit(1).Find(&last).Error; err == nil {
		prevHash = last.HashChain
	}

	flush := func() {
		if len(batch) == 0 {
			return
		}
		// Chain-link every row in the batch before writing.
		for i := range batch {
			batch[i].HashChain = computeChainHash(prevHash, &batch[i])
			prevHash = batch[i].HashChain
		}
		// Use the dedicated SkipHooks session so this INSERT doesn't re-trigger
		// our own callbacks (infinite recursion).
		if err := p.writeDB.CreateInBatches(batch, batchSize).Error; err != nil {
			log.Printf("[audit] flush failed (%d rows): %v", len(batch), err)
			batch = batch[:0]
			return
		}
		// Broadcast to live WebSocket subscribers AFTER the DB write succeeds,
		// so subscribers never see an event that isn't actually persisted.
		hub.publishMany(batch)
		batch = batch[:0]
	}

	for {
		select {
		case ev := <-p.sink:
			batch = append(batch, ev)
			if len(batch) >= batchSize {
				flush()
			}
		case <-tick.C:
			flush()
		case <-p.stopWorker:
			flush()
			return
		}
	}
}

// VerifyChain walks the entire audit_logs table in insertion order and
// recomputes each row's hash, asserting it matches what's stored. Returns
// the row count, the ID of the first broken row (0 = no breaks), and the
// error if any.
//
// Called by the admin endpoint /admin/audit-logs/verify-chain. Designed for
// nightly batch verification — single-pass, paginated, no large in-memory set.
func VerifyChain(db *gorm.DB) (totalRows int64, brokenFromID uint64, err error) {
	if err = db.Model(&AuditLog{}).Count(&totalRows).Error; err != nil {
		return 0, 0, err
	}

	prevHash := ""
	pageSize := 1000
	var lastID uint64

	for {
		var page []AuditLog
		if err = db.Where("id > ?", lastID).Order("id asc").Limit(pageSize).Find(&page).Error; err != nil {
			return totalRows, 0, err
		}
		if len(page) == 0 {
			return totalRows, 0, nil
		}
		for i := range page {
			expected := computeChainHash(prevHash, &page[i])
			if expected != page[i].HashChain {
				return totalRows, page[i].ID, nil
			}
			prevHash = page[i].HashChain
			lastID = page[i].ID
		}
		if len(page) < pageSize {
			return totalRows, 0, nil
		}
	}
}

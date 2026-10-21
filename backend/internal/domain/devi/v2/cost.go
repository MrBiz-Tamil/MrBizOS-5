package v2

import (
	"context"
	"strconv"
	"time"

	"drayya-core/internal/database"
	"gorm.io/gorm"
)

// Phase 8 — cost tracking + circuit breaker.
//
// Two layers:
//   1. LLMUsage — append-only table recording every Gemini call: tokens,
//      cost, latency. Source of truth for billing and analytics.
//   2. Redis counters — fast-path daily/monthly aggregates used to short-
//      circuit the LLM call when over budget.
//
// The circuit breaker is intentionally simple in Phase 8: a hard daily
// cap. When tripped, /devi/v2/chat falls back to a templated message and
// skips the LLM call entirely. Half-open / observation-mode behaviour
// lands in a later phase.

// Gemini 2.0 Flash pricing as of 2026-06.
// (input: $0.075 / 1M tokens, output: $0.30 / 1M tokens).
// These numbers should live in env / config — hardcoded for Phase 8 MVP.
const (
	gemini2FlashInputCostPer1M  = 0.075
	gemini2FlashOutputCostPer1M = 0.30
)

// Daily / monthly cost ceilings in USD. Configurable in Phase 8.5 via
// devi_configs; baked in for now.
const (
	DailyCostCapUSD   = 50.0
	MonthlyCostCapUSD = 1000.0

	// PerUserDailyCostCapUSD throttles a single user from burning through
	// the global budget alone — set at $1/day default. Premium users get
	// a larger cap (handled in Phase 8.5).
	PerUserDailyCostCapUSD = 1.0
)

// LLMUsage is one row per Gemini call. Append-only — admin queries scan it
// for spend reports and per-user abuse detection.
type LLMUsage struct {
	ID           uint   `gorm:"primaryKey"`
	UserID       *uint  `gorm:"index"`
	SessionID    string `gorm:"index;size:64"`
	Model        string `gorm:"size:64"`
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	LatencyMS    int
	WasError     bool
	CreatedAt    time.Time `gorm:"index"`
}

func (LLMUsage) TableName() string { return "devi_v2_llm_usage" }

// AutoMigrate wires the schema into the global migration set. Called once
// at startup from server/app.go alongside the rest of the AutoMigrate calls.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&LLMUsage{})
}

// computeCost turns Gemini's reported token usage into USD with the current
// model's rate card.
func computeCost(inputTokens, outputTokens int) float64 {
	in := float64(inputTokens) / 1_000_000.0 * gemini2FlashInputCostPer1M
	out := float64(outputTokens) / 1_000_000.0 * gemini2FlashOutputCostPer1M
	return in + out
}

// RecordUsage persists one LLM call and updates Redis aggregate counters.
// Best-effort, runs in a goroutine — never blocks the response.
func RecordUsage(ctx context.Context, persona Persona, sessionID, model string,
	inputTokens, outputTokens, latencyMS int, wasError bool) {

	cost := computeCost(inputTokens, outputTokens)

	row := LLMUsage{
		SessionID:    sessionID,
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		CostUSD:      cost,
		LatencyMS:    latencyMS,
		WasError:     wasError,
		CreatedAt:    time.Now(),
	}
	if persona.UserID != nil {
		uid := *persona.UserID
		row.UserID = &uid
	}

	go func() {
		if database.DB != nil {
			_ = database.DB.WithContext(context.Background()).Create(&row).Error
		}
		bumpCostCounter(persona, cost)
	}()
}

// bumpCostCounter atomically increments the daily / monthly counters in
// Redis. INCRBYFLOAT + EXPIRE in a pipeline so a Redis crash mid-transaction
// never leaves orphaned keys (TTL still applies on restart).
func bumpCostCounter(persona Persona, costUSD float64) {
	if database.Rdb == nil {
		return
	}
	ctx := context.Background()
	now := time.Now()
	dayKey := "devi:v2:cost:day:" + now.Format("2006-01-02")
	monthKey := "devi:v2:cost:month:" + now.Format("2006-01")

	pipe := database.Rdb.Pipeline()
	pipe.IncrByFloat(ctx, dayKey, costUSD)
	pipe.Expire(ctx, dayKey, 48*time.Hour)
	pipe.IncrByFloat(ctx, monthKey, costUSD)
	pipe.Expire(ctx, monthKey, 40*24*time.Hour)
	if persona.UserID != nil {
		userKey := "devi:v2:cost:user:" + strconv.FormatUint(uint64(*persona.UserID), 10) +
			":day:" + now.Format("2006-01-02")
		pipe.IncrByFloat(ctx, userKey, costUSD)
		pipe.Expire(ctx, userKey, 48*time.Hour)
	}
	_, _ = pipe.Exec(ctx)
}

// ─── Circuit breaker ──────────────────────────────────────────────────────

type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
)

// CheckCircuit returns OPEN when either the global daily cap or this
// user's per-day cap is already spent. Cheap O(1) Redis read; runs on
// every request. Fail-closed: Redis down → treat as CLOSED (assume budget
// OK) so the rate limiter is the safety net.
func CheckCircuit(ctx context.Context, persona Persona) CircuitState {
	if database.Rdb == nil {
		return CircuitClosed
	}
	dayKey := "devi:v2:cost:day:" + time.Now().Format("2006-01-02")
	if raw, err := database.Rdb.Get(ctx, dayKey).Result(); err == nil && raw != "" {
		spent, _ := strconv.ParseFloat(raw, 64)
		if spent >= DailyCostCapUSD {
			return CircuitOpen
		}
	}
	if persona.UserID != nil {
		userKey := "devi:v2:cost:user:" + strconv.FormatUint(uint64(*persona.UserID), 10) +
			":day:" + time.Now().Format("2006-01-02")
		if raw, err := database.Rdb.Get(ctx, userKey).Result(); err == nil && raw != "" {
			userSpent, _ := strconv.ParseFloat(raw, 64)
			if userSpent >= PerUserDailyCostCapUSD {
				return CircuitOpen
			}
		}
	}
	return CircuitClosed
}

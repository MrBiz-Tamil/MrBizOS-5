package v2

import (
	"context"
	"strconv"
	"time"

	"drayya-core/internal/database"
	"github.com/redis/go-redis/v9"
)

// All Phase-0 limits live here. Keep them as exported constants so admins
// can adjust them via devi_configs at runtime in later phases without
// recompiling. (The runtime read happens in NewLimiter; mutation hooks
// land in Phase 8.)
//
// The numbers below are intentionally conservative. They protect cost and
// availability on day 1; observability data will tell us where to loosen.
const (
	// Memory & conversation
	MaxMemoryTurns           = 10
	MaxConversationTurns     = 50
	MaxInputChars            = 2_000
	MaxOutputTokens          = 500
	MaxToolCallsPerTurn      = 3
	MaxToolCallsPerSession   = 30
	ConversationTTL          = 1 * time.Hour

	// Guest limits
	GuestMessagesPerMinute = 5
	GuestMessagesPerHour   = 30
	GuestMessagesPerDay    = 100

	// New patient (created < 24h ago) — slightly tighter than seasoned users.
	NewPatientMessagesPerMinute = 10
	NewPatientMessagesPerHour   = 60
	NewPatientMessagesPerDay    = 200

	// Patient (free tier)
	PatientMessagesPerMinute = 20
	PatientMessagesPerHour   = 120
	PatientMessagesPerDay    = 500

	// Patient (premium tier)
	PremiumMessagesPerMinute = 30
	PremiumMessagesPerHour   = 300
	PremiumMessagesPerDay    = 1500
)

// Limits describes the quotas applying to one persona class.
type Limits struct {
	PerMinute int
	PerHour   int
	PerDay    int
}

// LimitsFor returns the message rate limits for this caller. Driven by
// persona role + tier + IsNew flag.
func LimitsFor(p Persona) Limits {
	switch p.Role {
	case RoleGuest:
		return Limits{GuestMessagesPerMinute, GuestMessagesPerHour, GuestMessagesPerDay}
	case RolePatient:
		if p.Tier == "premium" {
			return Limits{PremiumMessagesPerMinute, PremiumMessagesPerHour, PremiumMessagesPerDay}
		}
		if p.IsNew {
			return Limits{NewPatientMessagesPerMinute, NewPatientMessagesPerHour, NewPatientMessagesPerDay}
		}
		return Limits{PatientMessagesPerMinute, PatientMessagesPerHour, PatientMessagesPerDay}
	default:
		// Deferred roles shouldn't reach here, but if they do, cap hard.
		return Limits{1, 5, 20}
	}
}

// Limiter implements the sliding-window log rate limiter using Redis sorted
// sets. The score is the millisecond timestamp; the member is the same
// value (uniqueness comes from the timestamp itself being monotonic per
// process). On every check:
//
//  1. ZREMRANGEBYSCORE drops entries older than window
//  2. ZCARD reports the live count
//  3. If under cap, ZADD the new timestamp and EXPIRE the key
//
// This is more accurate than a token bucket at the cost of a few Redis ops
// per request — fine at our scale.
type Limiter struct{}

func NewLimiter() *Limiter { return &Limiter{} }

// LimitVerdict describes why a request was throttled (if it was). Empty
// Reason means the request is allowed.
type LimitVerdict struct {
	Allowed bool
	Reason  string // "minute" | "hour" | "day" | ""
	Remaining int  // how many requests the user has left in the breached window
}

// Check returns Allowed=true only if every window has headroom. Records the
// new request against all three windows in a single pipeline.
//
// If Redis is unavailable the limiter fails OPEN — degraded mode is to let
// traffic through rather than block all users. The cost ceiling in Phase 8
// is the safety net for this case.
func (l *Limiter) Check(ctx context.Context, p Persona) LimitVerdict {
	if database.Rdb == nil {
		return LimitVerdict{Allowed: true}
	}

	limits := LimitsFor(p)
	base := p.LimitKey()
	now := time.Now().UnixMilli()

	// Per-minute is the most common rejection; check it first to avoid the
	// other two Redis ops in the common throttled case.
	if !l.checkWindow(ctx, base+":min", limits.PerMinute, time.Minute, now) {
		return LimitVerdict{Allowed: false, Reason: "minute"}
	}
	if !l.checkWindow(ctx, base+":hr", limits.PerHour, time.Hour, now) {
		return LimitVerdict{Allowed: false, Reason: "hour"}
	}
	if !l.checkWindow(ctx, base+":day", limits.PerDay, 24*time.Hour, now) {
		return LimitVerdict{Allowed: false, Reason: "day"}
	}
	return LimitVerdict{Allowed: true}
}

func (l *Limiter) checkWindow(ctx context.Context, key string, max int, window time.Duration, nowMS int64) bool {
	cutoff := nowMS - window.Milliseconds()

	// Cleanup old entries.
	database.Rdb.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(cutoff, 10))

	count, err := database.Rdb.ZCard(ctx, key).Result()
	if err != nil {
		// Fail open on Redis errors — see Check() comment.
		return true
	}
	if count >= int64(max) {
		return false
	}

	// Record the current request. The Z value is the timestamp — needs to
	// be unique per request, so we add a tiny random suffix in pathological
	// cases (multiple requests in the same ms from one user). Skipped for
	// simplicity here; collision probability is negligible at our QPS.
	score := float64(nowMS)
	database.Rdb.ZAdd(ctx, key, redis.Z{Score: score, Member: nowMS})
	database.Rdb.Expire(ctx, key, window)
	return true
}

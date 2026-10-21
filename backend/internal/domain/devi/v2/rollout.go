package v2

import (
	"context"
	"hash/fnv"
	"os"
	"strconv"

	"drayya-core/internal/database"
)

// RolloutKey is the Redis key storing an admin-set override of the env-
// var-default rollout percentage. When this key exists and is valid, it
// takes precedence over DEVI_V2_ROLLOUT_PCT — so admins can dial the
// rollout up/down at runtime without redeploying.
const RolloutKey = "devi:v2:rollout_pct"

// Phase 5 — A/B rollout.
//
// V2 lives alongside V1 during the rollout. Traffic-split is controlled by
// a single env var, DEVI_V2_ROLLOUT_PCT, which is the percentage of users
// routed to V2 (0–100). Bucketing is deterministic per user / browser_id
// so the same user gets a consistent experience across sessions.
//
// Usage from V1's chat handler:
//
//	if deviv2.ShouldRouteToV2(persona) {
//	    // forward the request to V2 agent
//	}
//
// Default is 0 — no traffic to V2 — until an admin sets the env var.

// rolloutPct reads the current rollout percentage. Resolution order:
//
//   1. Redis key RolloutKey (admin-set override at runtime)
//   2. DEVI_V2_ROLLOUT_PCT env var (deploy-time default)
//   3. 0 (safe default — V2 off)
//
// All values are clamped to [0, 100].
func rolloutPct() int {
	// Runtime override.
	if database.Rdb != nil {
		if raw, err := database.Rdb.Get(context.Background(), RolloutKey).Result(); err == nil && raw != "" {
			if n, err := strconv.Atoi(raw); err == nil {
				return clampPct(n)
			}
		}
	}
	// Env-var default.
	raw := os.Getenv("DEVI_V2_ROLLOUT_PCT")
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return clampPct(n)
}

func clampPct(n int) int {
	if n < 0 {
		return 0
	}
	if n > 100 {
		return 100
	}
	return n
}

// SetRolloutPct persists an admin-chosen rollout percentage to Redis. The
// new value takes effect on the next request — no restart needed. Returns
// the clamped value actually stored.
func SetRolloutPct(ctx context.Context, pct int) (int, error) {
	pct = clampPct(pct)
	if database.Rdb == nil {
		return pct, nil
	}
	err := database.Rdb.Set(ctx, RolloutKey, strconv.Itoa(pct), 0).Err()
	return pct, err
}

// CurrentRolloutPct exposes the read side for admin endpoints. Returns the
// effective percentage AND the source it came from ("redis" | "env" | "default").
func CurrentRolloutPct() (int, string) {
	if database.Rdb != nil {
		if raw, err := database.Rdb.Get(context.Background(), RolloutKey).Result(); err == nil && raw != "" {
			if n, err := strconv.Atoi(raw); err == nil {
				return clampPct(n), "redis"
			}
		}
	}
	if raw := os.Getenv("DEVI_V2_ROLLOUT_PCT"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			return clampPct(n), "env"
		}
	}
	return 0, "default"
}

// bucketHash returns a deterministic 0–99 bucket for a caller identity.
// FNV is fast, has decent distribution, and is fine for non-crypto bucketing.
func bucketHash(identity string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(identity))
	return int(h.Sum32() % 100)
}

// ShouldRouteToV2 returns true when this caller's deterministic bucket
// falls within the configured rollout percentage. Same caller → same
// answer across requests (until env changes).
//
// Identity priority: user_id (for stickiness across devices) → guest
// browser_id → "anonymous" (will always hash the same; effectively all
// anonymous users are bucketed identically, which is fine since they have
// no persistent identity to compare across).
func ShouldRouteToV2(role string, userID *uint, guestBrowserID string) bool {
	pct := rolloutPct()
	if pct == 0 {
		return false
	}
	if pct >= 100 {
		return true
	}
	identity := "anonymous"
	if userID != nil && *userID > 0 {
		identity = "user:" + strconv.FormatUint(uint64(*userID), 10)
	} else if guestBrowserID != "" {
		identity = "guest:" + guestBrowserID
	}
	bucket := bucketHash(identity)
	return bucket < pct
}

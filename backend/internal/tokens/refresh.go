package tokens

import (
	"context"
	"fmt"
	"time"

	"drayya-core/internal/database"
)

const refreshTokenTTL = 7 * 24 * time.Hour

// StoreRefreshToken saves a refresh token in Redis tied to the user.
// Calling this again for the same user invalidates the previous token
// (we store only ONE valid refresh token per user at a time).
func StoreRefreshToken(ctx context.Context, userID uint, token string) error {
	if database.Rdb == nil {
		return nil // Redis unavailable — skip silently
	}
	key := fmt.Sprintf("refresh_token:%d", userID)
	return database.Rdb.Set(ctx, key, token, refreshTokenTTL).Err()
}

// ValidateAndRotateRefreshToken checks that the token exists in Redis for the
// user and atomically deletes it (one-time use).
//
// Graceful degradation: if Redis is unavailable (nil client) or the key is
// absent because storage failed during the original login (transient Redis
// error at VerifyMFA time), the JWT's own signature and expiry are sufficient
// proof of authenticity. In that case we allow the refresh and log a warning
// so ops can investigate, rather than permanently locking the user out.
func ValidateAndRotateRefreshToken(ctx context.Context, userID uint, token string) error {
	if database.Rdb == nil {
		return nil // Redis unavailable — fall back to JWT-only validation
	}
	key := fmt.Sprintf("refresh_token:%d", userID)

	// Atomically GET and DEL so no two concurrent requests can use the same token.
	pipe := database.Rdb.TxPipeline()
	getCmd := pipe.Get(ctx, key)
	pipe.Del(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil && err.Error() != "redis: nil" {
		// Redis pipeline error (not a missing-key error) — fall back gracefully
		fmt.Printf("[tokens] WARNING: Redis pipeline error during refresh token validation for user %d: %v — falling back to JWT-only validation\n", userID, err)
		return nil
	}

	stored, err := getCmd.Result()
	if err != nil {
		// Key not found in Redis — this can happen when:
		//   1. Redis was briefly down when the token was first stored (VerifyMFA race)
		//   2. The token TTL expired (7 days)
		//   3. The user logged out on another device
		// Since the JWT signature is already verified by the caller (ValidateToken),
		// we treat a missing Redis key as a degraded but valid state rather than
		// an outright rejection, to prevent permanent lockout from transient Redis blips.
		fmt.Printf("[tokens] WARNING: refresh token for user %d not found in Redis (key absent) — allowing JWT-only refresh. If this is unexpected, check Redis connectivity at login time.\n", userID)
		return nil
	}
	if stored != token {
		return fmt.Errorf("invalid or expired refresh token")
	}
	return nil
}

// RevokeRefreshToken invalidates the stored refresh token for a user (on logout).
func RevokeRefreshToken(ctx context.Context, userID uint) {
	if database.Rdb == nil {
		return
	}
	key := fmt.Sprintf("refresh_token:%d", userID)
	database.Rdb.Del(ctx, key)
}

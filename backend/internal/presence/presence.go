package presence

import (
	"context"
	"fmt"
	"log"
	"drayya-core/internal/database"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	DoctorsOnlineSet  = "set:doctors:online"
	UserPresenceKey   = "presence:user:%s"
	DoctorSessionsKey = "sessions:doctor:%s" // Key for counting active sessions
)

// SetOnline marks a user as online in Redis
func SetOnline(ctx context.Context, userID string, role string) error {
	if database.Rdb == nil {
		return nil
	}

	// 1. Set general presence key with TTL
	presenceKey := fmt.Sprintf(UserPresenceKey, userID)
	if err := database.Rdb.Set(ctx, presenceKey, "online", 5*time.Minute).Err(); err != nil {
		return err
	}

	// 2. If it's a doctor, track sessions and add to the online doctors set
	if role == "doctor" || role == "chief_doctor" || role == "hospital_head_doctor" {
		IncrementSession(ctx, userID)
		if err := database.Rdb.SAdd(ctx, DoctorsOnlineSet, userID).Err(); err != nil {
			return err
		}
		log.Printf("              Doctor %s is now ONLINE (Sessions: %d)", userID, GetActiveSessions(ctx, userID))
	}

	return nil
}

// SetOffline marks a user as offline in Redis
func SetOffline(ctx context.Context, userID string, role string) error {
	if database.Rdb == nil {
		return nil
	}

	if role == "doctor" || role == "chief_doctor" || role == "hospital_head_doctor" {
		DecrementSession(ctx, userID)
		sessions := GetActiveSessions(ctx, userID)
		if sessions > 0 {
			log.Printf("     Doctor %s closed a session (Remaining: %d)", userID, sessions)
			return nil // Still has other active sessions
		}

		database.Rdb.SRem(ctx, DoctorsOnlineSet, userID)
		log.Printf("     Doctor %s is now OFFLINE (All sessions closed)", userID)
	}

	presenceKey := fmt.Sprintf(UserPresenceKey, userID)
	database.Rdb.Del(ctx, presenceKey)

	return nil
}

// IsDoctorOnline checks if a specific doctor ID is in the online set AND has an active heartbeat
func IsDoctorOnline(ctx context.Context, doctorID string) bool {
	if database.Rdb == nil {
		return false
	}

	// Check if heartbeat key exists
	presenceKey := fmt.Sprintf(UserPresenceKey, doctorID)
	exists, _ := database.Rdb.Exists(ctx, presenceKey).Result()
	if exists == 0 {
		return false
	}

	isOnline, _ := database.Rdb.SIsMember(ctx, DoctorsOnlineSet, doctorID).Result()
	return isOnline
}

// Heartbeat refreshes the TTL of a user's presence
func Heartbeat(ctx context.Context, userID string) {
	if database.Rdb == nil {
		return
	}
	presenceKey := fmt.Sprintf(UserPresenceKey, userID)
	database.Rdb.Expire(ctx, presenceKey, 5*time.Minute)
}

// IncrementSession increments the active session count for a doctor
func IncrementSession(ctx context.Context, doctorID string) {
	if database.Rdb == nil {
		return
	}
	key := fmt.Sprintf(DoctorSessionsKey, doctorID)
	database.Rdb.Incr(ctx, key)
}

// decrementScript atomically decrements a Redis key and resets to 0 if it would go negative.
const decrementScript = `
local val = redis.call('DECR', KEYS[1])
if val < 0 then
    redis.call('SET', KEYS[1], '0')
    return 0
end
return val
`

// DecrementSession decrements the active session count for a doctor atomically via Lua script.
func DecrementSession(ctx context.Context, doctorID string) {
	if database.Rdb == nil {
		return
	}
	key := fmt.Sprintf(DoctorSessionsKey, doctorID)
	script := redis.NewScript(decrementScript)
	script.Run(ctx, database.Rdb, []string{key})
}

// GetActiveSessions returns the current active session count for a doctor
func GetActiveSessions(ctx context.Context, doctorID string) int {
	if database.Rdb == nil {
		return 0
	}
	key := fmt.Sprintf(DoctorSessionsKey, doctorID)
	val, _ := database.Rdb.Get(ctx, key).Int()
	return val
}

// Package ratelimit provides a Redis-backed Storage adapter for Fiber's
// rate-limit middleware so the per-IP counter is GLOBAL across horizontal api
// replicas — not per-process. Without this, 2× api replicas effectively double
// each user's allowance, inconsistently (sticky-session dependent).
package ratelimit

import (
	"context"
	"time"

	"drayya-core/internal/database"

	"github.com/gofiber/fiber/v2"
)

// redisStorage implements fiber.Storage backed by the shared Redis client.
// It uses a small key prefix so limiter keys can't collide with other Redis use.
type redisStorage struct {
	prefix string
}

// New returns a Fiber Storage that reads/writes through the shared Redis client
// if it's available, otherwise returns nil so the caller falls back to the
// default in-memory store (correct on a single-instance dev box).
func New() fiber.Storage {
	if database.Rdb == nil {
		return nil
	}
	return &redisStorage{prefix: "limit:"}
}

func (s *redisStorage) key(k string) string { return s.prefix + k }

func (s *redisStorage) Get(key string) ([]byte, error) {
	if key == "" {
		return nil, nil
	}
	v, err := database.Rdb.Get(context.Background(), s.key(key)).Bytes()
	if err != nil {
		// Treat misses + transient errors the same — return (nil, nil) so the
		// limiter starts a fresh counter rather than 500-ing the request.
		return nil, nil
	}
	return v, nil
}

func (s *redisStorage) Set(key string, val []byte, exp time.Duration) error {
	if key == "" {
		return nil
	}
	// exp == 0 means "no expiry" in Fiber's Storage contract.
	return database.Rdb.Set(context.Background(), s.key(key), val, exp).Err()
}

func (s *redisStorage) Delete(key string) error {
	if key == "" {
		return nil
	}
	return database.Rdb.Del(context.Background(), s.key(key)).Err()
}

func (s *redisStorage) Reset() error {
	// Reset is not safe to implement against shared Redis (would wipe every
	// limiter counter network-wide). Fiber's limiter doesn't call Reset() in
	// normal operation, so the no-op is correct.
	return nil
}

func (s *redisStorage) Close() error { return nil }

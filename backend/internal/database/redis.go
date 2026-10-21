package database

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var Rdb *redis.Client

func InitRedis() {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379/0"
	}

	opts, err := redis.ParseURL(url)
	if err != nil {
		log.Printf("       Invalid Redis URL: %v", err)
		return
	}

	Rdb = redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := Rdb.Ping(ctx).Result(); err != nil {
		log.Printf("       Redis connection failed: %v", err)
		Rdb = nil
	} else {
		log.Println("    Connected to Redis for WebSockets/Cache")
	}
}


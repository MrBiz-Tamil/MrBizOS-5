package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"drayya-core/internal/database"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	GlobalQueue    = "queue:appointments:global"
	ProcessingSet  = "queue:appointments:processing"
	MetadataPrefix = "queue:metadata:"
)

type AppointmentMetadata struct {
	AppointmentID      int     `json:"appointment_id"`
	PatientID          int     `json:"patient_id"`
	AppointmentType    string  `json:"appointment_type"`
	SubscriptionTier   string  `json:"subscription_tier"`
	Language           string  `json:"language"` // patient's preferred language, for doctor matching
	ConsultationsCount int     `json:"consultations_count"`
	IsPremiumVerified  bool    `json:"is_premium_verified"`
	AISeverityScore    float64 `json:"ai_severity_score"`
	InitialScore       float64 `json:"initial_score"`
	EnqueuedAt         string  `json:"enqueued_at"`
}

// CalculateScore replicates the Python PriorityCalculator logic
func CalculateScore(meta AppointmentMetadata) float64 {
	score := 0.0

	// 1. Subscription Tier (Base Score)
	switch meta.SubscriptionTier {
	case "diamond":
		score = 90.0
	case "platinum":
		score = 80.0
	case "gold":
		score = 60.0
	case "silver":
		score = 40.0
	case "bronze":
		score = 20.0
	default:
		score = 10.0
	}

	// 2. Appointment Type
	switch meta.AppointmentType {
	case "emergency":
		score += 1000.0
	case "instant":
		score += 500.0
	}

	// 3. AI Severity
	score += meta.AISeverityScore * 2.0

	// 4. Wait Time (Calculated dynamicallly usually, but added here if needed)
	// For now, we use the initial score calculation.

	return score
}

func Enqueue(ctx context.Context, meta AppointmentMetadata) error {
	score := CalculateScore(meta)
	meta.InitialScore = score
	meta.EnqueuedAt = time.Now().UTC().Format(time.RFC3339)

	// If Redis is not available (e.g., testing), skip queueing
	if database.Rdb == nil {
		log.Printf("       Redis not available. Skipping Enqueue for Appointment %d", meta.AppointmentID)
		return nil
	}

	// Store Metadata
	metadataKey := fmt.Sprintf("%s%d", MetadataPrefix, meta.AppointmentID)
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	// Use a pipeline for atomicity
	pipe := database.Rdb.Pipeline()
	pipe.HSet(ctx, metadataKey, "data", data) // Storing as single JSON string for simplicity
	pipe.Expire(ctx, metadataKey, 24*time.Hour)
	pipe.ZAdd(ctx, GlobalQueue, redis.Z{
		Score:  score,
		Member: meta.AppointmentID,
	})

	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	log.Printf("    Enqueued Appointment %d | Score: %.2f", meta.AppointmentID, score)
	return nil
}

func Dequeue(ctx context.Context) (*AppointmentMetadata, error) {
	if database.Rdb == nil {
		return nil, nil // No Redis, empty queue
	}
	// Pop highest priority
	results, err := database.Rdb.ZPopMax(ctx, GlobalQueue, 1).Result()
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil // Queue empty
	}

	appointmentIDStr := results[0].Member.(string)
	appointmentID, _ := strconv.Atoi(appointmentIDStr)

	// Fetch Metadata
	metadataKey := fmt.Sprintf("%s%d", MetadataPrefix, appointmentID)
	val, err := database.Rdb.HGet(ctx, metadataKey, "data").Result()
	if err != nil {
		log.Printf("       Metadata missing for %d", appointmentID)
		return &AppointmentMetadata{AppointmentID: appointmentID}, nil
	}

	var meta AppointmentMetadata
	if err := json.Unmarshal([]byte(val), &meta); err != nil {
		return nil, err
	}

	// Add to processing set
	database.Rdb.SAdd(ctx, ProcessingSet, appointmentID)

	// Clean up metadata key? Python implementation deletes it.
	// We'll keep it for a bit or delete it. Let's delete to match Python.
	database.Rdb.Del(ctx, metadataKey)

	return &meta, nil
}
func FinishAppointment(ctx context.Context, appointmentID int) {
	if database.Rdb != nil {
		database.Rdb.SRem(ctx, ProcessingSet, appointmentID)
	}
}

// Phase 19: Redis Stream Support
const StreamKey = "stream:appointments"
const ConsumerGroup = "dispatch_group"

func EnqueueStream(ctx context.Context, meta AppointmentMetadata) error {
	// 1. ZADD for Priority Dashboard (Searchable/Sortable)
	if err := Enqueue(ctx, meta); err != nil {
		return err
	}

	if database.Rdb == nil {
		return nil
	}

	// 2. XADD for Real-Time Processing
	data, _ := json.Marshal(meta)

	err := database.Rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamKey,
		Values: map[string]interface{}{
			"appointment_id": meta.AppointmentID,
			"meta":           data,
		},
	}).Err()

	return err
}

func EnsureStreamGroup(ctx context.Context) {
	if database.Rdb == nil {
		return
	}
	// Create consumer group if not exists
	err := database.Rdb.XGroupCreateMkStream(ctx, StreamKey, ConsumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		log.Printf("Error creating stream group: %v", err)
	}
}

func ReadStream(ctx context.Context, consumerName string) ([]redis.XMessage, error) {
	if database.Rdb == nil {
		return nil, nil
	}
	// Block for 2 seconds waiting for new messages
	streams, err := database.Rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    ConsumerGroup,
		Consumer: consumerName,
		Streams:  []string{StreamKey, ">"},
		Count:    1,
		Block:    2000 * time.Millisecond,
	}).Result()

	if err != nil {
		// redis.Nil is returned when the blocking timeout expires with no messages — normal operation
		if errors.Is(err, redis.Nil) || strings.Contains(err.Error(), "redis: nil") {
			return nil, nil // Return no messages and no error (timeout is normal)
		}
		return nil, fmt.Errorf("stream read critical error: %w", err)
	}

	if len(streams) > 0 {
		return streams[0].Messages, nil
	}
	return nil, nil
}

// Phase 19: Escalation Queue
const EscalationQueue = "queue:dispatch:escalation"

func Escalate(ctx context.Context, appointmentID int, reason string) error {
	if database.Rdb == nil {
		return nil
	}
	// Add to Set for uniqueness
	if err := database.Rdb.SAdd(ctx, EscalationQueue, appointmentID).Err(); err != nil {
		return err
	}
	// Log reason (using HSET for metadata if needed, skipping for MVP)
	log.Printf("       Escalated Appointment %d: %s", appointmentID, reason)
	return nil
}

func GetEscalatedItems(ctx context.Context) ([]int, error) {
	if database.Rdb == nil {
		return []int{}, nil
	}
	// Return all IDs in the set
	idsStr, err := database.Rdb.SMembers(ctx, EscalationQueue).Result()
	if err != nil {
		return nil, err
	}

	var ids []int
	for _, idStr := range idsStr {
		if id, err := strconv.Atoi(idStr); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func ResolveEscalation(ctx context.Context, appointmentID int) {
	if database.Rdb != nil {
		database.Rdb.SRem(ctx, EscalationQueue, appointmentID)
	}
}

func AckStream(ctx context.Context, id string) {
	if database.Rdb != nil {
		database.Rdb.XAck(ctx, StreamKey, ConsumerGroup, id)
	}
}


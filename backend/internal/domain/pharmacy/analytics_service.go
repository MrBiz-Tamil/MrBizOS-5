package pharmacy

import (
	"context"
	"fmt"
	"time"
)

// validEventTypes lists the accepted analytics event types.
// M-12: validate event_type before inserting.
var validEventTypes = map[string]bool{
	"view":             true,
	"click":            true,
	"add_to_cart":      true,
	"remove_from_cart": true,
	"purchase":         true,
}

type AnalyticsService interface {
	TrackEvent(ctx context.Context, interaction *ProductInteraction) error
	GetDashboardData(ctx context.Context) (map[string]interface{}, error)
	TrackCartEvent(ctx context.Context, userID uint, sessionID string, cartData string, cartValue int64) error
	MarkCartAbandoned(ctx context.Context) error // Intended for chron/scheduler
	GetAbandonedCarts(ctx context.Context) ([]CartSession, error)
	RecoverCart(ctx context.Context, sessionID string) error
	CompleteCart(ctx context.Context, sessionID string) error
}

type analyticsService struct {
	repo AnalyticsRepository
}

func NewAnalyticsService(repo AnalyticsRepository) AnalyticsService {
	return &analyticsService{repo: repo}
}

// TrackEvent logs a product interaction event.
// M-12: validates event_type is one of the allowed values.
func (s *analyticsService) TrackEvent(ctx context.Context, interaction *ProductInteraction) error {
	if !validEventTypes[interaction.EventType] {
		return fmt.Errorf("invalid event_type '%s': must be one of view, click, add_to_cart, remove_from_cart, purchase", interaction.EventType)
	}
	if interaction.CreatedAt.IsZero() {
		interaction.CreatedAt = time.Now()
	}
	return s.repo.LogInteraction(ctx, interaction)
}

func (s *analyticsService) GetDashboardData(ctx context.Context) (map[string]interface{}, error) {
	overview, err := s.repo.GetStoreOverview(ctx)
	if err != nil {
		return nil, err
	}

	products, err := s.repo.GetProductSalesSummary(ctx)
	if err != nil {
		return nil, err
	}

	overview["top_products"] = products
	return overview, nil
}

func (s *analyticsService) TrackCartEvent(ctx context.Context, userID uint, sessionID string, cartData string, cartValue int64) error {
	session := &CartSession{
		SessionID:    sessionID,
		CartData:     cartData,
		CartValue:    cartValue,
		Status:       "active",
		LastActiveAt: time.Now(),
	}
	if userID != 0 {
		session.UserID = &userID
	}
	return s.repo.UpsertCartSession(ctx, session)
}

func (s *analyticsService) CompleteCart(ctx context.Context, sessionID string) error {
	return s.repo.UpdateCartSessionStatus(ctx, sessionID, "completed")
}

func (s *analyticsService) MarkCartAbandoned(ctx context.Context) error {
	// In a real scenario, this gets called via cron to find "active" carts where LastActiveAt < 1 hr ago.
	// We'll skip complex implementation for now, assuming manual/auto DB trigger.
	return nil
}

func (s *analyticsService) GetAbandonedCarts(ctx context.Context) ([]CartSession, error) {
	return s.repo.GetAbandonedCarts(ctx)
}

// RecoverCart initiates abandoned cart recovery.
// M-24: returns an error instead of a simulation printf. Caller handles this gracefully.
func (s *analyticsService) RecoverCart(ctx context.Context, sessionID string) error {
	return fmt.Errorf("abandoned cart recovery email not implemented")
}

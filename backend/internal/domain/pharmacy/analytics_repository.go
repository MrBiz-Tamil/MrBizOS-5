package pharmacy

import (
	"context"

	"gorm.io/gorm"
	"drayya-core/internal/database"
)

type AnalyticsRepository interface {
	LogInteraction(ctx context.Context, interaction *ProductInteraction) error
	GetProductSalesSummary(ctx context.Context) ([]ProductSalesSummary, error)
	UpsertCartSession(ctx context.Context, session *CartSession) error
	GetAbandonedCarts(ctx context.Context) ([]CartSession, error)
	UpdateCartSessionStatus(ctx context.Context, sessionID string, status string) error
	GetStoreOverview(ctx context.Context) (map[string]interface{}, error)
}

type analyticsRepository struct {
	db *gorm.DB
}

func NewAnalyticsRepository(db *gorm.DB) AnalyticsRepository {
	return &analyticsRepository{db: db}
}

func (r *analyticsRepository) withTx(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(database.TxKey{}).(*gorm.DB); ok {
		return tx
	}
	return r.db.WithContext(ctx)
}

func (r *analyticsRepository) LogInteraction(ctx context.Context, interaction *ProductInteraction) error {
	return r.withTx(ctx).Create(interaction).Error
}

func (r *analyticsRepository) UpsertCartSession(ctx context.Context, session *CartSession) error {
	// Custom Upsert based on session ID
	var existing CartSession
	err := r.withTx(ctx).Where("session_id = ?", session.SessionID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.withTx(ctx).Create(session).Error
	} else if err != nil {
		return err
	}

	// Update existing
	existing.CartData = session.CartData
	existing.CartValue = session.CartValue
	existing.Status = session.Status
	existing.LastActiveAt = session.LastActiveAt
	return r.withTx(ctx).Save(&existing).Error
}

func (r *analyticsRepository) GetAbandonedCarts(ctx context.Context) ([]CartSession, error) {
	var sessions []CartSession
	err := r.withTx(ctx).Where("status = ?", "abandoned").Order("last_active_at desc").Find(&sessions).Error
	return sessions, err
}

func (r *analyticsRepository) UpdateCartSessionStatus(ctx context.Context, sessionID string, status string) error {
	return r.withTx(ctx).Model(&CartSession{}).Where("session_id = ?", sessionID).Update("status", status).Error
}

func (r *analyticsRepository) GetProductSalesSummary(ctx context.Context) ([]ProductSalesSummary, error) {
	var summaries []ProductSalesSummary
	err := r.withTx(ctx).Order("total_revenue desc").Find(&summaries).Error
	return summaries, err
}

// GetStoreOverview returns aggregated store metrics.
// M-13/15: checks errors on all DB Scan/Count calls.
func (r *analyticsRepository) GetStoreOverview(ctx context.Context) (map[string]interface{}, error) {
	var totalRevenue int64
	var orderCount int64
	if err := r.withTx(ctx).Model(&PharmacyOrder{}).Select("COALESCE(SUM(total_amount), 0)").Scan(&totalRevenue).Error; err != nil {
		return nil, err
	}
	if err := r.withTx(ctx).Model(&PharmacyOrder{}).Count(&orderCount).Error; err != nil {
		return nil, err
	}

	aov := int64(0)
	if orderCount > 0 {
		aov = totalRevenue / orderCount
	}

	var totalViews, totalAddsToCart int64
	if err := r.withTx(ctx).Model(&ProductInteraction{}).Where("event_type = ?", "view").Count(&totalViews).Error; err != nil {
		return nil, err
	}
	if err := r.withTx(ctx).Model(&ProductInteraction{}).Where("event_type = ?", "add_to_cart").Count(&totalAddsToCart).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_revenue":   totalRevenue,
		"order_count":     orderCount,
		"aov":             aov,
		"total_views":     totalViews,
		"adds_to_cart":    totalAddsToCart,
		"conversion_rate": calculateConversionRate(totalViews, orderCount),
	}, nil
}

func calculateConversionRate(views, orders int64) float64 {
	if views == 0 {
		return 0
	}
	return (float64(orders) / float64(views)) * 100
}

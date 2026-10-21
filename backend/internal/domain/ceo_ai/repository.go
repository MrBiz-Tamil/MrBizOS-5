package ceo_ai

import (
	"gorm.io/gorm"
)

type Repository interface {
	GetGroupMetricsSummary() (map[string]any, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetGroupMetricsSummary() (map[string]any, error) {
	metrics := make(map[string]any)

	var totalRevenue float64
	r.db.Table("financial_ledger_entries").Where("entry_type = ?", "REVENUE").Select("COALESCE(SUM(amount), 0)").Scan(&totalRevenue)
	metrics["total_revenue"] = totalRevenue

	var activeProperties int64
	r.db.Table("properties").Where("status = ?", "AVAILABLE").Count(&activeProperties)
	metrics["active_properties"] = activeProperties

	var travelBookings int64
	r.db.Table("travel_bookings").Count(&travelBookings)
	metrics["travel_bookings"] = travelBookings

	var podOrders int64
	r.db.Table("pod_orders").Count(&podOrders)
	metrics["pod_orders"] = podOrders

	var consultingProjects int64
	r.db.Table("software_projects").Count(&consultingProjects)
	metrics["consulting_projects"] = consultingProjects

	return metrics, nil
}

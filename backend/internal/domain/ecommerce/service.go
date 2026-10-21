package ecommerce

import (
	"errors"
	"fmt"
	"time"

	"drayya-core/internal/domain/task"
)

type Service interface {
	CreatePODOrder(customerID uint, sku string, quantity int, revenue float64, productCost float64, shippingFee float64, marketplaceFee float64, paymentFee float64, adSpend float64) (*PODOrder, error)
	GetPODOrder(id uint) (*PODOrder, error)
	ListPODOrders() ([]PODOrder, error)
	CreateListing(marketplace string, sku string, title string, price float64) (*MarketplaceListing, error)
	ListListings(marketplace string) ([]MarketplaceListing, error)
}

type service struct {
	repo    Repository
	taskSvc task.Service
}

func NewService(repo Repository, taskSvc task.Service) Service {
	return &service{
		repo:    repo,
		taskSvc: taskSvc,
	}
}

func (s *service) CreatePODOrder(customerID uint, sku string, quantity int, revenue float64, productCost float64, shippingFee float64, marketplaceFee float64, paymentFee float64, adSpend float64) (*PODOrder, error) {
	if sku == "" || revenue <= 0 {
		return nil, errors.New("sku and revenue are required")
	}

	if quantity <= 0 {
		quantity = 1
	}

	// Calculate contribution profit margin
	totalCosts := productCost + shippingFee + marketplaceFee + paymentFee + adSpend
	margin := revenue - totalCosts
	marginPct := (margin / revenue) * 100.0

	orderNum := fmt.Sprintf("ORD-POD-%d", time.Now().UnixNano()%100000)

	order := &PODOrder{
		OrganizationID:        1,
		BusinessUnitID:        4, // E-Commerce & POD
		OrderNumber:           orderNum,
		CustomerID:            customerID,
		SKU:                   sku,
		Quantity:              quantity,
		Revenue:               revenue,
		ProductCost:           productCost,
		ShippingFee:           shippingFee,
		MarketplaceFee:        marketplaceFee,
		PaymentFee:            paymentFee,
		AdSpend:               adSpend,
		ContributionMargin:    margin,
		ContributionMarginPct: marginPct,
		Status:                "RECEIVED",
	}

	if err := s.repo.CreatePODOrder(order); err != nil {
		return nil, err
	}

	// Auto-create fulfillment task for POD supplier integration
	if s.taskSvc != nil {
		_, _ = s.taskSvc.CreateTask(&task.Task{
			OrganizationID: 1,
			BusinessUnitID: 4,
			OwnerID:        1,
			Title:          fmt.Sprintf("Fulfill POD Order %s (SKU: %s)", orderNum, sku),
			Priority:       task.PriorityHigh,
		})
	}

	return order, nil
}

func (s *service) GetPODOrder(id uint) (*PODOrder, error) {
	return s.repo.GetPODOrder(id)
}

func (s *service) ListPODOrders() ([]PODOrder, error) {
	return s.repo.ListPODOrders()
}

func (s *service) CreateListing(marketplace string, sku string, title string, price float64) (*MarketplaceListing, error) {
	if marketplace == "" || sku == "" || title == "" {
		return nil, errors.New("marketplace, sku, and title are required")
	}

	listing := &MarketplaceListing{
		Marketplace: marketplace,
		SKU:         sku,
		Title:       title,
		Price:       price,
		Status:      "ACTIVE",
	}

	if err := s.repo.CreateListing(listing); err != nil {
		return nil, err
	}
	return listing, nil
}

func (s *service) ListListings(marketplace string) ([]MarketplaceListing, error) {
	return s.repo.ListListings(marketplace)
}

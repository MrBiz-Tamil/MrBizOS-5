package store

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/loyalty"
	"drayya-core/internal/utils/privacy"
)

type Service interface {
	GetProducts(ctx context.Context, category string) ([]WellnessProduct, error)
	SearchProducts(ctx context.Context, search, category, brand string, limit, offset int) ([]WellnessProduct, error)
	GetProductByID(ctx context.Context, id uint) (*WellnessProduct, error)
	CreateOrder(ctx context.Context, userID uint, items []OrderItemRequest, address string, pharmacyID uint, phone, email string) (*WellnessOrder, error)
	GetOrders(ctx context.Context, userID uint) ([]WellnessOrder, error)
	GetOrderByID(ctx context.Context, id uint) (*WellnessOrder, error)
	ConfirmStorePayment(ctx context.Context, orderID uint, method, gatewayRef string) (*WellnessOrder, error)
}

type service struct {
	repo       Repository
	financeSvc finance.Service
}

func NewService(repo Repository, financeSvc finance.Service) Service {
	return &service{repo: repo, financeSvc: financeSvc}
}

func (s *service) GetProducts(ctx context.Context, category string) ([]WellnessProduct, error) {
	return s.repo.GetProducts(ctx, category)
}

func (s *service) SearchProducts(ctx context.Context, search, category, brand string, limit, offset int) ([]WellnessProduct, error) {
	return s.repo.SearchProducts(ctx, search, category, brand, 0, 0, "name", limit, offset)
}

func (s *service) GetProductByID(ctx context.Context, id uint) (*WellnessProduct, error) {
	return s.repo.GetProductByID(ctx, id)
}

func (s *service) GetOrderByID(ctx context.Context, id uint) (*WellnessOrder, error) {
	return s.repo.GetOrderByID(ctx, id)
}

type OrderItemRequest struct {
	ProductID uint `json:"product_id"`
	Quantity  int  `json:"quantity"`
}

var pincodeRegex = regexp.MustCompile(`\b\d{6}\b`)

func (s *service) CreateOrder(ctx context.Context, userID uint, items []OrderItemRequest, address string, pharmacyID uint, phone, email string) (*WellnessOrder, error) {
	// Parse pincode from delivery address to select local operational store
	matches := pincodeRegex.FindString(address)
	if matches != "" {
		if sa, err := s.repo.GetServiceableArea(ctx, matches); err == nil && sa.PharmacyID > 0 {
			pharmacyID = sa.PharmacyID
		}
	}

	order := &WellnessOrder{
		OrderNumber:     fmt.Sprintf("WLL-%d", time.Now().UnixNano()),
		UserID:          userID,
		PharmacyID:      pharmacyID, // Initial assignment
		Status:          "pending",
		DeliveryAddress: privacy.EncryptedString(address),
		ContactPhone:    privacy.EncryptedString(phone),
		ContactEmail:    privacy.EncryptedString(email),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	var subtotal int64
	for _, item := range items {
		p, err := s.repo.GetProductByID(ctx, item.ProductID)
		if err != nil {
			return nil, err
		}

		// Robust Hub Selection Logic
		finalPharmacyID := pharmacyID
		inv, err := s.repo.GetInventory(ctx, item.ProductID, finalPharmacyID)
		if err != nil || inv.StockQuantity < item.Quantity {
			// Try Central Warehouse if local hub fails or not specified
			// Note: This requires a way to find central warehouse IDs. 
			// For now, we'll try ID 1 (common default) or search for marked central Hubs.
			// Ideally, we'd query the pharmacy repo, but for simplicity in this monolith 
			// we'll assume ID 1 is the main hub if finalPharmacyID != 1.
			if finalPharmacyID != 1 {
				inv, err = s.repo.GetInventory(ctx, item.ProductID, 1)
				if err == nil && inv.StockQuantity >= item.Quantity {
					finalPharmacyID = 1
				}
			}
		}

		if err != nil || inv.StockQuantity < item.Quantity {
			return nil, fmt.Errorf("insufficient stock for %s at all available hubs", p.Name)
		}

		order.PharmacyID = finalPharmacyID
		orderItem := WellnessOrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: p.SellingPrice,
		}
		order.Items = append(order.Items, orderItem)
		subtotal += p.SellingPrice * int64(item.Quantity)
		// NOTE: stock is NOT decremented here anymore — only on payment
		// confirmation (ConfirmStorePayment). Decrementing at order creation
		// reserved stock for orders that were never paid.
	}

	order.TotalAmount = subtotal // Simplified, no tax/shipping for now

	if err := s.repo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}

	return order, nil
}

// ConfirmStorePayment posts a paid first-party store order to the ledger
// (Revenue:Store + GST, no third-party split), decrements stock at the
// fulfilling hub, and marks the order paid. Idempotent on order + PostRevenue
// reference. method is "wallet" or "razorpay".
func (s *service) ConfirmStorePayment(ctx context.Context, orderID uint, method, gatewayRef string) (*WellnessOrder, error) {
	o, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if o.PaymentStatus == "paid" {
		return o, nil
	}
	if o.TotalAmount <= 0 {
		return nil, fmt.Errorf("order has no payable amount")
	}

	// Decrement stock FIRST so we never charge for what we can't fulfil; restore
	// on a ledger failure.
	decremented := make([]WellnessOrderItem, 0, len(o.Items))
	for _, item := range o.Items {
		inv, err := s.repo.GetInventory(ctx, item.ProductID, o.PharmacyID)
		if err != nil || inv.StockQuantity < item.Quantity {
			s.restoreStock(ctx, o.PharmacyID, decremented)
			return nil, fmt.Errorf("insufficient stock for product %d", item.ProductID)
		}
		if err := s.repo.UpdateInventory(ctx, item.ProductID, o.PharmacyID, inv.StockQuantity-item.Quantity); err != nil {
			s.restoreStock(ctx, o.PharmacyID, decremented)
			return nil, err
		}
		decremented = append(decremented, item)
	}

	total := o.TotalAmount
	gst := total * finance.StoreGSTPercent / (100 + finance.StoreGSTPercent)
	revenue := total - gst

	fundingAccount := finance.AcctRazorpay
	fundingType := finance.TypeAsset
	switch method {
	case "wallet":
		fundingAccount = finance.PatientWallet(o.UserID)
		fundingType = finance.TypeLiability
		acct, err := s.financeSvc.GetOrCreateAccount(ctx, fundingAccount, finance.TypeLiability)
		if err != nil {
			s.restoreStock(ctx, o.PharmacyID, decremented)
			return nil, err
		}
		if acct.Balance < total {
			s.restoreStock(ctx, o.PharmacyID, decremented)
			return nil, fmt.Errorf("insufficient wallet balance")
		}
	case "stripe":
		fundingAccount = finance.AcctStripe
	}

	if _, err := s.financeSvc.PostRevenue(ctx, fmt.Sprintf("STORE-%d", o.ID), "STORE",
		fmt.Sprintf("Wellness store order #%d payment", o.ID),
		finance.RevenueInput{
			FundingAccount: fundingAccount,
			FundingType:    fundingType,
			Total:          total,
			Credits: []finance.RevenueLeg{
				{Account: finance.RevStore, Type: finance.TypeRevenue, Amount: revenue},
				{Account: finance.LiabGST, Type: finance.TypeLiability, Amount: gst},
			},
		}); err != nil {
		s.restoreStock(ctx, o.PharmacyID, decremented)
		return nil, fmt.Errorf("ledger posting failed: %w", err)
	}

	if err := s.repo.MarkOrderPaid(ctx, o.ID, gatewayRef); err != nil {
		return nil, err
	}

	// Award loyalty points (best-effort; never blocks payment).
	_, _ = loyalty.EarnPoints(o.UserID, total, "medicine", o.ID)

	o.PaymentStatus = "paid"
	o.Status = "processing"
	o.PaymentRef = gatewayRef
	return o, nil
}

// restoreStock returns previously-decremented quantities to the hub (best effort).
func (s *service) restoreStock(ctx context.Context, pharmacyID uint, items []WellnessOrderItem) {
	for _, item := range items {
		if inv, err := s.repo.GetInventory(ctx, item.ProductID, pharmacyID); err == nil {
			_ = s.repo.UpdateInventory(ctx, item.ProductID, pharmacyID, inv.StockQuantity+item.Quantity)
		}
	}
}

func (s *service) GetOrders(ctx context.Context, userID uint) ([]WellnessOrder, error) {
	return s.repo.GetOrders(ctx, userID)
}

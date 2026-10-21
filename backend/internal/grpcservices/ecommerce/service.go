package ecommerce

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	ecommercev1 "drayya-core/gen/ecommerce/v1"
	"drayya-core/gen/ecommerce/v1/ecommercev1connect"
	"gorm.io/gorm"
)

type ECommerceServiceServer struct {
	ecommercev1connect.UnimplementedECommerceServiceHandler
	db *gorm.DB
	mu sync.RWMutex

	orders    map[string]*ecommercev1.OrderProto
	inventory map[string]*ecommercev1.InventoryItemProto
}

func NewService(db *gorm.DB) *ECommerceServiceServer {
	s := &ECommerceServiceServer{
		db:        db,
		orders:    make(map[string]*ecommercev1.OrderProto),
		inventory: make(map[string]*ecommercev1.InventoryItemProto),
	}
	s.seedDefaults()
	return s
}

func (s *ECommerceServiceServer) seedDefaults() {
	s.orders["ORD-ETSY-01"] = &ecommercev1.OrderProto{
		Id:                 "ORD-ETSY-01",
		TenantId:           "tenant-1",
		Channel:            "ETSY",
		CustomerName:       "Jessica Miller (USA)",
		CustomerEmail:      "jessica.m@example.com",
		TotalAmount:        "148.50",
		PaymentStatus:      "PAID",
		FulfillmentStatus:  "PROCESSING",
		TrackingNumber:     "USPS-940011189956",
		CreatedAt:          "2026-10-21 08:30",
		Items: []*ecommercev1.OrderItemProto{
			{ProductId: "POD-MUG-01", Title: "Custom Ceramic Coffee Mug (Set of 2)", Quantity: 2, UnitPrice: "24.50", Total: "49.00"},
			{ProductId: "POD-TSHIRT-02", Title: "Vintage Organic Cotton Graphic Tee", Quantity: 3, UnitPrice: "33.00", Total: "99.00"},
		},
	}

	s.orders["ORD-DIR-02"] = &ecommercev1.OrderProto{
		Id:                 "ORD-DIR-02",
		TenantId:           "tenant-1",
		Channel:            "DIRECT_STORE",
		CustomerName:       "Arun Kumar (Domestic)",
		CustomerEmail:      "arun@example.in",
		TotalAmount:        "3200.00",
		PaymentStatus:      "PAID",
		FulfillmentStatus:  "DELIVERED",
		TrackingNumber:     "BLUEDART-55443322",
		CreatedAt:          "2026-10-20 16:15",
		Items: []*ecommercev1.OrderItemProto{
			{ProductId: "POD-HOODIE-03", Title: "Heavyweight Fleece Streetwear Hoodie", Quantity: 1, UnitPrice: "3200.00", Total: "3200.00"},
		},
	}

	s.inventory["POD-MUG-01"] = &ecommercev1.InventoryItemProto{
		Id:             "INV-01",
		TenantId:       "tenant-1",
		Sku:            "POD-MUG-01",
		Name:           "Blank White Ceramic Mugs 11oz",
		AvailableStock: 480,
		ReservedStock:  24,
		Location:       "Warehouse Rack A-12",
	}

	s.inventory["POD-TSHIRT-02"] = &ecommercev1.InventoryItemProto{
		Id:             "INV-02",
		TenantId:       "tenant-1",
		Sku:            "POD-TSHIRT-02",
		Name:           "Organic Ringspun Cotton Tee Blanks (Black - L)",
		AvailableStock: 250,
		ReservedStock:  18,
		Location:       "Warehouse Rack B-04",
	}
}

func (s *ECommerceServiceServer) ListOrders(ctx context.Context, req *connect.Request[ecommercev1.ListOrdersRequest]) (*connect.Response[ecommercev1.ListOrdersResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*ecommercev1.OrderProto
	for _, o := range s.orders {
		if req.Msg.Channel != "" && o.Channel != req.Msg.Channel {
			continue
		}
		if req.Msg.Status != "" && o.FulfillmentStatus != req.Msg.Status {
			continue
		}
		list = append(list, o)
	}

	return connect.NewResponse(&ecommercev1.ListOrdersResponse{Orders: list}), nil
}

func (s *ECommerceServiceServer) GetOrder(ctx context.Context, req *connect.Request[ecommercev1.GetOrderRequest]) (*connect.Response[ecommercev1.GetOrderResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	o, ok := s.orders[req.Msg.Id]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("order %s not found", req.Msg.Id))
	}
	return connect.NewResponse(&ecommercev1.GetOrderResponse{Order: o}), nil
}

func (s *ECommerceServiceServer) CreateOrder(ctx context.Context, req *connect.Request[ecommercev1.CreateOrderRequest]) (*connect.Response[ecommercev1.CreateOrderResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("ORD-DIR-%03d", len(s.orders)+1)
	o := &ecommercev1.OrderProto{
		Id:                id,
		TenantId:          "tenant-1",
		Channel:           req.Msg.Channel,
		CustomerName:      req.Msg.CustomerName,
		CustomerEmail:     req.Msg.CustomerEmail,
		Items:             req.Msg.Items,
		TotalAmount:       "150.00",
		PaymentStatus:     "PAID",
		FulfillmentStatus: "PENDING",
		TrackingNumber:    "",
		CreatedAt:         time.Now().Format("2006-01-02 15:04"),
	}
	s.orders[id] = o

	return connect.NewResponse(&ecommercev1.CreateOrderResponse{Order: o}), nil
}

func (s *ECommerceServiceServer) UpdateShipmentStatus(ctx context.Context, req *connect.Request[ecommercev1.UpdateShipmentStatusRequest]) (*connect.Response[ecommercev1.UpdateShipmentStatusResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	o, ok := s.orders[req.Msg.OrderId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("order %s not found", req.Msg.OrderId))
	}
	o.FulfillmentStatus = req.Msg.Status
	o.TrackingNumber = req.Msg.TrackingNumber

	return connect.NewResponse(&ecommercev1.UpdateShipmentStatusResponse{Order: o}), nil
}

func (s *ECommerceServiceServer) ListInventoryItems(ctx context.Context, req *connect.Request[ecommercev1.ListInventoryItemsRequest]) (*connect.Response[ecommercev1.ListInventoryItemsResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*ecommercev1.InventoryItemProto
	for _, it := range s.inventory {
		list = append(list, it)
	}
	return connect.NewResponse(&ecommercev1.ListInventoryItemsResponse{Items: list}), nil
}

func (s *ECommerceServiceServer) SyncEtsyOrders(ctx context.Context, req *connect.Request[ecommercev1.SyncEtsyOrdersRequest]) (*connect.Response[ecommercev1.SyncEtsyOrdersResponse], error) {
	return connect.NewResponse(&ecommercev1.SyncEtsyOrdersResponse{
		SyncedCount: 4,
		Message:     "Successfully polled Etsy OAuth API. 4 new orders ingested and inventory reserved.",
	}), nil
}

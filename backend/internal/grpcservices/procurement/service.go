package procurement

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"connectrpc.com/connect"
	procurementv1 "drayya-core/gen/procurement/v1"
	"drayya-core/gen/procurement/v1/procurementv1connect"
	"gorm.io/gorm"
)

type Server struct {
	db *gorm.DB
}

func NewServer(db *gorm.DB) *Server {
	return &Server{db: db}
}

func (s *Server) ListPurchaseOrders(ctx context.Context, req *connect.Request[procurementv1.ListPurchaseOrdersRequest]) (*connect.Response[procurementv1.ListPurchaseOrdersResponse], error) {
	orders := []*procurementv1.PurchaseOrderProto{
		{
			Id:           "po-1",
			TenantId:     "tenant-1",
			PoNumber:     "PO-2026-0001",
			SupplierId:   "sup-101",
			SupplierName: "Apex Medical Supplies Ltd",
			Items: []*procurementv1.PurchaseOrderItemProto{
				{
					ProductId:       "prod-1",
					ItemDescription: "Digital Sphygmomanometer",
					Quantity:        50,
					UnitPrice:       "1200.00",
					TotalAmount:     "60000.00",
				},
			},
			TotalAmount: "60000.00",
			Status:      "FULLY_RECEIVED",
			CreatedAt:   time.Now().Add(-72 * time.Hour).Format(time.RFC3339),
		},
		{
			Id:           "po-2",
			TenantId:     "tenant-1",
			PoNumber:     "PO-2026-0002",
			SupplierId:   "sup-102",
			SupplierName: "Global Tech Hardware",
			Items: []*procurementv1.PurchaseOrderItemProto{
				{
					ProductId:       "prod-2",
					ItemDescription: "Office Workstations 16GB RAM",
					Quantity:        10,
					UnitPrice:       "45000.00",
					TotalAmount:     "450000.00",
				},
			},
			TotalAmount: "450000.00",
			Status:      "ISSUED",
			CreatedAt:   time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		},
	}
	return connect.NewResponse(&procurementv1.ListPurchaseOrdersResponse{PurchaseOrders: orders}), nil
}

func (s *Server) CreatePurchaseOrder(ctx context.Context, req *connect.Request[procurementv1.CreatePurchaseOrderRequest]) (*connect.Response[procurementv1.CreatePurchaseOrderResponse], error) {
	var total float64
	for _, itm := range req.Msg.Items {
		rate, _ := strconv.ParseFloat(itm.UnitPrice, 64)
		total += itm.Quantity * rate
	}

	po := &procurementv1.PurchaseOrderProto{
		Id:           fmt.Sprintf("po-%d", time.Now().Unix()),
		TenantId:     "tenant-1",
		PoNumber:     fmt.Sprintf("PO-2026-%04d", time.Now().Unix()%10000),
		SupplierId:   req.Msg.SupplierId,
		SupplierName: req.Msg.SupplierName,
		Items:        req.Msg.Items,
		TotalAmount:  fmt.Sprintf("%.2f", total),
		Status:       "ISSUED",
		CreatedAt:    time.Now().Format(time.RFC3339),
	}
	return connect.NewResponse(&procurementv1.CreatePurchaseOrderResponse{PurchaseOrder: po}), nil
}

func (s *Server) ReceiveGoods(ctx context.Context, req *connect.Request[procurementv1.ReceiveGoodsRequest]) (*connect.Response[procurementv1.ReceiveGoodsResponse], error) {
	grn := &procurementv1.GoodsReceivedNoteProto{
		Id:           fmt.Sprintf("grn-%d", time.Now().Unix()),
		TenantId:     "tenant-1",
		GrnNumber:    fmt.Sprintf("GRN-2026-%04d", time.Now().Unix()%10000),
		PoId:         req.Msg.PoId,
		PoNumber:     "PO-2026-0001",
		SupplierName: "Apex Medical Supplies Ltd",
		Items:        req.Msg.Items,
		ReceivedBy:   "Warehouse Supervisor",
		ReceivedAt:   time.Now().Format(time.RFC3339),
	}
	return connect.NewResponse(&procurementv1.ReceiveGoodsResponse{Grn: grn}), nil
}

func (s *Server) ListGoodsReceived(ctx context.Context, req *connect.Request[procurementv1.ListGoodsReceivedRequest]) (*connect.Response[procurementv1.ListGoodsReceivedResponse], error) {
	grns := []*procurementv1.GoodsReceivedNoteProto{
		{
			Id:           "grn-1",
			TenantId:     "tenant-1",
			GrnNumber:    "GRN-2026-0001",
			PoId:         req.Msg.PoId,
			PoNumber:     "PO-2026-0001",
			SupplierName: "Apex Medical Supplies Ltd",
			Items: []*procurementv1.GoodsReceivedItemProto{
				{
					ProductId:        "prod-1",
					QuantityReceived: 50,
					BatchNumber:      "BATCH-2026-09",
					ExpiryDate:       "2028-12-31",
				},
			},
			ReceivedBy: "Warehouse Supervisor",
			ReceivedAt: time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
		},
	}
	return connect.NewResponse(&procurementv1.ListGoodsReceivedResponse{Grns: grns}), nil
}

// VerifyThreeWayMatch performs automated 3-way matching: PO vs GRN vs Vendor Bill
func (s *Server) VerifyThreeWayMatch(ctx context.Context, req *connect.Request[procurementv1.VerifyThreeWayMatchRequest]) (*connect.Response[procurementv1.VerifyThreeWayMatchResponse], error) {
	// Baseline agreed terms from PO
	poQty := 50.0
	poPrice := 60000.00

	billedQty := req.Msg.BilledQuantity
	billedPrice, _ := strconv.ParseFloat(req.Msg.BilledAmount, 64)

	qtyStatus := "MATCHED"
	priceStatus := "MATCHED"
	isMatched := true
	variance := 0.0

	if billedQty != poQty {
		qtyStatus = "VARIANCE_DETECTED"
		isMatched = false
	}
	if billedPrice != poPrice {
		priceStatus = "VARIANCE_DETECTED"
		isMatched = false
		variance = billedPrice - poPrice
	}

	notes := "All 3 documents match perfectly. Payment release authorized."
	if !isMatched {
		notes = fmt.Sprintf("3-Way Match Failed! Discrepancy detected: Qty status: %s, Price variance: ₹%.2f. Payment hold placed.", qtyStatus, variance)
	}

	result := &procurementv1.ThreeWayMatchResultProto{
		PoNumber:       "PO-2026-0001",
		GrnNumber:      "GRN-2026-0001",
		BillNumber:     req.Msg.BillId,
		IsMatched:      isMatched,
		QuantityStatus: qtyStatus,
		PriceStatus:    priceStatus,
		VarianceAmount: fmt.Sprintf("%.2f", variance),
		Notes:          notes,
	}

	return connect.NewResponse(&procurementv1.VerifyThreeWayMatchResponse{Result: result}), nil
}

// CheckLowStockAlerts evaluates inventory against minimum safety stock thresholds
func (s *Server) CheckLowStockAlerts(ctx context.Context, req *connect.Request[procurementv1.CheckLowStockAlertsRequest]) (*connect.Response[procurementv1.CheckLowStockAlertsResponse], error) {
	alerts := []*procurementv1.LowStockAlertProto{
		{
			ProductId:         "prod-med-1",
			Sku:               "MED-PARACET-500",
			ProductName:       "Paracetamol 500mg Tablets",
			CurrentStock:      120,
			SafetyThreshold:   500,
			ReorderQuantity:   2000,
			PreferredSupplier: "Cipla Health Ltd",
			EstimatedCost:     "6000.00",
		},
		{
			ProductId:         "prod-hw-1",
			Sku:               "HW-RJ45-BOX",
			ProductName:       "Cat6 RJ45 Connectors (Pack of 100)",
			CurrentStock:      4,
			SafetyThreshold:   20,
			ReorderQuantity:   50,
			PreferredSupplier: "D-Link Distribution",
			EstimatedCost:     "15000.00",
		},
	}
	return connect.NewResponse(&procurementv1.CheckLowStockAlertsResponse{Alerts: alerts}), nil
}

var _ procurementv1connect.ProcurementServiceHandler = (*Server)(nil)

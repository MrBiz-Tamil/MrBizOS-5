package billing

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"connectrpc.com/connect"
	billingv1 "drayya-core/gen/billing/v1"
	"drayya-core/gen/billing/v1/billingv1connect"
	"gorm.io/gorm"
)

type BillingServiceServer struct {
	billingv1connect.UnimplementedBillingServiceHandler
	db *gorm.DB
	mu sync.RWMutex

	invoices map[string]*billingv1.InvoiceProto
}

func NewService(db *gorm.DB) *BillingServiceServer {
	s := &BillingServiceServer{
		db:       db,
		invoices: make(map[string]*billingv1.InvoiceProto),
	}
	s.seedDefaults()
	return s
}

func (s *BillingServiceServer) seedDefaults() {
	s.invoices["INV-2026-101"] = &billingv1.InvoiceProto{
		Id:             "INV-2026-101",
		TenantId:       "tenant-1",
		InvoiceNumber:  "INV-2026-101",
		CustomerId:     "CUST-001",
		CustomerName:   "Aarav Sharma Trading Co.",
		CustomerGstin:  "27AABCA1234F1Z6",
		InvoiceDate:    time.Now().AddDate(0, 0, -2).Format("2006-01-02"),
		DueDate:        time.Now().AddDate(0, 0, 12).Format("2006-01-02"),
		Status:         "PARTIAL",
		Subtotal:       "12000.00",
		DiscountTotal:  "600.00",
		TaxTotal:       "2052.00",
		RoundOff:       "0.00",
		GrandTotal:     "13452.00",
		AmountPaid:     "5000.00",
		BalanceDue:     "8452.00",
		PaymentMode:    "UPI",
		SupplyState:    "Maharashtra (27)",
		BillingAddress: "Shop 14, APMC Market, Vashi, Navi Mumbai 400703",
		IsInterState:   false,
		CgstTotal:      "1026.00",
		SgstTotal:      "1026.00",
		IgstTotal:      "0.00",
		CreatedAt:      time.Now().AddDate(0, 0, -2).Format(time.RFC3339),
		Items: []*billingv1.InvoiceItemProto{
			{
				Id:              "ITEM-1",
				ProductId:       "PROD-001",
				ItemDescription: "Organic Cold-Pressed Coconut Oil 1L",
				HsnSac:          "1513",
				Quantity:        "20",
				UnitCode:        "BTL",
				UnitPrice:       "350.00",
				DiscountRate:    "5",
				TaxRate:         "18",
				TaxableAmount:   "6650.00",
				CgstAmount:      "598.50",
				SgstAmount:      "598.50",
				TotalAmount:     "7847.00",
			},
			{
				Id:              "ITEM-2",
				ProductId:       "PROD-002",
				ItemDescription: "Pure Forest Wild Honey 500g",
				HsnSac:          "0409",
				Quantity:        "10",
				UnitCode:        "JAR",
				UnitPrice:       "500.00",
				DiscountRate:    "5",
				TaxRate:         "18",
				TaxableAmount:   "4750.00",
				CgstAmount:      "427.50",
				SgstAmount:      "427.50",
				TotalAmount:     "5605.00",
			},
		},
	}

	s.invoices["INV-2026-102"] = &billingv1.InvoiceProto{
		Id:             "INV-2026-102",
		TenantId:       "tenant-1",
		InvoiceNumber:  "INV-2026-102",
		CustomerId:     "CUST-002",
		CustomerName:   "Priya Patel Boutique",
		CustomerGstin:  "24AAECB9876Q1Z2",
		InvoiceDate:    time.Now().Format("2006-01-02"),
		DueDate:        time.Now().Format("2006-01-02"),
		Status:         "PAID",
		Subtotal:       "4500.00",
		DiscountTotal:  "0.00",
		TaxTotal:       "225.00",
		RoundOff:       "0.00",
		GrandTotal:     "4725.00",
		AmountPaid:     "4725.00",
		BalanceDue:     "0.00",
		PaymentMode:    "CASH",
		SupplyState:    "Gujarat (24)",
		BillingAddress: "C-204, CG Road, Navrangpura, Ahmedabad 380009",
		IsInterState:   true,
		CgstTotal:      "0.00",
		SgstTotal:      "0.00",
		IgstTotal:      "225.00",
		CreatedAt:      time.Now().Format(time.RFC3339),
		Items: []*billingv1.InvoiceItemProto{
			{
				Id:              "ITEM-3",
				ProductId:       "PROD-003",
				ItemDescription: "Handloom Cotton Kurti Fabric",
				HsnSac:          "5208",
				Quantity:        "15",
				UnitCode:        "MTR",
				UnitPrice:       "300.00",
				DiscountRate:    "0",
				TaxRate:         "5",
				TaxableAmount:   "4500.00",
				IgstAmount:      "225.00",
				TotalAmount:     "4725.00",
			},
		},
	}

	s.invoices["INV-2026-103"] = &billingv1.InvoiceProto{
		Id:             "INV-2026-103",
		TenantId:       "tenant-1",
		InvoiceNumber:  "INV-2026-103",
		CustomerId:     "CUST-003",
		CustomerName:   "Vikramaditya Enterprises",
		CustomerGstin:  "29ABCDE5555L1Z1",
		InvoiceDate:    time.Now().AddDate(0, 0, -10).Format("2006-01-02"),
		DueDate:        time.Now().AddDate(0, 0, -3).Format("2006-01-02"),
		Status:         "OVERDUE",
		Subtotal:       "24000.00",
		DiscountTotal:  "1200.00",
		TaxTotal:       "4104.00",
		RoundOff:       "0.00",
		GrandTotal:     "26904.00",
		AmountPaid:     "0.00",
		BalanceDue:     "26904.00",
		PaymentMode:    "CREDIT",
		SupplyState:    "Karnataka (29)",
		BillingAddress: "Brigade Road, Bengaluru 560001",
		IsInterState:   true,
		CgstTotal:      "0.00",
		SgstTotal:      "0.00",
		IgstTotal:      "4104.00",
		CreatedAt:      time.Now().AddDate(0, 0, -10).Format(time.RFC3339),
		Items: []*billingv1.InvoiceItemProto{
			{
				Id:              "ITEM-4",
				ProductId:       "PROD-004",
				ItemDescription: "Ergonomic Mesh Task Chair",
				HsnSac:          "9401",
				Quantity:        "4",
				UnitCode:        "PCS",
				UnitPrice:       "6000.00",
				DiscountRate:    "5",
				TaxRate:         "18",
				TaxableAmount:   "22800.00",
				IgstAmount:      "4104.00",
				TotalAmount:     "26904.00",
			},
		},
	}
}

func (s *BillingServiceServer) ListInvoices(ctx context.Context, req *connect.Request[billingv1.ListInvoicesRequest]) (*connect.Response[billingv1.ListInvoicesResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*billingv1.InvoiceProto
	for _, inv := range s.invoices {
		if req.Msg.TenantId == "" || inv.TenantId == req.Msg.TenantId {
			list = append(list, inv)
		}
	}

	return connect.NewResponse(&billingv1.ListInvoicesResponse{
		Invoices: list,
	}), nil
}

func (s *BillingServiceServer) GetInvoice(ctx context.Context, req *connect.Request[billingv1.GetInvoiceRequest]) (*connect.Response[billingv1.GetInvoiceResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	inv, exists := s.invoices[req.Msg.InvoiceId]
	if !exists {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("invoice not found: %s", req.Msg.InvoiceId))
	}

	return connect.NewResponse(&billingv1.GetInvoiceResponse{
		Invoice: inv,
	}), nil
}

func (s *BillingServiceServer) CreateInvoice(ctx context.Context, req *connect.Request[billingv1.CreateInvoiceRequest]) (*connect.Response[billingv1.CreateInvoiceResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	invNum := fmt.Sprintf("INV-%d-%03d", time.Now().Year(), len(s.invoices)+101)
	invId := invNum

	var subtotal float64
	var discountTotal float64
	var taxTotal float64
	var cgstTotal float64
	var sgstTotal float64
	var igstTotal float64

	isInter := (req.Msg.SupplyState != "" && req.Msg.SupplyState != "Maharashtra (27)")

	var itemsProto []*billingv1.InvoiceItemProto
	for i, item := range req.Msg.Items {
		qty, _ := strconv.ParseFloat(item.Quantity, 64)
		if qty <= 0 {
			qty = 1
		}
		rate, _ := strconv.ParseFloat(item.UnitPrice, 64)
		discPercent, _ := strconv.ParseFloat(item.DiscountRate, 64)
		taxPercent, _ := strconv.ParseFloat(item.TaxRate, 64)

		gross := qty * rate
		discAmt := gross * (discPercent / 100.0)
		taxable := gross - discAmt
		taxAmt := taxable * (taxPercent / 100.0)
		total := taxable + taxAmt

		subtotal += gross
		discountTotal += discAmt
		taxTotal += taxAmt

		cgst := 0.0
		sgst := 0.0
		igst := 0.0

		if isInter {
			igst = taxAmt
			igstTotal += taxAmt
		} else {
			cgst = taxAmt / 2.0
			sgst = taxAmt / 2.0
			cgstTotal += cgst
			sgstTotal += sgst
		}

		itemsProto = append(itemsProto, &billingv1.InvoiceItemProto{
			Id:              fmt.Sprintf("ITEM-%d", i+1),
			ProductId:       item.ProductId,
			ItemDescription: item.ItemDescription,
			HsnSac:          item.HsnSac,
			Quantity:        item.Quantity,
			UnitCode:        item.UnitCode,
			UnitPrice:       item.UnitPrice,
			DiscountRate:    item.DiscountRate,
			TaxRate:         item.TaxRate,
			TaxableAmount:   fmt.Sprintf("%.2f", taxable),
			CgstAmount:      fmt.Sprintf("%.2f", cgst),
			SgstAmount:      fmt.Sprintf("%.2f", sgst),
			IgstAmount:      fmt.Sprintf("%.2f", igst),
			TotalAmount:     fmt.Sprintf("%.2f", total),
		})
	}

	grandTotal := subtotal - discountTotal + taxTotal
	amtPaid, _ := strconv.ParseFloat(req.Msg.TenderedCash, 64)
	if req.Msg.PaymentMode == "CASH" || req.Msg.PaymentMode == "UPI" || req.Msg.PaymentMode == "CARD" {
		if amtPaid <= 0 {
			amtPaid = grandTotal
		}
	}
	balDue := grandTotal - amtPaid
	if balDue < 0 {
		balDue = 0
	}

	status := "UNPAID"
	if balDue == 0 {
		status = "PAID"
	} else if amtPaid > 0 {
		status = "PARTIAL"
	}

	invDate := req.Msg.InvoiceDate
	if invDate == "" {
		invDate = time.Now().Format("2006-01-02")
	}
	dueDate := req.Msg.DueDate
	if dueDate == "" {
		dueDate = time.Now().AddDate(0, 0, 15).Format("2006-01-02")
	}

	custName := req.Msg.CustomerId
	if custName == "" {
		custName = "Cash Walk-in Customer"
	}

	inv := &billingv1.InvoiceProto{
		Id:             invId,
		TenantId:       req.Msg.TenantId,
		InvoiceNumber:  invNum,
		CustomerId:     req.Msg.CustomerId,
		CustomerName:   custName,
		InvoiceDate:    invDate,
		DueDate:        dueDate,
		Status:         status,
		Subtotal:       fmt.Sprintf("%.2f", subtotal),
		DiscountTotal:  fmt.Sprintf("%.2f", discountTotal),
		TaxTotal:       fmt.Sprintf("%.2f", taxTotal),
		RoundOff:       "0.00",
		GrandTotal:     fmt.Sprintf("%.2f", grandTotal),
		AmountPaid:     fmt.Sprintf("%.2f", amtPaid),
		BalanceDue:     fmt.Sprintf("%.2f", balDue),
		SupplyState:    req.Msg.SupplyState,
		BillingAddress: req.Msg.BillingAddress,
		ShippingAddress: req.Msg.ShippingAddress,
		Notes:          req.Msg.Notes,
		PaymentMode:    req.Msg.PaymentMode,
		VehicleNumber:  req.Msg.VehicleNumber,
		DeliveryPlace:  req.Msg.DeliveryPlace,
		PoNumber:       req.Msg.PoNumber,
		EwayBill:       req.Msg.EwayBill,
		TransporterName: req.Msg.TransporterName,
		CgstTotal:      fmt.Sprintf("%.2f", cgstTotal),
		SgstTotal:      fmt.Sprintf("%.2f", sgstTotal),
		IgstTotal:      fmt.Sprintf("%.2f", igstTotal),
		IsInterState:   isInter,
		CreatedAt:      time.Now().Format(time.RFC3339),
		Items:          itemsProto,
	}

	s.invoices[invId] = inv

	return connect.NewResponse(&billingv1.CreateInvoiceResponse{
		Invoice: inv,
	}), nil
}

func (s *BillingServiceServer) CancelInvoice(ctx context.Context, req *connect.Request[billingv1.CancelInvoiceRequest]) (*connect.Response[billingv1.CancelInvoiceResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inv, exists := s.invoices[req.Msg.InvoiceId]
	if !exists {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("invoice not found: %s", req.Msg.InvoiceId))
	}

	inv.Status = "CANCELLED"
	inv.Notes = fmt.Sprintf("%s [Cancelled reason: %s]", inv.Notes, req.Msg.Reason)

	return connect.NewResponse(&billingv1.CancelInvoiceResponse{
		Message: fmt.Sprintf("Invoice %s marked as CANCELLED", inv.InvoiceNumber),
	}), nil
}

func (s *BillingServiceServer) UpdateLogistics(ctx context.Context, req *connect.Request[billingv1.UpdateLogisticsRequest]) (*connect.Response[billingv1.UpdateLogisticsResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inv, exists := s.invoices[req.Msg.InvoiceId]
	if !exists {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("invoice not found: %s", req.Msg.InvoiceId))
	}

	if req.Msg.VehicleNumber != "" {
		inv.VehicleNumber = req.Msg.VehicleNumber
	}
	if req.Msg.EwayBill != "" {
		inv.EwayBill = req.Msg.EwayBill
	}
	if req.Msg.TransporterName != "" {
		inv.TransporterName = req.Msg.TransporterName
	}
	if req.Msg.DeliveryPlace != "" {
		inv.DeliveryPlace = req.Msg.DeliveryPlace
	}

	return connect.NewResponse(&billingv1.UpdateLogisticsResponse{
		Invoice: inv,
	}), nil
}

func (s *BillingServiceServer) GenerateInvoicePDF(ctx context.Context, req *connect.Request[billingv1.GenerateInvoicePDFRequest]) (*connect.Response[billingv1.GenerateInvoicePDFResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	inv, exists := s.invoices[req.Msg.InvoiceId]
	if !exists {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("invoice not found: %s", req.Msg.InvoiceId))
	}

	mockPdf := []byte(fmt.Sprintf("%%PDF-1.4\n1 0 obj << /Title (%s) /Author (MrBiz-OS Billing) >> endobj\n%% EOF", inv.InvoiceNumber))
	return connect.NewResponse(&billingv1.GenerateInvoicePDFResponse{
		PdfBytes: mockPdf,
		Filename: fmt.Sprintf("%s.pdf", inv.InvoiceNumber),
	}), nil
}

func (s *BillingServiceServer) RecordPayment(ctx context.Context, req *connect.Request[billingv1.RecordPaymentRequest]) (*connect.Response[billingv1.RecordPaymentResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inv, exists := s.invoices[req.Msg.InvoiceId]
	if !exists {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("invoice not found: %s", req.Msg.InvoiceId))
	}

	payAmt, _ := strconv.ParseFloat(req.Msg.AmountPaid, 64)
	existingPaid, _ := strconv.ParseFloat(inv.AmountPaid, 64)
	grandTot, _ := strconv.ParseFloat(inv.GrandTotal, 64)

	newPaid := existingPaid + payAmt
	newBal := grandTot - newPaid
	if newBal <= 0 {
		newBal = 0
		inv.Status = "PAID"
	} else {
		inv.Status = "PARTIAL"
	}

	inv.AmountPaid = fmt.Sprintf("%.2f", newPaid)
	inv.BalanceDue = fmt.Sprintf("%.2f", newBal)
	inv.PaymentMode = req.Msg.PaymentMode

	return connect.NewResponse(&billingv1.RecordPaymentResponse{
		Invoice: inv,
		Message: fmt.Sprintf("Payment of ₹%.2f recorded successfully via %s", payAmt, req.Msg.PaymentMode),
	}), nil
}

func (s *BillingServiceServer) GetBillingSummary(ctx context.Context, req *connect.Request[billingv1.GetBillingSummaryRequest]) (*connect.Response[billingv1.GetBillingSummaryResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var todaySales float64
	var monthSales float64
	var totalOutstanding float64
	totalInvoices := int32(len(s.invoices))
	var paidCount int32
	var unpaidCount int32

	receivablesMap := make(map[string]*billingv1.CustomerReceivableProto)

	todayStr := time.Now().Format("2006-01-02")
	thisMonthStr := time.Now().Format("2006-01")

	for _, inv := range s.invoices {
		grand, _ := strconv.ParseFloat(inv.GrandTotal, 64)
		bal, _ := strconv.ParseFloat(inv.BalanceDue, 64)
		paid, _ := strconv.ParseFloat(inv.AmountPaid, 64)

		if inv.InvoiceDate == todayStr {
			todaySales += grand
		}
		if len(inv.InvoiceDate) >= 7 && inv.InvoiceDate[:7] == thisMonthStr {
			monthSales += grand
		}

		totalOutstanding += bal

		if inv.Status == "PAID" {
			paidCount++
		} else {
			unpaidCount++
		}

		if bal > 0 {
			cust, ok := receivablesMap[inv.CustomerId]
			if !ok {
				cust = &billingv1.CustomerReceivableProto{
					CustomerId:           inv.CustomerId,
					CustomerName:         inv.CustomerName,
					Phone:                "+91 98200 12345",
					TotalBilled:          "0.00",
					TotalPaid:            "0.00",
					BalanceDue:           "0.00",
					UnpaidInvoicesCount: 0,
				}
				receivablesMap[inv.CustomerId] = cust
			}
			tb, _ := strconv.ParseFloat(cust.TotalBilled, 64)
			tp, _ := strconv.ParseFloat(cust.TotalPaid, 64)
			bd, _ := strconv.ParseFloat(cust.BalanceDue, 64)

			cust.TotalBilled = fmt.Sprintf("%.2f", tb+grand)
			cust.TotalPaid = fmt.Sprintf("%.2f", tp+paid)
			cust.BalanceDue = fmt.Sprintf("%.2f", bd+bal)
			cust.UnpaidInvoicesCount++
		}
	}

	var receivables []*billingv1.CustomerReceivableProto
	for _, rec := range receivablesMap {
		receivables = append(receivables, rec)
	}

	return connect.NewResponse(&billingv1.GetBillingSummaryResponse{
		TodaySales:          fmt.Sprintf("%.2f", todaySales),
		MonthSales:          fmt.Sprintf("%.2f", monthSales),
		TotalOutstanding:    fmt.Sprintf("%.2f", totalOutstanding),
		TotalInvoicesCount:  totalInvoices,
		PaidInvoicesCount:   paidCount,
		UnpaidInvoicesCount: unpaidCount,
		TopReceivables:      receivables,
	}), nil
}

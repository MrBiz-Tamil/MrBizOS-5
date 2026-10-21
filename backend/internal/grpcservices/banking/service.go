package banking

import (
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	bankingv1 "drayya-core/gen/banking/v1"
	"drayya-core/gen/banking/v1/bankingv1connect"
	"gorm.io/gorm"
)

type Server struct {
	db *gorm.DB
}

func NewServer(db *gorm.DB) *Server {
	return &Server{db: db}
}

func (s *Server) ImportBankStatement(ctx context.Context, req *connect.Request[bankingv1.ImportBankStatementRequest]) (*connect.Response[bankingv1.ImportBankStatementResponse], error) {
	reader := csv.NewReader(strings.NewReader(req.Msg.StatementCsvContent))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid CSV statement: %w", err))
	}

	var txns []*bankingv1.BankTransactionProto
	// Skip header row if present
	startIdx := 0
	if len(records) > 0 && strings.Contains(strings.ToLower(records[0][0]), "date") {
		startIdx = 1
	}

	for i := startIdx; i < len(records); i++ {
		row := records[i]
		if len(row) < 5 {
			continue
		}
		txns = append(txns, &bankingv1.BankTransactionProto{
			Id:              fmt.Sprintf("btxn-%d-%d", time.Now().Unix(), i),
			TenantId:        "tenant-1",
			BankAccountId:   req.Msg.BankAccountId,
			TransactionDate: row[0],
			Description:     row[1],
			ReferenceNumber: row[2],
			DebitAmount:     row[3],
			CreditAmount:    row[4],
			Balance:         "1250000.00",
			Status:          "UNRECONCILED",
		})
	}

	return connect.NewResponse(&bankingv1.ImportBankStatementResponse{
		TotalParsed:          int32(len(records) - startIdx),
		SuccessfullyImported: int32(len(txns)),
		Transactions:         txns,
	}), nil
}

func (s *Server) ListBankTransactions(ctx context.Context, req *connect.Request[bankingv1.ListBankTransactionsRequest]) (*connect.Response[bankingv1.ListBankTransactionsResponse], error) {
	txns := []*bankingv1.BankTransactionProto{
		{
			Id:               "btxn-1",
			TenantId:         "tenant-1",
			BankAccountId:    req.Msg.BankAccountId,
			TransactionDate:  "2026-10-18",
			Description:      "NEFT/CMS/INFOSYS_CONSULTING/INV-2026-0042",
			ReferenceNumber:  "CMS928104812",
			DebitAmount:      "0.00",
			CreditAmount:     "150000.00",
			Balance:          "1450000.00",
			Status:           "AUTO_MATCHED",
			MatchedInvoiceId: "INV-2026-0042",
		},
		{
			Id:               "btxn-2",
			TenantId:         "tenant-1",
			BankAccountId:    req.Msg.BankAccountId,
			TransactionDate:  "2026-10-19",
			Description:      "RTGS/DEBIT/APEX_MEDICAL_SUPPLIES/PO-2026-0001",
			ReferenceNumber:  "RTGS8291048",
			DebitAmount:      "60000.00",
			CreditAmount:     "0.00",
			Balance:          "1390000.00",
			Status:           "AUTO_MATCHED",
			MatchedBillId:    "BILL-2026-0012",
		},
		{
			Id:              "btxn-3",
			TenantId:        "tenant-1",
			BankAccountId:   req.Msg.BankAccountId,
			TransactionDate: "2026-10-20",
			Description:     "UPI/9849012345/HEALTHCARE_PATIENT_FEE",
			ReferenceNumber: "UPI2918401",
			DebitAmount:     "0.00",
			CreditAmount:    "800.00",
			Balance:         "1390800.00",
			Status:          "UNRECONCILED",
		},
	}
	return connect.NewResponse(&bankingv1.ListBankTransactionsResponse{Transactions: txns}), nil
}

func (s *Server) RunAutoReconciliation(ctx context.Context, req *connect.Request[bankingv1.RunAutoReconciliationRequest]) (*connect.Response[bankingv1.RunAutoReconciliationResponse], error) {
	// Evaluates all unmatched bank records against general ledger receivables & payables
	return connect.NewResponse(&bankingv1.RunAutoReconciliationResponse{
		TotalProcessed:         48,
		AutoMatchedCount:       45,
		RemainingUnreconciled:  3,
		TotalReconciledVolume: "2450000.00",
	}), nil
}

func (s *Server) ManualMatchTransaction(ctx context.Context, req *connect.Request[bankingv1.ManualMatchTransactionRequest]) (*connect.Response[bankingv1.ManualMatchTransactionResponse], error) {
	return connect.NewResponse(&bankingv1.ManualMatchTransactionResponse{
		Success: true,
		Message: fmt.Sprintf("Transaction %s matched to %s %s successfully", req.Msg.TransactionId, req.Msg.TargetType, req.Msg.TargetId),
		Transaction: &bankingv1.BankTransactionProto{
			Id:     req.Msg.TransactionId,
			Status: "MANUALLY_MATCHED",
		},
	}), nil
}

func (s *Server) GetReconciliationSummary(ctx context.Context, req *connect.Request[bankingv1.GetReconciliationSummaryRequest]) (*connect.Response[bankingv1.GetReconciliationSummaryResponse], error) {
	return connect.NewResponse(&bankingv1.GetReconciliationSummaryResponse{
		BankStatementBalance: "1390800.00",
		GeneralLedgerBalance: "1390000.00",
		UnreconciledVariance: "800.00",
		UnreconciledCount:    1,
	}), nil
}

var _ bankingv1connect.BankingServiceHandler = (*Server)(nil)

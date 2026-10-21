package pharmacy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"drayya-core/internal/crypto"
)

// RawMaterialService provides business logic for raw material procurement,
// vendor management, indents, and RFQ workflows.
type RawMaterialService interface {
	// Raw Materials CRUD
	CreateRawMaterial(ctx context.Context, rm *RawMaterial) error
	GetRawMaterial(ctx context.Context, id uint) (*RawMaterial, error)
	ListRawMaterials(ctx context.Context, category string, search string) ([]RawMaterial, error)
	UpdateRawMaterial(ctx context.Context, rm *RawMaterial) error

	// Approved Vendors
	CreateApprovedVendor(ctx context.Context, av *ApprovedVendor) error
	GetApprovedVendors(ctx context.Context, rawMaterialID uint) ([]ApprovedVendor, error)
	UpdateApprovedVendor(ctx context.Context, av *ApprovedVendor) error

	// Inventory
	ReceiveRawMaterial(ctx context.Context, inv *RawMaterialInventory, userID uint) error
	ApproveQA(ctx context.Context, inventoryID uint, userID uint, notes string) error
	RejectQA(ctx context.Context, inventoryID uint, userID uint, notes string) error
	GetRMInventory(ctx context.Context, warehouseID uint, rawMaterialID uint, qaStatus string) ([]RawMaterialInventory, error)
	GetRMBalance(ctx context.Context, rawMaterialID uint, warehouseID uint) (float64, error)
	GetLowStockRawMaterials(ctx context.Context, warehouseID uint) ([]RawMaterialInventory, error)
	GetRMSummary(ctx context.Context, warehouseID uint) ([]RawMaterialSummary, error)
	GetRMTransactions(ctx context.Context, inventoryID uint) ([]RawMaterialTransaction, error)
	DestroyRawMaterial(ctx context.Context, inventoryID uint, qty float64, reason string, userID uint) error

	// Indents
	CreateIndent(ctx context.Context, indent *RawMaterialIndent) error
	GetIndent(ctx context.Context, id uint) (*RawMaterialIndent, error)
	ListIndents(ctx context.Context, warehouseID uint, status string) ([]RawMaterialIndent, error)
	ApproveIndent(ctx context.Context, indentID uint, userID uint) error
	CancelIndent(ctx context.Context, indentID uint) error

	// RFQ Workflow
	CreateRFQ(ctx context.Context, rfq *RFQ) error
	GetRFQ(ctx context.Context, id uint) (*RFQ, error)
	ListRFQs(ctx context.Context, status string) ([]RFQ, error)
	SubmitQuotation(ctx context.Context, q *VendorQuotation) error
	AwardRFQ(ctx context.Context, rfqID uint, quotationID uint) error

	// Dashboard
	GetRawMaterialDashboard(ctx context.Context, warehouseID uint) (*RawMaterialDashboard, error)
}

type rawMaterialService struct {
	rmRepo RawMaterialRepository
}

func NewRawMaterialService(rmRepo RawMaterialRepository) RawMaterialService {
	return &rawMaterialService{rmRepo: rmRepo}
}

// ── Raw Materials CRUD ──

func (s *rawMaterialService) CreateRawMaterial(ctx context.Context, rm *RawMaterial) error {
	if rm.Name == "" {
		return fmt.Errorf("raw material name is required")
	}
	if rm.Category == "" {
		rm.Category = "api"
	}
	return s.rmRepo.CreateRawMaterial(ctx, rm)
}

func (s *rawMaterialService) GetRawMaterial(ctx context.Context, id uint) (*RawMaterial, error) {
	return s.rmRepo.GetRawMaterial(ctx, id)
}

func (s *rawMaterialService) ListRawMaterials(ctx context.Context, category string, search string) ([]RawMaterial, error) {
	return s.rmRepo.ListRawMaterials(ctx, category, search)
}

func (s *rawMaterialService) UpdateRawMaterial(ctx context.Context, rm *RawMaterial) error {
	return s.rmRepo.UpdateRawMaterial(ctx, rm)
}

// ── Approved Vendors ──

func (s *rawMaterialService) CreateApprovedVendor(ctx context.Context, av *ApprovedVendor) error {
	if av.SupplierID == 0 {
		return fmt.Errorf("supplier_id is required")
	}
	if av.RawMaterialID == 0 {
		return fmt.Errorf("raw_material_id is required")
	}
	return s.rmRepo.CreateApprovedVendor(ctx, av)
}

func (s *rawMaterialService) GetApprovedVendors(ctx context.Context, rawMaterialID uint) ([]ApprovedVendor, error) {
	return s.rmRepo.GetApprovedVendors(ctx, rawMaterialID)
}

func (s *rawMaterialService) UpdateApprovedVendor(ctx context.Context, av *ApprovedVendor) error {
	return s.rmRepo.UpdateApprovedVendor(ctx, av)
}

// ── Inventory ──

func (s *rawMaterialService) ReceiveRawMaterial(ctx context.Context, inv *RawMaterialInventory, userID uint) error {
	// Generate lot number
	inv.LotNumber = fmt.Sprintf("LOT-%s", strings.ToUpper(crypto.SecureToken(6)))
	inv.OriginalQty = inv.Quantity
	inv.QAStatus = "quarantine" // All new receipts go to quarantine
	inv.IsActive = true

	err := s.rmRepo.CreateRMInventory(ctx, inv)
	if err != nil {
		return err
	}

	// Create transaction log
	txn := RawMaterialTransaction{
		InventoryID:     inv.ID,
		TransactionType: "grn_receipt",
		Quantity:        inv.Quantity,
		BalanceAfter:    inv.Quantity,
		UnitOfMeasure:   inv.UnitOfMeasure,
		ReferenceType:   "grn",
		PerformedByID:   userID,
		Notes:           fmt.Sprintf("Received %s (Lot: %s)", inv.RawMaterial.Name, inv.LotNumber),
		CreatedAt:       time.Now(),
	}
	return s.rmRepo.CreateRMTransaction(ctx, &txn)
}

func (s *rawMaterialService) ApproveQA(ctx context.Context, inventoryID uint, userID uint, notes string) error {
	inv, err := s.rmRepo.GetRMInventory(ctx, inventoryID)
	if err != nil {
		return fmt.Errorf("inventory not found: %w", err)
	}

	if inv.QAStatus != "quarantine" {
		return fmt.Errorf("can only approve inventory in quarantine status (current: %s)", inv.QAStatus)
	}

	now := time.Now()
	inv.QAStatus = "approved"
	inv.QAApprovedByID = &userID
	inv.QAApprovedAt = &now
	inv.QANotes = notes

	if err := s.rmRepo.UpdateRMInventory(ctx, inv); err != nil {
		return err
	}

	// Log the QA release
	txn := RawMaterialTransaction{
		InventoryID:     inventoryID,
		TransactionType: "qa_release",
		Quantity:        0, // No qty change, just status
		BalanceAfter:    inv.Quantity,
		UnitOfMeasure:   inv.UnitOfMeasure,
		ReferenceType:   "qa_sample",
		PerformedByID:   userID,
		Notes:           fmt.Sprintf("QA approved. %s", notes),
		CreatedAt:       now,
	}
	return s.rmRepo.CreateRMTransaction(ctx, &txn)
}

func (s *rawMaterialService) RejectQA(ctx context.Context, inventoryID uint, userID uint, notes string) error {
	inv, err := s.rmRepo.GetRMInventory(ctx, inventoryID)
	if err != nil {
		return fmt.Errorf("inventory not found: %w", err)
	}

	if inv.QAStatus != "quarantine" {
		return fmt.Errorf("can only reject inventory in quarantine status (current: %s)", inv.QAStatus)
	}

	inv.QAStatus = "rejected"
	inv.QANotes = notes

	if err := s.rmRepo.UpdateRMInventory(ctx, inv); err != nil {
		return err
	}

	txn := RawMaterialTransaction{
		InventoryID:     inventoryID,
		TransactionType: "qa_release",
		Quantity:        0,
		BalanceAfter:    inv.Quantity,
		UnitOfMeasure:   inv.UnitOfMeasure,
		ReferenceType:   "qa_sample",
		PerformedByID:   userID,
		Notes:           fmt.Sprintf("QA REJECTED. %s", notes),
		CreatedAt:       time.Now(),
	}
	return s.rmRepo.CreateRMTransaction(ctx, &txn)
}

func (s *rawMaterialService) GetRMInventory(ctx context.Context, warehouseID uint, rawMaterialID uint, qaStatus string) ([]RawMaterialInventory, error) {
	return s.rmRepo.ListRMInventory(ctx, warehouseID, rawMaterialID, qaStatus)
}

func (s *rawMaterialService) GetRMBalance(ctx context.Context, rawMaterialID uint, warehouseID uint) (float64, error) {
	return s.rmRepo.GetRMBalance(ctx, rawMaterialID, warehouseID)
}

func (s *rawMaterialService) GetLowStockRawMaterials(ctx context.Context, warehouseID uint) ([]RawMaterialInventory, error) {
	return s.rmRepo.GetLowStockRawMaterials(ctx, warehouseID)
}

func (s *rawMaterialService) GetRMSummary(ctx context.Context, warehouseID uint) ([]RawMaterialSummary, error) {
	return s.rmRepo.GetRMSummary(ctx, warehouseID)
}

func (s *rawMaterialService) GetRMTransactions(ctx context.Context, inventoryID uint) ([]RawMaterialTransaction, error) {
	return s.rmRepo.GetRMTransactions(ctx, inventoryID)
}

func (s *rawMaterialService) DestroyRawMaterial(ctx context.Context, inventoryID uint, qty float64, reason string, userID uint) error {
	return s.rmRepo.ExecuteTransaction(ctx, func(txCtx context.Context) error {
		inv, err := s.rmRepo.GetRMInventory(txCtx, inventoryID)
		if err != nil {
			return fmt.Errorf("inventory not found: %w", err)
		}

		if qty > inv.Quantity {
			return fmt.Errorf("destruction qty (%.2f) exceeds available (%.2f)", qty, inv.Quantity)
		}

		inv.Quantity -= qty
		if inv.Quantity <= 0 {
			inv.IsActive = false
		}

		if err := s.rmRepo.UpdateRMInventory(txCtx, inv); err != nil {
			return err
		}

		txn := RawMaterialTransaction{
			InventoryID:     inventoryID,
			TransactionType: "destruction",
			Quantity:        -qty,
			BalanceAfter:    inv.Quantity,
			UnitOfMeasure:   inv.UnitOfMeasure,
			ReferenceType:   "destruction_record",
			PerformedByID:   userID,
			Notes:           fmt.Sprintf("Destroyed: %s", reason),
			CreatedAt:       time.Now(),
		}
		return s.rmRepo.CreateRMTransaction(txCtx, &txn)
	})
}

// ── Indents ──

func (s *rawMaterialService) CreateIndent(ctx context.Context, indent *RawMaterialIndent) error {
	indent.IndentNumber = fmt.Sprintf("IND-%s", strings.ToUpper(crypto.SecureToken(6)))
	indent.Status = "pending"
	return s.rmRepo.CreateIndent(ctx, indent)
}

func (s *rawMaterialService) GetIndent(ctx context.Context, id uint) (*RawMaterialIndent, error) {
	return s.rmRepo.GetIndent(ctx, id)
}

func (s *rawMaterialService) ListIndents(ctx context.Context, warehouseID uint, status string) ([]RawMaterialIndent, error) {
	return s.rmRepo.ListIndents(ctx, warehouseID, status)
}

func (s *rawMaterialService) ApproveIndent(ctx context.Context, indentID uint, userID uint) error {
	indent, err := s.rmRepo.GetIndent(ctx, indentID)
	if err != nil {
		return fmt.Errorf("indent not found: %w", err)
	}
	if indent.Status != "pending" {
		return fmt.Errorf("indent is not in pending status (current: %s)", indent.Status)
	}

	now := time.Now()
	indent.Status = "approved"
	indent.ApprovedByID = &userID
	indent.ApprovedAt = &now
	return s.rmRepo.UpdateIndent(ctx, indent)
}

func (s *rawMaterialService) CancelIndent(ctx context.Context, indentID uint) error {
	indent, err := s.rmRepo.GetIndent(ctx, indentID)
	if err != nil {
		return fmt.Errorf("indent not found: %w", err)
	}
	if indent.Status == "received" || indent.Status == "cancelled" {
		return fmt.Errorf("cannot cancel indent in %s status", indent.Status)
	}
	indent.Status = "cancelled"
	return s.rmRepo.UpdateIndent(ctx, indent)
}

// ── RFQ Workflow ──

func (s *rawMaterialService) CreateRFQ(ctx context.Context, rfq *RFQ) error {
	rfq.RFQNumber = fmt.Sprintf("RFQ-%s", strings.ToUpper(crypto.SecureToken(6)))
	rfq.Status = "open"
	return s.rmRepo.CreateRFQ(ctx, rfq)
}

func (s *rawMaterialService) GetRFQ(ctx context.Context, id uint) (*RFQ, error) {
	return s.rmRepo.GetRFQ(ctx, id)
}

func (s *rawMaterialService) ListRFQs(ctx context.Context, status string) ([]RFQ, error) {
	return s.rmRepo.ListRFQs(ctx, status)
}

func (s *rawMaterialService) SubmitQuotation(ctx context.Context, q *VendorQuotation) error {
	// Validate RFQ exists and is open
	rfq, err := s.rmRepo.GetRFQ(ctx, q.RFQID)
	if err != nil {
		return fmt.Errorf("RFQ not found: %w", err)
	}
	if rfq.Status != "open" {
		return fmt.Errorf("RFQ is not open for quotations (status: %s)", rfq.Status)
	}

	// Check deadline
	if time.Now().After(rfq.Deadline) {
		return fmt.Errorf("RFQ deadline has passed")
	}

	now := time.Now()
	q.SubmittedAt = &now
	return s.rmRepo.CreateQuotation(ctx, q)
}

func (s *rawMaterialService) AwardRFQ(ctx context.Context, rfqID uint, quotationID uint) error {
	return s.rmRepo.ExecuteTransaction(ctx, func(txCtx context.Context) error {
		rfq, err := s.rmRepo.GetRFQ(txCtx, rfqID)
		if err != nil {
			return fmt.Errorf("RFQ not found: %w", err)
		}
		if rfq.Status != "open" {
			return fmt.Errorf("RFQ is not open (status: %s)", rfq.Status)
		}

		// Mark the winning quotation
		quotation, err := s.rmRepo.GetQuotation(txCtx, quotationID)
		if err != nil {
			return fmt.Errorf("quotation not found: %w", err)
		}
		if quotation.RFQID != rfqID {
			return fmt.Errorf("quotation does not belong to this RFQ")
		}

		quotation.IsSelected = true
		if err := s.rmRepo.UpdateQuotation(txCtx, quotation); err != nil {
			return err
		}

		// Close the RFQ
		rfq.Status = "awarded"
		return s.rmRepo.UpdateRFQ(txCtx, rfq)
	})
}

// ── Dashboard ──

func (s *rawMaterialService) GetRawMaterialDashboard(ctx context.Context, warehouseID uint) (*RawMaterialDashboard, error) {
	return s.rmRepo.GetRawMaterialDashboard(ctx, warehouseID)
}

package pharmacy

import (
	"context"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"
)

// ═══════════════════════════════════════════════════════════════════════════
// AUDIT SERVICE
//
// Runs 25+ automated checks across the pharmacy admin area and returns a
// structured AuditReport. Designed to run on-demand from the audit UI.
//
// Adding a new check:
//   1. Add a CheckXxx() method below
//   2. Append it to allChecks slice in RunFullAudit
//   3. Set severity + owner + category + clear action
// ═══════════════════════════════════════════════════════════════════════════

type AuditService interface {
	RunFullAudit(ctx context.Context) (*AuditReport, error)
	RunChecksByCategory(ctx context.Context, category AuditCategory) (*AuditReport, error)
	RunChecksByOwner(ctx context.Context, owner AuditOwner) (*AuditReport, error)
	ListOwners() []OwnerInfo
}

type auditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) AuditService {
	return &auditService{db: db}
}

type checkFn func(ctx context.Context) *AuditIssue

// allChecks returns all registered checks in display order.
func (s *auditService) allChecks() []checkFn {
	return []checkFn{
		// Data integrity (foundational)
		s.checkActivePharmacyExists,
		s.checkCentralWarehouseExists,
		s.checkActiveSupplierExists,
		s.checkMedicinesHavePrices,
		s.checkMedicinesHaveBatches,
		s.checkOrphanedInventory,

		// Operations (day-to-day)
		s.checkPendingPrescriptions,
		s.checkOrdersStuckInPacked,
		s.checkUnverifiedRxBlockingOrders,
		s.checkLowStockItems,
		s.checkExpiredStockItems,
		s.checkPendingTransfers,

		// Manufacturing
		s.checkProductionOrdersStuckInQA,
		s.checkBOMsWithoutItems,

		// Procurement
		s.checkOpenPOsAwaitingDelivery,
		s.checkGRNsAwaitingQA,
		s.checkPendingIndents,
		s.checkOpenRFQsPastDeadline,

		// Compliance
		s.checkVendorGMPCertExpiring,
		s.checkColdChainBreaches,
		s.checkRawMaterialsInQuarantine,
		s.checkExpiringRawMaterials,

		// Financial
		s.checkUnsettledSupplierPayments,
		s.checkUnpaidOrders,

		// Network/Config (admin)
		s.checkInactivePharmacyCount,
		s.checkPharmaciesWithoutCity,
	}
}

func (s *auditService) RunFullAudit(ctx context.Context) (*AuditReport, error) {
	return s.runChecks(ctx, s.allChecks())
}

func (s *auditService) RunChecksByCategory(ctx context.Context, category AuditCategory) (*AuditReport, error) {
	report, err := s.RunFullAudit(ctx)
	if err != nil {
		return nil, err
	}
	filtered := []AuditIssue{}
	for _, iss := range report.Issues {
		if iss.Category == category {
			filtered = append(filtered, iss)
		}
	}
	report.Issues = filtered
	report.TotalIssues = len(filtered)
	return s.recomputeAggregates(report), nil
}

func (s *auditService) RunChecksByOwner(ctx context.Context, owner AuditOwner) (*AuditReport, error) {
	report, err := s.RunFullAudit(ctx)
	if err != nil {
		return nil, err
	}
	filtered := []AuditIssue{}
	for _, iss := range report.Issues {
		if iss.Owner == owner {
			filtered = append(filtered, iss)
		}
	}
	report.Issues = filtered
	report.TotalIssues = len(filtered)
	return s.recomputeAggregates(report), nil
}

func (s *auditService) ListOwners() []OwnerInfo {
	return ListOwners()
}

func (s *auditService) runChecks(ctx context.Context, checks []checkFn) (*AuditReport, error) {
	start := time.Now()
	report := &AuditReport{
		GeneratedAt: start,
		TotalChecks: len(checks),
		BySeverity:  map[AuditSeverity]int{},
		ByOwner:     map[AuditOwner]int{},
		ByCategory:  map[AuditCategory]int{},
		Issues:      []AuditIssue{},
	}

	for _, check := range checks {
		issue := check(ctx)
		if issue != nil {
			issue.DetectedAt = time.Now()
			report.Issues = append(report.Issues, *issue)
			report.BySeverity[issue.Severity]++
			report.ByOwner[issue.Owner]++
			report.ByCategory[issue.Category]++
		}
	}
	report.TotalIssues = len(report.Issues)
	report.DurationMS = time.Since(start).Milliseconds()

	// Sort issues by severity (critical first)
	sevOrder := map[AuditSeverity]int{
		SeverityCritical: 0,
		SeverityHigh:     1,
		SeverityMedium:   2,
		SeverityLow:      3,
		SeverityInfo:     4,
	}
	sort.SliceStable(report.Issues, func(i, j int) bool {
		return sevOrder[report.Issues[i].Severity] < sevOrder[report.Issues[j].Severity]
	})

	// Compute health score
	report.HealthScore = computeHealthScore(report)
	report.Status = healthStatus(report.HealthScore, report.BySeverity[SeverityCritical])

	return report, nil
}

func (s *auditService) recomputeAggregates(report *AuditReport) *AuditReport {
	report.BySeverity = map[AuditSeverity]int{}
	report.ByOwner = map[AuditOwner]int{}
	report.ByCategory = map[AuditCategory]int{}
	for _, iss := range report.Issues {
		report.BySeverity[iss.Severity]++
		report.ByOwner[iss.Owner]++
		report.ByCategory[iss.Category]++
	}
	report.HealthScore = computeHealthScore(report)
	report.Status = healthStatus(report.HealthScore, report.BySeverity[SeverityCritical])
	return report
}

func computeHealthScore(report *AuditReport) int {
	// Start at 100, subtract weighted penalties
	score := 100
	score -= report.BySeverity[SeverityCritical] * 20
	score -= report.BySeverity[SeverityHigh] * 8
	score -= report.BySeverity[SeverityMedium] * 3
	score -= report.BySeverity[SeverityLow] * 1
	if score < 0 {
		score = 0
	}
	return score
}

func healthStatus(score int, criticalCount int) string {
	if criticalCount > 0 {
		return "critical"
	}
	if score < 70 {
		return "degraded"
	}
	if score < 90 {
		return "warning"
	}
	return "healthy"
}

// ─────────────────────────────────────────────────────────────────────────
// DATA INTEGRITY CHECKS
// ─────────────────────────────────────────────────────────────────────────

func (s *auditService) checkActivePharmacyExists(ctx context.Context) *AuditIssue {
	var count int64
	s.db.WithContext(ctx).Model(&Pharmacy{}).Where("is_active = ?", true).Count(&count)
	if count == 0 {
		return &AuditIssue{
			ID: "no-active-pharmacy", Severity: SeverityCritical,
			Owner: OwnerCentralAdmin, Category: CategoryDataIntegrity,
			Title:       "No active pharmacy in the system",
			Description: "There are zero pharmacies marked active. The pharmacy module is non-functional without at least one.",
			Action:      "Create at least one pharmacy via /admin/pharmacy/units",
			Impact:      "Customers cannot place orders. Staff cannot log in to pharmacy-scoped routes.",
			FixURL:      "/admin/pharmacy/units",
		}
	}
	return nil
}

func (s *auditService) checkCentralWarehouseExists(ctx context.Context) *AuditIssue {
	var count int64
	s.db.WithContext(ctx).Model(&Pharmacy{}).Where("is_central_warehouse = ? AND is_active = ?", true, true).Count(&count)
	if count == 0 {
		return &AuditIssue{
			ID: "no-central-warehouse", Severity: SeverityHigh,
			Owner: OwnerCentralAdmin, Category: CategoryDataIntegrity,
			Title:       "No central warehouse configured",
			Description: "Without a central warehouse, auto-restock and multi-city distribution cannot run.",
			Action:      "Mark one pharmacy as is_central_warehouse=true",
			Impact:      "Auto-restock fails. POs cannot be auto-created.",
			FixURL:      "/admin/pharmacy/units",
		}
	}
	return nil
}

func (s *auditService) checkActiveSupplierExists(ctx context.Context) *AuditIssue {
	var count int64
	s.db.WithContext(ctx).Model(&Supplier{}).Where("is_active = ?", true).Count(&count)
	if count == 0 {
		return &AuditIssue{
			ID: "no-active-supplier", Severity: SeverityHigh,
			Owner: OwnerPharmacyManager, Category: CategoryDataIntegrity,
			Title:       "No active supplier registered",
			Description: "Without active suppliers, purchase orders cannot be created.",
			Action:      "Add at least one supplier from the Suppliers tab",
			Impact:      "Cannot create POs. Auto-restock fails silently.",
			FixURL:      "/pharmacy/admin/operations?tab=vendors",
		}
	}
	return nil
}

func (s *auditService) checkMedicinesHavePrices(ctx context.Context) *AuditIssue {
	type Row struct {
		ID   uint
		Name string
	}
	var rows []Row
	// `prices` is a jsonb column in production (text in SQLite tests). Comparing
	// jsonb to '' as text raises SQLSTATE 22P02. Detect dialect and use the
	// proper predicate for each.
	if s.db.Dialector.Name() == "postgres" {
		s.db.WithContext(ctx).Raw(`
			SELECT id, name FROM medicine
			WHERE is_active = true
			  AND (prices IS NULL OR prices::text = '{}' OR prices::text = 'null')
			LIMIT 50
		`).Scan(&rows)
	} else {
		s.db.WithContext(ctx).Raw(`
			SELECT id, name FROM medicine
			WHERE is_active = true
			  AND (prices IS NULL OR prices = '' OR prices = '{}')
			LIMIT 50
		`).Scan(&rows)
	}

	if len(rows) == 0 {
		return nil
	}
	items := make([]AuditItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, AuditItem{Type: "medicine", ID: r.ID, Name: r.Name})
	}
	return &AuditIssue{
		ID: "medicines-without-prices", Severity: SeverityHigh,
		Owner: OwnerPharmacyManager, Category: CategoryDataIntegrity,
		Title:       fmt.Sprintf("%d medicine(s) have no price set", len(rows)),
		Description: "Active medicines without a price cannot be sold or auto-reordered.",
		Action:      "Set INR price for each medicine via the catalog UI",
		Impact:      "Customers see no price. Auto-restock skips these. Lost sales.",
		Count:       len(rows),
		Items:       items,
		FixURL:      "/pharmacy/catalog",
	}
}

func (s *auditService) checkMedicinesHaveBatches(ctx context.Context) *AuditIssue {
	type Row struct {
		ID   uint
		Name string
	}
	var rows []Row
	s.db.WithContext(ctx).Raw(`
		SELECT m.id, m.name FROM medicine m
		WHERE m.is_active = true
		  AND NOT EXISTS (SELECT 1 FROM pharmacyinventory pi WHERE pi.medicine_id = m.id AND pi.is_active = true)
		LIMIT 50
	`).Scan(&rows)

	if len(rows) == 0 {
		return nil
	}
	items := make([]AuditItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, AuditItem{Type: "medicine", ID: r.ID, Name: r.Name})
	}
	return &AuditIssue{
		ID: "medicines-without-inventory", Severity: SeverityMedium,
		Owner: OwnerPharmacyManager, Category: CategoryDataIntegrity,
		Title:       fmt.Sprintf("%d medicine(s) have no inventory anywhere", len(rows)),
		Description: "Catalog lists medicines that have no stock in any pharmacy.",
		Action:      "Create POs to procure stock, or deactivate the medicine if discontinued",
		Impact:      "Customers see 'out of stock' on listed products. Confusing UX.",
		Count:       len(rows),
		Items:       items,
		FixURL:      "/pharmacy/admin/workflow?tab=procurement",
	}
}

func (s *auditService) checkOrphanedInventory(ctx context.Context) *AuditIssue {
	var count int64
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM pharmacyinventory pi
		WHERE pi.is_active = true
		  AND (NOT EXISTS (SELECT 1 FROM medicine m WHERE m.id = pi.medicine_id)
		    OR NOT EXISTS (SELECT 1 FROM pharmacy ph WHERE ph.id = pi.pharmacy_id))
	`).Scan(&count)

	if count == 0 {
		return nil
	}
	return &AuditIssue{
		ID: "orphaned-inventory", Severity: SeverityMedium,
		Owner: OwnerBackendDev, Category: CategoryDataIntegrity,
		Title:       fmt.Sprintf("%d orphaned inventory record(s)", count),
		Description: "Inventory rows reference deleted medicines or pharmacies.",
		Action:      "Run cleanup migration to deactivate orphans",
		Impact:      "Reports show ghost stock. Confuses inventory accounting.",
		Count:       int(count),
	}
}

// ─────────────────────────────────────────────────────────────────────────
// OPERATIONS CHECKS
// ─────────────────────────────────────────────────────────────────────────

func (s *auditService) checkPendingPrescriptions(ctx context.Context) *AuditIssue {
	type Row struct {
		ID        uint
		UserID    uint
		CreatedAt time.Time
	}
	var rows []Row
	threshold := time.Now().Add(-24 * time.Hour)
	s.db.WithContext(ctx).Raw(`
		SELECT id, user_id, created_at FROM prescriptions
		WHERE status IN ('uploaded', 'under_review') AND created_at < ?
		ORDER BY created_at ASC LIMIT 50
	`, threshold).Scan(&rows)

	if len(rows) == 0 {
		return nil
	}
	items := make([]AuditItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, AuditItem{
			Type: "prescription", ID: r.ID,
			Name:        fmt.Sprintf("Rx #%d (user %d)", r.ID, r.UserID),
			Description: fmt.Sprintf("Uploaded %s", r.CreatedAt.Format("2006-01-02 15:04")),
		})
	}
	severity := SeverityHigh
	if len(rows) > 10 {
		severity = SeverityCritical
	}
	return &AuditIssue{
		ID: "rx-queue-backlog", Severity: severity,
		Owner: OwnerPharmacist, Category: CategoryOperations,
		Title:       fmt.Sprintf("%d prescription(s) pending verification over 24h", len(rows)),
		Description: "Patients are waiting for prescription verification. Backlog hurts conversion.",
		Action:      "Open the Rx Verification queue and process oldest first",
		Impact:      "Customer churn. Rx-only orders cannot proceed.",
		Count:       len(rows),
		Items:       items,
		FixURL:      "/pharmacy/work?tab=prescriptions",
	}
}

func (s *auditService) checkOrdersStuckInPacked(ctx context.Context) *AuditIssue {
	var count int64
	threshold := time.Now().Add(-48 * time.Hour)
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM pharmacy_orders
		WHERE status = 'packed' AND updated_at < ?
	`, threshold).Scan(&count)

	if count == 0 {
		return nil
	}
	return &AuditIssue{
		ID: "orders-stuck-packed", Severity: SeverityHigh,
		Owner: OwnerPharmacist, Category: CategoryOperations,
		Title:       fmt.Sprintf("%d order(s) packed but not shipped (>48h)", count),
		Description: "Orders sitting in 'packed' state for 2+ days indicate a dispatch breakdown.",
		Action:      "Review dispense queue, assign to courier, or escalate to manager",
		Impact:      "Customer complaints. SLA breaches. Potential chargebacks.",
		Count:       int(count),
		FixURL:      "/pharmacy/work?tab=dispense",
	}
}

func (s *auditService) checkUnverifiedRxBlockingOrders(ctx context.Context) *AuditIssue {
	var count int64
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM pharmacy_orders WHERE status = 'pending_rx_verification'
	`).Scan(&count)

	if count == 0 {
		return nil
	}
	return &AuditIssue{
		ID: "orders-blocked-on-rx", Severity: SeverityMedium,
		Owner: OwnerPharmacist, Category: CategoryOperations,
		Title:       fmt.Sprintf("%d order(s) blocked awaiting Rx verification", count),
		Description: "Orders linked to prescriptions that haven't been verified yet.",
		Action:      "Verify Rx in the prescription queue → orders auto-unblock",
		Impact:      "Customer cannot pay. Order may be abandoned.",
		Count:       int(count),
		FixURL:      "/pharmacy/work?tab=prescriptions",
	}
}

func (s *auditService) checkLowStockItems(ctx context.Context) *AuditIssue {
	var count int64
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM pharmacyinventory
		WHERE is_active = true AND stock_quantity <= reorder_level
	`).Scan(&count)

	if count == 0 {
		return nil
	}
	severity := SeverityMedium
	if count > 20 {
		severity = SeverityHigh
	}
	return &AuditIssue{
		ID: "low-stock-items", Severity: severity,
		Owner: OwnerPharmacyManager, Category: CategoryOperations,
		Title:       fmt.Sprintf("%d SKU(s) at or below reorder level", count),
		Description: "Items needing reorder to prevent stockout.",
		Action:      "Run auto-restock or create POs manually",
		Impact:      "Future orders will fail. Customer disappointment.",
		Count:       int(count),
		FixURL:      "/pharmacy/stock?tab=alerts",
	}
}

func (s *auditService) checkExpiredStockItems(ctx context.Context) *AuditIssue {
	var count int64
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM pharmacyinventory
		WHERE is_active = true AND stock_quantity > 0 AND expiry_date < ?
	`, time.Now()).Scan(&count)

	if count == 0 {
		return nil
	}
	return &AuditIssue{
		ID: "expired-stock", Severity: SeverityCritical,
		Owner: OwnerComplianceOfficer, Category: CategoryCompliance,
		Title:       fmt.Sprintf("%d batch(es) past expiry but still active", count),
		Description: "Expired medicines must be quarantined immediately to prevent dispensing to patients.",
		Action:      "Dispose expired stock via /pharmacy/stock?tab=expiry — DO NOT dispense",
		Impact:      "PATIENT SAFETY RISK. Regulatory violation. Liability exposure.",
		Count:       int(count),
		FixURL:      "/pharmacy/stock?tab=expiry",
	}
}

func (s *auditService) checkPendingTransfers(ctx context.Context) *AuditIssue {
	var count int64
	threshold := time.Now().Add(-72 * time.Hour)
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM pharmacy_inventory_transfers
		WHERE status = 'requested' AND created_at < ?
	`, threshold).Scan(&count)

	if count == 0 {
		return nil
	}
	return &AuditIssue{
		ID: "stale-transfers", Severity: SeverityMedium,
		Owner: OwnerPharmacyManager, Category: CategoryOperations,
		Title:       fmt.Sprintf("%d inter-hub transfer(s) unfulfilled for 3+ days", count),
		Description: "Transfers requested but not processed indicate logistics issues.",
		Action:      "Process or cancel pending transfers",
		Impact:      "Local stockouts. Source hub holds inventory that's needed elsewhere.",
		Count:       int(count),
		FixURL:      "/pharmacy/stock?tab=transfers",
	}
}

// ─────────────────────────────────────────────────────────────────────────
// MANUFACTURING CHECKS
// ─────────────────────────────────────────────────────────────────────────

func (s *auditService) checkProductionOrdersStuckInQA(ctx context.Context) *AuditIssue {
	var count int64
	threshold := time.Now().Add(-72 * time.Hour)
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM production_orders
		WHERE status = 'qa_pending' AND updated_at < ?
	`, threshold).Scan(&count)

	if count == 0 {
		return nil
	}
	return &AuditIssue{
		ID: "production-stuck-qa", Severity: SeverityHigh,
		Owner: OwnerComplianceOfficer, Category: CategoryOperations,
		Title:       fmt.Sprintf("%d production batch(es) awaiting QA for 3+ days", count),
		Description: "Finished batches blocked at QA gate delay finished goods release.",
		Action:      "Complete QA testing and pass/fail decision",
		Impact:      "Finished goods inventory not replenished. Wasted manufacturing time.",
		Count:       int(count),
		FixURL:      "/pharmacy/admin/workflow?tab=manufacturing",
	}
}

func (s *auditService) checkBOMsWithoutItems(ctx context.Context) *AuditIssue {
	var count int64
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM bill_of_materials b
		WHERE b.is_active = true
		  AND NOT EXISTS (SELECT 1 FROM bom_items i WHERE i.bom_id = b.id)
	`).Scan(&count)

	if count == 0 {
		return nil
	}
	return &AuditIssue{
		ID: "empty-bom", Severity: SeverityMedium,
		Owner: OwnerPharmacyManager, Category: CategoryDataIntegrity,
		Title:       fmt.Sprintf("%d Bill of Materials with no ingredients", count),
		Description: "Active BOMs lacking items will fail availability checks.",
		Action:      "Open BOM editor and add ingredients",
		Impact:      "Cannot start production orders against these BOMs.",
		Count:       int(count),
		FixURL:      "/pharmacy/admin/operations?tab=bom",
	}
}

// ─────────────────────────────────────────────────────────────────────────
// PROCUREMENT CHECKS
// ─────────────────────────────────────────────────────────────────────────

func (s *auditService) checkOpenPOsAwaitingDelivery(ctx context.Context) *AuditIssue {
	var count int64
	threshold := time.Now().AddDate(0, 0, -3) // 3 days past expected
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM pharmacy_purchase_orders
		WHERE status = 'ordered' AND expected_date < ?
	`, threshold).Scan(&count)

	if count == 0 {
		return nil
	}
	return &AuditIssue{
		ID: "po-overdue", Severity: SeverityHigh,
		Owner: OwnerPharmacyManager, Category: CategoryOperations,
		Title:       fmt.Sprintf("%d purchase order(s) past expected delivery date", count),
		Description: "Suppliers have not delivered on time. May need follow-up or escalation.",
		Action:      "Contact suppliers, update expected dates, or cancel/reorder",
		Impact:      "Stockouts. Customer disappointment. Supplier accountability issues.",
		Count:       int(count),
		FixURL:      "/pharmacy/admin/workflow?tab=procurement",
	}
}

func (s *auditService) checkGRNsAwaitingQA(ctx context.Context) *AuditIssue {
	var count int64
	threshold := time.Now().Add(-24 * time.Hour)
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM goods_receipt_notes
		WHERE status IN ('pending_inspection', 'qa_in_progress') AND received_at < ?
	`, threshold).Scan(&count)

	if count == 0 {
		return nil
	}
	return &AuditIssue{
		ID: "grn-pending-qa", Severity: SeverityMedium,
		Owner: OwnerComplianceOfficer, Category: CategoryCompliance,
		Title:       fmt.Sprintf("%d GRN(s) pending QA over 24h", count),
		Description: "Received goods awaiting quality inspection cannot be added to sellable inventory.",
		Action:      "Complete QA inspection per item",
		Impact:      "Stock held but unsellable. Reduces effective inventory.",
		Count:       int(count),
		FixURL:      "/pharmacy/stock?tab=grn",
	}
}

func (s *auditService) checkPendingIndents(ctx context.Context) *AuditIssue {
	var count int64
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM raw_material_indents WHERE status = 'pending'
	`).Scan(&count)

	if count == 0 {
		return nil
	}
	return &AuditIssue{
		ID: "indents-pending", Severity: SeverityLow,
		Owner: OwnerPharmacyManager, Category: CategoryOperations,
		Title:       fmt.Sprintf("%d raw material indent(s) awaiting approval", count),
		Description: "Manufacturing planners have requested raw materials but no procurement action.",
		Action:      "Review and approve indents, then create RFQs",
		Impact:      "Future production runs may be blocked due to missing raw materials.",
		Count:       int(count),
		FixURL:      "/pharmacy/admin/workflow?tab=raw-materials",
	}
}

func (s *auditService) checkOpenRFQsPastDeadline(ctx context.Context) *AuditIssue {
	var count int64
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM rfqs WHERE status = 'open' AND deadline < ?
	`, time.Now()).Scan(&count)

	if count == 0 {
		return nil
	}
	return &AuditIssue{
		ID: "rfq-past-deadline", Severity: SeverityMedium,
		Owner: OwnerPharmacyManager, Category: CategoryOperations,
		Title:       fmt.Sprintf("%d RFQ(s) past deadline but still open", count),
		Description: "RFQs that should be closed and awarded are blocking procurement.",
		Action:      "Award to best quotation or cancel and reissue",
		Impact:      "Procurement delays. Suppliers waiting for decision.",
		Count:       int(count),
		FixURL:      "/pharmacy/admin/workflow?tab=raw-materials",
	}
}

// ─────────────────────────────────────────────────────────────────────────
// COMPLIANCE CHECKS
// ─────────────────────────────────────────────────────────────────────────

func (s *auditService) checkVendorGMPCertExpiring(ctx context.Context) *AuditIssue {
	type Row struct {
		ID         uint
		SupplierID uint
	}
	var rows []Row
	threshold := time.Now().AddDate(0, 2, 0) // 60 days
	s.db.WithContext(ctx).Raw(`
		SELECT id, supplier_id FROM approved_vendors
		WHERE is_active = true AND gmp_cert_expiry IS NOT NULL AND gmp_cert_expiry < ?
		LIMIT 50
	`, threshold).Scan(&rows)

	if len(rows) == 0 {
		return nil
	}
	items := make([]AuditItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, AuditItem{
			Type: "vendor", ID: r.ID,
			Name: fmt.Sprintf("Vendor #%d (Supplier %d)", r.ID, r.SupplierID),
		})
	}
	return &AuditIssue{
		ID: "vendor-gmp-expiring", Severity: SeverityHigh,
		Owner: OwnerComplianceOfficer, Category: CategoryCompliance,
		Title:       fmt.Sprintf("%d vendor GMP certificate(s) expiring within 60 days", len(rows)),
		Description: "Vendors without valid GMP certificates cannot be used for procurement (regulatory).",
		Action:      "Request renewed GMP certificate from each vendor and upload",
		Impact:      "Procurement from these vendors becomes non-compliant. POs may be voided.",
		Count:       len(rows),
		Items:       items,
		FixURL:      "/pharmacy/admin/operations?tab=vendors",
	}
}

func (s *auditService) checkColdChainBreaches(ctx context.Context) *AuditIssue {
	var count int64
	threshold := time.Now().Add(-7 * 24 * time.Hour)
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM cold_chain_metrics
		WHERE recorded_at > ? AND (temperature > 25 OR temperature < 2 OR humidity > 60)
	`, threshold).Scan(&count)

	if count == 0 {
		return nil
	}
	severity := SeverityHigh
	if count > 5 {
		severity = SeverityCritical
	}
	return &AuditIssue{
		ID: "cold-chain-breaches", Severity: severity,
		Owner: OwnerComplianceOfficer, Category: CategoryCompliance,
		Title:       fmt.Sprintf("%d cold chain breach(es) in the last 7 days", count),
		Description: "Temperature or humidity readings outside safe range detected.",
		Action:      "Investigate breaches, quarantine affected batches, fix HVAC",
		Impact:      "Cold-chain medicines may be unsafe. Regulatory exposure. Quality issues.",
		Count:       int(count),
		FixURL:      "/admin/pharmacy?tab=cold-chain",
	}
}

func (s *auditService) checkRawMaterialsInQuarantine(ctx context.Context) *AuditIssue {
	var count int64
	threshold := time.Now().Add(-7 * 24 * time.Hour)
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM raw_material_inventory
		WHERE is_active = true AND qa_status = 'quarantine' AND created_at < ?
	`, threshold).Scan(&count)

	if count == 0 {
		return nil
	}
	return &AuditIssue{
		ID: "rm-stuck-quarantine", Severity: SeverityMedium,
		Owner: OwnerComplianceOfficer, Category: CategoryCompliance,
		Title:       fmt.Sprintf("%d raw material lot(s) in quarantine for over 7 days", count),
		Description: "Lots awaiting QA testing block manufacturing.",
		Action:      "Complete QA testing or reject and dispose",
		Impact:      "Cannot use these materials for production. Capital tied up.",
		Count:       int(count),
		FixURL:      "/pharmacy/admin/workflow?tab=raw-materials",
	}
}

func (s *auditService) checkExpiringRawMaterials(ctx context.Context) *AuditIssue {
	var count int64
	threshold := time.Now().AddDate(0, 3, 0) // 90 days
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM raw_material_inventory
		WHERE is_active = true AND expiry_date < ?
	`, threshold).Scan(&count)

	if count == 0 {
		return nil
	}
	return &AuditIssue{
		ID: "rm-expiring", Severity: SeverityLow,
		Owner: OwnerComplianceOfficer, Category: CategoryCompliance,
		Title:       fmt.Sprintf("%d raw material lot(s) expiring within 90 days", count),
		Description: "Raw materials approaching expiry need to be used or disposed.",
		Action:      "Prioritize in upcoming production runs, or dispose if unusable",
		Impact:      "Waste of expensive raw materials if not used in time.",
		Count:       int(count),
		FixURL:      "/pharmacy/admin/workflow?tab=raw-materials",
	}
}

// ─────────────────────────────────────────────────────────────────────────
// FINANCIAL CHECKS
// ─────────────────────────────────────────────────────────────────────────

func (s *auditService) checkUnsettledSupplierPayments(ctx context.Context) *AuditIssue {
	var count int64
	threshold := time.Now().AddDate(0, 0, -30)
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM pharmacy_purchase_orders
		WHERE status = 'received' AND received_date < ?
	`, threshold).Scan(&count)

	if count == 0 {
		return nil
	}
	return &AuditIssue{
		ID: "unsettled-supplier-payments", Severity: SeverityMedium,
		Owner: OwnerFinanceManager, Category: CategoryFinancial,
		Title:       fmt.Sprintf("%d received PO(s) older than 30 days unsettled", count),
		Description: "Suppliers received goods but haven't been paid.",
		Action:      "Reconcile invoices and settle payments via finance module",
		Impact:      "Supplier relationship strain. May affect future credit terms.",
		Count:       int(count),
	}
}

func (s *auditService) checkUnpaidOrders(ctx context.Context) *AuditIssue {
	var count int64
	threshold := time.Now().Add(-7 * 24 * time.Hour)
	s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM pharmacy_orders
		WHERE payment_status = 'pending' AND created_at < ?
	`, threshold).Scan(&count)

	if count == 0 {
		return nil
	}
	return &AuditIssue{
		ID: "unpaid-orders", Severity: SeverityLow,
		Owner: OwnerFinanceManager, Category: CategoryFinancial,
		Title:       fmt.Sprintf("%d order(s) unpaid for over 7 days", count),
		Description: "Orders awaiting payment for a week — likely abandoned.",
		Action:      "Send payment reminder OR cancel and restore inventory",
		Impact:      "Reserved stock unavailable to other customers.",
		Count:       int(count),
	}
}

// ─────────────────────────────────────────────────────────────────────────
// ADMIN/CONFIG CHECKS
// ─────────────────────────────────────────────────────────────────────────

func (s *auditService) checkInactivePharmacyCount(ctx context.Context) *AuditIssue {
	var active, inactive int64
	s.db.WithContext(ctx).Model(&Pharmacy{}).Where("is_active = ?", true).Count(&active)
	s.db.WithContext(ctx).Model(&Pharmacy{}).Where("is_active = ?", false).Count(&inactive)

	if inactive == 0 || active == 0 {
		return nil
	}
	if inactive > active {
		return &AuditIssue{
			ID: "many-inactive-pharmacies", Severity: SeverityInfo,
			Owner: OwnerCentralAdmin, Category: CategoryDataIntegrity,
			Title:       fmt.Sprintf("More inactive (%d) than active (%d) pharmacies", inactive, active),
			Description: "Many pharmacies are marked inactive. Consider archiving if no longer needed.",
			Action:      "Review inactive pharmacies and archive obsolete records",
			Impact:      "Dropdown lists become cluttered. No operational impact.",
			Count:       int(inactive),
			FixURL:      "/admin/pharmacy/units",
		}
	}
	return nil
}

func (s *auditService) checkPharmaciesWithoutCity(ctx context.Context) *AuditIssue {
	type Row struct {
		ID   uint
		Name string
	}
	var rows []Row
	s.db.WithContext(ctx).Raw(`
		SELECT id, name FROM pharmacy
		WHERE is_active = true AND (city IS NULL OR city = '')
		LIMIT 50
	`).Scan(&rows)

	if len(rows) == 0 {
		return nil
	}
	items := make([]AuditItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, AuditItem{Type: "pharmacy", ID: r.ID, Name: r.Name})
	}
	return &AuditIssue{
		ID: "pharmacy-missing-city", Severity: SeverityMedium,
		Owner: OwnerCentralAdmin, Category: CategoryDataIntegrity,
		Title:       fmt.Sprintf("%d active pharmacy/pharmacies missing city", len(rows)),
		Description: "Multi-city dashboards exclude pharmacies without a city assignment.",
		Action:      "Set city for each pharmacy via admin panel",
		Impact:      "Pharmacies invisible in city-level reports and workflow dashboards.",
		Count:       len(rows),
		Items:       items,
		FixURL:      "/admin/pharmacy/units",
	}
}

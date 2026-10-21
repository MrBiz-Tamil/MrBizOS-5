package lab

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Repository defines all database operations for the lab domain.
type Repository interface {
	// ── Catalog ──────────────────────────────────────────────────────────────
	GetLabTestByID(ctx context.Context, id uint) (*LabTest, error)
	ListLabTests(ctx context.Context, category, concern, q string) ([]LabTest, error)
	ListPackages(ctx context.Context) ([]LabTest, error)
	GetHealthConcerns(ctx context.Context) ([]map[string]interface{}, error)

	// ── Partner network ───────────────────────────────────────────────────────
	FindLabCenters(ctx context.Context, pincode, district string, testID uint) ([]labCenterResult, error)
	GetLabCenterByID(ctx context.Context, id uint) (*LabCenter, error)
	GetCenterPricing(ctx context.Context, centerID, testID uint) (*LabCenterPricing, error)

	// ── Patient bookings ──────────────────────────────────────────────────────
	CreateBooking(ctx context.Context, b *LabBooking) error
	GetBookingByID(ctx context.Context, id uint) (*LabBooking, error)
	GetBookingsByPatient(ctx context.Context, patientID uint, status string) ([]LabBooking, error)
	UpdateBookingStatus(ctx context.Context, id uint, status string) error
	MarkBookingPaid(ctx context.Context, id uint, paymentRef string) error
	CancelBooking(ctx context.Context, id uint, reason string) error

	// ── Lab portal (technician) ───────────────────────────────────────────────
	GetPortalBookings(ctx context.Context, labCenterID uint, status string) ([]LabBooking, error)
	MarkSampleCollected(ctx context.Context, id uint) error
	SubmitBookingResult(ctx context.Context, id uint, fileURL, aiSummary, aiRec string) error
	VerifyBookingReport(ctx context.Context, id uint) error

	// ── Hospital internal (kept) ──────────────────────────────────────────────
	CreateLabRequest(ctx context.Context, r *LabRequest) error
	GetLabRequestByID(ctx context.Context, id uint) (*LabRequest, error)
	GetLabRequestsByPatient(ctx context.Context, patientID uint) ([]LabRequest, error)
	UpdateLabRequestStatus(ctx context.Context, id uint, status string) error
	CreateLabResult(ctx context.Context, res *LabResult) error
	GetLabResultsByRequest(ctx context.Context, requestID uint) ([]LabResult, error)
	RecordSpecimenCollection(ctx context.Context, s *Specimen) error
	GetSpecimensByRequest(ctx context.Context, requestID uint) ([]Specimen, error)

	// ── Seed helpers ──────────────────────────────────────────────────────────
	CountTests(ctx context.Context) (int64, error)
	BulkCreatePartners(ctx context.Context, partners []LabPartner) error
	BulkCreateCenters(ctx context.Context, centers []LabCenter) error
	BulkCreatePricing(ctx context.Context, rows []LabCenterPricing) error
	BulkCreateTests(ctx context.Context, tests []LabTest) error
	BulkCreatePackageItems(ctx context.Context, items []LabPackageItem) error

	AutoMigrate() error
}

// labCenterResult enriches a LabCenter with per-test pricing for API responses.
type labCenterResult struct {
	LabCenter
	PricePaise            int64 `json:"price_paise"`
	HomePickupChargePaise int64 `json:"home_pickup_charge_paise"`
	TurnaroundHours       int   `json:"turnaround_hours"`
}

type defaultRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &defaultRepository{db: db}
}

func (r *defaultRepository) AutoMigrate() error {
	return r.db.AutoMigrate(
		&LabTest{},
		&LabPackageItem{},
		&LabPartner{},
		&LabCenter{},
		&LabCenterPricing{},
		&LabBooking{},
		&LabRequest{},
		&LabResult{},
		&Specimen{},
	)
}

// ── Catalog ───────────────────────────────────────────────────────────────────

func (r *defaultRepository) GetLabTestByID(ctx context.Context, id uint) (*LabTest, error) {
	var t LabTest
	err := r.db.WithContext(ctx).First(&t, id).Error
	return &t, err
}

func (r *defaultRepository) ListLabTests(ctx context.Context, category, concern, q string) ([]LabTest, error) {
	var tests []LabTest
	qry := r.db.WithContext(ctx).Where("is_active = ? AND is_package = ?", true, false)
	if category != "" {
		qry = qry.Where("LOWER(category) = LOWER(?)", category)
	}
	if concern != "" {
		qry = qry.Where("LOWER(health_concern) = LOWER(?)", concern)
	}
	if q != "" {
		qry = qry.Where("LOWER(name) LIKE ?", "%"+q+"%")
	}
	err := qry.Order("is_popular DESC, name ASC").Find(&tests).Error
	return tests, err
}

func (r *defaultRepository) ListPackages(ctx context.Context) ([]LabTest, error) {
	var pkgs []LabTest
	err := r.db.WithContext(ctx).
		Where("is_active = ? AND is_package = ?", true, true).
		Order("is_popular DESC, name ASC").
		Find(&pkgs).Error
	return pkgs, err
}

func (r *defaultRepository) GetHealthConcerns(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := r.db.WithContext(ctx).
		Model(&LabTest{}).
		Select("health_concern, COUNT(*) as count").
		Where("is_active = ? AND is_package = ? AND health_concern != ?", true, false, "").
		Group("health_concern").
		Order("count DESC").
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []map[string]interface{}
	for rows.Next() {
		var concern string
		var count int64
		if err := rows.Scan(&concern, &count); err != nil {
			continue
		}
		out = append(out, map[string]interface{}{"concern": concern, "count": count})
	}
	return out, nil
}

// ── Partner network ───────────────────────────────────────────────────────────

func (r *defaultRepository) FindLabCenters(ctx context.Context, pincode, district string, testID uint) ([]labCenterResult, error) {
	var results []labCenterResult

	qry := r.db.WithContext(ctx).
		Table("lab_centers lc").
		Select(`lc.*, p.name AS partner_name, p.logo_url AS partner_logo,
			p.is_nabl, p.rating AS partner_rating,
			COALESCE(lcp.price_paise, 0) AS price_paise,
			COALESCE(lcp.home_pickup_charge_paise, 0) AS home_pickup_charge_paise,
			COALESCE(lcp.turnaround_hours, 24) AS turnaround_hours`).
		Joins("LEFT JOIN lab_partners p ON p.id = lc.partner_id").
		Where("lc.is_active = ?", true)

	if pincode != "" {
		qry = qry.Where("lc.pincode = ?", pincode)
	}
	if district != "" {
		qry = qry.Where("LOWER(lc.district) = LOWER(?)", district)
	}
	if testID > 0 {
		qry = qry.
			Joins("LEFT JOIN lab_center_pricing lcp ON lcp.lab_center_id = lc.id AND lcp.lab_test_id = ? AND lcp.is_available = ?", testID, true).
			Where("lcp.id IS NOT NULL")
	} else {
		qry = qry.Joins("LEFT JOIN lab_center_pricing lcp ON lcp.lab_center_id = lc.id AND lcp.is_available = ?", true)
	}

	err := qry.Order("partner_rating DESC").Scan(&results).Error
	return results, err
}

func (r *defaultRepository) GetLabCenterByID(ctx context.Context, id uint) (*LabCenter, error) {
	var c LabCenter
	err := r.db.WithContext(ctx).Preload("Partner").First(&c, id).Error
	return &c, err
}

// GetCenterPricing returns the server-authoritative price for a test at a center.
func (r *defaultRepository) GetCenterPricing(ctx context.Context, centerID, testID uint) (*LabCenterPricing, error) {
	var p LabCenterPricing
	err := r.db.WithContext(ctx).
		Where("lab_center_id = ? AND lab_test_id = ? AND is_available = ?", centerID, testID, true).
		First(&p).Error
	return &p, err
}

// ── Patient bookings ──────────────────────────────────────────────────────────

func (r *defaultRepository) CreateBooking(ctx context.Context, b *LabBooking) error {
	return r.db.WithContext(ctx).Create(b).Error
}

func (r *defaultRepository) GetBookingByID(ctx context.Context, id uint) (*LabBooking, error) {
	var b LabBooking
	err := r.db.WithContext(ctx).
		Preload("LabCenter").
		Preload("LabCenter.Partner").
		Preload("LabTest").
		First(&b, id).Error
	return &b, err
}

func (r *defaultRepository) GetBookingsByPatient(ctx context.Context, patientID uint, status string) ([]LabBooking, error) {
	var bookings []LabBooking
	qry := r.db.WithContext(ctx).
		Preload("LabCenter").
		Preload("LabCenter.Partner").
		Preload("LabTest").
		Where("patient_id = ?", patientID)
	if status != "" {
		qry = qry.Where("status = ?", status)
	}
	err := qry.Order("created_at DESC").Find(&bookings).Error
	return bookings, err
}

func (r *defaultRepository) UpdateBookingStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).Model(&LabBooking{}).Where("id = ?", id).Update("status", status).Error
}

func (r *defaultRepository) MarkBookingPaid(ctx context.Context, id uint, paymentRef string) error {
	return r.db.WithContext(ctx).Model(&LabBooking{}).Where("id = ?", id).Updates(map[string]interface{}{
		"payment_status": "paid",
		"status":         "confirmed",
		"payment_ref":    paymentRef,
		"updated_at":     time.Now(),
	}).Error
}

func (r *defaultRepository) CancelBooking(ctx context.Context, id uint, reason string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&LabBooking{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":        "cancelled",
		"cancel_reason": reason,
		"cancelled_at":  &now,
	}).Error
}

// ── Lab portal ────────────────────────────────────────────────────────────────

func (r *defaultRepository) GetPortalBookings(ctx context.Context, labCenterID uint, status string) ([]LabBooking, error) {
	var bookings []LabBooking
	qry := r.db.WithContext(ctx).
		Preload("LabTest").
		Where("lab_center_id = ?", labCenterID)
	if status != "" {
		qry = qry.Where("status = ?", status)
	} else {
		// Default: show active bookings only
		qry = qry.Where("status NOT IN ?", []string{"completed", "cancelled"})
	}
	err := qry.Order("preferred_date ASC, created_at ASC").Find(&bookings).Error
	return bookings, err
}

func (r *defaultRepository) MarkSampleCollected(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&LabBooking{}).Where("id = ?", id).
		Update("status", "sample_collected").Error
}

func (r *defaultRepository) SubmitBookingResult(ctx context.Context, id uint, fileURL, aiSummary, aiRec string) error {
	return r.db.WithContext(ctx).Model(&LabBooking{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":            "report_ready",
		"report_file_url":   fileURL,
		"ai_summary":        aiSummary,
		"ai_recommendation": aiRec,
	}).Error
}

func (r *defaultRepository) VerifyBookingReport(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&LabBooking{}).Where("id = ?", id).
		Update("status", "completed").Error
}

// ── Hospital internal ─────────────────────────────────────────────────────────

func (r *defaultRepository) CreateLabRequest(ctx context.Context, lr *LabRequest) error {
	return r.db.WithContext(ctx).Create(lr).Error
}

func (r *defaultRepository) GetLabRequestByID(ctx context.Context, id uint) (*LabRequest, error) {
	var lr LabRequest
	err := r.db.WithContext(ctx).Preload("Tests").Preload("Patient").Preload("Doctor").First(&lr, id).Error
	return &lr, err
}

func (r *defaultRepository) GetLabRequestsByPatient(ctx context.Context, patientID uint) ([]LabRequest, error) {
	var lrs []LabRequest
	err := r.db.WithContext(ctx).Where("patient_id = ?", patientID).Preload("Tests").Order("ordered_at desc").Find(&lrs).Error
	return lrs, err
}

func (r *defaultRepository) UpdateLabRequestStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).Model(&LabRequest{}).Where("id = ?", id).Update("status", status).Error
}

func (r *defaultRepository) CreateLabResult(ctx context.Context, res *LabResult) error {
	return r.db.WithContext(ctx).Create(res).Error
}

func (r *defaultRepository) GetLabResultsByRequest(ctx context.Context, requestID uint) ([]LabResult, error) {
	var results []LabResult
	err := r.db.WithContext(ctx).Where("request_id = ?", requestID).Preload("Test").Find(&results).Error
	return results, err
}

func (r *defaultRepository) RecordSpecimenCollection(ctx context.Context, s *Specimen) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *defaultRepository) GetSpecimensByRequest(ctx context.Context, requestID uint) ([]Specimen, error) {
	var specimens []Specimen
	err := r.db.WithContext(ctx).Where("request_id = ?", requestID).Find(&specimens).Error
	return specimens, err
}

// ── Seed helpers ──────────────────────────────────────────────────────────────

func (r *defaultRepository) CountTests(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&LabTest{}).Count(&count).Error
	return count, err
}

func (r *defaultRepository) BulkCreatePartners(ctx context.Context, partners []LabPartner) error {
	return r.db.WithContext(ctx).CreateInBatches(partners, 50).Error
}

func (r *defaultRepository) BulkCreateCenters(ctx context.Context, centers []LabCenter) error {
	return r.db.WithContext(ctx).CreateInBatches(centers, 50).Error
}

func (r *defaultRepository) BulkCreatePricing(ctx context.Context, rows []LabCenterPricing) error {
	return r.db.WithContext(ctx).CreateInBatches(rows, 200).Error
}

func (r *defaultRepository) BulkCreateTests(ctx context.Context, tests []LabTest) error {
	return r.db.WithContext(ctx).CreateInBatches(tests, 100).Error
}

func (r *defaultRepository) BulkCreatePackageItems(ctx context.Context, items []LabPackageItem) error {
	return r.db.WithContext(ctx).CreateInBatches(items, 200).Error
}

package clinical

import (
	"log"
	"time"

	"gorm.io/gorm"
)

// Onboarding pipeline stages.
const (
	StageClinicalReview = "clinical_review"
	StageCredentialing  = "credentialing"
	StageOrientation    = "orientation"
	StageGoLive         = "go_live"
	StageActive         = "active"
	StageRejected       = "rejected"
)

// Pod is the org home for an AYUSH system / specialty. Doctors are auto-routed
// to the pod matching their type at onboarding; the pod's lead is that system's
// head doctor. (The Fiber HTTP handlers that managed pods were removed with the
// REST server; the model, seed and migration helpers remain.)
type Pod struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Name         string `gorm:"not null" json:"name"`        // "Cardiology", "AYUSH / General"
	Specialty    string `gorm:"index" json:"specialty"`      // canonical key
	LeadDoctorID *uint  `gorm:"index" json:"lead_doctor_id"` // head doctor (chief_doctor user id)
	// DoctorTypeID links a pod to one AYUSH doctor type (Siddha/Ayurveda/…), so a
	// pod is the org home for an entire AYUSH system. Nullable for legacy/ad-hoc
	// pods not tied to a type.
	DoctorTypeID *uint     `gorm:"index" json:"doctor_type_id"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Pod) TableName() string { return "pods" }

// DoctorOnboarding tracks a doctor's credentialing + orientation pipeline.
type DoctorOnboarding struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	DoctorID uint   `gorm:"uniqueIndex;not null" json:"doctor_id"` // user id of the doctor
	Stage    string `gorm:"default:'clinical_review';index" json:"stage"`

	// Clinical interview round (head-doctor recommendation).
	ClinicalRecommendation string `gorm:"default:'pending'" json:"clinical_recommendation"` // pending | recommended | rejected
	ClinicalReviewerID     *uint  `json:"clinical_reviewer_id,omitempty"`
	ClinicalNotes          string `gorm:"type:text" json:"clinical_notes"`

	// Credentialing checklist.
	LicenseVerified  bool `gorm:"default:false" json:"license_verified"`
	IdentityVerified bool `gorm:"default:false" json:"identity_verified"`
	AgreementSigned  bool `gorm:"default:false" json:"agreement_signed"`

	// Orientation checklist.
	OrientationDone bool `gorm:"default:false" json:"orientation_done"`
	TestConsultDone bool `gorm:"default:false" json:"test_consult_done"`

	// Go-live.
	ActivatedAt *time.Time `json:"activated_at,omitempty"`
	ActivatedBy *uint      `json:"activated_by,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (DoctorOnboarding) TableName() string { return "doctor_onboardings" }

// SeedAYUSHPods creates one pod per doctor type and backfills existing doctors
// into their type's pod. Idempotent; invoked from the migration/seed path.
// Done in Go (not UPDATE…FROM) so it works on both Postgres and SQLite.
func SeedAYUSHPods(db *gorm.DB) {
	type dtRow struct {
		ID   uint
		Name string
		Slug string
	}
	var types []dtRow
	db.Table("doctor_types").Select("id, name, slug").Find(&types)
	for _, t := range types {
		tid := t.ID
		var existing int64
		db.Model(&Pod{}).Where("doctor_type_id = ?", tid).Count(&existing)
		if existing == 0 {
			db.Create(&Pod{
				Name:         t.Name + " — Online",
				Specialty:    t.Slug,
				DoctorTypeID: &tid,
				IsActive:     true,
			})
		}
	}
	// Backfill: route existing doctors with no pod into their type's pod.
	var pods []Pod
	db.Where("is_active = ? AND doctor_type_id IS NOT NULL", true).Find(&pods)
	for _, p := range pods {
		if p.DoctorTypeID == nil {
			continue
		}
		db.Table("doctorprofile").
			Where("doctor_type_id = ? AND (pod_id IS NULL OR pod_id = 0)", *p.DoctorTypeID).
			Update("pod_id", p.ID)
	}
}

// BackfillOnboarding creates onboarding rows for existing unverified doctors so
// the credentialing queue is populated on first migration. Idempotent and
// non-fatal. Invoked from the migration/seed path.
func BackfillOnboarding(db *gorm.DB) {
	// "user" is a reserved word — quote it (works on SQLite and PostgreSQL).
	// Timestamps are set explicitly since GORM auto-timestamps don't apply to
	// raw INSERT…SELECT.
	if err := db.Exec(`
		INSERT INTO doctor_onboardings (doctor_id, stage, clinical_recommendation, clinical_notes, created_at, updated_at)
		SELECT u.id, ?, 'pending', 'Backfilled from existing unverified doctor.', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		FROM "user" u
		JOIN doctorprofile dp ON dp.user_id = u.id
		WHERE u.role = 'doctor' AND dp.is_verified = false
		AND NOT EXISTS (SELECT 1 FROM doctor_onboardings o WHERE o.doctor_id = u.id)
	`, StageClinicalReview).Error; err != nil {
		// Non-fatal — log and continue; the queue still works for new hires.
		log.Printf("[clinical] BackfillOnboarding failed: %v", err)
	}
}

package identity

import (
	"database/sql/driver"
	"drayya-core/internal/money"
	"drayya-core/internal/utils/privacy"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&ActivityLog{},
		&PatientProfile{},
		&DoctorAvailability{},
		&FamilyMember{},
		&ExternalProvider{},
		&Consent{},
		&SOSBlock{},
		&OTPRecord{},
		&DoctorProfile{},
		&PasswordResetToken{},
		&EmailVerificationToken{},
		&DoctorType{},
		&DoctorSubSpecialty{},
		&DoctorSubSpecialtyAssignment{},
	)
}

type SOSBlock struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PatientID uint      `gorm:"index;not null" json:"patient_id"`
	Reason    string    `json:"reason"`
	BlockedBy uint      `json:"blocked_by"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (SOSBlock) TableName() string { return "sos_blocks" }

// OTPRecord stores short-lived phone OTPs for mobile login.
type OTPRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Phone     string    `gorm:"index;not null" json:"phone"`
	OTP       string    `gorm:"not null" json:"otp"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	Used      bool      `gorm:"default:false" json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

func (OTPRecord) TableName() string { return "otp_records" }

type PasswordResetToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Token     string    `gorm:"uniqueIndex;not null" json:"token"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	Used      bool      `gorm:"default:false" json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

func (PasswordResetToken) TableName() string { return "password_reset_tokens" }

type EmailVerificationToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	Token     string    `gorm:"uniqueIndex;not null" json:"token"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (EmailVerificationToken) TableName() string { return "email_verification_tokens" }

type UserRole string

const (
	RolePatient            UserRole = "patient"
	RoleDoctor             UserRole = "doctor"
	RoleAdmin              UserRole = "admin"
	RoleNurse              UserRole = "nurse"
	RolePharmacist         UserRole = "pharmacist"
	RoleLabTechnician      UserRole = "lab_technician"
	RoleHospitalHeadDoctor UserRole = "hospital_head_doctor"
	RoleChiefDoctor        UserRole = "chief_doctor"
	RoleBPO                UserRole = "bpo"
	RoleFinanceManager     UserRole = "finance_manager"
	RoleHRManager          UserRole = "hr_manager"
	RoleQAManager          UserRole = "qa_manager"
	RoleManager            UserRole = "manager"
	RoleStoreManager       UserRole = "store_manager"
	RoleFranchise          UserRole = "franchise_owner"
	RoleSupplierAgent      UserRole = "supplier_agent"
	RoleInsuranceAgent     UserRole = "insurance_agent"
	RoleAccountant         UserRole = "accountant"
	RoleLogisticsCoordinator UserRole = "logistics_coordinator"
	RoleHospitalAdmin        UserRole = "hospital_admin"
	RoleCorporatePartner     UserRole = "corporate_partner"
	RolePharmacyAdmin        UserRole = "pharmacy_admin"
	RoleBPOManager           UserRole = "bpo_manager"
	RoleDispatcher           UserRole = "dispatcher"
)

type User struct {
	ID                        uint           `gorm:"primaryKey" json:"id"`
	OrganizationID            uint           `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID            uint           `gorm:"index;default:2" json:"business_unit_id"` // Default to Healthcare (2) for legacy
	UHEID                     *string        `gorm:"column:uheid;uniqueIndex" json:"uheid"`
	Email                     string         `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash              string         `gorm:"not null" json:"-"`
	Role                      string         `gorm:"index:idx_user_hospital_role,priority:2;default:'patient'" json:"role"`
	FullName                  string         `json:"full_name"`
	Gender                    string         `json:"gender"`
	PhoneNumber               string         `gorm:"index" json:"phone_number"`
	IsActive                  bool           `gorm:"default:true" json:"is_active"`
	IsVerified                bool           `gorm:"default:false" json:"is_verified"`
	// Shadow fields for joined data
	IsOnline       bool    `gorm:"-" json:"is_online,omitempty"`
	Specialization string  `gorm:"-" json:"specialization,omitempty"`
	CurrentStatus  string  `gorm:"-" json:"current_status,omitempty"`
	AverageRating  float32 `gorm:"-" json:"average_rating,omitempty"`

	ProfilePicture            string         `json:"profile_picture"`
	Address                   string         `json:"address"`
	WhatsAppNumber            string         `json:"whatsapp_number"`
	HospitalID                *uint          `gorm:"index:idx_user_hospital_role,priority:1" json:"hospital_id"`
	Tier                      string         `gorm:"default:'free'" json:"tier"`
	IsVerifiedPremium         bool           `gorm:"default:false" json:"is_verified_premium"`
	PreferredLanguage         string         `gorm:"default:'english'" json:"preferred_language"`
	SecondaryLanguages        string         `json:"secondary_languages"`
	// PsychologyEligible gates access to psychology consultations — a restricted
	// medical system offered only to selected/referred patients. Granted by an
	// admin or a treating doctor.
	PsychologyEligible        bool           `gorm:"default:false" json:"psychology_eligible"`
	WalletBalance             int64          `gorm:"default:0" json:"wallet_balance"`
	PrakritiScore             string         `json:"prakriti_score"`
	FreeConsultsUsed          int            `gorm:"default:0" json:"free_consults_used"`
	MustChangePassword        bool           `gorm:"default:false" json:"must_change_password"`
	MFACode                   string         `json:"-"`
	MFALastSent               *time.Time     `json:"-"`
	SubscriptionStartDate     *time.Time     `json:"subscription_start_date"`
	SubscriptionEndDate       *time.Time     `json:"subscription_end_date"`
	SubscriptionAutoRenew     bool           `gorm:"default:false" json:"subscription_auto_renew"`
	SubscriptionPaymentMethod string         `json:"subscription_payment_method"`
	LastLogin                 *time.Time     `json:"last_login"`
	FCMToken                  string         `json:"fcm_token,omitempty"` // Firebase push token
	CreatedAt                 time.Time      `json:"created_at"`
	UpdatedAt                 time.Time      `json:"updated_at"`
	DeletedAt                 gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "user" // Keep legacy table name for compatibility
}

type ActivityLog struct {
	ID     uint `gorm:"primaryKey" json:"id"`
	UserID uint `gorm:"index" json:"user_id"`
	// User is loaded via Preload for display but the FK constraint is
	// intentionally NOT created (constraint:- tells GORM to skip it).
	// We allow user_id = 0 as a sentinel for anonymous events like failed
	// login attempts (no authenticated user yet). A real FK rejects those.
	User      User      `json:"user" gorm:"foreignKey:UserID;references:ID;constraint:-"`
	Action    string    `json:"action"`
	Entity    string    `json:"entity"`
	EntityID  string    `json:"entity_id"`
	Details   JSON      `json:"details" gorm:"type:text"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

type JSON map[string]interface{}

func (j JSON) Value() (driver.Value, error) {
	return json.Marshal(j)
}

// Scan handles JSON columns from multiple drivers:
//   - lib/pq (Postgres):  returns string for text/jsonb
//   - pgx (Postgres):     returns []byte for jsonb
//   - go-sqlite3:         returns string
//   - mysql:              returns []byte
// Previous implementation only handled []byte which caused "type assertion to
// []byte failed" errors when querying activity_logs from PostgreSQL.
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("JSON.Scan: unsupported type %T", value)
	}
	if len(raw) == 0 {
		*j = nil
		return nil
	}
	return json.Unmarshal(raw, j)
}

type PatientProfile struct {
	ID                uint                    `gorm:"primaryKey" json:"id"`
	UserID            uint                    `gorm:"uniqueIndex" json:"user_id"`
	DateOfBirth       *time.Time              `json:"date_of_birth"`
	Gender            string                  `json:"gender"`
	BloodGroup        string                  `json:"blood_group"`
	Height            float64                 `json:"height"`
	Weight            float64                 `json:"weight"`
	Allergies         privacy.EncryptedString `gorm:"type:text" json:"allergies"`
	ChronicConditions privacy.EncryptedString `gorm:"type:text" json:"chronic_conditions"`
	EmergencyContact  privacy.EncryptedString `json:"emergency_contact"`
	Address           privacy.EncryptedString `json:"address"`
	CreatedAt         time.Time               `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (PatientProfile) TableName() string {
	return "patient_profiles"
}

type DoctorProfile struct {
	ID                       uint       `gorm:"primaryKey" json:"id"`
	UserID                   uint       `gorm:"uniqueIndex;not null" json:"user_id"`
	// PodID assigns the doctor to a specialty pod for clinical oversight. Nullable
	// — unassigned doctors fall under network-wide (admin/manager) oversight.
	PodID                    *uint      `gorm:"index" json:"pod_id,omitempty"`
	// ServiceZoneID is the geographic zone the doctor serves for IN-PERSON /
	// hospital care — orthogonal to PodID (specialty). Drives zone-first matching
	// when a patient books an in-person visit. Nullable; online-only doctors leave
	// it unset.
	ServiceZoneID            *uint      `gorm:"index" json:"service_zone_id,omitempty"`
	Specialization           string     `gorm:"index" json:"specialization"`
	DoctorTypeID             *uint      `gorm:"index" json:"doctor_type_id"`
	LicenseNumber            string     `gorm:"uniqueIndex" json:"license_number"`
	// LicenseExpiry powers the compliance monitor (Phase 3) — flagged when within
	// 30 days of lapse, blocked when past. Nullable: unknown until the doctor/HR
	// records it.
	LicenseExpiry            *time.Time `gorm:"index" json:"license_expiry,omitempty"`
	YearsOfExperience        int        `json:"years_of_experience"`
	Qualification            string     `json:"qualification"`
	CollegeName              string     `json:"college_name"`
	ConsultationFees         money.RegionalPricing `gorm:"type:jsonb" json:"consultation_fees"`
	// Indexed: doctor listings filter on (is_verified, is_online, specialization)
	// constantly. Without indexes, every doctor-search query is a full table scan
	// (a hot path that runs on every patient dashboard load).
	IsVerified               bool                  `gorm:"default:false;index" json:"is_verified"`
	IsOnline                 bool       `gorm:"default:false;index" json:"is_online"`
	CurrentStatus            string     `gorm:"default:'offline';index" json:"current_status"`
	Gender                   string     `json:"gender"`
	Latitude                 float64    `json:"latitude"`
	Longitude                float64    `json:"longitude"`
	Bio                      string     `json:"bio"`
	AverageRating            float64    `json:"average_rating"`
	TotalConsultations       int        `json:"total_consultations"`
	TotalConsultationsToday  int        `gorm:"default:0" json:"total_consultations_today"`
	ProfileCompletionPercent int        `json:"profile_completion_percent"`
	LanguagesSpoken          string     `json:"languages_spoken"`
	AdminBoostScore          float64    `gorm:"default:0" json:"admin_boost_score"`
	OffersFreeConsultation   bool       `gorm:"default:false" json:"offers_free_consultation"`
	DailyFreeConsultLimit    int        `gorm:"default:0" json:"daily_free_consult_limit"`
	FreeConsultsGivenToday   int        `gorm:"default:0" json:"free_consults_given_today"`
	FreeType                 string     `gorm:"default:'first_time_only'" json:"free_type"`
	FreeDayOfWeek            int        `gorm:"default:0" json:"free_day_of_week"`
	// Indexed: presence cleanup goroutine sweeps stale pings every 5 minutes.
	LastPing                 *time.Time `gorm:"index" json:"last_ping"`
	// EmergencyCapable gates the SOS pool: only doctors explicitly marked
	// emergency-capable (e.g. MBBS / emergency-trained) may be matched to or
	// accept SOS/emergency requests. AYUSH scope-of-practice means specialty
	// doctors must NOT be the default SOS responder.
	EmergencyCapable    bool    `gorm:"default:false;index" json:"emergency_capable"`
	// Discovery & ranking fields
	TotalReviews        int     `gorm:"default:0" json:"total_reviews"`
	IsFeatured          bool    `gorm:"default:false" json:"is_featured"`
	PerformanceScore    float64 `gorm:"default:0" json:"performance_score"`
	WeeklyConsultations int     `gorm:"default:0" json:"weekly_consultations"`
	CompletionRate      float64 `gorm:"default:0" json:"completion_rate"`
	// Scorecard & tier (Phase 2): a weighted blend of rating, completion, QA
	// score and volume, bucketed into bronze/silver/gold/platinum. Drives ranking
	// and payout share. Recomputed by the clinical scorecard job.
	ScorecardScore      float64    `gorm:"default:0" json:"scorecard_score"`
	Tier                string     `gorm:"default:'bronze';index" json:"tier"`
	TierComputedAt      *time.Time `json:"tier_computed_at,omitempty"`
	// Payout details — set by doctor in profile settings
	PayoutUPI                string     `json:"payout_upi,omitempty"`
	BankAccountNumber        string     `json:"bank_account_number,omitempty"`
	BankIFSC                 string     `json:"bank_ifsc,omitempty"`
	// Tax identity — required to file Form 16A / 26Q TDS certificates for the
	// revenue-share earnings of freelance (panel) doctors.
	PAN                      string     `gorm:"column:pan" json:"pan,omitempty"`
	PANVerified              bool       `gorm:"default:false" json:"pan_verified"`
	RazorpayFundAccountID    string         `json:"-"` // cached after first payout
	CreatedAt                time.Time      `json:"created_at"`
	UpdatedAt                time.Time      `json:"updated_at"`
	DeletedAt                gorm.DeletedAt `gorm:"index" json:"-"`
}

func (DoctorProfile) TableName() string {
	return "doctorprofile"
}

// DoctorType represents one of the 4 AYUSH healing systems.
type DoctorType struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Slug        string    `gorm:"uniqueIndex;not null" json:"slug"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Color       string    `json:"color"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (DoctorType) TableName() string { return "doctor_types" }

// DoctorSubSpecialty is an add-on skill under a DoctorType (admin-managed).
type DoctorSubSpecialty struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	DoctorTypeID        uint       `gorm:"index;not null" json:"doctor_type_id"`
	DoctorType          DoctorType `gorm:"foreignKey:DoctorTypeID" json:"doctor_type,omitempty"`
	Name                string     `gorm:"not null" json:"name"`
	Slug                string     `gorm:"index;not null" json:"slug"`
	Description         string     `json:"description"`
	RequiresCertificate bool       `gorm:"default:false" json:"requires_certificate"`
	IsActive            bool       `gorm:"default:true" json:"is_active"`
	CreatedBy           uint       `json:"created_by"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func (DoctorSubSpecialty) TableName() string { return "doctor_sub_specialties" }

// DoctorSubSpecialtyAssignment links a doctor to a sub-specialty (assigned by admin/chief).
type DoctorSubSpecialtyAssignment struct {
	ID             uint               `gorm:"primaryKey" json:"id"`
	DoctorUserID   uint               `gorm:"index;not null" json:"doctor_user_id"`
	SubSpecialtyID uint               `gorm:"index;not null" json:"sub_specialty_id"`
	SubSpecialty   DoctorSubSpecialty `gorm:"foreignKey:SubSpecialtyID" json:"sub_specialty,omitempty"`
	AssignedBy     uint               `json:"assigned_by"`
	AssignedByRole string             `json:"assigned_by_role"`
	CertificateURL string             `json:"certificate_url"`
	IsActive       bool               `gorm:"default:true" json:"is_active"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

func (DoctorSubSpecialtyAssignment) TableName() string { return "doctor_sub_specialty_assignments" }

type DoctorAvailability struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	DoctorID    uint      `gorm:"index;not null" json:"doctor_id"`
	DayOfWeek   int       `gorm:"not null" json:"day_of_week"`
	StartTime   string    `gorm:"not null" json:"start_time"`
	EndTime     string    `gorm:"not null" json:"end_time"`
	IsAvailable bool      `gorm:"default:true" json:"is_available"`
	Version     uint      `gorm:"default:1;not null" json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (DoctorAvailability) TableName() string {
	return "doctor_availability"
}

type FamilyMember struct {
	ID                uint                    `gorm:"primaryKey" json:"id"`
	UserID            uint                    `gorm:"index;not null" json:"user_id"`
	FullName          string                  `gorm:"not null" json:"full_name"`
	Relationship      string                  `gorm:"not null" json:"relationship"`
	DateOfBirth       *time.Time              `json:"date_of_birth"`
	Gender            string                  `json:"gender"`
	BloodGroup        string                  `json:"blood_group"`
	Allergies         privacy.EncryptedString `gorm:"type:text" json:"allergies"`
	MedicalConditions privacy.EncryptedString `gorm:"type:text" json:"medical_conditions"`
	ProfilePhotoURL   string         `json:"profile_photo_url"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (FamilyMember) TableName() string {
	return "family_members"
}

type ExternalProvider struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Type      string    `gorm:"index" json:"type"` // HOSPITAL, CLINIC, INSURER
	APIKey    string    `json:"-"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

func (ExternalProvider) TableName() string {
	return "external_providers"
}

type ConsentScope string

const (
	ScopeFinancialProfile ConsentScope = "financial_profile"
	ScopeMedicalProfile   ConsentScope = "medical_profile"
	ScopeUHEIDOnly        ConsentScope = "uheid_only"
)

type Consent struct {
	ID                 uint         `gorm:"primaryKey" json:"id"`
	UserID             uint         `gorm:"index;not null" json:"user_id"`
	ExternalProviderID uint         `gorm:"index;not null" json:"external_provider_id"`
	Scope              ConsentScope `gorm:"type:varchar(50)" json:"scope"`
	ExpiresAt          time.Time    `json:"expires_at"`
	IsRevoked          bool         `gorm:"default:false" json:"is_revoked"`
	CreatedAt          time.Time    `json:"created_at"`
}

func (Consent) TableName() string {
	return "user_consents"
}

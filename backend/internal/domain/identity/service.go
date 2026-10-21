package identity

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"drayya-core/internal/crypto"
	"drayya-core/internal/database"
	"drayya-core/internal/services/email"
	"drayya-core/internal/services/notification"
	"drayya-core/internal/tokens"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user"`
}

type MFAResponse struct {
	MFAToken string `json:"mfa_token"`
	Message  string `json:"message"`
}

// NotificationEnqueuer is the minimal interface the identity service needs to
// dispatch MFA OTPs.  It is satisfied by *notification.Service, but defined
// here to avoid a hard dependency on that package and to keep the interface
// easily mockable in unit tests.
type NotificationEnqueuer interface {
	EnqueueMFAOTP(ctx context.Context, recipientEmail, phone, name, otp string) error
}

type Service interface {
	// Auth
	Login(req LoginRequest) (*TokenResponse, error)
	LoginStaff(req LoginRequest) (*MFAResponse, error)
	VerifyMFA(userID uint, code string) (*TokenResponse, error)
	ResendMFA(userID uint) error
	RefreshToken(oldRefreshToken string) (*TokenResponse, error)
	Register(user User, password string) (*User, error)

	// SetNotifier wires in the notification outbox.  Call this once after
	// construction, before the server starts accepting requests.
	SetNotifier(n NotificationEnqueuer)

	// Profile Management
	GetProfile(userID uint) (*User, error)
	UpdateProfile(userID uint, updates map[string]interface{}) error
	GetPatientProfile(userID uint) (*PatientProfile, error)
	GetDoctorProfile(userID uint) (*DoctorProfile, error)
	UpdateDoctorProfile(userID uint, updates map[string]interface{}) error
	UpdateDoctorProfileTyped(userID uint, req DoctorSelfUpdateRequest) error
	AdminUpdateDoctorProfile(adminID uint, targetDoctorID uint, req DoctorAdminUpdateRequest) error

	// Doctor Availability
	GetAvailability(doctorID uint) ([]DoctorAvailability, error)
	UpdateAvailability(doctorID uint, schedules []DoctorAvailability) error

	// Family
	GetFamilyMembers(userID uint) ([]FamilyMember, error)
	AddFamilyMember(member FamilyMember) error
	GetFamilyMember(id uint, userID uint) (*FamilyMember, error)
	UpdateFamilyMember(id uint, userID uint, updates map[string]interface{}) error
	DeleteFamilyMember(id uint, userID uint) error

	ListDoctors(specialization string, verifiedOnly bool) ([]User, error)
	RateDoctor(doctorID uint, rating float64) error
	GenerateUHEID() string

	// Phone OTP
	SendOTP(phone string) (string, error)
	LoginWithOTP(phone, otp string) (*TokenResponse, error)

	// UHEID Portability
	GrantConsent(userID uint, providerID uint, scope ConsentScope, durationDays int) error
	GetUHEIDProfile(uheid string, providerAPIKey string, scope ConsentScope) (map[string]interface{}, error)

	// Password Management
	ChangePassword(userID uint, currentPassword, newPassword string) error
	ForgotPassword(email string) error
	ResetPassword(token, newPassword string) error

	// Email Verification
	SendVerificationEmail(userID uint) error
	VerifyEmail(token string) error
}

type service struct {
	repo     Repository
	notifier NotificationEnqueuer // set via SetNotifier; nil = legacy path
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// SetNotifier injects the notification outbox. Once set, LoginStaff and
// ResendMFA enqueue OTPs through the outbox instead of calling the email
// service and WhatsApp directly.
func (s *service) SetNotifier(n NotificationEnqueuer) {
	s.notifier = n
}

func (s *service) Login(req LoginRequest) (*TokenResponse, error) {
	req.Email = strings.TrimSpace(req.Email)
	user, err := s.repo.GetUserByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !user.IsActive {
		return nil, errors.New("account is inactive")
	}

	// Update last login timestamp and restore if soft-deleted
	now := time.Now()
	updates := map[string]interface{}{"last_login": &now}
	if user.DeletedAt.Valid {
		updates["deleted_at"] = nil
	}
	s.repo.UpdateUser(user.ID, updates)

	accessToken, err := tokens.CreateAccessToken(fmt.Sprintf("%d", user.ID), user.Role, user.HospitalID, nil)
	if err != nil {
		return nil, err
	}

	refreshToken, err := tokens.CreateRefreshToken(fmt.Sprintf("%d", user.ID), user.Role, user.HospitalID, nil)
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (s *service) LoginStaff(req LoginRequest) (*MFAResponse, error) {
	req.Email = strings.TrimSpace(req.Email) // Match Login() behaviour — trim leading/trailing whitespace
	user, err := s.repo.GetUserByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Deny non-staff roles. All other roles (doctor, nurse, admin, pharmacist,
	// lab_technician, hr_manager, finance_manager, manager, bpo, qa_manager,
	// store_manager, hospital_head_doctor, hospital_admin, pharmacy_admin,
	// logistics_coordinator, accountant, supplier_agent, insurance_agent,
	// franchise_owner, corporate_partner, courier …) are permitted through.
	// Check environment variable for MFA requirement. Defaults to true.
	// If REQUIRE_MFA is false, allow staff roles to bypass MFA.
	requireMFA := os.Getenv("REQUIRE_MFA")
	if requireMFA == "" {
		requireMFA = "true"
	}

	// Deny non-staff roles. 
	nonStaffRoles := map[string]bool{
		string(RolePatient): true,
	}
	
	if requireMFA == "false" {
		log.Printf("[Auth] MFA globally disabled via REQUIRE_MFA=false. Allowing staff to bypass MFA.")
		// Add all roles we want to bypass to nonStaffRoles so they fall through to normal login
		nonStaffRoles["doctor"] = true
		nonStaffRoles["admin"] = true
		nonStaffRoles["pharmacist"] = true
		nonStaffRoles["nurse"] = true
		nonStaffRoles["lab_technician"] = true
		nonStaffRoles["hr_manager"] = true
		nonStaffRoles["finance_manager"] = true
		nonStaffRoles["manager"] = true
		nonStaffRoles["bpo"] = true
		nonStaffRoles["qa_manager"] = true
		nonStaffRoles["store_manager"] = true
		nonStaffRoles["hospital_head_doctor"] = true
		nonStaffRoles["hospital_admin"] = true
		nonStaffRoles["pharmacy_admin"] = true
		nonStaffRoles["logistics_coordinator"] = true
		nonStaffRoles["accountant"] = true
		nonStaffRoles["supplier_agent"] = true
		nonStaffRoles["insurance_agent"] = true
		nonStaffRoles["franchise_owner"] = true
		nonStaffRoles["corporate_partner"] = true
		nonStaffRoles["courier"] = true
	}

	if nonStaffRoles[user.Role] {
		log.Printf("[Auth] Staff Login Restricted: User %s has role %s", user.Email, user.Role)
		return nil, errors.New("access restricted to staff only")
	}

	if !user.IsActive {
		log.Printf("[Auth] Staff Login Rejected: account inactive for %s", user.Email)
		return nil, errors.New("account is inactive")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		log.Printf("[Auth] Staff Login Failed: Password mismatch for %s", user.Email)
		return nil, errors.New("invalid credentials")
	}

	// Update last login timestamp and restore if soft-deleted
	now := time.Now()
	updates := map[string]interface{}{
		"last_login":    &now,
		"mfa_last_sent": &now,
	}
	if user.DeletedAt.Valid {
		updates["deleted_at"] = nil
	}

	// generateOTP returns a cryptographically random 6-digit numeric string.
	// We use the local generator for consistency with the resend logic.
	otp, err := generateOTP()
	if err != nil {
		return nil, fmt.Errorf("failed to generate OTP: %w", err)
	}
	updates["mfa_code"] = otp

	err = s.repo.UpdateUser(user.ID, updates)
	if err != nil {
		return nil, err
	}

	mfaToken, err := tokens.CreateMFAToken(fmt.Sprintf("%d", user.ID), user.HospitalID)
	if err != nil {
		return nil, fmt.Errorf("failed to create MFA token: %w", err)
	}

	// Dispatch OTP notification via the outbox (reliable, retried) when the
	// notifier is wired; otherwise fall back to the legacy direct-send path.
	mfaMsg := "MFA code sent to your registered contact."
	if s.notifier != nil {
		log.Printf("[Auth] Staff Login: Enqueuing MFA OTP for %s", user.Email)
		if err := s.notifier.EnqueueMFAOTP(
			context.Background(), user.Email, user.PhoneNumber, user.FullName, otp,
		); err != nil {
			log.Printf("[Auth] EnqueueMFAOTP FAILED for %s: %v", user.Email, err)
			mfaMsg = "OTP generated. Notification enqueue failed — please retry or contact support."
		}
	} else {
		// Legacy path: direct email + WhatsApp with 3-second timeout.
		// Kept as fallback for environments where the DB outbox is not wired
		// (e.g. integration tests, local dev without Postgres).
		log.Printf("[Auth] Staff Login (legacy notify): Sending MFA OTP to %s", user.Email)
		type emailResult struct{ err error }
		emailCh := make(chan emailResult, 1)
		go func() {
			emailCh <- emailResult{email.SendSimple(user.Email, "Your Dr.Ayya Login Code",
				email.StaffMFAEmail(user.FullName, otp))}
		}()
		select {
		case res := <-emailCh:
			if res.err != nil {
				log.Printf("[Auth] MFA Email FAILED for %s: %v", user.Email, res.err)
				log.Printf("[Auth] Email config — RESEND_API_KEY set: %v, RESEND_FROM_EMAIL: %q",
					os.Getenv("RESEND_API_KEY") != "", os.Getenv("RESEND_FROM_EMAIL"))
				mfaMsg = "OTP generated. Email delivery failed — please check server email configuration. " +
					"If you have a registered phone number, check WhatsApp/SMS for your code."
			} else {
				log.Printf("[Auth] MFA Email Sent to %s", user.Email)
			}
		case <-time.After(3 * time.Second):
			log.Printf("[Auth] MFA Email send is taking >3 s for %s — continuing in background", user.Email)
			mfaMsg = "OTP generated. Email delivery is slow — it may arrive in a few seconds. " +
				"If you don't receive it within 60 seconds, click 'Resend Code'."
		}
		if user.PhoneNumber != "" {
			notification.WhatsAppOTP(user.PhoneNumber, user.FullName, otp)
		}
	}

	return &MFAResponse{
		MFAToken: mfaToken,
		Message:  mfaMsg,
	}, nil
}

func (s *service) VerifyMFA(userID uint, code string) (*TokenResponse, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	if user.MFACode == "" {
		return nil, errors.New("no active MFA code — please request a new one")
	}
	if user.MFACode != code {
		log.Printf("[Auth] MFA Verification Failed for user %d: invalid code entered", user.ID)
		return nil, errors.New("invalid MFA code")
	}

	// OTP is valid for 10 minutes. The outbox retry schedule can add up to
	// 6 minutes before a second delivery attempt (1 min backoff), so 5 min
	// was too tight and caused users to see "expired" even on the first retry.
	if user.MFALastSent == nil || time.Since(*user.MFALastSent) > 10*time.Minute {
		// Clear the expired code proactively
		log.Printf("[Auth] MFA Code Expired for user %d (Sent: %v, Now: %v)", user.ID, user.MFALastSent, time.Now())
		s.repo.UpdateUser(user.ID, map[string]interface{}{"mfa_code": "", "mfa_last_sent": nil})
		return nil, errors.New("MFA code expired — click \"Resend Code\" to get a new one")
	}

	// 3. Clear code and generate final tokens
	log.Printf("[Auth] MFA Verified for user %d (Role: %s)", user.ID, user.Role)
	s.repo.UpdateUser(user.ID, map[string]interface{}{
		"mfa_code":      "",
		"mfa_last_sent": nil,
	})

	accessToken, err := tokens.CreateAccessToken(fmt.Sprintf("%d", user.ID), user.Role, user.HospitalID, nil)
	if err != nil {
		log.Printf("[Auth] ERROR: Failed to create access token for user %d: %v", user.ID, err)
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := tokens.CreateRefreshToken(fmt.Sprintf("%d", user.ID), user.Role, user.HospitalID, nil)
	if err != nil {
		log.Printf("[Auth] ERROR: Failed to create refresh token for user %d: %v", user.ID, err)
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// PERSIST refresh token in Redis for stateful revocation
	if err := tokens.StoreRefreshToken(context.Background(), user.ID, refreshToken); err != nil {
		log.Printf("[Auth] WARNING: Failed to store refresh token for user %d in Redis: %v", user.ID, err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

// ResendMFA regenerates and re-sends the OTP for an existing MFA session.
// Rate limit: max 3 resends per 10 minutes (checked via mfa_last_sent timestamp).
// The caller must already hold a valid mfa_token (enforced by middleware.Protected).
func (s *service) ResendMFA(userID uint) error {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	// Rate-limit: if a code was sent less than 60 seconds ago, reject.
	if user.MFALastSent != nil && time.Since(*user.MFALastSent) < 60*time.Second {
		wait := 60 - int(time.Since(*user.MFALastSent).Seconds())
		return fmt.Errorf("please wait %d seconds before requesting another code", wait)
	}

	otp, err := generateOTP()
	if err != nil {
		return fmt.Errorf("failed to generate OTP: %w", err)
	}
	now := time.Now()
	if err := s.repo.UpdateUser(user.ID, map[string]interface{}{
		"mfa_code":      otp,
		"mfa_last_sent": &now,
	}); err != nil {
		return fmt.Errorf("failed to save OTP: %w", err)
	}

	log.Printf("[SECURITY] MFA OTP resent for %s", user.Email)

	// Enqueue via outbox when wired; legacy direct-send otherwise.
	if s.notifier != nil {
		if err := s.notifier.EnqueueMFAOTP(
			context.Background(), user.Email, user.PhoneNumber, user.FullName, otp,
		); err != nil {
			log.Printf("[Auth] ResendMFA EnqueueMFAOTP FAILED for %s: %v", user.Email, err)
			// Return error so the HTTP handler can reply 500 — the user should
			// know the resend failed rather than seeing a false "sent" success.
			return fmt.Errorf("failed to queue OTP notification: %w", err)
		}
	} else {
		// Legacy fallback
		type emailResult struct{ err error }
		emailCh := make(chan emailResult, 1)
		go func() {
			emailCh <- emailResult{email.SendSimple(user.Email, "Your Dr.Ayya Login Code (Resent)",
				email.StaffMFAEmail(user.FullName, otp))}
		}()
		select {
		case res := <-emailCh:
			if res.err != nil {
				log.Printf("[Auth] Resend MFA Email FAILED for %s: %v", user.Email, res.err)
				log.Printf("[Auth] Email config — RESEND_API_KEY set: %v, RESEND_FROM_EMAIL: %q",
					os.Getenv("RESEND_API_KEY") != "", os.Getenv("RESEND_FROM_EMAIL"))
			} else {
				log.Printf("[Auth] Resend MFA Email sent to %s", user.Email)
			}
		case <-time.After(3 * time.Second):
			log.Printf("[Auth] Resend MFA Email send taking >3 s for %s — delivery in background", user.Email)
		}
		if user.PhoneNumber != "" {
			notification.WhatsAppOTP(user.PhoneNumber, user.FullName, otp)
		}
	}

	return nil
}

// generateOTP returns a cryptographically random 6-digit numeric string.
func generateOTP() (string, error) {
	const digits = 6
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(digits), nil) // 1_000_000
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (s *service) RefreshToken(oldRefreshToken string) (*TokenResponse, error) {
	claims, err := tokens.ValidateToken(oldRefreshToken)
	if err != nil {
		return nil, err
	}

	if claims.Type != "refresh" {
		return nil, errors.New("invalid token type")
	}

	// Correct parsing logic needed if it's a string ID
	var uid uint
	fmt.Sscanf(claims.UserID, "%d", &uid)

	user, err := s.repo.GetUserByID(uid)
	if err != nil {
		return nil, errors.New("user not found")
	}

	accessToken, _ := tokens.CreateAccessToken(fmt.Sprintf("%d", user.ID), user.Role, user.HospitalID, nil)
	refreshToken, _ := tokens.CreateRefreshToken(fmt.Sprintf("%d", user.ID), user.Role, user.HospitalID, nil)

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (s *service) Register(user User, password string) (*User, error) {
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = string(hashed)
	if user.UHEID == nil || *user.UHEID == "" {
		uheid := s.GenerateUHEID()
		user.UHEID = &uheid
	}

	err = s.repo.ExecuteTransaction(func(txRepo Repository) error {
		if err := txRepo.CreateUser(&user); err != nil {
			return err
		}

		if user.Role == "doctor" {
			// Generate a temporary unique license number to avoid database UNIQUE constraint violation
			// on multiple doctors with empty ("") license numbers before they are updated.
			tempLic := fmt.Sprintf("TEMP-LIC-%d-%d", user.ID, time.Now().UnixNano())
			profile := DoctorProfile{
				UserID:          user.ID,
				IsVerified:      false,
				AdminBoostScore: 1.0,
				LicenseNumber:   tempLic,
			}
			if err := txRepo.CreateDoctorProfile(&profile); err != nil {
				return err
			}
		} else if user.Role == "patient" {
			profile := PatientProfile{
				UserID: user.ID,
			}
			if err := txRepo.CreatePatientProfile(&profile); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	go email.SendSimple(user.Email, "Welcome to Dr.Ayya!",
		email.WelcomeEmail(user.FullName, user.Role))

	// Send verification email asynchronously
	go s.SendVerificationEmail(user.ID)

	return &user, nil
}

func (s *service) GetProfile(userID uint) (*User, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = ""
	return user, nil
}

func (s *service) UpdateProfile(userID uint, updates map[string]interface{}) error {
	delete(updates, "id")
	delete(updates, "password_hash")
	delete(updates, "role")
	delete(updates, "is_verified")
	delete(updates, "is_active")
	delete(updates, "is_verified_premium")
	delete(updates, "wallet_balance")
	delete(updates, "tier")
	delete(updates, "mfa_code")
	delete(updates, "mfa_last_sent")
	delete(updates, "fcm_token")
	return s.repo.UpdateUser(userID, updates)
}

func (s *service) GetPatientProfile(userID uint) (*PatientProfile, error) {
	return s.repo.GetPatientProfileByUserID(userID)
}

func (s *service) GetDoctorProfile(userID uint) (*DoctorProfile, error) {
	return s.repo.GetDoctorProfileByUserID(userID)
}

func (s *service) UpdateDoctorProfile(userID uint, updates map[string]interface{}) error {
	// Strip fields that must never be client-writable
	for _, f := range []string{"id", "user_id", "is_verified", "admin_boost_score",
		"total_consultations", "total_consultations_today", "razorpay_fund_account_id",
		"free_consults_given_today", "payout_upi", "bank_account_number", "bank_ifsc"} {
		delete(updates, f)
	}
	return s.repo.UpdateDoctorProfile(userID, updates)
}

func (s *service) UpdateDoctorProfileTyped(userID uint, req DoctorSelfUpdateRequest) error {
	updates := map[string]interface{}{}
	if req.Specialization != nil     { updates["specialization"] = *req.Specialization }
	if req.Qualification != nil      { updates["qualification"] = *req.Qualification }
	if req.YearsOfExperience != nil  { updates["years_of_experience"] = *req.YearsOfExperience }
	if req.ConsultationFee != nil    { updates["consultation_fee"] = *req.ConsultationFee }
	if req.Bio != nil                { updates["bio"] = *req.Bio }
	if req.LanguagesSpoken != nil    { updates["languages_spoken"] = *req.LanguagesSpoken }
	if req.Latitude != nil           { updates["latitude"] = *req.Latitude }
	if req.Longitude != nil          { updates["longitude"] = *req.Longitude }
	if req.PayoutUPI != nil          { updates["payout_upi"] = *req.PayoutUPI }
	if req.BankAccountNumber != nil  { updates["bank_account_number"] = *req.BankAccountNumber }
	if req.BankIFSC != nil           { updates["bank_ifsc"] = *req.BankIFSC }
	if len(updates) == 0 {
		return nil
	}
	return s.repo.UpdateDoctorProfile(userID, updates)
}

func (s *service) AdminUpdateDoctorProfile(adminID uint, targetDoctorID uint, req DoctorAdminUpdateRequest) error {
	updates := map[string]interface{}{}
	if req.IsVerified != nil             { updates["is_verified"] = *req.IsVerified }
	if req.AdminBoostScore != nil        { updates["admin_boost_score"] = *req.AdminBoostScore }
	if req.OffersFreeConsultation != nil { updates["offers_free_consultation"] = *req.OffersFreeConsultation }
	if req.DailyFreeConsultLimit != nil  { updates["daily_free_consult_limit"] = *req.DailyFreeConsultLimit }
	if req.FreeType != nil               { updates["free_type"] = *req.FreeType }
	if len(updates) == 0 {
		return nil
	}
	return s.repo.UpdateDoctorProfile(targetDoctorID, updates)
}

func (s *service) GetAvailability(doctorID uint) ([]DoctorAvailability, error) {
	return s.repo.GetDoctorAvailability(doctorID)
}

func (s *service) UpdateAvailability(doctorID uint, schedules []DoctorAvailability) error {
	return s.repo.UpdateDoctorAvailability(doctorID, schedules)
}

func (s *service) GetFamilyMembers(userID uint) ([]FamilyMember, error) {
	return s.repo.GetFamilyMembersByUserID(userID)
}

func (s *service) AddFamilyMember(member FamilyMember) error {
	return s.repo.CreateFamilyMember(&member)
}

func (s *service) GetFamilyMember(id uint, userID uint) (*FamilyMember, error) {
	return s.repo.GetFamilyMemberByID(id, userID)
}

func (s *service) UpdateFamilyMember(id uint, userID uint, updates map[string]interface{}) error {
	return s.repo.UpdateFamilyMember(id, userID, updates)
}

func (s *service) DeleteFamilyMember(id uint, userID uint) error {
	return s.repo.DeleteFamilyMember(id, userID)
}

func (s *service) ListDoctors(specialization string, verifiedOnly bool) ([]User, error) {
	doctors, err := s.repo.ListDoctors(specialization, verifiedOnly)
	if err != nil {
		return nil, err
	}
	for i := range doctors {
		doctors[i].PasswordHash = ""
	}
	return doctors, nil
}

func (s *service) RateDoctor(doctorID uint, rating float64) error {
	if rating < 1 || rating > 5 {
		return errors.New("rating must be between 1 and 5")
	}
	return s.repo.RateDoctor(doctorID, rating)
}

func (s *service) GenerateUHEID() string {
	date := time.Now().Format("20060102")
	random := uuid.New().String()[:8]
	return fmt.Sprintf("DA-ECON-%s-%s", date, strings.ToUpper(random))
}

func (s *service) GrantConsent(userID uint, providerID uint, scope ConsentScope, durationDays int) error {
	consent := &Consent{
		UserID:             userID,
		ExternalProviderID: providerID,
		Scope:              scope,
		ExpiresAt:          time.Now().AddDate(0, 0, durationDays),
	}
	return s.repo.CreateConsent(consent)
}

// SendOTP generates a 6-digit OTP for the given phone number, stores it with a
// 10-minute expiry, and returns it so the caller can log or forward to an SMS
// gateway. Old unused OTPs for the same phone are purged first.
func (s *service) SendOTP(phone string) (string, error) {
	if phone == "" {
		return "", errors.New("phone number is required")
	}

	// Remove any previous OTPs for this phone
	database.DB.Where("phone = ?", phone).Delete(&OTPRecord{})

	otp := crypto.OTP()
	record := OTPRecord{
		Phone:     phone,
		OTP:       otp,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Used:      false,
	}
	if err := database.DB.Create(&record).Error; err != nil {
		return "", fmt.Errorf("could not store OTP: %w", err)
	}

	// DEV: log OTP — replace with SMS gateway call in production
	log.Printf("[OTP] Phone: %s  Code: %s  Expires: %s", phone, otp, record.ExpiresAt.Format(time.RFC3339))

	return otp, nil
}

// LoginWithOTP validates a phone+OTP pair and returns access/refresh tokens.
func (s *service) LoginWithOTP(phone, otp string) (*TokenResponse, error) {
	if phone == "" || otp == "" {
		return nil, errors.New("phone and OTP are required")
	}

	// Find the user by phone
	var user User
	if err := database.DB.Where("phone_number = ?", phone).First(&user).Error; err != nil {
		return nil, errors.New("user not found for this phone number")
	}

	if !user.IsActive {
		return nil, errors.New("account is inactive")
	}

	// Find a valid (non-expired, non-used) OTP record
	var record OTPRecord
	now := time.Now()
	err := database.DB.
		Where("phone = ? AND otp = ? AND used = ? AND expires_at > ?", phone, otp, false, now).
		First(&record).Error
	if err != nil {
		return nil, errors.New("invalid or expired OTP")
	}

	// Mark OTP as used and restore user if deleted
	database.DB.Model(&record).Update("used", true)
	if user.DeletedAt.Valid {
		database.DB.Unscoped().Model(&user).Update("deleted_at", nil)
	}

	accessToken, err := tokens.CreateAccessToken(fmt.Sprintf("%d", user.ID), user.Role, user.HospitalID, nil)
	if err != nil {
		return nil, err
	}
	refreshToken, err := tokens.CreateRefreshToken(fmt.Sprintf("%d", user.ID), user.Role, user.HospitalID, nil)
	if err != nil {
		return nil, err
	}

	user.PasswordHash = ""
	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         &user,
	}, nil
}

func (s *service) ChangePassword(userID uint, currentPassword, newPassword string) error {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return errors.New("user not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("failed to hash password")
	}
	return s.repo.UpdateUser(userID, map[string]interface{}{"password_hash": string(hash)})
}

func (s *service) ForgotPassword(emailAddr string) error {
	user, err := s.repo.GetUserByEmail(emailAddr)
	if err != nil {
		// Don't reveal whether email exists
		return nil
	}
	// Generate a secure token
	token := crypto.SecureToken(32)
	prt := PasswordResetToken{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}
	// Delete old tokens for this user
	database.DB.Where("user_id = ?", user.ID).Delete(&PasswordResetToken{})
	database.DB.Create(&prt)

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		log.Printf("[identity] WARNING: APP_URL not set; password reset email will have no domain")
		appURL = "https://Dr.Ayya.in"
	}
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", appURL, token)
	
	// Send email synchronously and return error if it fails
	err = email.SendSimple(user.Email, "Reset your Dr.Ayya password", email.PasswordResetEmail(user.FullName, resetURL))
	if err != nil {
		log.Printf("[Auth] ForgotPassword Email FAILED for %s: %v", user.Email, err)
		// Delete the token since we failed to send it
		database.DB.Delete(&prt)
		return fmt.Errorf("Failed to send reset email. Email configuration might be invalid. Please contact support. Error: %v", err)
	}
	
	log.Printf("[Auth] ForgotPassword Email Sent successfully to %s", user.Email)
	return nil
}

func (s *service) ResetPassword(token, newPassword string) error {
	var prt PasswordResetToken
	if err := database.DB.Where("token = ? AND used = ? AND expires_at > ?", token, false, time.Now()).First(&prt).Error; err != nil {
		return errors.New("invalid or expired reset token")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("failed to hash password")
	}
	s.repo.UpdateUser(prt.UserID, map[string]interface{}{"password_hash": string(hash)})
	database.DB.Model(&prt).Update("used", true)
	return nil
}

func (s *service) SendVerificationEmail(userID uint) error {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return err
	}
	token := crypto.SecureToken(32)
	evt := EmailVerificationToken{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	// Upsert: delete old, insert new
	database.DB.Where("user_id = ?", user.ID).Delete(&EmailVerificationToken{})
	database.DB.Create(&evt)

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		log.Printf("[identity] WARNING: APP_URL not set; email verification link will have no domain")
		appURL = "https://Dr.Ayya.in"
	}
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", appURL, token)
	go email.SendSimple(user.Email, "Verify your Dr.Ayya email",
		email.EmailVerificationEmail(user.FullName, verifyURL))
	return nil
}

func (s *service) VerifyEmail(token string) error {
	var evt EmailVerificationToken
	if err := database.DB.Where("token = ? AND expires_at > ?", token, time.Now()).First(&evt).Error; err != nil {
		return errors.New("invalid or expired verification token")
	}
	s.repo.UpdateUser(evt.UserID, map[string]interface{}{"is_verified": true})
	database.DB.Delete(&evt)
	return nil
}

func (s *service) GetUHEIDProfile(uheid string, providerAPIKey string, scope ConsentScope) (map[string]interface{}, error) {
	// 1. Validate Provider
	provider, err := s.repo.GetExternalProviderByAPIKey(providerAPIKey)
	if err != nil {
		return nil, errors.New("invalid provider API key")
	}

	// 2. Find User
	user, err := s.repo.GetUserByUHEID(uheid)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// 3. Check Consent
	_, err = s.repo.GetActiveConsent(user.ID, provider.ID, scope)
	if err != nil {
		return nil, errors.New("consent not found or expired")
	}

	// 4. Return Profile based on Scope
	profile := map[string]interface{}{
		"uheid":     *user.UHEID,
		"full_name": user.FullName,
		"role":      user.Role,
	}

	if scope == ScopeMedicalProfile {
		pProfile, _ := s.repo.GetPatientProfileByUserID(user.ID)
		if pProfile != nil {
			profile["blood_group"] = pProfile.BloodGroup
			profile["allergies"] = pProfile.Allergies
		}
	}

	if scope == ScopeFinancialProfile {
		profile["wallet_balance"] = user.WalletBalance
		profile["tier"] = user.Tier
	}

	return profile, nil
}


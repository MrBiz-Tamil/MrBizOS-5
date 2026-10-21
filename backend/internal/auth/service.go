package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/hr"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/tokens"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token"`
	User         *identity.User `json:"user"`
}

type MFAResponse struct {
	MFAToken string `json:"mfa_token"`
	Message  string `json:"message"`
}

func LoginUser(req LoginRequest) (*TokenResponse, error) {
	var user identity.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	// Verify Password
	// Note: If you have legacy PBKDF2 hashes, you need a custom validator here.
	// This currently only supports BCRYPT.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !user.IsActive {
		return nil, errors.New("account is inactive")
	}

	// Generate Tokens
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
		User:         &user,
	}, nil
}

func LoginStaff(req LoginRequest) (*MFAResponse, error) {
	var user identity.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return nil, errors.New("invalid credentials")
	}

	staffRoles := map[string]bool{
		string(identity.RoleAdmin):                true,
		string(identity.RoleManager):              true,
		string(identity.RoleFinanceManager):       true,
		string(identity.RoleHRManager):            true,
		string(identity.RolePharmacist):           true,
		string(identity.RolePharmacyAdmin):        true,
		string(identity.RoleStoreManager):         true,
		string(identity.RoleQAManager):            true,
		string(identity.RoleHospitalHeadDoctor):   true,
		string(identity.RoleChiefDoctor):          true,
		string(identity.RoleHospitalAdmin):        true,
		string(identity.RoleDoctor):               true,
		string(identity.RoleNurse):                true,
		string(identity.RoleLabTechnician):        true,
		string(identity.RoleBPO):                  true,
		string(identity.RoleBPOManager):           true,
		string(identity.RoleDispatcher):           true,
		string(identity.RoleFranchise):            true,
		string(identity.RoleAccountant):           true,
		string(identity.RoleLogisticsCoordinator): true,
		string(identity.RoleCorporatePartner):     true,
		string(identity.RoleSupplierAgent):        true,
		string(identity.RoleInsuranceAgent):       true,
	}
	if !staffRoles[user.Role] {
		return nil, errors.New("access restricted to staff only")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Generate 6-digit OTP
	otp := "123456" // Mock OTP for implementation phase
	now := time.Now()
	database.DB.Model(&user).Updates(map[string]interface{}{
		"mfa_code":      otp,
		"mfa_last_sent": &now,
	})

	// Generate short-lived MFA token (5 mins)
	mfaToken, _ := tokens.CreateAccessToken(fmt.Sprintf("%d", user.ID), "mfa_pending", user.HospitalID, nil)

	log.Printf("     [SECURITY] MFA Code for %s: %s", user.Email, otp)

	return &MFAResponse{
		MFAToken: mfaToken,
		Message:  "MFA code sent to your registered contact.",
	}, nil
}

func VerifyMFA(userID uint, code string) (*TokenResponse, error) {
	var user identity.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil, err
	}

	if user.MFACode == "" || user.MFACode != code {
		return nil, errors.New("invalid MFA code")
	}

	// Check expiry (e.g., 5 mins)
	if user.MFALastSent == nil || time.Since(*user.MFALastSent) > 5*time.Minute {
		return nil, errors.New("MFA code expired")
	}

	// Clear code and generate final staff tokens
	database.DB.Model(&user).Updates(map[string]interface{}{
		"mfa_code":      "",
		"mfa_last_sent": nil,
	})

	accessToken, _ := tokens.CreateAccessToken(fmt.Sprintf("%d", user.ID), user.Role, user.HospitalID, nil)
	refreshToken, _ := tokens.CreateRefreshToken(fmt.Sprintf("%d", user.ID), user.Role, user.HospitalID, nil)

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         &user,
	}, nil
}

func RefreshToken(oldRefreshToken string) (*TokenResponse, error) {
	claims, err := tokens.ValidateToken(oldRefreshToken)
	if err != nil {
		return nil, err
	}

	if claims.Type != "refresh" {
		return nil, errors.New("invalid token type")
	}

	// Find user
	var user identity.User
	if err := database.DB.First(&user, claims.UserID).Error; err != nil {
		return nil, errors.New("user not found")
	}

	// Generate New Tokens
	accessToken, _ := tokens.CreateAccessToken(fmt.Sprintf("%d", user.ID), user.Role, user.HospitalID, nil)
	refreshToken, _ := tokens.CreateRefreshToken(fmt.Sprintf("%d", user.ID), user.Role, user.HospitalID, nil)

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         &user,
	}, nil
}

// RegisterUser creates a new user and any required sub-profiles in a transaction.
// ctx should be c.UserContext() from the Fiber handler so the GORM audit plugin
// can record the correct actor_id on every INSERT in this transaction.
func RegisterUser(ctx context.Context, user identity.User, password string) (*identity.User, error) {
	// Hash Password
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = string(hashed)

	err = database.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			log.Printf("Error creating user: %v", err)
			return err
		}

		// Create Profile based on Role
		if user.Role == string(identity.RoleDoctor) {
			// Generate a temporary unique license number to avoid database UNIQUE constraint violation
			// on multiple doctors with empty ("") license numbers before they are updated.
			tempLic := fmt.Sprintf("TEMP-LIC-%d-%d", user.ID, time.Now().UnixNano())
			profile := identity.DoctorProfile{
				UserID:          user.ID,
				IsVerified:      false,
				AdminBoostScore: 1.0,
				LicenseNumber:   tempLic,
			}
			if err := tx.Create(&profile).Error; err != nil {
				return err
			}
			// Auto-create StaffProfile so the doctor appears in HR dashboard immediately.
			staffProfile := hr.StaffProfile{
				UserID:           user.ID,
				EmployeeID:       fmt.Sprintf("DOC-%d", user.ID),
				Department:       "Medical",
				Designation:      "Doctor",
				JoiningDate:      time.Now(),
				EmploymentStatus: "active",
			}
			if err := tx.Create(&staffProfile).Error; err != nil {
				log.Printf("[auth] warn: StaffProfile creation failed for user %d: %v", user.ID, err)
			}
		} else if user.Role == string(identity.RolePatient) {
			profile := identity.PatientProfile{
				UserID: user.ID,
			}
			if err := tx.Create(&profile).Error; err != nil {
				log.Printf("Error creating patient profile: %v", err)
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &user, nil
}


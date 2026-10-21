package mfa_test

import (
	"fmt"
	"log"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/services/notification"
	"drayya-core/internal/tokens"
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestMFAFlow(t *testing.T) {
	// 1. Setup DB
	// Pin to 1 connection — SQLite :memory: is per-connection.
	os.Setenv("JWT_SECRET", "test-secret")
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
	}
	database.DB = db

	// Migrate all identity models + notification outbox
	db.AutoMigrate(&identity.User{}, &identity.PatientProfile{}, &identity.DoctorProfile{}, &notification.NotificationJob{})

	repo := identity.NewRepository(db)
	svc := identity.NewService(repo)

	// 2. Create Staff User
	// Use lowercase — Register() normalises email to lower case, so mixed-case
	// lookups would miss the record when reading the OTP back from the DB.
	email := "staff@dr.ayya.in"
	password := "Staff123!"
	user := identity.User{
		Email:    email,
		Role:     "doctor", // staff role
		IsActive: true,
		FullName: "Staff User",
	}
	_, err = svc.Register(user, password)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// 3. Login Staff (Initiate MFA)
	resp, err := svc.LoginStaff(identity.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		t.Fatalf("LoginStaff failed: %v", err)
	}

	log.Printf("MFA Initiated. Token: %s\n", resp.MFAToken)

	// 4. Get the code from DB
	var dbUser identity.User
	db.Where("email = ?", email).First(&dbUser)
	log.Printf("Generated OTP: %s\n", dbUser.MFACode)

	// 5. Verify MFA
	claims, err := tokens.ValidateToken(resp.MFAToken)
	if err != nil {
		t.Fatalf("Token validation failed: %v", err)
	}
	var uid uint
	fmt.Sscanf(claims.UserID, "%d", &uid)

	tokenResp, err := svc.VerifyMFA(uid, dbUser.MFACode)
	if err != nil {
		t.Fatalf("VerifyMFA failed: %v", err)
	}

	log.Printf("MFA Verified! Access Token length: %d\n", len(tokenResp.AccessToken))
}

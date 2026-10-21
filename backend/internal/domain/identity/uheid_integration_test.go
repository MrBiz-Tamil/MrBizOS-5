package identity

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	
	// Migrate all needed models
	err = db.AutoMigrate(&User{}, &ExternalProvider{}, &Consent{}, &PatientProfile{}, &DoctorProfile{}, &ActivityLog{}, &FamilyMember{})
	assert.NoError(t, err)
	return db
}

func TestUHEIDPortabilityAndConsent(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo) // Correct signature

	// 1. Register User and Check UHEID
	t.Log("Registering user...")
	user := User{
		Email:    "test@example.com",
		FullName: "Test User",
	}
	registered, err := svc.Register(user, "password123")
	assert.NoError(t, err)
	assert.NotEmpty(t, *registered.UHEID)
	t.Logf("User registered with UHEID: %s", *registered.UHEID)

	// 2. Create External Provider
	t.Log("Creating provider...")
	provider := &ExternalProvider{
		Name:     "Cloud Hospital",
		Type:     "hospital",
		APIKey:   "hosp-key-123",
		IsActive: true,
	}
	db.Create(provider)

	// 3. Grant Consent
	t.Log("Granting consent...")
	err = svc.GrantConsent(registered.ID, provider.ID, ScopeFinancialProfile, 30)
	assert.NoError(t, err)

	// 4. Retrieve Profile via UHEID
	t.Log("Retrieving profile...")
	profile, err := svc.GetUHEIDProfile(*registered.UHEID, "hosp-key-123", ScopeFinancialProfile)
	assert.NoError(t, err)
	assert.Equal(t, *registered.UHEID, profile["uheid"])
	assert.Equal(t, registered.FullName, profile["full_name"])

	// 5. Check blocked access (wrong key)
	t.Log("Checking blocked access...")
	_, err = svc.GetUHEIDProfile(*registered.UHEID, "wrong-key", ScopeFinancialProfile)
	assert.Error(t, err)
	t.Log("Test completed successfully")
}

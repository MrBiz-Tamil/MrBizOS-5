package document

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Minimal stand-in for the EMR consent table the vault reads.
type testGrant struct {
	ID        uint `gorm:"primaryKey"`
	DoctorID  uint
	PatientID uint
	ExpiresAt time.Time
}

func (testGrant) TableName() string { return "emr_access_grants" }

// doctorHasActiveGrant is the consent core behind patient-document access: a
// doctor may read a patient's vault only while an unexpired grant exists.
func TestDoctorHasActiveGrant(t *testing.T) {
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&testGrant{}))

	now := time.Now()
	require.NoError(t, db.Create(&testGrant{DoctorID: 1, PatientID: 2, ExpiresAt: now.Add(time.Hour)}).Error)   // active
	require.NoError(t, db.Create(&testGrant{DoctorID: 3, PatientID: 2, ExpiresAt: now.Add(-time.Hour)}).Error)  // expired

	h := NewHandler(db, nil)

	assert.True(t, h.doctorHasActiveGrant(1, 2), "active grant → access")
	assert.False(t, h.doctorHasActiveGrant(3, 2), "expired grant → no access")
	assert.False(t, h.doctorHasActiveGrant(9, 2), "no grant → no access")
	assert.False(t, h.doctorHasActiveGrant(1, 5), "grant is patient-specific")
}

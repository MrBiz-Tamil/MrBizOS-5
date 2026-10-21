package mfa_test

import (
	"fmt"
	"drayya-core/internal/services/notification"
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestCheckMFAJobs(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	var db *gorm.DB
	var err error

	if dsn == "" {
		// Try local sqlite if postgres not set
		db, err = gorm.Open(sqlite.Open("../../../Dr.Ayya.db"), &gorm.Config{})
	} else {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	}

	if err != nil {
		t.Fatal(err)
	}

	var jobs []notification.NotificationJob
	err = db.Where("job_type = ?", "mfa_otp").Order("created_at DESC").Limit(10).Find(&jobs).Error
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("Recent MFA OTP Jobs:")
	for _, job := range jobs {
		fmt.Printf("ID: %d | Recipient: %s | Status: %s | Attempts: %d | Error: %s | CreatedAt: %v\n",
			job.ID, job.Recipient, job.Status, job.Attempts, job.LastError, job.CreatedAt)
	}
}

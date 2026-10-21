package e2e

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"testing"
)

func TestTableCreation(t *testing.T) {
	if database.DB == nil {
		t.Fatal("Database is nil")
	}

	// Check if table exists
	hasTable := database.DB.Migrator().HasTable(&finance.LedgerAccount{})
	t.Logf("Has LedgerAccount table: %v", hasTable)

	if !hasTable {
		err := database.DB.AutoMigrate(&finance.LedgerAccount{})
		if err != nil {
			t.Fatalf("AutoMigrate failed: %v", err)
		}
		hasTable = database.DB.Migrator().HasTable(&finance.LedgerAccount{})
		t.Logf("Has LedgerAccount table after migrate: %v", hasTable)
	}
}

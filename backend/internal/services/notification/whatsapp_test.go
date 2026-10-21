package notification

import (
	"testing"

	"drayya-core/internal/database"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestResolveDBWhatsAppTemplate(t *testing.T) {
	// Setup in-memory sqlite for this test
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Keep copy of old DB and restore at end
	oldDB := database.DB
	database.DB = db
	defer func() { database.DB = oldDB }()

	// Migrate dbWATemplate
	err = db.AutoMigrate(&dbWATemplate{})
	assert.NoError(t, err)

	fallback := []string{"FallbackName", "FallbackRole"}
	vars := map[string]string{"name": "Alice", "role": "doctor"}

	// Test case 1: DB missing template - should return fallback
	res := resolveDBWhatsAppTemplate("welcome_drayya", fallback, vars)
	assert.Equal(t, fallback, res)

	// Test case 2: Seed template in DB with variables order: name,role
	tmpl := dbWATemplate{
		Type:      "WhatsApp",
		Title:     "welcome_drayya",
		Content:   "Welcome {{name}}! Your Dr.Ayya {{role}} account is ready.",
		Variables: "name,role",
	}
	err = db.Create(&tmpl).Error
	assert.NoError(t, err)

	res = resolveDBWhatsAppTemplate("welcome_drayya", fallback, vars)
	assert.Equal(t, []string{"Alice", "doctor"}, res)

	// Test case 3: Update variables order in DB to: role,name (dynamic re-mapping check)
	tmpl.Variables = "role,name"
	err = db.Save(&tmpl).Error
	assert.NoError(t, err)

	res = resolveDBWhatsAppTemplate("welcome_drayya", fallback, vars)
	assert.Equal(t, []string{"doctor", "Alice"}, res)

	// Test case 4: Missing variable in vars map (should yield empty string for that position)
	res = resolveDBWhatsAppTemplate("welcome_drayya", fallback, map[string]string{"role": "doctor"})
	assert.Equal(t, []string{"doctor", ""}, res)
}

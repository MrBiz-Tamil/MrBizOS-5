package database_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"drayya-core/internal/database"
	"drayya-core/internal/server"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type TableSchema struct {
	Name string `json:"name"`
	SQL  string `json:"sql"`
}

// normalizeSQL sorts CONSTRAINT clauses inside a CREATE TABLE statement so
// that non-deterministic foreign-key ordering from GORM/SQLite does not cause
// spurious snapshot diffs.
func normalizeSQL(sql string) string {
	// Match the column/constraint list inside CREATE TABLE (...).
	re := regexp.MustCompile(`(?i)(CREATE TABLE[^(]+\()(.+)(\))$`)
	m := re.FindStringSubmatch(strings.TrimSpace(sql))
	if m == nil {
		return sql
	}

	prefix := m[1]
	body := m[2]
	suffix := m[3]

	// Split on top-level commas (not inside nested parens).
	parts := splitTopLevel(body)
	constraints := []string{}
	columns := []string{}
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if strings.HasPrefix(strings.ToUpper(trimmed), "CONSTRAINT") {
			constraints = append(constraints, trimmed)
		} else {
			columns = append(columns, trimmed)
		}
	}
	sort.Strings(constraints)

	all := append(columns, constraints...)
	return prefix + strings.Join(all, ",") + suffix
}

// splitTopLevel splits s on commas that are not inside parentheses.
func splitTopLevel(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i, c := range s {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// TestSchemaSnapshot ensures that GORM models exactly match the expected database schema.
// Run with UPDATE_SNAPSHOT=1 to update the snapshot file.
func TestSchemaSnapshot(t *testing.T) {
	// Setup an in-memory database specifically for this test
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// Temporarily override the global DB for migrations
	originalDB := database.DB
	database.DB = db
	defer func() { database.DB = originalDB }()

	// Run all migrations to generate the schema
	require.NotPanics(t, func() {
		server.RunMigrations()
	}, "RunMigrations should not panic or fail")

	// Dump the schema from SQLite
	var tables []TableSchema
	err = db.Raw(`SELECT name, sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name;`).Scan(&tables).Error
	require.NoError(t, err)

	// Normalize each table's SQL so CONSTRAINT ordering is stable
	for i := range tables {
		tables[i].SQL = normalizeSQL(tables[i].SQL)
	}

	// Format as JSON
	actualBytes, err := json.MarshalIndent(tables, "", "  ")
	require.NoError(t, err)

	snapshotPath := filepath.Join("testdata", "schema_snapshot.json")

	// If UPDATE_SNAPSHOT is set, write the new snapshot
	if os.Getenv("UPDATE_SNAPSHOT") == "1" {
		err = os.MkdirAll("testdata", 0755)
		require.NoError(t, err)
		err = os.WriteFile(snapshotPath, actualBytes, 0644)
		require.NoError(t, err)
		t.Logf("Snapshot updated successfully at %s", snapshotPath)
		return
	}

	// Compare against existing snapshot
	expectedBytes, err := os.ReadFile(snapshotPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("Snapshot file %s does not exist. Run test with UPDATE_SNAPSHOT=1 to create it.", snapshotPath)
		}
		t.Fatalf("Failed to read snapshot file: %v", err)
	}

	// Compare raw JSON bytes stringified for nice diffing in testify
	assert.Equal(t, string(expectedBytes), string(actualBytes), "Database schema has drifted! If this is intentional, run tests with UPDATE_SNAPSHOT=1")
}

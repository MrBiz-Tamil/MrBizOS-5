package admin_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"drayya-core/internal/database"
	"drayya-core/internal/tests/common"

	"github.com/stretchr/testify/assert"
)

func TestGetUsers(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	// Seed Data
	admin := common.CreateTestUser(db, "admin")
	common.CreateTestUser(db, "patient")

	// Create Request
	req := httptest.NewRequest("GET", "/api/admin/users", nil)
	token := common.GenerateTestToken(admin)
	req.Header.Set("Authorization", "Bearer "+token)

	// Execute Request
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Parse Response
	body, _ := io.ReadAll(resp.Body)
	var result []interface{}
	json.Unmarshal(body, &result)

	// Assertions
	assert.Len(t, result, 2)
}

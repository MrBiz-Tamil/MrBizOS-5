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

func TestGetSystemOverview(t *testing.T) {
	app := common.SetupApp()
	db := database.DB

	// Seed Data
	admin := common.CreateTestUser(db, "admin")
	common.CreateTestUser(db, "doctor")
	common.CreateTestUser(db, "patient")

	// Create Appointments
	common.CreateTestAppointment(db, "confirmed") // Should count as ongoing if time matches today
	common.CreateTestAppointment(db, "completed") // Count as today's appointment

	// Create Payments
	common.CreateTestPayment(db, 100.0, "completed")
	common.CreateTestPayment(db, 50.0, "completed")
	common.CreateTestPayment(db, 20.0, "pending")

	// Create Request
	req := httptest.NewRequest("GET", "/api/admin/dashboard/overview", nil)
	token := common.GenerateTestToken(admin)
	req.Header.Set("Authorization", "Bearer "+token)

	// Execute Request
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Parse Response
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	// Assertions
	// Total Users: 3 (admin + doctor + patient) created above + createTestUser might create duplicates if not careful?
	// Ah, common.CreateTestUser creates new user each call.
	// So 3 users minimum.
	assert.Equal(t, float64(3), result["total_users"])

	// Revenue: 100 + 50 = 150 (pending 20 ignored)
	assert.Equal(t, float64(150), result["revenue_today"])

	// Ongoing Consultations: 1 confirmed
	assert.Equal(t, float64(1), result["ongoing_consultations"])

	// Appointments Today: 2 (confirmed + completed)
	assert.Equal(t, float64(2), result["appointments_today"])
}

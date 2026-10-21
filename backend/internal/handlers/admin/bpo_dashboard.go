package admin

import (
	"github.com/gofiber/fiber/v2"
)

type BPODashboardOverview struct {
	ActiveTickets   int     `json:"active_tickets"`    // Mocked
	AvgResponseTime float64 `json:"avg_response_time"` // Mocked (minutes)
	ResolvedToday   int     `json:"resolved_today"`    // Mocked
	CustomerSatis   float64 `json:"customer_satis"`    // Mocked (percent)
}

func GetBPODashboardOverview(c *fiber.Ctx) error {
	var overview BPODashboardOverview

	// Currently no ticket model exists, so these are mocks to support the UI
	overview.ActiveTickets = 15
	overview.AvgResponseTime = 4.2
	overview.ResolvedToday = 28
	overview.CustomerSatis = 4.8

	return c.JSON(overview)
}

func GetBPOAgentPerformance(c *fiber.Ctx) error {
	// Mock agent list
	agents := []fiber.Map{
		{"name": "Agent Alpha", "score": 98, "status": "online"},
		{"name": "Agent Beta", "score": 85, "status": "on_call"},
		{"name": "Agent Gamma", "score": 92, "status": "offline"},
	}
	return c.JSON(agents)
}

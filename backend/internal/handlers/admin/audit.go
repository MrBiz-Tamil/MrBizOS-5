package admin

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/franchise"
	"drayya-core/internal/domain/hospital"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/utils/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetSystemActivityLogs retrieves all system logs with pagination
func GetSystemActivityLogs(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "100"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	var logs []identity.ActivityLog
	database.DB.Preload("User").
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&logs)

	return c.JSON(logs)
}

// GetLedgerAuditTrail visualizes the entries for a specific transaction
func GetLedgerAuditTrail(c *fiber.Ctx) error {
	txnID := c.Params("transaction_id")

	uid, err := uuid.Parse(txnID)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_TRANSACTION_UUID", "Invalid transaction UUID")
	}

	var entries []finance.LedgerEntry
	database.DB.Where("transaction_id = ?", uid).
		Preload("Account"). // Needs Account field in models.LedgerEntry
		Find(&entries)

	return c.JSON(entries)
}

// GetEcosystemStats returns high-level stats for franchises and hospitals
func GetEcosystemStats(c *fiber.Ctx) error {
	var franchiseCount, hospitalCount int64
	database.DB.Model(&franchise.Franchise{}).Count(&franchiseCount)
	database.DB.Model(&hospital.Hospital{}).Count(&hospitalCount)

	return c.JSON(fiber.Map{
		"franchises": franchiseCount,
		"hospitals":  hospitalCount,
	})
}

package admin

import (
	"encoding/json"
	"drayya-core/internal/utils/config"
	"drayya-core/internal/utils/response"
	"os"
	"sync"

	"github.com/gofiber/fiber/v2"
)

// Settings Structure (Simplified for migration)
type Settings struct {
	General       map[string]interface{} `json:"general"`
	Notifications map[string]interface{} `json:"notifications"`
	Security      map[string]interface{} `json:"security"`
	Payments      map[string]interface{} `json:"payments"`
	Growth        map[string]interface{} `json:"growth"`
}

var (
	settingsFile = "system_settings.json"
	settingsLock sync.RWMutex
)

// Load defaults
func defaultSettings() Settings {
	return Settings{
		General: map[string]interface{}{
			"siteName":        "Dr.Ayya",
			"siteDescription": "Your trusted assessment platform",
			"timezone":        "Asia/Kolkata",
		},
		Notifications: map[string]interface{}{
			"emailNotifications": true,
			"smsNotifications":   true,
		},
		Security: map[string]interface{}{
			"twoFactorAuth":  false,
			"sessionTimeout": 30,
		},
		Payments: map[string]interface{}{
			"currency":           "INR",
			"taxRate":            18.0,
			"platformCommission": 15.0,
		},
		Growth: map[string]interface{}{
			"dailyFreeLimit":   2,
			"patientFreeLimit": 1,
			"subsidyAmount":    40000,
			"enableAlgorithmBoost": true,
		},
	}
}

func GetSettings(c *fiber.Ctx) error {
	settingsLock.RLock()
	defer settingsLock.RUnlock()

	// Try to read file
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		// Return defaults if file missing
		return c.JSON(defaultSettings())
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return c.JSON(defaultSettings())
	}

	return c.JSON(settings)
}

func UpdateSettings(c *fiber.Ctx) error {
	var newSettings Settings
	if err := c.BodyParser(&newSettings); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_JSON", "Invalid JSON")
	}

	settingsLock.Lock()
	defer settingsLock.Unlock()

	data, err := json.MarshalIndent(newSettings, "", "  ")
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SETTINGS_SERIALIZE_FAILED", "Failed to serialize settings")
	}

	if err := os.WriteFile(settingsFile, data, 0644); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SETTINGS_SAVE_FAILED", "Failed to save settings")
	}

	config.ClearCache()

	return c.JSON(fiber.Map{"message": "Settings updated successfully", "settings": newSettings})
}

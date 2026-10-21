package config

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

type SystemSettings struct {
	General       map[string]interface{} `json:"general"`
	Notifications map[string]interface{} `json:"notifications"`
	Security      map[string]interface{} `json:"security"`
	Payments      map[string]interface{} `json:"payments"`
	Growth        map[string]interface{} `json:"growth"`
	Regional      map[string]interface{} `json:"regional"`
}

// AppTimezone returns the configured IANA timezone string.
// Resolution order: APP_TIMEZONE env var → system_settings.json regional.timezone → "Asia/Kolkata".
func AppTimezone() string {
	if tz := os.Getenv("APP_TIMEZONE"); tz != "" {
		return tz
	}
	s := Load()
	if s.Regional != nil {
		if tz, ok := s.Regional["timezone"].(string); ok && tz != "" {
			return tz
		}
	}
	return "Asia/Kolkata"
}

// AppCurrency returns the ISO 4217 default currency code.
// Resolution order: APP_CURRENCY env var → system_settings.json regional.currency → "INR".
func AppCurrency() string {
	if c := os.Getenv("APP_CURRENCY"); c != "" {
		return c
	}
	s := Load()
	if s.Regional != nil {
		if c, ok := s.Regional["currency"].(string); ok && c != "" {
			return c
		}
	}
	return "INR"
}

// LoadTimezone returns the *time.Location for AppTimezone, falling back to UTC on error.
func LoadTimezone() *time.Location {
	loc, err := time.LoadLocation(AppTimezone())
	if err != nil {
		return time.UTC
	}
	return loc
}

var (
	settingsFile = "system_settings.json"
	cached       *SystemSettings
	mu           sync.RWMutex
)

func Load() *SystemSettings {
	mu.RLock()
	if cached != nil {
		defer mu.RUnlock()
		return cached
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		return &SystemSettings{}
	}

	var s SystemSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return &SystemSettings{}
	}

	cached = &s
	return cached
}

func GetGrowthSetting(key string, defaultValue interface{}) interface{} {
	s := Load()
	if s.Growth != nil {
		if val, ok := s.Growth[key]; ok {
			return val
		}
	}
	return defaultValue
}

func GetPaymentSetting(key string, defaultValue interface{}) interface{} {
	s := Load()
	if s.Payments != nil {
		if val, ok := s.Payments[key]; ok {
			return val
		}
	}
	return defaultValue
}
func ClearCache() {
	mu.Lock()
	defer mu.Unlock()
	cached = nil
}

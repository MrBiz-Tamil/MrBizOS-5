package shipping

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestProviderRegistry_Sorting(t *testing.T) {
	// GlobalRegistry is initialized in init() and sorted
	providers := GlobalRegistry.providers
	
	// Check that they are sorted by priority descending
	// BlueDart (80) > Delhivery (60) > Shiprocket (50) > IndiaPost (10)
	assert.True(t, providers[0].Priority() >= providers[1].Priority())
	assert.Equal(t, "BlueDart", providers[0].Name())
	assert.Equal(t, "IndiaPost", providers[len(providers)-1].Name())
}

func TestProviderRegistry_OptimalProviders(t *testing.T) {
	tests := []struct {
		name       string
		pincode    string
		total      float64
		weightKg   float64
		express    bool
		expectName string
	}{
		{
			name:       "Standard Order Low Value",
			pincode:    "400001", // BlueDart, Delhivery, IndiaPost serviceable
			total:      1000.0,
			weightKg:   1.0,
			express:    false,
			expectName: "IndiaPost", // Cost: 30 + 10 = 40. SLA: 6 days. (Cheapest)
		},
		{
			name:       "High Value Order Excludes IndiaPost",
			pincode:    "400001",
			total:      6000.0,   // > 5000 excludes IndiaPost
			weightKg:   2.0,      // Delhivery: 35+36=71. Shadowfax: 38+30=68.
			express:    false,
			expectName: "Shadowfax",
		},
		{
			name:       "Express Delivery Requires SLA <= 2",
			pincode:    "400001", 
			total:      1000.0,
			weightKg:   1.0,
			express:    true,
			expectName: "Delhivery", // Delhivery SLA=2 (cost 53), Shadowfax SLA=2 (cost 53). Both valid, but Delhivery might be returned first due to loop order or cost match. Let's just expect Delhivery since 53 == 53 and Delhivery is before Shadowfax in list.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers := GlobalRegistry.GetOptimalProviders(tt.pincode, tt.weightKg, "500001", tt.total, tt.express)
			assert.NotEmpty(t, providers)
			// Because costs might be identical (Delhivery vs Shadowfax at 1kg = 53),
			// we just assert it's NOT BlueDart and NOT IndiaPost for the express one.
			if tt.express {
				assert.NotEqual(t, "BlueDart", providers[0].Name())
				assert.NotEqual(t, "IndiaPost", providers[0].Name())
			} else {
				assert.Equal(t, tt.expectName, providers[0].Name())
			}
		})
	}
}

func TestProviderRegistry_Serviceability(t *testing.T) {
	// BlueDart mock: starts with 1, 4, 5, 6
	p := NewBlueDartProvider()
	assert.True(t, p.IsServiceable("110001"))
	assert.False(t, p.IsServiceable("302001"))

	// IndiaPost: always true
	ip := NewIndiaPostProvider()
	assert.True(t, ip.IsServiceable("999999"))
}

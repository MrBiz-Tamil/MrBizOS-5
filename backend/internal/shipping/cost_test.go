package shipping

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestProviderRegistry_CostSorting(t *testing.T) {
	// Origin: Bangalore, Destination: Delhi, Weight: 2kg
	origin := "560001"
	dest := "110001"
	weight := 2.0

	providers := GlobalRegistry.GetProvidersByCost(dest, weight, origin)
	
	// Ensure list is not empty
	assert.NotEmpty(t, providers)

	// Check that they are sorted by cost ascending
	for i := 0; i < len(providers)-1; i++ {
		c1, _ := providers[i].CalculateCost(weight, origin, dest)
		c2, _ := providers[i+1].CalculateCost(weight, origin, dest)
		assert.True(t, c1 <= c2, "Providers should be sorted by cost ascending")
	}

	// Verify specific mock costs (based on providers.go implementation)
	// BlueDart: 150 + 2*40 = 230
	// Delhivery: 80 + 2*30 = 140
	// Shiprocket: 40 + 2*25 = 90
	// IndiaPost: 30 + 2*15 = 60
	// DTDC: 60 + 2*20 = 100
	// Shadowfax: 50 + 2*18 = 86
	
	// Expected Order by Cost: IndiaPost (60), Shadowfax (86), Shiprocket (90), DTDC (100), Delhivery (140), BlueDart (230)
	assert.Equal(t, "IndiaPost", providers[0].Name())
	assert.Equal(t, "Shadowfax", providers[1].Name())
}

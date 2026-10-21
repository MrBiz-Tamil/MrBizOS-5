package shipping

import (
	"sort"
)

// ProviderRegistry manages the list of available 3PL providers.
type ProviderRegistry struct {
	providers []ShippingProvider
}

// GlobalRegistry is the default registry used by the dispatch system.
var GlobalRegistry *ProviderRegistry

func init() {
	GlobalRegistry = &ProviderRegistry{
		providers: []ShippingProvider{
			NewBlueDartProvider(),
			NewDTDCProvider(),
			NewDelhiveryProvider(),
			NewShadowfaxProvider(),
			NewShiprocketProvider(),
			NewIndiaPostProvider(),
		},
	}
	// Sort by priority descending (highest first)
	sort.Slice(GlobalRegistry.providers, func(i, j int) bool {
		return GlobalRegistry.providers[i].Priority() > GlobalRegistry.providers[j].Priority()
	})
}

// GetServiceableProviders returns a list of providers that can service the given pincode.
func (r *ProviderRegistry) GetServiceableProviders(pincode string) []ShippingProvider {
	var available []ShippingProvider
	for _, p := range r.providers {
		if p.IsServiceable(pincode) {
			available = append(available, p)
		}
	}
	return available
}

// GetProvidersByCost returns serviceable providers sorted by estimated cost (lowest first).
func (r *ProviderRegistry) GetProvidersByCost(pincode string, weightKg float64, pickupPincode string) []ShippingProvider {
	serviceable := r.GetServiceableProviders(pincode)
	if len(serviceable) <= 1 {
		return serviceable
	}

	type providerCost struct {
		provider ShippingProvider
		cost     float64
	}
	var costs []providerCost

	for _, p := range serviceable {
		cost, err := p.CalculateCost(weightKg, pickupPincode, pincode)
		if err != nil {
			cost = 999999 // fallback for error
		}
		costs = append(costs, providerCost{p, cost})
	}

	sort.Slice(costs, func(i, j int) bool {
		return costs[i].cost < costs[j].cost
	})

	var sorted []ShippingProvider
	for _, pc := range costs {
		sorted = append(sorted, pc.provider)
	}
	return sorted
}

// GetOptimalProviders implements the SLA-Aware Cost Routing matrix.
// It balances order value, delivery speed, and cost to return a sorted list of optimal couriers.
func (r *ProviderRegistry) GetOptimalProviders(pincode string, weightKg float64, pickupPincode string, orderTotalINR float64, isExpress bool) []ShippingProvider {
	serviceable := r.GetServiceableProviders(pincode)
	if len(serviceable) == 0 {
		return nil
	}

	type candidate struct {
		provider ShippingProvider
		cost     float64
		slaDays  int
	}

	var candidates []candidate
	for _, p := range serviceable {
		cost, err := p.CalculateCost(weightKg, pickupPincode, pincode)
		if err != nil {
			continue // skip failing providers
		}
		sla := p.EstimatedDays(pickupPincode, pincode)

		// Delivery Speed Rule (Express must be <= 2 days)
		if isExpress && sla > 2 {
			continue
		}

		// Order Value Rule (> ₹5000 cannot use IndiaPost due to lack of real-time tracking)
		if orderTotalINR > 5000.0 && p.Name() == "IndiaPost" {
			continue
		}

		// Standard Rule (Filter out ridiculously slow providers > 7 days)
		if sla > 7 {
			continue
		}

		candidates = append(candidates, candidate{provider: p, cost: cost, slaDays: sla})
	}

	if len(candidates) == 0 {
		// Fallback: If strict rules filtered everyone out, just return the cheapest serviceable
		return r.GetProvidersByCost(pincode, weightKg, pickupPincode)
	}

	// Sort strictly by cost (cheapest wins, since SLA constraints are already satisfied)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].cost < candidates[j].cost
	})

	var result []ShippingProvider
	for _, c := range candidates {
		result = append(result, c.provider)
	}
	return result
}

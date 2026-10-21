package hr

// Pricing-tier recommendation for freelance panel doctors.
//
// This is ADVISORY only — it never changes a doctor's actual rates. It surfaces a
// suggested tier from performance signals so HR can decide whether to promote a
// doctor (via the existing pricing tools) or review an underperformer. Keeping it
// read-only avoids silently moving money on a metric swing.

const (
	TierNew         = "new"          // too little history to judge
	TierNeedsReview = "needs_review" // low rating — review before keeping on panel
	TierStandard    = "standard"
	TierPremium     = "premium" // high rating + reliability + volume
)

// SuggestPricingTier maps a doctor's performance to a recommended tier.
//
//	rating         average patient rating (0–5)
//	completionRate fraction of accepted consults completed; accepts either 0–1 or
//	               0–100 (percentages are normalised)
//	totalConsults  lifetime completed consultations
func SuggestPricingTier(rating, completionRate float64, totalConsults int) string {
	// Normalise completion rate to a 0–1 fraction.
	if completionRate > 1 {
		completionRate = completionRate / 100
	}

	switch {
	case totalConsults < 20:
		return TierNew
	case rating > 0 && rating < 3.5:
		return TierNeedsReview
	case rating >= 4.5 && completionRate >= 0.9 && totalConsults >= 100:
		return TierPremium
	default:
		return TierStandard
	}
}

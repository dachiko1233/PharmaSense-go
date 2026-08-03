package domain

// Unlimited marks a plan quota that is not capped.
const Unlimited = -1

// MaxInventoryBatches returns the maximum number of inventory_batches rows a
// pharmacy on the given plan may hold. Only the Free plan is capped; paid
// plans (pro, chain) are unlimited. Unknown/blank plans are treated as Free,
// which is the safe default (the pharmacies.plan column defaults to 'free').
//
// This is the single source of truth for the *enforced* Free inventory limit;
// the pricing-page copy must be kept in sync with it.
func MaxInventoryBatches(plan string) int {
	switch plan {
	case "pro", "chain":
		return Unlimited
	default: // "free" and anything unrecognized
		return 500
	}
}

// MaxEmailAlertsPerMonth returns how many alert emails (the daily digest) a
// pharmacy on the given plan may receive per calendar month. Only the Free
// plan is capped; paid plans (pro, chain) are unlimited. Unknown/blank plans
// are treated as Free (safe default).
//
// Single source of truth for the enforced Free email-alert limit; keep the
// pricing-page copy ("Email alerts (10/month)") in sync with it.
func MaxEmailAlertsPerMonth(plan string) int {
	switch plan {
	case "pro", "chain":
		return Unlimited
	default: // "free" and anything unrecognized
		return 10
	}
}

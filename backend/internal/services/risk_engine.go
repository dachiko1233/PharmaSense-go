package services

import (
	"math"
	"time"

	"pharmasense/internal/domain"
)

// RiskInput holds all data needed to calculate risk for one batch.
type RiskInput struct {
	CurrentQuantity int
	ExpiryDate      time.Time
	AvgDailySales   float64 // rolling 90-day average
	PurchasePrice   float64
	Today           time.Time
}

// RiskResult holds the calculated risk metrics.
type RiskResult struct {
	RiskLevel              string
	DaysUntilExpiry        int
	ExpectedSales          int
	EstimatedSurplus       int
	EstimatedLoss          float64
	SuggestedDiscountPct   int
}

// CalculateRisk computes the risk level and associated metrics for a single batch.
func CalculateRisk(input RiskInput) RiskResult {
	today := input.Today
	if today.IsZero() {
		today = time.Now().UTC().Truncate(24 * time.Hour)
	}

	// Negative means already expired; do NOT clamp to 0.
	daysUntilExpiry := int(math.Ceil(input.ExpiryDate.Sub(today).Hours() / 24))

	var expectedSales, surplus int
	if daysUntilExpiry > 0 {
		expectedSales = int(math.Round(input.AvgDailySales * float64(daysUntilExpiry)))
		surplus = input.CurrentQuantity - expectedSales
		if surplus < 0 {
			surplus = 0
		}
	} else {
		// Expired or expires today: no future sales possible; entire stock is at risk.
		expectedSales = 0
		surplus = input.CurrentQuantity
	}

	var riskLevel string
	var suggestedDiscount int
	var estimatedLoss float64

	switch {
	case daysUntilExpiry < 0:
		riskLevel = domain.RiskLevelExpired
		estimatedLoss = float64(input.CurrentQuantity) * input.PurchasePrice

	case daysUntilExpiry <= 30 && surplus > 0:
		riskLevel = domain.RiskLevelCritical
		suggestedDiscount = 40
		estimatedLoss = float64(surplus) * input.PurchasePrice

	case daysUntilExpiry <= 90 && float64(surplus) > input.AvgDailySales*float64(daysUntilExpiry)*0.5:
		riskLevel = domain.RiskLevelHigh
		suggestedDiscount = 20
		estimatedLoss = float64(surplus) * input.PurchasePrice

	case daysUntilExpiry <= 180 && float64(surplus) > input.AvgDailySales*float64(daysUntilExpiry)*0.3:
		riskLevel = domain.RiskLevelMedium
		suggestedDiscount = 10

	default:
		riskLevel = domain.RiskLevelLow
	}

	return RiskResult{
		RiskLevel:            riskLevel,
		DaysUntilExpiry:      daysUntilExpiry,
		ExpectedSales:        expectedSales,
		EstimatedSurplus:     surplus,
		EstimatedLoss:        estimatedLoss,
		SuggestedDiscountPct: suggestedDiscount,
	}
}

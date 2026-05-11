package services

import (
	"testing"
	"time"

	"pharmasense/internal/domain"
)

var today = time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)

func TestCalculateRisk_Critical(t *testing.T) {
	input := RiskInput{
		CurrentQuantity: 100,
		ExpiryDate:      today.AddDate(0, 0, 15), // 15 days — within 30
		AvgDailySales:   1.0,                     // expects 15 sold, surplus = 85
		PurchasePrice:   5.00,
		Today:           today,
	}
	result := CalculateRisk(input)

	if result.RiskLevel != domain.RiskLevelCritical {
		t.Errorf("expected CRITICAL, got %s", result.RiskLevel)
	}
	if result.DaysUntilExpiry != 15 {
		t.Errorf("expected 15 days, got %d", result.DaysUntilExpiry)
	}
	if result.EstimatedSurplus != 85 {
		t.Errorf("expected surplus 85, got %d", result.EstimatedSurplus)
	}
	if result.EstimatedLoss != 85*5.00 {
		t.Errorf("expected loss %.2f, got %.2f", 85*5.00, result.EstimatedLoss)
	}
	if result.SuggestedDiscountPct == 0 {
		t.Error("expected non-zero suggested discount for CRITICAL")
	}
}

func TestCalculateRisk_High(t *testing.T) {
	input := RiskInput{
		CurrentQuantity: 200,
		ExpiryDate:      today.AddDate(0, 0, 60), // 60 days — within 90
		AvgDailySales:   1.0,                     // expects 60 sold, surplus = 140 > 60*0.5=30
		PurchasePrice:   3.00,
		Today:           today,
	}
	result := CalculateRisk(input)

	if result.RiskLevel != domain.RiskLevelHigh {
		t.Errorf("expected HIGH, got %s", result.RiskLevel)
	}
	if result.EstimatedLoss <= 0 {
		t.Error("expected positive estimated loss for HIGH")
	}
}

func TestCalculateRisk_Medium(t *testing.T) {
	input := RiskInput{
		CurrentQuantity: 50,
		ExpiryDate:      today.AddDate(0, 0, 120), // 120 days — within 180
		AvgDailySales:   0.2,                      // expects 24 sold, surplus = 26 > 24*0.3=7.2
		PurchasePrice:   2.00,
		Today:           today,
	}
	result := CalculateRisk(input)

	if result.RiskLevel != domain.RiskLevelMedium {
		t.Errorf("expected MEDIUM, got %s", result.RiskLevel)
	}
	// medium doesn't carry estimated loss
	if result.EstimatedLoss != 0 {
		t.Errorf("expected zero loss for MEDIUM, got %.2f", result.EstimatedLoss)
	}
}

func TestCalculateRisk_Low(t *testing.T) {
	input := RiskInput{
		CurrentQuantity: 10,
		ExpiryDate:      today.AddDate(0, 0, 365), // 1 year out
		AvgDailySales:   5.0,                      // will sell all way before expiry
		PurchasePrice:   1.00,
		Today:           today,
	}
	result := CalculateRisk(input)

	if result.RiskLevel != domain.RiskLevelLow {
		t.Errorf("expected LOW, got %s", result.RiskLevel)
	}
	if result.EstimatedLoss != 0 {
		t.Errorf("expected zero loss for LOW, got %.2f", result.EstimatedLoss)
	}
}

func TestCalculateRisk_ZeroSales(t *testing.T) {
	input := RiskInput{
		CurrentQuantity: 50,
		ExpiryDate:      today.AddDate(0, 0, 25),
		AvgDailySales:   0,
		PurchasePrice:   10.00,
		Today:           today,
	}
	result := CalculateRisk(input)
	// Zero sales + items expiring in 25 days = CRITICAL
	if result.RiskLevel != domain.RiskLevelCritical {
		t.Errorf("expected CRITICAL with zero sales, got %s", result.RiskLevel)
	}
}

func TestCalculateRisk_Expired(t *testing.T) {
	input := RiskInput{
		CurrentQuantity: 10,
		ExpiryDate:      today.AddDate(0, 0, -5), // already expired
		AvgDailySales:   1.0,
		PurchasePrice:   5.00,
		Today:           today,
	}
	result := CalculateRisk(input)
	if result.DaysUntilExpiry != 0 {
		t.Errorf("expected 0 days for expired, got %d", result.DaysUntilExpiry)
	}
	if result.RiskLevel != domain.RiskLevelCritical {
		t.Errorf("expected CRITICAL for expired, got %s", result.RiskLevel)
	}
}

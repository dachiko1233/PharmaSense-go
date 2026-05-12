package billing

import (
	"context"
	"log/slog"
)

type MockBillingService struct {
	frontendOrigin string
}

func NewMockBillingService(frontendOrigin string) *MockBillingService {
	return &MockBillingService{frontendOrigin: frontendOrigin}
}

func (m *MockBillingService) CreateCheckoutSession(ctx context.Context, customerID, priceID, pharmacyID, plan string) (string, error) {
	slog.Info("[MOCK STRIPE] checkout session created",
		"customer_id", customerID,
		"price_id", priceID,
		"pharmacy_id", pharmacyID,
		"plan", plan,
	)
	return m.frontendOrigin + "/mock-stripe/checkout?plan=" + plan, nil
}

func (m *MockBillingService) CreatePortalSession(ctx context.Context, customerID string) (string, error) {
	slog.Info("[MOCK STRIPE] portal session created", "customer_id", customerID)
	return m.frontendOrigin + "/mock-stripe/portal", nil
}

// BillingProvider is the interface both real and mock implementations satisfy.
type BillingProvider interface {
	CreateCheckoutSession(ctx context.Context, customerID, priceID, pharmacyID, plan string) (string, error)
	CreatePortalSession(ctx context.Context, customerID string) (string, error)
}

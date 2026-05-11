package billing

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/stripe/stripe-go/v79"
	checkoutsession "github.com/stripe/stripe-go/v79/checkout/session"
	portalsession "github.com/stripe/stripe-go/v79/billingportal/session"
	stripewebhook "github.com/stripe/stripe-go/v79/webhook"
)

type StripeService struct {
	secretKey      string
	webhookSecret  string
	pricePro       string
	priceChain     string
	frontendOrigin string
}

func NewStripeService(secretKey, webhookSecret, pricePro, priceChain, frontendOrigin string) *StripeService {
	stripe.Key = secretKey
	return &StripeService{
		secretKey:      secretKey,
		webhookSecret:  webhookSecret,
		pricePro:       pricePro,
		priceChain:     priceChain,
		frontendOrigin: frontendOrigin,
	}
}

func (s *StripeService) CreateCheckoutSession(ctx context.Context, customerID, priceID, pharmacyID string) (string, error) {
	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String("subscription"),
		SuccessURL: stripe.String(s.frontendOrigin + "/billing?success=true"),
		CancelURL:  stripe.String(s.frontendOrigin + "/billing?canceled=true"),
		Metadata: map[string]string{
			"pharmacy_id": pharmacyID,
		},
	}

	sess, err := checkoutsession.New(params)
	if err != nil {
		return "", fmt.Errorf("create checkout session: %w", err)
	}
	return sess.URL, nil
}

func (s *StripeService) CreatePortalSession(ctx context.Context, customerID string) (string, error) {
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(s.frontendOrigin + "/billing"),
	}
	sess, err := portalsession.New(params)
	if err != nil {
		return "", fmt.Errorf("create portal session: %w", err)
	}
	return sess.URL, nil
}

func (s *StripeService) ConstructEvent(body []byte, sigHeader string) (stripe.Event, error) {
	event, err := stripewebhook.ConstructEvent(body, sigHeader, s.webhookSecret)
	if err != nil {
		return stripe.Event{}, fmt.Errorf("construct event: %w", err)
	}
	return event, nil
}

func (s *StripeService) ReadBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return body, nil
}

func (s *StripeService) PriceIDForPlan(plan string) string {
	switch plan {
	case "pro":
		return s.pricePro
	case "chain":
		return s.priceChain
	default:
		return ""
	}
}

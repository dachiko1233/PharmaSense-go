package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	dodoTestBaseURL = "https://test.dodopayments.com"
	dodoLiveBaseURL = "https://live.dodopayments.com"
)

type DodoService struct {
	apiKey         string
	webhookSecret  string
	baseURL        string
	productPro     string
	productChain   string
	frontendOrigin string
	httpClient     *http.Client
}

func NewDodoService(apiKey, webhookSecret, productPro, productChain, frontendOrigin string, live bool) *DodoService {
	baseURL := dodoTestBaseURL
	if live {
		baseURL = dodoLiveBaseURL
	}
	return &DodoService{
		apiKey:         apiKey,
		webhookSecret:  webhookSecret,
		baseURL:        baseURL,
		productPro:     productPro,
		productChain:   productChain,
		frontendOrigin: frontendOrigin,
		httpClient:     &http.Client{Timeout: 10 * time.Second},
	}
}

// ProductIDForPlan maps an internal plan key to the configured Dodo product id.
func (s *DodoService) ProductIDForPlan(plan string) string {
	switch plan {
	case "pro":
		return s.productPro
	case "chain":
		return s.productChain
	default:
		return ""
	}
}

// isKnownCustomerID reports whether customerID looks like a real, previously
// stored Dodo customer id rather than the caller's local placeholder
// (BillingHandler falls back to "mock_customer_<pharmacyID prefix>" when no
// provider customer id has been saved yet).
func isKnownCustomerID(customerID string) bool {
	return customerID != "" && !strings.HasPrefix(customerID, "mock_customer_")
}

type dodoCheckoutRequest struct {
	ProductCart []dodoProductCartItem `json:"product_cart"`
	Customer    *dodoCustomerRef      `json:"customer,omitempty"`
	Metadata    map[string]string     `json:"metadata,omitempty"`
	ReturnURL   string                `json:"return_url,omitempty"`
	CancelURL   string                `json:"cancel_url,omitempty"`
}

type dodoProductCartItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// dodoCustomerRef either references an existing customer by id, or (when
// customer_id is empty) asks Dodo to create/match one by email.
type dodoCustomerRef struct {
	CustomerID string `json:"customer_id,omitempty"`
	Email      string `json:"email,omitempty"`
}

type dodoCheckoutResponse struct {
	SessionID   string `json:"session_id"`
	CheckoutURL string `json:"checkout_url"`
}

func (s *DodoService) CreateCheckoutSession(ctx context.Context, customerID, priceID, pharmacyID, plan string) (string, error) {
	if priceID == "" {
		return "", fmt.Errorf("no dodo product configured for plan %q", plan)
	}

	req := dodoCheckoutRequest{
		ProductCart: []dodoProductCartItem{{ProductID: priceID, Quantity: 1}},
		Metadata: map[string]string{
			"pharmacy_id": pharmacyID,
			"plan":        plan,
		},
		ReturnURL: s.frontendOrigin + "/billing?success=true",
		CancelURL: s.frontendOrigin + "/billing?canceled=true",
	}
	if isKnownCustomerID(customerID) {
		req.Customer = &dodoCustomerRef{CustomerID: customerID}
	}

	var resp dodoCheckoutResponse
	if err := s.doJSON(ctx, http.MethodPost, "/checkouts", req, &resp); err != nil {
		return "", fmt.Errorf("create checkout session: %w", err)
	}
	if resp.CheckoutURL == "" {
		return "", fmt.Errorf("create checkout session: empty checkout_url in response")
	}
	return resp.CheckoutURL, nil
}

type dodoPortalResponse struct {
	Link string `json:"link"`
}

func (s *DodoService) CreatePortalSession(ctx context.Context, customerID string) (string, error) {
	if !isKnownCustomerID(customerID) {
		return "", fmt.Errorf("no dodo customer on file")
	}
	path := fmt.Sprintf("/customers/%s/customer-portal/session?return_url=%s", customerID, s.frontendOrigin+"/billing")

	var resp dodoPortalResponse
	if err := s.doJSON(ctx, http.MethodPost, path, nil, &resp); err != nil {
		return "", fmt.Errorf("create portal session: %w", err)
	}
	if resp.Link == "" {
		return "", fmt.Errorf("create portal session: empty link in response")
	}
	return resp.Link, nil
}

func (s *DodoService) doJSON(ctx context.Context, method, path string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("dodo api %s %s returned %d: %s", method, path, res.StatusCode, string(respBody))
	}

	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}
	return nil
}

// DodoEvent is a loosely-typed representation of a Dodo Payments webhook
// event. The exact payload shape isn't pinned down from public docs alone,
// so callers should extract fields defensively from Data rather than assume
// a fixed schema, and consult logged raw payloads to refine this as needed.
type DodoEvent struct {
	ID        string          `json:"business_id"`
	Type      string          `json:"type"`
	Timestamp string          `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
	Raw       []byte          `json:"-"`
}

// webhookMaxAge guards against replaying old, previously-valid signed
// webhooks (standard webhooks replay protection).
const webhookMaxAge = 5 * time.Minute

// ConstructEvent verifies a Dodo Payments webhook per the Standard Webhooks
// spec (https://www.standardwebhooks.com/): HMAC-SHA256 over
// "{webhook-id}.{webhook-timestamp}.{raw body}", using the base64-decoded
// secret (an optional "whsec_" prefix is stripped first, matching Svix
// convention). The signature header may carry multiple space-separated
// "v1,<base64 sig>" candidates; any match is accepted.
func (s *DodoService) ConstructEvent(body []byte, id, timestamp, signatureHeader string) (DodoEvent, error) {
	if id == "" || timestamp == "" || signatureHeader == "" {
		return DodoEvent{}, fmt.Errorf("missing webhook signature headers")
	}

	ts, err := parseUnixSeconds(timestamp)
	if err != nil {
		return DodoEvent{}, fmt.Errorf("invalid webhook-timestamp: %w", err)
	}
	if age := time.Since(ts); age > webhookMaxAge || age < -webhookMaxAge {
		return DodoEvent{}, fmt.Errorf("webhook timestamp outside tolerance")
	}

	secret := strings.TrimPrefix(s.webhookSecret, "whsec_")
	key, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		// Some dashboards issue the raw (non-base64) secret; fall back to
		// using it verbatim as the HMAC key.
		key = []byte(secret)
	}

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id + "." + timestamp + "." + string(body)))
	expected := mac.Sum(nil)

	valid := false
	for _, candidate := range strings.Fields(signatureHeader) {
		parts := strings.SplitN(candidate, ",", 2)
		sigPart := parts[len(parts)-1]
		got, err := base64.StdEncoding.DecodeString(sigPart)
		if err != nil {
			continue
		}
		if hmac.Equal(got, expected) {
			valid = true
			break
		}
	}
	if !valid {
		return DodoEvent{}, fmt.Errorf("signature mismatch")
	}

	var evt DodoEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		return DodoEvent{}, fmt.Errorf("unmarshal event: %w", err)
	}
	evt.Raw = body
	return evt, nil
}

func parseUnixSeconds(s string) (time.Time, error) {
	var sec int64
	if _, err := fmt.Sscanf(s, "%d", &sec); err != nil {
		return time.Time{}, err
	}
	return time.Unix(sec, 0), nil
}

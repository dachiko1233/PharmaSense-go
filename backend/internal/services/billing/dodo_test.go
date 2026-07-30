package billing

import (
	"net/url"
	"strings"
	"testing"
)

// The two live test products, mapped to internal plan keys:
//
//	pdt_0NkFSz0tF8O0pdbqjmrQu -> pro   (29 EUR)
//	pdt_0NkFU3fXCPK4jli7YKqTW -> chain (99 EUR)
const (
	productPro29   = "pdt_0NkFSz0tF8O0pdbqjmrQu"
	productChain99 = "pdt_0NkFU3fXCPK4jli7YKqTW"
)

func newTestService() *DodoService {
	return NewDodoService("", "", productPro29, productChain99, "https://app.example.com", false)
}

func TestProductPlanMappingBothDirections(t *testing.T) {
	s := newTestService()

	cases := []struct {
		plan    string
		product string
	}{
		{"pro", productPro29},     // 29 EUR
		{"chain", productChain99}, // 99 EUR
	}
	for _, tc := range cases {
		if got := s.ProductIDForPlan(tc.plan); got != tc.product {
			t.Errorf("ProductIDForPlan(%q) = %q, want %q", tc.plan, got, tc.product)
		}
		if got := s.PlanForProductID(tc.product); got != tc.plan {
			t.Errorf("PlanForProductID(%q) = %q, want %q", tc.product, got, tc.plan)
		}
	}

	// Neither plan is hardcoded to the other, and unknown ids resolve to "".
	if s.ProductIDForPlan("pro") == s.ProductIDForPlan("chain") {
		t.Fatal("pro and chain resolved to the same product id")
	}
	if got := s.PlanForProductID("pdt_unknown"); got != "" {
		t.Errorf("PlanForProductID(unknown) = %q, want empty", got)
	}
}

func TestStaticCheckoutURLCarriesMetadata(t *testing.T) {
	s := newTestService()

	// Paying via the 29 EUR link must target the pro plan and product.
	raw := s.StaticCheckoutURL(productPro29, "phrm-123", "pro", "user@example.com")

	if !strings.HasPrefix(raw, "https://test.checkout.dodopayments.com/buy/"+productPro29+"?") {
		t.Fatalf("unexpected base/product in URL: %s", raw)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("URL does not parse: %v", err)
	}
	q := u.Query()
	if q.Get("quantity") != "1" {
		t.Errorf("quantity = %q, want 1", q.Get("quantity"))
	}
	if q.Get("metadata_pharmacy_id") != "phrm-123" {
		t.Errorf("metadata_pharmacy_id = %q, want phrm-123", q.Get("metadata_pharmacy_id"))
	}
	if q.Get("metadata_plan") != "pro" {
		t.Errorf("metadata_plan = %q, want pro", q.Get("metadata_plan"))
	}
	if q.Get("email") != "user@example.com" {
		t.Errorf("email = %q, want user@example.com", q.Get("email"))
	}
	if q.Get("redirect_url") != "https://app.example.com/billing?success=true" {
		t.Errorf("redirect_url = %q", q.Get("redirect_url"))
	}

	// The 99 EUR link points at the chain product/plan.
	raw99 := s.StaticCheckoutURL(productChain99, "phrm-123", "chain", "")
	if !strings.Contains(raw99, "/buy/"+productChain99+"?") {
		t.Errorf("99 EUR link has wrong product: %s", raw99)
	}
	if q99, _ := url.Parse(raw99); q99.Query().Get("metadata_plan") != "chain" {
		t.Errorf("99 EUR link metadata_plan = %q, want chain", q99.Query().Get("metadata_plan"))
	}
}

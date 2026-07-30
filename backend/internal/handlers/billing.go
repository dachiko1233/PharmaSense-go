package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"pharmasense/internal/middleware"
	"pharmasense/internal/services/billing"
)

type BillingHandler struct {
	db       *pgxpool.Pool
	billing  billing.BillingProvider
	dodoReal *billing.DodoService
}

func NewBillingHandler(db *pgxpool.Pool, provider billing.BillingProvider, dodoReal *billing.DodoService) *BillingHandler {
	return &BillingHandler{
		db:       db,
		billing:  provider,
		dodoReal: dodoReal,
	}
}

func (h *BillingHandler) CheckoutSession(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)

	var req struct {
		Plan string `json:"plan" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// With Dodo configured, hand back a hosted static checkout link carrying
	// the pharmacy_id/plan as metadata. This needs no API key — only the
	// product IDs — and the webhook attributes the payment via that metadata.
	if h.dodoReal != nil {
		productID := h.dodoReal.ProductIDForPlan(req.Plan)
		if productID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan"})
			return
		}
		email := c.GetString("email") // set by JWT middleware; prefilled at checkout
		checkoutURL := h.dodoReal.StaticCheckoutURL(productID, pharmacyID.String(), req.Plan, email)
		slog.Info("dodo checkout: static link created",
			"pharmacy_id", pharmacyID, "plan", req.Plan, "product_id", productID)
		c.JSON(http.StatusOK, gin.H{"url": checkoutURL})
		return
	}

	// Mock provider fallback (no Dodo product IDs configured).
	customerID := "mock_customer_" + pharmacyID.String()[:8]
	url, err := h.billing.CreateCheckoutSession(c.Request.Context(), customerID, req.Plan, pharmacyID.String(), req.Plan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create checkout session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

func (h *BillingHandler) PortalSession(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)

	customerID := "mock_customer_" + pharmacyID.String()[:8]
	if h.dodoReal != nil {
		var dodoCustomerID *string
		_ = h.db.QueryRow(c.Request.Context(),
			`SELECT dodo_customer_id FROM pharmacies WHERE id = $1`, pharmacyID,
		).Scan(&dodoCustomerID)
		if dodoCustomerID != nil {
			customerID = *dodoCustomerID
		}
	}

	url, err := h.billing.CreatePortalSession(c.Request.Context(), customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create portal session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

func (h *BillingHandler) Subscription(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)

	var result struct {
		Plan                         string     `json:"plan"`
		SubscriptionStatus           *string    `json:"subscription_status"`
		SubscriptionCurrentPeriodEnd *time.Time `json:"subscription_current_period_end"`
	}
	_ = h.db.QueryRow(c.Request.Context(), `
		SELECT plan, subscription_status, subscription_current_period_end
		FROM pharmacies WHERE id = $1
	`, pharmacyID).Scan(&result.Plan, &result.SubscriptionStatus, &result.SubscriptionCurrentPeriodEnd)

	c.JSON(http.StatusOK, result)
}

// dodoPayload is a loosely-typed view over a Dodo Payments webhook event's
// "data" object. The exact field names weren't fully pinned down from public
// docs, so both common id-field shapes are accepted; check slog output from
// a real test-mode delivery if a field ever comes back empty unexpectedly.
type dodoPayload struct {
	Metadata        map[string]string `json:"metadata"`
	CustomerID      string            `json:"customer_id"`
	Customer        *dodoCustomerObj  `json:"customer"`
	SubscriptionID  string            `json:"subscription_id"`
	PaymentID       string            `json:"payment_id"`
	ProductID       string            `json:"product_id"`
	Status          string            `json:"status"`
	NextBillingDate string            `json:"next_billing_date"`
}

type dodoCustomerObj struct {
	CustomerID string `json:"customer_id"`
	ID         string `json:"id"`
	Email      string `json:"email"`
}

func (p dodoPayload) resolveCustomerID() string {
	if p.CustomerID != "" {
		return p.CustomerID
	}
	if p.Customer != nil {
		if p.Customer.CustomerID != "" {
			return p.Customer.CustomerID
		}
		return p.Customer.ID
	}
	return ""
}

func (p dodoPayload) resolveEmail() string {
	if p.Customer != nil {
		return p.Customer.Email
	}
	return ""
}

// DodoWebhook receives payment/subscription lifecycle events from Dodo
// Payments and applies them to the paying pharmacy's plan, verified via
// Dodo's Standard Webhooks-style HMAC signature.
func (h *BillingHandler) DodoWebhook(c *gin.Context) {
	slog.Info("dodo webhook: request received",
		"webhook_id", c.GetHeader("webhook-id"), "content_length", c.Request.ContentLength)

	if h.dodoReal == nil {
		slog.Warn("dodo webhook: dodo not configured (DODO_API_KEY unset), dropping event")
		c.JSON(http.StatusOK, gin.H{"message": "dodo not configured"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		slog.Error("dodo webhook: failed to read body", "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	event, err := h.dodoReal.ConstructEvent(
		body,
		c.GetHeader("webhook-id"),
		c.GetHeader("webhook-timestamp"),
		c.GetHeader("webhook-signature"),
	)
	if err != nil {
		slog.Error("dodo webhook: signature verification failed",
			"webhook_id", c.GetHeader("webhook-id"), "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signature"})
		return
	}

	slog.Info("dodo webhook: signature verified", "id", event.ID, "type", event.Type)
	slog.Debug("dodo webhook payload", "body", string(event.Raw))

	// Idempotency check
	var alreadyProcessed bool
	_ = h.db.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM dodo_events WHERE id = $1)`, event.ID,
	).Scan(&alreadyProcessed)
	if alreadyProcessed {
		slog.Info("dodo webhook: already processed, skipping", "id", event.ID, "type", event.Type)
		c.JSON(http.StatusOK, gin.H{"message": "already processed"})
		return
	}
	_, _ = h.db.Exec(c.Request.Context(),
		`INSERT INTO dodo_events (id, type) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		event.ID, event.Type,
	)

	var payload dodoPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		slog.Warn("dodo webhook: failed to parse payload", "type", event.Type, "id", event.ID, "err", err)
		c.JSON(http.StatusOK, gin.H{"received": true})
		return
	}
	customerID := payload.resolveCustomerID()

	switch event.Type {
	case "payment.succeeded", "subscription.active":
		pharmacyID := payload.Metadata["pharmacy_id"]
		matchedBy := "metadata.pharmacy_id"
		plan := payload.Metadata["plan"]
		if plan == "" {
			// Metadata can be dropped by checkouts created outside our own
			// CreateCheckoutSession call; the product_id is always present
			// and maps 1:1 to a plan, so fall back to that.
			plan = h.dodoReal.PlanForProductID(payload.ProductID)
		}
		if pharmacyID == "" {
			if email := payload.resolveEmail(); email != "" {
				// Metadata is the source of truth for which pharmacy paid;
				// this only covers checkouts created outside our own flow,
				// so fall back to the paying user's default pharmacy.
				matchedBy = "email:" + email
				_ = h.db.QueryRow(c.Request.Context(),
					`SELECT default_pharmacy_id FROM users WHERE email = $1 AND default_pharmacy_id IS NOT NULL`, email,
				).Scan(&pharmacyID)
			}
		}
		if pharmacyID == "" {
			slog.Warn("dodo webhook: user/pharmacy not found, plan NOT updated",
				"type", event.Type, "id", event.ID, "customer_id", customerID,
				"product_id", payload.ProductID, "attempted_match", matchedBy)
			break
		}
		if plan == "" {
			slog.Warn("dodo webhook: pharmacy matched but plan could not be resolved, plan NOT updated",
				"type", event.Type, "id", event.ID, "pharmacy_id", pharmacyID, "product_id", payload.ProductID)
			break
		}
		slog.Info("dodo webhook: pharmacy matched", "id", event.ID, "pharmacy_id", pharmacyID, "plan", plan, "matched_by", matchedBy)

		tag, err := h.db.Exec(c.Request.Context(), `
			UPDATE pharmacies SET
			  dodo_customer_id = COALESCE(NULLIF($1, ''), dodo_customer_id),
			  dodo_subscription_id = COALESCE(NULLIF($2, ''), dodo_subscription_id),
			  subscription_status = 'active',
			  plan = COALESCE(NULLIF($3, ''), plan),
			  updated_at = NOW()
			WHERE id = $4
		`, customerID, payload.SubscriptionID, plan, pharmacyID)
		if err != nil {
			slog.Error("dodo webhook: plan update FAILED", "id", event.ID, "pharmacy_id", pharmacyID, "plan", plan, "err", err)
		} else if tag.RowsAffected() == 0 {
			slog.Warn("dodo webhook: plan update matched 0 rows, pharmacy_id likely invalid",
				"id", event.ID, "pharmacy_id", pharmacyID, "plan", plan)
		} else {
			slog.Info("dodo webhook: plan updated", "id", event.ID, "pharmacy_id", pharmacyID, "plan", plan)
		}

	case "subscription.renewed", "subscription.updated":
		status := payload.Status
		if status == "" {
			status = "active"
		}
		_, _ = h.db.Exec(c.Request.Context(), `
			UPDATE pharmacies SET subscription_status = $1, updated_at = NOW()
			WHERE dodo_subscription_id = $2
		`, status, payload.SubscriptionID)

	case "subscription.plan_changed":
		plan := payload.Metadata["plan"]
		_, _ = h.db.Exec(c.Request.Context(), `
			UPDATE pharmacies SET
			  plan = COALESCE(NULLIF($1, ''), plan),
			  subscription_status = 'active',
			  updated_at = NOW()
			WHERE dodo_subscription_id = $2
		`, plan, payload.SubscriptionID)

	case "subscription.cancelled", "subscription.expired":
		_, _ = h.db.Exec(c.Request.Context(), `
			UPDATE pharmacies SET
			  plan = 'free',
			  subscription_status = 'canceled',
			  dodo_subscription_id = NULL,
			  updated_at = NOW()
			WHERE dodo_subscription_id = $1
		`, payload.SubscriptionID)

	case "subscription.on_hold", "payment.failed":
		_, _ = h.db.Exec(c.Request.Context(), `
			UPDATE pharmacies SET subscription_status = 'past_due', updated_at = NOW()
			WHERE dodo_subscription_id = $1 OR dodo_customer_id = $2
		`, payload.SubscriptionID, customerID)
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

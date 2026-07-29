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
	db         *pgxpool.Pool
	billing    billing.BillingProvider
	stripeReal *billing.StripeService
	dodoReal   *billing.DodoService
	isMock     bool
}

func NewBillingHandler(db *pgxpool.Pool, provider billing.BillingProvider, stripeReal *billing.StripeService, dodoReal *billing.DodoService, isMock bool) *BillingHandler {
	return &BillingHandler{
		db:         db,
		billing:    provider,
		stripeReal: stripeReal,
		dodoReal:   dodoReal,
		isMock:     isMock,
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

	customerID := "mock_customer_" + pharmacyID.String()[:8]
	priceID := req.Plan

	switch {
	case h.dodoReal != nil:
		var dodoCustomerID *string
		_ = h.db.QueryRow(c.Request.Context(),
			`SELECT dodo_customer_id FROM pharmacies WHERE id = $1`, pharmacyID,
		).Scan(&dodoCustomerID)
		if dodoCustomerID != nil {
			customerID = *dodoCustomerID
		}

		priceID = h.dodoReal.ProductIDForPlan(req.Plan)
		if priceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan"})
			return
		}

	case !h.isMock && h.stripeReal != nil:
		var stripeCustomerID *string
		_ = h.db.QueryRow(c.Request.Context(),
			`SELECT stripe_customer_id FROM pharmacies WHERE id = $1`, pharmacyID,
		).Scan(&stripeCustomerID)
		if stripeCustomerID != nil {
			customerID = *stripeCustomerID
		}

		priceID = h.stripeReal.PriceIDForPlan(req.Plan)
		if priceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan"})
			return
		}
	}

	url, err := h.billing.CreateCheckoutSession(c.Request.Context(), customerID, priceID, pharmacyID.String(), req.Plan)
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
	} else {
		var stripeCustomerID *string
		_ = h.db.QueryRow(c.Request.Context(),
			`SELECT stripe_customer_id FROM pharmacies WHERE id = $1`, pharmacyID,
		).Scan(&stripeCustomerID)
		if stripeCustomerID != nil {
			customerID = *stripeCustomerID
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

// stripeEvent is a minimal representation of a Stripe event for webhook processing.
type stripeEvent struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type stripeEventData struct {
	Object json.RawMessage `json:"object"`
}

type stripeCheckoutSession struct {
	Customer     *stripeRef `json:"customer"`
	Subscription *stripeRef `json:"subscription"`
	Metadata     map[string]string `json:"metadata"`
}

type stripeSubscription struct {
	ID                string    `json:"id"`
	Status            string    `json:"status"`
	CurrentPeriodEnd  int64     `json:"current_period_end"`
}

type stripeInvoice struct {
	Customer *stripeRef `json:"customer"`
}

type stripeRef struct {
	ID string `json:"id"`
}

func (h *BillingHandler) Webhook(c *gin.Context) {
	if h.isMock || h.stripeReal == nil {
		c.JSON(http.StatusOK, gin.H{"message": "mock webhook received"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	sigHeader := c.GetHeader("Stripe-Signature")
	event, err := h.stripeReal.ConstructEvent(body, sigHeader)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signature"})
		return
	}

	// Idempotency check
	var alreadyProcessed bool
	_ = h.db.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM stripe_events WHERE id = $1)`, event.ID,
	).Scan(&alreadyProcessed)
	if alreadyProcessed {
		c.JSON(http.StatusOK, gin.H{"message": "already processed"})
		return
	}

	_, _ = h.db.Exec(c.Request.Context(),
		`INSERT INTO stripe_events (id, type) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		event.ID, event.Type,
	)

	var evtData stripeEventData
	if err := json.Unmarshal(event.Data.Raw, &evtData); err != nil {
		c.JSON(http.StatusOK, gin.H{"received": true})
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		var sess stripeCheckoutSession
		if err := json.Unmarshal(evtData.Object, &sess); err == nil {
			if pharmacyID, ok := sess.Metadata["pharmacy_id"]; ok {
				customerID := ""
				subID := ""
				plan := sess.Metadata["plan"]
				if sess.Customer != nil {
					customerID = sess.Customer.ID
				}
				if sess.Subscription != nil {
					subID = sess.Subscription.ID
				}
				_, _ = h.db.Exec(c.Request.Context(), `
					UPDATE pharmacies SET
					  stripe_customer_id = $1,
					  stripe_subscription_id = $2,
					  subscription_status = 'active',
					  plan = COALESCE(NULLIF($3, ''), plan),
					  updated_at = NOW()
					WHERE id = $4
				`, customerID, subID, plan, pharmacyID)
			}
		}

	case "customer.subscription.updated":
		var sub stripeSubscription
		if err := json.Unmarshal(evtData.Object, &sub); err == nil {
			_, _ = h.db.Exec(c.Request.Context(), `
				UPDATE pharmacies SET
				  subscription_status = $1,
				  subscription_current_period_end = to_timestamp($2),
				  updated_at = NOW()
				WHERE stripe_subscription_id = $3
			`, sub.Status, sub.CurrentPeriodEnd, sub.ID)
		}

	case "customer.subscription.deleted":
		var sub stripeSubscription
		if err := json.Unmarshal(evtData.Object, &sub); err == nil {
			_, _ = h.db.Exec(c.Request.Context(), `
				UPDATE pharmacies SET
				  plan = 'free',
				  subscription_status = 'canceled',
				  stripe_subscription_id = NULL,
				  updated_at = NOW()
				WHERE stripe_subscription_id = $1
			`, sub.ID)
		}

	case "invoice.payment_failed":
		var inv stripeInvoice
		if err := json.Unmarshal(evtData.Object, &inv); err == nil && inv.Customer != nil {
			_, _ = h.db.Exec(c.Request.Context(), `
				UPDATE pharmacies SET subscription_status = 'past_due', updated_at = NOW()
				WHERE stripe_customer_id = $1
			`, inv.Customer.ID)
		}
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
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

// DodoWebhook receives payment/subscription lifecycle events from Dodo
// Payments and applies them to the paying pharmacy's plan. It's the Dodo
// analogue of Webhook (Stripe) above, using Dodo's Standard Webhooks-style
// HMAC signature instead of Stripe-Signature.
func (h *BillingHandler) DodoWebhook(c *gin.Context) {
	if h.dodoReal == nil {
		c.JSON(http.StatusOK, gin.H{"message": "dodo not configured"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signature"})
		return
	}

	slog.Info("dodo webhook received", "id", event.ID, "type", event.Type)
	slog.Debug("dodo webhook payload", "body", string(event.Raw))

	// Idempotency check
	var alreadyProcessed bool
	_ = h.db.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM dodo_events WHERE id = $1)`, event.ID,
	).Scan(&alreadyProcessed)
	if alreadyProcessed {
		c.JSON(http.StatusOK, gin.H{"message": "already processed"})
		return
	}
	_, _ = h.db.Exec(c.Request.Context(),
		`INSERT INTO dodo_events (id, type) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		event.ID, event.Type,
	)

	var payload dodoPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		slog.Warn("dodo webhook: failed to parse payload", "type", event.Type, "err", err)
		c.JSON(http.StatusOK, gin.H{"received": true})
		return
	}
	customerID := payload.resolveCustomerID()

	switch event.Type {
	case "payment.succeeded", "subscription.active":
		pharmacyID := payload.Metadata["pharmacy_id"]
		plan := payload.Metadata["plan"]
		if pharmacyID == "" {
			slog.Warn("dodo webhook: grant event missing metadata.pharmacy_id", "type", event.Type, "id", event.ID)
			break
		}
		_, _ = h.db.Exec(c.Request.Context(), `
			UPDATE pharmacies SET
			  dodo_customer_id = COALESCE(NULLIF($1, ''), dodo_customer_id),
			  dodo_subscription_id = COALESCE(NULLIF($2, ''), dodo_subscription_id),
			  subscription_status = 'active',
			  plan = COALESCE(NULLIF($3, ''), plan),
			  updated_at = NOW()
			WHERE id = $4
		`, customerID, payload.SubscriptionID, plan, pharmacyID)

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

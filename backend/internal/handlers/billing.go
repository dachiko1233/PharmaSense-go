package handlers

import (
	"encoding/json"
	"io"
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
	isMock     bool
}

func NewBillingHandler(db *pgxpool.Pool, provider billing.BillingProvider, stripeReal *billing.StripeService, isMock bool) *BillingHandler {
	return &BillingHandler{
		db:         db,
		billing:    provider,
		stripeReal: stripeReal,
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

	var stripeCustomerID *string
	_ = h.db.QueryRow(c.Request.Context(),
		`SELECT stripe_customer_id FROM pharmacies WHERE id = $1`, pharmacyID,
	).Scan(&stripeCustomerID)

	customerID := "mock_customer_" + pharmacyID.String()[:8]
	if stripeCustomerID != nil {
		customerID = *stripeCustomerID
	}

	priceID := req.Plan
	if !h.isMock && h.stripeReal != nil {
		priceID = h.stripeReal.PriceIDForPlan(req.Plan)
		if priceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan"})
			return
		}
	}

	url, err := h.billing.CreateCheckoutSession(c.Request.Context(), customerID, priceID, pharmacyID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create checkout session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

func (h *BillingHandler) PortalSession(c *gin.Context) {
	pharmacyID := middleware.GetPharmacyID(c)

	var stripeCustomerID *string
	_ = h.db.QueryRow(c.Request.Context(),
		`SELECT stripe_customer_id FROM pharmacies WHERE id = $1`, pharmacyID,
	).Scan(&stripeCustomerID)

	customerID := "mock_customer_" + pharmacyID.String()[:8]
	if stripeCustomerID != nil {
		customerID = *stripeCustomerID
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
					  updated_at = NOW()
					WHERE id = $3
				`, customerID, subID, pharmacyID)
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

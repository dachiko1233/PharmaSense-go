package server

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"pharmasense/internal/config"
	"pharmasense/internal/handlers"
	"pharmasense/internal/middleware"
	"pharmasense/internal/services"
	"pharmasense/internal/services/billing"
	"pharmasense/internal/services/notifications"
)

func New(cfg *config.Config, db *pgxpool.Pool) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.SlogLogger())

	corsConfig := cors.Config{
		AllowOrigins:     []string{cfg.FrontendOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	r.Use(cors.New(corsConfig))

	// Services
	authSvc := services.NewAuthService(db, cfg.JWTSecret, cfg.JWTExpiryHours, cfg.AppURL)
	emailSvc := notifications.NewEmailService(cfg.ResendAPIKey, cfg.ResendFromEmail)
	smsSvc := notifications.NewSMSService(cfg.TwilioAccountSID, cfg.TwilioAuthToken, cfg.TwilioFromNumber)

	// Billing provider (real or mock)
	isMockBilling := cfg.StripeSecretKey == "" || len(cfg.StripeSecretKey) >= 5 && cfg.StripeSecretKey[:5] == "mock_"
	var billingProvider billing.BillingProvider
	var stripeReal *billing.StripeService
	if isMockBilling {
		billingProvider = billing.NewMockBillingService(cfg.FrontendOrigin)
	} else {
		stripeReal = billing.NewStripeService(
			cfg.StripeSecretKey, cfg.StripeWebhookSecret,
			cfg.StripePricePro, cfg.StripePriceChain, cfg.FrontendOrigin,
		)
		billingProvider = stripeReal
	}

	// Handlers
	authH := handlers.NewAuthHandler(authSvc, emailSvc, cfg.AppURL)
	pharmacyH := handlers.NewPharmacyHandler(db, authSvc)
	chainH := handlers.NewChainHandler(db)
	invH := handlers.NewInventoryHandler(db)
	riskH := handlers.NewRiskHandler(db)
	alertH := handlers.NewAlertHandler(db)
	reportH := handlers.NewReportHandler(db)
	settingsH := handlers.NewSettingsHandler(db)
	billingH := handlers.NewBillingHandler(db, billingProvider, stripeReal, isMockBilling)

	// Suppress unused variable warning
	_ = smsSvc

	api := r.Group("/api/v1")

	// Health check — critical for Railway
	api.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Auth routes (public)
	authRateLimit := middleware.RateLimit(10, 0.5) // 10 burst, 1 req/2s
	auth := api.Group("/auth")
	auth.POST("/signup", authRateLimit, authH.Signup)
	auth.POST("/login", authRateLimit, authH.Login)
	auth.POST("/logout", authH.Logout)
	auth.POST("/verify-email", authH.VerifyEmail)
	auth.POST("/forgot-password", authH.ForgotPassword)
	auth.POST("/reset-password", authH.ResetPassword)

	// Stripe webhook (unauthenticated, signature-verified)
	api.POST("/billing/webhook", billingH.Webhook)

	// Protected routes
	protected := api.Group("/")
	protected.Use(middleware.JWTAuth(cfg.JWTSecret))

	// Auth
	protected.GET("/auth/me", authH.Me)

	// Pharmacies
	protected.GET("/pharmacies", pharmacyH.List)
	protected.GET("/pharmacies/:id", pharmacyH.Get)
	protected.PATCH("/pharmacies/:id", pharmacyH.Patch)
	protected.POST("/pharmacies/switch", pharmacyH.Switch)

	// Chains (chain_admin only, enforced in handler)
	protected.GET("/chains/:id", chainH.Get)
	protected.GET("/chains/:id/dashboard", chainH.Dashboard)

	// Products
	protected.GET("/products", func(c *gin.Context) {
		pharmacyID := middleware.GetPharmacyID(c)
		rows, err := db.Query(c.Request.Context(), `
			SELECT DISTINCT p.id, p.name, p.category, p.manufacturer, p.requires_prescription, p.created_at
			FROM products p
			JOIN inventory_batches ib ON ib.product_id = p.id
			WHERE ib.pharmacy_id = $1
			ORDER BY p.name LIMIT 200
		`, pharmacyID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
			return
		}
		defer rows.Close()
		type Prod struct {
			ID                   string  `json:"id"`
			Name                 string  `json:"name"`
			Category             *string `json:"category"`
			Manufacturer         *string `json:"manufacturer"`
			RequiresPrescription bool    `json:"requires_prescription"`
		}
		var prods []Prod
		for rows.Next() {
			var p Prod
			var createdAt time.Time
			if err := rows.Scan(&p.ID, &p.Name, &p.Category, &p.Manufacturer, &p.RequiresPrescription, &createdAt); err != nil {
				continue
			}
			prods = append(prods, p)
		}
		if prods == nil {
			prods = []Prod{}
		}
		c.JSON(http.StatusOK, prods)
	})

	// Inventory
	protected.GET("/inventory", invH.List)
	protected.POST("/inventory", invH.Create)
	protected.DELETE("/inventory/:id", invH.Delete)

	// CSV import — Pro plan required
	proGroup := protected.Group("/")
	proGroup.Use(func(c *gin.Context) {
		// Inject plan from DB for plan middleware
		pharmacyID := middleware.GetPharmacyID(c)
		var plan string
		_ = db.QueryRow(c.Request.Context(), `SELECT plan FROM pharmacies WHERE id = $1`, pharmacyID).Scan(&plan)
		c.Set("plan", plan)
		c.Next()
	})
	proGroup.Use(middleware.RequirePlan("pro"))
	proGroup.POST("/inventory/import", invH.Import)

	// Sales
	protected.GET("/sales/stats", func(c *gin.Context) {
		pharmacyID := middleware.GetPharmacyID(c)
		var total float64
		var count int
		_ = db.QueryRow(c.Request.Context(), `
			SELECT COUNT(*), COALESCE(SUM(total_amount), 0)
			FROM sales WHERE pharmacy_id = $1 AND sale_date >= NOW() - INTERVAL '30 days'
		`, pharmacyID).Scan(&count, &total)
		c.JSON(http.StatusOK, gin.H{"last_30_days_count": count, "last_30_days_total": total})
	})

	// Risk
	protected.GET("/risk/dashboard", riskH.Dashboard)
	protected.GET("/risk/assessments", riskH.Assessments)
	protected.POST("/risk/recalculate", riskH.Recalculate)
	protected.GET("/risk/timeline", riskH.Timeline)

	// Alerts
	protected.GET("/alerts", alertH.List)
	protected.POST("/alerts/:batch_id/action", alertH.Action)

	// Reports
	protected.GET("/reports/savings", reportH.Savings)
	protected.GET("/reports/waste", reportH.Waste)
	protected.GET("/reports/categories", reportH.Categories)

	// Settings
	protected.GET("/settings/notifications", settingsH.GetNotifications)
	protected.PATCH("/settings/notifications", settingsH.PatchNotifications)

	// Billing
	protected.POST("/billing/checkout-session", billingH.CheckoutSession)
	protected.POST("/billing/portal-session", billingH.PortalSession)
	protected.GET("/billing/subscription", billingH.Subscription)

	return r
}

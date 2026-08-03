package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL     string
	JWTSecret       string
	JWTExpiryHours  int
	FrontendOrigin  string
	FrontendOrigins []string
	Env             string
	Port            string
	AppURL          string

	ResendAPIKey    string
	ResendFromEmail string

	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFromNumber string

	DodoAPIKey         string
	DodoWebhookSecret  string
	DodoLiveMode       bool
	DodoProductIDPro   string
	DodoProductIDChain string

	DisableCron bool
}

func Load() *Config {
	// Load environment from .env in a CWD-independent way. godotenv.Load()
	// reads "./.env" relative to the working directory, so launching from the
	// repo root (instead of backend/) would silently miss backend/.env and
	// leave Dodo unconfigured ("billing not configured"). Try the common
	// locations, and an explicit ENV_FILE override, loading the first that
	// exists. Already-set real env vars always win (godotenv never overrides).
	candidates := []string{}
	if p := os.Getenv("ENV_FILE"); p != "" {
		candidates = append(candidates, p)
	}
	candidates = append(candidates, ".env", "backend/.env", "../.env")
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			_ = godotenv.Load(p)
			break
		}
	}

	hours, _ := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "24"))
	disableCron := getEnv("DISABLE_CRON", "false") == "true"

	origin := getEnv("FRONTEND_ORIGIN", "http://localhost:3000")
	var origins []string
	for _, o := range strings.Split(origin, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}

	return &Config{
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/pharmasense?sslmode=disable"),
		JWTSecret:       getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		JWTExpiryHours:  hours,
		FrontendOrigin:  origin,
		FrontendOrigins: origins,
		Env:             getEnv("ENV", "development"),
		Port:            getEnv("PORT", "3001"),
		AppURL:          getEnv("APP_URL", "http://localhost:3000"),

		ResendAPIKey:    getEnv("RESEND_API_KEY", ""),
		ResendFromEmail: getEnv("RESEND_FROM_EMAIL", "noreply@pharmasense.cy"),

		TwilioAccountSID: getEnv("TWILIO_ACCOUNT_SID", ""),
		TwilioAuthToken:  getEnv("TWILIO_AUTH_TOKEN", ""),
		TwilioFromNumber: getEnv("TWILIO_FROM_NUMBER", ""),

		DodoAPIKey:         getEnv("DODO_API_KEY", "mWbv36Cy7wlMqA-9.x9y6uN9v6iBCYhOyV6iEbmV4wQi0lTk8n-CbH9F4Ut-aEtJ0"),
		DodoWebhookSecret:  getEnv("DODO_WEBHOOK_SECRET", "whsec_rdnsE4PEfftQBMKmMm/15+oT3vUPb3Av"),
		DodoLiveMode:       getEnv("DODO_MODE", "test") == "live",
		DodoProductIDPro:   getEnv("DODO_PRODUCT_ID_PRO", "https://test.checkout.dodopayments.com/buy/pdt_0NkFSz0tF8O0pdbqjmrQu?quantity=1"),
		DodoProductIDChain: getEnv("DODO_PRODUCT_ID_CHAIN", "https://test.checkout.dodopayments.com/buy/pdt_0NkFU3fXCPK4jli7YKqTW?quantity=1"),

		DisableCron: disableCron,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

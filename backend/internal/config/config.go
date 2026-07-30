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

		DodoAPIKey:         getEnv("DODO_API_KEY", ""),
		DodoWebhookSecret:  getEnv("DODO_WEBHOOK_SECRET", ""),
		DodoLiveMode:       getEnv("DODO_MODE", "test") == "live",
		DodoProductIDPro:   getEnv("DODO_PRODUCT_ID_PRO", ""),
		DodoProductIDChain: getEnv("DODO_PRODUCT_ID_CHAIN", ""),

		DisableCron: disableCron,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

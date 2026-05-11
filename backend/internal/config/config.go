package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL    string
	JWTSecret      string
	JWTExpiryHours int
	FrontendOrigin string
	Env            string
	Port           string
	AppURL         string

	ResendAPIKey    string
	ResendFromEmail string

	TwilioAccountSID  string
	TwilioAuthToken   string
	TwilioFromNumber  string

	StripeSecretKey      string
	StripeWebhookSecret  string
	StripePricePro       string
	StripePriceChain     string

	DisableCron bool
}

func Load() *Config {
	_ = godotenv.Load()

	hours, _ := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "24"))
	disableCron := getEnv("DISABLE_CRON", "false") == "true"

	return &Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/pharmasense?sslmode=disable"),
		JWTSecret:      getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		JWTExpiryHours: hours,
		FrontendOrigin: getEnv("FRONTEND_ORIGIN", "http://localhost:3000"),
		Env:            getEnv("ENV", "development"),
		Port:           getEnv("PORT", "3001"),
		AppURL:         getEnv("APP_URL", "http://localhost:3000"),

		ResendAPIKey:    getEnv("RESEND_API_KEY", ""),
		ResendFromEmail: getEnv("RESEND_FROM_EMAIL", "noreply@pharmasense.cy"),

		TwilioAccountSID: getEnv("TWILIO_ACCOUNT_SID", ""),
		TwilioAuthToken:  getEnv("TWILIO_AUTH_TOKEN", ""),
		TwilioFromNumber: getEnv("TWILIO_FROM_NUMBER", ""),

		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		StripePricePro:      getEnv("STRIPE_PRICE_PRO", ""),
		StripePriceChain:    getEnv("STRIPE_PRICE_CHAIN", ""),

		DisableCron: disableCron,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

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

	DodoAPIKey         string
	DodoWebhookSecret  string
	DodoLiveMode       bool
	DodoProductIDPro   string
	DodoProductIDChain string

	// TripleCheck signup screening (external anti-abuse API). When
	// TripleCheckURL is empty, screening is disabled and all signups pass.
	TripleCheckURL       string
	TripleCheckAPIKey    string
	TripleCheckTimeoutMS int

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
	tripleCheckTimeout, _ := strconv.Atoi(getEnv("TRIPLECHECK_TIMEOUT_MS", "4000"))

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

		DodoAPIKey:        getEnv("DODO_API_KEY", ""),
		DodoWebhookSecret: getEnv("DODO_WEBHOOK_SECRET", ""),
		// Case-insensitive + trimmed so "live", "Live", or " LIVE " all enable
		// live mode. A strict "== live" compare silently kept a "Live" value in
		// test mode, pointing checkout at the test host.
		DodoLiveMode:       strings.EqualFold(strings.TrimSpace(getEnv("DODO_MODE", "test")), "live"),
		DodoProductIDPro:   getEnv("DODO_PRODUCT_ID_PRO", ""),
		DodoProductIDChain: getEnv("DODO_PRODUCT_ID_CHAIN", ""),

		TripleCheckURL:       getEnv("TRIPLECHECK_URL", ""),
		TripleCheckAPIKey:    getEnv("TRIPLECHECK_API_KEY", ""),
		TripleCheckTimeoutMS: tripleCheckTimeout,

		DisableCron: disableCron,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

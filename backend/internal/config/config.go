package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBDatabase string
	DBUsername string
	DBPassword string
	Port       string

	// JWTSecret verifies the Bearer tokens the main FlowPOS system issues.
	JWTSecret string
	// AllowDevTokens mounts POST /api/v1/dev/token for local testing. Must be
	// false in any environment reachable from outside the dev machine.
	AllowDevTokens bool
	// SigningSecret verifies the X-Flowpos-Signature header on
	// /install, /uninstall, and /webhooks.
	SigningSecret string

	// FlowposAPIURL is the base URL of the core FlowPOS API this app calls
	// on each tenant's behalf (using their installation api_key) to sync
	// locations and employees. Same idea as quotes' FLOWPOS_API_URL.
	FlowposAPIURL string
	// SyncInterval is how often the background sync job re-syncs every
	// installed tenant's locations/employees. There is no existing
	// cron/scheduler convention in the sibling apps to mirror (checked —
	// none of them have one), so this is an in-process time.Ticker; see
	// internal/modules/sync/scheduler.go for the single-replica caveat.
	SyncInterval time.Duration
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file loaded: %v", err)
	}

	jwtSecret := getenv("JWT_SECRET", "dev-insecure-secret-change-me")
	if jwtSecret == "dev-insecure-secret-change-me" {
		log.Printf("WARNING: JWT_SECRET not set — using an insecure dev secret")
	}
	signingSecret := getenv("FLOWPOS_SIGNING_SECRET", "dev-insecure-signing-secret-change-me")
	if signingSecret == "dev-insecure-signing-secret-change-me" {
		log.Printf("WARNING: FLOWPOS_SIGNING_SECRET not set — using an insecure dev secret")
	}

	return Config{
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBDatabase: os.Getenv("DB_DATABASE"),
		DBUsername: os.Getenv("DB_USERNAME"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		Port:       getenv("PORT", "8080"),

		JWTSecret:      jwtSecret,
		AllowDevTokens: os.Getenv("JWT_DEV_TOKENS") == "true",
		SigningSecret:  signingSecret,

		FlowposAPIURL: getenv("FLOWPOS_API_URL", "https://api.flowpos.dev/v1"),
		SyncInterval:  getenvDuration("SYNC_INTERVAL_MINUTES", 60*time.Minute),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	minutes, err := strconv.Atoi(v)
	if err != nil || minutes <= 0 {
		log.Printf("WARNING: invalid %s=%q, using default %s", key, v, fallback)
		return fallback
	}
	return time.Duration(minutes) * time.Minute
}

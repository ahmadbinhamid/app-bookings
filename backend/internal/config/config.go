package config

import (
	"log"
	"os"

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
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

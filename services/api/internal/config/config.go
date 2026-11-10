package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	AppEnv            string
	Port              string
	PublicBaseURL     string
	WebOrigin         string
	DemoMode          bool
	DatabaseURL       string
	SessionSecret     string
	EncryptionKey     string
	PayPalMode        string
	PayPalEnv         string
	PayPalAPIBase     string
	PayPalClientID     string
	PayPalClientSecret string
	PayPalWebhookID   string
	LLMEnabled        bool
}

func Load() (*Config, error) {
	c := &Config{
		AppEnv:            getEnv("APP_ENV", "local"),
		Port:              getEnv("PORT", "8080"),
		PublicBaseURL:     getEnv("PUBLIC_BASE_URL", "http://localhost:8080"),
		WebOrigin:         getEnv("WEB_ORIGIN", "http://localhost:5173"),
		DemoMode:          getEnv("DEMO_MODE", "true") == "true",
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://paybridge:paybridge_secret@localhost:5432/paybridge?sslmode=disable"),
		SessionSecret:     getEnv("SESSION_SECRET", "dev_session_secret_min_32_bytes_random_default"),
		EncryptionKey:     getEnv("ENCRYPTION_KEY", "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="),
		PayPalMode:        getEnv("PAYPAL_MODE", "fake"),
		PayPalEnv:         getEnv("PAYPAL_ENV", "sandbox"),
		PayPalAPIBase:     getEnv("PAYPAL_API_BASE", "https://api-m.sandbox.paypal.com"),
		PayPalClientID:     getEnv("PAYPAL_CLIENT_ID", "sandbox_client_id_default"),
		PayPalClientSecret: getEnv("PAYPAL_CLIENT_SECRET", "sandbox_client_secret_default"),
		PayPalWebhookID:   getEnv("PAYPAL_WEBHOOK_ID", "sandbox_webhook_id_default"),
		LLMEnabled:        getEnv("LLM_ENABLED", "false") == "true",
	}

	// Security Invariant: PRODUCTION host is strictly disabled for sandbox MVP
	if strings.Contains(strings.ToLower(c.PayPalAPIBase), "api.paypal.com") && !strings.Contains(strings.ToLower(c.PayPalAPIBase), "sandbox") {
		return nil, errors.New("security guard: production PayPal host is strictly disabled")
	}

	return c, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

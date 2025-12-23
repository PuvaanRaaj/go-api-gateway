package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port         int
	BackendA     string
	BackendB     string
	DatabaseURL  string
	JWTSecret    string
	APIKeyHeader string
	TokenTTL     time.Duration
}

func Load() *Config {
	port := parsePort(firstEnv([]string{"GATEWAY_PORT", "PORT"}, "8080"))
	ttl := parseDuration(firstEnv([]string{"TOKEN_TTL"}, "1h"))

	return &Config{
		Port:         port,
		BackendA:     firstEnv([]string{"BACKEND_A_URL", "BACKEND_A"}, "http://service-a:8080"),
		BackendB:     firstEnv([]string{"BACKEND_B_URL", "BACKEND_B"}, "http://service-b:8080"),
		DatabaseURL:  firstEnv([]string{"DATABASE_URL", "SUPABASE_DB_URL"}, "postgres://gateway:gateway@localhost:5432/gateway?sslmode=disable"),
		JWTSecret:    firstEnv([]string{"JWT_SECRET"}, "dev-secret"),
		APIKeyHeader: firstEnv([]string{"API_KEY_HEADER"}, "X-API-Key"),
		TokenTTL:     ttl,
	}
}

func firstEnv(keys []string, fallback string) string {
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok && value != "" {
			return value
		}
	}
	return fallback
}

func parsePort(raw string) int {
	port, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("invalid port %q, defaulting to 8080: %v", raw, err)
		return 8080
	}
	return port
}

func parseDuration(raw string) time.Duration {
	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("invalid duration %q, defaulting to 1h: %v", raw, err)
		return time.Hour
	}
	return d
}

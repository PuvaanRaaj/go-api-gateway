package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	Port     int
	BackendA string
	BackendB string
}

func Load() *Config {
	port := parsePort(firstEnv([]string{"GATEWAY_PORT", "PORT"}, "8080"))

	return &Config{
		Port:     port,
		BackendA: firstEnv([]string{"BACKEND_A_URL", "BACKEND_A"}, "http://service-a:8080"),
		BackendB: firstEnv([]string{"BACKEND_B_URL", "BACKEND_B"}, "http://service-b:8080"),
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

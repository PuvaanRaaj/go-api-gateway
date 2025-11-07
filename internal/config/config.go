package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port     int
	BackendA string
	BackendB string
}

func Load() *Config {
	port, _ := strconv.Atoi(getEnv("PORT", "8080"))

	return &Config{
		Port:     port,
		BackendA: getEnv("BACKEND_A", "http://localhost:8081"),
		BackendB: getEnv("BACKEND_B", "http://localhost:8082"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

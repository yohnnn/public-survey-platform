package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	GRPCAddr         string
	DatabaseURL      string
	AuthGRPCEndpoint string
}

func Load() (Config, error) {
	cfg := Config{
		GRPCAddr:         getEnv("GRPC_ADDR", ":50052"),
		DatabaseURL:      strings.TrimSpace(getEnv("DATABASE_URL", "")),
		AuthGRPCEndpoint: strings.TrimSpace(getEnv("AUTH_GRPC_ENDPOINT", "")),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.AuthGRPCEndpoint == "" {
		return Config{}, fmt.Errorf("AUTH_GRPC_ENDPOINT is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

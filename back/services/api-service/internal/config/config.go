package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr         string
	AuthGRPCEndpoint string
	PollGRPCEndpoint string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:         getEnv("HTTP_ADDR", ":8080"),
		AuthGRPCEndpoint: strings.TrimSpace(getEnv("AUTH_GRPC_ENDPOINT", "")),
		PollGRPCEndpoint: strings.TrimSpace(getEnv("POLL_GRPC_ENDPOINT", "")),
	}

	if cfg.AuthGRPCEndpoint == "" {
		return Config{}, fmt.Errorf("AUTH_GRPC_ENDPOINT is required")
	}
	if cfg.PollGRPCEndpoint == "" {
		return Config{}, fmt.Errorf("POLL_GRPC_ENDPOINT is required")
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

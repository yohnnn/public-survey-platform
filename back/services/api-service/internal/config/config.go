package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr              string
	AllowedOrigins        []string
	AuthGRPCEndpoint      string
	PollGRPCEndpoint      string
	VoteGRPCEndpoint      string
	FeedGRPCEndpoint      string
	AnalyticsGRPCEndpoint string
	RealtimeGRPCEndpoint  string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:              getEnv("HTTP_ADDR", ":8080"),
		AllowedOrigins:        getEnvCSV("ALLOWED_ORIGINS"),
		AuthGRPCEndpoint:      strings.TrimSpace(getEnv("AUTH_GRPC_ENDPOINT", "")),
		PollGRPCEndpoint:      strings.TrimSpace(getEnv("POLL_GRPC_ENDPOINT", "")),
		VoteGRPCEndpoint:      strings.TrimSpace(getEnv("VOTE_GRPC_ENDPOINT", "")),
		FeedGRPCEndpoint:      strings.TrimSpace(getEnv("FEED_GRPC_ENDPOINT", "")),
		AnalyticsGRPCEndpoint: strings.TrimSpace(getEnv("ANALYTICS_GRPC_ENDPOINT", "")),
		RealtimeGRPCEndpoint:  strings.TrimSpace(getEnv("REALTIME_GRPC_ENDPOINT", "")),
	}

	if len(cfg.AllowedOrigins) == 0 {
		cfg.AllowedOrigins = []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		}
	}

	if cfg.AuthGRPCEndpoint == "" {
		return Config{}, fmt.Errorf("AUTH_GRPC_ENDPOINT is required")
	}
	if cfg.PollGRPCEndpoint == "" {
		return Config{}, fmt.Errorf("POLL_GRPC_ENDPOINT is required")
	}
	if cfg.VoteGRPCEndpoint == "" {
		return Config{}, fmt.Errorf("VOTE_GRPC_ENDPOINT is required")
	}
	if cfg.FeedGRPCEndpoint == "" {
		return Config{}, fmt.Errorf("FEED_GRPC_ENDPOINT is required")
	}
	if cfg.AnalyticsGRPCEndpoint == "" {
		return Config{}, fmt.Errorf("ANALYTICS_GRPC_ENDPOINT is required")
	}
	if cfg.RealtimeGRPCEndpoint == "" {
		return Config{}, fmt.Errorf("REALTIME_GRPC_ENDPOINT is required")
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

func getEnvCSV(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		v := strings.TrimSpace(part)
		if v == "" {
			continue
		}
		out = append(out, v)
	}

	return out
}

package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	GRPCAddr          string
	DatabaseURL       string
	AuthGRPCEndpoint  string
	PollGRPCEndpoint  string
	OutboxInterval    time.Duration
	OutboxBatchSize   int
	EventPublisher    string
	KafkaBrokers      []string
	KafkaTopicPrefix  string
	KafkaWriteTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		GRPCAddr:          getEnv("GRPC_ADDR", ":50053"),
		DatabaseURL:       strings.TrimSpace(getEnv("DATABASE_URL", "")),
		AuthGRPCEndpoint:  strings.TrimSpace(getEnv("AUTH_GRPC_ENDPOINT", "")),
		PollGRPCEndpoint:  strings.TrimSpace(getEnv("POLL_GRPC_ENDPOINT", "")),
		OutboxInterval:    getEnvDuration("OUTBOX_RELAY_INTERVAL", time.Second),
		OutboxBatchSize:   getEnvInt("OUTBOX_RELAY_BATCH_SIZE", 100),
		EventPublisher:    strings.ToLower(strings.TrimSpace(getEnv("EVENT_PUBLISHER", "log"))),
		KafkaBrokers:      getEnvCSV("KAFKA_BROKERS"),
		KafkaTopicPrefix:  strings.TrimSpace(getEnv("KAFKA_TOPIC_PREFIX", "")),
		KafkaWriteTimeout: getEnvDuration("KAFKA_WRITE_TIMEOUT", 5*time.Second),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.AuthGRPCEndpoint == "" {
		return Config{}, fmt.Errorf("AUTH_GRPC_ENDPOINT is required")
	}
	if cfg.PollGRPCEndpoint == "" {
		return Config{}, fmt.Errorf("POLL_GRPC_ENDPOINT is required")
	}
	if cfg.EventPublisher != "log" && cfg.EventPublisher != "kafka" {
		return Config{}, fmt.Errorf("EVENT_PUBLISHER must be one of: log, kafka")
	}
	if cfg.EventPublisher == "kafka" && len(cfg.KafkaBrokers) == 0 {
		return Config{}, fmt.Errorf("KAFKA_BROKERS is required when EVENT_PUBLISHER=kafka")
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

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func getEnvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
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

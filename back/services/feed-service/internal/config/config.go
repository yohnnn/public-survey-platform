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
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
	KafkaBrokers      []string
	KafkaTopicPrefix  string
	KafkaReadTimeout  time.Duration
	KafkaCommitPeriod time.Duration
	KafkaGroupID      string
}

func Load() (Config, error) {
	cfg := Config{
		GRPCAddr:          getEnv("GRPC_ADDR", ":50054"),
		DatabaseURL:       strings.TrimSpace(getEnv("DATABASE_URL", "")),
		AuthGRPCEndpoint:  strings.TrimSpace(getEnv("AUTH_GRPC_ENDPOINT", "auth-service:50051")),
		RedisAddr:         strings.TrimSpace(getEnv("REDIS_ADDR", "")),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           getEnvInt("REDIS_DB", 0),
		KafkaBrokers:      getEnvCSV("KAFKA_BROKERS"),
		KafkaTopicPrefix:  strings.TrimSpace(getEnv("KAFKA_TOPIC_PREFIX", "")),
		KafkaReadTimeout:  getEnvDuration("KAFKA_READ_TIMEOUT", 10*time.Second),
		KafkaCommitPeriod: getEnvDuration("KAFKA_COMMIT_PERIOD", time.Second),
		KafkaGroupID:      strings.TrimSpace(getEnv("KAFKA_GROUP_ID", "feed-service")),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if len(cfg.KafkaBrokers) == 0 {
		return Config{}, fmt.Errorf("KAFKA_BROKERS is required")
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

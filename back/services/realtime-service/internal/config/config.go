package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	GRPCAddr             string
	HTTPAddr             string
	AuthGRPCEndpoint     string
	AllowedOrigins       []string
	MaxConnectionsPerIP  int
	ConnectRatePerMinute int
	KafkaBrokers         []string
	KafkaTopicPrefix     string
	KafkaReadTimeout     time.Duration
	KafkaCommitPeriod    time.Duration
	KafkaGroupID         string
	StreamBuffer         int
	EventDedupTTL        time.Duration
	SSEHeartbeatInterval time.Duration
	WSPingInterval       time.Duration
	WSWriteTimeout       time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		GRPCAddr:             getEnv("GRPC_ADDR", ":50056"),
		HTTPAddr:             getEnv("HTTP_ADDR", ":8081"),
		AuthGRPCEndpoint:     strings.TrimSpace(getEnv("AUTH_GRPC_ENDPOINT", "")),
		AllowedOrigins:       getEnvCSV("ALLOWED_ORIGINS"),
		MaxConnectionsPerIP:  getEnvInt("MAX_CONNECTIONS_PER_IP", 30),
		ConnectRatePerMinute: getEnvInt("CONNECT_RATE_PER_MINUTE", 120),
		KafkaBrokers:         getEnvCSV("KAFKA_BROKERS"),
		KafkaTopicPrefix:     strings.TrimSpace(getEnv("KAFKA_TOPIC_PREFIX", "")),
		KafkaReadTimeout:     getEnvDuration("KAFKA_READ_TIMEOUT", 10*time.Second),
		KafkaCommitPeriod:    getEnvDuration("KAFKA_COMMIT_PERIOD", time.Second),
		KafkaGroupID:         strings.TrimSpace(getEnv("KAFKA_GROUP_ID", "realtime-service")),
		StreamBuffer:         getEnvInt("STREAM_BUFFER", 256),
		EventDedupTTL:        getEnvDuration("EVENT_DEDUP_TTL", 2*time.Minute),
		SSEHeartbeatInterval: getEnvDuration("SSE_HEARTBEAT_INTERVAL", 20*time.Second),
		WSPingInterval:       getEnvDuration("WS_PING_INTERVAL", 25*time.Second),
		WSWriteTimeout:       getEnvDuration("WS_WRITE_TIMEOUT", 5*time.Second),
	}

	if len(cfg.AllowedOrigins) == 0 {
		cfg.AllowedOrigins = []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
			"http://localhost:8080",
			"http://127.0.0.1:8080",
		}
	}

	if cfg.AuthGRPCEndpoint == "" {
		return Config{}, fmt.Errorf("AUTH_GRPC_ENDPOINT is required")
	}
	if len(cfg.KafkaBrokers) == 0 {
		return Config{}, fmt.Errorf("KAFKA_BROKERS is required")
	}
	if strings.TrimSpace(cfg.KafkaGroupID) == "" {
		return Config{}, fmt.Errorf("KAFKA_GROUP_ID is required")
	}
	if cfg.MaxConnectionsPerIP <= 0 {
		cfg.MaxConnectionsPerIP = 30
	}
	if cfg.ConnectRatePerMinute <= 0 {
		cfg.ConnectRatePerMinute = 120
	}
	if cfg.StreamBuffer <= 0 {
		cfg.StreamBuffer = 256
	}
	if cfg.EventDedupTTL <= 0 {
		cfg.EventDedupTTL = 2 * time.Minute
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
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
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

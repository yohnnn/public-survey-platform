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
	UserGRPCEndpoint  string
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
	OutboxInterval    time.Duration
	OutboxBatchSize   int
	EventPublisher    string
	KafkaBrokers      []string
	KafkaTopicPrefix  string
	KafkaReadTimeout  time.Duration
	KafkaCommitPeriod time.Duration
	KafkaGroupID      string
	KafkaWriteTimeout time.Duration
	MinIOEndpoint     string
	MinIOAccessKey    string
	MinIOSecretKey    string
	MinIOBucket       string
	MinIOUseSSL       bool
	MinIOPublicBase   string
	MinIOPresignTTL   time.Duration
	MinIOMaxFileBytes int64
}

func Load() (Config, error) {
	cfg := Config{
		GRPCAddr:          getEnv("GRPC_ADDR", ":50052"),
		DatabaseURL:       strings.TrimSpace(getEnv("DATABASE_URL", "")),
		UserGRPCEndpoint:  strings.TrimSpace(getEnv("USER_GRPC_ENDPOINT", "")),
		RedisAddr:         strings.TrimSpace(getEnv("REDIS_ADDR", "")),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           getEnvInt("REDIS_DB", 0),
		OutboxInterval:    getEnvDuration("OUTBOX_RELAY_INTERVAL", time.Second),
		OutboxBatchSize:   getEnvInt("OUTBOX_RELAY_BATCH_SIZE", 100),
		EventPublisher:    strings.ToLower(strings.TrimSpace(getEnv("EVENT_PUBLISHER", "log"))),
		KafkaBrokers:      getEnvCSV("KAFKA_BROKERS"),
		KafkaTopicPrefix:  strings.TrimSpace(getEnv("KAFKA_TOPIC_PREFIX", "")),
		KafkaReadTimeout:  getEnvDuration("KAFKA_READ_TIMEOUT", 10*time.Second),
		KafkaCommitPeriod: getEnvDuration("KAFKA_COMMIT_PERIOD", time.Second),
		KafkaGroupID:      strings.TrimSpace(getEnv("KAFKA_GROUP_ID", "poll-service")),
		KafkaWriteTimeout: getEnvDuration("KAFKA_WRITE_TIMEOUT", 5*time.Second),
		MinIOEndpoint:     strings.TrimSpace(getEnv("MINIO_ENDPOINT", "")),
		MinIOAccessKey:    strings.TrimSpace(getEnv("MINIO_ACCESS_KEY", "")),
		MinIOSecretKey:    strings.TrimSpace(getEnv("MINIO_SECRET_KEY", "")),
		MinIOBucket:       strings.TrimSpace(getEnv("MINIO_BUCKET", "")),
		MinIOUseSSL:       getEnvBool("MINIO_USE_SSL", false),
		MinIOPublicBase:   strings.TrimSpace(getEnv("MINIO_PUBLIC_BASE_URL", "")),
		MinIOPresignTTL:   getEnvDuration("MINIO_PRESIGN_TTL", 15*time.Minute),
		MinIOMaxFileBytes: getEnvInt64("MINIO_MAX_FILE_BYTES", 10*1024*1024),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.UserGRPCEndpoint == "" {
		return Config{}, fmt.Errorf("USER_GRPC_ENDPOINT is required")
	}
	if cfg.EventPublisher != "log" && cfg.EventPublisher != "kafka" {
		return Config{}, fmt.Errorf("EVENT_PUBLISHER must be one of: log, kafka")
	}
	if len(cfg.KafkaBrokers) == 0 {
		return Config{}, fmt.Errorf("KAFKA_BROKERS is required for poll-service vote event consumer")
	}
	if strings.TrimSpace(cfg.KafkaGroupID) == "" {
		return Config{}, fmt.Errorf("KAFKA_GROUP_ID is required")
	}

	if cfg.MinIOPresignTTL <= 0 {
		return Config{}, fmt.Errorf("MINIO_PRESIGN_TTL must be > 0")
	}
	if cfg.MinIOMaxFileBytes <= 0 {
		return Config{}, fmt.Errorf("MINIO_MAX_FILE_BYTES must be > 0")
	}

	hasMinIOConfig := cfg.MinIOEndpoint != "" || cfg.MinIOAccessKey != "" || cfg.MinIOSecretKey != "" || cfg.MinIOBucket != "" || cfg.MinIOPublicBase != ""
	if hasMinIOConfig {
		if cfg.MinIOEndpoint == "" {
			return Config{}, fmt.Errorf("MINIO_ENDPOINT is required when MinIO is configured")
		}
		if cfg.MinIOAccessKey == "" {
			return Config{}, fmt.Errorf("MINIO_ACCESS_KEY is required when MinIO is configured")
		}
		if cfg.MinIOSecretKey == "" {
			return Config{}, fmt.Errorf("MINIO_SECRET_KEY is required when MinIO is configured")
		}
		if cfg.MinIOBucket == "" {
			return Config{}, fmt.Errorf("MINIO_BUCKET is required when MinIO is configured")
		}
		if cfg.MinIOPublicBase == "" {
			return Config{}, fmt.Errorf("MINIO_PUBLIC_BASE_URL is required when MinIO is configured")
		}
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

func getEnvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvInt64(key string, fallback int64) int64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}

	return parsed
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

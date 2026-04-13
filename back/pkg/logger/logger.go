package logger

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
)

const RequestIDHeader = "X-Request-Id"

// Logger is an application logger wrapper that exposes both slog and stdlog APIs.
type Logger struct {
	logger *slog.Logger
}

func NewJSON(service string) *Logger {
	return NewJSONWithWriter(service, os.Stdout)
}

func NewJSONWithWriter(service string, w io.Writer) *Logger {
	if w == nil {
		w = os.Stdout
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})
	sl := slog.New(handler)

	service = strings.TrimSpace(service)
	if service != "" {
		sl = sl.With("service", service)
	}

	return &Logger{logger: sl}
}

func (l *Logger) Slog() *slog.Logger {
	if l == nil || l.logger == nil {
		return slog.Default()
	}
	return l.logger
}

func (l *Logger) StdLogger() *log.Logger {
	return slog.NewLogLogger(l.Slog().Handler(), slog.LevelInfo)
}

func NewRequestID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(b[:])
}

func EnsureRequestID(raw string) string {
	requestID := strings.TrimSpace(raw)
	if requestID != "" {
		return requestID
	}

	requestID = NewRequestID()
	if requestID != "" {
		return requestID
	}

	return "unknown"
}

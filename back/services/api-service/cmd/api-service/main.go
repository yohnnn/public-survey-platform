package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	analyticsv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/analytics/v1"
	authv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/auth/v1"
	feedv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/feed/v1"
	pollv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/poll/v1"
	votev1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/vote/v1"
	applogger "github.com/yohnnn/public-survey-platform/back/pkg/logger"
	"github.com/yohnnn/public-survey-platform/back/services/api-service/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "psp_api_http_requests_total",
			Help: "Total number of HTTP requests handled by api-service gateway.",
		},
		[]string{"method", "status_code"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "psp_api_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds for api-service gateway.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "status_code"},
	)
)

func main() {
	serviceLogger := applogger.NewJSON("api-service")
	logger := serviceLogger.StdLogger()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mux := runtime.NewServeMux(
		runtime.WithMetadata(func(_ context.Context, req *http.Request) metadata.MD {
			md := metadata.MD{}

			authorization := strings.TrimSpace(req.Header.Get("Authorization"))
			if authorization != "" {
				md.Set("authorization", authorization)
			}

			xRequestID := strings.TrimSpace(req.Header.Get("X-Request-Id"))
			if xRequestID != "" {
				md.Set("x-request-id", xRequestID)
			}

			return md
		}),
	)

	dialOptions := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := authv1.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, cfg.AuthGRPCEndpoint, dialOptions); err != nil {
		logger.Fatalf("register auth gateway handlers: %v", err)
	}
	if err := pollv1.RegisterPollServiceHandlerFromEndpoint(ctx, mux, cfg.PollGRPCEndpoint, dialOptions); err != nil {
		logger.Fatalf("register poll gateway handlers: %v", err)
	}
	if err := votev1.RegisterVoteServiceHandlerFromEndpoint(ctx, mux, cfg.VoteGRPCEndpoint, dialOptions); err != nil {
		logger.Fatalf("register vote gateway handlers: %v", err)
	}
	if err := feedv1.RegisterFeedServiceHandlerFromEndpoint(ctx, mux, cfg.FeedGRPCEndpoint, dialOptions); err != nil {
		logger.Fatalf("register feed gateway handlers: %v", err)
	}
	if err := analyticsv1.RegisterAnalyticsServiceHandlerFromEndpoint(ctx, mux, cfg.AnalyticsGRPCEndpoint, dialOptions); err != nil {
		logger.Fatalf("register analytics gateway handlers: %v", err)
	}

	metricsHandler := promhttp.Handler()

	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/metrics" {
			metricsHandler.ServeHTTP(w, r)
			return
		}
		mux.ServeHTTP(w, r)
	})
	handler = corsMiddleware(newOriginPolicy(cfg.AllowedOrigins), handler)
	handler = requestLoggingMiddleware(serviceLogger.Slog(), handler)

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Printf("HTTP gateway started on %s", cfg.HTTPAddr)
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if shutdownErr := httpServer.Shutdown(shutdownCtx); shutdownErr != nil {
			logger.Printf("http shutdown error: %v", shutdownErr)
		}
	case serveErr := <-errCh:
		if serveErr != nil && serveErr != http.ErrServerClosed {
			logger.Fatalf("http serve: %v", serveErr)
		}
	}
}

func requestLoggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := applogger.EnsureRequestID(r.Header.Get(applogger.RequestIDHeader))
		r.Header.Set(applogger.RequestIDHeader, requestID)
		w.Header().Set(applogger.RequestIDHeader, requestID)

		start := time.Now()
		wrapped := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		statusCode := strconv.Itoa(wrapped.statusCode)
		httpRequestsTotal.WithLabelValues(r.Method, statusCode).Inc()
		httpRequestDuration.WithLabelValues(r.Method, statusCode).Observe(time.Since(start).Seconds())

		logger.InfoContext(r.Context(), "http request completed",
			"component", "http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration_ms", float64(time.Since(start).Microseconds())/1000,
			"request_id", requestID,
		)
	})
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

type originPolicy struct {
	allowAll bool
	allowed  map[string]struct{}
}

func newOriginPolicy(origins []string) originPolicy {
	policy := originPolicy{allowed: make(map[string]struct{}, len(origins))}
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			policy.allowAll = true
			continue
		}
		policy.allowed[origin] = struct{}{}
	}
	return policy
}

func (p originPolicy) Allow(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return true
	}
	if p.allowAll {
		return true
	}
	_, ok := p.allowed[origin]
	return ok
}

func corsMiddleware(policy originPolicy, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))

		if origin != "" {
			if !policy.Allow(origin) {
				http.Error(w, "origin is not allowed", http.StatusForbidden)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-Id")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

			if strings.EqualFold(strings.TrimSpace(r.Header.Get("Access-Control-Request-Private-Network")), "true") {
				w.Header().Set("Access-Control-Allow-Private-Network", "true")
				w.Header().Add("Vary", "Access-Control-Request-Private-Network")
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

package metrics

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type Logger interface {
	Printf(format string, v ...any)
}

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of completed HTTP requests.",
		},
		[]string{"service", "method", "path", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "path", "status"},
	)
	httpRequestsInFlight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being handled.",
		},
		[]string{"service", "method", "path"},
	)

	grpcRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_server_requests_total",
			Help: "Total number of completed gRPC requests.",
		},
		[]string{"service", "grpc_service", "grpc_method", "code"},
	)
	grpcRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_server_request_duration_seconds",
			Help:    "gRPC request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "grpc_service", "grpc_method", "code"},
	)
	grpcRequestsInFlight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "grpc_server_requests_in_flight",
			Help: "Current number of gRPC requests being handled.",
		},
		[]string{"service", "grpc_service", "grpc_method"},
	)
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		httpRequestsInFlight,
		grpcRequestsTotal,
		grpcRequestDuration,
		grpcRequestsInFlight,
	)
}

func StartServerFromEnv(ctx context.Context, defaultAddr string, logger Logger) *http.Server {
	addr := strings.TrimSpace(os.Getenv("METRICS_ADDR"))
	if addr == "" {
		addr = strings.TrimSpace(defaultAddr)
	}
	return StartServer(ctx, addr, logger)
}

func StartServer(ctx context.Context, addr string, logger Logger) *http.Server {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil
	}
	if logger == nil {
		logger = log.Default()
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Printf("metrics server started on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Printf("metrics server error: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Printf("metrics server shutdown error: %v", err)
		}
	}()

	return server
}

func HTTPMiddleware(service string, next http.Handler) http.Handler {
	service = strings.TrimSpace(service)
	if service == "" {
		service = "unknown"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := normalizeHTTPPath(r.URL.Path)
		method := r.Method
		start := time.Now()
		wrapped := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		httpRequestsInFlight.WithLabelValues(service, method, path).Inc()
		defer httpRequestsInFlight.WithLabelValues(service, method, path).Dec()

		next.ServeHTTP(wrapped, r)

		statusCode := strconv.Itoa(wrapped.statusCode)
		duration := time.Since(start).Seconds()
		httpRequestsTotal.WithLabelValues(service, method, path, statusCode).Inc()
		httpRequestDuration.WithLabelValues(service, method, path, statusCode).Observe(duration)
	})
}

func UnaryServerInterceptor(service string) grpc.UnaryServerInterceptor {
	service = strings.TrimSpace(service)
	if service == "" {
		service = "unknown"
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		grpcService, grpcMethod := splitFullMethod(info.FullMethod)
		start := time.Now()

		grpcRequestsInFlight.WithLabelValues(service, grpcService, grpcMethod).Inc()
		resp, err := handler(ctx, req)
		grpcRequestsInFlight.WithLabelValues(service, grpcService, grpcMethod).Dec()

		code := status.Code(err).String()
		duration := time.Since(start).Seconds()
		grpcRequestsTotal.WithLabelValues(service, grpcService, grpcMethod, code).Inc()
		grpcRequestDuration.WithLabelValues(service, grpcService, grpcMethod, code).Observe(duration)

		return resp, err
	}
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func normalizeHTTPPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return "/"
	}

	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if segment == "" {
			continue
		}
		if looksLikePathID(segment) {
			segments[i] = "{id}"
		}
	}
	return strings.Join(segments, "/")
}

func looksLikePathID(segment string) bool {
	if segment == "" {
		return false
	}
	if _, err := strconv.ParseInt(segment, 10, 64); err == nil {
		return true
	}

	if len(segment) < 12 {
		return false
	}

	hasDigit := false
	for _, r := range segment {
		if r >= '0' && r <= '9' {
			hasDigit = true
			continue
		}
		if (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') || r == '-' {
			continue
		}
		return false
	}
	return hasDigit
}

func splitFullMethod(fullMethod string) (string, string) {
	fullMethod = strings.Trim(strings.TrimSpace(fullMethod), "/")
	if fullMethod == "" {
		return "unknown", "unknown"
	}

	parts := strings.Split(fullMethod, "/")
	if len(parts) == 1 {
		return parts[0], "unknown"
	}
	return parts[0], parts[1]
}

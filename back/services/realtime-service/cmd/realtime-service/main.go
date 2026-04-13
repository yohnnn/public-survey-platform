package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	authv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/auth/v1"
	realtimev1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/realtime/v1"
	"github.com/yohnnn/public-survey-platform/back/pkg/events"
	"github.com/yohnnn/public-survey-platform/back/pkg/grpcinterceptor"
	applogger "github.com/yohnnn/public-survey-platform/back/pkg/logger"
	"github.com/yohnnn/public-survey-platform/back/services/realtime-service/internal/config"
	grpcHandler "github.com/yohnnn/public-survey-platform/back/services/realtime-service/internal/handler/grpc"
	"github.com/yohnnn/public-survey-platform/back/services/realtime-service/internal/hub"
	realtimekafka "github.com/yohnnn/public-survey-platform/back/services/realtime-service/internal/messaging/kafka"
	"github.com/yohnnn/public-survey-platform/back/services/realtime-service/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type tokenValidator func(ctx context.Context, token string) (string, error)

func main() {
	serviceLogger := applogger.NewJSON("realtime-service")
	logger := serviceLogger.StdLogger()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	h := hub.New()
	realtimeSvc := service.NewRealtimeService(h, cfg.StreamBuffer)

	authConn, err := grpc.NewClient(cfg.AuthGRPCEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("connect to auth service: %v", err)
	}
	defer authConn.Close()
	authClient := authv1.NewAuthServiceClient(authConn)

	validateToken := func(ctx context.Context, token string) (string, error) {
		resp, validateErr := authClient.ValidateToken(ctx, &authv1.ValidateTokenRequest{AccessToken: token})
		if validateErr != nil {
			return "", validateErr
		}
		if !resp.GetValid() {
			return "", errors.New("invalid token")
		}
		return resp.GetUserId(), nil
	}

	subscriber, err := events.NewKafkaSubscriber(events.KafkaSubscriberConfig{
		Brokers:      cfg.KafkaBrokers,
		GroupID:      cfg.KafkaGroupID,
		TopicPrefix:  cfg.KafkaTopicPrefix,
		ReadTimeout:  cfg.KafkaReadTimeout,
		CommitPeriod: cfg.KafkaCommitPeriod,
	})
	if err != nil {
		logger.Fatalf("create kafka subscriber: %v", err)
	}

	consumer := realtimekafka.NewRealtimeConsumer(subscriber, realtimeSvc, logger, cfg.EventDedupTTL)
	consumerDone := make(chan struct{})
	go func() {
		defer close(consumerDone)
		if runErr := consumer.Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			logger.Printf("consumer stopped: %v", runErr)
		}
	}()

	authInterceptor := grpcinterceptor.UnaryAuthInterceptor(validateToken, nil)
	streamAuth := streamAuthInterceptor(validateToken, nil)
	loggingInterceptor := grpcinterceptor.UnaryLoggingInterceptor(serviceLogger.Slog())

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(loggingInterceptor, authInterceptor),
		grpc.ChainStreamInterceptor(streamAuth),
	)
	realtimev1.RegisterRealtimeServiceServer(grpcServer, grpcHandler.NewHandler(realtimeSvc))
	reflection.Register(grpcServer)

	grpcLis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		logger.Fatalf("listen grpc %s: %v", cfg.GRPCAddr, err)
	}

	httpServer := buildHTTPServer(cfg, realtimeSvc, validateToken, serviceLogger.Slog())

	errCh := make(chan error, 2)
	go func() {
		logger.Printf("gRPC server started on %s", cfg.GRPCAddr)
		errCh <- grpcServer.Serve(grpcLis)
	}()
	go func() {
		logger.Printf("HTTP realtime server started on %s", cfg.HTTPAddr)
		errCh <- httpServer.ListenAndServe()
	}()

	var serveErr error
	select {
	case <-ctx.Done():
		logger.Println("shutdown signal received")
	case serveErr = <-errCh:
		if serveErr != nil && serveErr != http.ErrServerClosed && !errors.Is(serveErr, grpc.ErrServerStopped) {
			logger.Printf("serve error: %v", serveErr)
		}
		stop()
	}

	gracefulStopGRPC(logger, grpcServer, 10*time.Second)
	shutdownHTTPServer(logger, httpServer, 10*time.Second)
	waitForShutdown(logger, "realtime consumer", consumerDone, 10*time.Second)

	if serveErr != nil && serveErr != http.ErrServerClosed && !errors.Is(serveErr, grpc.ErrServerStopped) {
		logger.Fatal("service stopped with serve error")
	}
}

func gracefulStopGRPC(logger *log.Logger, srv *grpc.Server, timeout time.Duration) {
	if srv == nil {
		return
	}

	done := make(chan struct{})
	go func() {
		srv.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		logger.Printf("grpc graceful stop timed out after %s, forcing stop", timeout)
		srv.Stop()
	}
}

func shutdownHTTPServer(logger *log.Logger, server *http.Server, timeout time.Duration) {
	if server == nil {
		return
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Printf("http shutdown error: %v", err)
	}
}

func waitForShutdown(logger *log.Logger, name string, done <-chan struct{}, timeout time.Duration) {
	if done == nil {
		return
	}

	select {
	case <-done:
	case <-time.After(timeout):
		logger.Printf("timeout while waiting for %s shutdown after %s", name, timeout)
	}
}

func buildHTTPServer(cfg config.Config, svc service.RealtimeService, validate tokenValidator, logger *slog.Logger) *http.Server {
	originPolicy := newOriginPolicy(cfg.AllowedOrigins)
	limiter := newConnectionLimiter(cfg.MaxConnectionsPerIP, cfg.ConnectRatePerMinute)

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return originPolicy.Allow(strings.TrimSpace(r.Header.Get("Origin")))
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/v1/realtime/polls/", func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if !originPolicy.Allow(origin) {
			http.Error(w, "origin is not allowed", http.StatusForbidden)
			return
		}
		setCORSHeaders(w, origin, originPolicy)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := authorizeHTTPRequest(r.Context(), r, validate); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		clientIP := extractClientIP(r)
		if !limiter.Acquire(clientIP) {
			http.Error(w, "too many connections", http.StatusTooManyRequests)
			return
		}
		defer limiter.Release(clientIP)

		pollID, ok := parsePollIDFromSSEPath(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		handleSSEStream(w, r, svc, pollID, cfg.SSEHeartbeatInterval)
	})

	mux.HandleFunc("/v1/ws", func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if !originPolicy.Allow(origin) {
			http.Error(w, "origin is not allowed", http.StatusForbidden)
			return
		}
		setCORSHeaders(w, origin, originPolicy)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := authorizeHTTPRequest(r.Context(), r, validate); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		clientIP := extractClientIP(r)
		if !limiter.Acquire(clientIP) {
			http.Error(w, "too many connections", http.StatusTooManyRequests)
			return
		}
		defer limiter.Release(clientIP)

		if websocket.IsWebSocketUpgrade(r) {
			pollID := strings.TrimSpace(r.URL.Query().Get("poll_id"))
			if pollID == "" {
				http.Error(w, "poll_id query param is required", http.StatusBadRequest)
				return
			}
			handleWebSocketStream(w, r, svc, pollID, cfg.WSPingInterval, cfg.WSWriteTimeout, &upgrader)
			return
		}

		connectionID, err := svc.WSHandshake(r.Context())
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"connection_id": connectionID})
	})

	return &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           requestLoggingMiddleware(logger, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func handleSSEStream(w http.ResponseWriter, r *http.Request, svc service.RealtimeService, pollID string, heartbeatInterval time.Duration) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	updates, unsubscribe, err := svc.SubscribePollUpdates(r.Context(), pollID)
	if err != nil {
		http.Error(w, "invalid poll_id", http.StatusBadRequest)
		return
	}
	defer unsubscribe()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case update, ok := <-updates:
			if !ok {
				return
			}
			payload, err := json.Marshal(update)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "event: poll_update\ndata: %s\n\n", payload)
			flusher.Flush()
		case <-ticker.C:
			_, _ = w.Write([]byte(": heartbeat\n\n"))
			flusher.Flush()
		}
	}
}

func handleWebSocketStream(
	w http.ResponseWriter,
	r *http.Request,
	svc service.RealtimeService,
	pollID string,
	pingInterval time.Duration,
	writeTimeout time.Duration,
	upgrader *websocket.Upgrader,
) {
	updates, unsubscribe, err := svc.SubscribePollUpdates(r.Context(), pollID)
	if err != nil {
		http.Error(w, "invalid poll_id", http.StatusBadRequest)
		return
	}
	defer unsubscribe()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case update, ok := <-updates:
			if !ok {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := conn.WriteJSON(update); err != nil {
				return
			}
		case <-pingTicker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func parsePollIDFromSSEPath(path string) (string, bool) {
	const prefix = "/v1/realtime/polls/"
	const suffix = "/stream"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}
	pollID := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	pollID = strings.Trim(pollID, "/")
	if pollID == "" {
		return "", false
	}
	return pollID, true
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

func streamAuthInterceptor(validate tokenValidator, publicMethods map[string]struct{}) grpc.StreamServerInterceptor {
	if publicMethods == nil {
		publicMethods = map[string]struct{}{}
	}

	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if _, ok := publicMethods[info.FullMethod]; ok {
			return handler(srv, ss)
		}

		token, err := extractBearerTokenFromMetadata(ss.Context())
		if err != nil {
			return status.Error(codes.Unauthenticated, "missing or invalid bearer token")
		}

		if _, err := validate(ss.Context(), token); err != nil {
			return status.Error(codes.Unauthenticated, "invalid token")
		}

		return handler(srv, ss)
	}
}

func authorizeHTTPRequest(ctx context.Context, r *http.Request, validate tokenValidator) error {
	token := extractBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		token = strings.TrimSpace(r.URL.Query().Get("access_token"))
	}
	if token == "" {
		return errors.New("missing token")
	}

	_, err := validate(ctx, token)
	return err
}

func extractBearerTokenFromMetadata(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", errors.New("missing authorization")
	}

	token := extractBearerToken(values[0])
	if token == "" {
		return "", errors.New("invalid bearer token")
	}

	return token, nil
}

func extractBearerToken(raw string) string {
	parts := strings.SplitN(strings.TrimSpace(raw), " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
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
	if len(policy.allowed) == 0 && !policy.allowAll {
		policy.allowAll = true
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

func setCORSHeaders(w http.ResponseWriter, origin string, policy originPolicy) {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return
	}
	if !policy.Allow(origin) {
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
}

type connectionLimiter struct {
	mu          sync.Mutex
	maxActive   int
	maxPerMin   int
	connections map[string]*connectionState
}

type connectionState struct {
	active      int
	windowStart time.Time
	requests    int
	lastSeen    time.Time
}

func newConnectionLimiter(maxActive, maxPerMin int) *connectionLimiter {
	if maxActive <= 0 {
		maxActive = 30
	}
	if maxPerMin <= 0 {
		maxPerMin = 120
	}

	return &connectionLimiter{
		maxActive:   maxActive,
		maxPerMin:   maxPerMin,
		connections: make(map[string]*connectionState),
	}
}

func (l *connectionLimiter) Acquire(clientIP string) bool {
	clientIP = strings.TrimSpace(clientIP)
	if clientIP == "" {
		clientIP = "unknown"
	}

	now := time.Now().UTC()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanup(now)

	state, ok := l.connections[clientIP]
	if !ok {
		state = &connectionState{windowStart: now}
		l.connections[clientIP] = state
	}

	if now.Sub(state.windowStart) >= time.Minute {
		state.windowStart = now
		state.requests = 0
	}

	if state.requests >= l.maxPerMin {
		return false
	}
	if state.active >= l.maxActive {
		return false
	}

	state.requests++
	state.active++
	state.lastSeen = now
	return true
}

func (l *connectionLimiter) Release(clientIP string) {
	clientIP = strings.TrimSpace(clientIP)
	if clientIP == "" {
		clientIP = "unknown"
	}

	now := time.Now().UTC()

	l.mu.Lock()
	defer l.mu.Unlock()

	state, ok := l.connections[clientIP]
	if !ok {
		return
	}

	if state.active > 0 {
		state.active--
	}
	state.lastSeen = now

	if state.active == 0 && now.Sub(state.windowStart) >= time.Minute {
		delete(l.connections, clientIP)
	}
}

func (l *connectionLimiter) cleanup(now time.Time) {
	for key, state := range l.connections {
		if state.active == 0 && now.Sub(state.lastSeen) > 2*time.Minute {
			delete(l.connections, key)
		}
	}
}

func extractClientIP(r *http.Request) string {
	if forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}

	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && strings.TrimSpace(host) != "" {
		return strings.TrimSpace(host)
	}

	return strings.TrimSpace(r.RemoteAddr)
}

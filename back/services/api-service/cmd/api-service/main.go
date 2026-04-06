package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	authv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/auth/v1"
	pollv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/poll/v1"
	votev1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/vote/v1"
	"github.com/yohnnn/public-survey-platform/back/services/api-service/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	logger := log.New(os.Stdout, "[api-service] ", log.LstdFlags|log.Lmicroseconds)

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

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}
		mux.ServeHTTP(w, r)
	})

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

package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	authv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/auth/v1"
	"github.com/yohnnn/public-survey-platform/back/pkg/grpcinterceptor"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/config"
	handlergrpc "github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/handler/grpc"
	"github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/repository/postgres"
	"github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/security"
	"github.com/yohnnn/public-survey-platform/back/services/auth-service/internal/service"
	"google.golang.org/grpc"
)

func main() {
	logger := log.New(os.Stdout, "[auth-service] ", log.LstdFlags|log.Lmicroseconds)

	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Fatalf("ping database: %v", err)
	}

	userRepo := postgres.NewUserRepository(pool)
	sessionRepo := postgres.NewSessionRepository(pool)
	txManager := tx.NewManager(pool)

	hasher := security.NewBcryptHasher(cfg.BcryptCost)
	tokenManager := security.NewJWTManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)

	authService := service.NewAuthService(
		userRepo,
		sessionRepo,
		*txManager,
		hasher,
		tokenManager,
		service.NewSystemClock(),
		service.NewRandomIDGenerator(),
	)

	handler := handlergrpc.NewHandler(authService)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcinterceptor.UnaryLoggingInterceptor(logger),
			grpcinterceptor.UnaryAuthInterceptor(authService.ValidateToken, map[string]struct{}{
				authv1.AuthService_Register_FullMethodName:      {},
				authv1.AuthService_Login_FullMethodName:         {},
				authv1.AuthService_RefreshToken_FullMethodName:  {},
				authv1.AuthService_ValidateToken_FullMethodName: {},
			}),
		),
	)
	authv1.RegisterAuthServiceServer(grpcServer, handler)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		logger.Fatalf("listen %s: %v", cfg.GRPCAddr, err)
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Printf("gRPC server started on %s", cfg.GRPCAddr)
		errCh <- grpcServer.Serve(lis)
	}()

	select {
	case <-ctx.Done():
		logger.Println("shutdown signal received")
		grpcServer.GracefulStop()
	case serveErr := <-errCh:
		if serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			logger.Fatalf("gRPC serve: %v", serveErr)
		}
	}
}

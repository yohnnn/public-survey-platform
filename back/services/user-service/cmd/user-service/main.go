package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	userv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/user/v1"
	"github.com/yohnnn/public-survey-platform/back/pkg/grpcinterceptor"
	applogger "github.com/yohnnn/public-survey-platform/back/pkg/logger"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/user-service/internal/config"
	handlergrpc "github.com/yohnnn/public-survey-platform/back/services/user-service/internal/handler/grpc"
	"github.com/yohnnn/public-survey-platform/back/services/user-service/internal/repository/postgres"
	"github.com/yohnnn/public-survey-platform/back/services/user-service/internal/security"
	"github.com/yohnnn/public-survey-platform/back/services/user-service/internal/service"
	"google.golang.org/grpc"
)

func main() {
	serviceLogger := applogger.NewJSON("user-service")
	logger := serviceLogger.StdLogger()

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

	userService := service.NewUserService(
		userRepo,
		sessionRepo,
		*txManager,
		hasher,
		tokenManager,
		service.NewSystemClock(),
		service.NewRandomIDGenerator(),
	)

	handler := handlergrpc.NewHandler(userService)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcinterceptor.UnaryLoggingInterceptor(serviceLogger.Slog()),
			grpcinterceptor.UnaryAuthInterceptor(userService.ValidateToken, map[string]struct{}{
				userv1.UserService_Register_FullMethodName:              {},
				userv1.UserService_Login_FullMethodName:                 {},
				userv1.UserService_RefreshToken_FullMethodName:          {},
				userv1.UserService_ValidateToken_FullMethodName:         {},
				userv1.UserService_GetPublicProfile_FullMethodName:      {},
				userv1.UserService_BatchGetUserSummaries_FullMethodName: {},
			}),
		),
	)
	userv1.RegisterUserServiceServer(grpcServer, handler)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		logger.Fatalf("listen %s: %v", cfg.GRPCAddr, err)
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Printf("gRPC server started on %s", cfg.GRPCAddr)
		errCh <- grpcServer.Serve(lis)
	}()

	var serveErr error
	select {
	case <-ctx.Done():
		logger.Println("shutdown signal received")
	case serveErr = <-errCh:
		if serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			logger.Printf("gRPC serve error: %v", serveErr)
		}
		stop()
	}

	gracefulStopGRPC(logger, grpcServer, 10*time.Second)

	if serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
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

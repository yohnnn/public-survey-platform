package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	authv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/auth/v1"
	pollv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/poll/v1"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/config"
	grpcHandler "github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/handler/grpc"
	grpcinterceptors "github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/handler/grpc/interceptors"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/repository/postgres"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/service"
)

func main() {
	logger := log.New(os.Stdout, "[poll-service] ", log.LstdFlags|log.Lmicroseconds)

	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	authConn, err := grpc.NewClient(cfg.AuthGRPCEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("connect to auth service: %v", err)
	}
	defer authConn.Close()
	authClient := authv1.NewAuthServiceClient(authConn)

	pollRepo := postgres.NewPollRepository(pool)
	tagRepo := postgres.NewTagRepository(pool)
	txMgr := tx.NewManager(pool)
	clock := service.NewSystemClock()
	idGen := service.NewRandomIDGenerator()

	pollSvc := service.NewPollService(pollRepo, tagRepo, *txMgr, clock, idGen)

	authInterceptor := grpcinterceptors.UnaryAuthInterceptor(authClient)
	loggingInterceptor := grpcinterceptors.UnaryLoggingInterceptor(logger)

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(loggingInterceptor, authInterceptor),
	)
	pollv1.RegisterPollServiceServer(srv, grpcHandler.NewHandler(pollSvc))
	reflection.Register(srv)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		logger.Fatalf("listen: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Printf("gRPC server started on %s", cfg.GRPCAddr)
		errCh <- srv.Serve(lis)
	}()

	select {
	case <-ctx.Done():
		srv.GracefulStop()
		pool.Close()
	case serveErr := <-errCh:
		if serveErr != nil {
			logger.Fatalf("grpc serve: %v", serveErr)
		}
	}
}

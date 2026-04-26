package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	pollv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/poll/v1"
	userv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/user/v1"
	votev1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/vote/v1"
	"github.com/yohnnn/public-survey-platform/back/pkg/events"
	"github.com/yohnnn/public-survey-platform/back/pkg/grpcinterceptor"
	applogger "github.com/yohnnn/public-survey-platform/back/pkg/logger"
	"github.com/yohnnn/public-survey-platform/back/pkg/outbox"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/vote-service/internal/config"
	grpcHandler "github.com/yohnnn/public-survey-platform/back/services/vote-service/internal/handler/grpc"
	"github.com/yohnnn/public-survey-platform/back/services/vote-service/internal/repository/postgres"
	"github.com/yohnnn/public-survey-platform/back/services/vote-service/internal/service"
)

func main() {
	serviceLogger := applogger.NewJSON("vote-service")
	logger := serviceLogger.StdLogger()

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

	authConn, err := grpc.NewClient(cfg.UserGRPCEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("connect to user service: %v", err)
	}
	defer authConn.Close()
	authClient := userv1.NewUserServiceClient(authConn)

	pollConn, err := grpc.NewClient(cfg.PollGRPCEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("connect to poll service: %v", err)
	}
	defer pollConn.Close()
	pollClient := pollv1.NewPollServiceClient(pollConn)

	voteRepo := postgres.NewVoteRepository(pool)
	outboxRepo := postgres.NewOutboxRepository(pool)
	txMgr := tx.NewManager(pool)
	voteSvc := service.NewVoteService(voteRepo, outboxRepo, authClient, pollClient, *txMgr, service.NewSystemClock())

	var publisher events.Publisher = events.NewLogPublisher(logger)
	if cfg.EventPublisher == "kafka" {
		kafkaPublisher, pubErr := events.NewKafkaPublisher(events.KafkaPublisherConfig{
			Brokers:      cfg.KafkaBrokers,
			TopicPrefix:  cfg.KafkaTopicPrefix,
			WriteTimeout: cfg.KafkaWriteTimeout,
		})
		if pubErr != nil {
			logger.Fatalf("create kafka publisher: %v", pubErr)
		}
		defer func() {
			if closeErr := kafkaPublisher.Close(); closeErr != nil {
				logger.Printf("close kafka publisher error: %v", closeErr)
			}
		}()
		publisher = kafkaPublisher
	}

	outboxRelay := outbox.NewRelay(outboxRepo, publisher, outbox.NewSystemClock(), logger, cfg.OutboxInterval, cfg.OutboxBatchSize)
	outboxDone := make(chan struct{})
	go func() {
		defer close(outboxDone)
		if runErr := outboxRelay.Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			logger.Printf("outbox relay stopped: %v", runErr)
		}
	}()

	authInterceptor := grpcinterceptor.UnaryAuthInterceptor(
		func(ctx context.Context, token string) (string, error) {
			resp, err := authClient.ValidateToken(ctx, &userv1.ValidateTokenRequest{AccessToken: token})
			if err != nil {
				return "", err
			}
			if !resp.GetValid() {
				return "", fmt.Errorf("token is not valid")
			}
			return resp.GetUserId(), nil
		},
		nil,
	)
	loggingInterceptor := grpcinterceptor.UnaryLoggingInterceptor(serviceLogger.Slog())

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(loggingInterceptor, authInterceptor),
	)
	votev1.RegisterVoteServiceServer(srv, grpcHandler.NewHandler(voteSvc))
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

	var serveErr error
	select {
	case <-ctx.Done():
		logger.Println("shutdown signal received")
	case serveErr = <-errCh:
		if serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			logger.Printf("grpc serve error: %v", serveErr)
		}
		stop()
	}

	gracefulStopGRPC(logger, srv, 10*time.Second)
	waitForShutdown(logger, "vote outbox relay", outboxDone, 10*time.Second)

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

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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/yohnnn/public-survey-platform/back/api/events"
	authv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/auth/v1"
	pollv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/poll/v1"
	votev1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/vote/v1"
	"github.com/yohnnn/public-survey-platform/back/pkg/outbox"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/vote-service/internal/config"
	grpcHandler "github.com/yohnnn/public-survey-platform/back/services/vote-service/internal/handler/grpc"
	grpcinterceptors "github.com/yohnnn/public-survey-platform/back/services/vote-service/internal/handler/grpc/interceptors"
	"github.com/yohnnn/public-survey-platform/back/services/vote-service/internal/repository/postgres"
	"github.com/yohnnn/public-survey-platform/back/services/vote-service/internal/service"
)

func main() {
	logger := log.New(os.Stdout, "[vote-service] ", log.LstdFlags|log.Lmicroseconds)

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

	pollConn, err := grpc.NewClient(cfg.PollGRPCEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("connect to poll service: %v", err)
	}
	defer pollConn.Close()
	pollClient := pollv1.NewPollServiceClient(pollConn)

	voteRepo := postgres.NewVoteRepository(pool)
	outboxRepo := postgres.NewOutboxRepository(pool)
	txMgr := tx.NewManager(pool)
	voteSvc := service.NewVoteService(voteRepo, outboxRepo, pollClient, *txMgr, service.NewSystemClock())

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
	go func() {
		if runErr := outboxRelay.Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			logger.Printf("outbox relay stopped: %v", runErr)
		}
	}()

	authInterceptor := grpcinterceptors.UnaryAuthInterceptor(authClient)
	loggingInterceptor := grpcinterceptors.UnaryLoggingInterceptor(logger)

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

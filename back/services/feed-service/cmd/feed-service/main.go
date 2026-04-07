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
	"google.golang.org/grpc/reflection"

	"github.com/yohnnn/public-survey-platform/back/api/events"
	feedv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/feed/v1"
	"github.com/yohnnn/public-survey-platform/back/pkg/grpcinterceptor"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/config"
	grpcHandler "github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/handler/grpc"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/repository/postgres"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/service"
)

func main() {
	logger := log.New(os.Stdout, "[feed-service] ", log.LstdFlags|log.Lmicroseconds)

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

	feedRepo := postgres.NewFeedRepository(pool)
	txMgr := tx.NewManager(pool)
	feedSvc := service.NewFeedService(feedRepo)

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

	consumer := service.NewFeedConsumer(subscriber, feedRepo, txMgr, logger)
	go func() {
		if runErr := consumer.Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			logger.Printf("consumer stopped: %v", runErr)
		}
	}()

	loggingInterceptor := grpcinterceptor.UnaryLoggingInterceptor(logger)

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(loggingInterceptor),
	)
	feedv1.RegisterFeedServiceServer(srv, grpcHandler.NewHandler(feedSvc))
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

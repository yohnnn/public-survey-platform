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
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	analyticsv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/analytics/v1"
	"github.com/yohnnn/public-survey-platform/back/pkg/events"
	"github.com/yohnnn/public-survey-platform/back/pkg/grpcinterceptor"
	applogger "github.com/yohnnn/public-survey-platform/back/pkg/logger"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/config"
	grpcHandler "github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/handler/grpc"
	analyticskafka "github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/messaging/kafka"
	"github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/repository/postgres"
	"github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/service"
)

func main() {
	serviceLogger := applogger.NewJSON("analytics-service")
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

	analyticsRepo := postgres.NewAnalyticsRepository(pool)
	analyticsSvc := service.NewAnalyticsService(analyticsRepo)
	txMgr := tx.NewManager(pool)

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

	consumer := analyticskafka.NewAnalyticsConsumer(subscriber, analyticsRepo, txMgr, logger)
	consumerDone := make(chan struct{})
	go func() {
		defer close(consumerDone)
		if runErr := consumer.Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			logger.Printf("consumer stopped: %v", runErr)
		}
	}()

	loggingInterceptor := grpcinterceptor.UnaryLoggingInterceptor(serviceLogger.Slog())
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(loggingInterceptor))

	analyticsv1.RegisterAnalyticsServiceServer(srv, grpcHandler.NewHandler(analyticsSvc))
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
	waitForShutdown(logger, "analytics consumer", consumerDone, 10*time.Second)

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

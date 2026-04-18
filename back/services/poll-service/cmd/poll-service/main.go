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
	"github.com/yohnnn/public-survey-platform/back/pkg/cache/redisstore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	authv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/auth/v1"
	pollv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/poll/v1"
	"github.com/yohnnn/public-survey-platform/back/pkg/events"
	"github.com/yohnnn/public-survey-platform/back/pkg/grpcinterceptor"
	applogger "github.com/yohnnn/public-survey-platform/back/pkg/logger"
	"github.com/yohnnn/public-survey-platform/back/pkg/outbox"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/config"
	grpcHandler "github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/handler/grpc"
	pollkafka "github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/messaging/kafka"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/repository/postgres"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/service"
	pollcache "github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/service/cache"
)

func main() {
	serviceLogger := applogger.NewJSON("poll-service")
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

	authConn, err := grpc.NewClient(cfg.AuthGRPCEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("connect to auth service: %v", err)
	}
	defer authConn.Close()
	authClient := authv1.NewAuthServiceClient(authConn)

	pollRepo := postgres.NewPollRepository(pool)
	tagRepo := postgres.NewTagRepository(pool)
	outboxRepo := postgres.NewOutboxRepository(pool)
	txMgr := tx.NewManager(pool)
	clock := service.NewSystemClock()
	idGen := service.NewRandomIDGenerator()

	pollSvc := service.NewPollService(pollRepo, tagRepo, outboxRepo, *txMgr, clock, idGen)
	if cfg.RedisAddr != "" {
		cacheStore := redisstore.New(redisstore.Config{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})

		if pingErr := cacheStore.Ping(ctx); pingErr != nil {
			logger.Printf("redis cache disabled: %v", pingErr)
			_ = cacheStore.Close()
		} else {
			defer func() {
				if closeErr := cacheStore.Close(); closeErr != nil {
					logger.Printf("close redis cache store error: %v", closeErr)
				}
			}()
			pollSvc = pollcache.NewPollService(pollSvc, cacheStore, pollcache.DefaultConfig())
		}
	}

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

	voteConsumer := pollkafka.NewPollConsumer(subscriber, pollRepo, txMgr, logger)

	outboxDone := make(chan struct{})
	go func() {
		defer close(outboxDone)
		if runErr := outboxRelay.Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			logger.Printf("outbox relay stopped: %v", runErr)
		}
	}()
	voteConsumerDone := make(chan struct{})
	go func() {
		defer close(voteConsumerDone)
		if runErr := voteConsumer.Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			logger.Printf("vote consumer stopped: %v", runErr)
		}
	}()

	authInterceptor := grpcinterceptor.UnaryAuthInterceptor(
		func(ctx context.Context, token string) (string, error) {
			resp, err := authClient.ValidateToken(ctx, &authv1.ValidateTokenRequest{AccessToken: token})
			if err != nil {
				return "", err
			}
			if !resp.GetValid() {
				return "", fmt.Errorf("token is not valid")
			}
			return resp.GetUserId(), nil
		},
		map[string]struct{}{
			pollv1.PollService_ListPolls_FullMethodName: {},
			pollv1.PollService_GetPoll_FullMethodName:   {},
			pollv1.PollService_ListTags_FullMethodName:  {},
		},
	)
	loggingInterceptor := grpcinterceptor.UnaryLoggingInterceptor(serviceLogger.Slog())

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
	waitForShutdown(logger, "poll outbox relay", outboxDone, 10*time.Second)
	waitForShutdown(logger, "poll vote consumer", voteConsumerDone, 10*time.Second)

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

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
	"github.com/yohnnn/public-survey-platform/back/pkg/cache/redisstore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	feedv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/feed/v1"
	userv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/user/v1"
	"github.com/yohnnn/public-survey-platform/back/pkg/events"
	"github.com/yohnnn/public-survey-platform/back/pkg/grpcinterceptor"
	applogger "github.com/yohnnn/public-survey-platform/back/pkg/logger"
	appmetrics "github.com/yohnnn/public-survey-platform/back/pkg/metrics"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/config"
	grpcHandler "github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/handler/grpc"
	feedkafka "github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/messaging/kafka"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/repository/postgres"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/service"
	feedcache "github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/service/cache"
)

func main() {
	serviceLogger := applogger.NewJSON("feed-service")
	logger := serviceLogger.StdLogger()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	appmetrics.StartServerFromEnv(ctx, ":9105", logger)

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	userConn, err := grpc.NewClient(cfg.UserGRPCEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("connect to user service: %v", err)
	}
	defer userConn.Close()
	userClient := userv1.NewUserServiceClient(userConn)

	feedRepo := postgres.NewFeedRepository(pool)
	txMgr := tx.NewManager(pool)
	feedSvc := service.NewFeedService(feedRepo, userClient)
	var flushFeedCache func(context.Context)
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
			feedSvc = feedcache.NewFeedService(feedSvc, userClient, cacheStore, feedcache.DefaultConfig())
			flushFeedCache = func(ctx context.Context) {
				if err := cacheStore.DeleteByPattern(ctx, "feed-service:*"); err != nil {
					logger.Printf("flush feed cache error: %v", err)
				}
			}
		}
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

	consumer := feedkafka.NewFeedConsumer(subscriber, feedRepo, txMgr, logger, flushFeedCache)
	consumerDone := make(chan struct{})
	go func() {
		defer close(consumerDone)
		if runErr := consumer.Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
			logger.Printf("consumer stopped: %v", runErr)
		}
	}()

	authInterceptor := grpcinterceptor.UnaryUserIDInterceptor(
		map[string]struct{}{
			feedv1.FeedService_GetFeed_FullMethodName:      {},
			feedv1.FeedService_GetTrending_FullMethodName:  {},
			feedv1.FeedService_GetUserPolls_FullMethodName: {},
		},
	)
	loggingInterceptor := grpcinterceptor.UnaryLoggingInterceptor(serviceLogger.Slog())

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(appmetrics.UnaryServerInterceptor("feed-service"), loggingInterceptor, authInterceptor),
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
	waitForShutdown(logger, "feed consumer", consumerDone, 10*time.Second)

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

package outbox

import (
	"context"
	"strings"
	"time"

	"github.com/yohnnn/public-survey-platform/back/pkg/events"
)

type Event struct {
	ID      string
	Topic   string
	Key     string
	Payload []byte
}

type Repository interface {
	ListUnpublished(ctx context.Context, limit int) ([]Event, error)
	MarkPublished(ctx context.Context, eventID string, publishedAt time.Time) error
	MarkFailed(ctx context.Context, eventID, reason string) error
}

type Clock interface {
	Now() time.Time
}

type Logger interface {
	Printf(format string, v ...any)
}

type Relay struct {
	repo      Repository
	publisher events.Publisher
	clock     Clock
	logger    Logger
	interval  time.Duration
	batchSize int
}

func NewRelay(repo Repository, publisher events.Publisher, clock Clock, logger Logger, interval time.Duration, batchSize int) *Relay {
	if interval <= 0 {
		interval = time.Second
	}
	if batchSize <= 0 {
		batchSize = 50
	}

	return &Relay{
		repo:      repo,
		publisher: publisher,
		clock:     clock,
		logger:    logger,
		interval:  interval,
		batchSize: batchSize,
	}
}

func (r *Relay) Run(ctx context.Context) error {
	if r == nil {
		return nil
	}

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		if err := r.FlushOnce(ctx); err != nil && r.logger != nil {
			r.logger.Printf("outbox relay flush error: %v", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (r *Relay) FlushOnce(ctx context.Context) error {
	if r == nil || r.repo == nil || r.publisher == nil || r.clock == nil {
		return nil
	}

	eventsBatch, err := r.repo.ListUnpublished(ctx, r.batchSize)
	if err != nil {
		return err
	}

	for _, item := range eventsBatch {
		err = r.publisher.Publish(ctx, events.Message{
			Topic:   item.Topic,
			Key:     item.Key,
			Payload: item.Payload,
		})
		if err != nil {
			reason := strings.TrimSpace(err.Error())
			if markErr := r.repo.MarkFailed(ctx, item.ID, reason); markErr != nil && r.logger != nil {
				r.logger.Printf("outbox mark failed error: %v", markErr)
			}
			continue
		}

		if markErr := r.repo.MarkPublished(ctx, item.ID, r.clock.Now().UTC()); markErr != nil {
			if r.logger != nil {
				r.logger.Printf("outbox mark published error: %v", markErr)
			}
		}
	}

	return nil
}

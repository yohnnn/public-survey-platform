package kafka

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/yohnnn/public-survey-platform/back/pkg/events"
	"github.com/yohnnn/public-survey-platform/back/services/realtime-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/realtime-service/internal/service"
)

type RealtimeConsumer struct {
	consumer *events.Consumer
	svc      service.RealtimeService
	logger   *log.Logger

	dedupTTL time.Duration
	mu       sync.Mutex
	seen     map[string]time.Time
}

func NewRealtimeConsumer(subscriber events.Subscriber, svc service.RealtimeService, logger *log.Logger, dedupTTL time.Duration) *RealtimeConsumer {
	if dedupTTL <= 0 {
		dedupTTL = 2 * time.Minute
	}

	c := &RealtimeConsumer{
		svc:      svc,
		logger:   logger,
		dedupTTL: dedupTTL,
		seen:     make(map[string]time.Time),
	}

	handlers := map[string]events.HandlerFunc{
		events.TopicVoteCast:    c.handleVoteCast,
		events.TopicVoteRemoved: c.handleVoteRemoved,
	}

	c.consumer = events.NewConsumer(subscriber, handlers)
	return c
}

func (c *RealtimeConsumer) Run(ctx context.Context) error {
	return c.consumer.Run(ctx)
}

func (c *RealtimeConsumer) handleVoteCast(_ context.Context, msg events.Message) error {
	var payload voteCastPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.logger.Printf("failed to unmarshal vote.cast payload: %v", err)
		return err
	}
	if !c.shouldProcess(payload.EventID) {
		return nil
	}

	optionIDs := normalizeOptionIDs(payload.OptionIDs)
	event := models.PollUpdateEvent{
		Event:     events.TopicVoteCast,
		PollID:    strings.TrimSpace(payload.PollID),
		OptionIDs: optionIDs,
		Delta:     int64(len(optionIDs)),
		Timestamp: normalizeTimestamp(payload.VotedAt),
	}

	if event.PollID == "" {
		return nil
	}

	c.svc.PublishPollUpdate(event)
	return nil
}

func (c *RealtimeConsumer) handleVoteRemoved(_ context.Context, msg events.Message) error {
	var payload voteRemovedPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.logger.Printf("failed to unmarshal vote.removed payload: %v", err)
		return err
	}
	if !c.shouldProcess(payload.EventID) {
		return nil
	}

	optionIDs := normalizeOptionIDs(payload.OptionIDs)
	event := models.PollUpdateEvent{
		Event:     events.TopicVoteRemoved,
		PollID:    strings.TrimSpace(payload.PollID),
		OptionIDs: optionIDs,
		Delta:     -int64(len(optionIDs)),
		Timestamp: normalizeTimestamp(payload.RemovedAt),
	}

	if event.PollID == "" {
		return nil
	}

	c.svc.PublishPollUpdate(event)
	return nil
}

func (c *RealtimeConsumer) shouldProcess(eventID string) bool {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return true
	}

	now := time.Now().UTC()

	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := now.Add(-c.dedupTTL)
	for id, seenAt := range c.seen {
		if seenAt.Before(cutoff) {
			delete(c.seen, id)
		}
	}

	if _, ok := c.seen[eventID]; ok {
		return false
	}

	c.seen[eventID] = now
	return true
}

func normalizeOptionIDs(optionIDs []string) []string {
	if len(optionIDs) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(optionIDs))
	out := make([]string, 0, len(optionIDs))
	for _, optionID := range optionIDs {
		optionID = strings.TrimSpace(optionID)
		if optionID == "" {
			continue
		}
		if _, ok := seen[optionID]; ok {
			continue
		}
		seen[optionID] = struct{}{}
		out = append(out, optionID)
	}
	return out
}

func normalizeTimestamp(at time.Time) time.Time {
	if at.IsZero() {
		return time.Now().UTC()
	}
	return at.UTC()
}

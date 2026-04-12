package kafka

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/yohnnn/public-survey-platform/back/pkg/events"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/repository"
)

type FeedConsumer struct {
	consumer *events.Consumer
	repo     repository.FeedRepository
	tx       *tx.Manager
	logger   *log.Logger
}

func NewFeedConsumer(subscriber events.Subscriber, repo repository.FeedRepository, txMgr *tx.Manager, logger *log.Logger) *FeedConsumer {
	c := &FeedConsumer{
		repo:   repo,
		tx:     txMgr,
		logger: logger,
	}

	handlers := map[string]events.HandlerFunc{
		events.TopicPollCreated: c.handlePollCreated,
		events.TopicPollUpdated: c.handlePollUpdated,
		events.TopicPollDeleted: c.handlePollDeleted,
		events.TopicVoteCast:    c.handleVoteCast,
		events.TopicVoteRemoved: c.handleVoteRemoved,
	}

	c.consumer = events.NewConsumer(subscriber, handlers)
	return c
}

func (c *FeedConsumer) Run(ctx context.Context) error {
	return c.consumer.Run(ctx)
}

func (c *FeedConsumer) handlePollCreated(ctx context.Context, msg events.Message) error {
	var payload pollCreatedPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.logger.Printf("failed to unmarshal poll.created payload: %v", err)
		return err
	}

	pollID := strings.TrimSpace(payload.PollID)
	if pollID == "" {
		return nil
	}

	options := make([]models.FeedItemOption, 0, len(payload.Options))
	for _, opt := range payload.Options {
		options = append(options, models.FeedItemOption{
			ID:       opt.ID,
			Text:     opt.Text,
			Position: opt.Position,
		})
	}

	item := models.FeedItem{
		ID:        pollID,
		CreatorID: payload.CreatorID,
		Question:  payload.Question,
		CreatedAt: payload.CreatedAt,
	}

	return c.tx.WithTx(ctx, func(txCtx context.Context) error {
		processed, err := c.repo.MarkEventProcessed(txCtx, payload.EventID, msg.Topic)
		if err != nil {
			return err
		}
		if !processed {
			return nil
		}

		return c.repo.CreateFeedItem(txCtx, item, options, payload.Tags)
	})
}

func (c *FeedConsumer) handlePollUpdated(ctx context.Context, msg events.Message) error {
	var payload pollUpdatedPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.logger.Printf("failed to unmarshal poll.updated payload: %v", err)
		return err
	}

	pollID := strings.TrimSpace(payload.PollID)
	if pollID == "" {
		return nil
	}

	item := models.FeedItem{
		ID:       pollID,
		Question: payload.Question,
	}

	return c.tx.WithTx(ctx, func(txCtx context.Context) error {
		processed, err := c.repo.MarkEventProcessed(txCtx, payload.EventID, msg.Topic)
		if err != nil {
			return err
		}
		if !processed {
			return nil
		}

		return c.repo.UpdateFeedItem(txCtx, item, payload.Tags)
	})
}

func (c *FeedConsumer) handlePollDeleted(ctx context.Context, msg events.Message) error {
	var payload pollDeletedPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.logger.Printf("failed to unmarshal poll.deleted payload: %v", err)
		return err
	}

	pollID := strings.TrimSpace(payload.PollID)
	if pollID == "" {
		return nil
	}

	return c.tx.WithTx(ctx, func(txCtx context.Context) error {
		processed, err := c.repo.MarkEventProcessed(txCtx, payload.EventID, msg.Topic)
		if err != nil {
			return err
		}
		if !processed {
			return nil
		}

		return c.repo.DeleteFeedItem(txCtx, pollID)
	})
}

func (c *FeedConsumer) handleVoteCast(ctx context.Context, msg events.Message) error {
	var payload voteCastPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.logger.Printf("failed to unmarshal vote.cast payload: %v", err)
		return err
	}

	pollID := strings.TrimSpace(payload.PollID)
	if pollID == "" {
		return nil
	}

	optionIDs := normalizeOptionIDs(payload.OptionIDs)
	delta := int64(len(optionIDs))

	return c.tx.WithTx(ctx, func(txCtx context.Context) error {
		processed, err := c.repo.MarkEventProcessed(txCtx, payload.EventID, msg.Topic)
		if err != nil {
			return err
		}
		if !processed {
			return nil
		}

		for _, optionID := range optionIDs {
			if err := c.repo.IncrementOptionVotes(txCtx, optionID, 1); err != nil {
				return err
			}
		}
		return c.repo.UpdateTotalVotes(txCtx, pollID, delta)
	})
}

func (c *FeedConsumer) handleVoteRemoved(ctx context.Context, msg events.Message) error {
	var payload voteRemovedPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.logger.Printf("failed to unmarshal vote.removed payload: %v", err)
		return err
	}

	pollID := strings.TrimSpace(payload.PollID)
	if pollID == "" {
		return nil
	}

	optionIDs := normalizeOptionIDs(payload.OptionIDs)
	delta := int64(len(optionIDs))

	return c.tx.WithTx(ctx, func(txCtx context.Context) error {
		processed, err := c.repo.MarkEventProcessed(txCtx, payload.EventID, msg.Topic)
		if err != nil {
			return err
		}
		if !processed {
			return nil
		}

		for _, optionID := range optionIDs {
			if err := c.repo.IncrementOptionVotes(txCtx, optionID, -1); err != nil {
				return err
			}
		}
		return c.repo.UpdateTotalVotes(txCtx, pollID, -delta)
	})
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

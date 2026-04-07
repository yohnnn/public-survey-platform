package service

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/yohnnn/public-survey-platform/back/api/events"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/feed-service/internal/repository"
)

type pollCreatedPayload struct {
	PollID    string              `json:"poll_id"`
	CreatorID string              `json:"creator_id"`
	Question  string              `json:"question"`
	Options   []pollCreatedOption `json:"options"`
	Tags      []string            `json:"tags"`
	CreatedAt time.Time           `json:"created_at"`
}

type pollCreatedOption struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Position int32  `json:"position"`
}

type voteCastPayload struct {
	PollID    string   `json:"poll_id"`
	OptionIDs []string `json:"option_ids"`
}

type voteRemovedPayload struct {
	PollID    string   `json:"poll_id"`
	OptionIDs []string `json:"option_ids"`
}

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

	options := make([]models.FeedItemOption, 0, len(payload.Options))
	for _, opt := range payload.Options {
		options = append(options, models.FeedItemOption{
			ID:       opt.ID,
			Text:     opt.Text,
			Position: opt.Position,
		})
	}

	item := models.FeedItem{
		ID:        payload.PollID,
		CreatorID: payload.CreatorID,
		Question:  payload.Question,
		CreatedAt: payload.CreatedAt,
	}

	return c.tx.WithTx(ctx, func(txCtx context.Context) error {
		return c.repo.CreateFeedItem(txCtx, item, options, payload.Tags)
	})
}

func (c *FeedConsumer) handleVoteCast(ctx context.Context, msg events.Message) error {
	var payload voteCastPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.logger.Printf("failed to unmarshal vote.cast payload: %v", err)
		return err
	}

	delta := int64(len(payload.OptionIDs))

	return c.tx.WithTx(ctx, func(txCtx context.Context) error {
		for _, optionID := range payload.OptionIDs {
			if err := c.repo.IncrementOptionVotes(txCtx, optionID, 1); err != nil {
				return err
			}
		}
		return c.repo.UpdateTotalVotes(txCtx, payload.PollID, delta)
	})
}

func (c *FeedConsumer) handleVoteRemoved(ctx context.Context, msg events.Message) error {
	var payload voteRemovedPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.logger.Printf("failed to unmarshal vote.removed payload: %v", err)
		return err
	}

	delta := int64(len(payload.OptionIDs))

	return c.tx.WithTx(ctx, func(txCtx context.Context) error {
		for _, optionID := range payload.OptionIDs {
			if err := c.repo.IncrementOptionVotes(txCtx, optionID, -1); err != nil {
				return err
			}
		}
		return c.repo.UpdateTotalVotes(txCtx, payload.PollID, -delta)
	})
}

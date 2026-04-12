package kafka

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/yohnnn/public-survey-platform/back/pkg/events"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/poll-service/internal/repository"
)

type PollConsumer struct {
	consumer *events.Consumer
	repo     repository.PollRepository
	tx       *tx.Manager
	logger   *log.Logger
}

func NewPollConsumer(subscriber events.Subscriber, repo repository.PollRepository, txMgr *tx.Manager, logger *log.Logger) *PollConsumer {
	c := &PollConsumer{
		repo:   repo,
		tx:     txMgr,
		logger: logger,
	}

	handlers := map[string]events.HandlerFunc{
		events.TopicVoteCast:    c.handleVoteCast,
		events.TopicVoteRemoved: c.handleVoteRemoved,
	}

	c.consumer = events.NewConsumer(subscriber, handlers)
	return c
}

func (c *PollConsumer) Run(ctx context.Context) error {
	return c.consumer.Run(ctx)
}

func (c *PollConsumer) handleVoteCast(ctx context.Context, msg events.Message) error {
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
			if err := c.repo.IncrementOptionVotes(txCtx, pollID, optionID, 1); err != nil {
				return err
			}
		}
		return c.repo.UpdateTotalVotes(txCtx, pollID, delta)
	})
}

func (c *PollConsumer) handleVoteRemoved(ctx context.Context, msg events.Message) error {
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
			if err := c.repo.IncrementOptionVotes(txCtx, pollID, optionID, -1); err != nil {
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

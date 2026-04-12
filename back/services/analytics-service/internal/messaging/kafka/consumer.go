package kafka

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/yohnnn/public-survey-platform/back/pkg/events"
	"github.com/yohnnn/public-survey-platform/back/pkg/tx"
	"github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/repository"
)

type AnalyticsConsumer struct {
	consumer *events.Consumer
	repo     repository.AnalyticsRepository
	tx       *tx.Manager
	logger   *log.Logger
}

func NewAnalyticsConsumer(subscriber events.Subscriber, repo repository.AnalyticsRepository, txMgr *tx.Manager, logger *log.Logger) *AnalyticsConsumer {
	c := &AnalyticsConsumer{
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

func (c *AnalyticsConsumer) Run(ctx context.Context) error {
	return c.consumer.Run(ctx)
}

func (c *AnalyticsConsumer) handleVoteCast(ctx context.Context, msg events.Message) error {
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

		if country := strings.TrimSpace(payload.Country); country != "" {
			if err := c.repo.IncrementCountryVotes(txCtx, pollID, country, 1); err != nil {
				return err
			}
		}
		if gender := strings.TrimSpace(payload.Gender); gender != "" {
			if err := c.repo.IncrementGenderVotes(txCtx, pollID, gender, 1); err != nil {
				return err
			}
		}
		if payload.BirthYear != nil {
			if ageRange := detectAgeRange(*payload.BirthYear, payload.VotedAt); ageRange != "" {
				if err := c.repo.IncrementAgeVotes(txCtx, pollID, ageRange, 1); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (c *AnalyticsConsumer) handleVoteRemoved(ctx context.Context, msg events.Message) error {
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

		if country := strings.TrimSpace(payload.Country); country != "" {
			if err := c.repo.IncrementCountryVotes(txCtx, pollID, country, -1); err != nil {
				return err
			}
		}
		if gender := strings.TrimSpace(payload.Gender); gender != "" {
			if err := c.repo.IncrementGenderVotes(txCtx, pollID, gender, -1); err != nil {
				return err
			}
		}
		if payload.BirthYear != nil {
			if ageRange := detectAgeRange(*payload.BirthYear, payload.RemovedAt); ageRange != "" {
				if err := c.repo.IncrementAgeVotes(txCtx, pollID, ageRange, -1); err != nil {
					return err
				}
			}
		}

		return nil
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

func detectAgeRange(birthYear int32, at time.Time) string {
	if birthYear <= 0 {
		return ""
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}

	age := at.Year() - int(birthYear)
	if age <= 0 {
		return ""
	}

	switch {
	case age <= 24:
		return "18-24"
	case age <= 34:
		return "25-34"
	case age <= 44:
		return "35-44"
	default:
		return "45+"
	}
}

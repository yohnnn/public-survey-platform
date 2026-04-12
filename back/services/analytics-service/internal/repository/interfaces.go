package repository

import (
	"context"

	"github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/models"
)

type AnalyticsRepository interface {
	IncrementOptionVotes(ctx context.Context, pollID, optionID string, delta int64) error
	IncrementCountryVotes(ctx context.Context, pollID, country string, delta int64) error
	IncrementGenderVotes(ctx context.Context, pollID, gender string, delta int64) error
	IncrementAgeVotes(ctx context.Context, pollID, ageRange string, delta int64) error
	MarkEventProcessed(ctx context.Context, eventID, topic string) (bool, error)
	GetPollAnalytics(ctx context.Context, pollID string) (models.PollAnalytics, error)
	GetCountryStats(ctx context.Context, pollID string) ([]models.CountryStat, error)
	GetGenderStats(ctx context.Context, pollID string) ([]models.GenderStat, error)
	GetAgeStats(ctx context.Context, pollID string) ([]models.AgeStat, error)
}

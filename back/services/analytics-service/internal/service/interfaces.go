package service

import (
	"context"

	"github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/models"
)

type AnalyticsService interface {
	GetPollAnalytics(ctx context.Context, pollID string) (models.PollAnalytics, error)
	GetCountryStats(ctx context.Context, pollID string) ([]models.CountryStat, error)
	GetGenderStats(ctx context.Context, pollID string) ([]models.GenderStat, error)
	GetAgeStats(ctx context.Context, pollID string) ([]models.AgeStat, error)
}

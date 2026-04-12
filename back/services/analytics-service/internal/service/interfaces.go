package service

import (
	"context"
	"time"

	"github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/models"
)

type AnalyticsService interface {
	GetPollAnalytics(ctx context.Context, pollID string, from, to *time.Time, interval string) (models.PollAnalytics, error)
	GetCountryStats(ctx context.Context, pollID string, from, to *time.Time, interval string) ([]models.CountryStat, error)
	GetGenderStats(ctx context.Context, pollID string, from, to *time.Time, interval string) ([]models.GenderStat, error)
	GetAgeStats(ctx context.Context, pollID string, from, to *time.Time, interval string) ([]models.AgeStat, error)
}

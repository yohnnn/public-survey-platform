package service

import (
	"context"
	"strings"

	"github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/models"
	"github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/repository"
)

type analyticsService struct {
	repo repository.AnalyticsRepository
}

func NewAnalyticsService(repo repository.AnalyticsRepository) AnalyticsService {
	return &analyticsService{repo: repo}
}

func (s *analyticsService) GetPollAnalytics(ctx context.Context, pollID string) (models.PollAnalytics, error) {
	pollID = strings.TrimSpace(pollID)
	if pollID == "" {
		return models.PollAnalytics{}, models.ErrInvalidArgument
	}
	return s.repo.GetPollAnalytics(ctx, pollID)
}

func (s *analyticsService) GetCountryStats(ctx context.Context, pollID string) ([]models.CountryStat, error) {
	pollID = strings.TrimSpace(pollID)
	if pollID == "" {
		return nil, models.ErrInvalidArgument
	}
	return s.repo.GetCountryStats(ctx, pollID)
}

func (s *analyticsService) GetGenderStats(ctx context.Context, pollID string) ([]models.GenderStat, error) {
	pollID = strings.TrimSpace(pollID)
	if pollID == "" {
		return nil, models.ErrInvalidArgument
	}
	return s.repo.GetGenderStats(ctx, pollID)
}

func (s *analyticsService) GetAgeStats(ctx context.Context, pollID string) ([]models.AgeStat, error) {
	pollID = strings.TrimSpace(pollID)
	if pollID == "" {
		return nil, models.ErrInvalidArgument
	}
	return s.repo.GetAgeStats(ctx, pollID)
}

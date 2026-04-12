package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/models"
	mockrepo "github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/service/mock"
	"go.uber.org/mock/gomock"
)

func TestGetPollAnalyticsRejectsEmptyPollID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := NewAnalyticsService(mockrepo.NewMockAnalyticsRepository(ctrl))

	_, err := svc.GetPollAnalytics(context.Background(), "", nil, nil, "")
	if !errors.Is(err, models.ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}
}

func TestGetPollAnalyticsPassesThroughRepositoryResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expected := models.PollAnalytics{PollID: "poll-1", TotalVotes: 42}
	repo := mockrepo.NewMockAnalyticsRepository(ctrl)
	repo.EXPECT().GetPollAnalytics(gomock.Any(), "poll-1").Return(expected, nil)

	svc := NewAnalyticsService(repo)
	got, err := svc.GetPollAnalytics(context.Background(), "  poll-1  ", nil, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected analytics payload: got=%#v expected=%#v", got, expected)
	}
}

func TestGetCountryStatsRejectsEmptyPollID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := NewAnalyticsService(mockrepo.NewMockAnalyticsRepository(ctrl))

	_, err := svc.GetCountryStats(context.Background(), " ", nil, nil, "")
	if !errors.Is(err, models.ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}
}

func TestGetAgeStatsPassesThroughRepositoryResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expected := []models.AgeStat{{AgeRange: "18-24", Votes: 10}}
	repo := mockrepo.NewMockAnalyticsRepository(ctrl)
	repo.EXPECT().GetAgeStats(gomock.Any(), "poll-1").Return(expected, nil)

	svc := NewAnalyticsService(repo)
	got, err := svc.GetAgeStats(context.Background(), "poll-1", nil, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected age stats: got=%#v expected=%#v", got, expected)
	}
}

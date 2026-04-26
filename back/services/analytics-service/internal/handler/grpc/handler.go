package grpc

import (
	"context"

	analyticsv1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/analytics/v1"
	"github.com/yohnnn/public-survey-platform/back/services/analytics-service/internal/service"
)

type Handler struct {
	svc service.AnalyticsService
	analyticsv1.UnimplementedAnalyticsServiceServer
}

func NewHandler(svc service.AnalyticsService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetPollAnalytics(ctx context.Context, req *analyticsv1.GetPollAnalyticsRequest) (*analyticsv1.GetPollAnalyticsResponse, error) {
	res, err := h.svc.GetPollAnalytics(ctx, req.GetPollId())
	if err != nil {
		return nil, toStatusError(err)
	}

	return mapPollAnalytics(res), nil
}

func (h *Handler) GetCountryStats(ctx context.Context, req *analyticsv1.GetCountryStatsRequest) (*analyticsv1.GetCountryStatsResponse, error) {
	items, err := h.svc.GetCountryStats(ctx, req.GetPollId())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &analyticsv1.GetCountryStatsResponse{
		PollId: req.GetPollId(),
		Items:  mapCountryStats(items),
	}, nil
}

func (h *Handler) GetGenderStats(ctx context.Context, req *analyticsv1.GetGenderStatsRequest) (*analyticsv1.GetGenderStatsResponse, error) {
	items, err := h.svc.GetGenderStats(ctx, req.GetPollId())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &analyticsv1.GetGenderStatsResponse{
		PollId: req.GetPollId(),
		Items:  mapGenderStats(items),
	}, nil
}

func (h *Handler) GetAgeStats(ctx context.Context, req *analyticsv1.GetAgeStatsRequest) (*analyticsv1.GetAgeStatsResponse, error) {
	items, err := h.svc.GetAgeStats(ctx, req.GetPollId())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &analyticsv1.GetAgeStatsResponse{
		PollId: req.GetPollId(),
		Items:  mapAgeStats(items),
	}, nil
}

package grpc

import (
	"context"

	realtimev1 "github.com/yohnnn/public-survey-platform/back/api/gen/go/realtime/v1"
	"github.com/yohnnn/public-survey-platform/back/services/realtime-service/internal/service"
)

type Handler struct {
	svc service.RealtimeService
	realtimev1.UnimplementedRealtimeServiceServer
}

func NewHandler(svc service.RealtimeService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) StreamPollUpdates(req *realtimev1.StreamPollUpdatesRequest, stream realtimev1.RealtimeService_StreamPollUpdatesServer) error {
	updates, unsubscribe, err := h.svc.SubscribePollUpdates(stream.Context(), req.GetPollId())
	if err != nil {
		return toStatusError(err)
	}
	defer unsubscribe()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case update, ok := <-updates:
			if !ok {
				return nil
			}

			if len(update.OptionIDs) == 0 {
				if err := stream.Send(mapPollUpdateEvent(update, "")); err != nil {
					return err
				}
				continue
			}

			for _, optionID := range update.OptionIDs {
				if err := stream.Send(mapPollUpdateEvent(update, optionID)); err != nil {
					return err
				}
			}
		}
	}
}

func (h *Handler) WSHandshake(ctx context.Context, _ *realtimev1.WSHandshakeRequest) (*realtimev1.WSHandshakeResponse, error) {
	connectionID, err := h.svc.WSHandshake(ctx)
	if err != nil {
		return nil, toStatusError(err)
	}
	return &realtimev1.WSHandshakeResponse{ConnectionId: connectionID}, nil
}

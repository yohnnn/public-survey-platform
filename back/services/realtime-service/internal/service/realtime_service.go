package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/yohnnn/public-survey-platform/back/services/realtime-service/internal/hub"
	"github.com/yohnnn/public-survey-platform/back/services/realtime-service/internal/models"
)

type realtimeService struct {
	hub          *hub.Hub
	streamBuffer int
}

func NewRealtimeService(h *hub.Hub, streamBuffer int) RealtimeService {
	if streamBuffer <= 0 {
		streamBuffer = 256
	}
	return &realtimeService{hub: h, streamBuffer: streamBuffer}
}

func (s *realtimeService) SubscribePollUpdates(_ context.Context, pollID string) (<-chan models.PollUpdateEvent, func(), error) {
	pollID = strings.TrimSpace(pollID)
	if pollID == "" {
		return nil, nil, models.ErrInvalidArgument
	}

	_, ch, unsubscribe, err := s.hub.Subscribe(pollID, s.streamBuffer)
	if err != nil {
		return nil, nil, err
	}

	return ch, unsubscribe, nil
}

func (s *realtimeService) PublishPollUpdate(event models.PollUpdateEvent) {
	event.PollID = strings.TrimSpace(event.PollID)
	if event.PollID == "" {
		return
	}
	s.hub.Publish(event)
}

func (s *realtimeService) WSHandshake(context.Context) (string, error) {
	return newConnectionID(), nil
}

func newConnectionID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

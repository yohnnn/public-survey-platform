package service

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/yohnnn/public-survey-platform/back/services/realtime-service/internal/hub"
	"github.com/yohnnn/public-survey-platform/back/services/realtime-service/internal/models"
)

func TestSubscribePollUpdatesRejectsEmptyPollID(t *testing.T) {
	svc := NewRealtimeService(hub.New(), 8)

	_, _, err := svc.SubscribePollUpdates(context.Background(), " ")
	if !errors.Is(err, models.ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}
}

func TestPublishPollUpdateDeliversToSubscribers(t *testing.T) {
	svc := NewRealtimeService(hub.New(), 8)
	updates, unsubscribe, err := svc.SubscribePollUpdates(context.Background(), "poll-1")
	if err != nil {
		t.Fatalf("unexpected subscribe error: %v", err)
	}
	defer unsubscribe()

	expected := models.PollUpdateEvent{Event: "vote.cast", PollID: "poll-1", Delta: 1}
	svc.PublishPollUpdate(expected)

	select {
	case got := <-updates:
		if got.PollID != expected.PollID || got.Event != expected.Event || got.Delta != expected.Delta {
			t.Fatalf("unexpected event payload: %#v", got)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("did not receive poll update")
	}
}

func TestPublishPollUpdateIgnoresEmptyPollID(t *testing.T) {
	svc := NewRealtimeService(hub.New(), 8)
	updates, unsubscribe, err := svc.SubscribePollUpdates(context.Background(), "poll-1")
	if err != nil {
		t.Fatalf("unexpected subscribe error: %v", err)
	}
	defer unsubscribe()

	svc.PublishPollUpdate(models.PollUpdateEvent{PollID: "   "})

	select {
	case got := <-updates:
		t.Fatalf("did not expect any update, got %#v", got)
	case <-time.After(30 * time.Millisecond):
	}
}

func TestWSHandshakeReturnsHexConnectionID(t *testing.T) {
	svc := NewRealtimeService(hub.New(), 8)

	id, err := svc.WSHandshake(context.Background())
	if err != nil {
		t.Fatalf("unexpected handshake error: %v", err)
	}
	if len(id) != 32 {
		t.Fatalf("expected 32-char hex id, got %q (%d)", id, len(id))
	}
	if _, err := hex.DecodeString(id); err != nil {
		t.Fatalf("connection id is not valid hex: %v", err)
	}
}

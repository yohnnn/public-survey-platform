package hub

import (
	"fmt"
	"strings"
	"sync"

	"github.com/yohnnn/public-survey-platform/back/services/realtime-service/internal/models"
)

type Hub struct {
	mu          sync.RWMutex
	subscribers map[string]map[string]chan models.PollUpdateEvent
	nextID      uint64
}

func New() *Hub {
	return &Hub{
		subscribers: make(map[string]map[string]chan models.PollUpdateEvent),
	}
}

func (h *Hub) Subscribe(pollID string, buffer int) (string, <-chan models.PollUpdateEvent, func(), error) {
	if h == nil {
		return "", nil, nil, fmt.Errorf("hub is nil")
	}
	pollID = strings.TrimSpace(pollID)
	if pollID == "" {
		return "", nil, nil, fmt.Errorf("poll id is empty")
	}
	if buffer <= 0 {
		buffer = 256
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	subscriberID := fmt.Sprintf("sub-%d", h.nextID)

	if _, ok := h.subscribers[pollID]; !ok {
		h.subscribers[pollID] = make(map[string]chan models.PollUpdateEvent)
	}
	ch := make(chan models.PollUpdateEvent, buffer)
	h.subscribers[pollID][subscriberID] = ch

	unsubscribe := func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		pollSubs, ok := h.subscribers[pollID]
		if !ok {
			return
		}
		stream, ok := pollSubs[subscriberID]
		if !ok {
			return
		}
		delete(pollSubs, subscriberID)
		close(stream)
		if len(pollSubs) == 0 {
			delete(h.subscribers, pollID)
		}
	}

	return subscriberID, ch, unsubscribe, nil
}

func (h *Hub) Publish(event models.PollUpdateEvent) {
	if h == nil {
		return
	}
	pollID := strings.TrimSpace(event.PollID)
	if pollID == "" {
		return
	}

	h.mu.RLock()
	pollSubs := h.subscribers[pollID]
	channels := make([]chan models.PollUpdateEvent, 0, len(pollSubs))
	for _, ch := range pollSubs {
		channels = append(channels, ch)
	}
	h.mu.RUnlock()

	for _, ch := range channels {
		select {
		case ch <- event:
		default:
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- event:
			default:
			}
		}
	}
}

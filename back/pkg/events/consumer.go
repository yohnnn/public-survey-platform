package events

import (
	"context"
	"fmt"
	"strings"
)

type HandlerFunc func(ctx context.Context, msg Message) error

type Consumer struct {
	subscriber Subscriber
	handlers   map[string]HandlerFunc
}

func NewConsumer(subscriber Subscriber, handlers map[string]HandlerFunc) *Consumer {
	copyHandlers := make(map[string]HandlerFunc, len(handlers))
	for topic, handler := range handlers {
		topic = strings.TrimSpace(topic)
		if topic == "" || handler == nil {
			continue
		}
		copyHandlers[topic] = handler
	}

	return &Consumer{
		subscriber: subscriber,
		handlers:   copyHandlers,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("consumer is nil")
	}
	if c.subscriber == nil {
		return fmt.Errorf("subscriber is nil")
	}
	if len(c.handlers) == 0 {
		return fmt.Errorf("handlers are empty")
	}

	topics := make([]string, 0, len(c.handlers))
	for topic := range c.handlers {
		topics = append(topics, topic)
	}

	return c.subscriber.Subscribe(ctx, topics, func(msgCtx context.Context, msg Message) error {
		handler, ok := c.resolveHandler(msg.Topic)
		if !ok {
			return fmt.Errorf("no handler for topic %q", msg.Topic)
		}
		return handler(msgCtx, msg)
	})
}

func (c *Consumer) resolveHandler(topic string) (HandlerFunc, bool) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return nil, false
	}

	if handler, ok := c.handlers[topic]; ok {
		return handler, true
	}

	for expectedTopic, handler := range c.handlers {
		expectedTopic = strings.TrimSpace(expectedTopic)
		if expectedTopic == "" {
			continue
		}
		if strings.HasSuffix(topic, "."+expectedTopic) {
			return handler, true
		}
	}

	return nil, false
}

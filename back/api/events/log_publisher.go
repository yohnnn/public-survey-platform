package events

import (
	"context"
	"fmt"
	"log"
)

type LogPublisher struct {
	logger *log.Logger
}

func NewLogPublisher(logger *log.Logger) *LogPublisher {
	if logger == nil {
		logger = log.Default()
	}
	return &LogPublisher{logger: logger}
}

func (p *LogPublisher) Publish(_ context.Context, msg Message) error {
	if p == nil || p.logger == nil {
		return fmt.Errorf("log publisher is not initialized")
	}

	p.logger.Printf("event published topic=%s key=%s payload=%s", msg.Topic, msg.Key, string(msg.Payload))
	return nil
}

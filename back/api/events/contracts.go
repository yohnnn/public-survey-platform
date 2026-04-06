package events

import "context"

type Message struct {
	Topic   string
	Key     string
	Payload []byte
}

type Publisher interface {
	Publish(ctx context.Context, msg Message) error
}

type Subscriber interface {
	Subscribe(ctx context.Context, topics []string, handler func(context.Context, Message) error) error
}

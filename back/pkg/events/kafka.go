package events

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaPublisherConfig struct {
	Brokers      []string
	TopicPrefix  string
	WriteTimeout time.Duration
}

type KafkaPublisher struct {
	writer       *kafka.Writer
	topicPrefix  string
	writeTimeout time.Duration
}

func NewKafkaPublisher(cfg KafkaPublisherConfig) (*KafkaPublisher, error) {
	brokers := normalizeBrokers(cfg.Brokers)
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}

	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = 5 * time.Second
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireOne,
		WriteTimeout: cfg.WriteTimeout,
		BatchTimeout: 10 * time.Millisecond,
		Async:        false,
	}

	return &KafkaPublisher{
		writer:       writer,
		topicPrefix:  normalizePrefix(cfg.TopicPrefix),
		writeTimeout: cfg.WriteTimeout,
	}, nil
}

func (p *KafkaPublisher) Publish(ctx context.Context, msg Message) error {
	if p == nil || p.writer == nil {
		return fmt.Errorf("kafka publisher is not initialized")
	}

	topic := qualifyTopic(p.topicPrefix, msg.Topic)
	if strings.TrimSpace(topic) == "" {
		return fmt.Errorf("topic is empty")
	}

	writeCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		writeCtx, cancel = context.WithTimeout(ctx, p.writeTimeout)
		defer cancel()
	}

	return p.writer.WriteMessages(writeCtx, kafka.Message{
		Topic: topic,
		Key:   []byte(strings.TrimSpace(msg.Key)),
		Value: msg.Payload,
	})
}

func (p *KafkaPublisher) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

type KafkaSubscriberConfig struct {
	Brokers         []string
	GroupID         string
	TopicPrefix     string
	ReadTimeout     time.Duration
	CommitPeriod    time.Duration
	ShutdownTimeout time.Duration
}

type KafkaSubscriber struct {
	brokers         []string
	groupID         string
	topicPrefix     string
	readTimeout     time.Duration
	commitPeriod    time.Duration
	shutdownTimeout time.Duration
}

func NewKafkaSubscriber(cfg KafkaSubscriberConfig) (*KafkaSubscriber, error) {
	brokers := normalizeBrokers(cfg.Brokers)
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}
	if strings.TrimSpace(cfg.GroupID) == "" {
		return nil, fmt.Errorf("kafka group id is required")
	}

	if cfg.ReadTimeout <= 0 {
		cfg.ReadTimeout = 10 * time.Second
	}
	if cfg.CommitPeriod <= 0 {
		cfg.CommitPeriod = time.Second
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = 10 * time.Second
	}

	return &KafkaSubscriber{
		brokers:         brokers,
		groupID:         strings.TrimSpace(cfg.GroupID),
		topicPrefix:     normalizePrefix(cfg.TopicPrefix),
		readTimeout:     cfg.ReadTimeout,
		commitPeriod:    cfg.CommitPeriod,
		shutdownTimeout: cfg.ShutdownTimeout,
	}, nil
}

func (s *KafkaSubscriber) Subscribe(ctx context.Context, topics []string, handler func(context.Context, Message) error) error {
	if s == nil {
		return fmt.Errorf("kafka subscriber is nil")
	}
	if handler == nil {
		return fmt.Errorf("handler is nil")
	}

	groupTopics := make([]string, 0, len(topics))
	for _, topic := range topics {
		qualified := qualifyTopic(s.topicPrefix, topic)
		if qualified == "" {
			continue
		}
		groupTopics = append(groupTopics, qualified)
	}
	if len(groupTopics) == 0 {
		return fmt.Errorf("topics are empty")
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        s.brokers,
		GroupID:        s.groupID,
		GroupTopics:    groupTopics,
		MaxWait:        s.readTimeout,
		CommitInterval: s.commitPeriod,
	})
	defer reader.Close()

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}

		handleCtx := ctx
		handleCancel := func() {}
		if ctx.Err() != nil {
			handleCtx, handleCancel = context.WithTimeout(context.Background(), s.shutdownTimeout)
		}

		handleErr := handler(handleCtx, Message{
			Topic:   msg.Topic,
			Key:     string(msg.Key),
			Payload: msg.Value,
		})
		handleCancel()
		if handleErr != nil {
			return handleErr
		}

		commitCtx := ctx
		commitCancel := func() {}
		if ctx.Err() != nil {
			commitCtx, commitCancel = context.WithTimeout(context.Background(), s.shutdownTimeout)
		}

		if err := reader.CommitMessages(commitCtx, msg); err != nil {
			commitCancel()
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		commitCancel()

		if ctx.Err() != nil {
			return nil
		}
	}
}

func normalizeBrokers(brokers []string) []string {
	out := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		v := strings.TrimSpace(broker)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

func normalizePrefix(prefix string) string {
	v := strings.TrimSpace(prefix)
	v = strings.TrimSuffix(v, ".")
	return v
}

func qualifyTopic(prefix, topic string) string {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return ""
	}
	if prefix == "" {
		return topic
	}
	return prefix + "." + topic
}

package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kgo"
)

// HandlerFunc is the callback for processing consumed events.
type HandlerFunc func(ctx context.Context, topic string, key string, value []byte) error

// Consumer subscribes to Redpanda/Kafka topics and dispatches events.
type Consumer struct {
	client   *kgo.Client
	handlers map[string]HandlerFunc
	logger   *slog.Logger
}

// NewConsumer creates a Kafka consumer for the given topics and group.
func NewConsumer(brokers []string, groupID string, topics []string, logger *slog.Logger) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(topics...),
		kgo.AllowAutoTopicCreation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	return &Consumer{
		client:   client,
		handlers: make(map[string]HandlerFunc),
		logger:   logger,
	}, nil
}

// RegisterHandler sets a handler for a specific topic.
func (c *Consumer) RegisterHandler(topic string, handler HandlerFunc) {
	c.handlers[topic] = handler
}

// Start begins consuming messages. Blocks until ctx is cancelled.
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("consumer started",
		slog.Int("handler_count", len(c.handlers)),
	)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("consumer stopping")
			return ctx.Err()
		default:
		}

		fetches := c.client.PollFetches(ctx)
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, err := range errs {
				c.logger.Error("fetch error",
					slog.String("topic", err.Topic),
					slog.String("error", err.Err.Error()),
				)
			}
			continue
		}

		fetches.EachRecord(func(record *kgo.Record) {
			handler, ok := c.handlers[record.Topic]
			if !ok {
				c.logger.Warn("no handler for topic", slog.String("topic", record.Topic))
				return
			}

			if err := handler(ctx, record.Topic, string(record.Key), record.Value); err != nil {
				c.logger.Error("handler error",
					slog.String("topic", record.Topic),
					slog.String("key", string(record.Key)),
					slog.String("error", err.Error()),
				)
			}
		})
	}
}

// Close shuts down the consumer.
func (c *Consumer) Close() {
	c.client.Close()
}

// ParseEvent is a helper to unmarshal event JSON into a typed struct.
func ParseEvent[T any](data []byte) (T, error) {
	var event T
	err := json.Unmarshal(data, &event)
	return event, err
}

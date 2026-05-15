package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// HandlerFunc is the callback for processing consumed events.
type HandlerFunc func(ctx context.Context, topic string, key string, value []byte) error

// Consumer subscribes to Redpanda/Kafka topics and dispatches events.
type Consumer struct {
	client   *kgo.Client
	handlers map[string]HandlerFunc
	logger   *slog.Logger
	options  ConsumerOptions
}

// ConsumerOptions controls event dispatch behavior.
type ConsumerOptions struct {
	// PartitionConcurrency controls how many topic partitions can be processed at once.
	// Records within a partition are still processed sequentially, preserving Kafka ordering by key.
	PartitionConcurrency int
	FetchErrorBackoff    time.Duration
}

// NewConsumer creates a Kafka consumer for the given topics and group.
func NewConsumer(brokers []string, groupID string, topics []string, logger *slog.Logger) (*Consumer, error) {
	return NewConsumerWithOptions(brokers, groupID, topics, logger, ConsumerOptions{})
}

// NewConsumerWithOptions creates a Kafka consumer with explicit dispatch options.
func NewConsumerWithOptions(brokers []string, groupID string, topics []string, logger *slog.Logger, options ConsumerOptions) (*Consumer, error) {
	options = normalizeConsumerOptions(options)
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(topics...),
		kgo.AllowAutoTopicCreation(),
		kgo.AutoCommitMarks(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	return &Consumer{
		client:   client,
		handlers: make(map[string]HandlerFunc),
		logger:   logger,
		options:  options,
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
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.options.FetchErrorBackoff):
			}
			continue
		}

		c.dispatchFetches(ctx, fetches)
	}
}

func (c *Consumer) dispatchFetches(ctx context.Context, fetches kgo.Fetches) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, c.options.PartitionConcurrency)

	fetches.EachPartition(func(partition kgo.FetchTopicPartition) {
		if len(partition.Records) == 0 {
			return
		}

		sem <- struct{}{}
		wg.Add(1)
		go func(partition kgo.FetchTopicPartition) {
			defer wg.Done()
			defer func() { <-sem }()

			for _, record := range partition.Records {
				handler, ok := c.handlers[record.Topic]
				if !ok {
					c.logger.Warn("no handler for topic", slog.String("topic", record.Topic))
					continue
				}

				if err := handler(ctx, record.Topic, string(record.Key), record.Value); err != nil {
					c.logger.Error("handler error",
						slog.String("topic", record.Topic),
						slog.String("key", string(record.Key)),
						slog.String("error", err.Error()),
					)
					continue
				}

				c.client.MarkCommitRecords(record)
			}
		}(partition)
	})

	wg.Wait()
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

// RegisterJSONHandler adapts a typed JSON event handler to a raw Kafka handler.
func RegisterJSONHandler[T any](c *Consumer, topic string, handler func(context.Context, string, T) error) {
	c.RegisterHandler(topic, func(ctx context.Context, _ string, key string, value []byte) error {
		event, err := ParseEvent[T](value)
		if err != nil {
			return err
		}
		return handler(ctx, key, event)
	})
}

func normalizeConsumerOptions(options ConsumerOptions) ConsumerOptions {
	if options.PartitionConcurrency <= 0 {
		options.PartitionConcurrency = 4
	}
	if options.FetchErrorBackoff <= 0 {
		options.FetchErrorBackoff = 250 * time.Millisecond
	}
	return options
}

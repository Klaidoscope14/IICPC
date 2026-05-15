package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// RedisPublisher publishes lightweight event notifications via Redis Pub/Sub.
// This complements the durable Redpanda producer — Redis is used for real-time
// frontend notifications, while Redpanda handles the reliable async pipeline.
type RedisPublisher struct {
	client *redis.Client
	logger *slog.Logger
}

// NewRedisPublisher creates a Redis Pub/Sub publisher.
func NewRedisPublisher(addr string, password string, db int, logger *slog.Logger) (*RedisPublisher, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		PoolSize: 10,
	})

	// Verify connectivity.
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisPublisher{
		client: client,
		logger: logger,
	}, nil
}

// Redis channel names.
const (
	ChannelSubmissionCreated = "submission:created"
	ChannelSubmissionDeleted = "submission:deleted"
	ChannelSubmissionStatus  = "submission:status"
)

// SubmissionEvent is the lightweight Redis notification payload.
type SubmissionEvent struct {
	SubmissionID string `json:"submission_id"`
	ContestantID string `json:"contestant_id"`
	TeamName     string `json:"team_name"`
	Status       string `json:"status"`
	Version      int    `json:"version"`
}

// PublishSubmissionCreated publishes a submission creation notification via Redis Pub/Sub.
func (p *RedisPublisher) PublishSubmissionCreated(ctx context.Context, event SubmissionEvent) error {
	return p.publish(ctx, ChannelSubmissionCreated, event)
}

// PublishSubmissionDeleted publishes a submission deletion notification via Redis Pub/Sub.
func (p *RedisPublisher) PublishSubmissionDeleted(ctx context.Context, event SubmissionEvent) error {
	return p.publish(ctx, ChannelSubmissionDeleted, event)
}

// PublishSubmissionStatus publishes a submission status change notification via Redis Pub/Sub.
func (p *RedisPublisher) PublishSubmissionStatus(ctx context.Context, event SubmissionEvent) error {
	return p.publish(ctx, ChannelSubmissionStatus, event)
}

func (p *RedisPublisher) publish(ctx context.Context, channel string, event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := p.client.Publish(ctx, channel, data).Err(); err != nil {
		p.logger.Error("failed to publish redis event",
			slog.String("channel", channel),
			slog.String("error", err.Error()),
		)
		return err
	}

	p.logger.Debug("redis event published",
		slog.String("channel", channel),
	)
	return nil
}

// Close shuts down the Redis client.
func (p *RedisPublisher) Close() error {
	return p.client.Close()
}

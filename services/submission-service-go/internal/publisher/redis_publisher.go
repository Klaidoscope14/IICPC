package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/iicpc/pkg/events"
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
	ChannelValidationDone    = "validation:completed"
	ChannelEngineReady       = "engine:ready"
	ChannelBenchmarkStarted  = "benchmark:started"
	ChannelBenchmarkFinished = "benchmark:finished"
	ChannelTelemetrySnapshot = "telemetry:snapshot"
	ChannelLeaderboardUpdate = "leaderboard:updated"
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

func (p *RedisPublisher) PublishValidationCompleted(ctx context.Context, event events.ValidationCompletedEvent) error {
	return p.publish(ctx, ChannelValidationDone, event)
}

func (p *RedisPublisher) PublishEngineReady(ctx context.Context, event events.EngineReadyEvent) error {
	return p.publish(ctx, ChannelEngineReady, event)
}

func (p *RedisPublisher) PublishBenchmarkStarted(ctx context.Context, event events.BenchmarkStartedEvent) error {
	return p.publish(ctx, ChannelBenchmarkStarted, event)
}

func (p *RedisPublisher) PublishBenchmarkFinished(ctx context.Context, event events.BenchmarkFinishedEvent) error {
	return p.publish(ctx, ChannelBenchmarkFinished, event)
}

func (p *RedisPublisher) PublishTelemetrySnapshot(ctx context.Context, event events.TelemetrySnapshotEvent) error {
	return p.publish(ctx, ChannelTelemetrySnapshot, event)
}

func (p *RedisPublisher) PublishLeaderboardUpdated(ctx context.Context, event events.LeaderboardUpdatedEvent) error {
	return p.publish(ctx, ChannelLeaderboardUpdate, event)
}

// Publish sends a JSON payload to an explicit Redis channel.
func (p *RedisPublisher) Publish(ctx context.Context, channel string, event any) error {
	return p.publish(ctx, channel, event)
}

func (p *RedisPublisher) publish(ctx context.Context, channel string, event any) error {
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

package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Producer publishes events to Redpanda/Kafka topics.
type Producer struct {
	client *kgo.Client
	logger *slog.Logger
}

// NewProducer creates a Kafka producer connected to the given brokers.
func NewProducer(brokers []string, logger *slog.Logger) (*Producer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.AllowAutoTopicCreation(),
		kgo.ProducerBatchMaxBytes(1024*1024), // 1MB
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	return &Producer{
		client: client,
		logger: logger,
	}, nil
}

// PublishSubmissionCreated publishes a submission_created event.
func (p *Producer) PublishSubmissionCreated(ctx context.Context, event SubmissionCreatedEvent) error {
	return p.publish(ctx, TopicSubmissionCreated, event.SubmissionID, event)
}

// PublishSubmissionUploaded publishes a submission upload event.
func (p *Producer) PublishSubmissionUploaded(ctx context.Context, event SubmissionUploadedEvent) error {
	return p.publish(ctx, TopicSubmissionUploaded, event.SubmissionID, event)
}

// PublishValidationCompleted publishes a validation_completed event.
func (p *Producer) PublishValidationCompleted(ctx context.Context, event ValidationCompletedEvent) error {
	return p.publish(ctx, TopicValidationCompleted, event.SubmissionID, event)
}

// PublishBenchmarkCompleted publishes a benchmark_completed event.
func (p *Producer) PublishBenchmarkCompleted(ctx context.Context, event BenchmarkCompletedEvent) error {
	return p.publish(ctx, TopicBenchmarkCompleted, event.BenchmarkID, event)
}

// PublishBenchmarkFinished publishes a benchmark completion event.
func (p *Producer) PublishBenchmarkFinished(ctx context.Context, event BenchmarkFinishedEvent) error {
	return p.publish(ctx, TopicBenchmarkFinished, event.BenchmarkID, event)
}

// PublishDeploymentReady publishes a deployment_ready event.
func (p *Producer) PublishDeploymentReady(ctx context.Context, event DeploymentReadyEvent) error {
	return p.publish(ctx, TopicDeploymentReady, event.DeploymentID, event)
}

// PublishEngineReady publishes a deployment readiness event.
func (p *Producer) PublishEngineReady(ctx context.Context, event EngineReadyEvent) error {
	return p.publish(ctx, TopicEngineReady, event.DeploymentID, event)
}

// PublishTelemetrySnapshot publishes a real-time telemetry snapshot.
func (p *Producer) PublishTelemetrySnapshot(ctx context.Context, event TelemetrySnapshotEvent) error {
	return p.publish(ctx, TopicTelemetrySnapshot, event.BenchmarkID, event)
}

// PublishLeaderboardUpdated publishes a leaderboard refresh event.
func (p *Producer) PublishLeaderboardUpdated(ctx context.Context, event LeaderboardUpdatedEvent) error {
	return p.publish(ctx, TopicLeaderboardUpdated, event.BenchmarkID, event)
}

func (p *Producer) publish(ctx context.Context, topic string, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Value: data,
		Headers: []kgo.RecordHeader{
			{Key: "produced_at", Value: []byte(time.Now().Format(time.RFC3339))},
		},
	}

	results := p.client.ProduceSync(ctx, record)
	if err := results.FirstErr(); err != nil {
		p.logger.Error("failed to produce event",
			slog.String("topic", topic),
			slog.String("key", key),
			slog.String("error", err.Error()),
		)
		return err
	}

	p.logger.Info("event published",
		slog.String("topic", topic),
		slog.String("key", key),
	)

	return nil
}

// Close shuts down the producer.
func (p *Producer) Close() {
	p.client.Close()
}

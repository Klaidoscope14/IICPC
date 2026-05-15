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

// ProduceCallback receives the final broker acknowledgement for async produces.
type ProduceCallback func(topic string, key string, err error)

// NewProducer creates a Kafka producer connected to the given brokers.
func NewProducer(brokers []string, logger *slog.Logger) (*Producer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.AllowAutoTopicCreation(),
		kgo.ProducerBatchMaxBytes(1024*1024), // 1MB
		kgo.ProducerLinger(5*time.Millisecond),
		kgo.ProducerBatchCompression(kgo.SnappyCompression(), kgo.Lz4Compression()),
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

// PublishSubmissionDeleted publishes a submission deletion event.
func (p *Producer) PublishSubmissionDeleted(ctx context.Context, event SubmissionDeletedEvent) error {
	return p.publish(ctx, TopicSubmissionDeleted, event.SubmissionID, event)
}

// PublishValidationCompleted publishes a validation_completed event.
func (p *Producer) PublishValidationCompleted(ctx context.Context, event ValidationCompletedEvent) error {
	return p.publish(ctx, TopicValidationCompleted, event.SubmissionID, event)
}

// PublishBenchmarkStarted publishes a benchmark start event.
func (p *Producer) PublishBenchmarkStarted(ctx context.Context, event BenchmarkStartedEvent) error {
	return p.publish(ctx, TopicBenchmarkStarted, event.BenchmarkID, event)
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

// PublishAsync queues an event without blocking the caller on broker acknowledgement.
// Use it for high-frequency, non-critical updates such as telemetry snapshots.
func (p *Producer) PublishAsync(ctx context.Context, topic string, key string, value interface{}, cb ProduceCallback) error {
	record, err := p.buildRecord(topic, key, value)
	if err != nil {
		return err
	}

	p.client.Produce(ctx, record, func(record *kgo.Record, err error) {
		if err != nil {
			p.logger.Warn("failed to produce async event",
				slog.String("topic", topic),
				slog.String("key", key),
				slog.String("error", err.Error()),
			)
		}
		if cb != nil {
			cb(topic, key, err)
		}
	})

	return nil
}

// Flush waits for queued async records to be acknowledged.
func (p *Producer) Flush(ctx context.Context) error {
	return p.client.Flush(ctx)
}

func (p *Producer) publish(ctx context.Context, topic string, key string, value interface{}) error {
	record, err := p.buildRecord(topic, key, value)
	if err != nil {
		return err
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

func (p *Producer) buildRecord(topic string, key string, value interface{}) (*kgo.Record, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	return &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Value: data,
		Headers: []kgo.RecordHeader{
			{Key: "schema_version", Value: []byte(SchemaVersion)},
			{Key: "produced_at", Value: []byte(time.Now().UTC().Format(time.RFC3339Nano))},
		},
	}, nil
}

// Close shuts down the producer.
func (p *Producer) Close() {
	p.client.Close()
}

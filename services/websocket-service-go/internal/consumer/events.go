package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/iicpc/pkg/events"
	"github.com/iicpc/websocket-service-go/internal/ws"
)

// EventConsumer encapsulates the logic for consuming Redpanda events and forwarding to the WS hub.
type EventConsumer struct {
	consumer *events.Consumer
	hub      *ws.Hub
	logger   *slog.Logger
}

func NewEventConsumer(brokers []string, hub *ws.Hub, logger *slog.Logger) (*EventConsumer, error) {
	// We want to consume topics related to live updates.
	topics := []string{
		events.TopicTelemetrySnapshot,
		events.TopicLeaderboardUpdated,
		events.TopicBenchmarkStarted,
		events.TopicBenchmarkFinished,
		events.TopicEngineReady,
	}

	consumer, err := events.NewConsumerWithOptions(
		brokers,
		"websocket-service-group",
		topics,
		logger,
		events.ConsumerOptions{PartitionConcurrency: 16},
	)
	if err != nil {
		return nil, err
	}

	ec := &EventConsumer{
		consumer: consumer,
		hub:      hub,
		logger:   logger,
	}
	ec.registerHandlers()
	return ec, nil
}

func (ec *EventConsumer) registerHandlers() {
	events.RegisterJSONHandler(ec.consumer, events.TopicTelemetrySnapshot, func(ctx context.Context, key string, event events.TelemetrySnapshotEvent) error {
		// Broadcast to the specific benchmark room.
		payload, err := json.Marshal(event)
		if err == nil {
			ec.hub.Broadcast(event.BenchmarkID, payload)
		}
		return nil
	})

	events.RegisterJSONHandler(ec.consumer, events.TopicLeaderboardUpdated, func(ctx context.Context, key string, event events.LeaderboardUpdatedEvent) error {
		payload, err := json.Marshal(event)
		if err == nil {
			// Broadcast to the global room, and also the benchmark room if it is specified.
			ec.hub.Broadcast("", payload)
			if event.BenchmarkID != "" {
				ec.hub.Broadcast(event.BenchmarkID, payload)
			}
		}
		return nil
	})

	events.RegisterJSONHandler(ec.consumer, events.TopicBenchmarkStarted, func(ctx context.Context, key string, event events.BenchmarkStartedEvent) error {
		// Wrap with an event type tag so the client knows what it is.
		payload, _ := json.Marshal(map[string]interface{}{
			"event": "benchmark_started",
			"data":  event,
		})
		ec.hub.Broadcast(event.BenchmarkID, payload)
		return nil
	})

	events.RegisterJSONHandler(ec.consumer, events.TopicBenchmarkFinished, func(ctx context.Context, key string, event events.BenchmarkFinishedEvent) error {
		payload, _ := json.Marshal(map[string]interface{}{
			"event": "benchmark_ended", // Matches frontend expectation
			"data":  event,
		})
		ec.hub.Broadcast(event.BenchmarkID, payload)
		return nil
	})

	events.RegisterJSONHandler(ec.consumer, events.TopicEngineReady, func(ctx context.Context, key string, event events.EngineReadyEvent) error {
		payload, _ := json.Marshal(map[string]interface{}{
			"event": "container_status",
			"data":  event,
		})
		// EngineReady might not have BenchmarkID yet if it's just for deployment.
		// We can broadcast to submissionID or global if necessary, but we'll try global for now.
		ec.hub.Broadcast("", payload)
		return nil
	})
}

func (ec *EventConsumer) Start(ctx context.Context) error {
	ec.logger.Info("Starting event consumer for WebSocket service")
	return ec.consumer.Start(ctx)
}

func (ec *EventConsumer) Close() {
	ec.consumer.Close()
}

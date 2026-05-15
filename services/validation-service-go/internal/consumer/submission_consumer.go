package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/iicpc/pkg/events"
	"github.com/iicpc/validation-service-go/internal/service"
)

type SubmissionConsumer struct {
	valService *service.ValidationService
}

func NewSubmissionConsumer(valService *service.ValidationService) *SubmissionConsumer {
	return &SubmissionConsumer{valService: valService}
}

// HandleMessage implements the events.MessageHandler interface
func (c *SubmissionConsumer) HandleMessage(ctx context.Context, msg []byte) error {
	var event events.SubmissionCreatedEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		slog.Error("Failed to unmarshal submission.created event", "error", err)
		// We return nil because this is a poison pill (bad message format). We don't want to retry infinitely.
		return nil
	}

	slog.Info("Received submission.created event", "submission_id", event.SubmissionID)

	// Validate the submission. If this fails, we still return nil to the consumer
	// because the validation service already updates the DB status to "failed" and
	// publishes the validation.completed event. We don't want Redpanda to redeliver.
	if err := c.valService.ValidateSubmission(ctx, event.SubmissionID); err != nil {
		slog.Warn("Validation process encountered an error", "submission_id", event.SubmissionID, "error", err)
	}

	return nil
}

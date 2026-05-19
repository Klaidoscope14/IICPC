package events

import contractevents "github.com/iicpc/pkg/contracts/events"

const SchemaVersion = contractevents.SchemaVersion

// Topic names for the event bus.
const (
	TopicSubmissionCreated   = contractevents.TopicSubmissionUploaded
	TopicSubmissionUploaded  = contractevents.TopicSubmissionUploaded
	TopicSubmissionDeleted   = contractevents.TopicSubmissionDeleted
	TopicValidationCompleted = contractevents.TopicValidationCompleted
	TopicDeploymentReady     = contractevents.TopicEngineReady
	TopicEngineReady         = contractevents.TopicEngineReady
	TopicBenchmarkStarted    = contractevents.TopicBenchmarkStarted
	TopicBenchmarkCompleted  = contractevents.TopicBenchmarkFinished
	TopicBenchmarkFinished    = contractevents.TopicBenchmarkFinished
	TopicTelemetrySnapshot    = contractevents.TopicTelemetrySnapshot
	TopicLeaderboardUpdated   = contractevents.TopicLeaderboardUpdated
	TopicTraceAvailable       = contractevents.TopicTraceAvailable
	TopicCorrectnessEvaluated = contractevents.TopicCorrectnessEvaluated
)

// SubmissionCreatedEvent is published when a new submission is accepted.
type SubmissionCreatedEvent = contractevents.SubmissionUploadedEvent
type SubmissionUploadedEvent = contractevents.SubmissionUploadedEvent
type SubmissionDeletedEvent = contractevents.SubmissionDeletedEvent

// ValidationCompletedEvent is published when a submission has been validated.
type ValidationCompletedEvent = contractevents.ValidationCompletedEvent

// DeploymentReadyEvent is published when a submission's container is deployed.
type DeploymentReadyEvent = contractevents.EngineReadyEvent
type EngineReadyEvent = contractevents.EngineReadyEvent

// BenchmarkStartedEvent is published when a benchmark run begins.
type BenchmarkStartedEvent = contractevents.BenchmarkStartedEvent

// BenchmarkCompletedEvent is published when a benchmark run finishes.
type BenchmarkCompletedEvent = contractevents.BenchmarkFinishedEvent
type BenchmarkFinishedEvent = contractevents.BenchmarkFinishedEvent
type TelemetrySnapshotEvent = contractevents.TelemetrySnapshotEvent
type LeaderboardUpdatedEvent = contractevents.LeaderboardUpdatedEvent
type TraceAvailableEvent = contractevents.TraceAvailableEvent
type CorrectnessEvaluatedEvent = contractevents.CorrectnessEvaluatedEvent
type Envelope = contractevents.Envelope

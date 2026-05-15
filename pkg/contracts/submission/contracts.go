package submission

import "time"

type Status string

const (
	StatusPending          Status = "pending"
	StatusUploaded         Status = "uploaded"
	StatusValidationQueued Status = "validation_queued"
	StatusValidating       Status = "validating"
	StatusValidated        Status = "validated"
	StatusValidationFailed Status = "validation_failed"
	StatusDeploying        Status = "deploying"
	StatusDeployed         Status = "deployed"
	StatusBenchmarking     Status = "benchmarking"
	StatusCompleted        Status = "completed"
	StatusFailed           Status = "failed"
	StatusDeleted          Status = "deleted"
)

type Language string

const (
	LanguageCPP    Language = "cpp"
	LanguageC      Language = "c"
	LanguageRust   Language = "rust"
	LanguageGo     Language = "go"
	LanguagePython Language = "python"
)

type Metadata map[string]string

// UploadRequest is the canonical service-layer submission upload contract.
type UploadRequest struct {
	ContestantID     string   `json:"contestant_id"`
	TeamName         string   `json:"team_name"`
	Language         Language `json:"language"`
	ArchiveName      string   `json:"archive_name"`
	ArchiveBytes     []byte   `json:"-"`
	ArchiveSizeBytes int64    `json:"archive_size_bytes"`
	Dockerfile       string   `json:"dockerfile,omitempty"`
	IdempotencyKey   string   `json:"idempotency_key,omitempty"`
	Metadata         Metadata `json:"metadata,omitempty"`
}

type UploadResponse struct {
	SubmissionID     string    `json:"submission_id"`
	Status           Status    `json:"status"`
	Version          int       `json:"version"`
	Checksum         string    `json:"checksum"`
	ArchiveSizeBytes int64     `json:"archive_size_bytes"`
	OriginalFilename string    `json:"original_filename"`
	CreatedAt        time.Time `json:"created_at"`
}

type Summary struct {
	ID               string    `json:"id"`
	ContestantID     string    `json:"contestant_id"`
	TeamName         string    `json:"team_name"`
	Language         Language  `json:"language"`
	Status           Status    `json:"status"`
	Version          int       `json:"version"`
	Checksum         string    `json:"checksum,omitempty"`
	OriginalFilename string    `json:"original_filename,omitempty"`
	FileSize         int64     `json:"file_size,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ListRequest struct {
	ContestantID string `json:"contestant_id,omitempty" form:"contestant_id"`
	Status       Status `json:"status,omitempty" form:"status"`
	Limit        int    `json:"limit,omitempty" form:"limit"`
	PageToken    string `json:"page_token,omitempty" form:"page_token"`
}

type UpdateStatusRequest struct {
	Status       Status `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type LogLevel string

const (
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

type LogType string

const (
	LogTypeUpload     LogType = "upload"
	LogTypeValidation LogType = "validation"
	LogTypeBuild      LogType = "build"
	LogTypeRuntime    LogType = "runtime"
	LogTypeBenchmark  LogType = "benchmark"
)

type LogEntry struct {
	ID           string    `json:"id"`
	SubmissionID string    `json:"submission_id"`
	Type         LogType   `json:"type"`
	Level        LogLevel  `json:"level"`
	Message      string    `json:"message"`
	CreatedAt    time.Time `json:"created_at"`
}

func (s Status) IsTerminal() bool {
	return s == StatusCompleted || s == StatusFailed || s == StatusValidationFailed || s == StatusDeleted
}

func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusUploaded, StatusValidationQueued, StatusValidating, StatusValidated,
		StatusValidationFailed, StatusDeploying, StatusDeployed, StatusBenchmarking,
		StatusCompleted, StatusFailed, StatusDeleted:
		return true
	default:
		return false
	}
}

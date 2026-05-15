-- ============================================================================
-- IICPC Distributed Benchmarking Platform — Database Schema
-- ============================================================================

-- Create submissions table
CREATE TABLE IF NOT EXISTS submissions (
    id VARCHAR(255) PRIMARY KEY,
    contestant_id VARCHAR(255) NOT NULL,
    team_name VARCHAR(255) NOT NULL,
    language VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    version INTEGER NOT NULL DEFAULT 1,
    code_archive BYTEA NOT NULL,
    dockerfile TEXT,
    checksum VARCHAR(64),
    original_filename VARCHAR(255),
    file_size BIGINT,
    storage_path TEXT,
    idempotency_key VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create submission_logs table
CREATE TABLE IF NOT EXISTS submission_logs (
    id VARCHAR(255) PRIMARY KEY,
    submission_id VARCHAR(255) NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    log_type VARCHAR(50) NOT NULL, -- 'upload', 'build', 'runtime', 'validation'
    message TEXT NOT NULL,
    level VARCHAR(20) NOT NULL DEFAULT 'info', -- 'info', 'warn', 'error'
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create validation_results table
CREATE TABLE IF NOT EXISTS validation_results (
    id VARCHAR(255) PRIMARY KEY,
    submission_id VARCHAR(255) NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, running, passed, failed
    language VARCHAR(50),
    runtime VARCHAR(100),
    errors JSONB DEFAULT '[]',
    warnings JSONB DEFAULT '[]',
    report JSONB,
    validated_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create deployments table
CREATE TABLE IF NOT EXISTS deployments (
    id VARCHAR(255) PRIMARY KEY,
    submission_id VARCHAR(255) NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    container_id VARCHAR(255),
    container_image TEXT NOT NULL,
    service_url TEXT,
    exposed_ports TEXT[],
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    resource_limits JSONB,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create benchmarks table
CREATE TABLE IF NOT EXISTS benchmarks (
    id VARCHAR(255) PRIMARY KEY,
    submission_id VARCHAR(255) NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    deployment_id VARCHAR(255) NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    config JSONB,
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    elapsed_seconds BIGINT DEFAULT 0,
    error_message TEXT
);

-- Create benchmark results table (final summary metrics per benchmark)
CREATE TABLE IF NOT EXISTS benchmark_results (
    id VARCHAR(255) PRIMARY KEY,
    submission_id VARCHAR(255) UNIQUE NOT NULL,
    benchmark_id VARCHAR(255) REFERENCES benchmarks(id) ON DELETE CASCADE,
    tps DOUBLE PRECISION,
    p50_latency_ms DOUBLE PRECISION,
    p90_latency_ms DOUBLE PRECISION,
    p99_latency_ms DOUBLE PRECISION,
    correctness_score DOUBLE PRECISION,
    total_orders INTEGER,
    failed_orders INTEGER,
    composite_score DOUBLE PRECISION DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (submission_id) REFERENCES submissions(id) ON DELETE CASCADE
);

-- Create telemetry_snapshots table (time-series metrics during a benchmark run)
CREATE TABLE IF NOT EXISTS telemetry_snapshots (
    id BIGSERIAL PRIMARY KEY,
    benchmark_id VARCHAR(255) NOT NULL REFERENCES benchmarks(id) ON DELETE CASCADE,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    current_tps DOUBLE PRECISION,
    avg_latency_ms DOUBLE PRECISION,
    p50_latency_ms DOUBLE PRECISION,
    p90_latency_ms DOUBLE PRECISION,
    p99_latency_ms DOUBLE PRECISION,
    total_orders_sent INTEGER,
    total_orders_acknowledged INTEGER,
    total_errors INTEGER,
    active_connections INTEGER,
    cpu_usage_percent DOUBLE PRECISION,
    memory_usage_mb DOUBLE PRECISION
);

-- ============================================================================
-- Indexes
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_submissions_contestant_id ON submissions(contestant_id);
CREATE INDEX IF NOT EXISTS idx_submissions_status ON submissions(status);
CREATE INDEX IF NOT EXISTS idx_submissions_created_at ON submissions(created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_submissions_idempotency_key ON submissions(idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_submissions_contestant_version ON submissions(contestant_id, version DESC);
CREATE INDEX IF NOT EXISTS idx_submission_logs_submission_id ON submission_logs(submission_id);
CREATE INDEX IF NOT EXISTS idx_submission_logs_created_at ON submission_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_validation_results_submission_id ON validation_results(submission_id);
CREATE INDEX IF NOT EXISTS idx_validation_results_status ON validation_results(status);
CREATE INDEX IF NOT EXISTS idx_benchmark_results_submission_id ON benchmark_results(submission_id);
CREATE INDEX IF NOT EXISTS idx_benchmark_results_score ON benchmark_results(composite_score DESC);
CREATE INDEX IF NOT EXISTS idx_deployments_submission_id ON deployments(submission_id);
CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status);
CREATE INDEX IF NOT EXISTS idx_benchmarks_submission_id ON benchmarks(submission_id);
CREATE INDEX IF NOT EXISTS idx_benchmarks_deployment_id ON benchmarks(deployment_id);
CREATE INDEX IF NOT EXISTS idx_benchmarks_status ON benchmarks(status);
CREATE INDEX IF NOT EXISTS idx_telemetry_snapshots_benchmark_id ON telemetry_snapshots(benchmark_id);
CREATE INDEX IF NOT EXISTS idx_telemetry_snapshots_timestamp ON telemetry_snapshots(benchmark_id, timestamp DESC);

-- ============================================================================
-- Triggers
-- ============================================================================

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_submissions_updated_at BEFORE UPDATE ON submissions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_deployments_updated_at BEFORE UPDATE ON deployments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();


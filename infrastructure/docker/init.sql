-- ============================================================================
-- IICPC Distributed Benchmarking Platform - Database Schema
-- PostgreSQL 15 + TimescaleDB
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

-- ============================================================================
-- Core Metadata
-- ============================================================================

CREATE TABLE IF NOT EXISTS teams (
    contestant_id VARCHAR(255) PRIMARY KEY,
    team_name VARCHAR(255) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS submissions (
    id VARCHAR(255) PRIMARY KEY,
    contestant_id VARCHAR(255) NOT NULL REFERENCES teams(contestant_id) ON UPDATE CASCADE,
    team_name VARCHAR(255) NOT NULL,
    language VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    version INTEGER NOT NULL DEFAULT 1,
    -- Kept nullable for backwards compatibility; storage_path is the scalable source of truth.
    code_archive BYTEA,
    dockerfile TEXT,
    checksum VARCHAR(64),
    original_filename VARCHAR(255),
    file_size BIGINT,
    storage_path TEXT,
    idempotency_key VARCHAR(255),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT submissions_status_check CHECK (
        status IN (
            'pending', 'uploaded', 'validation_queued', 'validating', 'validated',
            'validation_failed', 'processing', 'deploying', 'deployed',
            'benchmarking', 'completed', 'failed', 'deleted'
        )
    ),
    CONSTRAINT submissions_file_size_check CHECK (file_size IS NULL OR file_size >= 0),
    CONSTRAINT submissions_version_check CHECK (version > 0)
);

CREATE TABLE IF NOT EXISTS submission_logs (
    id VARCHAR(255) PRIMARY KEY,
    submission_id VARCHAR(255) NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    log_type VARCHAR(50) NOT NULL,
    message TEXT NOT NULL,
    level VARCHAR(20) NOT NULL DEFAULT 'info',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT submission_logs_level_check CHECK (level IN ('debug', 'info', 'warn', 'error')),
    CONSTRAINT submission_logs_type_check CHECK (log_type IN ('upload', 'build', 'runtime', 'validation', 'benchmark', 'system'))
);

-- ============================================================================
-- Validation
-- ============================================================================

CREATE TABLE IF NOT EXISTS validation_results (
    id VARCHAR(255) PRIMARY KEY,
    submission_id VARCHAR(255) NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    language VARCHAR(50),
    runtime VARCHAR(100),
    errors JSONB NOT NULL DEFAULT '[]'::jsonb,
    warnings JSONB NOT NULL DEFAULT '[]'::jsonb,
    report JSONB,
    validated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT validation_results_status_check CHECK (status IN ('pending', 'running', 'passed', 'failed'))
);

-- One current validation row per submission; full details can still live in report JSON.
CREATE UNIQUE INDEX IF NOT EXISTS idx_validation_results_submission_unique
    ON validation_results(submission_id);

-- ============================================================================
-- Deployments and Benchmarks
-- ============================================================================

CREATE TABLE IF NOT EXISTS deployments (
    id VARCHAR(255) PRIMARY KEY,
    submission_id VARCHAR(255) NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    container_id VARCHAR(255),
    container_image TEXT NOT NULL,
    service_url TEXT,
    exposed_ports TEXT[] NOT NULL DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    resource_limits JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT deployments_status_check CHECK (status IN ('pending', 'building', 'deployed', 'failed', 'terminated'))
);

CREATE TABLE IF NOT EXISTS benchmarks (
    id VARCHAR(255) PRIMARY KEY,
    submission_id VARCHAR(255) NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    deployment_id VARCHAR(255) NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    elapsed_seconds BIGINT DEFAULT 0,
    error_message TEXT,
    CONSTRAINT benchmarks_status_check CHECK (status IN ('pending', 'running', 'completed', 'failed', 'stopped')),
    CONSTRAINT benchmarks_elapsed_check CHECK (elapsed_seconds IS NULL OR elapsed_seconds >= 0)
);

-- Current leaderboard row per submission.
CREATE TABLE IF NOT EXISTS benchmark_results (
    id VARCHAR(255) PRIMARY KEY,
    submission_id VARCHAR(255) UNIQUE NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    benchmark_id VARCHAR(255) REFERENCES benchmarks(id) ON DELETE CASCADE,
    tps DOUBLE PRECISION NOT NULL DEFAULT 0,
    p50_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    p90_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    p99_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    correctness_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    total_orders INTEGER NOT NULL DEFAULT 0,
    failed_orders INTEGER NOT NULL DEFAULT 0,
    composite_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT benchmark_results_scores_check CHECK (
        tps >= 0
        AND p50_latency_ms >= 0
        AND p90_latency_ms >= 0
        AND p99_latency_ms >= 0
        AND correctness_score >= 0
        AND total_orders >= 0
        AND failed_orders >= 0
        AND composite_score >= 0
    )
);

-- Immutable benchmark history, one row per completed benchmark run.
CREATE TABLE IF NOT EXISTS benchmark_history (
    id VARCHAR(255) PRIMARY KEY,
    submission_id VARCHAR(255) NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    benchmark_id VARCHAR(255) UNIQUE NOT NULL REFERENCES benchmarks(id) ON DELETE CASCADE,
    tps DOUBLE PRECISION NOT NULL DEFAULT 0,
    p50_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    p90_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    p99_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    correctness_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    total_orders INTEGER NOT NULL DEFAULT 0,
    failed_orders INTEGER NOT NULL DEFAULT 0,
    composite_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Leaderboard projection. It stays live while the underlying benchmark_results
-- table remains the indexed current-state source of truth.
CREATE OR REPLACE VIEW leaderboard AS
SELECT
    ROW_NUMBER() OVER (ORDER BY br.composite_score DESC, br.tps DESC, br.p99_latency_ms ASC) AS rank,
    br.submission_id,
    s.team_name,
    br.tps,
    br.p50_latency_ms,
    br.p90_latency_ms,
    br.p99_latency_ms,
    br.correctness_score,
    br.total_orders,
    br.failed_orders,
    br.composite_score,
    br.created_at
FROM benchmark_results br
JOIN submissions s ON s.id = br.submission_id
WHERE s.status != 'deleted';

-- ============================================================================
-- Time-Series Metrics
-- ============================================================================

CREATE TABLE IF NOT EXISTS telemetry_snapshots (
    id BIGSERIAL,
    benchmark_id VARCHAR(255) NOT NULL REFERENCES benchmarks(id) ON DELETE CASCADE,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    current_tps DOUBLE PRECISION NOT NULL DEFAULT 0,
    avg_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    p50_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    p90_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    p99_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    total_orders_sent INTEGER NOT NULL DEFAULT 0,
    total_orders_acknowledged INTEGER NOT NULL DEFAULT 0,
    total_errors INTEGER NOT NULL DEFAULT 0,
    active_connections INTEGER NOT NULL DEFAULT 0,
    cpu_usage_percent DOUBLE PRECISION NOT NULL DEFAULT 0,
    memory_usage_mb DOUBLE PRECISION NOT NULL DEFAULT 0,
    PRIMARY KEY (benchmark_id, timestamp, id),
    CONSTRAINT telemetry_non_negative_check CHECK (
        current_tps >= 0
        AND avg_latency_ms >= 0
        AND p50_latency_ms >= 0
        AND p90_latency_ms >= 0
        AND p99_latency_ms >= 0
        AND total_orders_sent >= 0
        AND total_orders_acknowledged >= 0
        AND total_errors >= 0
        AND active_connections >= 0
        AND cpu_usage_percent >= 0
        AND memory_usage_mb >= 0
    )
);

SELECT create_hypertable(
    'telemetry_snapshots',
    'timestamp',
    chunk_time_interval => INTERVAL '1 hour',
    if_not_exists => TRUE
);

SELECT add_retention_policy('telemetry_snapshots', INTERVAL '14 days', if_not_exists => TRUE);

-- ============================================================================
-- Indexes Optimized Around Existing Query Paths
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_teams_name ON teams(team_name);

CREATE INDEX IF NOT EXISTS idx_submissions_active_created
    ON submissions(created_at DESC)
    WHERE status != 'deleted';

CREATE INDEX IF NOT EXISTS idx_submissions_active_contestant_created
    ON submissions(contestant_id, created_at DESC)
    WHERE status != 'deleted';

CREATE INDEX IF NOT EXISTS idx_submissions_active_status_created
    ON submissions(status, created_at DESC)
    WHERE status != 'deleted';

CREATE UNIQUE INDEX IF NOT EXISTS idx_submissions_idempotency_key
    ON submissions(idempotency_key)
    WHERE idempotency_key IS NOT NULL AND status != 'deleted';

CREATE UNIQUE INDEX IF NOT EXISTS idx_submissions_contestant_version
    ON submissions(contestant_id, version)
    WHERE status != 'deleted';

CREATE INDEX IF NOT EXISTS idx_submission_logs_submission_created
    ON submission_logs(submission_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_validation_results_status_updated
    ON validation_results(status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_deployments_submission_created
    ON deployments(submission_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_deployments_status_updated
    ON deployments(status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_benchmarks_submission_started
    ON benchmarks(submission_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_benchmarks_deployment_started
    ON benchmarks(deployment_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_benchmarks_status_started
    ON benchmarks(status, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_benchmark_results_score
    ON benchmark_results(composite_score DESC, tps DESC, p99_latency_ms ASC);

CREATE INDEX IF NOT EXISTS idx_benchmark_history_submission_created
    ON benchmark_history(submission_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_benchmark_history_score
    ON benchmark_history(composite_score DESC, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_telemetry_snapshots_benchmark_time
    ON telemetry_snapshots(benchmark_id, timestamp DESC);

-- ============================================================================
-- Triggers and Helpers
-- ============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION upsert_team_from_submission()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO teams (contestant_id, team_name, updated_at)
    VALUES (NEW.contestant_id, NEW.team_name, NOW())
    ON CONFLICT (contestant_id) DO UPDATE SET
        team_name = EXCLUDED.team_name,
        updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_teams_updated_at ON teams;
CREATE TRIGGER update_teams_updated_at
    BEFORE UPDATE ON teams
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_submissions_updated_at ON submissions;
CREATE TRIGGER update_submissions_updated_at
    BEFORE UPDATE ON submissions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS upsert_team_from_submission_trigger ON submissions;
CREATE TRIGGER upsert_team_from_submission_trigger
    BEFORE INSERT OR UPDATE OF contestant_id, team_name ON submissions
    FOR EACH ROW EXECUTE FUNCTION upsert_team_from_submission();

DROP TRIGGER IF EXISTS update_validation_results_updated_at ON validation_results;
CREATE TRIGGER update_validation_results_updated_at
    BEFORE UPDATE ON validation_results
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_deployments_updated_at ON deployments;
CREATE TRIGGER update_deployments_updated_at
    BEFORE UPDATE ON deployments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

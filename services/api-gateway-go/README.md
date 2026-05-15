# API Gateway (Go)

The API gateway is the single external entry point for REST and WebSocket traffic.

## Current Responsibilities

- Routes `/api/v1/submissions/**` to submission-service.
- Routes `/api/v1/validations/**` to validation-service.
- Routes `/api/v1/deployments/**`, `/api/v1/benchmarks/**`, and `/api/v1/leaderboard` to benchmark-orchestrator.
- Proxies `/ws/**` WebSocket connections to benchmark-orchestrator.
- Applies sharded token-bucket rate limiting.
- Optionally enforces MVP bearer auth when `API_AUTH_TOKEN` is set.
- Validates API version, request size, request content type, and pagination query values.
- Normalizes gateway and backend error responses to the shared API error contract.
- Exposes `/health` with concurrent backend health checks and `/metrics` for Prometheus.

## Configuration

- `SERVER_PORT`
- `SUBMISSION_SERVICE_URL`
- `VALIDATION_SERVICE_URL`
- `ORCHESTRATOR_URL`
- `RATE_LIMIT_PER_MINUTE`
- `MAX_BODY_SIZE_MB`
- `API_AUTH_TOKEN`
- `LOG_LEVEL`

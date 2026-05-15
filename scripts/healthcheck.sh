#!/usr/bin/env sh
set -eu

check_http() {
  name="$1"
  url="$2"
  attempts="${3:-30}"
  i=1

  while [ "$i" -le "$attempts" ]; do
    if command -v curl >/dev/null 2>&1; then
      if curl -fsS "$url" >/dev/null 2>&1; then
        printf "%-26s %s\n" "$name" "ok"
        return 0
      fi
    elif wget -qO- "$url" >/dev/null 2>&1; then
      printf "%-26s %s\n" "$name" "ok"
      return 0
    fi
    i=$((i + 1))
    sleep 1
  done

  printf "%-26s %s\n" "$name" "failed"
  return 1
}

check_http "submission-service" "http://localhost:${SUBMISSION_SERVICE_PORT:-8080}/health"
check_http "validation-service" "http://localhost:${VALIDATION_SERVICE_PORT:-8084}/health"
check_http "benchmark-orchestrator" "http://localhost:${ORCHESTRATOR_PORT:-8081}/health"
check_http "api-gateway" "http://localhost:${API_GATEWAY_PORT:-8082}/health"
check_http "dashboard" "http://localhost:${DASHBOARD_PORT:-3000}" 45

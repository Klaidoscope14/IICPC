#include "telemetry/types.hpp"
#include <cstdlib>

namespace telemetry {

TelemetryConfig TelemetryConfig::from_env() {
    TelemetryConfig config;

    if (auto* val = std::getenv("INGESTION_PORT"))     config.ingestion_port = std::atoi(val);
    if (auto* val = std::getenv("STREAMING_PORT"))     config.streaming_port = std::atoi(val);
    if (auto* val = std::getenv("WINDOW_SECONDS"))     config.window_seconds = std::atoi(val);
    if (auto* val = std::getenv("MAX_SNAPSHOTS"))      config.max_snapshots = std::atoi(val);
    if (auto* val = std::getenv("DB_CONNECTION"))      config.db_connection_string = val;
    if (auto* val = std::getenv("BENCHMARK_ID"))       config.benchmark_id = val;

    return config;
}

} // namespace telemetry

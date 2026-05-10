#pragma once

#include <cstdint>
#include <string>
#include <chrono>

namespace telemetry {

/// A single raw metric data point from the bot fleet.
struct RawMetric {
    std::string benchmark_id;
    std::string order_id;
    std::string symbol;
    std::string side;           // "buy" or "sell"
    bool success;
    int64_t latency_us;         // Microseconds
    std::chrono::system_clock::time_point timestamp;
    double price;
    int32_t quantity;
    std::string error_message;
};

/// Aggregated snapshot of metrics over a time window.
struct AggregatedSnapshot {
    std::string benchmark_id;
    std::chrono::system_clock::time_point timestamp;

    // Throughput.
    double current_tps;
    int64_t total_orders_sent;
    int64_t total_orders_acknowledged;
    int64_t total_errors;

    // Latency percentiles (microseconds).
    double avg_latency_us;
    double p50_latency_us;
    double p90_latency_us;
    double p99_latency_us;
    double min_latency_us;
    double max_latency_us;

    // Resource usage (if available).
    int32_t active_connections;
    double cpu_usage_percent;
    double memory_usage_mb;
};

/// Configuration for the telemetry pipeline.
struct TelemetryConfig {
    /// Port for the ingestion server.
    int ingestion_port = 9100;

    /// Port for the streaming server.
    int streaming_port = 9101;

    /// Aggregation window in seconds.
    int window_seconds = 1;

    /// Maximum number of snapshots to retain in memory.
    int max_snapshots = 3600; // 1 hour at 1/sec

    /// Database connection string for persistent storage.
    std::string db_connection_string;

    /// Benchmark ID to filter on.
    std::string benchmark_id;

    /// Load from environment variables.
    static TelemetryConfig from_env();
};

} // namespace telemetry

#pragma once

#include "telemetry/types.hpp"
#include "telemetry/histogram.hpp"
#include "telemetry/sliding_window.hpp"
#include <mutex>
#include <atomic>

namespace telemetry {

/// Aggregates raw metrics into periodic snapshots.
///
/// Thread-safe: multiple ingestion threads can call ingest() concurrently.
/// A separate thread calls snapshot() periodically to produce AggregatedSnapshots.
class MetricsAggregator {
public:
    explicit MetricsAggregator(const std::string& benchmark_id);

    /// Ingest a single raw metric data point. Thread-safe.
    void ingest(const RawMetric& metric);

    /// Produce an aggregated snapshot and reset counters for the next window.
    AggregatedSnapshot snapshot();

    /// Get the current count of ingested metrics (since last snapshot).
    int64_t current_count() const { return orders_sent_.load(); }

private:
    std::string benchmark_id_;
    mutable std::mutex mutex_;

    Histogram latency_histogram_;
    SlidingWindow tps_window_;

    std::atomic<int64_t> orders_sent_{0};
    std::atomic<int64_t> orders_acked_{0};
    std::atomic<int64_t> errors_{0};
};

} // namespace telemetry

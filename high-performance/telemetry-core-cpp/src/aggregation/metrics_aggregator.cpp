#include "telemetry/metrics_aggregator.hpp"

namespace telemetry {

MetricsAggregator::MetricsAggregator(const std::string& benchmark_id)
    : benchmark_id_(benchmark_id)
    , latency_histogram_(10'000'000)  // 10 second max
    , tps_window_(std::chrono::seconds(1))
{}

void MetricsAggregator::ingest(const RawMetric& metric) {
    orders_sent_++;
    tps_window_.record();

    if (metric.success) {
        orders_acked_++;
    } else {
        errors_++;
    }

    // Record latency in the histogram (thread-safe via mutex).
    {
        std::lock_guard<std::mutex> lock(mutex_);
        latency_histogram_.record(metric.latency_us);
    }
}

AggregatedSnapshot MetricsAggregator::snapshot() {
    std::lock_guard<std::mutex> lock(mutex_);

    AggregatedSnapshot snap;
    snap.benchmark_id = benchmark_id_;
    snap.timestamp = std::chrono::system_clock::now();

    snap.current_tps = tps_window_.rate();
    snap.total_orders_sent = orders_sent_.load();
    snap.total_orders_acknowledged = orders_acked_.load();
    snap.total_errors = errors_.load();

    snap.avg_latency_us = latency_histogram_.mean();
    snap.p50_latency_us = latency_histogram_.percentile(0.50);
    snap.p90_latency_us = latency_histogram_.percentile(0.90);
    snap.p99_latency_us = latency_histogram_.percentile(0.99);
    snap.min_latency_us = static_cast<double>(latency_histogram_.min());
    snap.max_latency_us = static_cast<double>(latency_histogram_.max());

    snap.active_connections = 0;
    snap.cpu_usage_percent = 0.0;
    snap.memory_usage_mb = 0.0;

    // Reset histogram for next window (keep cumulative counters).
    latency_histogram_.reset();

    return snap;
}

} // namespace telemetry

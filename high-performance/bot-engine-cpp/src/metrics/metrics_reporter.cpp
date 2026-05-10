#include "bot_engine/metrics_reporter.hpp"
#include <iostream>
#include <sstream>

namespace bot_engine {

MetricsReporter::MetricsReporter(const std::string& endpoint, const std::string& benchmark_id)
    : endpoint_(endpoint)
    , benchmark_id_(benchmark_id)
{}

bool MetricsReporter::report(const PoolMetrics& metrics) {
    // In production, this would POST to the orchestrator's telemetry endpoint.
    // For now, log to stdout.
    std::cout << "[Metrics] benchmark=" << benchmark_id_
              << " tps=" << metrics.current_tps
              << " sent=" << metrics.total_orders_sent
              << " acked=" << metrics.total_orders_acked
              << " failed=" << metrics.total_orders_failed
              << " avg_latency_us=" << metrics.avg_latency_us
              << std::endl;
    return true;
}

} // namespace bot_engine

#pragma once

#include <string>
#include <chrono>
#include "bot_engine/bot_pool.hpp"

namespace bot_engine {

/// Reports aggregate metrics to the benchmark orchestrator.
///
/// Periodically sends PoolMetrics snapshots to the configured metrics endpoint
/// so the orchestrator can track the benchmark in real time.
class MetricsReporter {
public:
    MetricsReporter(const std::string& endpoint, const std::string& benchmark_id);

    /// Send a single metrics snapshot. Returns true on success.
    bool report(const PoolMetrics& metrics);

private:
    std::string endpoint_;
    std::string benchmark_id_;
};

} // namespace bot_engine
